package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	migmodel "example.com/go-migrator/internal/migrator/model"
)

const defaultCreatedDateTime = "2010-01-01T00:00:00.000Z"

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

func (c *Client) EnsureTeam(name string, t migmodel.TeamType) (string, error) {
	// simplified: create team and return id
	url := "https://graph.microsoft.com/v1.0/teams"

	var visibility string
	switch t {
	case migmodel.TeamPublic:
		visibility = "public"
	case migmodel.TeamPrivate:
		visibility = "private"
	}

	payload := map[string]any{
		"@microsoft.graph.teamCreationMode": "migration",
		"template@odata.bind":               "https://graph.microsoft.com/v1.0/teamsTemplates('standard')",
		"displayName":                       name,
		"visibility":                        visibility,
		"createdDateTime":                   defaultCreatedDateTime,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Successful synchronous creation (201/200)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		if id, ok := out["id"].(string); ok {
			return id, nil
		}
	}

	// Accepted - creation is processed async. Poll the Location until it completes.
	if resp.StatusCode == http.StatusAccepted {
		loc := resp.Header.Get("Location")
		if loc == "" {
			return "", fmt.Errorf("graph returned 202 but no Location header")
		}
		// Poll the operation resource until it reports completion or times out.
		timeout := time.After(60 * time.Second)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-timeout:
				return "", fmt.Errorf("timed out waiting for team creation")
			case <-ticker.C:
				reqOp, _ := http.NewRequest("GET", loc, nil)
				reqOp.Header.Set("Authorization", "Bearer "+c.token)
				respOp, err := http.DefaultClient.Do(reqOp)
				if err != nil {
					// continue polling on transient errors
					continue
				}
				if respOp.StatusCode == 200 {
					var op map[string]any
					if err := json.NewDecoder(respOp.Body).Decode(&op); err == nil {
						// operation may include targetResourceId or resourceLocation or status
						if tr, ok := op["targetResourceId"].(string); ok && tr != "" {
							respOp.Body.Close()
							return tr, nil
						}
						statusRaw, _ := op["status"].(string)
						status := strings.ToLower(statusRaw)
						// teamsAsyncOperationStatus: NotStarted, Running, Succeeded, Failed, UnknownFutureValue
						switch status {
						case "succeeded":
							if tr, ok := op["targetResourceId"].(string); ok && tr != "" {
								respOp.Body.Close()
								return tr, nil
							}
							if rid, ok := op["resourceLocation"].(string); ok && rid != "" {
								parts := strings.Split(rid, "/")
								respOp.Body.Close()
								return parts[len(parts)-1], nil
							}
							if loc2 := respOp.Header.Get("Location"); loc2 != "" {
								parts := strings.Split(loc2, "/")
								respOp.Body.Close()
								return parts[len(parts)-1], nil
							}
							respOp.Body.Close()
							return "", nil
						case "failed":
							// surface operation error if available
							if errObj, ok := op["error"].(map[string]any); ok {
								var msg string
								if m, ok := errObj["message"].(string); ok {
									msg = m
								}
								if code, ok := errObj["code"].(string); ok {
									respOp.Body.Close()
									return "", fmt.Errorf("teams async operation failed: %s: %s", code, msg)
								}
								respOp.Body.Close()
								return "", fmt.Errorf("teams async operation failed: %s", msg)
							}
							respOp.Body.Close()
							return "", fmt.Errorf("teams async operation failed")
						default:
							// NotStarted, Running, UnknownFutureValue -> keep polling
						}
					}
				}
				respOp.Body.Close()
			}
		}
	}

	// fallback: try to parse Location header if set
	if loc := resp.Header.Get("Location"); loc != "" {
		parts := strings.Split(loc, "/")
		return parts[len(parts)-1], nil
	}
	return "", nil
}

func (c *Client) EnsureChannel(teamID, name string, chType migmodel.ChannelType) (string, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)

	var membershipType string
	switch chType {
	case migmodel.ChannelStandard:
		membershipType = "standard"
	case migmodel.ChannelPrivate:
		membershipType = "private"
	case migmodel.ChannelShared:
		membershipType = "shared"
	}

	payload := map[string]any{
		"@microsoft.graph.channelCreationMode": "migration",
		"displayName":                          name,
		"membershipType":                       membershipType,
		"createdDateTime":                      defaultCreatedDateTime,
	}
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

func (c *Client) PostMessage(teamID, channelID string, zm migmodel.ZoomMessage) error {
	// Use the import messages API to preserve original sender/timestamp where possible.
	// Endpoint: POST /teams/{team-id}/channels/{channel-id}/messages/import
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages/import", teamID, channelID)
	// Build a minimal message import payload. We use "application" as the sender when
	// we don't have an Azure AD user id. The API expects messageId and createdDateTime.
	var createdDateTime string
	if zm.DateTime != "" {
		createdDateTime = zm.DateTime
	} else if zm.Timestamp > 0 {
		createdDateTime = time.Unix(0, zm.Timestamp*int64(time.Millisecond)).UTC().Format(time.RFC3339)
	} else {
		createdDateTime = defaultCreatedDateTime
	}

	display := zm.SenderDisplayName
	if display == "" && zm.Sender != "" {
		display = zm.Sender
	}

	// The import API expects the message body; include sender displayName for readability.
	msg := map[string]any{
		"messageId":       zm.ID,
		"createdDateTime": createdDateTime,
		"body": map[string]any{
			"contentType": "html",
			"content":     fmt.Sprintf("%s: %s", display, zm.Message),
		},
		"from": map[string]any{
			"application": map[string]any{
				"id":          "importer",
				"displayName": display,
			},
		},
	}

	payload := map[string]any{"messages": []any{msg}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// The import API may return 200/201 or 202 depending on processing. Treat non-2xx as error.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("graph import message error: %s: %s", resp.Status, string(body))
	}
	return nil
}
