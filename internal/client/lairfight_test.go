package client

import (
	"math"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// In a fight the camera keeps the view in the arena: wherever the player
// goes, every corner of the screen stays on the courtyard or the atoll (or
// just past its edge), for the views of a 1080p screen at zoom 0, drawn
// back in a fight, and 16:10.
func TestTheCameraKeepsTheViewInTheArena(t *testing.T) {
	lairs := []protocol.LairView{
		{Kind: sim.CreatureTitanoboa, X: 100, Y: 100, Layout: sim.LayoutTemple, RX: sim.TempleRX, RY: sim.TempleRY},
		{Kind: sim.CreatureMegalodon, X: 100, Y: 100, Layout: sim.LayoutAtoll, RX: sim.AtollLagoon + sim.AtollReef + 2.5, RY: sim.AtollLagoon + sim.AtollReef + 2.5},
	}
	views := [][2]float64{{640.0 / TileW / 2, 360.0 / TileH / 2}, {640 / 0.9 / TileW / 2, 360 / 0.9 / TileH / 2}, {640.0 / TileW / 2, 400.0 / TileH / 2}}
	for _, l := range lairs {
		g := &Game{}
		g.lf.clampK, g.lf.clampLair = 1, l
		for _, v := range views {
			hw, hh := v[0], v[1]
			for _, p := range [][2]float64{{0, 0}, {-30, 0}, {30, 5}, {0, -25}, {4, 22}, {-18, -14}, {22, 17}} {
				cx, cy := g.arenaCamera(100+p[0], 100+p[1], hw, hh)
				for _, c := range [4][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
					dx, dy := cx+c[0]*hw-100, cy+c[1]*hh-100
					my := float64(l.RY) + 0.8
					if dy > 0 {
						my = float64(l.RY) + 2.4
					}
					guarded := math.Abs(100+p[0]-cx) >= hw-3.01 || math.Abs(100+p[1]-cy) >= hh-4.01 // held to keep the player in view
					if q := sim.ArenaQ(l.Layout, float64(l.RX)+0.8, my, dx, dy); q > 1.001 && !guarded {
						t.Errorf("%s, view %.1fx%.1f tiles, player at %+v: a corner of the screen leaves the arena (q %.2f)", l.Kind.Name(), 2*hw, 2*hh, p, q)
					}
				}
			}
		}
	}
}
