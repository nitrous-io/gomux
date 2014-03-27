package gomux

import (
	"io"
	"os"
	"os/exec"
	"sync"

	du "github.com/dotcloud/docker/utils"
	"github.com/kr/pty"
)

type Session struct {
	sync.Mutex
	ServerId int // easy id for a session instead of user-generated id
	Pty      io.ReadWriter
	NumConn  int
	Name     string
	writers  map[io.WriteCloser]bool
	sizes    map[string]*Winsize
}

func NewSession(id int) *Session {
	s := &Session{ServerId: id}
	s.MaybeRecreate()
	return s
}

func (s *Session) MaybeRecreate() {
	if s.NumConn > 0 {
		return
	}
	s.sizes = make(map[string]*Winsize)
	s.writers = make(map[io.WriteCloser]bool)
	s.Pty = createPty(exec.Command("bash"))
	// session acts as the broadcaster so we copy output from pty directly to
	// the session
	go func() {
		io.Copy(s, s.Pty)
	}()
}

func (s *Session) Close() {
	for w, _ := range s.writers {
		w.Close()
	}
	s.writers = make(map[io.WriteCloser]bool)
	p := s.Pty.(*os.File)
	p.Close()
}

func (s *Session) Resize(id string, w, h int) error {
	// save the new size for the client
	ws, ok := s.sizes[id]
	if !ok {
		ws = &Winsize{Width: uint16(w), Height: uint16(h)}
		s.sizes[id] = ws
	} else {
		ws.Width = uint16(w)
		ws.Height = uint16(h)
	}
	// find smallest width and height in all winsizes
	var minw, minh uint16
	for _, ws := range s.sizes {
		if minw == 0 || minw > ws.Width {
			minw = ws.Width
		}
		if minh == 0 || minh > ws.Height {
			minh = ws.Height
		}
	}
	return SetWinsize(s.fd(), &Winsize{Width: minw, Height: minh})
}

func (s *Session) StdoutPipe() (io.ReadCloser, error) {
	s.Lock()
	defer s.Unlock()
	reader, writer := io.Pipe()
	s.writers[writer] = true
	return du.NewBufReader(reader), nil
}

func (s *Session) Write(p []byte) (n int, err error) {
	s.Lock()
	defer s.Unlock()
	for w := range s.writers {
		lp := p
		if n, err := w.Write(lp); err != nil || n != len(lp) {
			// On error, evict the writer
			delete(s.writers, w)
		}
	}
	return len(p), nil
}

func (s *Session) Attach(stdin io.ReadCloser, stdout io.Writer) chan error {
	errors := make(chan error, 2)
	// stdin
	go func() {
		defer func() {
			s.NumConn -= 1
			// if there isnt anymore connection, probably we should just close
			if s.NumConn == 0 {
				s.Close()
			}
		}()
		s.NumConn += 1
		_, err := io.Copy(s.Pty, stdin)
		if err == io.ErrClosedPipe {
			err = nil
		}
		errors <- err
	}()
	// stdout
	go func() {
		p, err := s.StdoutPipe()
		if err != nil {
			errors <- err
		} else {
			_, err := io.Copy(stdout, p)
			if err == io.ErrClosedPipe {
				err = nil
			}
			errors <- err
		}
	}()
	// force re-draw
	s.Redraw()
	return Go(func() error {
		for i := 0; i < 2; i += 1 {
			if err := <-errors; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Session) Winsize() *Winsize {
	ws, _ := GetWinsize(s.fd())
	return ws
}

// perform a resize to force-redraw of the session
func (s *Session) Redraw() error {
	ws, _ := GetWinsize(s.fd())
	err := SetWinsize(s.fd(), &Winsize{Width: ws.Width, Height: ws.Height - 1})
	if err != nil {
		return err
	}
	return SetWinsize(s.fd(), &Winsize{Width: ws.Width, Height: ws.Height})
}

func (s *Session) fd() uintptr {
	p := s.Pty.(*os.File)
	return p.Fd()
}

// if we cant create the pty, just panic
func createPty(cmd *exec.Cmd) *os.File {
	p, err := pty.Start(cmd)
	if err != nil {
		panic("failed to create new pty")
	}
	return p
}
