package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Sleepers in beds and tents that face other ways (sim/facing.go,
// facedart.go): a bed turned about has its head on the right, so its
// sleeper is the usual one mirrored; a bed lying lengthways has its head
// up or down the screen, so its sleeper is the one laid across turned a
// quarter and drawn shorter, as the ground going back is (three quarters).
// They sit down on the bed's open side and then lie straight down. In a
// tent they lie across it as ever, their head away from its door.

// sleepLie is how someone asleep lies, and where their head and their
// armour are from the bed's anchor.
type sleepLie struct {
	flip         bool
	stand        int8
	side         float64 // a lengthways bed's open side: -1 west, 1 east
	headX, headY float64
	armourDX     float64 // armourByBed's own offset taken off
	armourDY     float64
}

// sign is -1 for a sleeper mirrored, else 1.
func (l sleepLie) sign() float64 {
	if l.flip {
		return -1
	}
	return 1
}

// The pillow's middle on a bed lying lengthways, from its anchor: up the
// screen with the head up, by the front rail with the head down.
func pillowLong(up bool) float64 {
	if up {
		return bedLongPill
	}
	return bedLongFore - 3.6
}

// lieFor is how one lies with the head towards head (sim.DirLeft as ever).
func lieFor(head sim.Dir) sleepLie {
	switch head {
	case sim.DirRight:
		return sleepLie{flip: true, headX: -pillowX, headY: pillowY, armourDX: -80}
	case sim.DirUp:
		return sleepLie{stand: 1, side: -1, headY: pillowLong(true), armourDX: -20, armourDY: 2}
	case sim.DirDown:
		return sleepLie{stand: -1, side: 1, headY: pillowLong(false), armourDX: -60, armourDY: 2}
	}
	return sleepLie{headX: pillowX, headY: pillowY}
}

// sleepLie is how p lies, by the bed or tent on their tile.
func (g *Game) sleepLie(p protocol.PlayerView) sleepLie {
	i, ok := g.entAt[[2]int{int(math.Floor(float64(p.X))), int(math.Floor(float64(p.Y)))}]
	if !ok || i >= len(g.curr.Entities) {
		return lieFor(sim.DirLeft)
	}
	e := g.curr.Entities[i]
	switch e.Kind {
	case sim.EntityBed:
		return lieFor(bedHead(sim.FacingOf(e.Kind, e.Var, e.Face)))
	case sim.EntityLeafTent:
		if e.Face == sim.DirLeft {
			return lieFor(sim.DirRight) // the door west: the head away from it
		}
	}
	return lieFor(sim.DirLeft)
}

// laidCanvasTurned is laidCanvas for k, mirrored or turned lengthways as k
// says, with its anchor where the bed it lies on is anchored.
func laidCanvasTurned(k laidKey) (*canvas, float64, float64) {
	base := k
	base.flip, base.stand = false, 0
	c, ax, ay := laidCanvas(base)
	if k.flip {
		return flipCanvas(c), float64(c.w)/HDDensity - ax, ay
	}
	if k.stand == 0 || !k.lay {
		return c, ax, ay
	}
	// The head, image pixels, before and after the quarter turn.
	hx, hy := (ax+pillowX)*HDDensity, (ay+pillowY)*HDDensity
	r := newCanvas(c.h, c.w)
	r.dens = c.dens
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			p := c.px[y*c.w+x]
			if p.A == 0 {
				continue
			}
			if k.stand > 0 { // clockwise: the head, on the left, goes up
				r.px[x*r.w+(c.h-1-y)] = p
			} else { // the other way: down
				r.px[(c.w-1-x)*r.w+y] = p
			}
		}
	}
	if k.stand > 0 {
		hx, hy = float64(c.h)-hy, hx
	} else {
		hx, hy = hy, float64(c.w)-hx
	}
	s := squashY(r, 0.75)
	hy *= 0.75
	return s, hx / HDDensity, hy/HDDensity - pillowLong(k.stand > 0)
}

// squashY is c drawn f as tall, each row of the result the rows of c it
// covers averaged (their colours weighted by their alpha), so nothing is
// lost between them.
func squashY(c *canvas, f float64) *canvas {
	h := int(math.Ceil(float64(c.h) * f))
	s := newCanvas(c.w, h)
	s.dens = c.dens
	for j := 0; j < h; j++ {
		y0, y1 := float64(j)/f, float64(j+1)/f
		for x := 0; x < c.w; x++ {
			var r, gg, b, a, wsum float64
			for y := int(y0); y < int(math.Ceil(y1)) && y < c.h; y++ {
				w := math.Min(y1, float64(y+1)) - math.Max(y0, float64(y))
				if w <= 0 {
					continue
				}
				p := c.px[y*c.w+x]
				pa := float64(p.A) / 255 * w
				r += float64(p.R) * pa
				gg += float64(p.G) * pa
				b += float64(p.B) * pa
				a += pa
				wsum += w
			}
			if a <= 0 || wsum <= 0 {
				continue
			}
			alpha := a / wsum
			if alpha < 0.35 {
				continue // a thin edge: let it go rather than smear it
			}
			s.px[j*c.w+x] = color.RGBA{uint8(r / a), uint8(gg / a), uint8(b / a), uint8(math.Min(255, alpha*255+0.5))}
		}
	}
	return s
}
