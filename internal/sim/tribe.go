package sim

import (
	"math"
	"math/rand/v2"
)

// The woodland tribes. Far from where players start, in clearings of the
// meadows and the woods, stand villages of wooden huts round a carved post
// and a fire that never goes out. Their warriors, spear in hand, roam near
// home: they hunt the smaller beasts, and they do not like strangers. One
// who sees a player close by comes at them and thrusts with the spear; a
// badly hurt warrior runs for home. They are a little slower than a player.
// A warrior who falls drops what he carried; the village sends out another
// in time, while its huts stand.

// Tribe tuning.
const (
	TribeSightDay   = 7.0
	TribeSightNight = 5.0
	TribeGiveUp     = 13.0
	TribeLeash      = 16.0 // a warrior chases no further than this from home
	TribeThrust     = 1.75
	TribeDamage     = 12.0
	TribeThrustCD   = 7 * TicksPerSecond / 5
	TribeFleeBelow  = 3
	TribeRespawn    = 60 * TicksPerSecond

	villageRadius  = 7  // tiles cleared for a village
	villageHuts    = 6  // huts round the post, about
	villageFolk    = 5  // warriors per village
	villageMinAway = 48 // tiles from the spawn, at least
)

var tribesmanDef = creatureDef{
	name: "Thổ dân", hp: 10,
	walk: 1.3 / TicksPerSecond, run: 3.7 / TicksPerSecond,
	radius: 0.3, reach: 1.7,
	yield: ItemHide, yieldN: 1, drops: []Ingredient{{ItemBone, 1}},
	leash: 9, idle: [2]int{30, 120}, roam: [2]int{20, 70},
	fights: true, openOnly: true,
}

func init() {
	creatureDefs[CreatureTribesman] = tribesmanDef
	diets[CreatureTribesman] = dietDef{
		prey: []CreatureKind{CreatureRabbit, CreatureThesc, CreatureOrnitho, CreatureAchero},
		hunger: [2]int{120 * TicksPerSecond, 260 * TicksPerSecond},
		thirst: [2]int{150 * TicksPerSecond, 300 * TicksPerSecond}, eatTicks: 6 * TicksPerSecond, bite: 3,
	}
}

// Village is one settlement of a tribe.
type Village struct {
	X, Y float64 // the post at its heart
	Folk int     // how many warriors it keeps
}

// Villages returns the villages of the world.
func (w *World) Villages() []Village { return append([]Village(nil), w.villages...) }

// pickVillages chooses where the villages stand: on open meadow or wood,
// far from the spawn and from each other, with no water or rock close.
func (w *World) pickVillages(rng *rand.Rand, sx, sy int) {
	n := max(w.width, w.height)
	want := max(2, n/110)
	minAway := max(float64(villageMinAway), float64(n)/5)
	apart := max(40.0, float64(n)/4)
	ok := func(tx, ty int) bool {
		if math.Hypot(float64(tx-sx), float64(ty-sy)) < minAway {
			return false
		}
		for _, v := range w.villages {
			if math.Hypot(v.X-float64(tx), v.Y-float64(ty)) < apart {
				return false
			}
		}
		for y := ty - villageRadius - 1; y <= ty+villageRadius+1; y++ {
			for x := tx - villageRadius - 1; x <= tx+villageRadius+1; x++ {
				if t := w.TileAt(x, y); t != TileGrass && t != TileForest {
					return false
				}
			}
		}
		return true
	}
	for tries := 0; tries < 4000 && len(w.villages) < want; tries++ {
		tx := villageRadius + 2 + rng.IntN(max(1, w.width-2*villageRadius-4))
		ty := villageRadius + 2 + rng.IntN(max(1, w.height-2*villageRadius-4))
		if ok(tx, ty) {
			w.villages = append(w.villages, Village{X: float64(tx) + 0.5, Y: float64(ty) + 0.5, Folk: villageFolk})
		}
	}
}

// buildVillages clears each village site and raises its post, fire and huts.
func (w *World) buildVillages(rng *rand.Rand) {
	for _, v := range w.villages {
		cx, cy := int(v.X), int(v.Y)
		for ty := cy - villageRadius; ty <= cy+villageRadius; ty++ {
			for tx := cx - villageRadius; tx <= cx+villageRadius; tx++ {
				if e := w.entityAt(tx, ty); e != nil && math.Hypot(float64(tx-cx), float64(ty-cy)) <= villageRadius {
					w.removeEntity(e)
				}
				if w.TileAt(tx, ty) == TileForest && math.Hypot(float64(tx-cx), float64(ty-cy)) <= villageRadius-1 {
					w.tiles[ty*w.width+tx] = TileGrass // a trodden clearing
				}
			}
		}
		w.addEntity(EntityTotem, cx, cy)
		w.addEntity(EntityBonfire, cx+2, cy+1)
		huts := villageHuts + rng.IntN(2)
		a0 := rng.Float64() * 2 * math.Pi
		for i := range huts {
			a := a0 + 2*math.Pi*float64(i)/float64(huts)
			d := 4.5 + rng.Float64()*1.2
			tx, ty := cx+int(math.Round(math.Cos(a)*d)), cy+int(math.Round(math.Sin(a)*d*0.85))
			if w.canPlace(tx, ty) && w.TileAt(tx, ty+1).Walkable() {
				w.addEntity(EntityHut, tx, ty)
			}
		}
	}
}

// spawnTribes puts each village's warriors round it.
func (w *World) spawnTribes() {
	for i := range w.villages {
		for range w.villages[i].Folk {
			w.addTribesman(i)
		}
	}
}

// addTribesman puts a warrior of village i near its post.
func (w *World) addTribesman(i int) *Creature {
	v := w.villages[i]
	r := w.random()
	for range 12 {
		a, d := r.Float64()*2*math.Pi, 1.5+r.Float64()*3
		x, y := v.X+math.Cos(a)*d, v.Y+math.Sin(a)*d
		if w.creatureFits(CreatureTribesman, x, y) {
			c := w.addCreature(CreatureTribesman, x, y)
			c.homeX, c.homeY = v.X, v.Y
			c.village = i + 1
			return c
		}
	}
	return nil
}

// repopulateTribes sends out a new warrior from a village short of them, if
// no player is watching.
func (w *World) repopulateTribes() {
	if len(w.villages) == 0 || w.tick%TribeRespawn != 0 {
		return
	}
	count := make([]int, len(w.villages))
	for _, c := range w.fauna.creatures {
		if c.Kind == CreatureTribesman && c.village > 0 && !c.gone {
			count[c.village-1]++
		}
	}
	for i, v := range w.villages {
		if count[i] >= v.Folk {
			continue
		}
		if _, d := w.nearestPlayer(v.X, v.Y); d < creatureSpawnAway {
			continue
		}
		if w.villageHuts(v) == 0 {
			continue // a village burnt out sends no one
		}
		w.addTribesman(i)
	}
}

func (w *World) villageHuts(v Village) int {
	n := 0
	cx, cy := int(v.X), int(v.Y)
	w.EachEntityIn(cx-villageRadius, cy-villageRadius, cx+villageRadius, cy+villageRadius, func(e Entity) {
		if e.Kind == EntityHut {
			n++
		}
	})
	return n
}

func (w *World) stepTribesman(c *Creature) {
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
	sight := TribeSightDay
	if Ambient(w.tick) < 0.3 {
		sight = TribeSightNight
	}
	if w.huntOn(c, sight) {
		return // after game: see needs.go
	}
	if c.State == CreatureFlee { // hurt: back to the village
		if math.Hypot(c.homeX-c.X, c.homeY-c.Y) < 2 || c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
		} else {
			c.timer--
			w.runAt(c, c.homeX, c.homeY, d.run, 1)
		}
		return
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	if p != nil && c.HP <= TribeFleeBelow && dist < TribeGiveUp {
		c.State, c.timer, c.rage = CreatureFlee, 10*TicksPerSecond, 0
		return
	}
	homeD := math.Hypot(c.homeX-c.X, c.homeY-c.Y)
	if p == nil || dist > TribeGiveUp || homeD > TribeLeash || (c.State != CreatureChase && dist > sight) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		if homeD > TribeLeash {
			w.runAt(c, c.homeX, c.homeY, d.walk*1.5, 1)
			return
		}
		if !w.forage(c) {
			w.wander(c)
		}
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= TribeThrust {
		if c.biteCD == 0 {
			strike(c, p, TribeDamage, TribeThrustCD)
		}
		return
	}
	w.runAt(c, p.X, p.Y, d.run, TribeThrust*0.8)
}

// tribesmanLoot is what a fallen warrior leaves: his spear now and then,
// a hide, a bone.
func (w *World) tribesmanLoot() []Ingredient {
	out := []Ingredient{{Item: ItemHide, Count: 1}, {Item: ItemBone, Count: 1}}
	if w.random().Float64() < 0.4 {
		out = append(out, Ingredient{Item: ItemSpear, Count: 1})
	}
	return out
}
