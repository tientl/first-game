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
// when StyleIsles was added (StyleIsles itself: when StyleWoods was;
// StyleWoods: when StyleBeaches was; StyleBeaches, with its Ankylosaurus:
// when StyleWilds was; StyleWilds: when StyleCaves was). A
// change to generation must never change them: saves keep only the seed and
// style, and regenerate the terrain from it; an older save's woods are
// thinned by telling the trees of the world as it was made (thinwoods.go).
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
	{StyleIsles, 7, SizeMedium, 0x8227ef130cb50cf6},
	{StyleIsles, 11, SizeLarge, 0xe72383ba4feab520},
	{StyleWoods, 7, SizeMedium, 0x37d7fffed95b64bd},
	{StyleWoods, 11, SizeLarge, 0x3697e41d31d3f72e},
	{StyleBeaches, 7, SizeMedium, 0x58216778253cdef8},
	{StyleBeaches, 11, SizeLarge, 0x484ee25ca916f3cc},
	{StyleWilds, 7, SizeMedium, 0xd251e0e44a29943d},
	{StyleWilds, 11, SizeLarge, 0xd5d9c6fc707aa1f7},
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

// goldenRelief is the digest of the relief (levels and variants) of worlds
// of the older styles that have one, as each made when StyleBeaches was
// added (StyleBeaches: when StyleWilds was; StyleWilds: when StyleCaves was): the client draws the ground
// from it, so it must not change either.
var goldenRelief = []struct {
	style WorldStyle
	seed  uint64
	size  int
	want  uint64
}{
	{StyleIsles, 7, SizeMedium, 0x2fca9269f95872cc},
	{StyleIsles, 11, SizeLarge, 0x786d9383d492a24d},
	{StyleWoods, 7, SizeMedium, 0x474bc54a6fa098fc},
	{StyleWoods, 11, SizeLarge, 0xacecdb3a07441921},
	{StyleBeaches, 7, SizeMedium, 0x7bca2db16762a488},
	{StyleBeaches, 11, SizeLarge, 0x8e93bec7cc5e52fb},
	{StyleWilds, 7, SizeMedium, 0xea1548751b14e4e7},
	{StyleWilds, 11, SizeLarge, 0x5277314ef9ec7b17},
}

func TestOlderReliefIsAsBefore(t *testing.T) {
	for _, g := range goldenRelief {
		w, err := Generate(GenConfig{Seed: g.seed, Width: g.size, Height: g.size, Style: g.style})
		if err != nil {
			t.Fatal(err)
		}
		h := fnv.New64a()
		b := make([]byte, len(w.relief))
		for i, r := range w.relief {
			b[i] = byte(r)
		}
		h.Write(b)
		if got := h.Sum64(); got != g.want {
			t.Errorf("style %d seed %d size %d: relief digest %#x, want %#x", g.style, g.seed, g.size, got, g.want)
		}
	}
}
