package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	migmodel "example.com/go-migrator/internal/migrator/model"
)

type Client struct {
	token string
}

func NewClientFromEnv() (*Client, error) {
	tok := os.Getenv("ZOOM_TOKEN")
	if tok == "" {
		return nil, fmt.Errorf("ZOOM_TOKEN not set")
	}
	return &Client{token: tok}, nil
}

func (c *Client) FetchMessages(conversationID string) ([]migmodel.Message, error) {
	ctx := context.Background()
	url := fmt.Sprintf("https://api.zoom.us/v2/chat/conversations/%s/messages", conversationID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("zoom api error: %s: %s", resp.Status, string(b))
	}
	var parsed struct {
		Messages []struct {
			ID        string `json:"id"`
			Sender    string `json:"sender"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	out := make([]migmodel.Message, 0, len(parsed.Messages))
	for _, m := range parsed.Messages {
		mm := migmodel.Message{ID: m.ID, From: m.Sender, Content: m.Message}
		if m.Timestamp > 0 {
			mm.Timestamp = time.UnixMilli(m.Timestamp)
		}
		out = append(out, mm)
	}
	return out, nil
}
