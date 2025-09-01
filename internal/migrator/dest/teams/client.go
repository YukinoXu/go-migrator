package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	migmodel "example.com/go-migrator/internal/migrator/model"
)

type Client struct {
	token string
}

func NewClientFromEnv() (*Client, error) {
	tok := os.Getenv("GRAPH_TOKEN")
	if tok == "" {
		return nil, fmt.Errorf("GRAPH_TOKEN not set")
	}
	return &Client{token: tok}, nil
}

func (c *Client) EnsureTeam(name string) (string, error) {
	// simplified: create team and return id
	url := "https://graph.microsoft.com/v1.0/teams"
	payload := map[string]any{"template@odata.bind": "https://graph.microsoft.com/v1.0/teamsTemplates('standard')", "displayName": name}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("graph create team error: %s: %s", resp.Status, string(body))
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if id, ok := out["id"].(string); ok {
		return id, nil
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		parts := strings.Split(loc, "/")
		return parts[len(parts)-1], nil
	}
	return "", nil
}

func (c *Client) EnsureChannel(teamID, name string) (string, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)
	payload := map[string]any{"displayName": name}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("graph create channel error: %s: %s", resp.Status, string(body))
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *Client) PostMessage(teamID, channelID string, m migmodel.Message) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages", teamID, channelID)
	payload := map[string]any{"body": map[string]string{"content": fmt.Sprintf("%s: %s", m.From, m.Content)}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("graph post message error: %s: %s", resp.Status, string(body))
	}
	return nil
}
