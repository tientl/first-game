package sim

import "testing"

// hungry makes c want a meal now; thirsty, a drink.
func hungry(c *Creature) {
	c.fed = true
	c.hunger = c.due(c.Kind.diet().hunger, 17)
	c.thirst = 0
}

func thirsty(c *Creature) {
	c.fed = true
	c.thirst = c.due(c.Kind.diet().thirst, 31)
	c.hunger = 0
}

func TestHungryGrazersCropTheGrass(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureTrike, 16.5, 16.5)
	hungry(c)
	w.Step()
	if c.State != CreatureEat {
		t.Fatalf("state %d: a hungry three-horn on grass should graze", c.State)
	}
	stepN(w, trikeDiets().eatTicks+2)
	if c.State == CreatureEat || c.hunger > trikeDiets().eatTicks+5 { // the meal reset its hunger
		t.Fatalf("state %d hunger %d: the meal should end, and fill it", c.State, c.hunger)
	}
}

func trikeDiets() *dietDef { return CreatureTrike.diet() }

func TestThirstyBeastsWalkToTheWaterAndDrink(t *testing.T) {
	w := shoreWorld() // the sea from column 18
	c := w.addCreature(CreatureThesc, 8.5, 16.5)
	thirsty(c)
	for range 20 * TicksPerSecond {
		if w.Step(); c.State == CreatureDrink {
			break
		}
	}
	if c.State != CreatureDrink || c.X < 14 {
		t.Fatalf("state %d at x %.1f: it should walk to the shore and drink", c.State, c.X)
	}
}

func TestAHunterRunsDownPreyAndEatsIt(t *testing.T) {
	w := newWorld(64, 32)
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	hunter := w.addCreature(CreatureRaptor, 20.5, 16.5)
	prey := w.addCreature(CreatureThesc, 26.5, 16.5)
	hungry(hunter)
	prey.fed = true
	for range 40 * TicksPerSecond {
		if w.Step(); prey.gone {
			break
		}
	}
	if !prey.gone {
		t.Fatalf("the prey (hp %d) got away from a hungry raptor", prey.HP)
	}
	w.Step()
	if hunter.State != CreatureEat {
		t.Fatalf("state %d: after the kill it should eat", hunter.State)
	}
}

func TestHornedGrazersTurnAndFight(t *testing.T) {
	w := newWorld(64, 32)
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	rex := w.addCreature(CreatureRex, 20.5, 16.5)
	trike := w.addCreature(CreatureTrike, 26.5, 16.5)
	hungry(rex)
	trike.fed = true
	for range 20 * TicksPerSecond {
		if w.Step(); trike.gone || rex.HP < rexDef.hp {
			break
		}
	}
	if rex.HP == rexDef.hp {
		t.Fatalf("trike state %d gone %v: a hunted three-horn should gore the tyrant", trike.State, trike.gone)
	}
}

func TestPlayersComeBeforePrey(t *testing.T) {
	w := flatWorld()
	hunter := w.addCreature(CreatureRaptor, 20.5, 16.5)
	w.addCreature(CreatureThesc, 26.5, 16.5)
	hungry(hunter)
	stepN(w, 2*TicksPerSecond) // it picks its prey
	id := w.AddPlayer("a")
	w.players[id].X, w.players[id].Y = hunter.X-3, hunter.Y
	stepN(w, 4*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health == MaxHealth {
		t.Fatal("a hunting raptor ignored a player close by")
	}
}

func TestCrocodilesTakeAnimalsComeToDrink(t *testing.T) {
	w := marshWorld(true) // ponds from column 18
	croc := w.addCreature(CreatureCroc, 18.6, 16.5)
	croc.State = CreatureLurk
	c := w.addCreature(CreatureThesc, 17.2, 16.5)
	c.fed = true
	c.State, c.timer = CreatureDrink, DrinkTicks
	w.Step()
	if c.HP == thescDef.hp || croc.State != CreatureStrike {
		t.Fatalf("hp %d croc state %d: a lurking crocodile should take a drinker", c.HP, croc.State)
	}
	if c.hunter != croc.ID {
		t.Fatal("the drinker should bolt from the crocodile")
	}
}

func TestAPlayerDisturbsAMeal(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureTrike, 20.5, 16.5)
	hungry(c)
	w.Step()
	if c.State != CreatureEat {
		t.Fatalf("state %d: should be grazing", c.State)
	}
	id := w.AddPlayer("a")
	w.players[id].X = 19
	w.Step()
	if c.State == CreatureEat {
		t.Fatal("still grazing with a player right beside it")
	}
}
