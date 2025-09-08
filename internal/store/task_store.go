package store

import (
	"example.com/go-migrator/internal/model"
	"gorm.io/gorm"
)

type TaskStore struct {
	db *gorm.DB
}

func NewTaskStore(db *gorm.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) Create(task *model.Task) error {
	return s.db.Create(task).Error
}

func (s *TaskStore) GetByID(id string) (*model.Task, error) {
	var task model.Task
	err := s.db.First(&task, "id = ?", id).Error
	return &task, err
}

func (s *TaskStore) ListByProject(projectID, status string) ([]model.Task, error) {
	var tasks []model.Task
	query := s.db.Where("project_id = ?", projectID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (s *TaskStore) UpdateStatus(id, status string) error {
	return s.db.Model(&model.Task{}).Where("id = ?", id).Update("status", status).Error
}
