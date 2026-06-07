// Package chooser is the interactive color designer shown before the fire starts.
// It lets the user pick the gradient anchor colors (Base, Primary, Secondary,
// Tertiary, Peak), previews the interpolated ramp live, and returns the chosen
// stops. It assumes the terminal is already in raw/alt-screen mode.
package chooser

import (
	"io"
	"strconv"

	"doomfire/internal/fire"
)

// swatch is a named, selectable color.
type swatch struct {
	name string
	rgb  fire.RGB
}

// swatches is the palette of colors the user cycles through for each slot.
var swatches = []swatch{
	{"Black", fire.RGB{0x07, 0x07, 0x07}},
	{"Charcoal", fire.RGB{0x30, 0x30, 0x38}},
	{"Crimson", fire.RGB{0xC0, 0x14, 0x28}},
	{"Red", fire.RGB{0xFF, 0x28, 0x14}},
	{"Ember", fire.RGB{0xC8, 0x32, 0x0A}},
	{"Orange", fire.RGB{0xFF, 0x78, 0x14}},
	{"Amber", fire.RGB{0xFF, 0xA0, 0x1E}},
	{"Gold", fire.RGB{0xFF, 0xC8, 0x28}},
	{"Yellow", fire.RGB{0xFF, 0xF0, 0x50}},
	{"Lime", fire.RGB{0xA0, 0xFF, 0x3C}},
	{"Green", fire.RGB{0x28, 0xDC, 0x3C}},
	{"Emerald", fire.RGB{0x14, 0xC8, 0x78}},
	{"Teal", fire.RGB{0x14, 0xC8, 0xC8}},
	{"Cyan", fire.RGB{0x28, 0xE6, 0xFF}},
	{"Sky", fire.RGB{0x3C, 0xA0, 0xFF}},
	{"Blue", fire.RGB{0x28, 0x5A, 0xFF}},
	{"Indigo", fire.RGB{0x5A, 0x3C, 0xE6}},
	{"Violet", fire.RGB{0x96, 0x46, 0xFF}},
	{"Purple", fire.RGB{0xB4, 0x32, 0xDC}},
	{"Magenta", fire.RGB{0xFF, 0x28, 0xC8}},
	{"Pink", fire.RGB{0xFF, 0x6E, 0xB4}},
	{"White", fire.RGB{0xFF, 0xFF, 0xFF}},
	{"Silver", fire.RGB{0xC8, 0xC8, 0xC8}},
}

// slot is one gradient anchor the user configures.
type slot struct {
	label string
	sel   int // index into swatches
}

// indexOf returns the swatch index whose color matches c, or 0.
func indexOf(c fire.RGB) int {
	for i, s := range swatches {
		if s.rgb == c {
			return i
		}
	}
	return 0
}

// Run shows the designer (drawing to out, reading keys from in) and blocks until
// the user confirms or quits. On confirm it returns the chosen color stops
// (coldest first) and ok == true.
func Run(out io.Writer, in io.Reader) (stops []fire.RGB, ok bool) {
	labels := []string{"Base (coldest)", "Primary", "Secondary", "Tertiary", "Peak (hottest)"}
	slots := make([]slot, len(labels))
	for i, lbl := range labels {
		def := fire.RGB{0, 0, 0}
		if i < len(fire.ClassicStops) {
			def = fire.ClassicStops[i]
		}
		slots[i] = slot{label: lbl, sel: indexOf(nearest(def))}
	}

	cur := 0
	buf := make([]byte, 16)

	draw(out, slots, cur)
	for {
		n, err := in.Read(buf)
		if err != nil || n == 0 {
			return nil, false
		}
		// Process every keystroke in the buffer so rapid input is never dropped.
		dirty := false
		for data := buf[:n]; len(data) > 0; {
			act, c := next(data)
			if c == 0 {
				break
			}
			data = data[c:]
			switch act {
			case actQuit:
				return nil, false
			case actConfirm:
				stops = make([]fire.RGB, len(slots))
				for i, s := range slots {
					stops[i] = swatches[s.sel].rgb
				}
				return stops, true
			case actUp:
				if cur > 0 {
					cur--
					dirty = true
				}
			case actDown:
				if cur < len(slots)-1 {
					cur++
					dirty = true
				}
			case actLeft:
				slots[cur].sel = (slots[cur].sel - 1 + len(swatches)) % len(swatches)
				dirty = true
			case actRight:
				slots[cur].sel = (slots[cur].sel + 1) % len(swatches)
				dirty = true
			}
		}
		if dirty {
			draw(out, slots, cur)
		}
	}
}

// nearest snaps an arbitrary color to the closest swatch color so classic
// defaults land on real swatch entries.
func nearest(c fire.RGB) fire.RGB {
	best, bestD := swatches[0].rgb, 1<<31
	for _, s := range swatches {
		dr, dg, db := int(c[0])-int(s.rgb[0]), int(c[1])-int(s.rgb[1]), int(c[2])-int(s.rgb[2])
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = s.rgb, d
		}
	}
	return best
}

type action int

const (
	actNone action = iota
	actUp
	actDown
	actLeft
	actRight
	actConfirm
	actQuit
)

// next decodes the first key token at the front of b, returning its action and
// how many bytes it consumed, so a buffer holding several keystrokes can be
// drained token by token.
func next(b []byte) (action, int) {
	if len(b) == 0 {
		return actNone, 0
	}
	if b[0] == 0x1b {
		if len(b) >= 3 && b[1] == '[' { // CSI arrow sequence
			switch b[2] {
			case 'A':
				return actUp, 3
			case 'B':
				return actDown, 3
			case 'C':
				return actRight, 3
			case 'D':
				return actLeft, 3
			}
			return actNone, 3
		}
		return actQuit, 1 // lone Esc
	}
	switch b[0] {
	case 'k', 'w':
		return actUp, 1
	case 'j', 's':
		return actDown, 1
	case 'h', 'a':
		return actLeft, 1
	case 'l', 'd':
		return actRight, 1
	case '\r', '\n', ' ':
		return actConfirm, 1
	case 'q', 'Q', 3: // q, Ctrl-C
		return actQuit, 1
	}
	return actNone, 1
}

// parse returns the action for the token at the front of b.
func parse(b []byte) action { a, _ := next(b); return a }

// draw renders the whole designer screen in one write.
func draw(out io.Writer, slots []slot, cur int) {
	var b []byte
	b = append(b, "\x1b[?2026h\x1b[H\x1b[2J"...) // sync, home, clear

	b = append(b, "\x1b[1;38;2;255;176;0m  D O O M   F I R E  \x1b[0m\x1b[2m· color designer\x1b[0m\r\n\r\n"...)
	b = append(b, "  \x1b[2m\x1b[38;2;120;120;120m↑/↓ slot   ←/→ color   Enter ignite   q quit\x1b[0m\r\n\r\n"...)

	for i, s := range slots {
		sw := swatches[s.sel]
		if i == cur {
			b = append(b, "  \x1b[1m\x1b[38;2;255;255;255m> "...)
		} else {
			b = append(b, "    \x1b[38;2;160;160;160m"...)
		}
		b = appendPad(b, s.label, 17)
		b = append(b, "\x1b[0m "...)
		b = appendBlocks(b, sw.rgb, 6) // color swatch
		b = append(b, "  "...)
		if i == cur {
			b = append(b, "\x1b[1m\x1b[38;2;255;255;255m"...)
		} else {
			b = append(b, "\x1b[38;2;200;200;200m"...)
		}
		b = appendPad(b, sw.name, 9)
		b = append(b, "\x1b[0m\r\n"...)
	}

	// Live interpolated preview of the resulting flame ramp (coldest -> hottest).
	stops := make([]fire.RGB, len(slots))
	for i, s := range slots {
		stops[i] = swatches[s.sel].rgb
	}
	pal := fire.BuildPalette(stops)
	b = append(b, "\r\n  \x1b[2m\x1b[38;2;120;120;120mpreview\x1b[0m\r\n  "...)
	for i := range pal {
		b = appendBg(b, pal[i])
		b = append(b, "  "...) // two-wide cell per heat level
	}
	b = append(b, "\x1b[0m\r\n"...)

	b = append(b, "\x1b[?2026l"...)
	out.Write(b)
}

func appendBlocks(b []byte, c fire.RGB, w int) []byte {
	b = appendSGR(b, "38;2;", c)
	for range w {
		b = append(b, "\xe2\x96\x88"...) // █
	}
	return append(b, "\x1b[0m"...)
}

func appendBg(b []byte, c fire.RGB) []byte { return appendSGR(b, "48;2;", c) }

func appendSGR(b []byte, kind string, c fire.RGB) []byte {
	b = append(b, "\x1b["...)
	b = append(b, kind...)
	b = strconv.AppendUint(b, uint64(c[0]), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(c[1]), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(c[2]), 10)
	return append(b, 'm')
}

// appendPad appends s left-justified to width w (truncating if longer).
func appendPad(b []byte, s string, w int) []byte {
	if len(s) > w {
		s = s[:w]
	}
	b = append(b, s...)
	for range w - len(s) {
		b = append(b, ' ')
	}
	return b
}
