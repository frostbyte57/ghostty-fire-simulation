# ghostty-fire-simulation

The classic DOOM fire effect, animated across your whole terminal. Written in Go
and tuned for [Ghostty](https://ghostty.org) (24-bit truecolor + synchronized
output). Each character cell draws two pixels via the half-block glyph `▀`.

## Usage

```sh
go run .
# or
go build -o doomfire . && ./doomfire
```

Press `q` or `Ctrl-C` to quit. The fire fills the window and re-fits on resize.

## Performance

Allocation-free on the hot path. At 200×50 cells on an M2 Pro:

| Stage              | Time      | Allocs |
| ------------------ | --------- | ------ |
| `Step` (sim)       | ~80 µs    | 0      |
| `Frame` (sim+draw) | ~310 µs   | 0      |

Achieved by precomputing color escapes, diffing each frame against the last (only
changed cells are sent), an inline xorshift PRNG, and a cache-friendly `uint8`
grid. Benchmarks: `go test -bench=. -benchmem ./...`.

Zero dependencies — pure Go standard library.
