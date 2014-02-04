package gomux

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	du "github.com/dotcloud/docker/utils"
	"github.com/kr/pty"
)

type Session struct {
	sync.Mutex
	Pty *os.File

	NumConn int
	buf     *bytes.Buffer
	writers map[io.WriteCloser]bool
}

func NewSession() *Session {
	s := &Session{}
	s.MaybeRecreate()
	return s
}

func (s *Session) MaybeRecreate() {
	Debugln("num conn", s.NumConn)
	if s.NumConn > 0 {
		return
	}
	var err error
	s.writers = make(map[io.WriteCloser]bool)
	s.buf = bytes.NewBuffer(nil)
	s.Pty, err = pty.Start(exec.Command("bash"))
	if err != nil {
		panic("failed to create the session")
	}
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
	s.buf = bytes.NewBuffer(nil)
	s.Pty.Close()
}

func (s *Session) Resize(w, h int) error {
	return SetWinsize(s.Pty.Fd(),
		&Winsize{ws_row: uint16(w), ws_col: uint16(h)})
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
	s.buf.Write(p)
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
		_, err := CopyEscapable(s.Pty, stdin)
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
	return Go(func() error {
		for i := 0; i < 2; i += 1 {
			if err := <-errors; err != nil {
				return err
			}
		}
		return nil
	})
}

type Winsize struct {
	ws_row    uint16
	ws_col    uint16
	ws_xpixel uint16
	ws_ypixel uint16
}

func GetWinsize(fd uintptr) (*Winsize, error) {
	ws := &Winsize{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd,
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	if err == 0 {
		return ws, nil
	}
	return ws, err
}

func SetWinsize(fd uintptr, ws *Winsize) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd,
		uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
	if err == 0 {
		return nil
	}
	return err
}
