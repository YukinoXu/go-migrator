package model

import "time"

// Identity maps a Zoom user to a Teams user.
type Identity struct {
	ID                     uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ZoomUserID             string    `gorm:"size:64;index:idx_zoom_user_id" json:"zoom_user_id"`
	ZoomUserEmail          string    `gorm:"size:128;index:idx_zoom_user_email" json:"zoom_user_email"`
	ZoomUserDisplayName    string    `gorm:"size:128" json:"zoom_user_display_name"`
	TeamsUserID            string    `gorm:"size:64;index:idx_teams_user_id" json:"teams_user_id"`
	TeamsUserPrincipalName string    `gorm:"size:128;index:idx_teams_user_principal" json:"teams_user_principal_name"`
	TeamsUserDisplayName   string    `gorm:"size:128" json:"teams_user_display_name"`
	CreatedAt              time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
