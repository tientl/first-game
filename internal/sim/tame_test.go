package sim

import (
	"math"
	"testing"
)

// inHand puts n of k in p's selected slot.
func inHand(p *Player, k ItemKind, n int) {
	p.Inv[p.Selected] = Slot{Item: k, Count: n}
}

func TestTamingAThreeHornWithBerries(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureTrike, 18.2, 16.5)
	id := w.AddPlayer("a")
	p := w.players[id]
	inHand(p, ItemBerry, 10)
	hitUntil(w, id, 15*TicksPerSecond, func() bool { return c.Tamed() })
	if !c.Tamed() || c.Owner != id {
		t.Fatalf("tamed %v owner %d after %d meals: five berries should tame a three-horn", c.Tamed(), c.Owner, c.Tame)
	}
	if got := p.Inv.Count(ItemBerry); got != 10-CreatureTrike.TameMeals() {
		t.Errorf("%d berries left, want %d", got, 10-CreatureTrike.TameMeals())
	}
	if c.HP != trikeDef.hp*HealthScale || p.Health != MaxHealth {
		t.Errorf("hp %d, player health %.0f: taming hurts nobody", c.HP, p.Health)
	}
}

func TestHittingABeastUndoesItsTaming(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureTrike, 18.2, 16.5)
	id := w.AddPlayer("a")
	p := w.players[id]
	inHand(p, ItemBerry, 2)
	hitUntil(w, id, 5*TicksPerSecond, func() bool { return p.Inv.Count(ItemBerry) == 0 })
	if c.Tame != 2 {
		t.Fatalf("tame %d after two meals", c.Tame)
	}
	stepN(w, ActionCooldownTicks)
	c.X, c.Y = 18.2, 16.5
	hitUntil(w, id, 1, func() bool { return false }) // bare hands now: a hit
	if c.Tame != 0 || c.HP == trikeDef.hp*HealthScale {
		t.Fatalf("tame %d hp %d: a hit should hurt it and undo the taming", c.Tame, c.HP)
	}
}

func TestARaptorMustBeBeatenDownFirst(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureRaptor, 17.6, 16.5)
	id := w.AddPlayer("a")
	p := w.players[id]
	inHand(p, ItemMeat, 5)
	c.biteCD = 1000 // keep it from biting while it is fed
	w.SetInput(id, Input{Action: true})
	w.Step()
	w.SetInput(id, Input{})
	if c.Tame != 0 || p.Notice != NoticeTooWild {
		t.Fatalf("tame %d notice %d: a raptor on its feet takes no food", c.Tame, p.Notice)
	}
	c.HP = RaptorFleeBelow
	w.Step()
	if c.State != CreatureCower {
		t.Fatalf("state %d: a beaten raptor should cower", c.State)
	}
	c.X, c.Y = 17.6, 16.5
	hitUntil(w, id, 6*TicksPerSecond, func() bool { return c.Tamed() })
	if !c.Tamed() || p.Inv.Count(ItemMeat) != 5-CreatureRaptor.TameMeals() {
		t.Fatalf("tamed %v, meat %d: three meals should tame a cowering raptor", c.Tamed(), p.Inv.Count(ItemMeat))
	}
}

// tamedNear gives player id a pet of kind k at (x, y).
func tamedNear(w *World, id PlayerID, k CreatureKind, x, y float64) *Creature {
	c := w.addCreature(k, x, y)
	w.tame(c, w.players[id].Name)
	return c
}

func TestPetsFollowAndWait(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	pet := tamedNear(w, id, CreatureTrike, 17.5, 16.5)
	p.X, p.Y = 6.5, 8.5
	stepN(w, 6*TicksPerSecond)
	if d := math.Hypot(pet.X-p.X, pet.Y-p.Y); d > PetFollowFar+0.5 {
		t.Fatalf("pet %.1f tiles away after 6 s: it should follow its owner", d)
	}
	pet.X, pet.Y = p.X+1.5, p.Y
	hitUntil(w, id, 1, func() bool { return false })
	if !pet.Stay || p.Notice != NoticePetStay {
		t.Fatalf("stay %v notice %d: using a pet tells it to wait", pet.Stay, p.Notice)
	}
	x, y := pet.X, pet.Y
	p.X, p.Y = 26.5, 26.5
	stepN(w, 5*TicksPerSecond)
	if d := math.Hypot(pet.X-x, pet.Y-y); d > trikeDef.leash+1 {
		t.Fatalf("a waiting pet wandered %.1f tiles off", d)
	}
}

func TestPetsDefendTheirOwner(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	tamedNear(w, id, CreatureTrike, 15.5, 16.5)
	raptor := w.addCreature(CreatureRaptor, 20.5, 16.5)
	for range 20 * TicksPerSecond {
		if raptor.gone {
			break
		}
		w.Step()
	}
	if !raptor.gone && raptor.HP == raptorDef.hp*HealthScale {
		t.Fatal("the pet never fought the raptor going for its owner")
	}
	if raptor.gone && p.Inv.Count(ItemHide) == 0 {
		t.Error("the pet's kill should go to its owner")
	}
	if p.Dead {
		t.Error("the owner died with a three-horn guarding them")
	}
}

func TestRidingNeedsASaddleAndIsFast(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X, p.Y = 4.5, 16.5
	mount := tamedNear(w, id, CreatureTrike, 5.8, 16.5)
	w.Apply(id, Command{Kind: CmdRide})
	if p.Mount() != 0 || p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ItemSaddle {
		t.Fatalf("mount %d notice %d: riding needs a saddle", p.Mount(), p.Notice)
	}
	p.Inv.Add(ItemSaddle, 1)
	w.Apply(id, Command{Kind: CmdRide})
	if p.Mount() != mount.ID || mount.rider != id {
		t.Fatalf("mount %d: should be riding the three-horn", p.Mount())
	}
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 2*TicksPerSecond)
	w.SetInput(id, Input{})
	if got := p.X - 5.8; got < 2*PlayerSpeed*TicksPerSecond*1.2 || p.X != mount.X || p.Y != mount.Y {
		t.Fatalf("rode %.1f tiles in 2 s (player at %.1f, mount at %.1f): riding should beat walking", got, p.X, mount.X)
	}
	if len(w.creatureCandidates(p.X, p.Y)) != 0 {
		t.Error("a rider can target their own mount")
	}
	w.Apply(id, Command{Kind: CmdRide})
	if p.Mount() != 0 || mount.rider != 0 || (p.X == mount.X && p.Y == mount.Y) {
		t.Fatalf("mount %d: should be off, standing beside it", p.Mount())
	}
}

func TestOthersCannotHitOrRideYourPet(t *testing.T) {
	w := flatWorld()
	a := w.AddPlayer("a")
	b := w.AddPlayer("b")
	pet := tamedNear(w, a, CreatureTrike, 17.8, 16.5)
	w.players[a].X = 2.5
	w.players[b].Inv.Add(ItemSaddle, 1)
	hitUntil(w, b, 1, func() bool { return false })
	if pet.HP != trikeDef.hp*HealthScale || w.players[b].Notice != NoticeNotYours {
		t.Fatalf("hp %d notice %d: someone else's pet is not theirs to hit", pet.HP, w.players[b].Notice)
	}
	w.Apply(b, Command{Kind: CmdRide})
	if w.players[b].Mount() != 0 {
		t.Fatal("rode someone else's pet")
	}
}

func TestPetsAreSavedWithTheWorld(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 1, Width: 256, Height: 256, Style: StyleOpen})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("Tien")
	p := w.players[id]
	var pet *Creature
	for r := 1.5; pet == nil; r++ {
		for a := 0.0; a < 2*math.Pi && pet == nil; a += 0.5 {
			if x, y := p.X+math.Cos(a)*r, p.Y+math.Sin(a)*r; w.creatureFits(CreatureTrike, x, y) {
				pet = tamedNear(w, id, CreatureTrike, x, y)
			}
		}
	}
	pet.HP, pet.Stay = 9, true
	wild := len(w.Creatures())
	l, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	var got []Creature
	for _, c := range l.Creatures() {
		if c.Tamed() {
			got = append(got, c)
		}
	}
	if len(got) != 1 || got[0].Kind != CreatureTrike || got[0].HP != 9 || !got[0].Stay || got[0].ownerName != "Tien" {
		t.Fatalf("pets after loading: %+v", got)
	}
	if n := len(l.Creatures()); n != wild {
		t.Errorf("%d creatures after loading, %d before", n, wild)
	}
	id2 := l.AddPlayer("Tien")
	l.Step()
	for _, c := range l.Creatures() {
		if c.Tamed() && c.Owner != id2 {
			t.Errorf("owner %d, want the returning player %d", c.Owner, id2)
		}
	}
}
