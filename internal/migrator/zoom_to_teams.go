package migrator

import (
	"errors"
	"fmt"
	"time"

	"example.com/go-migrator/internal/models"
)

// Migrate performs a migration for the given task. This is a mock implementation
// that simulates migrating Zoom chat messages to Teams.
func Migrate(t *models.Task) error {
	if t.Source != "zoom" || t.Target != "teams" {
		return fmt.Errorf("no migrator for %s -> %s", t.Source, t.Target)
	}
	// simulate doing work
	time.Sleep(100 * time.Millisecond)

	// simulate failure for certain payloads
	if v, ok := t.Payload["conversation_id"]; ok && v == "fail-me" {
		return errors.New("simulated migration failure")
	}
	// normally here you'd call Zoom APIs to fetch messages and then call Teams APIs
	// to create messages; for this skeleton we just pretend it worked.
	return nil
}
