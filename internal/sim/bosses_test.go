package sim

import (
	"fmt"
	"math"
	"testing"
)

// The bosses' tiles: what land each lives on.
var bossLand = [NumBosses]Tile{TileForest, TileWater, TileSnow, TileDesert, TileAsh, TileWater}

func TestEveryArchipelagoHasItsSixBosses(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{1, SizeMedium}, {2, SizeMedium}, {7, SizeLarge}} {
		w := genSeas(t, c.seed, c.size)
		bs := w.Bosses()
		if len(bs) != NumBosses {
			t.Fatalf("seed %d: %d bosses, want %d", c.seed, len(bs), NumBosses)
		}
		far := float64(max(w.width, w.SurfaceHeight())) / 5
		for i, b := range bs {
			k := BossKind(i)
			if b.Kind != k || b.Defeated || b.ID == 0 {
				t.Errorf("seed %d: boss %d is %+v", c.seed, i, b)
				continue
			}
			cr := w.creatureByID(b.ID)
			if cr == nil || cr.Kind != k {
				t.Errorf("seed %d: %s is not in the world", c.seed, k.Name())
				continue
			}
			if tile := w.TileAt(int(b.LairX), int(b.LairY)); tile != bossLand[i] {
				t.Errorf("seed %d: %s's lair is on tile %d, want %d", c.seed, k.Name(), tile, bossLand[i])
			}
			if !w.creatureFits(k, cr.X, cr.Y) {
				t.Errorf("seed %d: %s does not fit at its lair (%.1f, %.1f)", c.seed, k.Name(), cr.X, cr.Y)
			}
			if d := math.Hypot(b.LairX-w.spawnX, b.LairY-w.spawnY); d < far {
				t.Errorf("seed %d: %s only %.0f tiles from the spawn", c.seed, k.Name(), d)
			}
			t.Logf("seed %d size %d: %s at (%.0f, %.0f), %.0f tiles out", c.seed, c.size, k.Name(), b.LairX, b.LairY, math.Hypot(b.LairX-w.spawnX, b.LairY-w.spawnY))
		}
		// The same seed raises the same bosses in the same places.
		again := genSeas(t, c.seed, c.size)
		for i, b := range again.Bosses() {
			if b.LairX != bs[i].LairX || b.LairY != bs[i].LairY {
				t.Errorf("seed %d: %s's lair moved between two generations", c.seed, b.Kind.Name())
			}
		}
	}
	island, err := Generate(GenConfig{Seed: 1, Width: 96, Height: 96})
	if err != nil {
		t.Fatal(err)
	}
	stepN(island, 5)
	if n := len(island.Bosses()); n != 0 {
		t.Errorf("%d bosses on the old island", n)
	}
}

func TestBossesStayHomeAfterAWhile(t *testing.T) {
	w := genSeas(t, 2, SizeMedium)
	stepN(w, 60*TicksPerSecond)
	for _, b := range w.Bosses() {
		c := w.creatureByID(b.ID)
		if c == nil {
			t.Errorf("%s is gone", b.Kind.Name())
			continue
		}
		if !w.creatureFits(c.Kind, c.X, c.Y) {
			t.Errorf("%s wandered to (%.1f, %.1f), outside its habitat", c.Kind.Name(), c.X, c.Y)
		}
		if d := math.Hypot(c.X-b.LairX, c.Y-b.LairY); d > c.Kind.def().leash+2 {
			t.Errorf("%s wandered %.1f tiles from its lair", c.Kind.Name(), d)
		}
	}
}

// --- an arena for one boss ------------------------------------------------------

// arenaSize is the side of a boss arena, in tiles.
const arenaSize = 72

// calmSeed is a world seed whose first few minutes bring no storm or
// whirlwind, so lightning does not stray into a fight being measured.
func calmSeed() uint64 {
	for seed := uint64(1); ; seed++ {
		calm := true
		for tick := uint64(StartTick); tick < StartTick+8*60*TicksPerSecond; tick += WeatherTicks / 2 {
			if wt := WeatherAt(seed, tick); wt == WeatherStorm || wt == WeatherTornado {
				calm = false
			}
		}
		if calm {
			return seed
		}
	}
}

// bossArena is a world of boss k's own land with its lair in the middle,
// and a player standing well off it with the best gear short of the
// legendary: an obsidian knife in hand, tyrant-tooth armour and a fur coat
// in the bag, and medicine. On the sea, they are in a sailing boat.
func bossArena(t *testing.T, k CreatureKind, potions, bigPotions int) (*World, *Player, *bossLair) {
	t.Helper()
	i, _ := BossIndex(k)
	w := newWorld(arenaSize, arenaSize)
	w.seed, w.style = calmSeed(), StyleSeas
	w.bosses.placed = true
	mid := float64(arenaSize) / 2
	for ty := range arenaSize {
		for tx := range arenaSize {
			tile := bossLand[i]
			if k == CreatureSpino { // a marsh, with a pond in the middle
				tile = TileMarsh
				if math.Hypot(float64(tx)+0.5-mid, float64(ty)+0.5-mid) < 3.6 {
					tile = TileWater
				}
			}
			w.tiles[ty*arenaSize+tx] = tile
		}
	}
	lx, ly := mid, mid
	w.spawnX, w.spawnY = mid-12, mid
	if !w.BossSetLair(k, lx, ly) {
		t.Fatalf("%s does not fit at its lair", k.Name())
	}
	p := w.players[w.AddPlayer("Thử")]
	p.Hunger = MaxHunger
	p.Inv = Inventory{}
	p.Inv[0] = Slot{Item: ItemDiamondSword, Count: 1}
	p.Inv.Add(ItemDiamondArmor, 1)
	p.Inv.Add(ItemFurCoat, 1)
	if potions > 0 {
		p.Inv.Add(ItemPotion, potions)
	}
	if bigPotions > 0 {
		p.Inv.Add(ItemBigPotion, bigPotions)
	}
	p.Selected = 0
	if k == CreatureMegalodon {
		w.nextBoat++
		b := &Boat{ID: w.nextBoat, Kind: BoatSail, X: p.X, Y: p.Y, HP: BoatSail.MaxHP(), Rider: p.ID}
		w.boats = append(w.boats, b)
		p.boat = b.ID
	}
	return w, p, &w.bosses.lairs[i]
}

// fightLog is how a scripted fight went.
type fightLog struct {
	won, died     bool
	secs          float64 // from the boss waking to its fall
	taken         float64 // health the player lost
	potions       int     // medicine drunk
	boatLeft      int     // hit points left on the boat, at sea
	enragedAt     float64 // seconds into the fight it was enraged; 0 if never
	strikes, held int     // blows it began, and times it seized the player
}

// fightBoss has p fight the boss of l to the end, or for at most limit
// ticks. The player goes for the boss and strikes whenever it is in reach,
// drinks medicine below half health, struggles in the coils, steps back
// from every special move it sees coming and, with dodge, from every other
// bite, and waits at the water's edge when it hides under the water.
func fightBoss(w *World, p *Player, l *bossLair, dodge bool, limit int) fightLog {
	return fightBossTraced(w, p, l, dodge, limit, nil)
}

// fightBossTraced is fightBoss, telling tr how the fight stands every second.
func fightBossTraced(w *World, p *Player, l *bossLair, dodge bool, limit int, tr func(string, ...any)) fightLog {
	var log fightLog
	start := -1
	wasStrike, wasHeld := false, false
	end := limit
	dodgeThis := false
	for tick := range limit {
		c := l.c
		if l.defeated || p.Dead {
			log.won, log.died, end = l.defeated, p.Dead, tick
			break
		}
		if start < 0 && l.awake {
			start = tick
		}
		if l.enraged && log.enragedAt == 0 && start >= 0 {
			log.enragedAt = float64(tick-start) / TicksPerSecond
		}
		if p.Health < 50 {
			for s, slot := range p.Inv {
				if slot.Item == ItemBigPotion || slot.Item == ItemPotion {
					w.Apply(p.ID, Command{Kind: CmdEat, Slot: uint8(s)})
					log.potions++
					break
				}
			}
		}
		in := Input{Action: true}
		dx, dy := c.X-p.X, c.Y-p.Y
		dist := math.Hypot(dx, dy)
		strike := c.State == CreatureStrike
		if strike && !wasStrike {
			log.strikes++
			dodgeThis = dodge && log.strikes%2 == 0
		}
		wasStrike = strike
		away := func() { // straight back, or aside, but not into the water
			for _, d := range [][2]float64{{-dx, -dy}, {-dy, dx}, {dy, -dx}} {
				if l := math.Hypot(d[0], d[1]); l > 0 && w.standable(p.X+d[0]/l*0.6, p.Y+d[1]/l*0.6) {
					in.MoveX, in.MoveY = sign(d[0]), sign(d[1])
					return
				}
			}
		}
		switch {
		case p.boat != 0: // at sea the boat keeps where it is; the player strikes from it
		case l.held == p.ID:
			in.MoveX = 1 // wriggling
		case c.State == CreatureSpecial && dist < 6:
			away()
		case strike && dodgeThis && c.timer > bossRecover && dist < bossTuneOf(c.Kind).reach+1.2:
			away()
		case c.State.OutOfReach(): // under the water: wait at its edge
			ex, ey := l.x-p.X, l.y-p.Y
			if d := math.Hypot(ex, ey); d > 4.5 {
				in.MoveX, in.MoveY = sign(ex), sign(ey)
			} else if d < 3.8 {
				in.MoveX, in.MoveY = sign(-ex), sign(-ey)
			}
		case dist > c.Kind.Reach()*0.8:
			in.MoveX, in.MoveY = sign(dx), sign(dy)
		}
		if held := l.held == p.ID; held && !wasHeld {
			log.held++
			wasHeld = true
		} else if !held {
			wasHeld = false
		}
		if tr != nil && tick%TicksPerSecond == 0 {
			tr("%3ds st=%2d hp=%5d dist=%4.1f me=%3.0f at (%.1f,%.1f) it (%.1f,%.1f) in %+v", tick/TicksPerSecond, c.State, c.HP, dist, p.Health, p.X, p.Y, c.X, c.Y, in)
		}
		w.SetInput(p.ID, in)
		hp := p.Health
		w.Step()
		log.taken += max(0, hp-p.Health)
		if tick == limit-1 {
			log.died = p.Dead
		}
	}
	if start >= 0 {
		log.secs = float64(end-start) / TicksPerSecond
	}
	if b := w.boatByID(p.boat); b != nil {
		log.boatLeft = b.HP
	}
	return log
}

func (l fightLog) String() string {
	return fmt.Sprintf("won %v died %v: %.0fs, lost %.0f health, %d medicines, %d blows begun, held %d times, enraged at %.0fs, boat %d",
		l.won, l.died, l.secs, l.taken, l.potions, l.strikes, l.held, l.enragedAt, l.boatLeft)
}
