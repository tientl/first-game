package client

import (
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// newVoices are the banks made in creaturevoice.go and tribesound.go (the
// bear's, recorded now, are tested in realvoice_test.go).
var newVoices = []string{
	"tribe_warcry", "tribe_shout", "tribe_grunt", "tribe_pain", "tribe_death",
	"bellow_hurt", "honk", "beast_death", "raptor_yelp",
	"spider_chitter", "scorpion_click", "sting", "monkey_chatter", "monkey_screech",
	"mosa_rumble", "kraken_groan",
}

var (
	bankOnce sync.Once
	bank     map[string][][]float32
)

// sharedBank is genSFX, made once for all the tests here.
func sharedBank() map[string][][]float32 {
	bankOnce.Do(func() { bank = genSFX() })
	return bank
}

func TestCreatureVoicesAreUsable(t *testing.T) {
	b := sharedBank()
	for _, name := range newVoices {
		vs := b[name]
		if len(vs) < 2 && name != "sting" {
			t.Errorf("%s: %d variants, want several", name, len(vs))
		}
		for _, x := range vs {
			checkFinite(t, name, x)
			peak, rms := stats(x)
			if peak > 1.0001 || peak < 0.5 || rms < 0.02 {
				t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
			}
			if end := math.Abs(float64(x[len(x)-1])); end > 0.01 {
				t.Errorf("%s: ends at %.3f, would click", name, end)
			}
		}
	}
	if n := len(b["tribe_warcry"]); n < 5 {
		t.Errorf("%d war cries: want several, so no two in a row are the same", n)
	}
	// Every voice a creature is given is a sound.
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		kv := voicesOf(k)
		for _, u := range []voiceUse{kv.alarm, kv.hurt, kv.herd, kv.death} {
			if u.name != "" && len(b[u.name]) == 0 {
				t.Errorf("%s says %q, which is not a sound", k.Name(), u.name)
			}
		}
		if v, ok := callOf(k); ok && len(b[v.name]) == 0 {
			t.Errorf("%s calls with %q, which is not a sound", k.Name(), v.name)
		}
	}
	for k, v := range seaCall {
		if len(b[v.name]) == 0 {
			t.Errorf("%s calls with %q, which is not a sound", k.Name(), v.name)
		}
	}
	// Bigger means lower: the bear's growl sits well under the monkey's chatter.
	if lo, hi := highShare(b["bear_growl"][0]), highShare(b["monkey_chatter"][0]); lo >= hi {
		t.Errorf("high end of the bear's growl %.3f, the monkey's chatter %.3f", lo, hi)
	}
}

// Every creature of the game makes some sound: a call, a cry, an alarm or
// a death cry, here or in the older files.
func TestEveryCreatureHasAVoice(t *testing.T) {
	older := map[sim.CreatureKind]bool{ // heard in creatures.go, dinosound.go and the like
		sim.CreatureSnake: true, sim.CreatureWorm: true, sim.CreatureFish: true, sim.CreatureCrab: true,
		sim.CreatureTribesman: true, // tribesound.go
	}
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		kv := voicesOf(k)
		_, calls := callOf(k)
		_, sea := seaCall[k]
		if !older[k] && !calls && !sea && kv.alarm.name == "" && kv.hurt.name == "" && kv.death.name == "" {
			t.Errorf("%s has no voice", k.Name())
		}
	}
}

// testVoices installs a fresh voice memory on a clock the test moves, and
// a silent sound system that records what it plays.
func testVoices(t *testing.T) (*voiceMemory, *[]string, func(time.Duration)) {
	t.Helper()
	var played []string
	s := &soundSystem{
		mix:      &mixer{},
		rng:      rand.New(rand.NewPCG(3, 4)),
		sfx:      sharedBank(),
		lastPlay: make(map[string]time.Time),
		onPlay:   func(n string) { played = append(played, n) },
	}
	clock := time.Unix(5000, 0)
	vm := newVoiceMemory()
	vm.now = func() time.Time { return clock }
	vm.roll = func() float64 { return 0.5 }
	oldSnd, oldVoices := snd, voices
	snd, voices = s, vm
	t.Cleanup(func() { snd, voices = oldSnd, oldVoices })
	return vm, &played, func(d time.Duration) { clock = clock.Add(d) }
}

func count(xs []string, x string) int {
	n := 0
	for _, s := range xs {
		if s == x {
			n++
		}
	}
	return n
}

func warrior(id sim.CreatureID, x, y float32, st sim.CreatureState) protocol.CreatureView {
	return protocol.CreatureView{ID: id, Kind: sim.CreatureTribesman, X: x, Y: y, State: st, HP: 15, MaxHP: 15}
}

func TestTribesmanSpottingAStrangerGivesAWarCry(t *testing.T) {
	_, played, tick := testVoices(t)
	snd.setListener(14, 10)
	g := readyGame()
	me := []protocol.PlayerView{{ID: 1, X: 14, Y: 10}}
	idle := protocol.Snapshot{Tick: 1, Players: me, Creatures: []protocol.CreatureView{
		warrior(5, 10, 10, sim.CreatureIdle), warrior(6, 12, 11, sim.CreatureWander), warrior(7, 9, 9, sim.CreatureIdle),
		warrior(8, 40, 40, sim.CreatureIdle), // too far off to hear him
	}}
	seen := idle
	seen.Tick = 2
	seen.Creatures = slices.Clone(idle.Creatures)
	seen.Creatures[0].State = sim.CreatureChase
	g.snapshotSounds(idle, seen)
	if count(*played, "tribe_warcry") != 1 {
		t.Fatalf("played %v: the warrior who saw me should cry out", *played)
	}
	if count(*played, "tribe_shout") != 0 {
		t.Fatalf("played %v: the others answer a moment later, not at once", *played)
	}
	// A moment later, the warriors near him answer one after another.
	for range 10 {
		tick(150 * time.Millisecond)
		g.snapshotSounds(seen, seen)
	}
	if n := count(*played, "tribe_shout"); n != 2 {
		t.Fatalf("played %v: the two warriors near him should answer", *played)
	}

	// They join the chase: having answered, they give no war cry of their own.
	join := seen
	join.Creatures = slices.Clone(seen.Creatures)
	join.Creatures[1].State, join.Creatures[2].State = sim.CreatureChase, sim.CreatureChase
	clear(snd.lastPlay)
	g.snapshotSounds(seen, join)
	if n := count(*played, "tribe_warcry"); n != 1 {
		t.Fatalf("played %v: the ones who answered should not cry out again", *played)
	}
}

func TestTribeWarCriesCoolDown(t *testing.T) {
	_, played, tick := testVoices(t)
	snd.setListener(14, 10)
	g := readyGame()
	me := []protocol.PlayerView{{ID: 1, X: 14, Y: 10}}
	at := func(st sim.CreatureState) protocol.Snapshot {
		return protocol.Snapshot{Players: me, Creatures: []protocol.CreatureView{warrior(5, 10, 10, st)}}
	}
	g.snapshotSounds(at(sim.CreatureIdle), at(sim.CreatureChase))
	// He loses sight of me for a moment, and sees me again: the same fight.
	for range 3 {
		tick(time.Second)
		clear(snd.lastPlay)
		g.snapshotSounds(at(sim.CreatureChase), at(sim.CreatureIdle))
		tick(time.Second)
		g.snapshotSounds(at(sim.CreatureIdle), at(sim.CreatureChase))
	}
	if n := count(*played, "tribe_warcry"); n != 1 {
		t.Fatalf("%d war cries in one fight: %v", n, *played)
	}
	// Long after, a new fight: a new cry.
	tick(tribeReCry + time.Second)
	clear(snd.lastPlay)
	g.snapshotSounds(at(sim.CreatureIdle), at(sim.CreatureChase))
	if n := count(*played, "tribe_warcry"); n != 2 {
		t.Fatalf("%d war cries, want a second for a new fight: %v", n, *played)
	}

	// And two warriors spotting me at once do not cry over each other.
	tick(tribeReCry + time.Second)
	two := func(st sim.CreatureState) protocol.Snapshot {
		return protocol.Snapshot{Players: me, Creatures: []protocol.CreatureView{warrior(5, 10, 10, st), warrior(9, 21, 10, st)}}
	}
	clear(snd.lastPlay)
	g.snapshotSounds(two(sim.CreatureIdle), two(sim.CreatureChase))
	if n := count(*played, "tribe_warcry"); n != 3 {
		t.Fatalf("%d war cries, want one more for two at once: %v", n, *played)
	}
}

func TestTribesmanHuntingGameShoutsButNoWarCry(t *testing.T) {
	_, played, _ := testVoices(t)
	g := readyGame()
	far := []protocol.PlayerView{{ID: 1, X: 25, Y: 10}} // out of his sight: he is after a rabbit
	a := protocol.Snapshot{Players: far, Creatures: []protocol.CreatureView{warrior(5, 10, 10, sim.CreatureWander)}}
	b := protocol.Snapshot{Players: far, Creatures: []protocol.CreatureView{warrior(5, 10, 10, sim.CreatureChase)}}
	snd.setListener(14, 10)
	g.snapshotSounds(a, b)
	if count(*played, "tribe_warcry") != 0 || count(*played, "tribe_shout") != 1 {
		t.Fatalf("played %v: a hunting shout, not a war cry", *played)
	}
}

func TestTribesmanThrustsHurtsAndDies(t *testing.T) {
	_, played, tick := testVoices(t)
	snd.setListener(11, 10)
	g := readyGame()
	me := []protocol.PlayerView{{ID: 1, X: 11, Y: 10}}
	w := warrior(5, 10, 10, sim.CreatureChase)
	snap := func(cs ...protocol.CreatureView) protocol.Snapshot {
		return protocol.Snapshot{Players: me, Creatures: cs}
	}
	thrust := w
	thrust.State = sim.CreatureStrike
	g.snapshotSounds(snap(w), snap(thrust))
	if count(*played, "tribe_grunt") != 1 {
		t.Fatalf("played %v: a grunt with the thrust", *played)
	}
	hurt := w
	hurt.HP = 9
	g.snapshotSounds(snap(w), snap(hurt))
	hurt2 := hurt
	hurt2.HP = 6
	g.snapshotSounds(snap(hurt), snap(hurt2)) // hit again at once: one cry is enough
	if n := count(*played, "tribe_pain"); n != 1 {
		t.Fatalf("played %v: one cry of pain", *played)
	}
	tick(time.Second)
	hurt3 := hurt2
	hurt3.HP = 2
	g.snapshotSounds(snap(hurt2), snap(hurt3))
	if n := count(*played, "tribe_pain"); n != 2 {
		t.Fatalf("played %v: a cry for a later blow", *played)
	}
	// He falls: his body lies where he stood.
	body := protocol.EntityView{ID: 50, Kind: sim.EntityCorpse, TX: 10, TY: 9, Var: uint8(sim.CreatureTribesman)}
	gone := snap()
	gone.Entities = []protocol.EntityView{body}
	g.snapshotSounds(snap(hurt3), gone)
	if count(*played, "tribe_death") != 1 {
		t.Fatalf("played %v: a death cry", *played)
	}
}

func TestNoTwoWarCriesInARowAreTheSame(t *testing.T) {
	vm, played, _ := testVoices(t)
	snd.setListener(0, 0)
	last := -1
	for range 40 {
		vm.callPick("tribe_warcry", 1, 1, 1, 1, 20)
		i := vm.lastVar["tribe_warcry"]
		if i == last {
			t.Fatalf("variant %d twice in a row", i)
		}
		last = i
	}
	if len(*played) != 40 {
		t.Fatalf("%d played, want 40", len(*played))
	}
}

func TestAGrazerHitCallsOutAndItsHerdAnswers(t *testing.T) {
	_, played, tick := testVoices(t)
	snd.setListener(10, 10)
	g := readyGame()
	trike := protocol.CreatureView{ID: 3, Kind: sim.CreatureTrike, X: 12, Y: 10, State: sim.CreatureEat, HP: 100, MaxHP: 100}
	mate := protocol.CreatureView{ID: 4, Kind: sim.CreatureTrike, X: 15, Y: 11, State: sim.CreatureEat, HP: 100, MaxHP: 100}
	hit := trike
	hit.HP, hit.State = 80, sim.CreatureHurt
	g.snapshotSounds(snapOf(trike, mate), snapOf(hit, mate))
	if count(*played, "trike_hurt") != 1 {
		t.Fatalf("played %v: a three-horn hit should cry out", *played)
	}
	tick(time.Second)
	g.snapshotSounds(snapOf(hit, mate), snapOf(hit, mate))
	if count(*played, "trike_call") != 1 {
		t.Fatalf("played %v: the herd should call in alarm", *played)
	}
	// Hit again as it charges: its cry stands for the bellow it gives as
	// it turns on whoever hit it, and the herd has called already.
	tick(2 * time.Second) // past its last cry
	charging := trike
	charging.State = sim.CreatureChase
	again := hit
	again.HP = 60
	g.snapshotSounds(snapOf(charging, mate), snapOf(again, mate))
	charging.HP = 60
	g.snapshotSounds(snapOf(again, mate), snapOf(charging, mate))
	tick(time.Second)
	g.snapshotSounds(snapOf(charging, mate), snapOf(charging, mate))
	if count(*played, "trike_hurt") != 2 || count(*played, "trike_call") != 1 || count(*played, "trike_alarm") != 0 {
		t.Fatalf("played %v: one more cry, and no bellow over it", *played)
	}
	clear(snd.lastPlay)

	// A long-neck, which runs when hit and so never bellows at whoever hit it.
	brachio := protocol.CreatureView{ID: 9, Kind: sim.CreatureBrachio, X: 8, Y: 10, State: sim.CreatureWander, HP: 200, MaxHP: 200}
	hb := brachio
	hb.HP = 170
	g.snapshotSounds(snapOf(brachio), snapOf(hb))
	if count(*played, "brachio_hurt") != 1 {
		t.Fatalf("played %v: a long-neck hit should cry out too", *played)
	}
}

func TestBeastsDieCrying(t *testing.T) {
	_, played, _ := testVoices(t)
	snd.setListener(10, 10)
	g := readyGame()
	me := []protocol.PlayerView{{ID: 1, X: 10, Y: 10}}
	bear := protocol.CreatureView{ID: 3, Kind: sim.CreatureBear, X: 11, Y: 10, State: sim.CreatureChase, HP: 4, MaxHP: 60}
	body := protocol.EntityView{ID: 9, Kind: sim.EntityCorpse, TX: 11, TY: 10, Var: uint8(sim.CreatureBear)}
	g.snapshotSounds(protocol.Snapshot{Players: me, Creatures: []protocol.CreatureView{bear}},
		protocol.Snapshot{Players: me, Entities: []protocol.EntityView{body}})
	if count(*played, "bear_death") != 1 {
		t.Fatalf("played %v: the bear should groan as it dies", *played)
	}
	// A healthy beast walking out of view is not dying.
	*played = nil
	trike := protocol.CreatureView{ID: 4, Kind: sim.CreatureTrike, X: 16, Y: 10, State: sim.CreatureWander, HP: 100, MaxHP: 100}
	g.snapshotSounds(protocol.Snapshot{Players: me, Creatures: []protocol.CreatureView{trike}}, protocol.Snapshot{Players: me})
	if len(*played) != 0 {
		t.Fatalf("played %v for a beast going out of view", *played)
	}
}

func TestCaveAndSeaBeastsSpeak(t *testing.T) {
	_, played, _ := testVoices(t)
	snd.setListener(10, 10)
	g := readyGame()
	for _, c := range []struct {
		kind     sim.CreatureKind
		from, to sim.CreatureState
		want     string
	}{
		{sim.CreatureBear, sim.CreatureWander, sim.CreatureChase, "bear_roar"},
		{sim.CreatureSpider, sim.CreatureIdle, sim.CreatureChase, "spider_chitter"},
		{sim.CreatureScorpion, sim.CreatureIdle, sim.CreatureChase, "scorpion_click"},
		{sim.CreatureScorpion, sim.CreatureChase, sim.CreatureStrike, "sting"},
		{sim.CreatureMonkey, sim.CreatureIdle, sim.CreatureFlee, "monkey_screech"},
		{sim.CreatureMosa, sim.CreatureLurk, sim.CreatureChase, "mosa_call"},
		{sim.CreatureKraken, sim.CreatureLurk, sim.CreatureGrab, "kraken_call"},
	} {
		a := protocol.CreatureView{ID: sim.CreatureID(100 + c.kind), Kind: c.kind, X: 12, Y: 10, State: c.from, HP: 50, MaxHP: 50}
		b := a
		b.State = c.to
		g.snapshotSounds(snapOf(a), snapOf(b))
		if !slices.Contains(*played, c.want) {
			t.Errorf("%s from %d to %d: played %v, want %s", c.kind.Name(), c.from, c.to, *played, c.want)
		}
		*played = nil
	}
}

func TestSeaMonstersCallFromTheDeepNowAndThen(t *testing.T) {
	vm, played, tick := testVoices(t)
	snd.setListener(10, 10)
	beasts.any = time.Time{}
	mosa := protocol.CreatureView{ID: 1, Kind: sim.CreatureMosa, X: 14, Y: 10, State: sim.CreatureLurk}
	vm.roll = func() float64 { return 0 }
	vm.seaCalls(snapOf(mosa)) // starts the clock
	tick(200 * time.Millisecond)
	vm.seaCalls(snapOf(mosa))
	if count(*played, "mosa_call") != 1 {
		t.Fatalf("played %v: the Mosasaurus should rumble", *played)
	}
	tick(3 * time.Second)
	vm.seaCalls(snapOf(mosa))
	if count(*played, "mosa_call") != 1 {
		t.Fatalf("played %v: and not again so soon", *played)
	}
	vm.roll = func() float64 { return 0.999 }
	tick(20 * time.Second)
	vm.seaCalls(snapOf(mosa))
	if count(*played, "mosa_call") != 1 {
		t.Fatalf("played %v: calls should be rare", *played)
	}
}
