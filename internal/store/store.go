package store

import (
	"errors"

	models "example.com/go-migrator/internal/task"
)

var ErrNotFound = errors.New("task not found")

// Store provides task persistence and a queue for processing.
type Store interface {
	CreateTask(t *models.Task) (string, error)
	GetTask(id string) (*models.Task, error)
	UpdateTask(t *models.Task) error
	ListTasks() ([]*models.Task, error)
}

// Identity maps a Zoom user to a Teams user.
type Identity struct {
	ZoomUserID             string
	ZoomUserEmail          string
	ZoomUserDisplayName    string
	TeamsUserID            string
	TeamsUserPrincipalName string
	TeamsUserDisplayName   string
	CreatedAt              string
	UpdatedAt              string
}

// Identity methods
type IdentityStore interface {
	CreateOrUpdateIdentity(i *Identity) error
	GetIdentityByZoomUserID(zoomUserID string) (*Identity, error)
	GetIdentityByTeamsUserID(teamsUserID string) (*Identity, error)
}

// The in-memory store has been removed. Use a persistent store (MySQL) and
// RabbitMQ for queueing. The MySQL store implementation persists tasks.
