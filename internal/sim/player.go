package sim

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PlayerID identifies a connected player. It is never reused within a world.
type PlayerID uint32

// MaxNameRunes caps player names.
const MaxNameRunes = 16

// Player is one survivor.
type Player struct {
	ID     PlayerID
	Name   string
	X, Y   float64 // centre, in tiles
	Dir    Dir
	Health float64
	Hunger float64
	Inv    Inventory
	Dead   bool
	Cold   bool // freezing this tick

	// The latest notice for this player. NoticeSeq increases on every new
	// notice so the client can tell a repeat of the same message apart.
	Notice    NoticeKind
	NoticeArg uint8
	NoticeSeq uint32

	input    Input
	actionCD int    // ticks until the next action is allowed
	actedAt  uint64 // tick of the last action; 0 = never
}

// Acting reports whether the player did an action recently enough to show
// the working pose at tick.
func (p *Player) Acting(tick uint64) bool {
	return p.actedAt != 0 && tick-p.actedAt < ActingTicks
}

func (p *Player) notify(kind NoticeKind, arg uint8) {
	p.Notice, p.NoticeArg = kind, arg
	p.NoticeSeq++
}

func (p *Player) reset(x, y float64) {
	p.X, p.Y = x, y
	p.Dir = DirDown
	p.Health = MaxHealth
	p.Hunger = SpawnHunger
	p.Inv = Inventory{}
	p.Dead = false
	p.Cold = false
	p.actionCD = 0
	p.actedAt = 0
	// input is kept: it mirrors what the client is holding right now, and the
	// client only resends it when it changes.
}

// cleanName trims a client-supplied name, drops invalid UTF-8 and control
// characters, and caps its length.
func cleanName(name string) string {
	name = strings.ToValidUTF8(name, "")
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxNameRunes {
		name = strings.TrimSpace(string([]rune(name)[:MaxNameRunes]))
	}
	return name
}

// AddPlayer creates a player at the spawn point and returns its id. An empty
// name is replaced by a generated one.
func (w *World) AddPlayer(name string) PlayerID {
	w.nextPlayer++
	id := w.nextPlayer
	name = cleanName(name)
	if name == "" {
		name = "Người chơi " + strconv.Itoa(int(id))
	}
	p := &Player{ID: id, Name: name}
	p.reset(w.spawnX, w.spawnY)
	w.players[id] = p
	w.playerOrder = append(w.playerOrder, id) // ids only grow, so this stays sorted
	return id
}

// RemovePlayer deletes a player. Unknown ids are ignored.
func (w *World) RemovePlayer(id PlayerID) {
	if _, ok := w.players[id]; !ok {
		return
	}
	delete(w.players, id)
	if i, found := slices.BinarySearch(w.playerOrder, id); found {
		w.playerOrder = slices.Delete(w.playerOrder, i, i+1)
	}
}

// Player returns a copy of the player with the given id.
func (w *World) Player(id PlayerID) (Player, bool) {
	p, ok := w.players[id]
	if !ok {
		return Player{}, false
	}
	return *p, true
}

// Players returns copies of all players in id order.
func (w *World) Players() []Player {
	out := make([]Player, 0, len(w.playerOrder))
	for _, id := range w.playerOrder {
		out = append(out, *w.players[id])
	}
	return out
}

// SetInput replaces the player's held input. Out-of-range values (the input
// may come from the network) are clamped. It is stored even while the player
// is dead, so a key held through a respawn keeps working.
func (w *World) SetInput(id PlayerID, in Input) {
	if p, ok := w.players[id]; ok {
		p.input = in.sanitized()
	}
}

// Apply runs a discrete command for the player. Invalid commands are ignored.
func (w *World) Apply(id PlayerID, cmd Command) {
	p, ok := w.players[id]
	if !ok {
		return
	}
	switch cmd.Kind {
	case CmdEat:
		if !p.Dead {
			w.eat(p, cmd.Item)
		}
	case CmdCraft:
		if !p.Dead {
			w.craft(p, cmd.Recipe)
		}
	case CmdRespawn:
		if p.Dead {
			p.reset(w.spawnX, w.spawnY)
		}
	}
}

func (w *World) stepPlayer(p *Player) {
	if p.Dead {
		return
	}

	if mx, my := float64(p.input.MoveX), float64(p.input.MoveY); mx != 0 || my != 0 {
		l := math.Hypot(mx, my)
		dx, dy := mx/l*PlayerSpeed, my/l*PlayerSpeed
		// Resolve each axis on its own so the player slides along coasts.
		if w.standable(p.X+dx, p.Y) {
			p.X += dx
		}
		if w.standable(p.X, p.Y+dy) {
			p.Y += dy
		}
		p.Dir = dirToward(mx, my)
	}

	if p.actionCD > 0 {
		p.actionCD--
	}
	if p.input.Action && p.actionCD == 0 {
		if e := w.findTarget(p); e != nil {
			p.Dir = dirToward(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y)
			w.interact(p, e)
			p.actionCD = ActionCooldownTicks
			p.actedAt = w.tick
		}
	}

	p.Hunger = max(0, p.Hunger-HungerPerTick)
	p.Cold = Ambient(w.tick) < ColdAmbient && !w.nearFire(p.X, p.Y)
	damage := 0.0
	if p.Hunger == 0 {
		damage += StarveDamagePerTick
	}
	if p.Cold {
		damage += ColdDamagePerTick
	}
	if damage > 0 {
		p.Health -= damage
	} else if p.Hunger >= WellFedHunger {
		p.Health = min(MaxHealth, p.Health+RegenPerTick)
	}
	if p.Health <= 0 {
		p.Health = 0
		p.Dead = true
		p.Cold = false
	}
}

// findTarget returns the entity the player's Action would hit, or nil.
func (w *World) findTarget(p *Player) *Entity {
	id, ok := NearestTarget(p.X, p.Y, p.Inv[ItemWood] > 0, func(tx, ty int) (TargetCandidate, bool) {
		e := w.entityAt(tx, ty)
		if e == nil {
			return TargetCandidate{}, false
		}
		return TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready}, true
	})
	if !ok {
		return nil
	}
	return w.entities[id]
}

// TargetCandidate is what targeting needs to know about the entity on a tile.
type TargetCandidate struct {
	ID    EntityID
	Kind  EntityKind
	Ready bool
}

// NearestTarget is the targeting rule: the nearest interactable entity whose
// tile centre is within InteractReach of (x, y), ties going to the lower id.
// The server uses it to resolve Action and the client to highlight the same
// entity; at returns the entity on a tile, ok=false when the tile is empty.
func NearestTarget(x, y float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool)) (EntityID, bool) {
	reach := int(math.Ceil(InteractReach))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	var (
		bestID   EntityID
		bestDist = math.Inf(1)
		found    bool
	)
	for ty := cy - reach; ty <= cy+reach; ty++ {
		for tx := cx - reach; tx <= cx+reach; tx++ {
			c, ok := at(tx, ty)
			if !ok || !Interactable(c.Kind, c.Ready, hasWood) {
				continue
			}
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			d := dx*dx + dy*dy
			if d > InteractReach*InteractReach {
				continue
			}
			if !found || d < bestDist || (d == bestDist && c.ID < bestID) {
				bestID, bestDist, found = c.ID, d, true
			}
		}
	}
	return bestID, found
}

// interact performs one hit of work by p on e.
func (w *World) interact(p *Player, e *Entity) {
	switch e.Kind {
	case EntityTree:
		w.harvest(p, e, ItemWood, TreeYield, func() {
			e.Ready = false // stump until it regrows
			e.RegrowAt = w.tick + TreeRegrowTicks
		})
	case EntityBoulder:
		w.harvest(p, e, ItemStone, BoulderYield, func() { w.removeEntity(e) })
	case EntityBerryBush:
		w.harvest(p, e, ItemBerry, BushYield, func() {
			e.Ready = false
			e.RegrowAt = w.tick + BushRegrowTicks
		})
	case EntityGrassTuft:
		w.harvest(p, e, ItemGrass, TuftYield, func() {
			e.Ready = false
			e.RegrowAt = w.tick + TuftRegrowTicks
		})
	case EntityCampfire:
		if p.Inv[ItemWood] == 0 {
			p.notify(NoticeNoneLeft, uint8(ItemWood))
			return
		}
		p.Inv[ItemWood]--
		e.Fuel = min(FireMaxFuel, e.Fuel+FireFuelPerWood)
	}
}

// harvest lands one hit on e. When the last hit lands, the yield goes to the
// player and done updates the entity. A hit is refused while the yield cannot
// fit, so a full bag never destroys resources.
func (w *World) harvest(p *Player, e *Entity, item ItemKind, yield int, done func()) {
	if p.Inv.Room(item) < yield {
		p.notify(NoticeBagFull, uint8(item))
		return
	}
	e.HP--
	if e.HP > 0 {
		return
	}
	p.Inv.Add(item, yield)
	done()
}

func (w *World) eat(p *Player, item ItemKind) {
	switch {
	case !item.Valid() || !item.Edible():
		p.notify(NoticeNotEdible, uint8(item))
	case p.Inv[item] == 0:
		p.notify(NoticeNoneLeft, uint8(item))
	default:
		def := item.Def()
		p.Inv[item]--
		p.Hunger = min(MaxHunger, p.Hunger+def.Hunger)
		p.Health = min(MaxHealth, p.Health+def.Health)
	}
}

func (w *World) craft(p *Player, id RecipeID) {
	r, ok := RecipeByID(id)
	if !ok {
		return
	}
	if !p.Inv.HasAll(r.Inputs) {
		p.notify(NoticeNotEnough, uint8(id))
		return
	}
	tx, ty, ok := w.placeSpot(p)
	if !ok {
		p.notify(NoticeNoSpace, uint8(id))
		return
	}
	p.Inv.TakeAll(r.Inputs)
	w.addEntity(r.Places, tx, ty)
	p.notify(NoticeCrafted, uint8(id))
}

// placeSpot picks the tile for a new structure: the one the player faces,
// else another free neighbour. The player's own tile is never used.
func (w *World) placeSpot(p *Player) (tx, ty int, ok bool) {
	px, py := int(math.Floor(p.X)), int(math.Floor(p.Y))
	order := []Dir{p.Dir, DirDown, DirRight, DirUp, DirLeft}
	for _, d := range order {
		dx, dy := d.Delta()
		if w.canPlace(px+dx, py+dy) {
			return px + dx, py + dy, true
		}
	}
	return 0, 0, false
}
