package sim

import "math"

// Taming and riding.
//
// A grazer is tamed by feeding it from the hand: hold its food and use it
// on the beast, again and again, while it is calm. A hunter takes food only
// once it has been beaten down and lies cowering. Hitting a beast undoes the
// taming so far. A tamed beast is its tamer's for good, and is saved with
// the world.
//
// A pet follows its owner, and the ones that fight go for any beast that
// menaces them. Using a pet tells it to wait, or to follow again; with its
// food in hand, it eats and mends instead. Some pets can be ridden: with a
// saddle in the bag, the ride command gets on the nearest one, and the
// player steers it, faster than on foot.

// Taming tuning.
const (
	PetFollowNear  = 2.6  // a following pet stops this close to its owner
	PetFollowFar   = 4.0  // and sets off after them past this
	PetCatchUp     = 24.0 // a pet left this far behind is brought along
	PetGuardDist   = 6.0  // pets fight beasts menacing their owner this close
	PetStrikeCD    = 3 * TicksPerSecond / 2
	PetHealTicks   = 3 * TicksPerSecond // a pet mends a point this often
	PetMealHeal    = 4                  // hit points a meal gives back to a pet
	FeedCD         = TicksPerSecond     // the least time between two meals
	MountReach     = 2.8                // the ride command finds a pet this close
	RideReachBonus = 0.6                // a rider reaches creatures further, from up there

	RaptorCowerTicks = 8 * TicksPerSecond // how long a beaten raptor lies low
)

type tameDef struct {
	food  ItemKind // what it is tamed with; ItemNone if it cannot be
	meals int      // how many
	ride  float64  // tiles per tick when ridden; 0 if it cannot be ridden
	bite  int      // damage to a beast when it defends its owner; 0 if it does not fight
	cower bool     // a hunter: it must be beaten down before it takes food
}

var tameDefs = [NumCreatureKinds]tameDef{
	CreatureTrike:  {food: ItemBerry, meals: 5, ride: 5.5 / TicksPerSecond, bite: 4},
	CreatureStego:  {food: ItemBerry, meals: 5, bite: 4},
	CreatureRaptor: {food: ItemMeat, meals: 3, ride: 6.5 / TicksPerSecond, bite: 2, cower: true},
}

func (k CreatureKind) tameDef() tameDef {
	if k >= NumCreatureKinds {
		return tameDef{}
	}
	return tameDefs[k]
}

// TameFood is what a creature of kind k is tamed with: ItemNone if it
// cannot be tamed.
func (k CreatureKind) TameFood() ItemKind { return k.tameDef().food }

// TameMeals is how many meals tame a creature of kind k.
func (k CreatureKind) TameMeals() int { return k.tameDef().meals }

// Rideable reports whether a tamed creature of kind k can be ridden.
func (k CreatureKind) Rideable() bool { return k.tameDef().ride > 0 }

// MustCower reports whether a creature of kind k must be beaten down before
// it takes food.
func (k CreatureKind) MustCower() bool { return k.tameDef().cower }

// Tamed reports whether c belongs to someone.
func (c *Creature) Tamed() bool { return c.ownerName != "" }

// Ridden reports whether someone is riding c.
func (c *Creature) Ridden() bool { return c.rider != 0 }

// Mount is the creature p is riding, 0 when on foot.
func (p *Player) Mount() CreatureID { return p.mount }

// RideReach is the extra reach of a rider, for the client's highlight.
func RideReach(mounted bool) float64 {
	if mounted {
		return RideReachBonus
	}
	return 0
}

// petAction handles an Action by p on c that tames, feeds or orders it
// rather than hitting it. handled is false for an ordinary hit; acted says
// whether anything was done.
func (w *World) petAction(p *Player, c *Creature) (handled, acted bool) {
	td := c.Kind.tameDef()
	if c.Tamed() {
		if c.ownerName != p.Name {
			p.notify(NoticeNotYours, uint8(c.Kind))
			return true, false
		}
		if td.food.Valid() && p.Held() == td.food && c.HP < c.Kind.MaxHP() {
			if c.fedCD > 0 {
				return true, false
			}
			p.Inv.TakeSlot(p.Selected, 1)
			c.HP = min(c.Kind.MaxHP(), c.HP+PetMealHeal*HealthScale)
			c.fedCD = FeedCD
			p.notify(NoticeHealed, uint8(c.Kind))
			return true, true
		}
		c.Stay = !c.Stay
		c.homeX, c.homeY = c.X, c.Y
		if c.Stay {
			p.notify(NoticePetStay, uint8(c.Kind))
		} else {
			p.notify(NoticePetCome, uint8(c.Kind))
		}
		return true, true
	}
	if !td.food.Valid() || p.Held() != td.food {
		return false, false // a hit
	}
	switch {
	case (c.Kind == CreatureOrnitho || c.Kind == CreatureMonkey) && c.State == CreatureFlee:
		p.notify(NoticeSkittish, uint8(c.Kind))
		return true, false
	case td.cower && c.State != CreatureCower:
		p.notify(NoticeTooWild, uint8(c.Kind))
		return true, false
	case c.rage > 0 || c.State == CreatureChase || c.State == CreatureStrike || c.State == CreatureFlee:
		p.notify(NoticeAngry, uint8(c.Kind))
		return true, false
	case c.fedCD > 0:
		return true, false
	}
	p.Inv.TakeSlot(p.Selected, 1)
	c.Tame++
	c.fedCD = FeedCD
	if c.State != CreatureCower {
		c.State, c.timer = CreatureIdle, 2*TicksPerSecond // it stops to eat
	}
	if c.Kind == CreatureOrnitho || c.Kind == CreatureMonkey {
		c.cool = OrnithoCalmTicks // and lets the feeder stay close a while
	}
	if c.Tame < td.meals {
		p.notify(NoticeFed, uint8(c.Kind))
		return true, true
	}
	w.tame(c, p.Name)
	c.Owner = p.ID
	p.notify(NoticeTamed, uint8(c.Kind))
	return true, true
}

// tame makes c the pet of the player called owner.
func (w *World) tame(c *Creature, owner string) {
	c.ownerName = owner
	c.Tame = c.Kind.TameMeals()
	c.State, c.timer, c.rage, c.foe, c.cool = CreatureIdle, 0, 0, 0, 0
	c.HP = c.Kind.MaxHP()
	c.Stay = false
	c.homeX, c.homeY = c.X, c.Y
}

// wound takes dmg from c, hit by by, and reports whether that killed it.
// Being hurt undoes any taming so far.
func (w *World) wound(c *Creature, dmg int, by *Player) (killed bool) {
	d := c.Kind.def()
	c.HP -= dmg
	c.Tame = 0
	if c.HP <= 0 {
		w.fell(c) // the body falls; see drops.go
		return true
	}
	if d.tough && c.State != CreatureHurt {
		c.after, c.afterT = c.State, c.timer
	}
	w.provoke(c, by) // see dino.go: it may change what comes after
	if c.windingUp() {
		return false // drawn back for a blow, it does not flinch: see telegraph.go
	}
	if !d.tough && c.flinchCD > 0 {
		return false // it shrugs off a blow so soon after the last: see combat.go
	}
	c.State, c.timer = CreatureHurt, CreatureHurtTicks
	c.flinchCD = FlinchEvery
	if d.tough {
		c.timer = toughHurtTicks
	}
	return false
}

// ownerOf finds c's owner among the players online, and keeps c.Owner up
// to date; nil when they are away.
func (w *World) ownerOf(c *Creature) *Player {
	for _, id := range w.playerOrder {
		if p := w.players[id]; p.Name == c.ownerName {
			c.Owner = p.ID
			return p
		}
	}
	c.Owner = 0
	return nil
}

func (w *World) stepPet(c *Creature) {
	d := c.Kind.def()
	td := c.Kind.tameDef()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.HP < c.Kind.MaxHP() {
		if c.healT--; c.healT <= 0 {
			c.HP = min(c.Kind.MaxHP(), c.HP+HealthScale)
			c.healT = PetHealTicks
		}
	}
	owner := w.ownerOf(c)
	if c.rider != 0 {
		return // its rider moves it
	}
	if c.State == CreatureStrike {
		if c.timer--; c.timer > 0 {
			return
		}
		c.State = CreatureIdle
	}
	if owner == nil || owner.Dead || c.Stay {
		w.wander(c) // round the spot it was left at
		return
	}
	if td.bite > 0 {
		if foe := w.threatTo(owner, c); foe != nil {
			w.petFight(c, foe, owner, td.bite)
			return
		}
	}
	dist := math.Hypot(owner.X-c.X, owner.Y-c.Y)
	if dist > PetCatchUp {
		w.bringAlong(c, owner)
		return
	}
	c.homeX, c.homeY = c.X, c.Y // wherever it is with its owner is home
	if dist > PetFollowFar || (c.State == CreatureWander && dist > PetFollowNear+0.1) {
		c.State = CreatureWander
		speed := d.run
		if dist < 6 {
			speed = max(d.walk*2.5, d.run*0.6)
		}
		if !w.runAt(c, owner.X, owner.Y, speed, PetFollowNear) {
			c.dodge(w, owner.X, owner.Y)
			w.moveCreature(c, c.hx*speed, c.hy*speed)
		}
		return
	}
	c.State = CreatureIdle
	c.faceToward(owner.X)
}

// bringAlong puts a pet that was left far behind next to its owner.
func (w *World) bringAlong(c *Creature, p *Player) {
	for _, o := range [][2]float64{{-1.6, 0.6}, {1.6, 0.6}, {0, 1.6}, {-1.6, -0.8}, {1.6, -0.8}, {0, -1.6}} {
		if x, y := p.X+o[0], p.Y+o[1]; w.creatureFits(c.Kind, x, y) {
			c.X, c.Y = x, y
			c.homeX, c.homeY = x, y
			return
		}
	}
}

// threatTo is the nearest wild beast menacing p, within a pet's guard.
func (w *World) threatTo(p *Player, pet *Creature) *Creature {
	var best *Creature
	bestD := PetGuardDist
	for _, o := range w.fauna.creatures {
		if o.gone || o == pet || o.Tamed() || !o.Kind.def().fights || o.State.OutOfReach() {
			continue
		}
		menacing := (o.rage > 0 && o.foe == p.ID) || o.State == CreatureChase || o.State == CreatureStrike || o.State == CreatureRoar
		if !menacing {
			continue
		}
		if d := math.Hypot(o.X-p.X, o.Y-p.Y); d < bestD {
			best, bestD = o, d
		}
	}
	return best
}

// petFight has pet go for foe on owner's behalf; a kill goes to the owner.
func (w *World) petFight(pet, foe *Creature, owner *Player, bite int) {
	reach := pet.Kind.def().radius + foe.Kind.def().radius + 0.9
	dist := math.Hypot(foe.X-pet.X, foe.Y-pet.Y)
	if dist > reach {
		pet.State = CreatureChase
		w.runAt(pet, foe.X, foe.Y, pet.Kind.def().run, reach*0.8)
		return
	}
	pet.faceToward(foe.X)
	if pet.biteCD > 0 {
		return
	}
	pet.State, pet.timer, pet.biteCD = CreatureStrike, AttackTicks, PetStrikeCD
	w.wound(foe, bite*HealthScale, owner) // a kill's body lies there for its owner to butcher
}

// ride gets p on its nearest pet that can be ridden, or off the one it rides.
func (w *World) ride(p *Player) {
	if p.mount != 0 {
		w.dismount(p)
		return
	}
	var best, other *Creature
	bestD := MountReach
	for _, c := range w.fauna.creatures {
		if c.gone || c.ownerName != p.Name || c.rider != 0 {
			continue
		}
		d := math.Hypot(c.X-p.X, c.Y-p.Y)
		if d > MountReach {
			continue
		}
		if !c.Kind.Rideable() {
			other = c
			continue
		}
		if d <= bestD {
			best, bestD = c, d
		}
	}
	switch {
	case best == nil && other != nil:
		p.notify(NoticeCantRide, uint8(other.Kind))
	case best == nil:
		p.notify(NoticeNoMount, 0)
	case p.Inv.Count(ItemSaddle) == 0:
		p.notify(NoticeNeedTool, uint8(ItemSaddle))
	default:
		p.mount, best.rider = best.ID, p.ID
		best.State, best.timer, best.Stay = CreatureIdle, 0, false
		p.X, p.Y = best.X, best.Y
	}
}

// dismount gets p off its mount, onto free ground beside it.
func (w *World) dismount(p *Player) {
	c := w.creatureByID(p.mount)
	p.mount = 0
	if c == nil {
		return
	}
	c.rider = 0
	c.State, c.timer = CreatureIdle, 0
	c.homeX, c.homeY = c.X, c.Y
	for _, o := range [][2]float64{{0, 0.9}, {0.9, 0}, {-0.9, 0}, {0, -0.9}, {0.8, 0.8}, {-0.8, 0.8}} {
		if w.standable(c.X+o[0], c.Y+o[1]) {
			p.X, p.Y = c.X+o[0], c.Y+o[1]
			return
		}
	}
}

// rideStep moves p's mount as p steers it, and p with it. It reports
// whether p is still riding.
func (w *World) rideStep(p *Player, moving bool) bool {
	c := w.creatureByID(p.mount)
	if c == nil || c.rider != p.ID {
		p.mount = 0
		return false
	}
	c.State = CreatureIdle
	if moving {
		mx, my := float64(p.input.MoveX), float64(p.input.MoveY)
		l := math.Hypot(mx, my)
		sp := c.Kind.tameDef().ride * w.speedAt(c.X, c.Y)
		if w.moveCreature(c, mx/l*sp, my/l*sp) {
			c.State = CreatureWander
		}
		p.Dir = dirToward(mx, my)
	}
	p.X, p.Y = c.X, c.Y
	return true
}

// PetSave is a tamed beast in a save.
type PetSave struct {
	Kind  CreatureKind
	X, Y  float64
	HP    int
	Owner string
	Stay  bool `json:",omitempty"`
}

func (w *World) savePets() []PetSave {
	var out []PetSave
	for _, c := range w.fauna.creatures {
		if c.Tamed() && !c.gone {
			out = append(out, PetSave{Kind: c.Kind, X: c.X, Y: c.Y, HP: c.HP, Owner: c.ownerName, Stay: c.Stay})
		}
	}
	return out
}

// loadPets puts saved pets back: where they were, or as near as they fit.
func (w *World) loadPets(pets []PetSave) {
	for _, ps := range pets {
		name := cleanName(ps.Owner)
		if ps.Kind >= NumCreatureKinds || ps.Kind.TameMeals() == 0 || name == "" || !finite(ps.X, ps.Y) {
			continue
		}
		x, y, ok := ps.X, ps.Y, w.creatureFits(ps.Kind, ps.X, ps.Y)
		for r := 1.0; !ok && r <= 8; r++ {
			for a := 0.0; a < 2*math.Pi && !ok; a += math.Pi / 6 {
				x, y = ps.X+math.Cos(a)*r, ps.Y+math.Sin(a)*r
				ok = w.creatureFits(ps.Kind, x, y)
			}
		}
		if !ok {
			continue
		}
		c := w.addCreature(ps.Kind, x, y)
		w.tame(c, name)
		c.HP = min(max(ps.HP, 1), ps.Kind.MaxHP())
		c.Stay = ps.Stay
	}
}
