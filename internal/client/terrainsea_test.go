package client

import (
	"slices"
	"testing"

	"survivalgame/internal/sim"
)

// The shallows are drawn as one smooth depth: no pixel of the sea comes out
// see-through, and no two pixels side by side differ much in how much of
// the moving water under them shows, except where coral or foam is drawn
// (and in the shadow at a cliff's foot, which is not looked at). (Shoals once overflowed their alpha into blotches of bare water.)
func TestShallowsAreSmooth(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	w, err := sim.Generate(sim.GenConfig{Seed: 42, Width: 256, Height: 256, Style: sim.StyleLatest})
	if err != nil {
		t.Fatal(err)
	}
	W, H := w.Size()
	terr := newTerrain(W, H, w.Tiles())
	terr.setRelief(w.Relief())
	// The chunks with the most shoal, reef and lagoon in them.
	counts := map[chunkKey]int{}
	rel := w.Relief()
	for i, k := range w.Tiles() {
		if k == sim.TileShallow && rel[i].Variant() != sim.VarPlain {
			counts[chunkKey{i % W / chunkTiles, i / W / chunkTiles}]++
		}
	}
	var keys []chunkKey
	for k, n := range counts {
		if n > chunkTiles*chunkTiles/4 {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(a, b chunkKey) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		if a.cy != b.cy {
			return a.cy - b.cy
		}
		return a.cx - b.cx
	})
	if len(keys) == 0 {
		t.Fatal("no chunk with shoals, reefs or lagoons")
	}
	for _, k := range keys[:min(len(keys), 8)] {
		g, _, _ := terr.bakeChunk(k)
		alpha := func(x, y int) int { return int(g[(y*chunkPW+x)*4+3]) }
		jumps, sea := 0, 0
		for y := 1; y < chunkPH; y++ {
			for x := 1; x < chunkPW; x++ {
				u := (float64(k.cx*chunkPW+x) + 0.5) / hdTW
				v := (float64(k.cy*chunkPH+y) + 0.5) / hdTH
				if terr.coast(u, v) > -0.5 || terr.has(int(u), int(v), featCliff) {
					continue // the shore, its foam and the land; cliffs and their shadow
				}
				sea++
				a := alpha(x, y)
				if a < 160 {
					t.Fatalf("chunk %v: sea pixel %d,%d is %d/255 opaque", k, x, y, a)
				}
				if d := max(iabs(a-alpha(x-1, y)), iabs(a-alpha(x, y-1))); d > 12 {
					jumps++
				}
			}
		}
		if jumps*50 > sea {
			t.Errorf("chunk %v: %d of %d sea pixels jump in alpha from the next", k, jumps, sea)
		}
	}
}
