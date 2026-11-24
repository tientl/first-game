package client

import (
	"slices"
	"testing"

	"survivalgame/internal/sim"
)

func recipeIDs(rs []sim.Recipe) []sim.RecipeID {
	var ids []sim.RecipeID
	for _, r := range rs {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestRelatedRecipesMakeThenUse(t *testing.T) {
	if rs := relatedRecipes(sim.ItemNone); len(rs) != 0 {
		t.Errorf("nothing clicked lists %v, want none", recipeIDs(rs))
	}
	// The axe is made, and nothing is made from it.
	if got := recipeIDs(relatedRecipes(sim.ItemAxe)); len(got) != 1 || got[0] != sim.RecipeAxe {
		t.Errorf("axe lists %v, want just the axe recipe", got)
	}
	// Raw meat is used by cooking; cooked meat is made by it.
	for _, item := range []sim.ItemKind{sim.ItemMeat, sim.ItemCookedMeat} {
		if got := recipeIDs(relatedRecipes(item)); len(got) != 1 || got[0] != sim.RecipeCookMeat {
			t.Errorf("%s lists %v, want the cooking recipe", item.Def().Name, got)
		}
	}
	// Everything listed for wood really involves wood.
	for _, r := range relatedRecipes(sim.ItemWood) {
		uses := r.Gives == sim.ItemWood
		for _, in := range r.Inputs {
			uses = uses || in.Item == sim.ItemWood
		}
		if !uses {
			t.Errorf("wood lists %s, which neither makes nor uses wood", r.Name)
		}
	}
	if cardRows < 1 {
		t.Errorf("cardRows = %d", cardRows)
	}
}

func TestClickingAnItemListsItsRecipesAndACardCraftsOne(t *testing.T) {
	g, c := bagForMouse() // wood in hotbar slot 0
	b := &g.bag
	b.cardX, b.cardY, b.cardW, b.cardRowsShown = 100, 700, 200, cardRows
	if len(b.shownRecipes()) != 0 {
		t.Fatalf("before any click the bag lists %v", recipeIDs(b.shownRecipes()))
	}

	x0, y0 := centre(b, 0)
	g.bagMouse(x0, y0, false, true, false, false) // click the wood: lifted, and its recipes listed
	g.bagMouse(x0, y0, false, false, true, false)
	want := relatedRecipes(sim.ItemWood)
	if b.focus != sim.ItemWood || len(b.shownRecipes()) != len(want) || len(want) == 0 {
		t.Fatalf("after clicking the wood: focus %v, listing %v, want %v", b.focus, recipeIDs(b.shownRecipes()), recipeIDs(want))
	}

	// On the way down to the cards the pointer crosses other slots: the list stays.
	x1, y1 := centre(b, 4)
	g.bagMouse(x1, y1, true, false, false, false)
	if b.focus != sim.ItemWood {
		t.Fatalf("hovering slot 4 changed the focus to %v", b.focus)
	}

	// A card for an item crafts it at once.
	item := slices.IndexFunc(want, func(r sim.Recipe) bool { return r.Gives.Valid() })
	cx, cy := b.cardXY(item)
	g.bagMouse(cx+5, cy+5, true, true, false, false)
	if len(c.cmds) != 1 || c.cmds[0].Kind != sim.CmdCraft || c.cmds[0].Recipe != want[item].ID || c.cmds[0].At {
		t.Fatalf("clicking the %s card sent %v, want crafting it", want[item].Name, c.cmds)
	}
	if b.lifting {
		t.Error("the wood is still lifted after crafting from its recipes")
	}

	// A card for a structure (the campfire comes first) sends nothing: with
	// the ingredients, it closes the bag and waits for a tile to build on.
	if want[0].Gives.Valid() {
		t.Fatalf("the first recipe for wood is %s, want a structure", want[0].Name)
	}
	for _, in := range want[0].Inputs {
		g.curr.Self.Inv.Add(in.Item, in.Count)
	}
	cx, cy = b.cardXY(0)
	g.bagMouse(cx+5, cy+5, true, true, false, false)
	if len(c.cmds) != 1 || !g.build.on || g.build.recipe != want[0].ID || g.overlay != overlayNone {
		t.Fatalf("clicking %s: sent %v, build mode %+v, overlay %d; want build mode, the bag closed",
			want[0].Name, c.cmds[1:], g.build, g.overlay)
	}
}

func TestTheRecipesScrollWhenTheyDoNotFit(t *testing.T) {
	g, _ := bagForMouse()
	b := &g.bag
	b.setFocus(sim.ItemStick) // every tool takes sticks
	rows := (len(b.shownRecipes()) + cardCols - 1) / cardCols
	if rows < 3 {
		t.Skipf("only %d rows of stick recipes", rows)
	}
	b.cardRowsShown = 2
	if !b.cardShown(0) || b.cardShown(2*cardCols) {
		t.Fatal("before scrolling: want the first two rows shown, no more")
	}
	b.scrollCards(1)
	if b.cardShown(0) || !b.cardShown(2*cardCols) {
		t.Fatal("after one row down: want rows 2 and 3 shown")
	}
	b.scrollCards(100)
	if b.cardScroll != rows-2 {
		t.Errorf("scrolled to row %d, want the last full view at %d", b.cardScroll, rows-2)
	}
	b.scrollCards(-100)
	if b.cardScroll != 0 {
		t.Errorf("scrolled back to row %d, want 0", b.cardScroll)
	}
	b.scrollCards(3)
	b.setFocus(sim.ItemWood)
	if b.cardScroll != 0 {
		t.Error("clicking another item kept the old scroll")
	}
}
