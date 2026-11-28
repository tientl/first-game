package client

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// CacheStat is how many entries one of the client's caches holds and, for
// those that hold images, how many bytes of image (4 per pixel) they take.
type CacheStat struct {
	Name  string
	N     int
	Bytes int64
}

func imgBytes(img *ebiten.Image) int64 {
	if img == nil {
		return 0
	}
	b := img.Bounds()
	return int64(4 * b.Dx() * b.Dy())
}

func sprBytes[K comparable](m map[K]sprite) (n int, bytes int64) {
	for _, s := range m {
		bytes += imgBytes(s.img)
	}
	return len(m), bytes
}

// CacheStats counts what the client's process-wide caches hold, biggest
// first, for the soak tool (cmd/soak) to see which ones grow as a player
// travels. It must be called from the game loop's goroutine, as the caches
// are not locked.
func CacheStats() []CacheStat {
	var out []CacheStat
	add := func(name string, n int, bytes int64) { out = append(out, CacheStat{name, n, bytes}) }
	addS := func(name string, n int, bytes int64) { add(name, n, bytes) }

	chunks, gbytes, _ := GroundCost()
	add("ground chunks", chunks, gbytes)
	ab, _, _ := ArtCost()
	add("art drawn in code (all, cumulative)", 0, ab)

	n, b := 0, int64(0)
	for _, f := range rigFrames {
		n++
		b += imgBytes(f.s.img)
	}
	add("dino rig frames", n, b)
	n, b = sprBytes(critCache)
	addS("critter frames", n, b)
	n = 0
	b = 0
	for k, s := range mirrorCache {
		_ = k
		n++
		b += imgBytes(s.img)
	}
	add("mirror images", n, b)
	n, b = sprBytes(fallenCache)
	addS("fallen cache", n, b)
	n, b = sprBytes(fallenCacheBody)
	addS("fallen body cache", n, b)
	n, b = sprBytes(hairCache)
	addS("hair cache", n, b)
	n, b = sprBytes(laidCache)
	addS("laid cache", n, b)
	n, b = sprBytes(toadCache)
	addS("toad cache", n, b)
	n, b = sprBytes(ankyloSmears)
	addS("ankylo smears", n, b)
	add("leg cuts", len(legCuts), 0)

	n, b = 0, 0
	for _, i := range iconBaked {
		n++
		b += imgBytes(i)
	}
	add("item icons baked", n, b)
	n, b = 0, 0
	for _, i := range uiIconBaked {
		n++
		b += imgBytes(i)
	}
	add("ui icons baked", n, b)
	n, b = 0, 0
	for _, t := range textBakes {
		n++
		b += imgBytes(t.img)
	}
	add("text bakes", n, b)
	add("text widths", len(textWidths), 0)
	n, b = 0, 0
	for _, p := range panelBakes {
		n++
		b += imgBytes(p.img)
	}
	for _, p := range pieceBakes {
		n++
		b += imgBytes(p.img)
	}
	add("panel bakes", n, b)
	n = 0
	coverMasks.Range(func(_, _ any) bool { n++; return true })
	add("cover masks", n, 0)

	add("voice cooldowns", len(voices.cool), 0)
	add("voice engaged", len(voices.engaged), 0)
	add("boss clocks", len(bossClocks), 0)
	add("blow clocks", len(blowClocks), 0)
	add("strike clocks", len(strikeClocks), 0)
	add("skill clocks", len(skillClocks), 0)
	add("held marks", len(heldMarks), 0)

	sort.SliceStable(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out
}
