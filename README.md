# ghostty-fire-simulation

The classic DOOM fire effect, animated across your whole terminal. Written in Go
and tuned for [Ghostty](https://ghostty.org).

https://github.com/frostbyte57/ghostty-fire-simulation/raw/main/assets/demo.mov

## Run

```sh
go run .
```

Pick your flame colors in the interactive designer, press `Enter` to ignite,
`q` to quit. Zero dependencies — pure Go standard library.

## How it works

Runs at 60 fps with zero allocations on the hot path. The tricks:

- **Precomputed escapes** — every color sequence is built once at startup; the
  render loop just copies bytes.
- **Frame diffing** — only changed cells are redrawn, assembled into one buffer
  and flushed in a single write per frame.
- **Synchronized output** (DEC 2026) so each frame is presented atomically — no
  tearing — over a cache-friendly `uint8` grid.

A frame takes ~220 µs (sim + draw), about 1.3% of the 16.7 ms budget.
