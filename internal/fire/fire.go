// Package fire implements the DOOM PSX fire simulation: a heat grid whose bottom
// row is a constant source and whose cells cool as they drift upward.
package fire

import "time"

// Fire is the heat grid. width == terminal columns; height == terminal rows*2,
// since the renderer packs two vertical pixels per cell.
type Fire struct {
	width, height int
	pixels        []uint8
	rng           uint32
}

func New(width, height int) *Fire {
	f := &Fire{
		width:  width,
		height: height,
		pixels: make([]uint8, width*height),
		rng:    uint32(time.Now().UnixNano()) | 1,
	}
	f.ignite()
	return f
}

func (f *Fire) Width() int      { return f.width }
func (f *Fire) Height() int     { return f.height }
func (f *Fire) Pixels() []uint8 { return f.pixels }

func (f *Fire) ignite() {
	bottom := (f.height - 1) * f.width
	for x := range f.width {
		f.pixels[bottom+x] = MaxHeat
	}
}

// rand3 returns a value in {0,1,2} via inline xorshift32.
func (f *Fire) rand3() int {
	x := f.rng
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	f.rng = x
	return int(x % 3)
}

// Step advances the fire one frame. Iterated row-major so the read row (y) and
// write row (y-1) stay cache-hot.
func (f *Fire) Step() {
	w := f.width
	for y := 1; y < f.height; y++ {
		rowBase := y * w
		upBase := rowBase - w
		for x := range w {
			heat := f.pixels[rowBase+x]
			if heat == 0 {
				f.pixels[upBase+x] = 0
				continue
			}
			decay := f.rand3()
			dstX := x - decay + 1
			if dstX < 0 {
				dstX = 0
			} else if dstX >= w {
				dstX = w - 1
			}
			f.pixels[upBase+dstX] = heat - uint8(decay&1)
		}
	}
}
