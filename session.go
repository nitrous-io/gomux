package gomux

import (
	"github.com/kr/pty"
	"log"
	"net"
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
	Width  int
	Height int
	Pty    *os.File
	Conns  []net.Conn
}

func NewSession() *Session {
	pty, err := pty.Start(exec.Command("/bin/bash"))
	if err != nil {
		panic("should not failed")
	}
	log.Println("new session")
	return &Session{Pty: pty}
}

func (s *Session) Close() {
	for _, c := range s.Conns {
		c.Close()
	}
	s.Pty.Close()
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
