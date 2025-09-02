package model

type ZoomUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"name"`
	Email       string `json:"email"`
}

type ZoomChannel struct {
	ID          string `json:"id"`
	JID         string `json:"jid"`
	DisplayName string `json:"name"`
}

// Zoom API models for the chat messages endpoint response. These mirror the
// fields returned by Zoom so the JSON can be unmarshaled directly.
type ZoomFile struct {
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// (No canonical Message type - we post ZoomMessage directly to DestinationClient)
// ZoomMessage represents a single chat message returned by Zoom chat APIs.
type ZoomMessage struct {
	ID                string     `json:"id"`
	Message           string     `json:"message"`
	Sender            string     `json:"sender"`
	SendMemberID      string     `json:"send_member_id"`
	SenderDisplayName string     `json:"sender_display_name"`
	DateTime          string     `json:"date_time"` // RFC3339 timestamp string
	Timestamp         int64      `json:"timestamp"`
	MessageType       string     `json:"message_type"`
	CustomEmoji       bool       `json:"custom_emoji"`
	Files             []ZoomFile `json:"files,omitempty"`
	// Zoom sometimes includes top-level file fields duplicated for convenience
	FileID      string            `json:"file_id,omitempty"`
	FileName    string            `json:"file_name,omitempty"`
	FileSize    int64             `json:"file_size,omitempty"`
	DownloadURL string            `json:"download_url,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

type ZoomMessagesResponse struct {
	From          string        `json:"from"`
	To            string        `json:"to"`
	PageSize      int           `json:"page_size"`
	NextPageToken string        `json:"next_page_token"`
	Messages      []ZoomMessage `json:"messages"`
}

// TeamType represents whether a team is public or private.
type TeamType string

const (
	TeamPublic  TeamType = "public"
	TeamPrivate TeamType = "private"
)

// ChannelType represents types of Teams channels.
type ChannelType string

const (
	ChannelStandard ChannelType = "standard"
	ChannelPrivate  ChannelType = "private"
	ChannelShared   ChannelType = "shared"
)

// Teams models to mirror the Graph message shape used by Teams APIs.
type TeamsUserIdentity struct {
	ODataType        string `json:"@odata.type,omitempty"`
	ID               string `json:"id,omitempty"`
	DisplayName      string `json:"displayName,omitempty"`
	UserIdentityType string `json:"userIdentityType,omitempty"`
	TenantID         string `json:"tenantId,omitempty"`
}

type TeamsFrom struct {
	Application interface{}        `json:"application,omitempty"`
	Device      interface{}        `json:"device,omitempty"`
	User        *TeamsUserIdentity `json:"user,omitempty"`
}

type TeamsBody struct {
	ContentType string `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"`
}

type TeamsChannelIdentity struct {
	TeamID    string `json:"teamId,omitempty"`
	ChannelID string `json:"channelId,omitempty"`
}

type TeamsAttachment struct {
	ID           string  `json:"id,omitempty"`
	ContentType  string  `json:"contentType,omitempty"`
	ContentURL   *string `json:"contentUrl,omitempty"`
	Content      string  `json:"content,omitempty"`
	Name         *string `json:"name,omitempty"`
	ThumbnailURL *string `json:"thumbnailUrl,omitempty"`
}

type TeamsMessage struct {
	ODataContext         string                `json:"@odata.context,omitempty"`
	ID                   string                `json:"id,omitempty"`
	ReplyToID            *string               `json:"replyToId,omitempty"`
	ETag                 string                `json:"etag,omitempty"`
	MessageType          string                `json:"messageType,omitempty"`
	CreatedDateTime      string                `json:"createdDateTime,omitempty"`
	LastModifiedDateTime *string               `json:"lastModifiedDateTime,omitempty"`
	LastEditedDateTime   *string               `json:"lastEditedDateTime,omitempty"`
	DeletedDateTime      *string               `json:"deletedDateTime,omitempty"`
	Subject              *string               `json:"subject,omitempty"`
	Summary              *string               `json:"summary,omitempty"`
	ChatID               *string               `json:"chatId,omitempty"`
	Importance           *string               `json:"importance,omitempty"`
	Locale               *string               `json:"locale,omitempty"`
	WebURL               *string               `json:"webUrl,omitempty"`
	PolicyViolation      interface{}           `json:"policyViolation,omitempty"`
	EventDetail          interface{}           `json:"eventDetail,omitempty"`
	From                 *TeamsFrom            `json:"from,omitempty"`
	Body                 *TeamsBody            `json:"body,omitempty"`
	ChannelIdentity      *TeamsChannelIdentity `json:"channelIdentity,omitempty"`
	Attachments          []TeamsAttachment     `json:"attachments,omitempty"`
	Mentions             []interface{}         `json:"mentions,omitempty"`
	Reactions            []interface{}         `json:"reactions,omitempty"`
	MessageHistory       []interface{}         `json:"messageHistory,omitempty"`
}

// SourceClient fetches messages from a provider (Zoom, Slack...)
type SourceClient interface {
	GetUsers() ([]ZoomUser, error)
	GetUserChannels(userID string) ([]ZoomChannel, error)
	FetchMessages(userID string, channelID string) ([]ZoomMessage, error)
}

// DestinationClient posts messages and ensures destination resources.
type DestinationClient interface {
	EnsureTeam(name string, t TeamType) (teamID string, err error)
	EnsureChannel(teamID, name string, c ChannelType) (channelID string, err error)
	PostMessage(teamID, channelID string, m ZoomMessage) error
}
