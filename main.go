// Command doomfire renders the DOOM fire effect across the full terminal,
// animated and tuned for Ghostty. Press q or Ctrl-C to quit.
package main

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"doomfire/internal/chooser"
	"doomfire/internal/fire"
	"doomfire/internal/render"
	"doomfire/internal/terminal"
)

const fps = 60

func main() {
	term := terminal.Setup()
	defer term.Restore()
	out := term.Out()

	// Color designer: let the user pick the flame's gradient before igniting.
	stops, ok := chooser.Run(out, os.Stdin)
	if !ok {
		return // user quit during selection
	}
	fire.Palette = fire.BuildPalette(stops)
	render.BuildEscapes()
	term.Clear()

	var quit atomic.Bool

	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		quit.Store(true)
	}()

	go watchInput(&quit)

	rows, cols := terminal.Size()
	f := fire.New(cols, rows*2)
	r := render.New(f)

	ticker := time.NewTicker(time.Second / fps)
	defer ticker.Stop()

	for !quit.Load() {
		select {
		case <-resize:
			rows, cols = terminal.Size()
			f = fire.New(cols, rows*2)
			r = render.New(f)
			term.Clear()
		case <-ticker.C:
			f.Step()
			r.Frame(out, f)
		}
	}
}

// watchInput flips quit on q, Q, or Ctrl-C.
func watchInput(quit *atomic.Bool) {
	b := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(b)
		if err != nil {
			return
		}
		if n > 0 && (b[0] == 'q' || b[0] == 'Q' || b[0] == 3) {
			quit.Store(true)
			return
		}
	}
}
