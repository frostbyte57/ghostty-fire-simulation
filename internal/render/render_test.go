package render

import (
	"bufio"
	"io"
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
