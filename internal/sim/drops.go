package sim

import (
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Things on the ground. A felled tree, a broken boulder, a butchered kill
// or an old chest does not put its yield straight into the bag: it falls
// on the ground around it, and the player picks it up, with Space (the
// nearest pile within reach comes first) or by walking right over it.
// Piles that nobody takes rot away after a while.
//
// A beast that is killed lies where it fell, in its blood. Working on the
// body (Space) butchers it: the meat, hide, bones and its head fall out on
// the ground. Bodies nobody butchers rot away too.

// Drop and corpse tuning.
const (
	DropReach        = 1.4                // Space picks up a pile this close, before anything else
	DropTouch        = 0.5                // walking this close to a pile picks it up
	DropPickupDelay  = TicksPerSecond / 2 // a fresh pile cannot be walked up at once: it is still falling
	DropLifeTicks    = 5 * 60 * TicksPerSecond
	DropScatter      = 0.9                // how far piles spread from where they fell
	PickCooldown     = TicksPerSecond / 5 // Space picks up piles quickly, one after another
	CorpseRotTicks   = 4 * 60 * TicksPerSecond
	TwigChance       = 0.7  // a felled tree drops branches this often...
	TreeSeedChance   = 0.3  // ...and a seed this often
	PebbleChance     = 0.35 // a broken boulder leaves a pebble this often
	DinoHideFromBig  = 2    // dinosaur hides from a big dinosaur
	ButcherKnifeBits = 2    // a knife butchers this many times faster
)

// DropID identifies a pile on the ground. It is never reused within a world.
type DropID uint32

// Drop is a pile of one kind of item lying on the ground.
type Drop struct {
	ID    DropID
	Item  ItemKind
	Count int
	X, Y  float64 // where it lies
	FromX float64 `json:",omitempty"` // where it fell from: the client animates the fall
	FromY float64 `json:",omitempty"`
	Born  uint64  // tick it fell
	gone  bool

	tosser PlayerID // who threw it out of their bag, till they step off it; see toss.go
}

// Drops returns copies of every pile on the ground, in id order.
func (w *World) Drops() []Drop {
	out := make([]Drop, 0, len(w.drops))
	for _, d := range w.drops {
		out = append(out, *d)
	}
	return out
}

// EachDropIn calls fn with every pile whose centre lies in the rectangle.
func (w *World) EachDropIn(x0, y0, x1, y1 float64, fn func(Drop)) {
	for _, d := range w.drops {
		if d.X >= x0 && d.X <= x1 && d.Y >= y0 && d.Y <= y1 {
			fn(*d)
		}
	}
}

// dropAt lets items fall on the ground around (x, y), each kind as its own
// pile, spread so they do not all lie on one spot.
func (w *World) dropAt(x, y float64, items []Ingredient) {
	r := w.random()
	for i, it := range items {
		if !it.Item.Valid() || it.Count <= 0 {
			continue
		}
		// Stacks bigger than one slot holds fall as several piles.
		for left := it.Count; left > 0; {
			n := min(left, it.Item.MaxCount())
			left -= n
			px, py := x, y
			for range 10 {
				a := r.Float64()*2*math.Pi + float64(i)
				d := 0.3 + r.Float64()*DropScatter
				cx, cy := x+math.Cos(a)*d, y+math.Sin(a)*d*0.8
				if w.walkableAt(cx, cy) {
					px, py = cx, cy
					break
				}
			}
			w.nextDrop++
			w.drops = append(w.drops, &Drop{ID: w.nextDrop, Item: it.Item, Count: n, X: px, Y: py, FromX: x, FromY: y, Born: w.tick})
		}
	}
}

// stepDrops picks up the piles players walk over and rots the old ones.
func (w *World) stepDrops() {
	if len(w.drops) == 0 {
		return
	}
	for _, d := range w.drops {
		if w.tick-d.Born > dropLife(d.Item) {
			d.gone = true
			continue
		}
		w.settleToss(d)
		if w.tick-d.Born < DropPickupDelay {
			continue
		}
		for _, id := range w.playerOrder {
			p := w.players[id]
			if id == d.tosser || p.Dead || p.Asleep || math.Abs(p.X-d.X) > DropTouch || math.Abs(p.Y-d.Y) > DropTouch {
				continue
			}
			if w.pickUp(p, d, false) {
				break
			}
		}
	}
	w.drops = slices.DeleteFunc(w.drops, func(d *Drop) bool { return d.gone })
}

// pickUp moves as much of d as fits into p's bag and reports whether any
// did. loud says whether a full bag is worth a notice.
func (w *World) pickUp(p *Player, d *Drop, loud bool) bool {
	n := p.Inv.Add(d.Item, d.Count)
	if n == 0 {
		if loud {
			p.notify(NoticeBagFull, uint8(d.Item))
		}
		return false
	}
	d.Count -= n
	if d.Count <= 0 {
		d.gone = true
	}
	p.notify(NoticePickedUp, uint8(d.Item))
	return true
}

// DropCandidate is what targeting needs to know about a pile.
type DropCandidate struct {
	ID   DropID
	X, Y float64
}

// NearestDrop is the pile Space picks up: the nearest within DropReach of
// (x, y), the lower id on a tie. The server and the client both use it.
func NearestDrop(x, y float64, drops []DropCandidate) (DropID, bool) {
	var best DropID
	bestD := DropReach * DropReach
	found := false
	for _, d := range drops {
		dd := (d.X-x)*(d.X-x) + (d.Y-y)*(d.Y-y)
		if dd < bestD || (dd == bestD && found && d.ID < best) {
			best, bestD, found = d.ID, dd, true
		}
	}
	return best, found
}

func (w *World) dropCandidates(x, y float64) []DropCandidate {
	var out []DropCandidate
	for _, d := range w.drops {
		if !d.gone && math.Abs(d.X-x) <= DropReach && math.Abs(d.Y-y) <= DropReach {
			out = append(out, DropCandidate{ID: d.ID, X: d.X, Y: d.Y})
		}
	}
	return out
}

// pickNearest is Space on the ground: it picks up the nearest pile in
// reach, if there is one.
func (w *World) pickNearest(p *Player) (acted, found bool) {
	id, ok := NearestDrop(p.X, p.Y, w.dropCandidates(p.X, p.Y))
	if !ok {
		return false, false
	}
	for _, d := range w.drops {
		if d.ID == id {
			p.Dir = dirToward(d.X-p.X, d.Y-p.Y)
			return w.pickUp(p, d, true), true
		}
	}
	return false, false
}

// --- kills and bodies --------------------------------------------------------

// leavesBody reports whether a creature of kind k leaves a body to butcher
// when killed; the rest drop what they give at once.
func leavesBody(k CreatureKind) bool {
	switch k {
	case CreatureWorm, CreatureFish, CreatureNone:
		return false
	}
	return k < NumCreatureKinds
}

// butcherHits is how many blows it takes to butcher a body of kind k.
func butcherHits(k CreatureKind) int {
	return min(6, max(1, (k.def().hp+3)/4))
}

// fell is what happens when c is killed: its body falls where it stood, or
// for small things what it gives drops at once.
func (w *World) fell(c *Creature) {
	c.gone = true
	if !leavesBody(c.Kind) {
		w.dropAt(c.X, c.Y, c.Kind.loot())
		return
	}
	cx, cy := int(math.Floor(c.X)), int(math.Floor(c.Y))
	best, bestD := [2]int{}, math.Inf(1)
	for ty := cy - 2; ty <= cy+2; ty++ {
		for tx := cx - 2; tx <= cx+2; tx++ {
			if !w.canPlace(tx, ty) {
				continue
			}
			if d := math.Hypot(float64(tx)+0.5-c.X, float64(ty)+0.5-c.Y); d < bestD {
				best, bestD = [2]int{tx, ty}, d
			}
		}
	}
	if math.IsInf(bestD, 1) {
		w.dropAt(c.X, c.Y, w.bodyLoot(c.Kind)) // nowhere to lie: it is butchered where it fell
		return
	}
	e := w.addEntity(EntityCorpse, best[0], best[1])
	e.Var = uint8(c.Kind)
	e.Big = c.FaceLeft // the way it was facing when it fell
	e.HP = butcherHits(c.Kind) * HitPower
	e.Ready = true
	e.RegrowAt = w.tick + CorpseRotTicks
	w.active[e.ID] = e
}

// bodyLoot is everything a butchered body of kind k gives: its loot, its
// head, and dinosaur hide from a dinosaur.
func (w *World) bodyLoot(k CreatureKind) []Ingredient {
	if k == CreatureTribesman {
		return w.tribesmanLoot()
	}
	out := k.loot()
	if n := dinoHides(k); n > 0 {
		out = append(out, Ingredient{Item: ItemDinoHide, Count: n})
	}
	if h := HeadItem(k); h.Valid() {
		out = append(out, Ingredient{Item: h, Count: 1})
	}
	return out
}

// dinoHides is how much dinosaur hide a body of kind k gives.
func dinoHides(k CreatureKind) int {
	switch k {
	case CreatureTrike, CreatureStego, CreatureBrachio, CreatureRex, CreatureToro, CreatureQuetzal, CreaturePlesio:
		return DinoHideFromBig
	case CreatureRaptor, CreatureAchero, CreatureThesc, CreatureOrnitho, CreatureAnzu, CreaturePtero:
		return 1
	}
	return 0
}

// butcher is one blow of work on a body: with a knife in hand it goes
// faster. The last blow spills what the body gives on the ground.
func (w *World) butcher(p *Player, e *Entity) bool {
	dmg := HitPower
	if _, ok := SwordTier(p.Held()); ok {
		dmg *= ButcherKnifeBits
	}
	e.HP -= dmg
	if e.HP > 0 {
		return true
	}
	k := CreatureKind(e.Var)
	x, y := float64(e.TX)+0.5, float64(e.TY)+0.5
	w.removeEntity(e)
	w.dropAt(x, y, w.bodyLoot(k))
	return true
}

// --- heads -------------------------------------------------------------------

// HeadItem is the trophy head of a beast of kind k, or ItemNone if it has
// none (worms, fish and people do not).
func HeadItem(k CreatureKind) ItemKind {
	switch k {
	case CreatureNone, CreatureWorm, CreatureFish, CreatureTribesman:
		return ItemNone
	}
	if k >= NumCreatureKinds {
		return ItemNone
	}
	return ItemHeadBase + ItemKind(k)
}

// HeadOf is the creature kind whose head k is; ok is false if k is no head.
func HeadOf(k ItemKind) (CreatureKind, bool) {
	if k < ItemHeadBase || k >= NumItems {
		return CreatureNone, false
	}
	c := CreatureKind(k - ItemHeadBase)
	return c, HeadItem(c) == k
}

func init() {
	if int(NumCreatureKinds) > maxHeads || itemsEnd > ItemHeadBase {
		panic("sim: no room left for the heads of every creature kind")
	}
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		if h := ItemHeadBase + ItemKind(k); HeadItem(k) == h {
			itemDefs[h] = ItemDef{Name: "Đầu " + lowerFirst(k.Name()), Stack: 5}
		}
	}
}

// lowerFirst lower-cases the first letter of a Vietnamese common noun
// ("Khủng long ba sừng"), leaving proper names ("Dakotaraptor") alone.
func lowerFirst(s string) string {
	if strings.ContainsRune(s, ' ') || strings.HasPrefix(s, "Thỏ") || strings.HasPrefix(s, "Rắn") ||
		strings.HasPrefix(s, "Cua") || strings.HasPrefix(s, "Gấu") || strings.HasPrefix(s, "Nhện") {
		r, n := utf8.DecodeRuneInString(s)
		return string(unicode.ToLower(r)) + s[n:]
	}
	return s
}

// placeTrophy puts the head held in slot i on a post in front of p.
func (w *World) placeHeld(p *Player) { w.placeHeldAt(p, false, 0, 0) }

// placeHeldAt is placeHeld; with at set, a torch, head or seed goes on tile
// (atX, atY) if it passes BuildSpot, and dung on the growing thing there;
// a boat is still launched in front. See aim.go.
func (w *World) placeHeldAt(p *Player, at bool, atX, atY int) {
	held := p.Held()
	if at {
		if k, soft, ok := PlacedBy(held); ok && !w.buildSpot(p, atX, atY, k, soft) {
			return
		}
		if held == ItemDung {
			w.fertilizeAt(p, atX, atY)
			return
		}
	}
	tx, ty, ok := w.placeSpot(p)
	if _, _, placed := PlacedBy(held); at && placed {
		tx, ty, ok = atX, atY, true // checked above
	}
	place := func(kind EntityKind) *Entity {
		if !ok {
			p.notify(NoticeNoSpace, 0)
			return nil
		}
		p.Inv.TakeSlot(p.Selected, 1)
		return w.addEntity(kind, tx, ty)
	}
	if c, isHead := HeadOf(held); isHead {
		if e := place(EntityTrophy); e != nil {
			e.Var = uint8(c)
		}
		return
	}
	if k := boatOf(held); k != BoatNone {
		w.launch(p, k) // see boats.go
		return
	}
	if held == ItemDung {
		w.fertilize(p) // see dung.go
		return
	}
	switch held {
	case ItemTorch:
		if e := place(EntityTorchStand); e != nil {
			e.Fuel = TorchStandTicks
			w.watch(e)
		}
		if p.Held() != ItemTorch {
			p.torchFuel = 0 // the one burning in the hand went into the ground
		}
	case ItemFlowerSeed, ItemBerrySeed, ItemTreeSeed, ItemGlowSeed, ItemManEaterSeed:
		if !ok || !w.TileAt(tx, ty).Soft() {
			p.notify(NoticeNoSpace, 0)
			return
		}
		if e := place(EntitySapling); e != nil {
			e.Var = uint8(seedOf(held))
			e.Ready = false
			e.RegrowAt = w.tick + seedOf(held).growTicks()
			w.watch(e)
		}
	default:
		p.notify(NoticeCantPlace, uint8(held))
	}
}
