package store

import (
	"example.com/go-migrator/internal/model"
	"gorm.io/gorm"
)

type ProjectStore struct {
	db *gorm.DB
}

func NewProjectStore(db *gorm.DB) *ProjectStore {
	return &ProjectStore{db: db}
}

func (s *ProjectStore) Create(project *model.Project) error {
	return s.db.Create(project).Error
}

func (s *ProjectStore) GetByID(id string) (*model.Project, error) {
	var project model.Project
	err := s.db.First(&project, "id = ?", id).Error
	return &project, err
}

func (s *ProjectStore) ListByConnector(connectorID string) ([]model.Project, error) {
	var projects []model.Project
	err := s.db.Where("source_connector_id = ? OR target_connector_id = ?", connectorID, connectorID).Find(&projects).Error
	return projects, err
}
