package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Fireflies drift over grass, woods and marsh from late dusk into the night,
// blinking on and off. Like the embers they are a pure function of place and
// time: each tile rolls whether a firefly lives over it, and its path and
// blink come from the clock, so there is nothing to keep between frames.

// fireflyChance is how likely a firefly lives over a tile of t.
func fireflyChance(t sim.Tile) float64 {
	switch t {
	case sim.TileMarsh:
		return 0.12
	case sim.TileForest:
		return 0.07
	case sim.TileGrass:
		return 0.04
	}
	return 0
}

const (
	fireflyRoam  = 1.1   // how far, in tiles, a firefly strays from its tile
	fireflyFade  = 0.012 // of a day: how long one takes to come out or settle
	fireflyFlash = 0.3   // of its blink period: how long a flash lasts
)

// firefly is one firefly where and how brightly it is right now.
type firefly struct {
	x, y   float64 // over the ground, in tiles
	height float64 // above the ground, in art pixels
	glow   float64 // 0 (dark or not out) to 1 (at the top of a flash)
}

// fireflyAt returns the firefly that lives over tile (tx, ty) of kind t, at
// day fraction f and secs seconds of the client's clock, and whether there is
// one out at all. Each comes out somewhere in late dusk or early night and
// settles somewhere in the night, so they thin out towards dawn.
func fireflyAt(tx, ty int, t sim.Tile, f, secs float64) (firefly, bool) {
	chance := fireflyChance(t)
	if chance == 0 {
		return firefly{}, false
	}
	// Clumped: some meadows and groves have many, others few.
	chance *= 0.4 + 1.2*vnoise(float64(tx)/5, float64(ty)/5, 731)
	if hashf(tx, ty, 730) >= chance {
		return firefly{}, false
	}
	h := func(i uint32) float64 { return hashf(tx, ty, 740+i) }

	start := sim.DuskStart + 0.06 + h(0)*(sim.NightStart-sim.DuskStart)
	end := 0.84 + h(1)*0.14
	life := clamp01((f-start)/fireflyFade) * clamp01((end-f)/fireflyFade)
	if life <= 0 {
		return firefly{}, false
	}

	// A slow, wandering loop around a spot on the tile.
	w := func(i uint32) float64 { return 0.25 + 0.4*h(i) }
	p := func(i uint32) float64 { return 2 * math.Pi * h(i) }
	dx := 0.7*math.Sin(secs*w(2)+p(3)) + 0.3*math.Sin(secs*w(4)*1.7+p(5))
	dy := 0.5*math.Sin(secs*w(6)+p(7)) + 0.3*math.Cos(secs*w(8)*1.9+p(9))
	ff := firefly{
		x:      float64(tx) + 0.5 + (h(10)-0.5)*0.6 + dx*fireflyRoam,
		y:      float64(ty) + 0.5 + (h(11)-0.5)*0.6 + dy*fireflyRoam,
		height: 10 + 4*math.Sin(secs*(0.6+0.5*h(12))+p(13)),
	}

	// The blink: a short flash every few seconds, a faint glow between.
	period := 2.2 + 1.8*h(14)
	ph := math.Mod(secs/period+h(15), 1)
	flash := 0.0
	if ph < fireflyFlash {
		s := math.Sin(math.Pi * ph / fireflyFlash)
		flash = s * s
	}
	ff.glow = life * (0.1 + 0.9*flash)
	return ff, true
}

// drawFireflies draws the fireflies in view, over the darkness so they glow.
func (g *Game) drawFireflies(dst drawTarget, cam camera) {
	if g.caveIn() > 0.5 || g.tiles == nil {
		return // underground: no fireflies
	}
	f := sim.DayFraction(g.curr.Tick)
	if f < sim.DuskStart || f > 0.99 {
		return // none out at all: skip the tile walk
	}
	secs := float64(g.frame) / 60
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	const pad = 2 // tiles: a firefly may roam into view from off screen
	tx0, ty0 := int(math.Floor(cam.ox/TileW))-pad, int(math.Floor(cam.oy/TileH))-pad
	tx1, ty1 := int(math.Ceil((cam.ox+w)/TileW))+pad, int(math.Ceil((cam.oy+h)/TileH))+pad
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			ff, ok := fireflyAt(tx, ty, g.tileAt(tx, ty), f, secs)
			if !ok || ff.glow < 0.02 {
				continue
			}
			sx, sy := cam.toScreen(ff.x, ff.y)
			sy -= math.Round(ff.height)
			k := float32(ff.glow)

			// A wide, faint yellow-green halo and a small bright one, added
			// to the scene...
			for _, l := range [...]struct{ r, k float32 }{{0.32, 0.45}, {0.11, 0.9}} {
				op := g.lightOpAt(sx+0.5, sy+0.5, float64(l.r))
				kk := k * l.k
				op.ColorScale.Scale(0.55*kk, 0.8*kk, 0.2*kk, kk)
				op.Blend = ebiten.BlendLighter
				dst.DrawImage(g.art.light, op)
			}
			// ...around a bright one-pixel body.
			a := min(1, 0.25+k)
			fillRect(dst, float32(sx), float32(sy), 1, 1,
				color.RGBA{uint8(225 * a), uint8(255 * a), uint8(130 * a), uint8(255 * a)}, false)
		}
	}
}
