package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"example.com/go-migrator/internal/models"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

// MySQLStore implements Store backed by MySQL.
type MySQLStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewMySQLStore opens a connection and ensures schema exists. DSN should be in the
// format accepted by github.com/go-sql-driver/mysql, e.g. user:pass@tcp(127.0.0.1:3306)/dbname
func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &MySQLStore{db: db}
	if err := s.ensureSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *MySQLStore) ensureSchema() error {
	schema := `CREATE TABLE IF NOT EXISTS tasks (
  id VARCHAR(36) PRIMARY KEY,
  source VARCHAR(100) NOT NULL,
  target VARCHAR(100) NOT NULL,
  payload TEXT,
  status VARCHAR(20) NOT NULL,
  result TEXT,
  error TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);`
	_, err := s.db.Exec(schema)
	return err
}

// enqueuePending removed; queueing is handled by RabbitMQ client

func (s *MySQLStore) CreateTask(t *models.Task) (string, error) {
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
		t.Status = models.StatusPending
	}
	payloadB, _ := json.Marshal(t.Payload)
	_, err := s.db.Exec(`INSERT INTO tasks (id, source, target, payload, status, result, error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Source, t.Target, string(payloadB), string(t.Status), t.Result, t.Error, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return "", err
	}
	// publishing to queue is handled externally by the queue client
	return t.ID, nil
}

func (s *MySQLStore) GetTask(id string) (*models.Task, error) {
	row := s.db.QueryRow(`SELECT id, source, target, payload, status, result, error, created_at, updated_at FROM tasks WHERE id = ?`, id)
	var (
		t        models.Task
		payloadS sql.NullString
		result   sql.NullString
		errStr   sql.NullString
	)
	var createdAt, updatedAt time.Time
	if err := row.Scan(&t.ID, &t.Source, &t.Target, &payloadS, &t.Status, &result, &errStr, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.CreatedAt = createdAt
	t.UpdatedAt = updatedAt
	if payloadS.Valid {
		_ = json.Unmarshal([]byte(payloadS.String), &t.Payload)
	}
	if result.Valid {
		t.Result = result.String
	}
	if errStr.Valid {
		t.Error = errStr.String
	}
	return &t, nil
}

func (s *MySQLStore) UpdateTask(t *models.Task) error {
	if t == nil || t.ID == "" {
		return errors.New("invalid task")
	}
	t.UpdatedAt = time.Now().UTC()
	payloadB, _ := json.Marshal(t.Payload)
	_, err := s.db.Exec(`UPDATE tasks SET source=?, target=?, payload=?, status=?, result=?, error=?, updated_at=? WHERE id = ?`,
		t.Source, t.Target, string(payloadB), string(t.Status), t.Result, t.Error, t.UpdatedAt, t.ID)
	if err != nil {
		return err
	}
	return nil
}

func (s *MySQLStore) ListTasks() ([]*models.Task, error) {
	rows, err := s.db.Query(`SELECT id, source, target, payload, status, result, error, created_at, updated_at FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.Task{}
	for rows.Next() {
		var (
			t        models.Task
			payloadS sql.NullString
			result   sql.NullString
			errStr   sql.NullString
		)
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&t.ID, &t.Source, &t.Target, &payloadS, &t.Status, &result, &errStr, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		t.CreatedAt = createdAt
		t.UpdatedAt = updatedAt
		if payloadS.Valid {
			_ = json.Unmarshal([]byte(payloadS.String), &t.Payload)
		}
		if result.Valid {
			t.Result = result.String
		}
		if errStr.Valid {
			t.Error = errStr.String
		}
		out = append(out, &t)
	}
	return out, nil
}

// Enqueue and Queue removed; use external RabbitMQ client for queueing
