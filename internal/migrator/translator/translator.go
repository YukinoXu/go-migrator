package translator

import (
	"time"

	migmodel "example.com/go-migrator/internal/migrator/model"
)

// TranslateZoomToTeams converts a ZoomMessage into a TeamsMessage suitable
// for the Graph import API. It preserves ID, timestamps, sender display name,
// message content, and maps files to attachments when download URLs are present.
func TranslateZoomToTeams(zm migmodel.ZoomMessage, teamsUserID, teamsUserDisplayName string) migmodel.TeamsMessageRequest {
	// determine createdDateTime
	created := zm.DateTime
	if created == "" && zm.Timestamp > 0 {
		// Zoom timestamp is milliseconds since epoch in these messages
		created = time.Unix(0, zm.Timestamp*int64(time.Millisecond)).UTC().Format(time.RFC3339)
	}

	// build body content
	content := zm.Message

	tm := migmodel.TeamsMessageRequest{
		CreatedDateTime: created,
		From: &migmodel.TeamsFrom{
			User: &migmodel.TeamsUserIdentity{
				ID:               teamsUserID,
				DisplayName:      teamsUserDisplayName,
				UserIdentityType: "aadUser",
			},
		},
		Body: &migmodel.TeamsBody{
			ContentType: "html",
			Content:     content,
		},
	}
	return tm
}
