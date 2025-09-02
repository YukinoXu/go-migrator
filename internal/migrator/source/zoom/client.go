package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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

func (c *Client) GetUsers() ([]migmodel.ZoomUser, error) {
	ctx := context.Background()
	url := "https://api.zoom.us/v2/users"
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
		Users []migmodel.ZoomUser `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Users, nil
}

func (c *Client) GetUserChannels(userID string) ([]migmodel.ZoomChannel, error) {
	ctx := context.Background()
	url := fmt.Sprintf("https://api.zoom.us/v2/chat/users/%s/channels", userID)
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
		Channels []migmodel.ZoomChannel `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Channels, nil
}

func (c *Client) FetchMessages(userID string, channelID string) ([]migmodel.ZoomMessage, error) {
	ctx := context.Background()

	defaultFrom := "1970-01-01T00:00:00Z"
	url := fmt.Sprintf("https://api.zoom.us/v2/chat/users/%s/messages?to_channel=%s&from=%s&page_size=50", userID, channelID, defaultFrom)

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

	var parsed migmodel.ZoomMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	return parsed.Messages, nil
}
