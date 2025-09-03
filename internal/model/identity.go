package model

// Identity maps a Zoom user to a Teams user.
type Identity struct {
	ZoomUserID             string `json:"zoom_user_id"`
	ZoomUserEmail          string `json:"zoom_user_email"`
	ZoomUserDisplayName    string `json:"zoom_user_display_name"`
	TeamsUserID            string `json:"teams_user_id"`
	TeamsUserPrincipalName string `json:"teams_user_principal_name"`
	TeamsUserDisplayName   string `json:"teams_user_display_name"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}
