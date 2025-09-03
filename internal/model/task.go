package model

import "time"

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Target    string            `json:"target"`
	Payload   map[string]string `json:"payload"`
	Status    TaskStatus        `json:"status"`
	Result    string            `json:"result,omitempty"`
	Error     string            `json:"error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
