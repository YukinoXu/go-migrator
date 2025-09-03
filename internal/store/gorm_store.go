package store

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"example.com/go-migrator/internal/model"
)

// GormStore implements Store using GORM and MySQL driver.
type GormStore struct {
	db *gorm.DB
}

type Task struct {
	ID        string    `gorm:"primaryKey;size:36"`
	Source    string    `gorm:"size:100;not null"`
	Target    string    `gorm:"size:100;not null;index:idx_target"`
	Payload   string    `gorm:"type:longtext"`
	Status    string    `gorm:"size:20;not null;index:idx_status"`
	Result    string    `gorm:"type:text"`
	Error     string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type Identity struct {
	ZoomUserID             string    `gorm:"primaryKey;size:100"`
	ZoomUserEmail          string    `gorm:"size:255"`
	ZoomUserDisplayName    string    `gorm:"size:255"`
	TeamsUserID            string    `gorm:"size:100;uniqueIndex:idx_teams_id"`
	TeamsUserPrincipalName string    `gorm:"size:255"`
	TeamsUserDisplayName   string    `gorm:"size:255"`
	CreatedAt              time.Time `gorm:"autoCreateTime"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime"`
}

// NewGormStore opens a GORM connection using the provided DSN and runs AutoMigrate.
func NewGormStore(dsn string) (*GormStore, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := RunMigrations(db); err != nil {
		return nil, err
	}
	return &GormStore{db: db}, nil
}

// NewGormStoreFromDB constructs a GormStore from an existing *gorm.DB. This is
// useful for tests (sqlite in-memory) or when the caller manages the DB.
func NewGormStoreFromDB(db *gorm.DB) (*GormStore, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if err := RunMigrations(db); err != nil {
		return nil, err
	}
	return &GormStore{db: db}, nil
}

// DB exposes the underlying *gorm.DB for callers that need low-level access
// (for example, running migrations or raw queries).
func (s *GormStore) DB() *gorm.DB { return s.db }

func RunMigrations(db *gorm.DB) error {
	mig := db.Migrator()

	if !mig.HasTable(&Task{}) {
		if err := mig.CreateTable(&Task{}); err != nil {
			return err
		}
	}
	if !mig.HasTable(&Identity{}) {
		if err := mig.CreateTable(&Identity{}); err != nil {
			return err
		}
	}

	if !mig.HasIndex(&Task{}, "idx_status") {
		if err := mig.CreateIndex(&Task{}, "idx_status"); err != nil {
			return err
		}
	}
	if !mig.HasIndex(&Task{}, "idx_target") {
		if err := mig.CreateIndex(&Task{}, "idx_target"); err != nil {
			return err
		}
	}

	if !mig.HasIndex(&Identity{}, "idx_teams_id") {
		if err := mig.CreateIndex(&Identity{}, "idx_teams_id"); err != nil {
			return err
		}
	}

	return nil
}

func (s *GormStore) CreateTask(t *model.Task) (string, error) {
	if t == nil {
		return "", errors.New("nil task")
	}
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = model.StatusPending
	}
	payloadB, _ := json.Marshal(t.Payload)
	gt := &Task{
		ID:        t.ID,
		Source:    t.Source,
		Target:    t.Target,
		Payload:   string(payloadB),
		Status:    string(t.Status),
		Result:    t.Result,
		Error:     t.Error,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
	if err := s.db.Create(gt).Error; err != nil {
		return "", err
	}
	return gt.ID, nil
}

func (s *GormStore) GetTask(id string) (*model.Task, error) {
	var gt Task
	if err := s.db.First(&gt, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var payload map[string]string
	if gt.Payload != "" {
		_ = json.Unmarshal([]byte(gt.Payload), &payload)
	}
	return &model.Task{
		ID:        gt.ID,
		Source:    gt.Source,
		Target:    gt.Target,
		Payload:   payload,
		Status:    model.TaskStatus(gt.Status),
		Result:    gt.Result,
		Error:     gt.Error,
		CreatedAt: gt.CreatedAt,
		UpdatedAt: gt.UpdatedAt,
	}, nil
}

func (s *GormStore) UpdateTask(t *model.Task) error {
	if t == nil || t.ID == "" {
		return errors.New("invalid task")
	}
	t.UpdatedAt = time.Now().UTC()
	payloadB, _ := json.Marshal(t.Payload)
	updates := map[string]interface{}{
		"source":     t.Source,
		"target":     t.Target,
		"payload":    string(payloadB),
		"status":     string(t.Status),
		"result":     t.Result,
		"error":      t.Error,
		"updated_at": t.UpdatedAt,
	}
	if err := s.db.Model(&Task{}).Where("id = ?", t.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormStore) ListTasks() ([]*model.Task, error) {
	var gts []Task
	if err := s.db.Find(&gts).Error; err != nil {
		return nil, err
	}
	out := make([]*model.Task, 0, len(gts))
	for _, gt := range gts {
		var payload map[string]string
		if gt.Payload != "" {
			_ = json.Unmarshal([]byte(gt.Payload), &payload)
		}
		out = append(out, &model.Task{
			ID:        gt.ID,
			Source:    gt.Source,
			Target:    gt.Target,
			Payload:   payload,
			Status:    model.TaskStatus(gt.Status),
			Result:    gt.Result,
			Error:     gt.Error,
			CreatedAt: gt.CreatedAt,
			UpdatedAt: gt.UpdatedAt,
		})
	}
	return out, nil
}

func (s *GormStore) CreateOrUpdateIdentity(i *model.Identity) error {
	if i == nil {
		return errors.New("nil identity")
	}
	gi := Identity{
		ZoomUserID:             i.ZoomUserID,
		ZoomUserEmail:          i.ZoomUserEmail,
		ZoomUserDisplayName:    i.ZoomUserDisplayName,
		TeamsUserID:            i.TeamsUserID,
		TeamsUserPrincipalName: i.TeamsUserPrincipalName,
		TeamsUserDisplayName:   i.TeamsUserDisplayName,
	}
	// upsert using ON CONFLICT to update fields when zoom_user_id exists
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "zoom_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"zoom_user_email", "zoom_user_display_name", "teams_user_id", "teams_user_principal_name", "teams_user_display_name", "updated_at"}),
	}).Create(&gi).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormStore) GetIdentityByZoomUserID(zoomUserID string) (*model.Identity, error) {
	var gi Identity
	if err := s.db.First(&gi, "zoom_user_id = ?", zoomUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &model.Identity{
		ZoomUserID:             gi.ZoomUserID,
		ZoomUserEmail:          gi.ZoomUserEmail,
		ZoomUserDisplayName:    gi.ZoomUserDisplayName,
		TeamsUserID:            gi.TeamsUserID,
		TeamsUserPrincipalName: gi.TeamsUserPrincipalName,
		TeamsUserDisplayName:   gi.TeamsUserDisplayName,
		CreatedAt:              gi.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              gi.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *GormStore) GetIdentityByTeamsUserID(teamsUserID string) (*model.Identity, error) {
	var gi Identity
	if err := s.db.First(&gi, "teams_user_id = ?", teamsUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &model.Identity{
		ZoomUserID:             gi.ZoomUserID,
		ZoomUserEmail:          gi.ZoomUserEmail,
		ZoomUserDisplayName:    gi.ZoomUserDisplayName,
		TeamsUserID:            gi.TeamsUserID,
		TeamsUserPrincipalName: gi.TeamsUserPrincipalName,
		TeamsUserDisplayName:   gi.TeamsUserDisplayName,
		CreatedAt:              gi.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              gi.UpdatedAt.Format(time.RFC3339),
	}, nil
}
