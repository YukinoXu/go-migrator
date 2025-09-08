package model

import "time"

type Project struct {
	ID                string    `gorm:"primaryKey;size:36" json:"project_id"`
	Name              string    `gorm:"size:128;not null" json:"name"`
	SourceConnectorID string    `gorm:"size:64;index:idx_project_source_connector" json:"source_connector_id"`
	TargetConnectorID string    `gorm:"size:64;index:idx_project_target_connector" json:"target_connector_id"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
