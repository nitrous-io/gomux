package gomux

import (
	"fmt"
)

type Info struct {
	ActiveSessions []SessionInfo `json:"sessions"`
}

func (i Info) String() string {
	return fmt.Sprintf("%s", i.ActiveSessions)
}

type SessionInfo struct {
	Id       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	ServerId int    `json:"serverId,omitempty"`
}

type Message struct {
	Event  string `json:"event"`
	Id     string `json:"id,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`

	// server sequentially generated id instead of user generated id
	Name     string `json:"name,omitempty"`
	ServerId int    `json:"serverId,omitempty"`

	// list of all current sessions. This is helpful for the first time the
	// client connect to events stream. He can catch up on existings sessions
	// with this and then listen to new_client event after
	Sessions []SessionInfo `json:"sessions,omitempty"`
}
