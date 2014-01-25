package gomux

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	Attach         = "a"[0]
	Resize         = "r"[0]
	Data           = "d"[0]
	Ack            = "k"[0]
	ErrInvalidType = errors.New("Invalid type")
)

type Message struct {
	Type      uint8
	SessionId int
	Width     int
	Height    int
	Data      string
}

func (m *Message) Encode() []byte {
	switch m.Type {
	case Attach:
		return []byte(fmt.Sprintf("%s|%d", string(m.Type), m.SessionId))
	case Resize:
		return []byte(fmt.Sprintf("%s|%d|%d:%d", string(m.Type), m.SessionId,
			m.Width, m.Height))
	case Data:
		return []byte(fmt.Sprintf("%s|%d|%s", string(m.Type), m.SessionId,
			m.Data))
	}
	return []byte{}
}

func (m *Message) ParseAttach(d string) (err error) {
	m.SessionId, err = strconv.Atoi(d)
	return
}

func (m *Message) ParseResize(d string) (err error) {
	_, err = fmt.Sscanf(d, "%d|%d:%d", m.SessionId, m.Width, m.Height)
	return
}

func (m *Message) ParseData(d string) (err error) {
	_, err = fmt.Sscanf(d, "%d|%s", m.SessionId, m.Data)
	return
}

// message has format type|sessionId|data
func ParseMessage(str string) (*Message, error) {
	m := &Message{Type: str[0]}
	switch str[0] {
	case Attach:
		err := m.ParseAttach((str[2:len(str)]))
		return m, err
	case Resize:
		err := m.ParseResize(str[2:len(str)])
		return m, err
	case Data:
		err := m.ParseData(str[2:len(str)])
		return m, err
	}
	return nil, ErrInvalidType
}
