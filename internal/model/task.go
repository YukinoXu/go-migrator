package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID         string     `gorm:"primaryKey;size:36" json:"id"`
	ProjectID  string     `gorm:"size:64;index:idx_task_project_status,priority:1" json:"project_id"`
	SourcePath string     `gorm:"size:255;uniqueIndex:uq_task_source_path" json:"source_path"`
	TargetPath string     `gorm:"size:255" json:"target_path"`
	Status     TaskStatus `gorm:"size:20;index:idx_task_status;index:idx_task_project_status,priority:2" json:"status"`
	CreatedAt  time.Time  `gorm:"index:idx_task_created_at" json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// BeforeCreate is a GORM hook that ensures a UUID is assigned to Task.ID
func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.Status == "" {
		t.Status = StatusPending
	}
	return nil
}
