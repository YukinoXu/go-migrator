package migrator

import (
	"fmt"

	"example.com/go-migrator/internal/migrator/model"
)

// Orchestrator runs a migration from source to destination.
type Orchestrator struct {
	Source model.SourceClient
	Dest   model.DestinationClient
}

func NewOrchestrator(s model.SourceClient, d model.DestinationClient) *Orchestrator {
	return &Orchestrator{Source: s, Dest: d}
}

// Run migrates messages from the conversation on source to a team/channel on destination.
func (o *Orchestrator) Run(conversationID, teamName, channelName string) error {
	msgs, err := o.Source.FetchMessages(conversationID)
	if err != nil {
		return fmt.Errorf("fetch messages: %w", err)
	}
	teamID, err := o.Dest.EnsureTeam(teamName)
	if err != nil {
		return fmt.Errorf("ensure team: %w", err)
	}
	chID, err := o.Dest.EnsureChannel(teamID, channelName)
	if err != nil {
		return fmt.Errorf("ensure channel: %w", err)
	}
	for _, m := range msgs {
		if err := o.Dest.PostMessage(teamID, chID, m); err != nil {
			return fmt.Errorf("post message: %w", err)
		}
	}
	return nil
}
