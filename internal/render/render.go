// Package render turns a fire grid into ANSI frames using half-block glyphs (▀):
// foreground is the top pixel, background the bottom. Frames are wrapped in DEC
// synchronized output (2026) and diffed against the previous frame.
//
// Each frame is assembled into a single reusable byte buffer and written in one
// call, so the steady state is allocation-free and costs one write per frame.
package render

import (
	"io"

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
//
// All sequences are slices into one contiguous arena: a single allocation
// instead of ~1.4k tiny ones, with no per-object overhead or size-class
// rounding waste. The arena is sized for the worst case up front and must
// never grow once slicing starts, or earlier slices would alias a stale array.
func BuildEscapes() {
	const (
		maxSingle = len("\x1b[38;2;255;255;255m")
		maxPair   = len("\x1b[38;2;255;255;255;48;2;255;255;255m")
	)
	arena := make([]byte, 0, 2*n*maxSingle+n*n*maxPair)
	// take returns the bytes appended since start, capacity-capped so a later
	// append through the slice can never clobber a neighboring sequence.
	take := func(start int) []byte { return arena[start:len(arena):len(arena)] }

	for i, c := range fire.Palette {
		s := len(arena)
		arena = appendRGB(append(arena, "\x1b[38;2;"...), c)
		arena = append(arena, 'm')
		fgSeq[i] = take(s)

		s = len(arena)
		arena = appendRGB(append(arena, "\x1b[48;2;"...), c)
		arena = append(arena, 'm')
		bgSeq[i] = take(s)
	}
	for t := range fire.Palette {
		for b := range fire.Palette {
			s := len(arena)
			arena = appendRGB(append(arena, "\x1b[38;2;"...), fire.Palette[t])
			arena = appendRGB(append(arena, ";48;2;"...), fire.Palette[b])
			arena = append(arena, 'm')
			pairSeq[t][b] = take(s)
		}
	}
}

// appendRGB appends "r;g;b" without allocating.
func appendRGB(b []byte, c [3]uint8) []byte {
	b = appendUint(b, int(c[0]))
	b = append(b, ';')
	b = appendUint(b, int(c[1]))
	b = append(b, ';')
	return appendUint(b, int(c[2]))
}

type Renderer struct {
	prev []uint8
	full bool   // force a full redraw
	buf  []byte // reused frame scratch buffer
}

func New(f *fire.Fire) *Renderer {
	r := &Renderer{}
	r.Resize(f)
	return r
}

// bufCap is the worst-case frame size, so the buffer never grows mid-animation
// (a reallocation would trigger GC and a visible hitch). A drawn cell costs at
// most a combined fg+bg SGR (36 bytes) + glyph (3) = 39. Cursor moves (<=24
// bytes) only follow a skipped cell or start a row: a skipped cell's unused
// 39-byte budget covers its run's move, leaving one move per row unpaid.
func bufCap(w, h int) int { return w*(h/2)*39 + (h/2)*24 + 64 }

// Resize re-fits the renderer to f's dimensions and forces a full redraw,
// reusing existing capacity so a stream of resize events doesn't churn
// hundreds of KB of garbage per event.
func (r *Renderer) Resize(f *fire.Fire) {
	// prev needs no clearing: the forced full frame overwrites every cell.
	if size := f.Width() * f.Height(); cap(r.prev) >= size {
		r.prev = r.prev[:size]
	} else {
		r.prev = make([]uint8, size)
	}
	if c := bufCap(f.Width(), f.Height()); cap(r.buf) < c {
		r.buf = make([]byte, 0, c)
	}
	r.full = true
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
