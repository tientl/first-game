package sim

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"slices"
	"testing"
)

// worldHash is a digest of everything generation decides: the tiles, the
// entities, the creatures, the villages and the spawn.
func worldHash(w *World) uint64 {
	h := fnv.New64a()
	put := func(vs ...uint64) {
		var b [8]byte
		for _, v := range vs {
			binary.LittleEndian.PutUint64(b[:], v)
			h.Write(b[:])
		}
	}
	tiles := make([]byte, len(w.tiles))
	for i, t := range w.tiles {
		tiles[i] = byte(t)
	}
	h.Write(tiles)
	put(uint64(w.width), uint64(w.height), math.Float64bits(w.spawnX), math.Float64bits(w.spawnY))
	ids := make([]EntityID, 0, len(w.entities))
	for id := range w.entities {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		e := w.entities[id]
		big := uint64(0)
		if e.Big {
			big = 1
		}
		put(uint64(e.ID), uint64(e.Kind), uint64(e.TX), uint64(e.TY), uint64(e.Var), uint64(e.HP), big)
	}
	cs := w.Creatures()
	slices.SortFunc(cs, func(a, b Creature) int { return int(a.ID) - int(b.ID) })
	for _, c := range cs {
		put(uint64(c.ID), uint64(c.Kind), math.Float64bits(c.X), math.Float64bits(c.Y))
	}
	for _, v := range w.villages {
		put(math.Float64bits(v.X), math.Float64bits(v.Y))
	}
	return h.Sum64()
}

// goldenWorlds are worlds of every older style, with the digest each made
// when StyleIsles was added. A change to generation must never change them:
// saves keep only the seed and style, and regenerate the terrain from it.
var goldenWorlds = []struct {
	style WorldStyle
	seed  uint64
	size  int
	want  uint64
}{
	{StyleIsland, 42, 96, 0x90715a64acfbd6ff},
	{StyleOpenV1, 3, 160, 0xc84b9c7276d91763},
	{StyleOpenV2, 3, 160, 0x7f7ed89fc7393028},
	{StyleOpen, 3, 160, 0xea4f4539583f9e1},
	{StyleIce, 5, 192, 0x123fa153f1799ef4},
	{StyleJungle, 5, 192, 0xe2235b4cf1b0ec69},
	{StyleTribal, 7, SizeMedium, 0x9add9592bbc2fe81},
	{StyleSeas, 7, SizeMedium, 0x9ce15ddca64da26f},
	{StyleSeas, 11, SizeLarge, 0x81a3e39fa5e3573f},
}

func TestOlderStylesGenerateAsBefore(t *testing.T) {
	for _, g := range goldenWorlds {
		w, err := Generate(GenConfig{Seed: g.seed, Width: g.size, Height: g.size, Style: g.style})
		if err != nil {
			t.Fatal(err)
		}
		if got := worldHash(w); got != g.want {
			t.Errorf("style %d seed %d size %d: digest %#x, want %#x", g.style, g.seed, g.size, got, g.want)
		}
	}
}
