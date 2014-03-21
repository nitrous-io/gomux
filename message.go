package gomux

import (
	"fmt"
)

type Info struct {
	ActiveSessions []string `json:"activeSessions"`
}

func (i Info) String() string {
	return fmt.Sprintf("%s", i.ActiveSessions)
}

type SessionInfo struct {
	SessionId       string
	SessionServerId int
}

type Message struct {
	Event     string `json:"event"`
	SessionId string `json:"sessionId,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`

	// server sequentially generated id instead of user generated id
	SessionServerId int `json:"sessionServerId,omitempty"`

	// list of all current sessions. This is helpful for the first time the
	// client connect to events stream. He can catch up on existings sessions
	// with this and then listen to new_client event after
	Sessions []SessionInfo `json:"sessions,omitempty"`
}
