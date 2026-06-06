package fire

import "testing"

// Representative full-screen size: 200 cols x 50 rows => 200x100 pixel grid.
const benchW, benchH = 200, 100

func BenchmarkStep(b *testing.B) {
	f := New(benchW, benchH)
	for range 300 { // reach steady-state flame
		f.Step()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.Step()
	}
}
