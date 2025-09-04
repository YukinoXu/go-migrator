package translator

import (
	"testing"

	migmodel "example.com/go-migrator/internal/migrator/model"
)

func TestTranslateZoomToTeams_Minimal(t *testing.T) {
	zm := migmodel.ZoomMessage{
		ID:                "msg-1",
		Message:           "Hello from Zoom",
		Sender:            "zoom.user",
		SenderDisplayName: "Zoom User",
		Timestamp:         1693728000000, // 2023-09-03T00:00:00Z
	}
	tm := TranslateZoomToTeams(zm, "teams-user-1", "Teams User")
	if tm.Body == nil || tm.Body.Content == "" {
		t.Fatalf("expected body content set")
	}
	// verify created date was converted from milliseconds (date only to avoid TZ issues)
	if len(tm.CreatedDateTime) < 10 || tm.CreatedDateTime[:10] != "2023-09-03" {
		t.Fatalf("unexpected created date: %s", tm.CreatedDateTime)
	}
	// verify the body content equals zoom message
	expectedBody := "Hello from Zoom"
	if tm.Body.Content != expectedBody {
		t.Fatalf("unexpected body content. want=%q got=%q", expectedBody, tm.Body.Content)
	}
	// verify From user identity was set from args
	if tm.From == nil || tm.From.User == nil {
		t.Fatalf("expected From.User to be set")
	}
	if tm.From.User.DisplayName != "Teams User" {
		t.Fatalf("unexpected From.User.DisplayName: %s", tm.From.User.DisplayName)
	}
}

func TestTranslateZoomToTeams_WithFiles(t *testing.T) {
	zm := migmodel.ZoomMessage{
		ID:      "msg-2",
		Message: "File attached",
		Sender:  "u2",
		Files: []migmodel.ZoomFile{
			{FileID: "f1", FileName: "doc.txt", DownloadURL: "https://example.com/doc.txt"},
		},
	}
	tm := TranslateZoomToTeams(zm, "teams-user-2", "Teams User 2")
	if tm.From == nil || tm.From.User == nil || tm.From.User.ID != "teams-user-2" {
		t.Fatalf("expected From.User.ID to equal teams-user-2 got %v", tm.From)
	}
}
