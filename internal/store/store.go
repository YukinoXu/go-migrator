package store

import (
	"errors"

	"example.com/go-migrator/internal/model"
)

var ErrNotFound = errors.New("task not found")

// Store provides task persistence and a queue for processing.
type Store interface {
	CreateTask(t *model.Task) (string, error)
	GetTask(id string) (*model.Task, error)
	UpdateTask(t *model.Task) error
	ListTasks() ([]*model.Task, error)
	CreateOrUpdateIdentity(i *model.Identity) error
	GetIdentityByZoomUserID(zoomUserID string) (*model.Identity, error)
	GetIdentityByTeamsUserID(teamsUserID string) (*model.Identity, error)
}
