package sim

import "testing"

func TestABodyLooksLikeItsPlayer(t *testing.T) {
	for i, l := range []Look{{}, {Female: true, Hair: HairShaggy, HairColor: NumHairColors - 1, Skin: NumSkinTones - 1}} {
		wear := []Wear{0, WearTunic | WearBoots}[i]
		fuel := int32(packGraveLook(l, 13, wear))
		got, col := GraveLook(fuel)
		if got != l.Sanitized() || col != 13 || GraveWear(fuel) != wear {
			t.Errorf("look %+v in %b came back as %+v in %b, colour %d", l, wear, got, GraveWear(fuel), col)
		}
	}
}

func TestDyingAgainTurnsTheOldBodyToBones(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 2, Width: 64, Height: 64, Difficulty: DiffSkilled})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayerWithLook("a", Look{Female: true, Hair: HairLong})
	p := w.players[id]
	die := func(item ItemKind) {
		p.Inv = Inventory{}
		if item != ItemNone {
			p.Inv.Add(item, 2)
		}
		hurt(p, MaxHealth)
		w.Apply(id, Command{Kind: CmdRespawn})
	}
	bodies := func() (fresh, bones []*Entity) {
		for _, e := range w.entities {
			if e.Kind != EntityGrave {
				continue
			}
			if e.Var == GraveBones {
				bones = append(bones, e)
			} else {
				fresh = append(fresh, e)
			}
		}
		return
	}
	die(ItemStone)
	fresh, bones := bodies()
	if len(fresh) != 1 || len(bones) != 0 {
		t.Fatalf("after one death: %d bodies, %d skeletons", len(fresh), len(bones))
	}
	if l, _ := GraveLook(int32(fresh[0].Fuel)); !l.Female || l.Hair != HairLong {
		t.Fatalf("the body looks like %+v, not its player", l)
	}
	first := fresh[0].ID

	// Back from a save, then dead again on the way to it, with nothing on.
	w, err = Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	id = w.AddPlayer("a")
	p = w.players[id]
	die(ItemNone)
	fresh, bones = bodies()
	if len(fresh) != 0 || len(bones) != 1 || bones[0].ID != first || w.BodyItems(first) != 1 {
		t.Fatalf("after dying again empty-handed: %d bodies, %d skeletons", len(fresh), len(bones))
	}
	die(ItemWood) // and again, carrying something
	fresh, bones = bodies()
	if len(fresh) != 1 || len(bones) != 1 {
		t.Fatalf("after a third death: %d bodies, %d skeletons", len(fresh), len(bones))
	}
}
