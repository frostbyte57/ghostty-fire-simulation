package fire

// N is the number of heat levels in the palette (a cell's heat indexes it).
const N = 37

// RGB is a 24-bit color.
type RGB = [3]uint8

// Palette ramps from the coldest heat (index 0) to the hottest (index N-1). It is
// the live palette used by the renderer; build a custom one with BuildPalette and
// assign it before rendering. The default reproduces the classic DOOM fire
// (black -> red -> orange -> yellow -> white).
var Palette = BuildPalette(ClassicStops)

// ClassicStops are the gradient anchors of the original DOOM fire, coldest first.
var ClassicStops = []RGB{
	{0x07, 0x07, 0x07}, // Base (coldest)
	{0x9F, 0x2F, 0x07}, // Primary
	{0xDF, 0x5F, 0x07}, // Secondary
	{0xCF, 0xCF, 0x6F}, // Tertiary
	{0xFF, 0xFF, 0xFF}, // Peak (hottest)
}

// MaxHeat is the hottest palette index — the fire source temperature.
const MaxHeat = uint8(N - 1)

// BuildPalette spreads the given color stops evenly across the N heat levels and
// linearly interpolates between them, so any number of stops (>=1) yields a smooth
// N-entry ramp. stops[0] is the coldest color, stops[len-1] the hottest.
func BuildPalette(stops []RGB) [N][3]uint8 {
	var p [N][3]uint8
	switch len(stops) {
	case 0:
		return p
	case 1:
		for i := range p {
			p[i] = stops[0]
		}
		return p
	}
	segs := len(stops) - 1
	for i := range N {
		x := float64(i) / float64(N-1) * float64(segs) // position in [0, segs]
		s := int(x)
		if s >= segs {
			s = segs - 1
		}
		f := x - float64(s)
		a, b := stops[s], stops[s+1]
		p[i] = RGB{lerp(a[0], b[0], f), lerp(a[1], b[1], f), lerp(a[2], b[2], f)}
	}
	return p
}

func lerp(a, b uint8, f float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*f + 0.5)
}
