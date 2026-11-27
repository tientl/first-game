package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

const maxDarkness = 0.88 // deep night still shows faint outlines

var colCaveDark = color.RGBA{4, 6, 12, 255} // the dark underground

// drawLighting darkens the scene by time of day and lets campfires cut holes
// in the darkness. The light radius is sim.FireRadius, the same radius the
// server uses for warmth, so "inside the light" means "safe from the cold".
func (g *Game) drawLighting(screen drawTarget, cam camera) {
	g.caveDark.on = false
	if g.caveDarkOn() {
		g.drawCaveDark(screen, cam) // the natural caves have a light of their own: see cavedark.go
		return
	}
	amb := sim.Ambient(g.curr.Tick)
	night := (1 - amb) * maxDarkness
	cave := g.caveDarkness() // inside a cave it is dark at any hour
	if g.caveIn() < 0.5 {
		night = math.Max(night, g.weatherDim()) // an overcast sky
	}
	dark := math.Max(night, cave)
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Lava lights up its surroundings.
	lx0, ly0, lava := g.tileMask(&g.lavaMask, cam, sw, sh, isLava)

	if dark > 0.01 {
		// The dark is drawn a pixel bigger than the view all round, so it
		// covers the screen's edges wherever the camera's grid falls: see
		// overlay.go.
		img := sizedImage(&g.darkness, sw+2*overlayMargin, sh+2*overlayMargin)
		img.Clear()
		nc := nightColor(amb)
		if cave > night { // underground the dark is blue-black, whatever the hour
			nc = mix(nc, colCaveDark, cave/caveInsideDark)
		}
		f := dark
		img.Fill(color.RGBA{uint8(float64(nc.R) * f), uint8(float64(nc.G) * f), uint8(float64(nc.B) * f), uint8(255 * f)})
		darkness := overlayTarget{img}
		g.crystalLights(darkness, cam, true)
		if lava {
			g.lavaLight(darkness, cam, lx0, ly0, true, 0)
		}
		for _, e := range g.curr.Entities {
			r, ok := sim.FireReach(e.Kind, int(e.Fuel)) // campfires and lit furnaces
			if !ok {
				continue
			}
			op := g.lightOp(cam, e.TX, e.TY, r*g.flicker(uint32(e.ID)))
			op.Blend = ebiten.BlendDestinationOut
			darkness.DrawImage(g.art.light, op)
		}
		for _, t := range g.torches(cam) {
			op := g.lightOpAt(t.sx, t.sy, sim.TorchRadius*g.flicker(t.id))
			op.Blend = ebiten.BlendDestinationOut
			darkness.DrawImage(g.art.light, op)
		}
		g.glowLights(darkness, cam, true, dark) // see tribalrender.go
		g.ownLight(darkness, cam)
		drawOverlay(screen, img)
	}
	if lava {
		g.lavaLight(screen, cam, lx0, ly0, false, float32((0.12+0.3*dark)*g.caveLavaGlow())) // see cavelights.go
	}
	g.crystalLights(screen, cam, false)

	// Warm additive glow, stronger at night.
	for _, e := range g.curr.Entities {
		r, ok := sim.FireReach(e.Kind, int(e.Fuel))
		if !ok {
			continue
		}
		k := float32(0.34 * (0.25 + dark) * g.flicker(uint32(e.ID)))
		op := g.lightOp(cam, e.TX, e.TY, r*0.75)
		if e.Kind == sim.EntityCaveExit { // daylight down the shaft: pale, steady
			k = float32(0.3 * dark)
			op.ColorScale.Scale(k, k*0.95, k*0.8, k)
		} else {
			op.ColorScale.Scale(k, k*0.36, k*0.06, k)
		}
		op.Blend = ebiten.BlendLighter
		screen.DrawImage(g.art.light, op)
	}
	g.glowLights(screen, cam, false, dark)
	for _, t := range g.torches(cam) {
		k := float32(0.28 * (0.2 + dark) * g.flicker(t.id))
		op := g.lightOpAt(t.sx, t.sy, sim.TorchRadius*0.7)
		op.ColorScale.Scale(k, k*0.4, k*0.08, k)
		op.Blend = ebiten.BlendLighter
		screen.DrawImage(g.art.light, op)
	}
}

type torchLight struct {
	sx, sy float64 // where the flame is, in art pixels on screen
	id     uint32
}

// torches lists the lit torches in view: every player holding one, awake.
// The server warms everyone within sim.TorchRadius of a lit torch, the same
// radius as its light.
func (g *Game) torches(cam camera) []torchLight {
	var out []torchLight
	alpha := g.alpha()
	for _, p := range g.curr.Players {
		if p.Held != sim.ItemTorch || p.Dead || p.Asleep {
			continue
		}
		x, y, _ := g.lerpPlayer(p, alpha)
		sx, sy := cam.toScreenFine(x, y)
		sy -= 12
		if m := g.heldMarkOf(p.ID); m != nil && m.flame { // the flame as drawn; see heldhd.go
			sx, sy = m.fx, m.fy-2
		}
		out = append(out, torchLight{sx, sy, uint32(p.ID) * 7919})
	}
	return out
}

// lightOpAt is lightOp for a point in screen art pixels.
func (g *Game) lightOpAt(sx, sy, radius float64) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-lightSize/2, -lightSize/2)
	op.GeoM.Scale(radius*TileW/(lightSize/2), radius*TileH/(lightSize/2))
	op.GeoM.Translate(sx, sy)
	op.Filter = ebiten.FilterLinear
	return op
}

func (g *Game) flicker(id uint32) float64 {
	if fireHD { // the light leaps with the flames; see fireart.go
		return 1 + (g.fireBright(id)-1)*0.14
	}
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
	return color.RGBA{mix(6, 84), mix(10, 40), mix(34, 44), 255}
}

// ownLight lets this player see a little way round themselves in the deep
// caves: a faint ring of sight, so the passages can be felt out without a
// torch. A torch lights far more.
func (g *Game) ownLight(dark drawTarget, cam camera) {
	if !g.underground() {
		return
	}
	for _, p := range g.curr.Players {
		if p.ID != g.myID || p.Dead {
			continue
		}
		x, y, _ := g.lerpPlayer(p, g.alpha())
		sx, sy := cam.toScreenFine(x, y)
		op := g.lightOpAt(sx, sy-6, ownLightRadius)
		op.ColorScale.ScaleAlpha(0.8)
		op.Blend = ebiten.BlendDestinationOut
		dark.DrawImage(g.art.light, op)
	}
}
