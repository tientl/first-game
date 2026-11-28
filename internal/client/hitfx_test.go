package client

import (
	"testing"

	"survivalgame/internal/sim"
)

// Every creature has a material and a matter; flesh, shell and bone never
// throw fire sparks (only stone and metal do), and no creature is made of
// stone or metal.
func TestHitMaterialOfEveryCreature(t *testing.T) {
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		m := hitMaterial(k)
		if m >= numHitMats {
			t.Errorf("creature %d: material %d out of range", k, m)
		}
		if m == matStone || m == matMetal || m == matWood {
			t.Errorf("creature %d: a beast is not %d", k, m)
		}
		if mt := hitMatter(k); mt < 0 || mt >= len(matters) {
			t.Errorf("creature %d: matter %d out of range", k, mt)
		}
	}
	if hitMaterial(sim.CreatureSpider) != matChitin || hitMaterial(sim.CreatureRaptor) != matFlesh || hitMaterial(sim.CreatureNone) != matFlesh {
		t.Error("spiders are chitin, raptors flesh, the unknown flesh")
	}
	if m, ok := entityHitMat(sim.EntityTree); !ok || m != matWood {
		t.Error("a tree is wood")
	}
	if m, ok := entityHitMat(sim.EntityBoulder); !ok || m != matStone {
		t.Error("a boulder is stone")
	}
}

// A blow on flesh, shell, bone or wood draws nothing in the spark palette's
// hot tones: orange and yellow pixels never appear.
func TestFleshHitsThrowNoFire(t *testing.T) {
	for _, mat := range []hitMat{matFlesh, matChitin, matBone, matWood} {
		for lvl := 0; lvl < 3; lvl++ {
			for ti := 0; ti < 60; ti += 2 {
				var b fxBatch
				r := fxRng{12345}
				hitFX(&b, &r, 32, 20, 36, 0.3, 1, lvl, mat, mtFurBrown, dustGrass, float64(ti)/60)
				for _, v := range b.vs {
					rr, gg, bb := v.ColorR, v.ColorG, v.ColorB
					a := v.ColorA
					if a <= 0 {
						continue
					}
					// fire: strongly red-orange or yellow: r high, b low, g mid-high
					if rr > 0.85 && bb < 0.3 && gg > 0.3 && gg < 0.85 && a > 0 {
						t.Fatalf("mat %d lvl %d t %d: a fiery vertex %v", mat, lvl, ti, v)
					}
				}
			}
		}
	}
}
