package sim

// Most entities sit still: a full-grown tree or a boulder needs nothing done
// to it from one tick to the next. A big world has a hundred thousand of
// them, so stepEntities only looks at the few that do something over time:
// what is regrowing, burning fires and furnaces, tilled soil waiting to be
// planted or to go back to grass. That set is kept here.

// activeSweepTicks is how often every entity is checked anyway, in case
// some code changed one (made it not Ready, lit it) without telling watch.
const activeSweepTicks = 10 * TicksPerSecond

// needsTick reports whether stepEntities has work to do on e over time.
func needsTick(e *Entity) bool {
	switch e.Kind {
	case EntityCampfire, EntityFarmland, EntityCorpse, EntityTorchStand:
		return true
	case EntityFurnace:
		return e.Fuel > 0
	case EntityTreasure, EntityTrap, EntityBonfire, EntityTotem, EntityHut, EntityManEater, EntityPyramid, EntityPyramidStone:
		return false // an open chest or a sprung snare waits for a player
	}
	return !e.Ready
}

// watch adds e to the active set if it needs ticking. Call it whenever an
// entity is added or changed in a way that starts something over time.
func (w *World) watch(e *Entity) {
	if w.entities[e.ID] == e && needsTick(e) {
		w.active[e.ID] = e
	}
}

// watchAll rebuilds the active set from every entity.
func (w *World) watchAll() {
	for _, e := range w.entities {
		w.watch(e)
	}
}
