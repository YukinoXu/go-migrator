package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	tenantID := os.Getenv("TEAMS_TENANT_ID")
	clientID := os.Getenv("TEAMS_CLIENT_ID")
	clientSecret := os.Getenv("TEAMS_CLIENT_SECRET")

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")

	log.Printf("teams: requesting token POST %s", tokenURL)
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		log.Printf("teams: token request failed %s: %s", resp.Status, bodyStr)
		return nil, fmt.Errorf("failed to obtain token: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	log.Printf("teams: obtained token (len=%d)", len(result["access_token"].(string)))
	return &Client{token: result["access_token"].(string)}, nil
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
	log.Printf("teams: POST %s (create team %q)", url, name)
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
		if !strings.HasPrefix(loc, "http") {
			loc = "https://graph.microsoft.com/v1.0" + loc
		}
		log.Printf("teams: async create accepted, polling %s", loc)
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
				log.Printf("teams: polling operation %s", loc)
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
							log.Printf("teams: async create succeeded targetResourceId=%s", tr)
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
							respOp.Body.Close()
							log.Printf("teams: async create completed with no id")
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
									log.Printf("teams: async operation failed code=%s msg=%s", code, msg)
									return "", fmt.Errorf("teams async operation failed: %s: %s", code, msg)
								}
								respOp.Body.Close()
								log.Printf("teams: async operation failed msg=%s", msg)
								return "", fmt.Errorf("teams async operation failed: %s", msg)
							}
							respOp.Body.Close()
							log.Printf("teams: async operation failed")
							return "", fmt.Errorf("teams async operation failed")
						default:
							// NotStarted, Running -> keep polling
						}
					}
				}
				respOp.Body.Close()
			}
		}
	}

	// fallback: try to parse Location header if set
	if loc := resp.Header.Get("Location"); loc != "" {
		log.Printf("teams: create returned Location %s", loc)
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
	log.Printf("teams: POST %s (create channel %q)", url, name)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("teams: create channel failed %s: %s", resp.Status, string(body))
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

func (c *Client) PostMessage(teamID, channelID string, tm migmodel.TeamsMessageRequest) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages", teamID, channelID)

	b, _ := json.Marshal(tm)
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
		log.Printf("teams: import message failed %s: %s", resp.Status, string(body))
		return fmt.Errorf("graph import message error: %s: %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) AddMemberToTeam(teamID, userID string, owner bool) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/members", teamID)

	member := NewTeamsGraphMember(userID, owner)
	b, _ := json.Marshal(member)
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
		log.Printf("teams: add member to team failed %s: %s", resp.Status, string(body))
		return fmt.Errorf("graph add member to team error: %s: %s", resp.Status, string(body))
	}
	log.Printf("teams: added member %s to team %s", userID, teamID)
	return nil
}

func (c *Client) CompleteMigrationChannel(teamID, channelID string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/completeMigration", teamID, channelID)

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("teams: complete migration channel failed %s: %s", resp.Status, string(body))
		return fmt.Errorf("graph complete migration channel error: %s: %s", resp.Status, string(body))
	}
	log.Printf("teams: completed migration for channel %s in team %s", channelID, teamID)
	return nil
}

func (c *Client) CompleteMigrationTeam(teamID string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/completeMigration", teamID)

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("teams: complete migration team failed %s: %s", resp.Status, string(body))
		return fmt.Errorf("graph complete migration team error: %s: %s", resp.Status, string(body))
	}
	log.Printf("teams: completed migration for team %s", teamID)
	return nil
}

func (c *Client) ListChannels(teamID string) ([]migmodel.TeamsChannel, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("teams: list channels failed %s: %s", resp.Status, string(body))
		return nil, fmt.Errorf("graph list channels error: %s: %s", resp.Status, string(body))
	}

	var channelResponse migmodel.TeamsChannelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&channelResponse); err != nil {
		return nil, fmt.Errorf("decode channels response: %w", err)
	}

	channels := channelResponse.Value
	return channels, nil
}

func NewTeamsGraphMember(userID string, owner bool) *migmodel.TeamsGraphMember {
	var roles []string
	if owner {
		roles = []string{"owner"}
	} else {
		roles = []string{}
	}
	return &migmodel.TeamsGraphMember{
		ODataType:     "#microsoft.graph.aadUserConversationMember",
		Roles:         roles,
		UserODataBind: fmt.Sprintf("https://graph.microsoft.com/v1.0/users('%s')", userID),
	}
}
