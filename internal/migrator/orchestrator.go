package migrator

import (
	"fmt"

	migmodel "example.com/go-migrator/internal/migrator/model"
	"example.com/go-migrator/internal/migrator/translator"
	"example.com/go-migrator/internal/store"
	"github.com/google/uuid"
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
func (o *Orchestrator) Run(zoomUserID, zoomChannelID, teamName, channelName string, teamType migmodel.TeamType, channelType migmodel.ChannelType, stm *store.StoreManager) error {
	msgs, err := o.Source.FetchMessages(zoomUserID, zoomChannelID)
	if err != nil {
		return fmt.Errorf("fetch messages: %w", err)
	}

	// Get zoom channel members
	zmembers, err := o.Source.FetchChannelMembers(zoomUserID, zoomChannelID)
	if err != nil {
		return fmt.Errorf("fetch channel members: %w", err)
	}

	// Build memberID to userID map
	memberIDToUserID := make(map[string]string)
	for _, member := range zmembers {
		memberIDToUserID[member.MemberID] = member.ID
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
		// Find Teams user ID and display name from identity mapping
		zoomUserID := memberIDToUserID[zm.SendMemberID]
		var teamUserID, teamUserDisplayName string
		identity, err := stm.Identity.GetByZoomID(zoomUserID)
		if identity == nil {
			teamUserID = uuid.New().String()
			teamUserDisplayName = "Unknown User"
		} else {
			teamUserID = identity.TeamsUserID
			teamUserDisplayName = identity.TeamsUserDisplayName
		}
		if err != nil {
			return fmt.Errorf("unable to get identity by zoom user ID: %w", err)
		}

		tm := translator.TranslateZoomToTeams(zm, teamUserID, teamUserDisplayName)

		if err := o.Dest.PostMessage(teamID, chID, tm); err != nil {
			return fmt.Errorf("post message: %w", err)
		}
	}
	return nil
}
