// Package render turns a fire grid into ANSI frames using half-block glyphs (▀):
// foreground is the top pixel, background the bottom. Frames are wrapped in DEC
// synchronized output (2026) and diffed against the previous frame.
package render

import (
	"bufio"
	"strconv"

	"doomfire/internal/fire"
)

// Precomputed SGR fragments (e.g. "38;2;r;g;b") so the render loop never formats ints.
var (
	fgParam   [len(fire.Palette)][]byte
	bgParam   [len(fire.Palette)][]byte
	halfBlock = []byte("▀")
)

func init() {
	for i, c := range fire.Palette {
		rgb := strconv.Itoa(int(c[0])) + ";" + strconv.Itoa(int(c[1])) + ";" + strconv.Itoa(int(c[2]))
		fgParam[i] = []byte("38;2;" + rgb)
		bgParam[i] = []byte("48;2;" + rgb)
	}
}

type Renderer struct {
	prev []uint8
	full bool // force a full redraw
}

func New(f *fire.Fire) *Renderer {
	return &Renderer{prev: make([]uint8, f.Width()*f.Height()), full: true}
}

// Reset forces the next Frame to redraw every cell (e.g. after a resize/clear).
func (r *Renderer) Reset() { r.full = true }

// Frame writes one diffed frame of f to buf and flushes it.
func (r *Renderer) Frame(buf *bufio.Writer, f *fire.Fire) {
	px := f.Pixels()
	w := f.Width()
	rows := f.Height() / 2

	buf.WriteString("\x1b[?2026h")

	lastTop, lastBot := -1, -1
	curRow, curCol := -1, -1 // tracked cursor (1-based); -1 == unknown

	for row := range rows {
		base0 := (row * 2) * w
		base1 := base0 + w
		for col := range w {
			top := px[base0+col]
			bot := px[base1+col]

			if !r.full && top == r.prev[base0+col] && bot == r.prev[base1+col] {
				continue
			}
			r.prev[base0+col] = top
			r.prev[base1+col] = bot

			if curRow != row+1 || curCol != col+1 {
				buf.WriteString("\x1b[")
				writeUint(buf, row+1)
				buf.WriteByte(';')
				writeUint(buf, col+1)
				buf.WriteByte('H')
			}

			it, ib := int(top), int(bot)
			switch {
			case it != lastTop && ib != lastBot:
				buf.WriteString("\x1b[")
				buf.Write(fgParam[top])
				buf.WriteByte(';')
				buf.Write(bgParam[bot])
				buf.WriteByte('m')
				lastTop, lastBot = it, ib
			case it != lastTop:
				buf.WriteString("\x1b[")
				buf.Write(fgParam[top])
				buf.WriteByte('m')
				lastTop = it
			case ib != lastBot:
				buf.WriteString("\x1b[")
				buf.Write(bgParam[bot])
				buf.WriteByte('m')
				lastBot = ib
			}

			buf.Write(halfBlock)
			curRow, curCol = row+1, col+2
		}
	}

	buf.WriteString("\x1b[?2026l")
	buf.Flush()
	r.full = false
}

// writeUint writes a non-negative int as decimal, allocation-free.
func writeUint(buf *bufio.Writer, n int) {
	if n == 0 {
		buf.WriteByte('0')
		return
	}
	var tmp [10]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}
	for ; i < len(tmp); i++ {
		buf.WriteByte(tmp[i])
	}
}
