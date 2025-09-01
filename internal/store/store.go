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

// The in-memory store has been removed. Use a persistent store (MySQL) and
// RabbitMQ for queueing. The MySQL store implementation persists tasks.
