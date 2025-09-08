package store

import (
	"example.com/go-migrator/internal/model"
	"gorm.io/gorm"
)

type IdentityStore struct {
	db *gorm.DB
}

func NewIdentityStore(db *gorm.DB) *IdentityStore {
	return &IdentityStore{db: db}
}

func (s *IdentityStore) Create(identity *model.Identity) error {
	return s.db.Create(identity).Error
}

func (s *IdentityStore) GetByZoomID(zoomID string) (*model.Identity, error) {
	var identity model.Identity
	err := s.db.First(&identity, "zoom_user_id = ?", zoomID).Error
	return &identity, err
}

func (s *IdentityStore) GetByTeamsID(teamsID string) (*model.Identity, error) {
	var identity model.Identity
	err := s.db.First(&identity, "teams_user_id = ?", teamsID).Error
	return &identity, err
}
