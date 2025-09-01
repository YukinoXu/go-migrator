package store_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"example.com/go-migrator/internal/store"
	models "example.com/go-migrator/internal/task"
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
	tasks map[string]*models.Task
}

func newFakeStore() *fakeStore { return &fakeStore{tasks: make(map[string]*models.Task)} }

func (s *fakeStore) CreateTask(t *models.Task) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = fmt.Sprintf("t-%d", time.Now().UnixNano())
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = models.StatusPending
	}
	s.tasks[t.ID] = t
	return t.ID, nil
}

func (s *fakeStore) GetTask(id string) (*models.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return t, nil
}

func (s *fakeStore) UpdateTask(t *models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return store.ErrNotFound
	}
	t.UpdatedAt = time.Now()
	s.tasks[t.ID] = t
	return nil
}

func (s *fakeStore) ListTasks() ([]*models.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out, nil
}

func TestEndToEnd(t *testing.T) {
	// decide which store to use
	var st store.Store
	dsn := os.Getenv("MYSQL_DSN")
	if dsn != "" {
		ms, err := store.NewMySQLStore(dsn)
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
	task := &models.Task{Source: "zoom", Target: "teams", Payload: map[string]string{"conversation_id": "room-1"}}
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
		if tk.Status == models.StatusSuccess || tk.Status == models.StatusFailed {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task did not finish in time")
}
