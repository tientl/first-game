package sim

import (
	"math"
	"testing"
)

func TestPhasesAreLongEnough(t *testing.T) {
	minutes := func(frac float64) float64 { return frac * DayTicks / TicksPerSecond / 60 }
	if m := minutes(DuskStart); m < 4 {
		t.Errorf("the day lasts %.1f minutes, want at least 4", m)
	}
	if m := minutes(NightStart - DuskStart); m < 1 {
		t.Errorf("dusk lasts %.1f minutes, want at least 1", m)
	}
	if m := minutes(1 - NightStart); m < 2 {
		t.Errorf("the night lasts %.1f minutes, want at least 2", m)
	}
}

func TestHungerAndSleepKeepTheirPace(t *testing.T) {
	// Hunger is paced by the clock on the wall, not by the length of a day.
	if m := MaxHunger / HungerPerTick / TicksPerSecond / 60; math.Abs(m-10) > 0.01 {
		t.Errorf("a full belly lasts %.2f minutes, want 10", m)
	}
	// Going to bed at dusk and waking at full daylight costs what it did
	// when days were half as long: under a third of the bar.
	bedTime := (1 - DuskStart + transition) * DayTicks
	if cost := bedTime * HungerPerTick * SleepHungerFactor; cost > MaxHunger/3 {
		t.Errorf("a night in bed costs %.1f hunger, want at most a third of %v", cost, MaxHunger)
	}
}

func TestVersion6SaveKeepsTheDayAndTheTimeOfDay(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 48, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	d.Version = 6
	// Day 3, three quarters through: night, in the old, shorter days.
	d.Tick = 2*v6DayTicks + 3*v6DayTicks/4
	i := -1
	for j, e := range d.Entities {
		if e.Kind == EntityTree {
			i = j
			break
		}
	}
	if i < 0 {
		t.Fatal("no tree to cut down")
	}
	const left = v6DayTicks / 2 // half a day until it grows back
	d.Entities[i].Ready, d.Entities[i].RegrowAt = false, d.Tick+left

	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if DayOf(w2.Tick()) != 3 || DayFraction(w2.Tick()) != 0.75 || PhaseOf(w2.Tick()) != PhaseNight {
		t.Errorf("after loading: day %d at %.3f (%v), want day 3 at 0.750, night", DayOf(w2.Tick()), DayFraction(w2.Tick()), PhaseOf(w2.Tick()))
	}
	stump := w2.entities[d.Entities[i].ID]
	if stump == nil || stump.Ready || stump.RegrowAt-w2.Tick() != DayTicks/2 {
		t.Fatalf("stump after loading: %+v at tick %d, want half a (new) day of regrowth left", stump, w2.Tick())
	}

	// A version 7 save keeps its ticks as they are.
	d2 := w2.Save()
	w3, err := Load(d2)
	if err != nil {
		t.Fatal(err)
	}
	if w3.Tick() != w2.Tick() || w3.entities[stump.ID].RegrowAt != stump.RegrowAt {
		t.Errorf("saved and loaded again: tick %d regrow %d, want %d and %d", w3.Tick(), w3.entities[stump.ID].RegrowAt, w2.Tick(), stump.RegrowAt)
	}
}

func TestSummerDaysAreLongerThanWinterDays(t *testing.T) {
	if m := float64(DayTicks) / TicksPerSecond / 60; m < 15 || m > 20 {
		t.Errorf("a day lasts %.1f minutes, want 15 to 20", m)
	}
	day := func(s Season, n int) uint64 { return uint64(int(s)*SeasonDays+n) * DayTicks }
	summer, winter := day(Summer, SeasonDays/2), day(Winter, SeasonDays/2)
	if DuskAt(summer) <= DuskAt(winter)+0.1 || NightAt(summer) <= NightAt(winter)+0.1 {
		t.Errorf("summer dusk %.2f night %.2f, winter dusk %.2f night %.2f: summer days should be clearly longer",
			DuskAt(summer), NightAt(summer), DuskAt(winter), NightAt(winter))
	}
	for d := range SeasonDays * int(NumSeasons) {
		tk := uint64(d) * DayTicks
		if NightAt(tk)-DuskAt(tk) < 0.1 || 1-NightAt(tk) < 0.18 {
			t.Fatalf("day %d: dusk %.2f night %.2f", d+1, DuskAt(tk), NightAt(tk))
		}
	}
	// Within one day the times do not move.
	if DuskAt(summer) != DuskAt(summer+DayTicks-1) {
		t.Error("dusk moves during the day")
	}
}

func TestVersion12SaveKeepsTheDayAndTheTimeOfDay(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 48, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	d.Version = 12
	d.Tick = 4*v12DayTicks + v12DayTicks/4 // day 5, a quarter through
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if DayOf(w2.Tick()) != 5 || DayFraction(w2.Tick()) != 0.25 {
		t.Errorf("after loading: day %d at %.3f, want day 5 at 0.250", DayOf(w2.Tick()), DayFraction(w2.Tick()))
	}
}
