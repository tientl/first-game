package sim

import (
	"math"
	"testing"
)

func TestEveryIslandHasACaveAndATribe(t *testing.T) {
	for _, c := range []struct {
		style WorldStyle
		size  int
	}{{StyleSeas, SizeMedium}, {StyleSeas, SizeLarge}, {StyleIsles, SizeMedium}, {StyleIsles, SizeLarge}} {
		size := c.size
		for seed := uint64(1); seed <= 3; seed++ {
			w := genStyle(t, c.style, seed, size)
			W, H := w.width, w.surfaceH
			sx, sy := w.Spawn()
			islands := 0
			for _, isl := range w.landMasses() {
				if isl[0]/W >= H || len(isl) < mouthIslandMin {
					continue
				}
				islands++
				in := make(map[int]bool, len(isl))
				for _, i := range isl {
					in[i] = true
				}
				home := in[int(sy)*W+int(sx)]
				mouths, villages := 0, 0
				for _, e := range w.entities {
					if e.Kind != EntityCaveMouth || !in[e.TY*W+e.TX] {
						continue
					}
					mouths++
					if d := math.Hypot(float64(e.TX)+0.5-sx, float64(e.TY)+0.5-sy); home && d < homeMouthAway {
						t.Errorf("size %d seed %d: a cave mouth %.0f tiles from the spawn", size, seed, d)
					}
				}
				for _, v := range w.villages {
					if !in[int(v.Y)*W+int(v.X)] {
						continue
					}
					villages++
					if d := math.Hypot(v.X-sx, v.Y-sy); home && d < homeVillageAway {
						t.Errorf("size %d seed %d: a village %.0f tiles from the spawn", size, seed, d)
					}
				}
				if mouths == 0 || villages == 0 && len(isl) >= villageIslandMin {
					t.Errorf("size %d seed %d: an island of %d tiles (home %v) has %d cave mouths and %d villages",
						size, seed, len(isl), home, mouths, villages)
				}
			}
			if islands < 10 {
				t.Errorf("size %d seed %d: only %d islands", size, seed, islands)
			}
			if _, _, ok := w.Pyramid(); !ok {
				t.Errorf("size %d seed %d: the villages left no room for the pyramid", size, seed)
			}
		}
	}
}

func TestEveryVillageHasHutsAndWarriors(t *testing.T) {
	for _, style := range archipelagos {
		testVillages(t, genStyle(t, style, 1, SizeMedium))
	}
}

func testVillages(t *testing.T, w *World) {
	for i, v := range w.villages {
		huts, folk := 0, 0
		for _, e := range w.entities {
			if e.Kind == EntityHut && math.Hypot(float64(e.TX)+0.5-v.X, float64(e.TY)+0.5-v.Y) <= float64(v.radius()) {
				huts++
			}
		}
		for _, c := range w.fauna.creatures {
			if c.Kind == CreatureTribesman && c.village == i+1 {
				folk++
			}
		}
		if huts == 0 || folk == 0 {
			t.Errorf("village %d at (%.0f,%.0f), small %v: %d huts, %d warriors", i, v.X, v.Y, v.Small, huts, folk)
		}
	}
}
