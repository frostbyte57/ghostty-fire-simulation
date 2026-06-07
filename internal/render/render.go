// Package render turns a fire grid into ANSI frames using half-block glyphs (▀):
// foreground is the top pixel, background the bottom. Frames are wrapped in DEC
// synchronized output (2026) and diffed against the previous frame.
//
// Each frame is assembled into a single reusable byte buffer and written in one
// call, so the steady state is allocation-free and costs one write per frame.
package render

import (
	"io"
	"strconv"

	"doomfire/internal/fire"
)

const n = len(fire.Palette)

// Fully precomputed escape sequences so the render loop only ever copies bytes.
//
//	fgSeq[i]      = "\x1b[38;2;r;g;bm"            set foreground to palette i
//	bgSeq[i]      = "\x1b[48;2;r;g;bm"            set background to palette i
//	pairSeq[t][b] = "\x1b[38;2;..;48;2;..m"       set both at once (the common case)
var (
	fgSeq     [n][]byte
	bgSeq     [n][]byte
	pairSeq   [n][n][]byte
	halfBlock = []byte("▀")
	syncOn    = []byte("\x1b[?2026h")
	syncOff   = []byte("\x1b[?2026l")
)

func init() { BuildEscapes() }

// BuildEscapes (re)computes the precomputed SGR tables from the current
// fire.Palette. Call it after changing the palette and before rendering.
func BuildEscapes() {
	rgb := func(c [3]uint8) string {
		return strconv.Itoa(int(c[0])) + ";" + strconv.Itoa(int(c[1])) + ";" + strconv.Itoa(int(c[2]))
	}
	for i, c := range fire.Palette {
		fr := rgb(c)
		fgSeq[i] = []byte("\x1b[38;2;" + fr + "m")
		bgSeq[i] = []byte("\x1b[48;2;" + fr + "m")
	}
	for t := range fire.Palette {
		ft := rgb(fire.Palette[t])
		for b := range fire.Palette {
			pairSeq[t][b] = []byte("\x1b[38;2;" + ft + ";48;2;" + rgb(fire.Palette[b]) + "m")
		}
	}
}

type Renderer struct {
	prev []uint8
	full bool   // force a full redraw
	buf  []byte // reused frame scratch buffer
}

func New(f *fire.Fire) *Renderer {
	w, h := f.Width(), f.Height()
	return &Renderer{
		prev: make([]uint8, w*h),
		full: true,
		// Worst case per cell: cursor move (~10) + combined fg+bg SGR (~36) +
		// glyph (3) ≈ 49 bytes. Over-allocate to 56 so the buffer never grows
		// mid-animation (a reallocation would trigger GC and a visible hitch).
		buf: make([]byte, 0, w*(h/2)*56+64),
	}
}

// Reset forces the next Frame to redraw every cell (e.g. after a resize/clear).
func (r *Renderer) Reset() { r.full = true }

// Frame builds one diffed frame of f into the internal buffer and writes it to w
// in a single call.
func (r *Renderer) Frame(w io.Writer, f *fire.Fire) {
	px := f.Pixels()
	width := f.Width()
	rows := f.Height() / 2

	b := r.buf[:0]
	b = append(b, syncOn...)

	lastTop, lastBot := -1, -1
	curRow, curCol := -1, -1 // tracked cursor (1-based); -1 == unknown

	for row := range rows {
		base0 := (row * 2) * width
		base1 := base0 + width
		for col := range width {
			top := px[base0+col]
			bot := px[base1+col]

			if !r.full && top == r.prev[base0+col] && bot == r.prev[base1+col] {
				continue
			}
			r.prev[base0+col] = top
			r.prev[base1+col] = bot

			if curRow != row+1 || curCol != col+1 {
				b = append(b, '\x1b', '[')
				b = appendUint(b, row+1)
				b = append(b, ';')
				b = appendUint(b, col+1)
				b = append(b, 'H')
			}

			it, ib := int(top), int(bot)
			switch {
			case it != lastTop && ib != lastBot:
				b = append(b, pairSeq[top][bot]...)
				lastTop, lastBot = it, ib
			case it != lastTop:
				b = append(b, fgSeq[top]...)
				lastTop = it
			case ib != lastBot:
				b = append(b, bgSeq[bot]...)
				lastBot = ib
			}

			b = append(b, halfBlock...)
			curRow, curCol = row+1, col+2
		}
	}

	b = append(b, syncOff...)
	r.buf = b
	w.Write(b)
	r.full = false
}

// appendUint appends a non-negative int as decimal, allocation-free.
func appendUint(b []byte, num int) []byte {
	if num == 0 {
		return append(b, '0')
	}
	var tmp [10]byte
	i := len(tmp)
	for num > 0 {
		i--
		tmp[i] = byte('0' + num%10)
		num /= 10
	}
	return append(b, tmp[i:]...)
}
