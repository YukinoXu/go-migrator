package migrator

import (
	"fmt"

	teamdest "example.com/go-migrator/internal/migrator/dest/teams"
	zoomsrc "example.com/go-migrator/internal/migrator/source/zoom"

	"example.com/go-migrator/internal/migrator/model"
	"example.com/go-migrator/internal/store"
)

// MigrateTask is a thin adapter used by the worker: it instantiates provider clients
// from environment and runs the orchestrator. It accepts an IdentityStore so
// the orchestrator can resolve user mappings.
func MigrateTask(zoomUserID, zoomChannelID, teamName, channelName string, idStore store.IdentityStore) error {
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
