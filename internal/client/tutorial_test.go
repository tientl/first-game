package client

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

const tutMe = sim.PlayerID(1)

// tutSnap is a snapshot of a player at (50, 50) holding what is given.
func tutSnap(items map[sim.ItemKind]int) *protocol.Snapshot {
	s := &protocol.Snapshot{Tick: 100, Players: []protocol.PlayerView{{ID: tutMe, X: 50, Y: 50}}}
	s.Self.Hunger = sim.MaxHunger
	for k, n := range items {
		s.Self.Inv.Add(k, n)
	}
	return s
}

func newTestTutorial() *tutorial { return &tutorial{flash: -1} }

// stepAfter feeds s to t and returns the goal it is at.
func stepAfter(t *tutorial, s *protocol.Snapshot, under bool) int {
	t.observe(s, tutMe, under, time.Now())
	return t.step
}

// Each goal is done by the right thing, and not before.
func TestTutorialProgression(t *testing.T) {
	tu := newTestTutorial()
	bag := map[sim.ItemKind]int{}
	step := func(want int, why string) {
		t.Helper()
		if got := stepAfter(tu, tutSnap(bag), false); got != want {
			t.Fatalf("%s: at goal %d, want %d", why, got, want)
		}
	}
	step(0, "an empty bag")
	bag[sim.ItemStick] = 3
	step(0, "sticks and no pebble")
	bag[sim.ItemPebble] = 1
	step(1, "sticks and a pebble")
	bag[sim.ItemStick], bag[sim.ItemPebble] = 0, 0 // used up on the axe
	step(1, "the bag emptied")
	bag[sim.ItemAxe] = 1
	step(2, "an axe")
	bag[sim.ItemWood] = 1
	step(2, "one wood")
	bag[sim.ItemWood] = 2
	step(3, "two wood")
	bag[sim.ItemStone], bag[sim.ItemGrass] = 3, 1
	step(3, "stone but too little grass")
	bag[sim.ItemGrass] = 2
	step(4, "stone and grass")

	// A fire: only a burning campfire near counts, not one far away or out.
	s := tutSnap(bag)
	s.Entities = []protocol.EntityView{{Kind: sim.EntityCampfire, TX: 90, TY: 50, Fuel: 500}}
	if stepAfter(tu, s, false) != 4 {
		t.Fatal("a campfire far away counts")
	}
	s.Entities = []protocol.EntityView{{Kind: sim.EntityCampfire, TX: 52, TY: 50, Fuel: 0}}
	if stepAfter(tu, s, false) != 4 {
		t.Fatal("a dead campfire counts")
	}
	s.Entities[0].Fuel = 500
	if stepAfter(tu, s, false) != 5 {
		t.Fatal("a burning campfire near does not count")
	}

	// Eating: the hunger going up, not the hunger being high.
	s = tutSnap(bag)
	s.Self.Hunger = 40
	if stepAfter(tu, s, false) != 5 {
		t.Fatal("being hungry counts as a meal")
	}
	s = tutSnap(bag)
	s.Self.Hunger = 41
	if stepAfter(tu, s, false) != 5 {
		t.Fatal("a sliver of hunger counts as a meal")
	}
	s = tutSnap(bag)
	s.Self.Hunger = 55
	if stepAfter(tu, s, false) != 6 {
		t.Fatal("a meal does not count")
	}

	step(6, "nothing yet for the torch")
	bag[sim.ItemTorch] = 1
	step(7, "a torch")
	bag[sim.ItemLeafTent] = 1
	step(8, "a tent")
	if stepAfter(tu, tutSnap(bag), false) != 8 || tu.over {
		t.Fatal("done before the cave")
	}
	tu.observe(tutSnap(bag), tutMe, true, time.Now())
	if !tu.over || tu.step != len(tutSteps) || tu.finishedAt.IsZero() {
		t.Fatalf("not finished underground: step %d, over %v", tu.step, tu.over)
	}
}

// A tent standing near counts as one made; so does a bag that holds what
// a later goal wants (the guide skips what is done).
func TestTutorialSkipsWhatIsDone(t *testing.T) {
	tu := newTestTutorial()
	s := tutSnap(map[sim.ItemKind]int{sim.ItemAxe: 1, sim.ItemWood: 4, sim.ItemStone: 3, sim.ItemGrass: 2})
	s.Entities = []protocol.EntityView{{Kind: sim.EntityLeafTent, TX: 49, TY: 50}}
	if got := stepAfter(tu, s, false); got != 4 {
		t.Errorf("at goal %d, want 4 (the fire)", got)
	}
	s.Entities = append(s.Entities, protocol.EntityView{Kind: sim.EntityCampfire, TX: 51, TY: 50, Fuel: 100})
	if got := stepAfter(tu, s, false); got != 5 {
		t.Errorf("at goal %d, want 5 (the meal)", got)
	}
}

func TestTutorialPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tutorial.json")
	key := worldKey(42, "Tiến")

	tu := loadTutorial(path, key, true)
	if tu.over || tu.step != 0 {
		t.Fatalf("a new game: step %d, over %v", tu.step, tu.over)
	}
	stepAfter(tu, tutSnap(map[sim.ItemKind]int{sim.ItemStick: 2, sim.ItemPebble: 1}), false)
	if tu.step != 1 {
		t.Fatalf("step %d", tu.step)
	}

	// "Chơi tiếp": the same world carries on where it was, whatever `fresh`.
	for _, fresh := range []bool{false, true} {
		again := loadTutorial(path, key, fresh)
		if again.step != 1 || again.over {
			t.Errorf("reloaded (fresh %v): step %d, over %v", fresh, again.step, again.over)
		}
	}
	// Another world, or another player, starts afresh.
	if o := loadTutorial(path, worldKey(43, "Tiến"), true); o.step != 0 || o.over {
		t.Errorf("another seed: step %d, over %v", o.step, o.over)
	}
	if o := loadTutorial(path, worldKey(42, "Lan"), true); o.step != 0 || o.over {
		t.Errorf("another player: step %d, over %v", o.step, o.over)
	}
	// A save with no record (from before the guide) has none.
	if o := loadTutorial(path, worldKey(7, "Cũ"), false); !o.over {
		t.Error("an old save got a guide")
	}
	// Finished: never again.
	all := map[sim.ItemKind]int{sim.ItemAxe: 1, sim.ItemWood: 2, sim.ItemStone: 3, sim.ItemGrass: 2, sim.ItemTorch: 1, sim.ItemLeafTent: 1}
	hungry := tutSnap(all)
	hungry.Self.Hunger = 30
	tu.observe(hungry, tutMe, true, time.Now())
	fed := tutSnap(all)
	fed.Entities = []protocol.EntityView{{Kind: sim.EntityCampfire, TX: 51, TY: 50, Fuel: 100}}
	tu.observe(fed, tutMe, true, time.Now())
	if !tu.over {
		t.Fatal("the guide did not finish")
	}
	if again := loadTutorial(path, key, true); !again.over {
		t.Error("a finished guide came back")
	}
	// A broken file is not fatal.
	if o := loadTutorial(filepath.Join(t.TempDir(), "none", "t.json"), key, true); o.over {
		t.Error("a missing file ended the guide")
	}
}

func TestTutorialNeverInDev(t *testing.T) {
	g := &Game{tut: newTestTutorial()}
	if !g.tutorialOn() {
		t.Fatal("off in a normal game")
	}
	g.dev = &devConsole{}
	if g.tutorialOn() {
		t.Error("on in a dev game")
	}
	g.dev = nil
	keep := settings
	defer func() { settings = keep }()
	settings.NoTutorial = true
	if g.tutorialOn() {
		t.Error("on though the player turned it off")
	}
	if (&Game{}).tutorialOn() {
		t.Error("on with no guide")
	}
}

// The tips: each once, in its moment, and remembered across worlds.
func TestTutorialTips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tutorial.json")
	tu := loadTutorial(path, worldKey(1, "A"), true)
	now := time.Now()
	next := func() time.Time { now = now.Add(tutTipGap + time.Second); return now }
	tip := func(s *protocol.Snapshot, under bool) string {
		_, tip := tu.observe(s, tutMe, under, next())
		return tip
	}

	day := tutSnap(nil)
	if got := tip(day, false); got != "" {
		t.Errorf("a quiet day said %q", got)
	}

	dusk := tutSnap(nil)
	dusk.Tick = uint64(sim.DuskAt(0)*sim.DayTicks) + 10
	if sim.PhaseOf(dusk.Tick) == sim.PhaseDay {
		t.Fatal("test tick is not evening")
	}
	if got := tip(dusk, false); !strings.Contains(got, "tối") {
		t.Errorf("nightfall said %q", got)
	}
	if got := tip(dusk, false); got != "" {
		t.Errorf("nightfall said twice: %q", got)
	}

	cold := tutSnap(nil)
	cold.Self.Cold = true
	if got := tip(cold, false); !strings.Contains(got, "Lạnh") {
		t.Errorf("cold said %q", got)
	}
	hungry := tutSnap(nil)
	hungry.Self.Hunger = 10
	if got := tip(hungry, false); !strings.Contains(got, "Đói") {
		t.Errorf("hunger said %q", got)
	}
	if got := tip(tutSnap(nil), true); !strings.Contains(got, "đuốc") {
		t.Errorf("a dark cave said %q", got)
	}
	lit := tutSnap(nil)
	lit.Players[0].Held = sim.ItemTorch
	lair := lit
	lair.Lairs = []protocol.LairView{{Kind: sim.CreatureTitanoboa, X: 60, Y: 50}}
	if got := tip(lair, false); !strings.Contains(got, "trùm") {
		t.Errorf("a lair said %q", got)
	}

	// A red zone under the player says to leave it, and to raise a shield if
	// there is one; one far away says nothing.
	far := tutSnap(nil)
	far.Zones = []protocol.ZoneView{{Shape: sim.ZoneCircle, X: 80, Y: 50, Reach: 3, Left: 10, Total: 20}}
	if got := tip(far, false); got != "" {
		t.Errorf("a far zone said %q", got)
	}
	near := tutSnap(nil)
	near.Players[0].Shield = true
	near.Zones = []protocol.ZoneView{{Shape: sim.ZoneCircle, X: 51, Y: 50, Reach: 3, Left: 10, Total: 20}}
	if got := tip(near, false); !strings.Contains(got, "Q") {
		t.Errorf("a zone with a shield said %q", got)
	}

	// All said: nothing more, here or in a later world.
	for _, s := range []*protocol.Snapshot{dusk, cold, hungry, lair, near} {
		if got := tip(s, true); got != "" {
			t.Errorf("a tip said again: %q", got)
		}
	}
	other := loadTutorial(path, worldKey(2, "B"), true)
	if _, got := other.observe(cold, tutMe, false, time.Now()); got != "" {
		t.Errorf("the cold tip came back in another world: %q", got)
	}

	// Not twice in a row: two moments in one breath say the first only.
	fresh := loadTutorial(filepath.Join(t.TempDir(), "t.json"), worldKey(3, "C"), true)
	both := tutSnap(nil)
	both.Self.Cold = true
	both.Self.Hunger = 5
	n := time.Now()
	_, first := fresh.observe(both, tutMe, false, n)
	_, second := fresh.observe(both, tutMe, false, n.Add(time.Second))
	if first == "" || second != "" {
		t.Errorf("tips %q then %q", first, second)
	}
}

// Every goal's text fits the tracker in the lines it has.
func TestTutorialTextFits(t *testing.T) {
	for i, s := range tutSteps {
		l := trackLayout(s)
		if len(l.title) > trackLines || len(l.hint) > trackLines {
			t.Errorf("goal %d %q: %d title lines, %d hint lines", i, s.title, len(l.title), len(l.hint))
		}
		if l.h > trackMaxH {
			t.Errorf("goal %d is %.0f tall, over %.0f", i, l.h, float64(trackMaxH))
		}
	}
	l := trackLayout(tutStep{title: "Hướng dẫn xong!", hint: "F1 xem lại các phím. Chúc bạn sống sót!"})
	if len(l.title) > 1 || len(l.hint) > trackLines || l.h > trackMaxH {
		t.Errorf("the end: %+v", l)
	}
}
