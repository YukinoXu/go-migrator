package migrator

import (
	"fmt"
	"log"

	teamdest "example.com/go-migrator/internal/migrator/dest/teams"
	zoomsrc "example.com/go-migrator/internal/migrator/source/zoom"

	"example.com/go-migrator/internal/migrator/model"
	"example.com/go-migrator/internal/store"
)

// MigrateTask is a thin adapter used by the worker: it instantiates provider clients
// from environment and runs the orchestrator. It accepts an IdentityStore so
// the orchestrator can resolve user mappings.
func MigrateTask(zoomUserID, zoomChannelID, teamName, channelName string, idStore store.Store) error {
	src, err := zoomsrc.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("zoom client: %w", err)
	}
	dst, err := teamdest.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("teams client: %w", err)
	}
	orchestrator := NewOrchestrator(src, dst)
	return orchestrator.Run(zoomUserID, zoomChannelID, teamName, channelName, model.TeamPublic, model.ChannelStandard, idStore)
}

func CompleteMigration(teamID string) error {
	dst, err := teamdest.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("teams client: %w", err)
	}

	// list channels
	channels, err := dst.ListChannels(teamID)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}
	for _, channel := range channels {
		log.Printf("teams: channel found %s: %s", channel.ID, channel.Name)
		// complete migration for each channel
		if err := dst.CompleteMigrationChannel(teamID, channel.ID); err != nil {
			return fmt.Errorf("complete migration channel: %w", err)
		}
	}

	// complete migration for team
	if err := dst.CompleteMigrationTeam(teamID); err != nil {
		return fmt.Errorf("complete migration team: %w", err)
	}
	return nil
}
