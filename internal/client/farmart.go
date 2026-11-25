package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Pebbles on the ground, the hoe, tilled soil and sweet potatoes: their art,
// and the mark on the tile a held hoe would till.

var (
	soilRamp   = mkRamp(0x2a1c13, 0x3b281a, 0x4f3622, 0x66472c, 0x7d5a38)
	potatoRamp = mkRamp(0x3d1630, 0x5e2245, 0x86325a, 0xa84a6c, 0xc8708a)
	stemCol    = hex(0x3f6a2e)
)

// farmArt is the farming part of the art.
type farmArt struct {
	pebbles  sprite
	farmland sprite                     // flat: drawn with the ground, under everything
	crops    [sim.CropStages + 1]sprite // growing stages, then ripe
}

func genFarmArt() farmArt {
	var f farmArt
	f.pebbles = genPebbles()
	f.farmland = genFarmland()
	for s := range f.crops {
		f.crops[s] = genCrop(s)
	}
	return f
}

// genFarmIcons draws the bag icons of the pebble, hoe and sweet potatoes.
func genFarmIcons(icons *[sim.NumItems]*ebiten.Image) {
	icons[sim.ItemPebble] = iconPebbles()
	icons[sim.ItemPotato] = iconPotato(false)
	icons[sim.ItemRoastPotato] = iconPotato(true)
}

// stones draws round stones, back to front, each lit from the top-left.
func stones(c *canvas, ss []struct{ x, y, rx, ry, l float64 }) {
	for _, s := range ss {
		c.ellipse(s.x, s.y, s.rx, s.ry, func(x, y int, nx, ny float64) color.RGBA {
			return stoneRamp.pick(lum(nx, ny)*s.l, x, y)
		})
	}
}

func genPebbles() sprite {
	c := newCanvas(16, 10)
	stones(c, []struct{ x, y, rx, ry, l float64 }{
		{8, 4.5, 2.2, 1.8, 1}, {5, 6.5, 3, 2.3, 0.9}, {10.5, 7, 2.6, 2, 0.75},
	})
	c.outline(0.7)
	return mk(c, 8, 8)
}

func iconPebbles() *ebiten.Image {
	c := newCanvas(16, 16)
	stones(c, []struct{ x, y, rx, ry, l float64 }{
		{9, 6, 3.4, 2.8, 1}, {5.5, 10, 4, 3.2, 0.9}, {11, 11, 3.4, 2.7, 0.75},
	})
	c.outline(0.8)
	return c.image()
}

// genFarmland is a bed of tilled soil a little smaller than a tile, with
// furrows across it.
func genFarmland() sprite {
	c := newCanvas(22, 15)
	const cx, cy, rx, ry = 11.0, 7.5, 10.5, 7.0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if nx*nx*nx*nx+ny*ny*ny*ny > 1 {
				continue
			}
			// Ridges every 4 rows: lit on top, shadowed under.
			l := [4]float64{0.85, 0.6, 0.3, 0.15}[(y+1)%4]
			l += (vnoise(float64(x)*0.5, float64(y)*0.7, 500)-0.5)*0.25 - math.Abs(nx)*0.1
			c.set(x, y, soilRamp.pick(l, x, y))
		}
	}
	c.outline(0.35)
	return mk(c, 11, 8)
}

// genCrop is a sweet potato plant, without its soil: stage 0 to
// sim.CropStages-1 growing, sim.CropStages ripe, with the tubers showing.
func genCrop(stage int) sprite {
	sizes := [sim.CropStages + 1]struct {
		w, h int
		r    float64
	}{{12, 9, 2.2}, {16, 12, 3.1}, {20, 15, 4}, {22, 17, 4.4}}
	sz := sizes[stage]
	ripe := stage == sim.CropStages
	c := newCanvas(sz.w, sz.h)
	cx, base := math.Round(float64(sz.w)/2), float64(sz.h-3)
	rng := rand.New(rand.NewPCG(uint64(stage)+31, 7))

	n := 2 + 2*stage
	var blobs []blob
	for i := range n {
		a := math.Pi * (0.08 + 0.84*float64(i)/float64(n-1))
		d := sz.r * (0.7 + rng.Float64()*0.5)
		blobs = append(blobs, blob{
			cx - math.Cos(a)*d*1.35, base - sz.r*0.5 - math.Sin(a)*d,
			sz.r * (0.5 + rng.Float64()*0.2),
		})
	}
	for _, b := range blobs {
		c.line(cx, base, b.x, b.y, func(int, int, float64) color.RGBA { return stemCol })
	}
	leafCluster(c, blobs, cx, base-sz.r, sz.r*1.6, sz.r*1.3, leafRamps[1], 300+uint32(stage))
	if ripe {
		for _, t := range []struct{ x, y, rx, ry float64 }{{cx - 4, base + 0.5, 2.6, 1.6}, {cx + 3.5, base + 1, 2.4, 1.5}} {
			c.ellipse(t.x, t.y, t.rx, t.ry, func(x, y int, nx, ny float64) color.RGBA {
				return potatoRamp.pick(lum(nx, ny), x, y)
			})
		}
	}
	c.outline(0.7)
	return mk(c, cx, base)
}

func iconPotato(roast bool) *ebiten.Image {
	c := newCanvas(16, 16)
	rp := potatoRamp
	if roast {
		rp = roastRamp
	}
	// A long tuber lying across the icon: two overlapping lobes.
	for _, l := range []struct{ x, y, rx, ry float64 }{{6, 10, 4.2, 3.2}, {10, 7, 4.2, 3.3}} {
		c.ellipse(l.x, l.y, l.rx, l.ry, func(x, y int, nx, ny float64) color.RGBA {
			return rp.pick(lum(nx, ny)*0.95, x, y)
		})
	}
	if roast {
		grillMarks(c, 2)
	} else {
		for _, e := range [][2]int{{6, 9}, {9, 6}, {11, 8}} { // eyes
			c.set(e[0], e[1], rp[1])
		}
		c.set(1, 12, rp[1]) // root tails
		c.set(14, 4, rp[1])
	}
	c.outline(0.8)
	return c.image()
}

// cropSprite is the plant of a crop entity (its soil is drawn with the ground).
func (g *Game) cropSprite(e protocol.EntityView) sprite {
	if e.Ready {
		return g.art.farm.crops[sim.CropStages]
	}
	return g.art.farm.crops[min(int(e.Stage), sim.CropStages-1)]
}

// drawSoil draws the tilled soil under farmland and crops, before anything
// upright, so players and plants stand on it.
func (g *Game) drawSoil(dst drawTarget, list []drawable) {
	for _, d := range list {
		if d.isPlayer || d.creature || d.drop || d.boat {
			continue
		}
		if e := g.curr.Entities[d.idx]; e.Kind == sim.EntityFarmland || e.Kind == sim.EntityCrop {
			drawAt(dst, g.art.farm.farmland, d.sx, d.sy, 1, 1, 0, nil)
			if e.Kind == sim.EntityFarmland && e.Big {
				drawAt(dst, fedSoilSprite(), d.sx, d.sy, 1, 1, 0, nil) // fed with dung
			}
		}
	}
}

// digSpot is the tile my held hoe would till now, if any: the rule the
// server uses.
func (g *Game) digSpot() (tx, ty int, ok bool) {
	for _, p := range g.curr.Players {
		if p.ID != g.myID || p.Dead || !isHoe(p.Held) || p.Cast != sim.CastNone {
			continue
		}
		if ax, ay, aim := g.aim(); aim {
			if u, in := g.underPointer(p, ax, ay); u.ok && in {
				return 0, 0, false // what the pointer is on comes first, as on the server
			}
		}
		return sim.DigSpot(float64(p.X), float64(p.Y), p.Dir, g.tileAt, func(tx, ty int) bool {
			_, taken := g.entAt[[2]int{tx, ty}]
			return taken
		})
	}
	return 0, 0, false
}

// drawDigSpot outlines the tile the held hoe would till.
func (g *Game) drawDigSpot(dst drawTarget, cam camera) {
	tx, ty, ok := g.digSpot()
	if !ok || g.curr.Self.Dead {
		return
	}
	x, y := cam.toScreen(float64(tx), float64(ty))
	a := uint8(110 + 70*math.Sin(float64(g.frame)*0.12))
	fillRect(dst, float32(x)+2, float32(y)+2, TileW-4, TileH-4, color.RGBA{uint8(int(soilRamp[2].R) * 60 / 255), uint8(int(soilRamp[2].G) * 60 / 255), uint8(int(soilRamp[2].B) * 60 / 255), 60}, false)
	strokeRect(dst, float32(x)+1.5, float32(y)+1.5, TileW-3, TileH-3, 1, color.RGBA{a, a, a, a}, false)
}

// drawDigLabel names what Space does on the marked tile, at screen resolution.
func (g *Game) drawDigLabel(screen *ebiten.Image, cam camera, scale float64) {
	tx, ty, ok := g.digSpot()
	if !ok || g.curr.Self.Dead {
		return
	}
	x, y := cam.toScreen(float64(tx)+0.5, float64(ty))
	g.text(screen, "Xới đất", x*scale, (y-4)*scale-18, colWhite, text.AlignCenter, 1)
}
