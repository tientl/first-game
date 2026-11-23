package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// Creatures are the island's animals. Unlike entities they move freely in
// tile units and do not occupy a tile. They are not saved: a loaded world is
// repopulated from its seed, which nobody can tell apart from animals that
// wandered off.
//
//   - Rabbits graze on meadows and in the woods. They bolt from a player who
//     comes close, but tire after a short sprint, and sleep at night.
//   - Snakes lurk on rocks and beaches, go for a player who comes close and
//     bite. They are slower than a player, so walking away works.
//   - Worms crawl about on grass. They are bait.
//   - Fish swim in the water along the shore and are caught with a worm.

// CreatureID identifies a creature. It is never reused within a world.
type CreatureID uint32

// CreatureKind is the species of a creature.
type CreatureKind uint8

const (
	CreatureNone CreatureKind = iota
	CreatureRabbit
	CreatureSnake
	CreatureWorm
	CreatureFish
	CreatureCrab // see crab.go
	CreatureCroc // a crocodile in the marsh; see croc.go
	NumCreatureKinds
)

// CreatureState is what a creature is doing; the client animates by it.
type CreatureState uint8

const (
	CreatureIdle   CreatureState = iota
	CreatureWander               // walking somewhere at its own pace
	CreatureFlee                 // a rabbit bolting from a player
	CreatureRest                 // a rabbit out of breath after bolting
	CreatureSleep                // a rabbit asleep at night
	CreatureChase                // a snake going for a player
	CreatureStrike               // a snake biting
	CreatureHurt                 // flinching from a hit
	CreatureHooked               // a fish under someone's float; see fishing.go
	CreatureHidden               // down its burrow or den: out of sight and reach; see homes.go
	CreatureLurk                 // a crocodile in the water, only its eyes showing; out of reach
)

// Creature tuning. Speeds are in tiles per tick.
const (
	RabbitFleeDist   = 3.0  // an awake rabbit bolts from a player this close
	RabbitFleeTicks  = 25   // how long it sprints
	RabbitRestTicks  = 24   // then it must catch its breath: the moment to grab it
	RabbitSleepBelow = 0.15 // ambient light below which rabbits sleep

	SnakeAggroDist    = 3.5 // a snake goes for a player this close
	SnakeGiveUpDist   = 7.0 // and gives up once the player is this far
	SnakeBiteRange    = 1.1
	SnakeBiteDamage   = 10.0
	SnakeBiteCooldown = 3 * TicksPerSecond / 2
	SnakeStrikeTicks  = 6
	CreatureHurtTicks = 6

	CreatureRespawnTicks = 10 * TicksPerSecond // how often the world tops up its animals
	creatureSpawnAway    = 14.0                // new animals appear at least this far from every player
	snakeSpawnClear      = 12                  // tiles around the spawn kept free of snakes at worldgen
	creatureMinPerKind   = 2                   // even a small island gets a few of each
)

type creatureDef struct {
	name       string
	hp         int
	walk, run  float64 // tiles per tick
	radius     float64 // half-size of its collision box
	reach      float64 // a player's Action reaches it from this far (centre to centre)
	yield      ItemKind
	yieldN     int
	bait       ItemKind   // used up to catch it; ItemNone if none is needed
	tool       ItemKind   // must be held to catch it; ItemNone if bare hands do
	water      bool       // swims in the shallows instead of walking on land
	amphibious bool       // walks on land and wades in the shallows alike
	seaside    bool       // appears only on sand by the water
	waterside  bool       // appears only on its spawn tiles where they touch water
	onLine     bool       // caught on a fishing line: cast, wait for the bite, strike
	homeKind   EntityKind // lives in a burrow or den of this kind; EntityNone if it roams free
	perHome    int        // animals to a home
	leash      float64    // wanders back home once this far away
	idle, roam [2]int     // min and max ticks of an idle spell and a walk
	spawnOn    []Tile
	tilesPer   int // one of this kind per this many spawn tiles
}

var creatureDefs = [NumCreatureKinds]creatureDef{
	CreatureNone: {name: "?"},
	CreatureRabbit: {
		name: "Thỏ", hp: 1,
		walk: 1.2 / TicksPerSecond, run: 5.2 / TicksPerSecond,
		radius: 0.2, reach: 1.4,
		yield: ItemMeat, yieldN: 2,
		homeKind: EntityBurrow, perHome: 3,
		leash: 10, idle: [2]int{20, 80}, roam: [2]int{15, 50},
		spawnOn: []Tile{TileGrass, TileForest}, tilesPer: 140,
	},
	CreatureSnake: {
		name: "Rắn", hp: 3,
		walk: 0.8 / TicksPerSecond, run: 2.6 / TicksPerSecond,
		radius: 0.25, reach: 1.4,
		yield: ItemMeat, yieldN: 1,
		homeKind: EntityDen, perHome: 2,
		leash: 7, idle: [2]int{30, 120}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileRocky, TileSand, TileDesert, TileAsh}, tilesPer: 160,
	},
	CreatureWorm: {
		name: "Giun", hp: 1,
		walk:   0.35 / TicksPerSecond,
		radius: 0.15, reach: 1.4,
		yield: ItemWorm, yieldN: 1,
		leash: 3, idle: [2]int{40, 160}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileGrass, TileForest, TileMarsh}, tilesPer: 150,
	},
	CreatureFish: {
		name: "Cá", hp: 1,
		walk:   1.1 / TicksPerSecond,
		radius: 0.25, reach: 2.2, // cast from the shore into the water
		yield: ItemFish, yieldN: 1, bait: ItemWorm, tool: ItemRod,
		water: true, onLine: true,
		leash: 8, idle: [2]int{10, 60}, roam: [2]int{15, 50},
		spawnOn: []Tile{TileWater}, tilesPer: 40,
	},
	CreatureCrab: {
		name: "Cua", hp: 1,
		walk: 0.9 / TicksPerSecond, run: 3.6 / TicksPerSecond, // a little slower than a player
		radius: 0.2, reach: 1.4,
		yield: ItemCrabMeat, yieldN: 1,
		amphibious: true, seaside: true,
		leash: 3, idle: [2]int{20, 90}, roam: [2]int{10, 40},
		spawnOn: []Tile{TileSand}, tilesPer: 30,
	},
	CreatureCroc: {
		name: "Cá sấu", hp: 6,
		walk: 0.7 / TicksPerSecond, run: 2.4 / TicksPerSecond, // slow on land: a player outruns it
		radius: 0.35, reach: 1.6,
		yield: ItemMeat, yieldN: 3,
		amphibious: true, waterside: true,
		leash: 5, idle: [2]int{60, 200}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileMarsh}, tilesPer: 220,
	},
}

func (k CreatureKind) def() *creatureDef {
	if k >= NumCreatureKinds {
		return &creatureDefs[CreatureNone]
	}
	return &creatureDefs[k]
}

// Name is the display name of k.
func (k CreatureKind) Name() string { return k.def().name }

// Reach is how far, centre to centre, a player's Action reaches a creature of
// kind k.
func (k CreatureKind) Reach() float64 { return k.def().reach }

// Bait is the item used up to catch a creature of kind k, or ItemNone.
func (k CreatureKind) Bait() ItemKind { return k.def().bait }

// Tool is the item that must be held to catch a creature of kind k, or
// ItemNone.
func (k CreatureKind) Tool() ItemKind { return k.def().tool }

// Creature is one animal.
type Creature struct {
	ID       CreatureID
	Kind     CreatureKind
	X, Y     float64 // centre, in tiles
	FaceLeft bool    // creatures are drawn side-on
	HP       int
	State    CreatureState

	homeX, homeY float64
	hx, hy       float64 // heading while walking or fleeing, a unit vector
	timer        int     // ticks left in State
	biteCD       int
	hookedBy     PlayerID // whose float it is under, while CreatureHooked
	home         EntityID // its burrow or den; 0 if none
	goalX, goalY float64  // where a fleeing crab is making for
	gone         bool     // caught; removed at the end of the tick
}

// fauna is the world's animal life.
type fauna struct {
	creatures []*Creature // ascending ID; creatures step in this order
	nextID    CreatureID
	rng       *rand.Rand
	shallow   []bool                    // per tile: water along the shore, where fish swim
	spawnAt   [NumCreatureKinds][]int32 // per kind: tile indices where it may appear
	target    [NumCreatureKinds]int     // per kind: the population the world tops up to
	homes     []homeRec                 // burrows and dens, in the order they were dug
	seated    bool                      // homes checked after loading; see reseatHomes
}

func (w *World) random() *rand.Rand {
	if w.fauna.rng == nil {
		w.fauna.rng = rand.New(rand.NewPCG(w.seed, 0xfa0a))
	}
	return w.fauna.rng
}

// isShallow reports whether fish may swim on tile (tx, ty): water with land
// next to it, so a player on the shore can reach them.
func (w *World) isShallow(tx, ty int) bool {
	if w.fauna.shallow == nil {
		w.fauna.shallow = make([]bool, w.width*w.height)
		for y := 0; y < w.height; y++ {
			for x := 0; x < w.width; x++ {
				if w.tiles[y*w.width+x] != TileWater {
					continue
				}
				for dy := -1; dy <= 1 && !w.fauna.shallow[y*w.width+x]; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if w.inBounds(x+dx, y+dy) && w.TileAt(x+dx, y+dy).Walkable() {
							w.fauna.shallow[y*w.width+x] = true
							break
						}
					}
				}
			}
		}
	}
	return w.inBounds(tx, ty) && w.fauna.shallow[ty*w.width+tx]
}

// creatureFits reports whether a creature of kind k can be centred at (x, y).
func (w *World) creatureFits(k CreatureKind, x, y float64) bool {
	d := k.def()
	r := d.radius
	for _, c := range [4][2]float64{{-r, -r}, {r, -r}, {-r, r}, {r, r}} {
		tx, ty := int(math.Floor(x+c[0])), int(math.Floor(y+c[1]))
		land := w.TileAt(tx, ty).Walkable() && !w.blocked(tx, ty)
		switch {
		case d.water && !w.isShallow(tx, ty):
			return false
		case d.amphibious && !land && !w.isShallow(tx, ty):
			return false
		case !d.water && !d.amphibious && !land:
			return false
		}
	}
	return true
}

// addCreature puts a new creature of kind k at (x, y). The caller must check
// creatureFits first.
func (w *World) addCreature(k CreatureKind, x, y float64) *Creature {
	f := &w.fauna
	f.nextID++
	c := &Creature{ID: f.nextID, Kind: k, X: x, Y: y, HP: k.def().hp, homeX: x, homeY: y}
	c.FaceLeft = w.random().IntN(2) == 0
	f.creatures = append(f.creatures, c) // ids only grow, so this stays sorted
	return c
}

// spawnCreatures fills a freshly generated world with animals and records
// where each kind may reappear later.
func (w *World) spawnCreatures() {
	f := &w.fauna
	sx, sy := int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY))
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		d := k.def()
		var spots []int32
		for ty := 0; ty < w.height; ty++ {
			for tx := 0; tx < w.width; tx++ {
				t := w.tiles[ty*w.width+tx]
				if !slices.Contains(d.spawnOn, t) || (d.water && !w.isShallow(tx, ty)) || (d.seaside && !w.seaside(tx, ty)) || (d.waterside && !w.waterside(tx, ty)) {
					continue
				}
				if !w.creatureFits(k, float64(tx)+0.5, float64(ty)+0.5) {
					continue
				}
				spots = append(spots, int32(ty*w.width+tx))
			}
		}
		f.spawnAt[k] = spots
		if len(spots) == 0 {
			continue
		}
		f.target[k] = max(creatureMinPerKind, len(spots)/d.tilesPer)
		if d.homeKind != EntityNone {
			// Families around homes, none too close to where players start.
			ok := func(tx, ty int) bool {
				away := clearSpawnRadius + 2
				if k == CreatureSnake {
					away = snakeSpawnClear + 2
				}
				return max(abs(tx-sx), abs(ty-sy)) >= away
			}
			homes := (f.target[k] + d.perHome - 1) / d.perHome
			if dug := w.placeHomes(k, spots, homes, ok); dug > 0 {
				f.target[k] = max(f.target[k], dug*d.perHome)
				continue
			}
		}
		for range f.target[k] {
			i := spots[w.random().IntN(len(spots))]
			tx, ty := int(i)%w.width, int(i)/w.width
			if k == CreatureSnake && max(abs(tx-sx), abs(ty-sy)) < snakeSpawnClear {
				continue // the island is kind for the first few minutes
			}
			w.addCreature(k, float64(tx)+0.5, float64(ty)+0.5)
		}
	}
}

// repopulate brings back at most one animal of each kind that is below its
// target, somewhere no player is looking.
func (w *World) repopulate() {
	f := &w.fauna
	var count [NumCreatureKinds]int
	for _, c := range f.creatures {
		count[c.Kind]++
	}
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		spots := f.spawnAt[k]
		if count[k] >= f.target[k] || len(spots) == 0 {
			continue
		}
		if k.def().homeKind != EntityNone && w.repopulateAtHome(k) {
			continue // out of a burrow or den
		}
		for range 8 {
			i := spots[w.random().IntN(len(spots))]
			x, y := float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5
			if _, d := w.nearestPlayer(x, y); d < creatureSpawnAway {
				continue
			}
			w.addCreature(k, x, y)
			break
		}
	}
}

// Creatures returns copies of every creature, in id order.
func (w *World) Creatures() []Creature {
	out := make([]Creature, 0, len(w.fauna.creatures))
	for _, c := range w.fauna.creatures {
		out = append(out, *c)
	}
	return out
}

// EachCreatureIn calls fn with a copy of every creature whose centre lies in
// the rectangle [x0, x1] x [y0, y1], in id order.
func (w *World) EachCreatureIn(x0, y0, x1, y1 float64, fn func(Creature)) {
	for _, c := range w.fauna.creatures {
		if c.X >= x0 && c.X <= x1 && c.Y >= y0 && c.Y <= y1 {
			fn(*c)
		}
	}
}

func (w *World) creatureByID(id CreatureID) *Creature {
	cs := w.fauna.creatures
	i, ok := slices.BinarySearchFunc(cs, id, func(c *Creature, id CreatureID) int { return int(c.ID) - int(id) })
	if !ok || cs[i].gone {
		return nil
	}
	return cs[i]
}

// nearestPlayer returns the closest living player to (x, y) and how far it
// is; nil and +Inf if there is none.
func (w *World) nearestPlayer(x, y float64) (*Player, float64) {
	var best *Player
	bestD := math.Inf(1)
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead {
			continue
		}
		if d := math.Hypot(p.X-x, p.Y-y); d < bestD {
			best, bestD = p, d
		}
	}
	return best, bestD
}

func (w *World) stepCreatures() {
	f := &w.fauna
	if !f.seated {
		w.reseatHomes() // a loaded world may have lost its homes
		f.seated = true
	}
	for _, c := range f.creatures {
		if !c.gone {
			w.stepCreature(c)
		}
	}
	f.creatures = slices.DeleteFunc(f.creatures, func(c *Creature) bool { return c.gone })
	if w.tick%CreatureRespawnTicks == 0 {
		w.repopulate()
	}
}

func (w *World) stepCreature(c *Creature) {
	if c.State == CreatureHooked {
		if !w.stillHooked(c) { // the angler left, died or fell asleep
			c.State, c.timer, c.hookedBy = CreatureIdle, 0, 0
		}
		return // holds still under the float
	}
	if c.State == CreatureHidden {
		w.stepHidden(c)
		return
	}
	if c.State == CreatureHurt {
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
		}
		return
	}
	switch c.Kind {
	case CreatureRabbit:
		w.stepRabbit(c)
	case CreatureSnake:
		w.stepSnake(c)
	case CreatureCrab:
		w.stepCrab(c)
	case CreatureCroc:
		w.stepCroc(c)
	default:
		w.wander(c)
	}
}

func (w *World) stepRabbit(c *Creature) {
	d := c.Kind.def()
	p, dist := w.nearestPlayer(c.X, c.Y)
	switch c.State {
	case CreatureFlee:
	case CreatureRest:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
		}
		return
	default:
		night := Ambient(w.tick) < RabbitSleepBelow
		switch {
		case night:
			c.State, c.timer = CreatureSleep, 0 // too sleepy to notice anyone
			return
		case p != nil && dist < RabbitFleeDist:
			c.State, c.timer = CreatureFlee, RabbitFleeTicks+1
			c.hx, c.hy = 0, 0
		default:
			if c.State == CreatureSleep {
				c.State, c.timer = CreatureIdle, 0
			}
			w.wander(c)
			return
		}
	}

	// Bolting.
	if c.timer--; c.timer <= 0 {
		c.State, c.timer = CreatureRest, RabbitRestTicks
		return
	}
	if p != nil {
		k := 0.35
		if c.hx == 0 && c.hy == 0 {
			k = 1 // the first leap goes straight away
		}
		if e := w.burrowToRun(c, p); e != nil { // for home, and down the hole
			hx, hy := homeCentre(e)
			if math.Hypot(hx-c.X, hy-c.Y) <= homeReach+d.run {
				c.goIn(e)
				return
			}
			c.steerAway(2*c.X-hx, 2*c.Y-hy, max(k, 0.6))
		} else {
			c.steerAway(p.X, p.Y, k)
		}
	}
	if !w.moveCreature(c, c.hx*d.run, c.hy*d.run) && p != nil {
		c.dodge(w, p.X, p.Y) // cornered against the water: break sideways
	}
}

func (w *World) stepSnake(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.State == CreatureStrike {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	if w.headHome(c) {
		return // nightfall: down into the den, chase or no chase
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	if p == nil || dist > SnakeGiveUpDist || (c.State != CreatureChase && dist > SnakeAggroDist) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		w.wander(c)
		return
	}
	c.State = CreatureChase
	dx, dy := p.X-c.X, p.Y-c.Y
	if dx != 0 {
		c.FaceLeft = dx < 0
	}
	if dist <= SnakeBiteRange {
		if c.biteCD == 0 {
			c.State, c.timer, c.biteCD = CreatureStrike, SnakeStrikeTicks, SnakeBiteCooldown
			struck(p, SnakeBiteDamage)
			p.notify(NoticeBitten, uint8(c.Kind))
		}
		return
	}
	step := min(d.run, dist-SnakeBiteRange*0.8)
	w.moveCreature(c, dx/dist*step, dy/dist*step)
}

// wander alternates idle spells and short walks in a random direction,
// heading back home when the creature has strayed too far.
func (w *World) wander(c *Creature) {
	d := c.Kind.def()
	r := w.random()
	if c.State != CreatureIdle && c.State != CreatureWander {
		c.State, c.timer = CreatureIdle, 0
	}
	if c.timer > 0 {
		c.timer--
	}
	if c.timer == 0 {
		if c.State == CreatureIdle && r.Float64() < 0.75 {
			c.State = CreatureWander
			c.timer = d.roam[0] + r.IntN(d.roam[1]-d.roam[0]+1)
			a := r.Float64() * 2 * math.Pi
			c.hx, c.hy = math.Cos(a), math.Sin(a)
			if hd := math.Hypot(c.homeX-c.X, c.homeY-c.Y); hd > d.leash {
				c.hx, c.hy = (c.homeX-c.X)/hd, (c.homeY-c.Y)/hd
			}
		} else {
			c.State = CreatureIdle
			c.timer = d.idle[0] + r.IntN(d.idle[1]-d.idle[0]+1)
		}
	}
	if c.State == CreatureWander && !w.moveCreature(c, c.hx*d.walk, c.hy*d.walk) {
		c.State, c.timer = CreatureIdle, 1 // bumped into the coast: think again
	}
}

// steerAway turns c's heading away from (x, y); k in (0, 1] is how sharply.
func (c *Creature) steerAway(x, y, k float64) {
	ax, ay := c.X-x, c.Y-y
	if l := math.Hypot(ax, ay); l > 1e-9 {
		ax, ay = ax/l, ay/l
	} else {
		ax, ay = 1, 0
	}
	hx, hy := c.hx+(ax-c.hx)*k, c.hy+(ay-c.hy)*k
	if l := math.Hypot(hx, hy); l > 1e-9 {
		c.hx, c.hy = hx/l, hy/l
	} else {
		c.hx, c.hy = ax, ay
	}
}

// dodge turns a blocked heading a quarter turn, to whichever side leads
// further from (x, y) and is open.
func (c *Creature) dodge(w *World, x, y float64) {
	best, bestD := [2]float64{}, -1.0
	for _, h := range [2][2]float64{{-c.hy, c.hx}, {c.hy, -c.hx}} {
		nx, ny := c.X+h[0]*0.5, c.Y+h[1]*0.5
		if !w.creatureFits(c.Kind, nx, ny) {
			continue
		}
		if d := math.Hypot(nx-x, ny-y); d > bestD {
			best, bestD = h, d
		}
	}
	if bestD >= 0 {
		c.hx, c.hy = best[0], best[1]
	}
}

// moveCreature moves c by (dx, dy), each axis on its own so it slides along
// coasts, and reports whether it moved at all.
func (w *World) moveCreature(c *Creature, dx, dy float64) bool {
	moved := false
	if dx != 0 && w.creatureFits(c.Kind, c.X+dx, c.Y) {
		c.X += dx
		c.FaceLeft = dx < 0
		moved = true
	}
	if dy != 0 && w.creatureFits(c.Kind, c.X, c.Y+dy) {
		c.Y += dy
		moved = true
	}
	return moved
}

// hurt takes health from p, killing it at zero. An animal's attack goes
// through struck (gear.go) instead, so armour stops part of it.
func hurt(p *Player, dmg float64) {
	p.wake()
	p.Health -= dmg
	if p.Health <= 0 {
		p.Health, p.Dead, p.Cold = 0, true, false
	}
}

// catch is one Action by p on c: a grab, a hit or a cast. It reports whether
// anything was done (false: refused, the player does not swing).
func (w *World) catch(p *Player, c *Creature) bool {
	d := c.Kind.def()
	if d.tool.Valid() && p.Held() != d.tool {
		p.notify(NoticeNeedTool, uint8(d.tool))
		return false
	}
	if d.bait.Valid() && p.Inv.Count(d.bait) == 0 {
		p.notify(NoticeNeedBait, uint8(d.bait))
		return false
	}
	if p.Inv.Room(d.yield) < d.yieldN {
		p.notify(NoticeBagFull, uint8(d.yield))
		return false
	}
	if d.onLine {
		w.cast(p, c) // see fishing.go
		return true
	}
	c.HP -= damageWith(p)
	if c.HP > 0 {
		c.State, c.timer = CreatureHurt, CreatureHurtTicks
		return true
	}
	if d.bait.Valid() {
		p.Inv.Take(d.bait, 1)
	}
	p.Inv.Add(d.yield, d.yieldN)
	c.gone = true
	return true
}

// CreatureCandidate is what targeting needs to know about a creature.
type CreatureCandidate struct {
	ID   CreatureID
	Kind CreatureKind
	X, Y float64
}

// Target is what a player's Action works on: an entity or a creature.
// Exactly one of the two ids is set.
type Target struct {
	Entity   EntityID
	Creature CreatureID
}

// PickTarget is the full targeting rule, shared by the server and the
// client: the nearest of the entity NearestTarget would pick and every
// creature within its kind's reach plus reachBonus (a held spear, see
// ReachBonusFor). A tie goes to the entity, then to the lower creature id.
func PickTarget(x, y float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool), creatures []CreatureCandidate, reachBonus float64) (Target, bool) {
	var best Target
	bestD := math.Inf(1)
	if id, d, ok := nearestEntity(x, y, hasWood, at); ok {
		best, bestD = Target{Entity: id}, d
	}
	for _, c := range creatures {
		r := c.Kind.Reach() + reachBonus
		dx, dy := c.X-x, c.Y-y
		d := dx*dx + dy*dy
		if c.Kind == CreatureNone || c.Kind >= NumCreatureKinds || d > r*r {
			continue
		}
		if d < bestD || (d == bestD && best.Creature != 0 && c.ID < best.Creature) {
			best, bestD = Target{Creature: c.ID}, d
		}
	}
	return best, !math.IsInf(bestD, 1)
}

// creatureCandidates lists the creatures that could be in reach of (x, y).
func (w *World) creatureCandidates(x, y float64) []CreatureCandidate {
	const far = 3.0 // more than any creature's reach, spear included
	var out []CreatureCandidate
	w.EachCreatureIn(x-far, y-far, x+far, y+far, func(c Creature) {
		if !c.gone && c.State != CreatureHooked && c.State != CreatureHidden && c.State != CreatureLurk { // someone else's fish; down a hole; under water
			out = append(out, CreatureCandidate{ID: c.ID, Kind: c.Kind, X: c.X, Y: c.Y})
		}
	})
	return out
}
