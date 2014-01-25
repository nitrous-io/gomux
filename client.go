package gomux

import (
	"io"
	"net"
	"os"
	"time"
)

type Client struct {
	SessionId int
	Conn      net.Conn
}

func (c *Client) Connect() {
	conn, err := net.Dial("unix", "/tmp/gomux.sock")
	if err != nil {
		return
	}
	c.Conn = conn
	attachMsg := &Message{Type: Attach, SessionId: c.SessionId}
	c.Conn.Write(attachMsg.Encode())
	time.Sleep(100 * time.Millisecond)

	for {
		io.Copy(os.Stdout, c.Conn)
	}
}
