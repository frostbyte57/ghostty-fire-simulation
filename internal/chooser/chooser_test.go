package chooser

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   []byte
		want action
	}{
		{[]byte{0x1b, '[', 'A'}, actUp},
		{[]byte{0x1b, '[', 'B'}, actDown},
		{[]byte{0x1b, '[', 'C'}, actRight},
		{[]byte{0x1b, '[', 'D'}, actLeft},
		{[]byte{'k'}, actUp},
		{[]byte{'j'}, actDown},
		{[]byte{'h'}, actLeft},
		{[]byte{'l'}, actRight},
		{[]byte{'\r'}, actConfirm},
		{[]byte{' '}, actConfirm},
		{[]byte{'q'}, actQuit},
		{[]byte{3}, actQuit},    // Ctrl-C
		{[]byte{0x1b}, actQuit}, // lone Esc
		{[]byte{'z'}, actNone},
	}
	for _, c := range cases {
		if got := parse(c.in); got != c.want {
			t.Errorf("parse(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRunQuitReturnsNotOK(t *testing.T) {
	_, ok := Run(io.Discard, strings.NewReader("q"))
	if ok {
		t.Fatal("quit should return ok=false")
	}
}

func TestRunConfirmDefaults(t *testing.T) {
	stops, ok := Run(io.Discard, strings.NewReader("\r"))
	if !ok {
		t.Fatal("Enter should confirm")
	}
	if len(stops) != 5 {
		t.Fatalf("got %d stops, want 5", len(stops))
	}
}

func TestRunRightChangesBaseSlot(t *testing.T) {
	// Default selection (Enter) vs. one right-press on the Base slot then Enter:
	// only the first stop should differ, advancing to the next swatch color.
	def, _ := Run(io.Discard, strings.NewReader("\r"))
	got, ok := Run(io.Discard, bytes.NewReader([]byte{0x1b, '[', 'C', '\r'})) // →, Enter
	if !ok {
		t.Fatal("expected confirm")
	}
	if got[0] == def[0] {
		t.Errorf("Base color unchanged after →: %v", got[0])
	}
	for i := 1; i < len(def); i++ {
		if got[i] != def[i] {
			t.Errorf("stop %d changed unexpectedly: %v != %v", i, got[i], def[i])
		}
	}
}

func TestRunDownThenRightChangesSecondSlot(t *testing.T) {
	def, _ := Run(io.Discard, strings.NewReader("\r"))
	// ↓ to Primary, → to change it, Enter. Base stays, Primary changes.
	got, ok := Run(io.Discard, bytes.NewReader([]byte{0x1b, '[', 'B', 0x1b, '[', 'C', '\r'}))
	if !ok {
		t.Fatal("expected confirm")
	}
	if got[0] != def[0] {
		t.Errorf("Base should be unchanged: %v != %v", got[0], def[0])
	}
	if got[1] == def[1] {
		t.Errorf("Primary should have changed: still %v", got[1])
	}
}

func TestNearestSnapsToSwatch(t *testing.T) {
	// An exact swatch color must snap to itself.
	for _, s := range swatches {
		if got := nearest(s.rgb); got != s.rgb {
			t.Errorf("nearest(%v) = %v, want identity", s.rgb, got)
		}
	}
}
