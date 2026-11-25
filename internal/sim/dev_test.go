package sim

import (
	"math"
	"strings"
	"sync"
	"testing"
)

// devWorldOnce is one dev-sized world, generated once for the dev tests;
// each test works on its own copy, made through a save.
var devWorldOnce = sync.OnceValues(func() (SaveData, error) {
	w, err := Generate(GenConfig{Seed: 20240601, Width: SizeMedium, Height: SizeMedium, Style: StyleLatest})
	if err != nil {
		return SaveData{}, err
	}
	return w.Save(), nil
})

func newDevWorld(t *testing.T) (*World, PlayerID) {
	t.Helper()
	d, err := devWorldOnce()
	if err != nil {
		t.Fatal(err)
	}
	w, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	return w, w.AddPlayer("Dev")
}

func devRun(t *testing.T, w *World, id PlayerID, line string) DevResult {
	t.Helper()
	r := w.DevCommand(id, line)
	t.Logf("%s -> %s", line, strings.Join(r.Lines, " | "))
	return r
}

func TestFoldMatchesWithoutAccents(t *testing.T) {
	if got := fold("Rìu Đá-lửa!"); got != "riudalua" {
		t.Errorf("fold = %q", got)
	}
	for q, want := range map[string]ItemKind{"go": ItemWood, "GỖ": ItemWood, "wood": ItemWood, "riu da lua": ItemCopperAxe, "1": ItemWood, "thit nuong": ItemCookedMeat} {
		if k, ok := FindItem(q); !ok || k != want {
			t.Errorf("FindItem(%q) = %v %v, want %v", q, k, ok, want)
		}
	}
	if _, ok := FindItem("riu"); ok {
		t.Error("an ambiguous name should match nothing")
	}
	for q, want := range map[string]CreatureKind{"tho": CreatureRabbit, "rex": CreatureRex, "bao chua": CreatureRex, "khung long ba sung": CreatureTrike} {
		if k, ok := FindCreature(q); !ok || k != want {
			t.Errorf("FindCreature(%q) = %v %v, want %v", q, k, ok, want)
		}
	}
}

func TestDevGive(t *testing.T) {
	w, id := newDevWorld(t)
	devRun(t, w, id, "give gỗ 25")
	devRun(t, w, id, `give "Rìu đá lửa"`)
	p := w.players[id]
	if n := p.Inv.Count(ItemWood); n != 25 {
		t.Errorf("wood = %d, want 25", n)
	}
	if n := p.Inv.Count(ItemCopperAxe); n != 1 {
		t.Errorf("flint axes = %d, want 1", n)
	}
	if r := devRun(t, w, id, "give khongcodo 3"); len(r.Lines) == 0 || !strings.Contains(r.Lines[0], "Không biết") {
		t.Errorf("an unknown item should be refused: %v", r.Lines)
	}
}

func TestDevTeleport(t *testing.T) {
	w, id := newDevWorld(t)
	p := w.players[id]
	for _, line := range []string{"tp village", "tp làng", "tp boss 1", "tp trùm 6", "tp hang", "tp spawn"} {
		before := [2]float64{p.X, p.Y}
		r := devRun(t, w, id, line)
		if !strings.HasPrefix(r.Lines[0], "Đã tới") {
			t.Errorf("%s: %v", line, r.Lines)
			continue
		}
		if !w.standable(p.X, p.Y) && !w.swimmable(p.X, p.Y) {
			t.Errorf("%s: landed where no one can be, %.1f %.1f", line, p.X, p.Y)
		}
		if line != "tp spawn" && line != "tp làng" && before == [2]float64{p.X, p.Y} {
			t.Errorf("%s: did not move", line)
		}
	}

	devRun(t, w, id, "tp village")
	v := w.villages[0]
	best := math.Inf(1)
	for _, o := range w.villages {
		best = min(best, math.Hypot(o.X-p.X, o.Y-p.Y))
	}
	if best > 6 {
		t.Errorf("tp village landed %.1f tiles from the nearest village (first at %.0f %.0f)", best, v.X, v.Y)
	}

	devRun(t, w, id, "tp boss 2")
	lair := w.bosses.lairs[1]
	if d := math.Hypot(lair.x-p.X, lair.y-p.Y); d < 7 || d > devLandReach+1 {
		t.Errorf("tp boss 2 landed %.1f tiles from the lair", d)
	}

	devRun(t, w, id, "tp cave")
	if w.surfaceH > 0 && !w.Underground(int(p.Y)) {
		t.Errorf("tp cave left the player above ground at %.1f %.1f", p.X, p.Y)
	}

	r := devRun(t, w, id, "tp kim tự tháp")
	if _, _, ok := w.Pyramid(); ok != strings.HasPrefix(r.Lines[0], "Đã tới") {
		t.Errorf("tp pyramid (the world has one: %v): %v", ok, r.Lines)
	}

	devRun(t, w, id, "tp 40.5 41.5")
	if p.X != 40.5 || p.Y != 41.5 {
		t.Errorf("tp x y: at %.1f %.1f", p.X, p.Y)
	}
	if r := devRun(t, w, id, "tp 99999 5"); p.X != 40.5 || !strings.Contains(r.Lines[0], "ngoài") {
		t.Errorf("tp off the map should be refused: %v", r.Lines)
	}
}

func TestDevSpawnAndKill(t *testing.T) {
	w, id := newDevWorld(t)
	devRun(t, w, id, "tp spawn")
	devRun(t, w, id, "kill 30")
	p := w.players[id]
	count := func(k CreatureKind) int {
		n := 0
		for _, c := range w.fauna.creatures {
			if !c.gone && c.Kind == k && math.Hypot(c.X-p.X, c.Y-p.Y) <= DevKillRadius {
				n++
			}
		}
		return n
	}
	devRun(t, w, id, "spawn thỏ 4")
	devRun(t, w, id, "spawn trike 2")
	if n := count(CreatureRabbit); n != 4 {
		t.Errorf("rabbits near = %d, want 4", n)
	}
	if n := count(CreatureTrike); n != 2 {
		t.Errorf("three-horns near = %d, want 2", n)
	}
	if r := devRun(t, w, id, "spawn titanoboa"); count(CreatureTitanoboa) != 0 || len(r.Lines) == 0 {
		t.Error("a boss should not be spawned")
	}
	r := devRun(t, w, id, "kill")
	if count(CreatureRabbit)+count(CreatureTrike) != 0 {
		t.Errorf("kill left beasts: %v", r.Lines)
	}
	w.Step()
	if n := count(CreatureRabbit); n != 0 {
		t.Errorf("after a step, %d rabbits are left", n)
	}
}

func TestDevTimeSeasonWeather(t *testing.T) {
	w, id := newDevWorld(t)
	start := w.Tick()
	devRun(t, w, id, "time night")
	if PhaseOf(w.Tick()) != PhaseNight || w.Tick() < start {
		t.Errorf("time night: phase %v at %.2f", PhaseOf(w.Tick()), DayFraction(w.Tick()))
	}
	devRun(t, w, id, "time 0.6")
	if f := DayFraction(w.Tick()); math.Abs(f-0.6) > 0.001 || PhaseOf(w.Tick()) != PhaseDusk {
		t.Errorf("time 0.6: at %.3f", f)
	}
	devRun(t, w, id, "time day")
	if PhaseOf(w.Tick()) != PhaseDay {
		t.Error("time day: not day")
	}

	before := w.Tick()
	devRun(t, w, id, "season đông")
	if w.Season() != Winter || w.Tick() <= before {
		t.Errorf("season winter: %v", w.Season())
	}
	devRun(t, w, id, "season spring")
	if w.Season() != Spring {
		t.Errorf("season spring: %v", w.Season())
	}

	devRun(t, w, id, "weather storm")
	for range 3 * WeatherTicks / 100 {
		w.Step()
	}
	if w.Weather() != WeatherStorm || !w.DevWeatherForced() {
		t.Errorf("weather storm: %v", w.Weather())
	}
	if _, s := w.Wind(); s != 1 {
		t.Errorf("a storm's wind = %v, want 1", s)
	}
	devRun(t, w, id, "weather tuyết")
	if w.Weather() != WeatherSnow {
		t.Errorf("weather snow: %v", w.Weather())
	}
	devRun(t, w, id, "weather auto")
	if w.DevWeatherForced() || w.Weather() != WeatherAt(w.seed, w.Tick()) {
		t.Error("weather auto should follow the seed again")
	}
}

func TestDevGodMode(t *testing.T) {
	w, id := newDevWorld(t)
	p := w.players[id]
	if r := devRun(t, w, id, "god"); !w.DevGod(id) || !strings.Contains(r.Lines[0], "bật") {
		t.Fatalf("god did not turn on: %v", r.Lines)
	}
	devRun(t, w, id, "season winter")
	devRun(t, w, id, "time night")
	p.Hunger = 0
	hurt(p, 500)
	for range 200 {
		w.Step()
	}
	if p.Dead || p.Health < MaxHealth-1 || p.Hunger != MaxHunger || p.Cold {
		t.Errorf("in god mode: dead %v, health %.1f, hunger %.1f, cold %v", p.Dead, p.Health, p.Hunger, p.Cold)
	}

	devRun(t, w, id, "god off")
	hurt(p, 30)
	if w.DevGod(id) || p.Health > MaxHealth-29 {
		t.Errorf("god off: health %.1f", p.Health)
	}
	devRun(t, w, id, "heal")
	if p.Health != MaxHealth || p.Hunger != MaxHunger {
		t.Errorf("heal: health %.1f hunger %.1f", p.Health, p.Hunger)
	}
}

func TestDevSpeed(t *testing.T) {
	w, id := newDevWorld(t)
	p := w.players[id]
	walk := func() float64 {
		devRun(t, w, id, "tp spawn")
		x0, y0 := p.X, p.Y
		best := 0.0
		for _, d := range [][2]int8{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			w.DevTeleport(id, x0, y0)
			w.SetInput(id, Input{MoveX: d[0], MoveY: d[1]})
			w.Step()
			best = max(best, math.Hypot(p.X-x0, p.Y-y0))
		}
		w.SetInput(id, Input{})
		return best
	}
	base := walk()
	devRun(t, w, id, "speed 3")
	fast := walk()
	if base == 0 || math.Abs(fast/base-3) > 0.01 {
		t.Errorf("a step at speed 3 is %.3f, at 1 %.3f", fast, base)
	}
	if r := devRun(t, w, id, "speed 1000"); !strings.Contains(r.Lines[0], "8") {
		t.Errorf("speed should be capped: %v", r.Lines)
	}
}

func TestDevReveal(t *testing.T) {
	w, id := newDevWorld(t)
	r := devRun(t, w, id, "reveal")
	if !r.Revealed {
		t.Error("reveal should say the map changed")
	}
	wd, ht := w.Size()
	if n := ExploredFrom(wd, ht, w.Explored(id)).Count(); n != wd*ht {
		t.Errorf("explored %d of %d tiles", n, wd*ht)
	}
}

func TestDevCommandsUnknown(t *testing.T) {
	w, id := newDevWorld(t)
	for _, line := range []string{"nope", "tp nowhere", "time abc", "season foo", "weather hail", "list"} {
		if r := devRun(t, w, id, line); len(r.Lines) == 0 {
			t.Errorf("%q answered nothing", line)
		}
	}
	if r := devRun(t, w, id, "help"); len(r.Lines) != len(DevHelp) {
		t.Error("help should list the commands")
	}
	if r := devRun(t, w, id, "list beasts"); len(r.Lines) == 0 || !strings.Contains(strings.Join(r.Lines, " "), "Thỏ") {
		t.Errorf("list beasts: %v", r.Lines)
	}
	if r := devRun(t, w, id, "list items riu"); !strings.Contains(strings.Join(r.Lines, " "), "Rìu đá") {
		t.Errorf("list items riu: %v", r.Lines)
	}
}

// The dev state never reaches a save: a world saved after dev commands
// loads with no god mode, no forced weather and normal speed.
func TestDevStateNotSaved(t *testing.T) {
	w, id := newDevWorld(t)
	devRun(t, w, id, "god on")
	devRun(t, w, id, "weather storm")
	devRun(t, w, id, "speed 4")
	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	if w2.DevWeatherForced() {
		t.Error("forced weather was saved")
	}
	id2 := w2.AddPlayer("Dev")
	if w2.DevGod(id2) || w2.players[id2].dev.speedFactor() != 1 {
		t.Error("god mode or speed came back from the save")
	}
}
