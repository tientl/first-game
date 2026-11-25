package sim

import (
	"math"
	"slices"
)

// The six bosses. Every open world has one great beast to each kind of
// wild land, living at a lair deep inside it and never straying far:
//
//   - Titanoboa, a snake as long as a boat, lies coiled in the undergrowth
//     of the deepest woods. It strikes, and it seizes a player in its
//     coils and squeezes until they break free or hurt it enough.
//   - Spinosaurus lurks in a marsh pond with only its sail and eyes above
//     the water, bursts out to bite whoever comes to the edge, and on land
//     swipes with its claws; now and then it slides back under.
//   - The mammoth king, huge and shaggy, roams the heart of the snow with
//     its herd. It swings its tusks, charges and throws players back, rears
//     up to stomp the ground round it, and calls the mammoths to the fight.
//   - Carcharodontosaurus hunts the deep desert: it bites, and charges at
//     a run, head low, through whoever is in its way.
//   - Giganotosaurus, bigger than the tyrant, rules the ash round the
//     volcano: a terrible bite, and a sweep of its tail that throws back
//     everyone round it.
//   - Megalodon, a shark as big as a whale, circles in the deep sea with
//     only its fin showing. It rams boats, bites swimmers and those aboard,
//     and now and then leaps clear of the water onto its prey.
//
// A boss sleeps at its lair until someone comes near, then roars and
// fights; it never goes further from its lair than its leash, and when
// everyone has gone it goes home and heals. Below half its health it is
// enraged: it roars again and attacks faster. A boss that is killed never
// comes back, which the save keeps. Its body gives a great deal of meat,
// hide and bone, and where it falls it drops one legendary item (see
// legendary.go). Once every boss of a world has fallen, every player is
// shown that the game is beaten, once.

// NumBosses is how many bosses there are, one to each kind of wild land.
const NumBosses = 6

// Boss tuning shared by all six. Each boss's own is in bossTunes.
const (
	BossBarRange   = 14.0 // a player this close to an awake boss sees its health bar
	bossSeeRange   = 16.0 // a player this close to a boss or its lair has seen it, for the map
	bossLeash      = 18.0 // a boss never chases further than this from its lair
	bossLoseTicks  = 3 * TicksPerSecond
	BossRoarTicks  = 3 * TicksPerSecond / 2 // a boss roars this long when a fight starts and when enraged
	bossWindup     = 12                     // ticks between a boss baring its teeth and the blow landing
	bossRecover    = 6                      // ticks after the blow before it moves again
	bossHealEvery  = TicksPerSecond         // at home with nobody about, it heals...
	bossHealShare  = 40                     // ...this share (1/n) of its health each time
	bossEnragePct  = 65                     // an enraged boss's cooldowns, in percent of the calm ones
	bossEnrageRun  = 1.15                   // and how much faster it runs
	bossWoundWake  = 6.0                    // a sleeping boss that is hurt turns on a player this close
	bossStuckTicks = 12 * TicksPerSecond    // a boss that cannot find its way home is put back
)

// bossTune is one boss's fighting numbers. Damage is before armour.
type bossTune struct {
	sight     float64 // a sleeping boss wakes for a player this close
	reach     float64 // its main blow reaches this far, centre to centre
	bite      float64 // what that blow does
	biteCD    int     // ticks between two of them
	knock     float64 // tiles that blow throws a player back
	specialCD int     // ticks between two of its special moves
	leash     float64 // how far from its lair it fights
	epithet   string  // what the players call it, after its name
}

var bossTunes = [NumBosses]bossTune{
	{sight: 7, reach: 2.8, bite: 36, biteCD: 50, specialCD: 9 * TicksPerSecond, leash: bossLeash, epithet: "Chúa rừng rậm"},
	{sight: 5, reach: 2.6, bite: 30, biteCD: 42, specialCD: 0, leash: bossLeash, epithet: "Chúa đầm lầy"},
	{sight: 9, reach: 3.2, bite: 34, biteCD: 56, knock: 1.5, specialCD: 8 * TicksPerSecond, leash: bossLeash, epithet: "Chúa xứ tuyết"},
	{sight: 10, reach: 2.8, bite: 38, biteCD: 46, specialCD: 6 * TicksPerSecond, leash: bossLeash, epithet: "Chúa sa mạc"},
	{sight: 10, reach: 3.0, bite: 46, biteCD: 54, specialCD: 7 * TicksPerSecond, leash: bossLeash, epithet: "Chúa núi lửa"},
	{sight: 16, reach: 2.6, bite: 24, biteCD: 44, specialCD: 14 * TicksPerSecond, leash: 34, epithet: "Chúa biển sâu"},
}

var (
	titanoboaDef = creatureDef{
		name: "Titanoboa", hp: 1650,
		walk: 0.8 / TicksPerSecond, run: 3.0 / TicksPerSecond,
		radius: 0.9, reach: 3.0,
		yield: ItemMeat, yieldN: 20, drops: []Ingredient{{ItemHide, 10}, {ItemBone, 10}},
		leash: 2, idle: [2]int{100, 300}, roam: [2]int{20, 40},
		fights: true, tough: true, openOnly: true,
	}
	spinoDef = creatureDef{
		name: "Spinosaurus", hp: 1500,
		walk: 0.9 / TicksPerSecond, run: 3.1 / TicksPerSecond,
		radius: 0.8, reach: 3.0,
		yield: ItemMeat, yieldN: 22, drops: []Ingredient{{ItemHide, 8}, {ItemBone, 8}, {ItemDinoHide, 4}},
		leash: 4, idle: [2]int{60, 200}, roam: [2]int{20, 60},
		wades:  true,
		fights: true, tough: true, openOnly: true,
	}
	mammothKingDef = creatureDef{
		name: "Voi Ma Mút Chúa", hp: 1650,
		walk: 0.8 / TicksPerSecond, run: 3.0 / TicksPerSecond,
		radius: 1.25, reach: 3.4,
		yield: ItemMeat, yieldN: 24, drops: []Ingredient{{ItemFur, 10}, {ItemIvory, 4}, {ItemHide, 4}, {ItemBone, 6}},
		leash: 5, idle: [2]int{60, 220}, roam: [2]int{30, 80},
		fights: true, tough: true, openOnly: true,
	}
	carcharoDef = creatureDef{
		name: "Carcharodontosaurus", hp: 1550,
		walk: 1.0 / TicksPerSecond, run: 3.6 / TicksPerSecond,
		radius: 0.95, reach: 3.0,
		yield: ItemMeat, yieldN: 20, drops: []Ingredient{{ItemHide, 8}, {ItemBone, 8}, {ItemDinoHide, 4}, {ItemRexTooth, 2}},
		leash: 6, idle: [2]int{60, 200}, roam: [2]int{30, 90},
		fights: true, tough: true, openOnly: true,
	}
	giganotoDef = creatureDef{
		name: "Giganotosaurus", hp: 1600,
		walk: 0.9 / TicksPerSecond, run: 3.3 / TicksPerSecond,
		radius: 1.1, reach: 3.3,
		yield: ItemMeat, yieldN: 24, drops: []Ingredient{{ItemHide, 8}, {ItemBone, 10}, {ItemDinoHide, 4}, {ItemRexTooth, 3}},
		leash: 6, idle: [2]int{60, 200}, roam: [2]int{30, 90},
		fights: true, tough: true, openOnly: true,
	}
	megalodonDef = creatureDef{
		name: "Megalodon", hp: 1000,
		walk: 1.8 / TicksPerSecond, run: 6.0 / TicksPerSecond,
		radius: 1.2, reach: 3.4,
		yield: ItemMeat, yieldN: 20, drops: []Ingredient{{ItemBone, 8}, {ItemHide, 6}},
		leash: 8, idle: [2]int{40, 120}, roam: [2]int{40, 120},
		deep:   true,
		fights: true, tough: true, openOnly: true,
	}
)

func init() {
	creatureDefs[CreatureTitanoboa] = titanoboaDef
	creatureDefs[CreatureSpino] = spinoDef
	creatureDefs[CreatureMammothKing] = mammothKingDef
	creatureDefs[CreatureCarcharo] = carcharoDef
	creatureDefs[CreatureGiganoto] = giganotoDef
	creatureDefs[CreatureMegalodon] = megalodonDef
}

// Boss reports whether k is one of the six bosses.
func (k CreatureKind) Boss() bool { return k >= CreatureTitanoboa && k <= CreatureMegalodon }

// BossKind is boss i, from Titanoboa (0) to Megalodon (5).
func BossKind(i int) CreatureKind { return CreatureTitanoboa + CreatureKind(i) }

// BossIndex is which boss k is, 0 to NumBosses-1; ok is false for any other kind.
func BossIndex(k CreatureKind) (int, bool) {
	if !k.Boss() {
		return 0, false
	}
	return int(k - CreatureTitanoboa), true
}

// BossEpithet is what players call boss k, after its name: "Chúa đầm lầy".
func BossEpithet(k CreatureKind) string {
	if i, ok := BossIndex(k); ok {
		return bossTunes[i].epithet
	}
	return ""
}

func bossTuneOf(k CreatureKind) *bossTune {
	i, _ := BossIndex(k)
	return &bossTunes[i]
}

// --- the world's bosses -----------------------------------------------------

// bossWorld is what a world knows of its bosses.
type bossWorld struct {
	lairs  [NumBosses]bossLair
	placed bool             // the lairs have been looked for, and the bosses raised
	won    uint64           // the tick the last boss fell; 0 while any still stands
	shown  map[string]bool  // players who have closed the victory screen, by name
	seen   map[string]uint8 // the lairs each player has seen, a bit per boss, by name
}

// bossLair is one boss: where it lives, and how its fight goes.
type bossLair struct {
	x, y     float64 // the lair
	ok       bool    // this world has somewhere for it
	defeated bool
	c        *Creature // the boss, while it lives

	awake   bool     // in a fight
	roared  bool     // it has roared its challenge in this fight
	enraged bool     // below half its health: it roars and attacks faster
	foe     PlayerID // who it fights
	hp      int      // its health a tick ago: a blow wakes it
	lost    int      // ticks since it last had a foe
	special int      // ticks to its next special move
	charge  int      // the mammoth king: ticks to its next charge
	landed  bool     // the move under way has struck
	grab    bool     // the bite under way is a grab (Titanoboa)
	held    PlayerID // the player in its coils
	grabHP  int      // its health when it seized them
	hx, hy  float64  // a charge's heading; where a leap lands
	act     int      // ticks into what it is doing: circling, out of the water
	stuck   int      // ticks it has been unable to get home
}

// placeBosses looks for a lair for each boss and raises it there. A world
// made by Generate places them at once; one put together by hand (tests)
// on its first step.
func (w *World) placeBosses() {
	b := &w.bosses
	b.placed = true
	if w.style == StyleIsland {
		return
	}
	for i := range b.lairs {
		l := &b.lairs[i]
		if !bossActive(BossKind(i)) {
			continue // not in the demo: see demo.go
		}
		l.x, l.y, l.ok = w.bossLairSite(BossKind(i))
		if l.ok {
			w.raiseBoss(i)
		}
	}
}

// raiseBoss puts boss i at its lair, asleep and whole.
func (w *World) raiseBoss(i int) {
	l := &w.bosses.lairs[i]
	k := BossKind(i)
	c := w.addCreature(k, l.x, l.y)
	c.homeX, c.homeY = l.x, l.y
	c.State = CreatureIdle
	if k == CreatureSpino || k == CreatureMegalodon {
		c.State = CreatureLurk
	}
	*l = bossLair{x: l.x, y: l.y, ok: true, c: c, hp: c.HP}
	if k == CreatureMammothKing { // its herd grazes round it
		for _, o := range [][2]float64{{-3.2, 1.6}, {3.0, 2.2}, {0.5, -3.4}} {
			if x, y := l.x+o[0], l.y+o[1]; w.creatureFits(CreatureMammoth, x, y) {
				m := w.addCreature(CreatureMammoth, x, y)
				m.homeX, m.homeY = l.x, l.y
			}
		}
	}
}

// BossSetLair moves boss k's lair to (x, y) and raises it there afresh, for
// tests and tools that build a scene by hand. It reports whether k is a
// boss that fits there.
func (w *World) BossSetLair(k CreatureKind, x, y float64) bool {
	i, ok := BossIndex(k)
	if !ok || !w.creatureFits(k, x, y) {
		return false
	}
	if !w.bosses.placed {
		w.placeBosses()
	}
	l := &w.bosses.lairs[i]
	if l.c != nil {
		l.c.gone = true
		w.fauna.creatures = slices.DeleteFunc(w.fauna.creatures, func(c *Creature) bool { return c == l.c })
	}
	l.x, l.y, l.ok, l.defeated = x, y, true, false
	w.raiseBoss(i)
	w.bosses.won = 0
	return true
}

// bossLairSite finds where boss k lives: deep inside its kind of land, far
// from where players start and from the villages. Where no place is that
// good (a small world, a crowded island) it settles for less, and only a
// world with none of its land at all has no lair for it.
func (w *World) bossLairSite(k CreatureKind) (x, y float64, ok bool) {
	H := w.SurfaceHeight()
	size := float64(max(w.width, H))
	step := 2
	if w.width*H > 600*600 {
		step = 3
	}
	var offshore []uint8
	if k == CreatureMegalodon {
		offshore = w.coastDistance(60)
	}
	px, py, pyramid := w.Pyramid()
	for _, lv := range []struct {
		far, village, need float64 // tiles from the spawn and from a village; share of the usual land round it
		sea                int     // the shark: tiles of sea at least between it and the nearest land
	}{
		{size / 5, 14, 1, 12},
		{size / 8, 7, 0.6, 7},
		{min(24, size/8), 0, 0.25, 4},
	} {
		best := math.Inf(-1)
		for ty := 6; ty < H-6; ty += step {
			for tx := 6; tx < w.width-6; tx += step {
				cx, cy := float64(tx)+0.5, float64(ty)+0.5
				if math.Hypot(cx-w.spawnX, cy-w.spawnY) < lv.far {
					continue
				}
				if k == CreatureCarcharo && pyramid && lv.village > 0 && math.Hypot(float64(tx-px), float64(ty-py)) < 16 {
					continue // the scorpions keep the pyramid
				}
				score, good := w.lairScore(k, tx, ty, offshore, lv.need, lv.sea)
				if !good || (lv.village > 0 && w.nearVillage(cx, cy, lv.village)) || !w.creatureFits(k, cx, cy) {
					continue
				}
				score += unit(hash2(int32(tx), int32(ty), uint32(w.seed)^(0xb055+uint32(k)))) // ties: somewhere
				if score > best {
					best, x, y, ok = score, cx, cy, true
				}
			}
		}
		if ok {
			return x, y, true
		}
	}
	return 0, 0, false
}

// lairScore is how well tile (tx, ty) suits boss k's lair, and whether it
// can be its lair at all: need is the share of the usual amount of its land
// round it that will do, sea the open water the shark wants round it.
func (w *World) lairScore(k CreatureKind, tx, ty int, offshore []uint8, need float64, sea int) (float64, bool) {
	t := w.TileAt(tx, ty)
	count := func(want ...Tile) (n, sand int) {
		for y := ty - 6; y <= ty+6; y += 2 {
			for x := tx - 8; x <= tx+8; x += 2 {
				switch u := w.TileAt(x, y); {
				case slices.Contains(want, u):
					n++
				case u == TileSand:
					sand++
				}
			}
		}
		return n, sand
	}
	enough := func(n int, usual float64) bool { return float64(n) >= usual*need }
	switch k {
	case CreatureTitanoboa:
		n, _ := count(TileForest)
		return float64(n), t == TileForest && enough(n, 30)
	case CreatureSpino: // in a pond of the marsh, not the sea
		marsh, sand := count(TileMarsh)
		water, _ := count(TileWater)
		return float64(marsh) + 0.5*float64(water), t == TileWater && sand == 0 && enough(marsh, 20) && water >= 3
	case CreatureMammothKing:
		n, _ := count(TileSnow)
		return float64(n), t == TileSnow && enough(n, 30)
	case CreatureCarcharo:
		n, _ := count(TileDesert)
		return float64(n), t == TileDesert && enough(n, 30)
	case CreatureGiganoto:
		n, _ := count(TileAsh)
		lava, _ := count(TileLava)
		return float64(n) + 0.5*float64(lava), t == TileAsh && enough(n, 25)
	case CreatureMegalodon:
		// Out in the deep, but not off the edge of the world where nobody sails.
		o := int(offshore[ty*w.width+tx])
		inside := min(tx, ty, w.width-1-tx, w.SurfaceHeight()-1-ty) >= 16
		return 20 - math.Abs(float64(o)-14), t == TileWater && o >= sea && inside
	}
	return 0, false
}

// nearVillage reports whether (x, y) is within d tiles of a village.
func (w *World) nearVillage(x, y, d float64) bool {
	for _, v := range w.villages {
		if math.Hypot(v.X-x, v.Y-y) < d {
			return true
		}
	}
	return false
}

// stepBosses keeps the world's books on its bosses, once a tick after the
// creatures have stepped: a boss that has fallen, and who has seen what.
func (w *World) stepBosses() {
	b := &w.bosses
	if !b.placed {
		w.placeBosses()
	}
	for i := range b.lairs {
		l := &b.lairs[i]
		switch {
		case l.c != nil && l.c.gone && l.c.HP <= 0:
			w.bossFell(i)
		case l.c != nil && l.c.gone: // taken off the map some other way: it comes back
			l.c = nil
		case l.c == nil && l.ok && !l.defeated:
			if _, d := w.nearestPlayer(l.x, l.y); d > creatureSpawnAway && w.creatureFits(BossKind(i), l.x, l.y) {
				w.raiseBoss(i)
			}
		}
	}
	if w.tick%10 == 0 {
		w.noteBossesSeen()
	}
}

// bossFell is boss i's death: its legendary item falls where it died, and
// if it was the last, the game is won.
func (w *World) bossFell(i int) {
	b := &w.bosses
	l := &b.lairs[i]
	c := l.c
	w.bossLetGo(c, l)
	l.c, l.defeated, l.awake = nil, true, false
	w.dropAt(c.X, c.Y, []Ingredient{{Item: LegendaryItem(c.Kind), Count: 1}})
	for _, o := range b.lairs {
		if o.ok && !o.defeated {
			return
		}
	}
	b.won = max(w.tick, 1)
}

// noteBossesSeen marks the lairs each player is close to as seen.
func (w *World) noteBossesSeen() {
	b := &w.bosses
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead {
			continue
		}
		for i, l := range b.lairs {
			if !l.ok {
				continue
			}
			near := math.Hypot(l.x-p.X, l.y-p.Y) <= bossSeeRange
			if c := l.c; c != nil && math.Hypot(c.X-p.X, c.Y-p.Y) <= bossSeeRange {
				near = true
			}
			if near {
				if b.seen == nil {
					b.seen = make(map[string]uint8)
				}
				b.seen[p.Name] |= 1 << i
			}
		}
	}
}

// --- what players are told --------------------------------------------------

// BossStatus is a boss as players know it.
type BossStatus struct {
	Kind         CreatureKind
	ID           CreatureID // 0 once it is dead
	X, Y         float64    // where it is; its lair once it is dead
	LairX, LairY float64
	HP, MaxHP    int
	Awake        bool // in a fight
	Enraged      bool
	Defeated     bool
	Holds        PlayerID // the player in its coils, if any
}

func (w *World) bossStatus(i int) BossStatus {
	l := &w.bosses.lairs[i]
	k := BossKind(i)
	s := BossStatus{Kind: k, X: l.x, Y: l.y, LairX: l.x, LairY: l.y, MaxHP: k.MaxHP(), Defeated: l.defeated}
	if c := l.c; c != nil && !c.gone {
		s.ID, s.X, s.Y, s.HP = c.ID, c.X, c.Y, c.HP
		s.Awake, s.Enraged, s.Holds = l.awake, l.enraged, l.held
	}
	return s
}

// Bosses returns every boss this world has a lair for, alive or dead.
func (w *World) Bosses() []BossStatus {
	var out []BossStatus
	for i, l := range w.bosses.lairs {
		if l.ok {
			out = append(out, w.bossStatus(i))
		}
	}
	return out
}

// BossFight returns the awake boss closest to player id within
// BossBarRange: the one whose health bar they see. ok is false if none.
func (w *World) BossFight(id PlayerID) (s BossStatus, ok bool) {
	p := w.players[id]
	if p == nil {
		return BossStatus{}, false
	}
	best := BossBarRange
	for i, l := range w.bosses.lairs {
		c := l.c
		if c == nil || c.gone || !l.awake {
			continue
		}
		if d := math.Hypot(c.X-p.X, c.Y-p.Y); d <= best {
			s, ok, best = w.bossStatus(i), true, d
		}
	}
	return s, ok
}

// BossesSeenBy returns the bosses player id has seen, or seen the lair of:
// what their map marks.
func (w *World) BossesSeenBy(id PlayerID) []BossStatus {
	p := w.players[id]
	if p == nil {
		return nil
	}
	bits := w.bosses.seen[p.Name]
	var out []BossStatus
	for i, l := range w.bosses.lairs {
		if l.ok && bits&(1<<i) != 0 {
			out = append(out, w.bossStatus(i))
		}
	}
	return out
}

// Victory reports whether player id is to be shown that the game is beaten:
// every boss has fallen and they have not closed the screen yet. day is the
// day the last one fell, and beaten the bosses that fell.
func (w *World) Victory(id PlayerID) (show bool, day int, beaten []CreatureKind) {
	b := &w.bosses
	p := w.players[id]
	if p == nil || b.won == 0 {
		return false, 0, nil
	}
	for i, l := range b.lairs {
		if l.ok && l.defeated {
			beaten = append(beaten, BossKind(i))
		}
	}
	return !b.shown[p.Name], DayOf(b.won), beaten
}

// victorySeen marks that p has closed the victory screen.
func (w *World) victorySeen(p *Player) {
	b := &w.bosses
	if b.won == 0 {
		return
	}
	if b.shown == nil {
		b.shown = make(map[string]bool)
	}
	b.shown[p.Name] = true
}

// --- saving -----------------------------------------------------------------

// BossSave is what a save keeps of the bosses: which have fallen, who has
// seen which lair, and the victory. Where the lairs are is found again from
// the terrain.
type BossSave struct {
	Defeated []CreatureKind   `json:",omitempty"`
	Seen     map[string]uint8 `json:",omitempty"` // a bit per boss, by player name
	Won      uint64           `json:",omitempty"` // the tick the last one fell
	Shown    []string         `json:",omitempty"` // players who have closed the victory screen
}

func (w *World) saveBosses() *BossSave {
	b := &w.bosses
	s := &BossSave{Won: b.won}
	for i, l := range b.lairs {
		if l.defeated {
			s.Defeated = append(s.Defeated, BossKind(i))
		}
	}
	if len(b.seen) > 0 {
		s.Seen = make(map[string]uint8, len(b.seen))
		for name, bits := range b.seen {
			s.Seen[name] = bits
		}
	}
	for name := range b.shown {
		s.Shown = append(s.Shown, name)
	}
	slices.Sort(s.Shown)
	if s.Won == 0 && len(s.Defeated) == 0 && len(s.Seen) == 0 && len(s.Shown) == 0 {
		return nil
	}
	return s
}

// loadBosses applies a save to the freshly raised bosses: the fallen are
// taken off the map for good. A save from before the bosses has none, so
// they all stand.
func (w *World) loadBosses(s *BossSave) {
	b := &w.bosses
	if !b.placed {
		w.placeBosses()
	}
	if s == nil {
		return
	}
	for _, k := range s.Defeated {
		i, ok := BossIndex(k)
		if !ok {
			continue
		}
		l := &b.lairs[i]
		if l.c != nil {
			c := l.c
			w.fauna.creatures = slices.DeleteFunc(w.fauna.creatures, func(o *Creature) bool { return o == c })
			l.c = nil
		}
		l.defeated = true
	}
	for name, bits := range s.Seen {
		if name = cleanName(name); name != "" {
			if b.seen == nil {
				b.seen = make(map[string]uint8)
			}
			b.seen[name] = bits & (1<<NumBosses - 1)
		}
	}
	for _, name := range s.Shown {
		if name = cleanName(name); name != "" {
			if b.shown == nil {
				b.shown = make(map[string]bool)
			}
			b.shown[name] = true
		}
	}
	b.won = 0
	lairs, standing := 0, 0
	for _, l := range b.lairs {
		if l.ok {
			lairs++
			if !l.defeated {
				standing++
			}
		}
	}
	if lairs > 0 && standing == 0 {
		b.won = max(s.Won, 1)
	}
}
