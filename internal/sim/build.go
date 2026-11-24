package sim

import "math"

// Building a home and keeping it safe:
//
//   - A wooden door sits in a wall like any other piece of it, but players
//     walk through it; beasts and the tribes do not.
//   - Wooden floor is planks on the ground: it changes nothing but the look.
//   - A fence of sharpened stakes blocks the way and wounds any beast that
//     presses against it.
//   - A snare holds the first beast that steps in it for a while (a big one
//     only briefly) and hurts it; Space sets a sprung snare again.
//   - A torch stuck in the ground (V with a torch in hand) burns for
//     minutes, lighting and warming like a small campfire; Space pulls it
//     out again while it still has life in it.
//   - The last bed slept in is where a player comes back after dying.
//
// And to mend: a healing draught brewed at the fire, a bigger one, and a
// bandage of spider silk that also draws out poison.

// Building tuning.
const (
	SpikesHits      = 4
	SpikeDamage     = 1 // to a beast against the stakes, every SpikeEvery ticks
	SpikeEvery      = TicksPerSecond
	SpikeReach      = 0.45                // a beast's edge this close to the stakes is pricked
	TrapHoldTicks   = 15 * TicksPerSecond // a small beast struggles this long
	TrapHoldTough   = 4 * TicksPerSecond  // a big one breaks free sooner
	TrapDamage      = 2
	TorchStandTicks = 4 * 60 * TicksPerSecond
	TorchStandReach = 3.5 // tiles of light and warmth
	ManEaterHits    = 10

	PotionHealth    = 45.0
	BigPotionHealth = 100.0
	BandageHealth   = 20.0
)

// blocksPlayers reports whether a structure of kind k stops a player.
func blocksPlayers(k EntityKind) bool {
	switch k {
	case EntityWall, EntityStoneWall, EntitySpikes, EntityHut, EntityTotem, EntityPyramid, EntityPyramidStone, EntityCaveRock,
		EntityGiantTree, EntityGiantRoot:
		return true
	}
	return false
}

// blocksBeasts reports whether a structure of kind k stops a creature.
func blocksBeasts(k EntityKind) bool {
	return blocksPlayers(k) || k == EntityDoor || k == EntityBonfire
}

// beastBlocked is blocked for creatures: doors and village fires stop them too.
func (w *World) beastBlocked(tx, ty int) bool {
	e := w.entityAt(tx, ty)
	return e != nil && blocksBeasts(e.Kind)
}

// stepTrapsAndSpikes runs the stakes and snares around c, a few times a second.
func (w *World) stepTrapsAndSpikes(c *Creature) {
	d := c.Kind.def()
	if d.flies || d.water || d.deep || c.Tamed() || c.State.OutOfReach() {
		return
	}
	if c.State == CreatureTrapped {
		return
	}
	// A snare under its feet.
	if e := w.entityAt(int(math.Floor(c.X)), int(math.Floor(c.Y))); e != nil && e.Kind == EntityTrap && e.Ready {
		e.Ready = false
		hold := TrapHoldTicks
		if d.tough {
			hold = TrapHoldTough
		}
		c.State, c.timer = CreatureTrapped, hold
		if c.HP -= TrapDamage * HealthScale; c.HP <= 0 {
			w.fell(c)
		}
		return
	}
	if (w.tick+uint64(c.ID))%SpikeEvery != 0 {
		return
	}
	w.manEaterBitesBeast(c)
	if c.gone {
		return
	}
	r := d.radius + SpikeReach
	for ty := int(math.Floor(c.Y - r)); ty <= int(math.Floor(c.Y+r)); ty++ {
		for tx := int(math.Floor(c.X - r)); tx <= int(math.Floor(c.X+r)); tx++ {
			e := w.entityAt(tx, ty)
			if e == nil || e.Kind != EntitySpikes {
				continue
			}
			nx := math.Max(float64(tx), math.Min(c.X, float64(tx)+1))
			ny := math.Max(float64(ty), math.Min(c.Y, float64(ty)+1))
			if math.Hypot(nx-c.X, ny-c.Y) <= r {
				if c.HP -= SpikeDamage * HealthScale; c.HP <= 0 {
					w.fell(c)
				} else if !d.tough && c.State != CreatureFlee {
					c.State, c.timer = CreatureHurt, CreatureHurtTicks
				}
				return
			}
		}
	}
}

// stepTrapped is a tick of a beast caught in a snare.
func (w *World) stepTrapped(c *Creature) {
	if c.timer--; c.timer <= 0 {
		c.State, c.timer = CreatureIdle, 0
	}
}

// takeTorch pulls a standing torch out of the ground: back into the bag as
// a torch while it has more than half its life, gone otherwise.
func (w *World) takeTorch(p *Player, e *Entity) bool {
	if e.Fuel*2 >= TorchStandTicks {
		if p.Inv.Add(ItemTorch, 1) == 0 {
			p.notify(NoticeBagFull, uint8(ItemTorch))
			return false
		}
	}
	w.removeEntity(e)
	return true
}

// dismantle takes down something a player built with the right tool,
// spilling what it gives back on the ground.
func (w *World) dismantle(p *Player, e *Entity, tool ToolClass, refund []Ingredient) bool {
	t, ok := p.Inv.BestTool(tool)
	if !ok {
		p.notify(NoticeNeedTool, uint8(ToolItem(tool, TierStone)))
		return false
	}
	e.HP -= ToolPower(t)
	if e.HP > 0 {
		return true
	}
	x, y := float64(e.TX)+0.5, float64(e.TY)+0.5
	w.removeEntity(e)
	w.dropAt(x, y, refund)
	return true
}

// homeBed is where p comes back after dying: beside the last bed slept in,
// while it stands, else the world's spawn.
func (w *World) homeBed(p *Player) (x, y float64) {
	if b := w.entities[p.home]; b != nil && b.Kind == EntityBed {
		cx, cy := float64(b.TX)+0.5, float64(b.TY)+0.5
		for _, d := range [][2]float64{{0, 0}, {0, 1}, {1, 0}, {-1, 0}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
			if w.standable(cx+d[0], cy+d[1]) {
				return cx + d[0], cy + d[1]
			}
		}
	}
	p.home = 0
	return w.spawnX, w.spawnY
}

// HomeBed is the bed p comes back to, 0 if none.
func (p *Player) HomeBed() EntityID { return p.home }

// cure draws the poison out of p.
func (p *Player) cure() { p.poison = 0 }
