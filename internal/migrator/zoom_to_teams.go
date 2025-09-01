package migrator

import (
	"fmt"

	teamdest "example.com/go-migrator/internal/migrator/dest/teams"
	zoomsrc "example.com/go-migrator/internal/migrator/source/zoom"
)

// MigrateTask is a thin adapter used by the worker: it instantiates provider clients
// from environment and runs the orchestrator.
func MigrateTask(convID string, opts map[string]string) error {
	src, err := zoomsrc.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("zoom client: %w", err)
	}
	dst, err := teamdest.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("teams client: %w", err)
	}
	orchestrator := NewOrchestrator(src, dst)
	teamName := opts["team_name"]
	if teamName == "" {
		teamName = fmt.Sprintf("Migrated-%s", convID)
	}
	channelName := opts["channel_name"]
	if channelName == "" {
		channelName = "general"
	}
	return orchestrator.Run(convID, teamName, channelName)
}
