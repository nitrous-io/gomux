package gomux

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
)

type Mux struct {
	Sessions map[int]*Session
}

func NewMux() *Mux {
	return &Mux{Sessions: make(map[int]*Session)}
}

func (m *Mux) Connect(i int) *Session {
	s, ok := m.Sessions[i]
	if ok {
		return s
	}
	s = NewSession()
	m.Sessions[i] = s
	return s
}

func (m *Mux) Close() {
	for _, s := range m.Sessions {
		s.Close()
	}
}

func (m *Mux) Handle(c net.Conn) {
	defer m.Close()
	b := make([]byte, 32748)
	for {
		n, err := c.Read(b)
		if err != nil {
			return
		}
		msg, err := ParseMessage(string(b[0:n]))
		if err != nil {
			return
		}
		fmt.Println(msg)
		s := m.Connect(msg.SessionId)
		switch msg.Type {
		case Attach:
			s.Conns = append(s.Conns, c)
			go func() {
				io.Copy(c, s.Pty)
			}()
		case Resize:
			go s.Resize(msg.Width, msg.Height)
		case Data:
			b := bytes.NewBuffer([]byte(msg.Data))
			go io.Copy(s.Pty, b)
		}
	}
}

func (s *Mux) Listen() {
	// remove the socket if not exist
	os.RemoveAll("/tmp/gomux.sock")

	l, err := net.Listen("unix", "/tmp/gomux.sock")
	if err != nil {
		panic(fmt.Sprintf("listen error %v", err))
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("accept error %v \n", err)
			return
		}
		go s.Handle(conn)
	}
}
