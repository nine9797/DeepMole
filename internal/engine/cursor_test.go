package engine

import (
	"testing"

	"deepmole/internal/strategy"
)

func TestCursorPresetsThenNeighbors(t *testing.T) {
	c := newCursor(nil, nil)
	for i := range strategy.Proven {
		if !c.next() {
			t.Fatalf("stopped at preset %d", i)
		}
		if c.deep {
			t.Fatalf("neighbors started before presets ended at %d", i)
		}
		if c.cur.Key() != strategy.Proven[i].Key() {
			t.Fatalf("preset %d out of order", i)
		}
	}
	c.best, c.score = strategy.Proven[2], 0.5
	n := 0
	for c.next() {
		if !c.deep {
			t.Fatal("deep flag not set")
		}
		if c.tried[c.cur.Key()] != true {
			t.Fatal("untracked candidate")
		}
		n++
		if n > 100 {
			break
		}
	}
	if n < 10 {
		t.Fatalf("too few neighbors: %d", n)
	}
}

func TestCursorSeedsWithoutBest(t *testing.T) {
	c := newCursor(nil, nil)
	for c.next() && !c.deep {
	}
	if !c.deep {
		t.Fatal("no fallback neighbors without a partial result")
	}
	seen := map[string]bool{c.cur.Key(): true}
	for c.next() {
		if seen[c.cur.Key()] {
			t.Fatal("candidate repeated")
		}
		seen[c.cur.Key()] = true
	}
	if len(seen) < 2*fallbackSeeds {
		t.Fatalf("expected neighbors of %d seeds, got %d", fallbackSeeds, len(seen))
	}
}

func TestCursorSeedFirst(t *testing.T) {
	seed := strategy.Proven[5]
	c := newCursor(&seed, nil)
	if !c.next() || c.cur.Key() != seed.Key() {
		t.Fatal("saved strategy must be tried first")
	}
	for c.next() {
		if c.cur.Key() == seed.Key() {
			t.Fatal("saved strategy tried twice")
		}
		if c.deep {
			break
		}
	}
}
