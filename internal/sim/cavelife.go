package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// The caves of the tribal age run deep under the mountains, with chambers
// and side passages, and things live in the dark:
//
//   - Venomous spiders. One that sees a player spits a web at them from a
//     few tiles off (a player caught in it is slowed for a moment), then
//     runs in and bites; the bite poisons, and poison eats health for a
//     while. A bandage or the big healing draught draws it out.
//   - Cave bears, slow but strong: a swipe of the claws, then a bite.
//   - Old chests, deep in, far from any way out. Opened, they spill what
//     someone hid there long ago; the deeper, the better.

// Cave tuning.
const (
	SpiderSight     = 7.0
	SpiderGiveUp    = 11.0
	SpiderBiteRange = 1.15
	SpiderBite      = 6.0
	SpiderBiteCD    = 2 * TicksPerSecond
	SpiderSpitMin   = 2.2
	SpiderSpitMax   = 6.5
	SpiderSpitCD    = 4 * TicksPerSecond
	SpiderSpitTicks = 10

	WebSpeed   = 8.0 / TicksPerSecond
	WebLife    = TicksPerSecond
	WebHit     = 0.5
	WebTicks   = 5 * TicksPerSecond / 2 // a webbed player is slowed this long
	WebSlow    = 0.35                   // to this share of their speed
	PoisonTick = 1.5 / TicksPerSecond   // health lost per tick while poisoned
	PoisonLong = 6 * TicksPerSecond     // a bite poisons this long

	BearSight     = 6.0
	BearGiveUp    = 12.0
	BearClawRange = 1.9
	BearClaw      = 14.0
	BearClawCD    = 3 * TicksPerSecond / 2
	BearBite      = 20.0
	BearBiteCD    = 9 * TicksPerSecond / 4

	treasureDepth   = 12 // tiles of cave from the nearest mouth before chests appear
	treasureSpacing = 10
	treasureEvery   = 140 // one chest per this many deep cave tiles
)

var (
	spiderDef = creatureDef{
		name: "Nhện độc", hp: 5,
		walk: 1.2 / TicksPerSecond, run: 3.2 / TicksPerSecond, // a player outruns it
		radius: 0.35, reach: 1.6,
		yield: ItemSilk, yieldN: 2,
		leash: 9, idle: [2]int{30, 120}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileCave}, tilesPer: 70,
		fights: true, openOnly: true, away: 18,
	}
	bearDef = creatureDef{
		name: "Gấu hang", hp: 22,
		walk: 0.9 / TicksPerSecond, run: 3.4 / TicksPerSecond,
		radius: 0.6, reach: 2.1,
		yield: ItemMeat, yieldN: 6, drops: []Ingredient{{ItemFur, 3}, {ItemBone, 2}},
		leash: 10, idle: [2]int{60, 200}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileCave}, tilesPer: 320,
		fights: true, tough: true, openOnly: true, away: 24,
	}
)

func init() {
	creatureDefs[CreatureSpider] = spiderDef
	creatureDefs[CreatureBear] = bearDef
}

// Web is a strand of web a spider spat, flying.
type Web struct {
	ID     uint32
	X, Y   float64
	VX, VY float64
	life   int
}

// Webs returns copies of the webs in the air.
func (w *World) Webs() []Web {
	out := make([]Web, 0, len(w.webs))
	for _, b := range w.webs {
		out = append(out, *b)
	}
	return out
}

func (w *World) stepWebs() {
	for _, b := range w.webs {
		b.life--
		b.X += b.VX
		b.Y += b.VY
		if !w.landAt(int(math.Floor(b.X)), int(math.Floor(b.Y))) {
			b.life = 0
			continue
		}
		for _, id := range w.playerOrder {
			p := w.players[id]
			if !p.Dead && math.Hypot(p.X-b.X, p.Y-b.Y) <= WebHit {
				p.webbed = WebTicks
				p.notify(NoticeWebbed, 0)
				b.life = 0
				break
			}
		}
	}
	w.webs = slices.DeleteFunc(w.webs, func(b *Web) bool { return b.life <= 0 })
}

func (w *World) stepSpider(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	if c.State == CreatureStrike || c.State == CreatureSpit {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	if p == nil || dist > SpiderGiveUp || (c.State != CreatureChase && dist > SpiderSight) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		w.wander(c)
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= SpiderBiteRange {
		if c.biteCD == 0 {
			strike(c, p, SpiderBite, SpiderBiteCD)
			p.poison = max(p.poison, PoisonLong)
		}
		return
	}
	if c.cool == 0 && dist >= SpiderSpitMin && dist <= SpiderSpitMax {
		c.State, c.timer, c.cool = CreatureSpit, SpiderSpitTicks, SpiderSpitCD
		w.nextWeb++
		w.webs = append(w.webs, &Web{
			ID: w.nextWeb, X: c.X, Y: c.Y - 0.1,
			VX: (p.X - c.X) / dist * WebSpeed, VY: (p.Y - c.Y) / dist * WebSpeed,
			life: WebLife,
		})
		return
	}
	w.runAt(c, p.X, p.Y, d.run, SpiderBiteRange*0.8)
}

func (w *World) stepBear(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.State == CreatureStrike || c.State == CreatureClaw {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	if p == nil || dist > BearGiveUp || (c.State != CreatureChase && dist > BearSight) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		w.wander(c)
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= BearClawRange {
		if c.biteCD == 0 {
			// A swipe, then a bite, turn about.
			if c.ang == 0 {
				strike(c, p, BearClaw, BearClawCD)
				c.State, c.ang = CreatureClaw, 1
			} else {
				strike(c, p, BearBite, BearBiteCD)
				c.ang = 0
			}
		}
		return
	}
	w.runAt(c, p.X, p.Y, d.run, BearClawRange*0.8)
}

// stepPoison is a tick of what spiders did to p: poison and webs.
func (w *World) stepPoison(p *Player) {
	if p.webbed > 0 {
		p.webbed--
	}
	if p.poison > 0 {
		p.poison--
		hurt(p, PoisonTick)
	}
}

// Poisoned reports whether p has venom in them.
func (p *Player) Poisoned() bool { return p.poison > 0 }

// Webbed reports whether p is tangled in web.
func (p *Player) Webbed() bool { return p.webbed > 0 }

// --- the deep caves ------------------------------------------------------------

// caveDepth returns, for every cave tile, how many steps it lies from the
// nearest way out; -1 for every other tile.
func (w *World) caveDepth() []int32 {
	depth := make([]int32, len(w.tiles))
	var queue []int32
	for i, t := range w.tiles {
		depth[i] = -1
		if t != TileCave {
			continue
		}
		x, y := i%w.width, i/w.width
		if w.caveOutside(x, y+1) || w.caveOutside(x, y-1) || w.caveOutside(x+1, y) || w.caveOutside(x-1, y) {
			depth[i] = 0
			queue = append(queue, int32(i))
		}
	}
	for k := 0; k < len(queue); k++ {
		i := int(queue[k])
		x, y := i%w.width, i/w.width
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.inBounds(nx, ny) {
				continue
			}
			if j := ny*w.width + nx; depth[j] < 0 && w.tiles[j] == TileCave {
				depth[j] = depth[i] + 1
				queue = append(queue, int32(j))
			}
		}
	}
	return depth
}

// placeTreasure hides old chests in the deep parts of the caves.
func (w *World) placeTreasure(rng *rand.Rand) {
	depth := w.caveDepth()
	var deep []int
	for i, d := range depth {
		if d >= treasureDepth {
			deep = append(deep, i)
		}
	}
	rng.Shuffle(len(deep), func(a, b int) { deep[a], deep[b] = deep[b], deep[a] })
	want := max(2, len(deep)/treasureEvery)
	var placed [][2]int
	for _, i := range deep {
		if len(placed) >= want {
			break
		}
		x, y := i%w.width, i/w.width
		if !w.canPlace(x, y) {
			continue
		}
		near := false
		for _, q := range placed {
			if abs(q[0]-x)+abs(q[1]-y) < treasureSpacing {
				near = true
				break
			}
		}
		if near {
			continue
		}
		e := w.addEntity(EntityTreasure, x, y)
		e.Var = uint8(min(255, depth[i])) // how deep: the deeper, the better the hoard
		placed = append(placed, [2]int{x, y})
	}
}

// treasureLoot is what an old chest at depth holds, decided by its id.
func treasureLoot(id EntityID, depth int) []Ingredient {
	r := rand.New(rand.NewPCG(uint64(id), uint64(depth)*7919+0x7ea5))
	deep := min(1, float64(depth-treasureDepth)/30) // 0 at the shallowest chests, 1 far down
	type pick struct {
		item     ItemKind
		lo, hi   int
		p, pDeep float64
	}
	table := []pick{
		{ItemPotion, 1, 2, 0.8, 0.9},
		{ItemTorch, 2, 4, 0.6, 0.5},
		{ItemCoal, 3, 6, 0.5, 0.4},
		{ItemBone, 2, 5, 0.5, 0.4},
		{ItemIronOre, 2, 4, 0.35, 0.5},
		{ItemPlatinumOre, 1, 3, 0.15, 0.45},
		{ItemDiamond, 1, 2, 0.03, 0.3},
		{ItemRexTooth, 1, 1, 0.03, 0.2},
		{ItemBigPotion, 1, 1, 0.1, 0.45},
		{ItemGlowSeed, 1, 3, 0.3, 0.3},
		{ItemTreeSeed, 1, 3, 0.25, 0.15},
		{ItemManEaterSeed, 1, 1, 0.05, 0.2},
		{ItemDinoHide, 1, 3, 0.2, 0.35},
		{ItemIvorySpear, 1, 1, 0.02, 0.12},
	}
	var out []Ingredient
	for _, t := range table {
		if r.Float64() < t.p+(t.pDeep-t.p)*deep {
			out = append(out, Ingredient{Item: t.item, Count: t.lo + r.IntN(t.hi-t.lo+1)})
		}
	}
	if len(out) == 0 {
		out = append(out, Ingredient{Item: ItemPotion, Count: 1})
	}
	return out
}

// openTreasure opens an old chest: its hoard spills on the ground, and it
// stays open and empty.
func (w *World) openTreasure(p *Player, e *Entity) bool {
	e.Ready = false
	e.RegrowAt = math.MaxUint64 // never fills again
	w.dropAt(float64(e.TX)+0.5, float64(e.TY)+0.9, treasureLoot(e.ID, int(e.Var)))
	p.notify(NoticeTreasure, 0)
	return true
}
