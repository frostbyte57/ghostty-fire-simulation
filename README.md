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

Runs at 60 fps, allocation-free on the hot path. At 200×50 cells on an M2 Pro:

| Stage              | Time      | Allocs |
| ------------------ | --------- | ------ |
| `Step` (sim)       | ~80 µs    | 0      |
| `Frame` (sim+draw) | ~220 µs   | 0      |

That leaves a single frame using ~1.3% of the 16.7 ms budget, so the animation
stays smooth with enormous headroom. Achieved by:

- **Fully precomputed escapes** — every foreground, background, _and_ combined
  fg+bg SGR sequence is built once at startup; the render loop only copies bytes.
- **One write per frame** — each frame is assembled into a single reusable buffer
  (pre-sized so it never reallocates mid-animation) and flushed in one syscall.
- **Frame diffing** — only cells that changed since the last frame are emitted.
- **Zero GC pressure** — no allocations on the hot path means no GC pauses, the
  main source of animation jitter.
- **Synchronized output** (DEC 2026) so each frame is presented atomically — no
  tearing — plus an inline xorshift PRNG and a cache-friendly `uint8` grid.

Benchmarks: `go test -bench=. -benchmem ./...`.

Zero dependencies — pure Go standard library.
