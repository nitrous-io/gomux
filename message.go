package gomux

import (
	"fmt"
)

type Info struct {
	ActiveSessions []int `json:"activeSessions"`
}

func (i Info) String() string {
	return fmt.Sprintf("%s", i.ActiveSessions)
}

type Message struct {
	Event     string `json:"event"`
	SessionId int    `json:"sessionId"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}
