package gomux

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/dotcloud/docker/pkg/term"
)

const SERVER = "/tmp/gomux.sock"

type Client struct {
	in         io.ReadCloser
	out        io.Writer
	sessionId  int
	conn       net.Conn
	terminalFd uintptr
}

func NewClient(in io.ReadCloser, out io.Writer, sessionId int) *Client {
	file, ok := in.(*os.File)
	if !ok {
		panic("require in to be *os.File")
	}
	return &Client{
		in:         in,
		out:        out,
		sessionId:  sessionId,
		terminalFd: file.Fd(),
	}
}

func (c *Client) Connect() error {
	in := c.in
	stdout := c.out

	req, err := http.NewRequest("POST", fmt.Sprintf("/session/%d", c.sessionId), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "plain/text")
	req.Host = SERVER

	dial, err := net.Dial("unix", SERVER)
	if err != nil {
		panic("Can't connect to gomux daemon")
	}
	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()
	// Server hijacks the connection, error 'connection closed' expected
	clientconn.Do(req)

	rwc, br := clientconn.Hijack()
	defer rwc.Close()

	// restore the terminal back to old state when we finish
	var oldState *term.State
	oldState, err = term.SetRawTerminal(c.terminalFd)
	if err != nil {
		return err
	}
	defer term.RestoreTerminal(c.terminalFd, oldState)

	receiveStdout := Go(func() (err error) {
		defer func() {
			if in != nil {
				term.RestoreTerminal(c.terminalFd, oldState)
				in.Close()
			}
		}()
		_, err = io.Copy(stdout, br)
		Debugf("[hijack] End of stdout")
		return err
	})

	go func() {
		if in != nil {
			io.Copy(rwc, in)
			Debugf("[hijack] End of stdin")
		}
		if tcpc, ok := rwc.(*net.TCPConn); ok {
			if err := tcpc.CloseWrite(); err != nil {
				Debugf("Couldn't send EOF: %s\n", err)
			}
		} else if unixc, ok := rwc.(*net.UnixConn); ok {
			if err := unixc.CloseWrite(); err != nil {
				Debugf("Couldn't send EOF: %s\n", err)
			}
		}
	}()

	return <-receiveStdout
}
