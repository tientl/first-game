package sim

import (
	"math"
	"testing"
)

func TestOresLieInVeinsAndFlintIsCommonest(t *testing.T) {
	w := genSeas(t, 3, SizeMedium)
	count := map[EntityKind]int{}
	alone, near := 0, false
	for _, e := range w.entities {
		if _, ok := oreOf(e.Kind); !ok {
			continue
		}
		count[e.Kind]++
		mate := false
		for dy := -1; dy <= 1 && !mate; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if o := w.entityAt(e.TX+dx, e.TY+dy); o != nil && o != e && o.Kind == e.Kind {
					mate = true
					break
				}
			}
		}
		if !mate {
			alone++
		}
		if e.Kind == EntityCopperOre && math.Hypot(float64(e.TX)-w.spawnX, float64(e.TY)-w.spawnY) <= homeFlintFar+4 {
			near = true
		}
	}
	total := count[EntityCopperOre] + count[EntityIronOre] + count[EntityPlatinumOre] + count[EntityDiamondOre]
	t.Logf("flint %d, jade %d, agate %d, obsidian %d; %d of %d alone", count[EntityCopperOre], count[EntityIronOre], count[EntityPlatinumOre], count[EntityDiamondOre], alone, total)
	if count[EntityCopperOre] <= count[EntityPlatinumOre] || count[EntityCopperOre] <= count[EntityDiamondOre] {
		t.Error("flint, the first tier, should be commoner than agate and obsidian")
	}
	if alone*5 > total {
		t.Errorf("%d of %d ore blocks lie alone: they should be in veins", alone, total)
	}
	if !near {
		t.Error("no flint near the spawn")
	}
}
