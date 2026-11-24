package sim

import (
	"math"
	"testing"
)

func TestEachBossHasItsOwnLegendaryItem(t *testing.T) {
	seen := map[ItemKind]bool{}
	for i := range NumBosses {
		it := LegendaryItem(BossKind(i))
		if !it.Valid() || !it.Legendary() || seen[it] || !it.Def().Tool || it.MaxCount() != 1 {
			t.Errorf("%s drops %d (%q)", BossKind(i).Name(), it, it.Def().Name)
		}
		seen[it] = true
	}
	if LegendaryItem(CreatureRex) != ItemNone {
		t.Errorf("the tyrant drops a legendary item")
	}
}

func TestTheScaleArmourIsTheBestAndKeepsVenomOut(t *testing.T) {
	var inv Inventory
	inv.Add(ItemDiamondArmor, 1)
	inv.Add(ItemDinoArmor, 1)
	best := ArmorShare(&inv)
	inv.Add(ItemTitanArmor, 1)
	if ArmorShare(&inv) <= best {
		t.Errorf("scale armour stops %.2f, the best other %.2f", ArmorShare(&inv), best)
	}
	w := flatWorld()
	p := w.players[w.AddPlayer("a")]
	p.Inv.Add(ItemTitanArmor, 1)
	p.poison = PoisonLong
	hp := p.Health
	w.Step()
	if p.Poisoned() || p.Health < hp {
		t.Errorf("in scale armour, poisoned %v, health %.1f of %.1f", p.Poisoned(), p.Health, hp)
	}
}

func TestTheLegendaryWeapons(t *testing.T) {
	strongest, hardest := 0, 0
	for k := ItemKind(1); k < NumItems; k++ {
		if k.Valid() && !k.Legendary() {
			strongest = max(strongest, WeaponDamage(k))
			hardest = max(hardest, WeaponDamage(k)*TicksPerSecond/SwingTicks(k))
		}
	}
	// The spear: the hardest thrust of any spear, and the longest reach of all.
	if WeaponDamage(ItemSpinoSpear) <= IvorySpearDamage || ReachBonusFor(ItemSpinoSpear) <= IvorySpearReach {
		t.Errorf("the tooth spear does %d and reaches %.1f further", WeaponDamage(ItemSpinoSpear), ReachBonusFor(ItemSpinoSpear))
	}
	// The sword: the quickest blows, harder than any knife's.
	if SwingTicks(ItemCarchaSword) >= SwingTicks(ItemDiamondSword) || WeaponDamage(ItemCarchaSword) <= WeaponDamage(ItemDiamondSword) {
		t.Errorf("the tooth sword does %d every %d ticks", WeaponDamage(ItemCarchaSword), SwingTicks(ItemCarchaSword))
	}
	// The axe: the hardest single blow of any weapon.
	if WeaponDamage(ItemGigaAxe) <= strongest || WeaponDamage(ItemGigaAxe) <= WeaponDamage(ItemCarchaSword) {
		t.Errorf("the fang axe does %d, the strongest other %d", WeaponDamage(ItemGigaAxe), strongest)
	}
	for _, k := range []ItemKind{ItemCarchaSword, ItemGigaAxe} {
		if dps := WeaponDamage(k) * TicksPerSecond / SwingTicks(k); dps <= hardest {
			t.Errorf("%s does %d a second, no better than the best other's %d", k.Def().Name, dps, hardest)
		}
	}
}

// strokesToFell counts the strokes it takes to fell a giant of the woods
// with axe in the bag.
func strokesToFell(t *testing.T, axe ItemKind) int {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv.Add(axe, 1)
	e := w.addEntity(EntityGiantTree, 17, 16)
	n := 0
	for e.Ready && n < 100 {
		p.X, p.Y, p.actionCD = 16.5, 16.5, 0
		w.interact(p, e)
		n++
	}
	return n
}

func TestTheFangAxeFellsTreesFast(t *testing.T) {
	if a, b := strokesToFell(t, ItemGigaAxe), strokesToFell(t, ItemDiamondAxe); a >= b {
		t.Errorf("the fang axe fells a giant tree in %d strokes, an obsidian axe in %d", a, b)
	}
}

func TestTheMammothKingsCoatKeepsEveryColdOut(t *testing.T) {
	w := flatWorld()
	for i := range w.tiles {
		w.tiles[i] = TileSnow
	}
	p := w.players[w.AddPlayer("a")]
	w.tick = 3*SeasonDays*DayTicks + DayTicks*95/100 // a winter night, on the snow
	w.Step()
	if !p.Cold {
		t.Fatalf("not cold on the snow on a winter night with nothing on")
	}
	p.Inv.Add(ItemKingCoat, 1)
	hp := p.Health
	stepN(w, TicksPerSecond)
	if p.Cold || p.Health < hp {
		t.Errorf("in the king's coat: cold %v, health %.1f of %.1f", p.Cold, p.Health, hp)
	}
}

func TestTheSharkToothNecklaceSwimsAndSailsFaster(t *testing.T) {
	swim := func(neck bool) (dist float64, breath float64) {
		w := newWorld(64, 32)
		w.style = StyleSeas
		w.spawnX, w.spawnY = 4.5, 16.5
		p := w.players[w.AddPlayer("a")]
		if neck {
			p.Inv.Add(ItemMegaNecklace, 1)
		}
		w.SetInput(p.ID, Input{MoveX: 1})
		stepN(w, 4*TicksPerSecond)
		return p.X - 4.5, p.Breath()
	}
	plain, plainBreath := swim(false)
	fast, fastBreath := swim(true)
	if fast <= plain*1.3 || fastBreath <= plainBreath {
		t.Errorf("swimming 4 seconds: %.1f tiles with the necklace (breath %.2f), %.1f without (%.2f)", fast, fastBreath, plain, plainBreath)
	}
	sail := func(neck bool) float64 {
		w := newWorld(96, 32)
		w.style = StyleSeas
		w.spawnX, w.spawnY = 4.5, 16.5
		p := w.players[w.AddPlayer("a")]
		if neck {
			p.Inv.Add(ItemMegaNecklace, 1)
		}
		w.nextBoat++
		b := &Boat{ID: w.nextBoat, Kind: BoatRaft, X: p.X, Y: p.Y, HP: BoatRaft.MaxHP(), Rider: p.ID}
		w.boats = append(w.boats, b)
		p.boat = b.ID
		w.SetInput(p.ID, Input{MoveX: 1})
		stepN(w, 8*TicksPerSecond)
		return b.X - 4.5
	}
	if a, b := sail(true), sail(false); a <= b*1.15 || math.IsNaN(a) {
		t.Errorf("paddling 8 seconds: %.1f tiles with the necklace, %.1f without", a, b)
	}
}
