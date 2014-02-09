package gomux

import (
	"os/exec"
	"testing"
)

func TestResize(t *testing.T) {
	p := createPty(exec.Command("bash"))
	defer p.Close()
	fd := p.Fd()
	var ws *Winsize
	SetWinsize(fd, &Winsize{Width: 80, Height: 24})
	ws, _ = GetWinsize(p.Fd())
	if ws.Width != 80 || ws.Height != 24 {
		t.Fail()
	}
	SetWinsize(fd, &Winsize{Width: 80, Height: 80})
	ws, _ = GetWinsize(p.Fd())
	if ws.Width != 80 || ws.Height != 80 {
		t.Fail()
	}
}
