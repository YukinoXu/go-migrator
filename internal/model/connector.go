package model

import "time"

type ConnectorType string

const (
	Zoom  ConnectorType = "zoom"
	Teams ConnectorType = "teams"
)

type Connector struct {
	ID        string        `gorm:"primaryKey;size:36" json:"id"`
	UserID    string        `gorm:"size:64;index:idx_connector_user" json:"user_id"`
	Type      ConnectorType `gorm:"size:20;index:idx_connector_type" json:"type"`
	Data      string        `gorm:"type:text" json:"data"`
	CreatedAt time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}
