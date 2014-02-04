package gomux

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/dotcloud/docker/pkg/term"
)

type Client struct {
	in         io.ReadCloser
	out        io.Writer
	sessionId  int
	conn       net.Conn
	terminalFd uintptr
	connected  bool
	server     string
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

func (c *Client) Connect(server string) error {
	in := c.in
	stdout := c.out
	c.server = server

	req, err := http.NewRequest("POST", fmt.Sprintf("/session/%d", c.sessionId), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "plain/text")
	req.Host = server

	dial, err := net.Dial("unix", server)
	if err != nil {
		return fmt.Errorf("Can't connect to gomux daemon %v", server)
	}
	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()
	// Server hijacks the connection, error 'connection closed' expected
	clientconn.Do(req)

	// hijack the http connection here. We have the stream now
	rwc, br := clientconn.Hijack()
	defer rwc.Close()

	// restore the terminal back to old state when we finish
	var oldState *term.State
	oldState, err = term.SetRawTerminal(c.terminalFd)
	if err != nil {
		return err
	}
	defer term.RestoreTerminal(c.terminalFd, oldState)

	// whatever data is sent through the socket, we want to display it
	receiveStdout := Go(func() (err error) {
		defer func() {
			term.RestoreTerminal(c.terminalFd, oldState)
			in.Close()
		}()
		// if this is first time we connect, do some more setup here
		if !c.connected {
			c.monitorSize()
			c.connected = true
		}
		_, err = io.Copy(stdout, br)
		return err
	})

	// whatever we type into the terminal, it will be sent through the socket
	go func() {
		io.Copy(rwc, in)
	}()

	return <-receiveStdout
}

func (c *Client) postResize() error {
	w, h := c.getSize()
	values := make(url.Values)
	values.Set("w", strconv.Itoa(w))
	values.Set("h", strconv.Itoa(h))
	req, err := http.NewRequest("PUT",
		fmt.Sprintf("/session/%d/resize", c.sessionId),
		bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = c.server

	dial, err := net.Dial("unix", c.server)
	if err != nil {
		return fmt.Errorf("Can't connect to gomux daemon %v", c.server)
	}
	defer dial.Close()
	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()
	_, err = clientconn.Do(req)
	return err
}

// monitor size of the current client and post the change to server
func (c *Client) monitorSize() error {
	c.postResize()
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)
	go func() {
		for _ = range sigchan {
			c.postResize()
		}
	}()
	return nil
}

// get current size of the client terminal
func (c *Client) getSize() (int, int) {
	ws, err := GetWinsize(c.terminalFd)
	if err != nil {
		// just use 80, 24 as the default
		return 80, 24
	}
	return int(ws.ws_row), int(ws.ws_col)
}
