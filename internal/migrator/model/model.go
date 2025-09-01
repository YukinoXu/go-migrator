package model

import "time"

type Message struct {
	ID        string            `json:"id"`
	From      string            `json:"from"`
	Content   string            `json:"content"`
	Timestamp time.Time         `json:"timestamp"`
	Meta      map[string]string `json:"meta,omitempty"`
}

type Conversation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// SourceClient fetches messages from a provider (Zoom, Slack...)
type SourceClient interface {
	FetchMessages(conversationID string) ([]Message, error)
}

// DestinationClient posts messages and ensures destination resources.
type DestinationClient interface {
	EnsureTeam(name string) (teamID string, err error)
	EnsureChannel(teamID, name string) (channelID string, err error)
	PostMessage(teamID, channelID string, m Message) error
}
