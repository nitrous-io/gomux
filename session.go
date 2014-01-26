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

type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16
	y      uint16
}

type Session struct {
	sync.Mutex
	Width  int
	Height int
	Pty    *os.File

	NumConn int
	buf     *bytes.Buffer
	writers map[io.WriteCloser]bool
}

func NewSession() *Session {
	pty, err := pty.Start(exec.Command("/bin/bash"))
	if err != nil {
		panic("should not failed")
	}
	s := &Session{Pty: pty, writers: make(map[io.WriteCloser]bool),
		buf: bytes.NewBuffer(nil)}
	// session acts as the broadcaster so we copy output from pty directly to
	// the session
	go func() {
		io.Copy(s, s.Pty)
	}()
	return s
}

func (s *Session) Resize(w, h int) error {
	ws := &Winsize{Width: uint16(s.Width), Height: uint16(s.Height)}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, s.Pty.Fd(),
		uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
	// skip errno = 0
	if err == 0 {
		return nil
	}
	return err
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

func (s *Session) CloseWriters() error {
	s.Lock()
	defer s.Unlock()
	for w := range s.writers {
		w.Close()
	}
	s.writers = make(map[io.WriteCloser]bool)
	return nil
}

func (s *Session) Attach(stdin io.ReadCloser, stdout io.Writer) chan error {
	s.NumConn += 1
	errors := make(chan error, 2)
	// stdin
	go func() {
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

// Code c/c from io.Copy() modified to handle escape sequence
func CopyEscapable(dst io.Writer, src io.ReadCloser) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			// ---- Docker addition
			// char 16 is C-p
			if nr == 1 && buf[0] == 16 {
				nr, er = src.Read(buf)
				// char 17 is C-q
				if nr == 1 && buf[0] == 17 {
					if err := src.Close(); err != nil {
						return 0, err
					}
					return 0, nil
				}
			}
			// ---- End of docker
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err
}
