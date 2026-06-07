package fire

import "testing"

func TestBuildPaletteEndpoints(t *testing.T) {
	stops := []RGB{{10, 20, 30}, {200, 100, 50}, {255, 255, 255}}
	p := BuildPalette(stops)
	if p[0] != stops[0] {
		t.Errorf("coldest = %v, want %v", p[0], stops[0])
	}
	if p[N-1] != stops[len(stops)-1] {
		t.Errorf("hottest = %v, want %v", p[N-1], stops[len(stops)-1])
	}
}

func TestBuildPaletteSingleStop(t *testing.T) {
	p := BuildPalette([]RGB{{42, 42, 42}})
	for i := range p {
		if p[i] != (RGB{42, 42, 42}) {
			t.Fatalf("index %d = %v, want solid color", i, p[i])
		}
	}
}

func TestBuildPaletteMonotonicMidpoint(t *testing.T) {
	// Two stops -> the middle palette entry should sit between them on each channel.
	p := BuildPalette([]RGB{{0, 0, 0}, {255, 255, 255}})
	mid := p[N/2]
	if mid[0] == 0 || mid[0] == 255 {
		t.Errorf("midpoint not interpolated: %v", mid)
	}
}
