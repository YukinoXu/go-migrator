package store

import (
	"example.com/go-migrator/internal/model"
	"gorm.io/gorm"
)

type ConnectorStore struct {
	db *gorm.DB
}

func NewConnectorStore(db *gorm.DB) *ConnectorStore {
	return &ConnectorStore{db: db}
}

func (s *ConnectorStore) Create(connector *model.Connector) error {
	return s.db.Create(connector).Error
}

func (s *ConnectorStore) GetByID(id string) (*model.Connector, error) {
	var connector model.Connector
	err := s.db.First(&connector, "id = ?", id).Error
	return &connector, err
}

func (s *ConnectorStore) GetByUserAndType(userID string, ctype model.ConnectorType) (*model.Connector, error) {
	var connector model.Connector
	err := s.db.First(&connector, "user_id = ? AND type = ?", userID, ctype).Error
	return &connector, err
}
