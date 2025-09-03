package store_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"example.com/go-migrator/internal/model"
	"example.com/go-migrator/internal/store"
	"example.com/go-migrator/internal/worker"
)

// fakeQueue implements queue.Client for tests (in-memory channel)
type fakeQueue struct {
	ch chan string
}

func newFakeQueue() *fakeQueue { return &fakeQueue{ch: make(chan string, 100)} }

func (f *fakeQueue) Publish(ctx context.Context, id string) error {
	select {
	case f.ch <- id:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fakeQueue) Consume(ctx context.Context) (<-chan string, error) {
	out := make(chan string)
	go func() {
		defer close(out)
		for {
			select {
			case id, ok := <-f.ch:
				if !ok {
					return
				}
				select {
				case out <- id:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func (f *fakeQueue) Close() error { close(f.ch); return nil }

// fakeStore is a simple in-test store implementation used when MYSQL_DSN is not set.
type fakeStore struct {
	mu    sync.RWMutex
	tasks map[string]*model.Task
	ids   map[string]*model.Identity
}

func newFakeStore() *fakeStore { return &fakeStore{tasks: make(map[string]*model.Task)} }

func (s *fakeStore) CreateTask(t *model.Task) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = fmt.Sprintf("t-%d", time.Now().UnixNano())
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = model.StatusPending
	}
	s.tasks[t.ID] = t
	return t.ID, nil
}

func (s *fakeStore) GetTask(id string) (*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return t, nil
}

func (s *fakeStore) UpdateTask(t *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return store.ErrNotFound
	}
	t.UpdatedAt = time.Now()
	s.tasks[t.ID] = t
	return nil
}

func (s *fakeStore) ListTasks() ([]*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out, nil
}

// Identity methods to satisfy store.Store in tests
func (s *fakeStore) CreateOrUpdateIdentity(i *model.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ids == nil {
		s.ids = make(map[string]*model.Identity)
	}
	s.ids[i.ZoomUserID] = i
	return nil
}

func (s *fakeStore) GetIdentityByZoomUserID(zoomUserID string) (*model.Identity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ids == nil {
		return nil, store.ErrNotFound
	}
	if v, ok := s.ids[zoomUserID]; ok {
		return v, nil
	}
	return nil, store.ErrNotFound
}

func (s *fakeStore) GetIdentityByTeamsUserID(teamsUserID string) (*model.Identity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ids == nil {
		return nil, store.ErrNotFound
	}
	for _, v := range s.ids {
		if v != nil && v.TeamsUserID == teamsUserID {
			return v, nil
		}
	}
	return nil, store.ErrNotFound
}

func TestEndToEnd(t *testing.T) {
	// decide which store to use
	var st store.Store
	dsn := os.Getenv("MYSQL_DSN")
	if dsn != "" {
		ms, err := store.NewGormStore(dsn)
		if err != nil {
			t.Fatalf("failed to open mysql store: %v", err)
		}
		st = ms
	} else {
		st = newFakeStore()
	}

	// use fakeQueue for worker consumption
	fq := newFakeQueue()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wk := worker.NewWorker(st, fq, 2)
	wk.Start(ctx)

	// create a task that should succeed
	task := &model.Task{Source: "zoom", Target: "teams", Payload: map[string]string{"conversation_id": "room-1"}}
	id, err := st.CreateTask(task)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// publish task id to queue
	if err := fq.Publish(ctx, id); err != nil {
		t.Fatalf("publish id: %v", err)
	}

	// wait up to 5s for completion
	dead := time.Now().Add(5 * time.Second)
	for time.Now().Before(dead) {
		tk, _ := st.GetTask(id)
		if tk.Status == model.StatusSuccess || tk.Status == model.StatusFailed {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task did not finish in time")
}
