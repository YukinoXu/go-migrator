package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"example.com/go-migrator/internal/model"
)

func setupInMemoryStore(t *testing.T) *GormStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := NewGormStoreFromDB(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}

func TestGormStoreTaskLifecycle(t *testing.T) {
	s := setupInMemoryStore(t)

	tk := &model.Task{Source: "zoom", Target: "teams", Payload: map[string]string{"conversation_id": "room-1"}}
	id, err := s.CreateTask(tk)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == "" {
		t.Fatalf("empty id")
	}
	got, err := s.GetTask(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Source != "zoom" || got.Target != "teams" {
		t.Fatalf("unexpected fields")
	}
	// update
	got.Status = model.StatusSuccess
	got.Result = "ok"
	if err := s.UpdateTask(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	later, err := s.GetTask(id)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if later.Status != model.StatusSuccess {
		t.Fatalf("status not updated")
	}
}

func TestGormStoreIdentityUpsertAndLookup(t *testing.T) {
	s := setupInMemoryStore(t)

	i := &model.Identity{
		ZoomUserID:          "z-1",
		ZoomUserEmail:       "a@x.com",
		ZoomUserDisplayName: "A",
		TeamsUserID:         "t-1",
	}
	if err := s.CreateOrUpdateIdentity(i); err != nil {
		t.Fatalf("create identity: %v", err)
	}
	got, err := s.GetIdentityByZoomUserID("z-1")
	if err != nil {
		t.Fatalf("get by zoom id: %v", err)
	}
	if got.TeamsUserID != "t-1" {
		t.Fatalf("teams id mismatch")
	}
	// update teams id via upsert
	i.TeamsUserID = "t-2"
	if err := s.CreateOrUpdateIdentity(i); err != nil {
		t.Fatalf("upsert identity: %v", err)
	}
	got2, err := s.GetIdentityByZoomUserID("z-1")
	if err != nil {
		t.Fatalf("get after upsert: %v", err)
	}
	if got2.TeamsUserID != "t-2" {
		t.Fatalf("upsert didn't update teams id")
	}
	// lookup by teams id
	if _, err := s.GetIdentityByTeamsUserID("t-2"); err != nil {
		t.Fatalf("get by teams id: %v", err)
	}
	// ensure CreatedAt/UpdatedAt are set
	if got2.CreatedAt == "" || got2.UpdatedAt == "" {
		t.Fatalf("timestamps not set")
	}
	// sanity check payload marshalling in tasks
	tk := &model.Task{Source: "zoom", Target: "teams", Payload: map[string]string{"k": "v"}}
	id, err := s.CreateTask(tk)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	r, _ := s.GetTask(id)
	b, _ := json.Marshal(r.Payload)
	if string(b) != "{\"k\":\"v\"}" {
		t.Fatalf("payload mismatch: %s", string(b))
	}
	// time fields sanity
	if time.Since(r.CreatedAt) > time.Hour || time.Since(r.UpdatedAt) > time.Hour {
		t.Fatalf("timestamps weird")
	}
}
