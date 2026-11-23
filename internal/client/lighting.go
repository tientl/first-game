package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

const maxDarkness = 0.9 // deep night still shows faint outlines

// drawLighting darkens the scene by time of day and lets campfires cut holes
// in the darkness. The light radius is sim.FireRadius, the same radius the
// server uses for warmth, so "inside the light" means "safe from the cold".
func (g *Game) drawLighting(screen *ebiten.Image, cam camera) {
	amb := sim.Ambient(g.curr.Tick)
	dark := (1 - amb) * maxDarkness
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	if dark > 0.01 {
		if g.darkness == nil || g.darkness.Bounds().Dx() != sw || g.darkness.Bounds().Dy() != sh {
			if g.darkness != nil {
				g.darkness.Deallocate()
			}
			g.darkness = ebiten.NewImage(sw, sh)
		}
		g.darkness.Fill(nightColor(amb))
		for _, e := range g.curr.Entities {
			if e.Kind != sim.EntityCampfire {
				continue
			}
			op := g.lightOp(cam, e.TX, e.TY, sim.FireRadius(int(e.Fuel))*g.flicker(uint32(e.ID)))
			op.Blend = ebiten.BlendDestinationOut
			g.darkness.DrawImage(g.art.light, op)
		}
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(float32(dark))
		screen.DrawImage(g.darkness, op)
	}

	// Warm additive glow, stronger at night.
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityCampfire {
			continue
		}
		k := float32(0.28 * (0.3 + dark) * g.flicker(uint32(e.ID)))
		op := g.lightOp(cam, e.TX, e.TY, sim.FireRadius(int(e.Fuel))*0.7)
		op.ColorScale.Scale(k, k*0.55, k*0.2, k)
		op.Blend = ebiten.BlendLighter
		screen.DrawImage(g.art.light, op)
	}
}

func (g *Game) flicker(id uint32) float64 {
	return 1 + 0.03*math.Sin(float64(g.frame)*0.31+float64(id))
}

// lightOp places the light image over tile (tx, ty) with the given radius in
// tiles. The ground is foreshortened, so the lit area is an ellipse.
func (g *Game) lightOp(cam camera, tx, ty int16, radius float64) *ebiten.DrawImageOptions {
	sx, sy := cam.toScreen(float64(tx)+0.5, float64(ty)+0.5)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-lightSize/2, -lightSize/2)
	op.GeoM.Scale(radius*TileW/(lightSize/2), radius*TileH/(lightSize/2))
	op.GeoM.Translate(sx, sy)
	op.Filter = ebiten.FilterLinear
	return op
}

// nightColor is the tint of the darkness: warm at dusk, deep blue at night.
func nightColor(amb float64) color.RGBA {
	t := max(0, min(1, amb/0.6))
	mix := func(night, dusk float64) uint8 { return uint8(night + (dusk-night)*t) }
	return color.RGBA{mix(8, 70), mix(10, 36), mix(28, 30), 255}
}
