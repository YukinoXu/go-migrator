package store

import (
	"errors"

	"example.com/go-migrator/internal/model"
	"gorm.io/gorm"
)

// ========================
// 接口定义
// ========================

var ErrNotFound = errors.New("task not found")

type TaskStoreInterface interface {
	Create(task *model.Task) error
	GetByID(id string) (*model.Task, error)
	ListByProject(projectID, status string) ([]model.Task, error)
	UpdateStatus(id, status string) error
}

type IdentityStoreInterface interface {
	Create(identity *model.Identity) error
	GetByZoomID(zoomID string) (*model.Identity, error)
	GetByTeamsID(teamsID string) (*model.Identity, error)
}

type ProjectStoreInterface interface {
	Create(project *model.Project) error
	GetByID(id string) (*model.Project, error)
	ListByConnector(connectorID string) ([]model.Project, error)
}

type ConnectorStoreInterface interface {
	Create(connector *model.Connector) error
	GetByID(id string) (*model.Connector, error)
	GetByUserAndType(userID string, ctype model.ConnectorType) (*model.Connector, error)
}

// ========================
// StoreManager 统一管理
// ========================

type StoreManager struct {
	Task      TaskStoreInterface
	Identity  IdentityStoreInterface
	Project   ProjectStoreInterface
	Connector ConnectorStoreInterface
}

// NewStoreManager 初始化所有 Store
func NewStoreManager(db *gorm.DB) *StoreManager {
	return &StoreManager{
		Task:      NewTaskStore(db),
		Identity:  NewIdentityStore(db),
		Project:   NewProjectStore(db),
		Connector: NewConnectorStore(db),
	}
}
