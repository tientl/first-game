package client

import "survivalgame/internal/sim"

// Cooking: raw meat and fish become better food at a burning campfire. It is
// ordinary crafting with recipes marked NearFire; a right click with raw
// food in hand cooks it by the fire (see rightClickUse in mouse.go), and the
// crafting window and the recipe cards in the bag work too. The icons of the
// cooked food and the sizzle are in craftart.go.

// cookRecipes are the recipes made at a fire.
var cookRecipes = func() []sim.Recipe {
	var rs []sim.Recipe
	for id := sim.RecipeID(0); id < sim.NumRecipes; id++ {
		if r, ok := sim.RecipeByID(id); ok && r.NearFire {
			rs = append(rs, r)
		}
	}
	return rs
}()

// hasRawFood reports whether the player carries anything to cook at a fire.
func (g *Game) hasRawFood() bool {
	for _, r := range cookRecipes {
		if g.curr.Self.Inv.HasAll(r.Inputs) {
			return true
		}
	}
	return false
}

// byFire reports whether the player stands in the light of a burning
// campfire: the server's test for cooking, and for warmth.
func (g *Game) byFire() bool {
	for _, p := range g.curr.Players {
		if p.ID != g.myID {
			continue
		}
		x, y := float64(p.X), float64(p.Y)
		for _, e := range g.curr.Entities {
			r, ok := sim.FireReach(e.Kind, int(e.Fuel)) // a campfire, or a lit furnace
			if !ok {
				continue
			}
			if dx, dy := float64(e.TX)+0.5-x, float64(e.TY)+0.5-y; dx*dx+dy*dy <= r*r {
				return true
			}
		}
	}
	return false
}
