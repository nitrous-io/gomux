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

type Message struct {
	Event     string `json:"event"`
	SessionId string `json:"sessionId"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`

	// server sequentially generated id instead of user generated id
	SessionServerId int `json:"sessionServerId"`
}
