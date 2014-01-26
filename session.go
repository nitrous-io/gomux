package gomux

import (
	"github.com/kr/pty"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16
	y      uint16
}

type Session struct {
	Width   int
	Height  int
	Pty     *os.File
	NumConn int
}

func NewSession() *Session {
	pty, err := pty.Start(exec.Command("/bin/bash"))
	if err != nil {
		panic("should not failed")
	}
	return &Session{Pty: pty}
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
	go func() {
		_, err := io.Copy(stdout, s.Pty)
		if err == io.ErrClosedPipe {
			err = nil
		}
		errors <- err
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
