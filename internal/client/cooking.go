package client

import "survivalgame/internal/sim"

// Cooking: raw meat and fish become better food at a burning campfire. It is
// ordinary crafting with recipes marked NearFire; F cooks whatever raw food
// the player has, and the recipe cards in the bag work too. The icons of the
// cooked food and the sizzle are in craftart.go.

// cookRecipes are the recipes made at a fire, in the order F tries them.
var cookRecipes = func() []sim.Recipe {
	var rs []sim.Recipe
	for id := sim.RecipeID(0); id < sim.NumRecipes; id++ {
		if r, ok := sim.RecipeByID(id); ok && r.NearFire {
			rs = append(rs, r)
		}
	}
	return rs
}()

// cookRecipe is what F cooks: the first fire recipe the player has the food
// for, else the first one, so the server says what is missing.
func (g *Game) cookRecipe() sim.RecipeID {
	for _, r := range cookRecipes {
		if g.curr.Self.Inv.HasAll(r.Inputs) {
			return r.ID
		}
	}
	if len(cookRecipes) == 0 {
		return sim.NumRecipes // nothing to cook: the server ignores it
	}
	return cookRecipes[0].ID
}

// hasRawFood reports whether the player carries anything F could cook.
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

// foodScore ranks food for E: what heals most first (cooked food), raw food
// last.
func foodScore(k sim.ItemKind) float64 {
	d := k.Def()
	return d.Health*100 + d.Hunger
}
