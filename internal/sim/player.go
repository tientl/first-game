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
	Look   Look    // body and hair, chosen when the character was made
	X, Y   float64 // centre, in tiles
	Dir    Dir
	Health float64
	Hunger float64
	Inv    Inventory
	// Selected is the hotbar slot the player holds, 0 to HotbarSlots-1.
	Selected int
	Dead     bool
	Cold     bool // freezing this tick
	Asleep   bool // in bed; see stepAsleep

	// The latest notice for this player. NoticeSeq increases on every new
	// notice so the client can tell a repeat of the same message apart.
	Notice    NoticeKind
	NoticeArg uint8
	NoticeSeq uint32

	explored *Explored // the tiles this player has seen, for the map

	input     Input
	actionCD  int        // ticks until the next action is allowed
	actedAt   uint64     // tick of the last action; 0 = never
	torchFuel int        // ticks left on the torch being burnt; 0 = none lit
	bed       EntityID   // the bed slept in
	line      line       // the fishing line, while it is out; see fishing.go
	pressed   bool       // Action went down since the last tick, even if already released
	picked    EntityID   // the fruit tree this hold of Action picked: not chopped till Action is pressed again
	chest     EntityID   // the chest open, 0 when none; see chest.go
	mount     CreatureID // the beast being ridden, 0 when on foot; see tame.go
	home      EntityID   // the last bed slept in: where p comes back after dying; see build.go
	poison    int        // ticks of spider venom left; see cavelife.go
	webbed    int        // ticks left tangled in web
	bees      int        // ticks a swarm keeps after them; see jungle.go
	harm      float64    // beasts' blows on them are multiplied by this: the difficulty; 0 means 1
	hurtSeq   uint8      // blows taken that make them jolt; see flinch in gear.go
	deed      Deed       // the last thing done that others see done; see deeds.go
	deedItem  ItemKind
	deedSeq   uint8
	beeHive   [2]float64 // where the angry bees came from
	boat      BoatID     // the boat p is aboard, 0 if none; see boats.go
	swim      int        // ticks spent swimming without rest
	travelCD  int        // ticks before a cave mouth takes p again
	cook      cookPlayer // the cooking station open, and the buffs of dishes; see cooking.go
	dev       devPlayer  // dev mode: god mode and speed, never saved, and free build, kept in a dev save; see dev.go, devbuild.go
}

// Acting reports whether the player did an action recently enough to show
// the working pose at tick.
func (p *Player) Acting(tick uint64) bool {
	return p.actedAt != 0 && tick-p.actedAt < ActingTicks
}

// Held returns the kind of item in the selected hotbar slot, or ItemNone.
func (p *Player) Held() ItemKind {
	if s := p.Inv[p.Selected]; !s.Empty() {
		return s.Item
	}
	return ItemNone
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
	p.wake()
	p.torchFuel = 0
	p.actionCD = 0
	p.actedAt = 0
	p.line = line{}
	p.poison, p.webbed = 0, 0
	p.boat, p.swim = 0, 0
	p.cook = cookPlayer{}
	// input and Selected are kept: they mirror what the client is holding
	// right now, and the client only resends them when they change.
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
func (w *World) AddPlayer(name string) PlayerID { return w.AddPlayerWithLook(name, Look{}) }

// AddPlayerWithLook is AddPlayer for a player who made a character that
// looks like l. A player coming back to a saved world keeps the saved look.
func (w *World) AddPlayerWithLook(name string, l Look) PlayerID {
	w.nextPlayer++
	id := w.nextPlayer
	name = cleanName(name)
	if name == "" {
		name = "Người chơi " + strconv.Itoa(int(id))
	}
	p := &Player{ID: id, Name: name, Look: l.Sanitized()}
	p.harm = w.difficulty.tune().harm
	p.reset(w.spawnX, w.spawnY)
	p.explored = NewExplored(w.width, w.height)
	if ps, ok := w.offline[name]; ok {
		w.restorePlayer(p, ps)
		delete(w.offline, name)
	}
	p.explored.Reveal(p.X, p.Y)
	w.players[id] = p
	w.playerOrder = append(w.playerOrder, id) // ids only grow, so this stays sorted
	return id
}

// RemovePlayer deletes a player. Unknown ids are ignored.
func (w *World) RemovePlayer(id PlayerID) {
	p, ok := w.players[id]
	if !ok {
		return
	}
	if p.mount != 0 {
		w.dismount(p)
	}
	w.leaveBoat(p)
	w.offline[p.Name] = savePlayer(p) // kept so the player resumes on rejoin
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
		in = in.sanitized()
		if in.Action && !p.input.Action {
			p.pressed = true // kept for the next step, so a quick tap is not lost
		}
		if !in.Action || !p.input.Action {
			p.picked = 0 // let go, or pressed afresh
		}
		p.input = in
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
			w.eat(p, int(cmd.Slot))
		}
	case CmdSelect:
		if IsHotbar(int(cmd.Slot)) {
			p.Selected = int(cmd.Slot) // allowed while dead: it is only which slot is picked
		}
	case CmdMoveSlot:
		if !p.Dead {
			p.Inv.Move(int(cmd.Slot), int(cmd.To))
		}
	case CmdQuickMove:
		if !p.Dead {
			p.Inv.QuickMove(int(cmd.Slot))
		}
	case CmdCraft:
		if !p.Dead {
			w.craftAt(p, cmd.Recipe, cmd.At, int(cmd.TX), int(cmd.TY))
		}
	case CmdChestPut, CmdChestTake, CmdChestClose, CmdChestTakeDown:
		if !p.Dead {
			w.chestCommand(p, cmd)
		}
	case CmdCookPut, CmdCookTake, CmdCookClose, CmdCookTakeDown, CmdCookStoke:
		if !p.Dead {
			w.cookCommand(p, cmd) // see cooking.go
		}
	case CmdRespawn:
		if p.Dead {
			// At the basic difficulty everything comes back with you; at the
			// harder ones it stays with your body where you fell (grave.go).
			kept := p.Inv
			if !w.difficulty.tune().keepItems && w.leaveBody(p) {
				kept = Inventory{}
			}
			p.reset(w.homeBed(p)) // by the last bed slept in, if it stands
			p.Inv = kept
		}
	case CmdPlace:
		if !p.Dead && !p.Asleep {
			w.placeHeldAt(p, cmd.At, int(cmd.TX), int(cmd.TY)) // see drops.go
		}
	case CmdInteract:
		w.interactKey(p) // see aim.go
	case CmdFeed:
		w.feedKey(p)
	case CmdRide:
		if !p.Dead && !p.Asleep && !w.board(p) { // a boat first; see boats.go
			w.ride(p) // see tame.go
		}
	case CmdVictorySeen:
		w.victorySeen(p) // see bosses.go
	case CmdToss:
		w.toss(p, int(cmd.Slot), int(cmd.Count)) // see toss.go
	case CmdDevBuild:
		if !p.Dead && !p.Asleep {
			w.devBuild(p, int(cmd.Slot), cmd.At, int(cmd.TX), int(cmd.TY)) // free build only; see devbuild.go
		}
	case CmdDevRemove:
		if !p.Dead && !p.Asleep {
			w.devRemove(p, cmd.At, int(cmd.TX), int(cmd.TY))
		}
	}
}

func (w *World) stepPlayer(p *Player) {
	pressed := p.pressed
	p.pressed = false
	w.keepChest(p)
	w.keepCook(p)
	if p.mount != 0 && (p.Dead || p.Asleep) {
		w.dismount(p)
	}
	if p.Dead {
		return
	}
	if p.Asleep {
		w.stepAsleep(p)
		return
	}

	moving := p.input.MoveX != 0 || p.input.MoveY != 0
	swims := w.style >= StyleSeas // in an archipelago anyone can swim
	if p.boat != 0 && w.sailStep(p) {
		// aboard: the boat carries p; see boats.go
	} else if p.mount != 0 && w.rideStep(p, moving) {
		// riding: the mount carries p; see tame.go
	} else if mx, my := float64(p.input.MoveX), float64(p.input.MoveY); moving {
		l := math.Hypot(mx, my)
		sp := PlayerSpeed * w.speedAt(p.X, p.Y) * p.dev.speedFactor() * p.buffSpeed()
		if p.webbed > 0 {
			sp *= WebSlow
		}
		if swims && w.inWater(p.X, p.Y) {
			sp = PlayerSpeed * SwimSpeed * swimBoost(&p.Inv) * p.dev.speedFactor() // see legendary.go
		}
		dx, dy := mx/l*sp, my/l*sp
		can := w.standable
		if swims {
			can = w.swimmable
		}
		// Resolve each axis on its own so the player slides along coasts.
		if can(p.X+dx, p.Y) {
			p.X += dx
		}
		if can(p.X, p.Y+dy) {
			p.Y += dy
		}
		p.Dir = dirToward(mx, my)
	}
	w.travel(p)   // through a cave mouth; see underground.go
	w.caveStep(p) // the natural caves' webs and lava; see caveprops.go
	p.explored.Reveal(p.X, p.Y)

	w.faceAim(p) // the player faces the mouse; see aim.go

	if p.actionCD > 0 {
		p.actionCD--
	}
	if p.line.state != CastNone {
		w.stepLine(p, pressed, moving) // fishing: Action strikes instead
	} else if (p.input.Action || pressed) && p.actionCD == 0 && !w.Swimming(p) { // a swimmer's hands are busy
		acted, found, cd := w.act(p) // see aim.go
		if found {
			if acted {
				p.actedAt = w.tick // a refused action (no axe, full bag) is no swing
			}
			p.actionCD = cd
		}
	}

	w.burnTorch(p)
	p.Hunger = max(0, p.Hunger-HungerPerTick*w.difficulty.tune().hunger*p.buffHunger())
	w.stepPoison(p) // see cavelife.go
	w.stepBees(p)   // see jungle.go
	w.stepSwim(p)   // see boats.go
	w.manEaterBites(p)
	p.Cold = (w.winterNight() || w.freezing(p) || w.wintry(p)) && !w.warm(p.X, p.Y) && !NeverCold(&p.Inv) // a winter night, the snow, winter weather: see iceage.go, weather.go, legendary.go
	p.devNeeds()                                                                                          // god mode: see dev.go
	// What a dish gave: its buffs run down, and a hot soup keeps the cold out. See cooking.go.
	p.stepBuffs()
	damage := 0.0
	if p.Hunger == 0 {
		damage += StarveDamagePerTick
	}
	if p.Cold {
		damage += ColdDamagePerTick * w.difficulty.tune().cold
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

// findTarget returns the entity or creature the player's Action would hit
// (piles and the hoe aside): see targetFor in aim.go.
func (w *World) findTarget(p *Player) (*Entity, *Creature) { return w.targetFor(p) }

// TargetCandidate is what targeting needs to know about the entity on a tile.
type TargetCandidate struct {
	ID    EntityID
	Kind  EntityKind
	Ready bool
	Var   uint8 // tells a fruit tree from its stump
}

// targetable reports whether c can be acted on: as Interactable says, except
// that a standing fruit tree is a target with fruit (to pick) or without (to
// fell), and its stump is not.
func (c TargetCandidate) targetable(hasWood bool) bool {
	if IsFruitTree(c.Kind) {
		return !IsFruitStump(c.Kind, c.Var)
	}
	if StationOf(c.Kind, c.Var) != StationNone {
		return true // a fire with a pot on it is opened, wood in the bag or not; see cooking.go
	}
	return Interactable(c.Kind, c.Ready, hasWood)
}

// NearestTarget is the targeting rule: the nearest interactable entity whose
// tile centre is within InteractReach of (x, y), ties going to the lower id.
// The server uses it to resolve Action and the client to highlight the same
// entity; at returns the entity on a tile, ok=false when the tile is empty.
func NearestTarget(x, y float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool)) (EntityID, bool) {
	id, _, ok := nearestEntity(x, y, hasWood, at)
	return id, ok
}

// nearestEntity is NearestTarget that also returns the squared distance.
func nearestEntity(x, y float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool)) (EntityID, float64, bool) {
	return nearestEntityWeighed(x, y, hasWood, at, nil)
}

// nearestEntityWeighed is nearestEntity with the distances weighed by
// weigh, if not nil; reach is still judged by the true distance.
func nearestEntityWeighed(x, y float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool), weigh func(dx, dy, d2 float64) float64) (EntityID, float64, bool) {
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
			if !ok || !c.targetable(hasWood) {
				continue
			}
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			d := dx*dx + dy*dy
			if d > InteractReach*InteractReach {
				continue
			}
			if weigh != nil {
				d = weigh(dx, dy, d)
			}
			if !found || d < bestDist || (d == bestDist && c.ID < bestID) {
				bestID, bestDist, found = c.ID, d, true
			}
		}
	}
	return bestID, bestDist, found
}

// interact performs one hit of work by p on e and reports whether any work
// was done (false: refused, the player does not swing).
func (w *World) interact(p *Player, e *Entity) bool {
	free := w.freeBuild(p) // one blow fells, breaks or takes down anything, tool or not: see devbuild.go
	if o, ok := oreOf(e.Kind); ok {
		return w.mine(p, e, o, free)
	}
	// withTool is the damage of p's best tool of class c, or 0 (refused,
	// with a notice) when p has none.
	withTool := func(c ToolClass) int {
		if free {
			return max(e.HP, 1)
		}
		if c == ToolAxe && legendaryChop(&p.Inv) > 0 {
			return legendaryChop(&p.Inv) // the fang axe: see legendary.go
		}
		t, ok := p.Inv.BestTool(c)
		if !ok {
			p.notify(NoticeNeedTool, uint8(ToolItem(c, TierStone)))
			return 0
		}
		return ToolPower(t)
	}
	switch e.Kind {
	case EntityTree:
		if w.takeHoney(p, e) { // see jungle.go
			return true
		}
		dmg := withTool(ToolAxe)
		return dmg > 0 && w.knock(e, dmg, func(x, y float64) {
			if e.HasHive() {
				w.angerBees(p, e) // the hive comes down with it
				e.Var = 0
			}
			e.Ready = false // a stump, until a sapling comes up by it: see regrow.go
			e.RegrowAt = w.tick + StumpTicks
			w.dropAt(x, y, w.treeYield(e))
		})
	case EntityGiantTree:
		if w.takeHoney(p, e) {
			return true
		}
		dmg := withTool(ToolAxe)
		return dmg > 0 && w.fellGiant(p, e, dmg)
	case EntityBoulder:
		dmg := withTool(ToolPickaxe)
		return dmg > 0 && w.knock(e, dmg, func(x, y float64) {
			w.removeEntity(e)
			w.dropAt(x, y, w.boulderYield(e))
		})
	case EntityStalagmite, EntityCaveProp:
		return w.touchCave(p, e, withTool) // see caveprops.go
	case EntityCorpse:
		return w.butcher(p, e) // see drops.go
	case EntityTrophy:
		h := HeadItem(CreatureKind(e.Var))
		if p.Inv.Add(h, 1) == 0 {
			p.notify(NoticeBagFull, uint8(h))
			return false
		}
		w.removeEntity(e)
		return true
	case EntityDoor, EntityFloor:
		return w.dismantle(p, e, ToolAxe, []Ingredient{{Item: ItemWood, Count: 1}}, free)
	case EntitySpikes:
		return w.dismantle(p, e, ToolAxe, []Ingredient{{Item: ItemStick, Count: 1}}, free)
	case EntityTrap:
		if e.Ready {
			return w.dismantle(p, e, ToolAxe, []Ingredient{{Item: ItemStick, Count: 1}}, free)
		}
		e.Ready = true // set again
		return true
	case EntityTorchStand:
		return w.takeTorch(p, e)
	case EntityFlower:
		return w.pickFlower(p, e)
	case EntityGlowPlant:
		return w.pickGlow(p, e)
	case EntityManEater:
		return w.cutManEater(p, e)
	case EntityGrave:
		return w.lootBody(p, e) // see grave.go
	case EntityTreasure:
		return w.openTreasure(p, e)
	case EntityWall:
		dmg := withTool(ToolAxe)
		return dmg > 0 && w.harvest(p, e, ItemWood, WallRefund, dmg, func() { w.removeEntity(e) })
	case EntityStoneWall:
		dmg := withTool(ToolPickaxe)
		return dmg > 0 && w.harvest(p, e, ItemStone, StoneWallRefund, dmg, func() { w.removeEntity(e) })
	case EntityBed, EntityLeafTent:
		return w.useBed(p, e) // slept in, or taken down by day with an axe: see tent.go
	case EntityChest:
		return w.openChest(p, e)
	case EntityBerryBush:
		return w.harvest(p, e, ItemBerry, BushYield, HitPower, func() {
			e.Ready = false
			e.RegrowAt = w.tick + BushRegrowTicks
			if w.style >= StyleTribal && w.random().Float64() < BerrySeedChance {
				p.Inv.Add(ItemBerrySeed, 1)
			}
		})
	case EntityGrassTuft:
		return w.harvest(p, e, ItemGrass, TuftYield, HitPower, func() {
			e.Ready = false
			e.RegrowAt = w.tick + TuftRegrowTicks
		})
	case EntityClam:
		return w.harvest(p, e, ItemClam, ClamYield, HitPower, func() {
			e.Ready = false // the tide brings more
			e.RegrowAt = w.tick + ClamRegrowTicks
		})
	case EntityTwig:
		return w.harvest(p, e, ItemStick, TwigYield, HitPower, func() {
			e.Ready = false // gone until another branch falls here
			e.RegrowAt = w.tick + TwigRegrowTicks
		})
	case EntityPebble:
		return w.harvest(p, e, ItemPebble, PebbleYield, HitPower, func() {
			e.Ready = false // gone until the ground turns up another
			e.RegrowAt = w.tick + PebbleRegrowTicks
		})
	case EntityFarmland:
		return w.plant(p, e)
	case EntityCrop:
		return w.reap(p, e) // see flora.go
	case EntityFlora:
		dmg := HitPower
		if FloraKind(e.Var).def().axe {
			if dmg = withTool(ToolAxe); dmg == 0 {
				return false
			}
		}
		return w.pickFlora(p, e, dmg) // see flora.go
	case EntityCampfire:
		if e.Var != 0 {
			return w.openCook(p, e) // a pot, grill or pan on it; see cooking.go
		}
		return w.stoke(p, e, FireMaxFuel) // coal first: it burns longest
	case EntityMortar, EntityOven, EntityDryRack:
		return w.openCook(p, e)
	case EntityClay:
		return w.harvest(p, e, ItemClay, ClayYield, HitPower, func() {
			e.Ready = false // the water lays down more
			e.RegrowAt = w.tick + ClayRegrowTicks
		})
	case EntityFurnace:
		return w.stoke(p, e, FurnaceMaxFuel)
	case EntityCactus:
		dmg := withTool(ToolAxe)
		return dmg > 0 && w.harvest(p, e, ItemCactus, CactusYield, dmg, func() {
			e.Ready = false
			e.RegrowAt = w.tick + CactusRegrowTicks
		})
	case EntityFigTree, EntityGinkgoTree, EntityPalm, EntityBananaTree: // see fruit.go
		f, _ := fruitOf(e.Kind)
		switch {
		case e.FruitStump():
			return false
		case e.Ready:
			return w.pickFruit(p, e, f) // ripe fruit first, by hand
		case p.picked == e.ID:
			return false // the hold that picked it: a fresh press chops
		}
		dmg := withTool(ToolAxe)
		return dmg > 0 && w.fellFruit(e, f, dmg)
	case EntityReed:
		return w.harvest(p, e, ItemGrass, ReedYield, HitPower, func() {
			e.Ready = false
			e.RegrowAt = w.tick + ReedRegrowTicks
		})
	case EntityMushroom:
		return w.harvest(p, e, ItemMushroom, MushroomYield, HitPower, func() {
			e.Ready = false
			e.RegrowAt = w.tick + MushroomRegrowTick
		})
	case EntityCoal:
		dmg := withTool(ToolPickaxe)
		return dmg > 0 && w.harvest(p, e, ItemCoal, CoalYield, dmg, func() { w.removeEntity(e) })
	}
	return false
}

// harvest lands one hit on e. When the last hit lands, the yield goes to the
// player and done updates the entity. A hit is refused while the yield cannot
// fit, so a full bag never destroys resources.
func (w *World) harvest(p *Player, e *Entity, item ItemKind, yield, dmg int, done func()) bool {
	if p.Inv.Room(item) < yield {
		p.notify(NoticeBagFull, uint8(item))
		return false
	}
	e.HP -= dmg
	if e.HP > 0 {
		return true
	}
	p.Inv.Add(item, yield)
	done()
	w.watch(e) // it may now be regrowing, or turned into something that ticks
	return true
}

// eat eats one of what is in slot i.
func (w *World) eat(p *Player, i int) {
	if !ValidSlot(i) {
		return // garbage from the network
	}
	switch s := p.Inv[i]; {
	case s.Empty() || !s.Item.Edible():
		p.notify(NoticeNotEdible, uint8(s.Item))
	default:
		def := s.Item.Def()
		p.Inv.TakeSlot(i, 1)
		if def.Cures {
			if p.Poisoned() {
				p.notify(NoticeCured, uint8(s.Item))
			}
			p.cure() // draws the venom out
		}
		if def.Poison > 0 && !PoisonProof(&p.Inv) && !p.dev.god {
			p.poison = max(p.poison, def.Poison) // the wrong mushroom, the wrong berry
			p.notify(NoticeFoodPoison, uint8(s.Item))
		}
		p.Hunger = min(MaxHunger, p.Hunger+def.Hunger)
		// Raw food hurts a little, but a meal never kills.
		p.Health = min(MaxHealth, max(p.Health+def.Health, min(p.Health, 1)))
		w.ate(p, s.Item) // a dish's buffs, and its bowl back; see cooking.go
		p.did(DeedEat, s.Item)
	}
}

func (w *World) craft(p *Player, id RecipeID) { w.craftAt(p, id, false, 0, 0) }

// craftAt is craft; a structure goes on tile (tx, ty) when at is set, if
// that tile passes BuildSpot, else in front of p. In free build nothing is
// needed, no fire or furnace either, and nothing is used up (devbuild.go).
func (w *World) craftAt(p *Player, id RecipeID, at bool, tx, ty int) {
	r, ok := RecipeByID(id)
	if !ok {
		return
	}
	free := w.freeBuild(p)
	if free {
		r.Inputs = nil
	}
	if r.Gives.Valid() && r.Gives.Def().Tool && p.Inv.Count(r.Gives) > 0 {
		p.notify(NoticeHaveOne, uint8(r.Gives))
		return
	}
	if !p.Inv.HasAll(r.Inputs) {
		p.notify(NoticeNotEnough, uint8(id))
		return
	}
	if r.NearFire && !free && !w.nearFire(p.X, p.Y) {
		p.notify(NoticeNeedFire, uint8(id))
		return
	}
	if r.NearFurnace && !free && !w.nearFurnace(p.X, p.Y) {
		p.notify(NoticeNeedFurnace, uint8(id))
		return
	}
	if r.Gives.Valid() {
		// The ingredients go first, so the slot they free can take the result.
		after := p.Inv
		after.TakeAll(r.Inputs)
		if after.Add(r.Gives, 1) == 0 {
			p.notify(NoticeBagFull, uint8(r.Gives))
			return
		}
		p.Inv = after
		p.notify(NoticeCrafted, uint8(id))
		return
	}
	if at {
		if !w.buildSpot(p, tx, ty, r.Places, false) {
			return
		}
	} else if tx, ty, ok = w.placeSpot(p); !ok {
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

// knock lands one blow on a tree or boulder; the blow that brings it down
// calls done with where its yield falls.
func (w *World) knock(e *Entity, dmg int, done func(x, y float64)) bool {
	e.HP -= dmg
	if e.HP > 0 {
		return true
	}
	done(float64(e.TX)+0.5, float64(e.TY)+0.5)
	w.watch(e)
	return true
}

// treeYield is what a felled tree drops: its wood, often branches, and in a
// tribal world sometimes a seed.
func (w *World) treeYield(e *Entity) []Ingredient {
	r := w.random()
	out := []Ingredient{{Item: ItemWood, Count: bySize(e.Big, TreeYield, BigTreeYield)}}
	if r.Float64() < TwigChance {
		out = append(out, Ingredient{Item: ItemStick, Count: 1 + r.IntN(bySize(e.Big, 2, 3))})
	}
	if w.style >= StyleTribal && r.Float64() < TreeSeedChance {
		out = append(out, Ingredient{Item: ItemTreeSeed, Count: 1})
	}
	return out
}

// boulderYield is what a broken boulder drops: its stone, and sometimes a pebble.
func (w *World) boulderYield(e *Entity) []Ingredient {
	out := []Ingredient{{Item: ItemStone, Count: bySize(e.Big, BoulderYield, BigBoulderYield)}}
	if w.random().Float64() < PebbleChance {
		out = append(out, Ingredient{Item: ItemPebble, Count: 1})
	}
	return out
}
