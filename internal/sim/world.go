package sim

import (
	"math"
	"slices"
)

// World is the whole authoritative game state. It is not safe for concurrent
// use: the server owns it from a single goroutine.
type World struct {
	width, height int
	tiles         []Tile     // row-major, width*height
	occ           []EntityID // per tile: the entity on it, 0 if none

	tick     uint64
	entities map[EntityID]*Entity
	active   map[EntityID]*Entity    // the entities stepEntities looks at; see active.go
	chests   map[EntityID]*Inventory // what each chest holds; see chest.go
	nextEnt  EntityID

	players     map[PlayerID]*Player
	playerOrder []PlayerID // ascending; players step in this order
	nextPlayer  PlayerID

	spawnX, spawnY float64

	seed    uint64                // the GenConfig seed, kept for saving
	style   WorldStyle            // the GenConfig style, kept for saving
	offline map[string]PlayerSave // players who left, by name, restored on rejoin

	fauna fauna // animals; see creature.go
}

func newWorld(width, height int) *World {
	return &World{
		width:    width,
		height:   height,
		tiles:    make([]Tile, width*height),
		occ:      make([]EntityID, width*height),
		tick:     StartTick,
		entities: make(map[EntityID]*Entity),
		active:   make(map[EntityID]*Entity),
		chests:   make(map[EntityID]*Inventory),
		players:  make(map[PlayerID]*Player),
		offline:  make(map[string]PlayerSave),
	}
}

// Size returns the map size in tiles.
func (w *World) Size() (width, height int) { return w.width, w.height }

// Tick returns the current simulation tick.
func (w *World) Tick() uint64 { return w.tick }

// Spawn returns the point where players appear.
func (w *World) Spawn() (x, y float64) { return w.spawnX, w.spawnY }

// Tiles returns a copy of the tile map, row-major.
func (w *World) Tiles() []Tile { return slices.Clone(w.tiles) }

func (w *World) inBounds(tx, ty int) bool {
	return tx >= 0 && ty >= 0 && tx < w.width && ty < w.height
}

// TileAt returns the tile at (tx, ty); outside the map is water.
func (w *World) TileAt(tx, ty int) Tile {
	if !w.inBounds(tx, ty) {
		return TileWater
	}
	return w.tiles[ty*w.width+tx]
}

// EntityAt returns a copy of the entity on tile (tx, ty), if any.
func (w *World) EntityAt(tx, ty int) (Entity, bool) {
	e := w.entityAt(tx, ty)
	if e == nil {
		return Entity{}, false
	}
	return *e, true
}

func (w *World) entityAt(tx, ty int) *Entity {
	if !w.inBounds(tx, ty) {
		return nil
	}
	id := w.occ[ty*w.width+tx]
	if id == 0 {
		return nil
	}
	return w.entities[id]
}

// EntityCount returns how many entities exist.
func (w *World) EntityCount() int { return len(w.entities) }

// EachEntityIn calls fn with a copy of every entity whose tile lies in the
// inclusive rectangle [tx0, tx1] x [ty0, ty1], in row-major order.
func (w *World) EachEntityIn(tx0, ty0, tx1, ty1 int, fn func(Entity)) {
	tx0, ty0 = max(tx0, 0), max(ty0, 0)
	tx1, ty1 = min(tx1, w.width-1), min(ty1, w.height-1)
	for ty := ty0; ty <= ty1; ty++ {
		row := ty * w.width
		for tx := tx0; tx <= tx1; tx++ {
			if id := w.occ[row+tx]; id != 0 {
				fn(*w.entities[id])
			}
		}
	}
}

// canPlace reports whether a new entity may go on tile (tx, ty).
func (w *World) canPlace(tx, ty int) bool {
	return w.inBounds(tx, ty) && w.tiles[ty*w.width+tx].Walkable() && w.occ[ty*w.width+tx] == 0
}

// addEntity places a new entity of kind on (tx, ty). The caller must check
// canPlace first.
func (w *World) addEntity(kind EntityKind, tx, ty int) *Entity {
	w.nextEnt++
	e := &Entity{ID: w.nextEnt, Kind: kind, TX: tx, TY: ty, HP: maxHP(kind, false), Ready: true}
	switch kind {
	case EntityCampfire:
		e.Fuel = FireStartFuel
	case EntityChest:
		w.chests[e.ID] = &Inventory{}
	}
	w.entities[e.ID] = e
	w.occ[ty*w.width+tx] = e.ID
	w.watch(e)
	return e
}

func (w *World) removeEntity(e *Entity) {
	delete(w.entities, e.ID)
	delete(w.active, e.ID)
	delete(w.chests, e.ID)
	w.occ[e.TY*w.width+e.TX] = 0
}

// standable reports whether a player's collision box centred at (x, y) lies
// entirely on walkable tiles.
func (w *World) standable(x, y float64) bool {
	r := PlayerRadius
	return w.walkableAt(x-r, y-r) && w.walkableAt(x+r, y-r) &&
		w.walkableAt(x-r, y+r) && w.walkableAt(x+r, y+r)
}

func (w *World) walkableAt(x, y float64) bool {
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	return w.TileAt(tx, ty).Walkable() && !w.blocked(tx, ty)
}

// Step advances the world by one tick.
func (w *World) Step() {
	w.tick++
	for _, id := range w.playerOrder {
		w.stepPlayer(w.players[id])
	}
	w.stepEntities()
	w.stepCreatures()
}

func (w *World) stepEntities() {
	if w.tick%activeSweepTicks == 0 {
		w.watchAll() // the safety net; see active.go
	}
	for _, e := range w.active {
		if !needsTick(e) {
			delete(w.active, e.ID) // settled: nothing more to do over time
			continue
		}
		switch e.Kind {
		case EntityTree, EntityBerryBush, EntityGrassTuft, EntityTwig, EntityPebble, EntityCrop,
			EntityCactus, EntityReed, EntityMushroom, EntityClam, EntityFigTree, EntityGinkgoTree, EntityPalm:
			if !e.Ready && w.tick >= e.RegrowAt {
				e.Ready = true // for a crop: ripe
				e.HP = maxHP(e.Kind, e.Big)
			}
		case EntityFarmland:
			if w.tick >= e.RegrowAt {
				w.removeEntity(e) // left empty: grass again
			}
		case EntityCampfire:
			e.Fuel--
			if e.Fuel <= 0 {
				w.removeEntity(e) // deleting during range is safe in Go
			}
		case EntityFurnace:
			if e.Fuel > 0 {
				e.Fuel-- // a cold furnace stays, waiting for fuel
			}
		}
	}
}

// nearFire reports whether (x, y) is inside the light of any burning campfire,
// or beside a lit furnace: both warm and cook.
func (w *World) nearFire(x, y float64) bool {
	reach := int(math.Ceil(MaxFireRadius))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - reach; ty <= cy+reach; ty++ {
		for tx := cx - reach; tx <= cx+reach; tx++ {
			e := w.entityAt(tx, ty)
			if e == nil {
				continue
			}
			r, ok := FireReach(e.Kind, e.Fuel)
			if !ok {
				continue
			}
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			if dx*dx+dy*dy <= r*r {
				return true
			}
		}
	}
	return false
}
