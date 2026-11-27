package sim

import "math"

// The things of the natural caves a player meets: stalagmites broken for
// stone, and the cave props (see cavethemes.go), and what the caves do to
// whoever walks in them. A torch set in the natural caves burns on for
// good (see stepEntities), so a player can mark the way they came.

// Cave props tuning.
const (
	StalagmiteYield = 3 // stone from a cluster of stalagmites
)

// touchCave is a player's blow at, or look at, a thing of the caves.
func (w *World) touchCave(p *Player, e *Entity, withTool func(ToolClass) int) bool {
	switch e.Kind {
	case EntityStalagmite:
		dmg := withTool(ToolPickaxe)
		return dmg > 0 && w.knock(e, dmg, func(x, y float64) {
			w.removeEntity(e)
			w.dropAt(x, y, []Ingredient{{Item: ItemStone, Count: StalagmiteYield}})
		})
	case EntityCaveProp:
		return w.touchProp(p, e) // see cavethemes.go
	}
	return false
}

// caveStep is what the natural caves do to p this tick.
func (w *World) caveStep(p *Player) {
	if w.caves == nil || w.caveLevel(int(math.Floor(p.Y))) < 0 {
		return
	}
	w.caveHazards(p) // see cavethemes.go
}
