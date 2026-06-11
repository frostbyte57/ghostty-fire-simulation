package render

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"testing"

	"doomfire/internal/fire"
)

// Representative full-screen size: 200 cols x 50 rows => 200x100 pixel grid.
const benchW, benchH = 200, 100

// BenchmarkFrame measures a realistic step+draw cycle: the fire changes every
// iteration, so frame diffing reflects a true steady-state flame.
func BenchmarkFrame(b *testing.B) {
	f := fire.New(benchW, benchH)
	for range 300 { // reach steady-state flame
		f.Step()
	}
	r := New(f)
	w := bufio.NewWriter(io.Discard)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.Step()
		r.Frame(w, f)
	}
	b.StopTimer()
	w.Flush()
}

// TestEscapes checks the arena-built SGR tables against a straightforward
// string construction.
func TestEscapes(t *testing.T) {
	rgb := func(c [3]uint8) string {
		return strconv.Itoa(int(c[0])) + ";" + strconv.Itoa(int(c[1])) + ";" + strconv.Itoa(int(c[2]))
	}
	for i, c := range fire.Palette {
		if want := "\x1b[38;2;" + rgb(c) + "m"; string(fgSeq[i]) != want {
			t.Fatalf("fgSeq[%d] = %q, want %q", i, fgSeq[i], want)
		}
		if want := "\x1b[48;2;" + rgb(c) + "m"; string(bgSeq[i]) != want {
			t.Fatalf("bgSeq[%d] = %q, want %q", i, bgSeq[i], want)
		}
	}
	for a := range fire.Palette {
		for b := range fire.Palette {
			want := "\x1b[38;2;" + rgb(fire.Palette[a]) + ";48;2;" + rgb(fire.Palette[b]) + "m"
			if string(pairSeq[a][b]) != want {
				t.Fatalf("pairSeq[%d][%d] = %q, want %q", a, b, pairSeq[a][b], want)
			}
		}
	}
}

// TestFrameFitsBuf steps many frames and verifies the frame buffer never grows
// past its precomputed capacity (the no-realloc guarantee bufCap promises).
func TestFrameFitsBuf(t *testing.T) {
	f := fire.New(benchW, benchH)
	r := New(f)
	c0 := cap(r.buf)
	var sink bytes.Buffer
	for range 500 {
		f.Step()
		r.Frame(&sink, f)
		sink.Reset()
	}
	if cap(r.buf) != c0 {
		t.Fatalf("frame buffer grew: cap %d -> %d", c0, cap(r.buf))
	}
}

// TestResizeReuse verifies resizing within capacity reuses buffers (0 allocs)
// and that the first frame after a resize is a full redraw.
func TestResizeReuse(t *testing.T) {
	f := fire.New(benchW, benchH)
	r := New(f)
	allocs := testing.AllocsPerRun(10, func() {
		f.Resize(benchW/2, benchH/2)
		r.Resize(f)
		f.Resize(benchW, benchH)
		r.Resize(f)
	})
	if allocs != 0 {
		t.Fatalf("resize within capacity allocated %v times", allocs)
	}

	f.Resize(60, 30)
	r.Resize(f)
	var out bytes.Buffer
	r.Frame(&out, f)
	if got := bytes.Count(out.Bytes(), []byte("▀")); got != 60*15 {
		t.Fatalf("full redraw drew %d cells, want %d", got, 60*15)
	}
}
