package sim

import (
	"math"
	"testing"
)

func TestTreesNeedAnAxe(t *testing.T) {
	w := flatWorld()
	tree := w.addEntity(EntityTree, 17, 16)
	id := w.AddPlayer("a")
	w.SetInput(id, Input{Action: true})
	stepN(w, 1+TreeHits*ActionCooldownTicks)

	p := mustPlayer(t, w, id)
	if tree.HP != maxHP(EntityTree, false) || p.Inv.Count(ItemWood) != 0 {
		t.Fatalf("tree hp %d, wood %d: chopped without an axe", tree.HP, p.Inv.Count(ItemWood))
	}
	if p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ItemAxe {
		t.Fatalf("notice = %v/%d, want need-tool for the axe", p.Notice, p.NoticeArg)
	}
	if p.Acting(w.Tick()) {
		t.Error("a refused hit shows the working pose")
	}
	// Still the target, so the client can say why nothing happens.
	if got, _ := w.findTarget(w.players[id]); got != tree {
		t.Errorf("target = %v, want the tree", got)
	}
}

func TestFromTwigsToAnAxe(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	twig := w.addEntity(EntityTwig, 17, 16)
	w.Apply(id, Command{Kind: CmdInteract}) // E picks it up: see pickup.go
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemStick) != TwigYield || twig.Ready {
		t.Fatalf("sticks %d, twig ready %v: want it picked in one go", p.Inv.Count(ItemStick), twig.Ready)
	}

	w.players[id].Inv = InventoryWith(Ingredient{ItemStick, 2}, Ingredient{ItemPebble, 2})
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeAxe})
	p := mustPlayer(t, w, id)
	if want := InventoryWith(Ingredient{ItemPebble, 1}, Ingredient{ItemAxe, 1}); p.Inv.Counts() != want.Counts() || p.Notice != NoticeCrafted {
		t.Fatalf("after crafting: inv %v notice %v, want %v and crafted", p.Inv, p.Notice, want)
	}
	if _, ok := w.EntityAt(16, 17); ok {
		t.Error("crafting an item placed something in the world")
	}

	// One axe is enough: a second is refused and costs nothing.
	w.players[id].Inv.Add(ItemStick, 2)
	w.players[id].Inv.Add(ItemPebble, 1)
	before := w.players[id].Inv
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeAxe})
	if p := mustPlayer(t, w, id); p.Inv != before || p.Notice != NoticeHaveOne {
		t.Fatalf("second axe: inv %v notice %v, want unchanged and have-one", p.Inv, p.Notice)
	}

	// With the axe the tree comes down.
	tree := w.addEntity(EntityTree, 15, 16)
	w.players[id].Dir = DirLeft
	w.SetInput(id, Input{Action: true})
	stepN(w, (TreeHits+1)*ActionCooldownTicks) // the twig pick left a cooldown
	if tree.Ready {
		t.Error("tree still standing with an axe in hand")
	}

	w.RemovePlayer(id) // keep the regrow wait free of night-time deaths
	stepN(w, TwigRegrowTicks)
	if !twig.Ready {
		t.Error("twig did not come back")
	}
}

func TestEveryIslandHasTheFirstToolsNearTheSpawn(t *testing.T) {
	for seed := uint64(1); seed <= 40; seed++ {
		w, err := Generate(GenConfig{Seed: seed, Width: 64, Height: 64})
		if err != nil {
			t.Fatal(err)
		}
		cx, cy := int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY))
		var count [NumEntityKinds]int
		w.EachEntityIn(cx-starterMaxR, cy-starterMaxR, cx+starterMaxR, cy+starterMaxR, func(e Entity) { count[e.Kind]++ })
		// Where each ingredient comes from, and how much one gives. Small
		// boulders give the least.
		sources := map[ItemKind]struct {
			kind  EntityKind
			yield int
		}{
			ItemStick:  {EntityTwig, TwigYield},
			ItemGrass:  {EntityGrassTuft, TuftYield},
			ItemStone:  {EntityBoulder, BoulderYield},
			ItemPebble: {EntityPebble, PebbleYield},
		}
		// Each first tool alone, and all three tools together.
		var all [NumItems]int
		for _, id := range []RecipeID{RecipeAxe, RecipePickaxe, RecipeHoe, RecipeCampfire} {
			r, _ := RecipeByID(id)
			for _, in := range r.Inputs {
				if id != RecipeCampfire {
					all[in.Item] += in.Count
				}
				src, ok := sources[in.Item]
				if !ok {
					continue // wood: trees are everywhere, and need the axe first
				}
				if got := count[src.kind] * src.yield; got < in.Count {
					t.Errorf("seed %d: %d %v near the spawn, %s needs %d", seed, got, in.Item.Def().Name, r.Name, in.Count)
				}
			}
		}
		if got := count[EntityPebble] * PebbleYield; got < all[ItemPebble] {
			t.Errorf("seed %d: %d pebbles near the spawn, the axe, pickaxe and hoe need %d", seed, got, all[ItemPebble])
		}
	}
}

func TestOldSavesKeepChopping(t *testing.T) {
	w, _ := Generate(GenConfig{Seed: 5, Width: 48, Height: 48})
	d := w.Save()
	d.Version = 1
	kept := d.Entities[:0]
	for _, e := range d.Entities {
		if e.Kind != EntityTwig { // a version 1 world had none
			kept = append(kept, e)
		}
	}
	d.Entities = kept
	d.Players = []PlayerSave{{Name: "Cũ", X: w.spawnX, Y: w.spawnY, Health: 80, Hunger: 50, Inv: []int{ItemWood: 4}}}

	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	twigs := 0
	w2.EachEntityIn(0, 0, 48, 48, func(e Entity) {
		if e.Kind == EntityTwig {
			twigs++
		}
	})
	if twigs == 0 {
		t.Error("upgraded world has no twigs")
	}
	p := mustPlayer(t, w2, w2.AddPlayer("Cũ"))
	if p.Inv.Count(ItemAxe) != 1 || p.Inv.Count(ItemWood) != 4 {
		t.Errorf("upgraded player inventory = %v, want the wood and an axe", p.Inv)
	}
	if fresh := mustPlayer(t, w2, w2.AddPlayer("Mới")); fresh.Inv.Count(ItemAxe) != 0 {
		t.Error("a new player got a free axe")
	}
}
