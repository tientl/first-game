package sim

import (
	"math"
	"testing"
)

// winterNight is a tick deep in a winter night.
var winterNightTick = uint64(3*SeasonDays*DayTicks) + deepNight

func TestOnlyWinterNightsFreeze(t *testing.T) {
	if SeasonOf(winterNightTick) != Winter || Ambient(winterNightTick) >= ColdAmbient {
		t.Fatalf("test tick: season %v, light %.2f", SeasonOf(winterNightTick), Ambient(winterNightTick))
	}
	for _, c := range []struct {
		tick uint64
		cold bool
	}{{deepNight, false}, {winterNightTick, true}} {
		w := flatWorld()
		w.tick = c.tick
		id := w.AddPlayer("a")
		w.Step()
		if p := mustPlayer(t, w, id); p.Cold != c.cold {
			t.Errorf("season %v night: cold %v, want %v", SeasonOf(c.tick), p.Cold, c.cold)
		}
	}
}

func huntWorld(tick uint64) (*World, PlayerID) {
	w := flatWorld()
	w.style = StyleSeas
	w.tick = tick
	id := w.AddPlayer("a")
	return w, id
}

func troodons(w *World) []*Creature {
	var out []*Creature
	for _, c := range w.fauna.creatures {
		if c.Kind == CreatureTroodon && !c.gone {
			out = append(out, c)
		}
	}
	return out
}

func TestTroodonsHuntAtNightAndFearFire(t *testing.T) {
	w, id := huntWorld(deepNight - deepNight%TroodonSpawnEvery + TroodonSpawnEvery - 1)
	w.players[id].Inv = Inventory{}
	w.Step()
	pack := troodons(w)
	if len(pack) == 0 {
		t.Fatal("no night hunters came out")
	}
	p := w.players[id]
	for _, c := range pack {
		if d := math.Hypot(c.X-p.X, c.Y-p.Y); d < troodonSpawnNear-3 {
			t.Errorf("a troodon came out %.1f tiles from the player, in plain sight", d)
		}
	}
	// Close the distance, and they bite.
	for _, c := range pack {
		c.X, c.Y = p.X+3, p.Y
	}
	h := p.Health
	stepN(w, 4*TicksPerSecond)
	if p.Health >= h {
		t.Fatal("the pack did not bite")
	}

	// By a fire they keep off and wait at the edge of the light.
	w2, id2 := huntWorld(deepNight)
	q := w2.players[id2]
	w2.addEntity(EntityCampfire, int(q.X)+1, int(q.Y))
	fire := w2.addCreature(CreatureTroodon, q.X+6, q.Y)
	h2 := q.Health
	stepN(w2, 6*TicksPerSecond)
	if q.Health < h2 || w2.inLight(fire.X, fire.Y) {
		t.Fatalf("by the fire: health %.0f of %.0f, troodon in the light %v", q.Health, h2, w2.inLight(fire.X, fire.Y))
	}
}

func TestTroodonsLeaveAtDawnAndSpareSleepers(t *testing.T) {
	w, id := huntWorld(deepNight)
	p := w.players[id]
	c := w.addCreature(CreatureTroodon, p.X+20, p.Y)
	w.tick = uint64(2*DayTicks) + uint64(0.3*DayTicks) // the next morning
	stepN(w, 10*TicksPerSecond)
	if !c.gone {
		t.Fatal("a troodon still about by day")
	}
	w2, id2 := huntWorld(deepNight)
	w2.players[id2].Asleep = true
	if q, _ := w2.troodonPrey(w2.addCreature(CreatureTroodon, w2.players[id2].X+3, w2.players[id2].Y)); q != nil {
		t.Fatal("night hunters go after a sleeper")
	}
}
