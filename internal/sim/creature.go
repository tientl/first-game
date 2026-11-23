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

	// The great beasts of the open world: see dino.go.
	CreatureTrike   // three-horned grazer, in herds on the meadows
	CreatureStego   // plate-backed grazer of the woods, swings a spiked tail
	CreatureBrachio // long-necked giant, harmless, flees
	CreatureRaptor  // pack hunter of the woods
	CreatureRex     // the tyrant, far from the spawn
	CreaturePtero   // a flying reptile over the hills and beaches
	CreaturePlesio  // a long-necked sea reptile off the coast

	// The ice age: see iceage.go.
	CreatureMammoth // woolly, in herds on the snow
	CreatureSabre   // a sabre-toothed cat that stalks the snow

	// More of the tyrant's neighbours: see hellcreek.go.
	CreatureOrnitho // Ornithomimus: an ostrich of a dinosaur, the fastest runner
	CreatureThesc   // Thescelosaurus: a gentle middling grazer
	CreatureAnzu    // a giant crested bird of a dinosaur that kicks
	CreatureToro    // Torosaurus: the three-horn's bigger cousin
	CreatureAchero  // Acheroraptor: a smaller raptor in bigger packs
	CreatureQuetzal // Quetzalcoatlus: a giant flying reptile that stalks the ground
	CreatureBrachy  // Brachychampsa: a blunt-snouted crocodile of the ponds
	CreatureChampso // Champsosaurus: a long-snouted crocodile of the ponds

	// Cave life and the tribes: see cavelife.go and tribe.go.
	CreatureSpider    // a venomous cave spider that spits webs
	CreatureBear      // a cave bear that claws and bites
	CreatureTribesman // a warrior of the woodland tribes, spear in hand
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
	CreatureSoar                 // a flying reptile high in the air; out of reach
	CreatureRoar                 // a tyrant roaring before it charges
	CreatureCower                // a hunter beaten down: it lies low, and can be tamed; see tame.go
	CreatureEat                  // grazing, or eating a kill: head down, chewing; see needs.go
	CreatureDrink                // drinking at the water's edge
	CreatureClaw                 // a bear's swipe (its bite is CreatureStrike)
	CreatureTrapped              // caught in a snare: it struggles, going nowhere
	CreatureSpit                 // a spider spitting web
)

// OutOfReach reports whether a creature in state s cannot be targeted: on
// someone's fishing line, down a hole, under water or high in the air. The
// server and the client both use it.
func (s CreatureState) OutOfReach() bool {
	switch s {
	case CreatureHooked, CreatureHidden, CreatureLurk, CreatureSoar:
		return true
	}
	return false
}

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

	drops    []Ingredient // given besides yield when it is killed
	fights   bool         // fights back: hits land even when the bag cannot take the kill
	tough    bool         // big: a hit barely staggers it, and it goes on with what it was doing
	flies    bool         // crosses anything; out of reach while soaring
	deep     bool         // swims anywhere in the water, appearing a little off the coast
	openOnly bool         // never on the old island maps
	herd     int          // appears in groups of this many
	away     float64      // never appears this close to where players start

	// How a kind with a home lives; see homes.go.
	homeOn      map[Tile]float64 // how well each tile suits a home (0: never); nil: every spawn tile alike
	cover       Tile             // homes only within coverDist tiles of this tile, when coverDist > 0
	coverDist   int
	family      [2]int // animals to a home at first, fewest and most; perHome is the most it ever holds
	homeSpacing int    // tiles kept between two homes of this kind
	respawn     int    // ticks between top-ups of this kind; 0 means CreatureRespawnTicks
}

// loot is everything a creature of kind k gives when it is killed.
func (k CreatureKind) loot() []Ingredient {
	d := k.def()
	return append([]Ingredient{{Item: d.yield, Count: d.yieldN}}, d.drops...)
}

// Loot is everything a creature of kind k gives when it is killed.
func (k CreatureKind) Loot() []Ingredient { return k.loot() }

var creatureDefs = [NumCreatureKinds]creatureDef{
	CreatureNone: {name: "?"},
	CreatureRabbit: {
		name: "Thỏ", hp: 1,
		walk: 1.2 / TicksPerSecond, run: 5.2 / TicksPerSecond,
		radius: 0.2, reach: 1.4,
		yield: ItemMeat, yieldN: 2,
		// Small colonies, two or three to a burrow, dug in open meadow near the
		// cover of a wood's edge; colonies keep apart, and the young come fast.
		homeKind: EntityBurrow, perHome: 3, family: [2]int{2, 3},
		homeOn: map[Tile]float64{TileGrass: 1}, cover: TileForest, coverDist: 4,
		homeSpacing: 9, respawn: 30 * TicksPerSecond,
		leash: 10, idle: [2]int{20, 80}, roam: [2]int{15, 50},
		spawnOn: []Tile{TileGrass, TileForest}, tilesPer: 300, // per meadow tile
	},
	CreatureSnake: {
		name: "Rắn", hp: 3,
		walk: 0.8 / TicksPerSecond, run: 2.6 / TicksPerSecond,
		radius: 0.25, reach: 1.4,
		yield: ItemMeat, yieldN: 1,
		// Solitary, one to a den and a wide range each; dens are crevices in
		// rock first, sparse out on sand, desert and ash. Slow to come back.
		homeKind: EntityDen, perHome: 1, family: [2]int{1, 1},
		homeOn:      map[Tile]float64{TileRocky: 1, TileDesert: 0.12, TileSand: 0.1, TileAsh: 0.15},
		homeSpacing: 14, respawn: 120 * TicksPerSecond,
		leash: 7, idle: [2]int{30, 120}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileRocky, TileSand, TileDesert, TileAsh}, tilesPer: 320, // per weighted den tile
		fights: true,
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
		fights: true,
	},
	CreatureTrike:   trikeDef,
	CreatureStego:   stegoDef,
	CreatureBrachio: brachioDef,
	CreatureRaptor:  raptorDef,
	CreatureRex:     rexDef,
	CreaturePtero:   pteroDef,
	CreaturePlesio:  plesioDef,
	CreatureMammoth: mammothDef,
	CreatureSabre:   sabreDef,
	CreatureOrnitho: ornithoDef,
	CreatureThesc:   thescDef,
	CreatureAnzu:    anzuDef,
	CreatureToro:    toroDef,
	CreatureAchero:  acheroDef,
	CreatureQuetzal: quetzalDef,
	CreatureBrachy:  brachyDef,
	CreatureChampso: champsoDef,
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

	// Taming: see tame.go.
	Owner PlayerID // the online player it belongs to; 0 if wild, or its owner is away
	Tame  int      // meals towards taming it, while wild
	Stay  bool     // a pet told to wait where it is

	ownerName string   // who tamed it, kept across sessions; "" while wild
	rider     PlayerID // who is riding it, 0 if nobody
	fedCD     int      // ticks before it takes another meal
	healT     int      // ticks to its next point of mending, as a pet

	village      int // the village a warrior belongs to, 1-based; 0 for everyone else

	homeX, homeY float64
	hx, hy       float64 // heading while walking or fleeing, a unit vector
	timer        int     // ticks left in State
	biteCD       int
	hookedBy     PlayerID // whose float it is under, while CreatureHooked
	home         EntityID // its burrow or den; 0 if none
	goalX, goalY float64  // where a fleeing crab is making for
	gone         bool     // caught; removed at the end of the tick

	foe    PlayerID      // who it is angry with, while rage lasts
	rage   int           // ticks it stays angry with foe
	cool   int           // ticks before a hunter notices players again
	ang    float64       // a flying reptile's place on its circle
	after  CreatureState // what a tough beast goes back to after a hit
	afterT int           // and the timer it goes back to

	// Hunger, thirst and the hunt: see needs.go.
	hunger, thirst int        // ticks since its last meal and drink
	fed            bool       // hunger and thirst have been set going
	prey, hunter   CreatureID // what it hunts; what hunts it
	huntedT        int        // ticks left of the hunter's attention
	fleeT          int        // ticks it has run from a hunter: it tires
	trip           int        // ticks left of a walk to water
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
	drinkers  []*Creature               // those drinking at the start of this tick; see drinkerNear
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
	if x < r || y < r || x > float64(w.width)-r || y > float64(w.height)-r {
		return false
	}
	if d.flies {
		return true
	}
	// The corners of its box, and for a big beast points in between too, so
	// no wall or rock slips between two corners.
	n := max(1, int(math.Ceil(2*r/0.5)))
	for i := 0; i <= n; i++ {
		for j := 0; j <= n; j++ {
			if n > 1 && i != 0 && i != n && j != 0 && j != n {
				continue // the edge of the box is enough
			}
			tx := int(math.Floor(x - r + 2*r*float64(i)/float64(n)))
			ty := int(math.Floor(y - r + 2*r*float64(j)/float64(n)))
			land := w.landAt(tx, ty) && !w.beastBlocked(tx, ty)
			switch {
			case d.deep && w.TileAt(tx, ty) != TileWater:
				return false
			case d.water && !w.isShallow(tx, ty):
				return false
			case d.amphibious && !land && !w.isShallow(tx, ty):
				return false
			case !d.water && !d.deep && !d.amphibious && !land:
				return false
			}
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
	var offshore []uint8 // tiles from land, for sea creatures; see coastDistance
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		d := k.def()
		if d.openOnly && w.style == StyleIsland {
			continue
		}
		if d.deep && offshore == nil {
			offshore = w.coastDistance(plesioFar + 1)
		}
		var spots []int32
		for ty := 0; ty < w.height; ty++ {
			for tx := 0; tx < w.width; tx++ {
				t := w.tiles[ty*w.width+tx]
				if !slices.Contains(d.spawnOn, t) || (d.water && !w.isShallow(tx, ty)) || (d.seaside && !w.seaside(tx, ty)) || (d.waterside && !w.waterside(tx, ty)) {
					continue
				}
				if d.deep {
					if o := offshore[ty*w.width+tx]; o < plesioNear || o > plesioFar {
						continue
					}
				}
				if d.away > 0 && math.Hypot(float64(tx-sx), float64(ty-sy)) < d.away {
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
		if d.herd > 1 {
			for n := 0; n < f.target[k]; {
				i := spots[w.random().IntN(len(spots))]
				got := w.addHerd(k, float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5, min(d.herd, f.target[k]-n))
				n += max(got, 1) // a spot with no room still counts, so this ends
			}
			continue
		}
		if d.homeKind != EntityNone {
			// Families around homes, none too close to where players start.
			ok := func(tx, ty int) bool {
				away := clearSpawnRadius + 2
				if k == CreatureSnake {
					away = snakeSpawnClear + 2
				}
				return max(abs(tx-sx), abs(ty-sy)) >= away
			}
			f.target[k] = max(creatureMinPerKind, int(w.homeWeight(k, spots)/float64(d.tilesPer)))
			homes := max(creatureMinPerKind, int(math.Ceil(float64(f.target[k])/d.meanFamily())))
			if placed := w.placeHomes(k, spots, homes, ok); placed > 0 {
				f.target[k] = placed // what the world tops itself up to
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
		if !c.Tamed() { // pets do not count: the wild tops itself up
			count[c.Kind]++
		}
	}
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		spots := f.spawnAt[k]
		if count[k] >= f.target[k] || len(spots) == 0 {
			continue
		}
		if every := k.def().respawn; every > 0 && w.tick%uint64(every) != 0 {
			continue // this kind comes back more slowly
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
			if h := k.def().herd; h > 1 {
				w.addHerd(k, x, y, min(h, f.target[k]-count[k]))
				break
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
	f.drinkers = f.drinkers[:0]
	for _, c := range f.creatures {
		if c.State == CreatureDrink {
			f.drinkers = append(f.drinkers, c)
		}
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
	if c.State == CreatureTrapped {
		w.stepTrapped(c)
		return
	}
	if !c.Tamed() {
		w.stepTrapsAndSpikes(c) // see build.go
		if c.gone || c.State == CreatureTrapped {
			return
		}
	}
	if c.State == CreatureHurt {
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
			if c.Kind.def().tough && c.after != CreatureHurt {
				c.State, c.timer = c.after, c.afterT // a big beast shrugs it off
			}
		}
		return
	}
	if c.fedCD > 0 {
		c.fedCD--
	}
	if !c.Tamed() && w.stepNeeds(c) {
		return // eating, drinking, or running from a hunter: see needs.go
	}
	if c.Tamed() {
		w.stepPet(c) // see tame.go
		return
	}
	switch c.Kind {
	case CreatureTrike, CreatureStego, CreatureBrachio, CreatureMammoth, CreatureToro, CreatureThesc:
		w.stepGrazer(c)
	case CreatureSabre:
		w.stepSabre(c) // see iceage.go
	case CreatureRaptor, CreatureAchero:
		w.stepRaptor(c)
	case CreatureOrnitho:
		w.stepRunner(c) // see hellcreek.go
	case CreatureAnzu:
		w.stepAnzu(c)
	case CreatureQuetzal:
		w.stepQuetzal(c)
	case CreatureRex:
		w.stepRex(c)
	case CreaturePtero:
		w.stepPtero(c)
	case CreaturePlesio:
		w.stepPlesio(c)
	case CreatureRabbit:
		w.stepRabbit(c)
	case CreatureSnake:
		w.stepSnake(c)
	case CreatureCrab:
		w.stepCrab(c)
	case CreatureCroc, CreatureBrachy, CreatureChampso:
		w.stepCroc(c)
	case CreatureSpider:
		w.stepSpider(c) // see cavelife.go
	case CreatureBear:
		w.stepBear(c)
	case CreatureTribesman:
		w.stepTribesman(c) // see tribe.go
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
	if handled, acted := w.petAction(p, c); handled {
		return acted // tamed, fed or told what to do: see tame.go
	}
	if d.tool.Valid() && p.Held() != d.tool {
		p.notify(NoticeNeedTool, uint8(d.tool))
		return false
	}
	if d.bait.Valid() && p.Inv.Count(d.bait) == 0 {
		p.notify(NoticeNeedBait, uint8(d.bait))
		return false
	}
	// What a kill gives falls on the ground (see drops.go), so a full bag
	// never stops a blow; only a fish comes straight off the line into it.
	dmg := damageWith(p)
	fits := p.Inv
	if missing := addAll(&fits, c.Kind.loot()); missing.Valid() && d.onLine {
		p.notify(NoticeBagFull, uint8(missing))
		return false
	}
	if d.onLine {
		w.cast(p, c) // see fishing.go
		return true
	}
	if w.wound(c, dmg, p) && d.bait.Valid() {
		p.Inv.Take(d.bait, 1)
	}
	return true
}

// addAll puts every ingredient into inv, as much of each as fits, and
// returns the first kind that did not all fit (ItemNone if everything did).
func addAll(inv *Inventory, items []Ingredient) ItemKind {
	missing := ItemNone
	for _, in := range items {
		if inv.Add(in.Item, in.Count) < in.Count && missing == ItemNone {
			missing = in.Item
		}
	}
	return missing
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
	const far = CandidateRange
	var out []CreatureCandidate
	w.EachCreatureIn(x-far, y-far, x+far, y+far, func(c Creature) {
		if !c.gone && !c.State.OutOfReach() && c.rider == 0 && !w.underIce(&c) { // someone else's fish; down a hole; under water; up in the air; ridden; under the ice
			out = append(out, CreatureCandidate{ID: c.ID, Kind: c.Kind, X: c.X, Y: c.Y})
		}
	})
	return out
}
