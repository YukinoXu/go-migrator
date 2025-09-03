package migrator

import (
	"fmt"

	migmodel "example.com/go-migrator/internal/migrator/model"
	"example.com/go-migrator/internal/model"
	"example.com/go-migrator/internal/store"
)

// Orchestrator runs a migration from source to destination.
type Orchestrator struct {
	Source migmodel.SourceClient
	Dest   migmodel.DestinationClient
}

func NewOrchestrator(s migmodel.SourceClient, d migmodel.DestinationClient) *Orchestrator {
	return &Orchestrator{Source: s, Dest: d}
}

// Run migrates messages from the conversation on source to a team/channel on destination.
// It accepts the Store so it can resolve Zoom user IDs to Teams identities.
func (o *Orchestrator) Run(zoomUserID, zoomChannelID, teamName, channelName string, teamType migmodel.TeamType, channelType migmodel.ChannelType, idStore store.Store) error {
	msgs, err := o.Source.FetchMessages(zoomUserID, zoomChannelID)
	if err != nil {
		return fmt.Errorf("fetch messages: %w", err)
	}

	// attempt to resolve identity mapping for the zoom user
	var identity *model.Identity
	if idStore != nil {
		if id, err := idStore.GetIdentityByZoomUserID(zoomUserID); err == nil {
			identity = id
		}
	}

	teamID, err := o.Dest.EnsureTeam(teamName, teamType)
	if err != nil {
		return fmt.Errorf("ensure team: %w", err)
	}
	chID, err := o.Dest.EnsureChannel(teamID, channelName, channelType)
	if err != nil {
		return fmt.Errorf("ensure channel: %w", err)
	}

	for _, zm := range msgs {
		// ensure meta map exists
		if zm.Meta == nil {
			zm.Meta = map[string]string{}
		}
		// attach identity mapping to message meta when available
		if identity != nil {
			if identity.TeamsUserID != "" {
				zm.Meta["teams_user_id"] = identity.TeamsUserID
			}
			if identity.TeamsUserPrincipalName != "" {
				zm.Meta["teams_user_principal_name"] = identity.TeamsUserPrincipalName
			}
			if identity.TeamsUserDisplayName != "" && zm.SenderDisplayName == "" {
				zm.SenderDisplayName = identity.TeamsUserDisplayName
			}
		}

		if err := o.Dest.PostMessage(teamID, chID, zm); err != nil {
			return fmt.Errorf("post message: %w", err)
		}
	}
	return nil
}
