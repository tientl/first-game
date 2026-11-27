package sim

import (
	"slices"
	"testing"
)

// seaLilies counts the lilies and lotus of w standing on water joined to
// the open sea, and those on ponds.
func seaLilies(w *World) (sea, pond int) {
	joined := seaJoined(w.width, w.height, w.tiles)
	for _, e := range w.entities {
		if e.Kind != EntityFlora || !FloraKind(e.Var).Pad() {
			continue
		}
		if joined[e.TY*w.width+e.TX] {
			sea++
		} else {
			pond++
		}
	}
	return sea, pond
}

func TestNoLilyGrowsOnTheSea(t *testing.T) {
	for _, g := range []struct {
		seed uint64
		size int
	}{{7, SizeMedium}, {11, SizeLarge}} {
		w, err := Generate(GenConfig{Seed: g.seed, Width: g.size, Height: g.size, Style: StyleLairs})
		if err != nil {
			t.Fatal(err)
		}
		sea, pond := seaLilies(w)
		t.Logf("seed %d size %d: %d lilies and lotus on ponds, %d on the sea", g.seed, g.size, pond, sea)
		if sea > 0 {
			t.Errorf("seed %d size %d: %d lilies or lotus on water joined to the sea", g.seed, g.size, sea)
		}
	}
}

func TestOlderStylesKeepTheirPonds(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 11, Width: SizeLarge, Height: SizeLarge, Style: StyleLairs - 1})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(PondMaskFor(StyleLairs-1, w.width, w.height, w.tiles), PondMask(w.width, w.height, w.tiles)) {
		t.Errorf("the ponds of a world of the style before StyleLairs changed")
	}
	if slices.Equal(PondMaskFor(StyleLairs, w.width, w.height, w.tiles), PondMask(w.width, w.height, w.tiles)) {
		t.Errorf("by the rule of StyleLairs the same world's sea pockets are still ponds: the test proves nothing")
	}
}

func TestAnOldSaveHasItsSeaLiliesPulled(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 11, Width: SizeLarge, Height: SizeLarge, Style: StyleLairs - 1})
	if err != nil {
		t.Fatal(err)
	}
	sea, pond := seaLilies(w)
	if sea == 0 {
		t.Fatalf("no lily on the sea in the older world: pick another seed")
	}
	d := w.Save()
	d.Version = SaveVersion - 1
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	sea2, pond2 := seaLilies(w2)
	if sea2 != 0 || pond2 != pond {
		t.Errorf("loaded: %d on the sea (was %d), %d on ponds (was %d); want none on the sea and the ponds' kept", sea2, sea, pond2, pond)
	}
	// Only once: a save of this version keeps what it has.
	d = w.Save()
	w3, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if s3, _ := seaLilies(w3); s3 != sea {
		t.Errorf("a save of version %d lost its sea lilies too: %d of %d", SaveVersion, s3, sea)
	}
}
