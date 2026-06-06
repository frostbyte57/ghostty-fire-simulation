// Package terminal handles raw fullscreen mode (alt screen, hidden cursor, no
// echo), sizing, and restoring the original state on exit.
package terminal

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

type winsize struct {
	rows, cols, xpixel, ypixel uint16
}

// Size returns the terminal dimensions in cells, or a 24x80 fallback.
func Size() (rows, cols int) {
	var ws winsize
	syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if ws.rows == 0 || ws.cols == 0 {
		return 24, 80
	}
	return int(ws.rows), int(ws.cols)
}

type Terminal struct {
	out      *os.File
	sttyOrig string
}

func Setup() *Terminal {
	orig, _ := exec.Command("stty", "-g").Output()
	t := &Terminal{out: os.Stdout, sttyOrig: string(orig)}
	t.stty("-echo", "-icanon", "min", "1", "time", "0")
	t.out.WriteString("\x1b[?1049h")
	t.out.WriteString("\x1b[?25l")
	return t
}

func (t *Terminal) Restore() {
	t.out.WriteString("\x1b[0m")
	t.out.WriteString("\x1b[?25h")
	t.out.WriteString("\x1b[?1049l")
	if t.sttyOrig != "" {
		t.stty(t.sttyOrig)
	} else {
		t.stty("sane")
	}
}

func (t *Terminal) Clear()        { t.out.WriteString("\x1b[2J") }
func (t *Terminal) Out() *os.File { return t.out }

func (t *Terminal) stty(args ...string) {
	c := exec.Command("stty", args...)
	c.Stdin = os.Stdin
	c.Run()
}
