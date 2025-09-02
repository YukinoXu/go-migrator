package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	models "example.com/go-migrator/internal/task"
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
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// identities table for mapping Zoom users to Teams users
	idSchema := `CREATE TABLE IF NOT EXISTS identities (
	zoom_user_id VARCHAR(100) PRIMARY KEY,
	zoom_user_email VARCHAR(255),
	zoom_user_display_name VARCHAR(255),
	teams_user_id VARCHAR(100),
	teams_user_principal_name VARCHAR(255),
	teams_user_display_name VARCHAR(255),
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);`
	_, err := s.db.Exec(idSchema)
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

// CreateOrUpdateIdentity inserts or updates an identity row keyed by zoom_id.
func (s *MySQLStore) CreateOrUpdateIdentity(i *Identity) error {
	if i == nil {
		return errors.New("nil identity")
	}
	now := time.Now().UTC()
	// try update first
	_, err := s.db.Exec(`UPDATE identities SET zoom_user_email=?, zoom_user_display_name=?, teams_user_id=?, teams_user_principal_name=?, teams_user_display_name=?, updated_at=? WHERE zoom_user_id = ?`,
		i.ZoomUserEmail, i.ZoomUserDisplayName, i.TeamsUserID, i.TeamsUserPrincipalName, i.TeamsUserDisplayName, now, i.ZoomUserID)
	if err != nil {
		return err
	}
	// attempt insert if not exists
	_, err = s.db.Exec(`INSERT INTO identities (zoom_user_id, zoom_user_email, zoom_user_display_name, teams_user_id, teams_user_principal_name, teams_user_display_name, created_at, updated_at) SELECT ?,?,?,?,?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM identities WHERE zoom_user_id = ?)`,
		i.ZoomUserID, i.ZoomUserEmail, i.ZoomUserDisplayName, i.TeamsUserID, i.TeamsUserPrincipalName, i.TeamsUserDisplayName, now, now, i.ZoomUserID)
	return err
}

func scanIdentityRow(rows *sql.Row) (*Identity, error) {
	var id Identity
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&id.ZoomUserID, &id.ZoomUserEmail, &id.ZoomUserDisplayName, &id.TeamsUserID, &id.TeamsUserPrincipalName, &id.TeamsUserDisplayName, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	id.CreatedAt = createdAt.String()
	id.UpdatedAt = updatedAt.String()
	return &id, nil
}

func (s *MySQLStore) GetIdentityByZoomID(zoomID string) (*Identity, error) {
	row := s.db.QueryRow(`SELECT zoom_user_id, zoom_user_email, zoom_user_display_name, teams_user_id, teams_user_principal_name, teams_user_display_name, created_at, updated_at FROM identities WHERE zoom_user_id = ?`, zoomID)
	return scanIdentityRow(row)
}

func (s *MySQLStore) GetIdentityByTeamsID(teamsID string) (*Identity, error) {
	row := s.db.QueryRow(`SELECT zoom_user_id, zoom_user_email, zoom_user_display_name, teams_user_id, teams_user_principal_name, teams_user_display_name, created_at, updated_at FROM identities WHERE teams_user_id = ?`, teamsID)
	return scanIdentityRow(row)
}

// Enqueue and Queue removed; use external RabbitMQ client for queueing
