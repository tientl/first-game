package client

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Going into a cave changes the scene: the rock roof over the cave fades
// away to show the tunnels, the world outside sinks into darkness, crystals
// glow in the walls, and the sounds and music change. From outside, a cave
// is only its dark mouth in a cliff, and nothing inside it can be seen.

const (
	caveFadeStep   = 1.0 / 36 // per frame: about 0.6 s to go in or out
	caveInsideDark = 0.9      // how dark a cave is away from any light
	crystalRadius  = 1.4      // tiles of light around a cluster of crystals
)

// updateCave eases the cave view towards whether this player stands in a
// cave. Call it once per frame before drawing.
func (g *Game) updateCave() {
	if g.ground == nil {
		return
	}
	in := 0.0
	if !g.demo && g.hasCurr {
		for _, p := range g.curr.Players {
			if p.ID != g.myID || p.Dead {
				continue
			}
			x, y, _ := g.lerpPlayer(p, g.alpha())
			if g.tileAt(int(math.Floor(x)), int(math.Floor(y))) == sim.TileCave {
				in = 1
			}
		}
	}
	gc := g.ground
	gc.caveIn = approach(gc.caveIn, in, caveFadeStep)
	gc.roofAlpha = float32(1 - smooth(gc.caveIn))
}

// caveIn is how far into a cave this player is: 0 outside, 1 inside.
func (g *Game) caveIn() float64 {
	if g.ground == nil {
		return 0
	}
	return g.ground.caveIn
}

// hiddenInCave reports whether something at (x, y) is out of sight under a
// cave roof: anything in a cave while the player is outside.
func (g *Game) hiddenInCave(x, y float64) bool {
	return g.ground != nil && g.ground.roofAlpha > 0.5 &&
		g.tileAt(int(math.Floor(x)), int(math.Floor(y))) == sim.TileCave
}

// crystalLight is one glowing crystal cluster in view.
type crystalLight struct{ sx, sy float64 }

// crystals lists the crystal clusters in view, at the foot of tunnel walls,
// the same spots crystalPixel draws.
func (g *Game) crystals(cam camera, w, h int) []crystalLight {
	var out []crystalLight
	tx0, ty0 := int(math.Floor(cam.ox/TileW))-2, int(math.Floor(cam.oy/TileH))-2
	for ty := ty0; ty <= ty0+h/TileH+4; ty++ {
		for tx := tx0; tx <= tx0+w/TileW+4; tx++ {
			if g.tileAt(tx, ty) != sim.TileCave || g.tileAt(tx, ty-1) != sim.TileMountain {
				continue
			}
			if cx, ok := crystalX(tx, ty*TileH); ok {
				out = append(out, crystalLight{cx - cam.ox, float64(ty*TileH) - 5 - cam.oy})
			}
		}
	}
	return out
}

// caveDarkness is how dark the whole view is for a player in a cave.
func (g *Game) caveDarkness() float64 {
	d := caveInsideDark
	if g.underground() {
		d = caveDeepDark // the deep caves: no light gets in at all
	}
	return d * smooth(g.caveIn())
}

// caveDeepDark is how dark the caves under the world are, away from light.
const caveDeepDark = 0.97

// ownLightRadius is the faint light a player sees by underground with no
// torch: enough to make out the walls close by.
const ownLightRadius = 1.9

// crystalLights punches the crystals' light out of the darkness (punch) or
// adds their cold glow to the scene.
func (g *Game) crystalLights(dst *ebiten.Image, cam camera, punch bool) {
	in := g.caveIn()
	if in <= 0 {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	for i, c := range g.crystals(cam, w, h) {
		pulse := 1 + 0.08*math.Sin(float64(g.frame)*0.05+float64(i)*1.7)
		op := g.lightOpAt(c.sx, c.sy, crystalRadius*pulse)
		if punch {
			op.Blend = ebiten.BlendDestinationOut
			op.ColorScale.ScaleAlpha(float32(0.75 * in))
		} else {
			k := float32(0.22 * in * pulse)
			op.ColorScale.Scale(k*0.35, k*0.8, k, k)
			op.Blend = ebiten.BlendLighter
		}
		dst.DrawImage(g.art.light, op)
	}
}
