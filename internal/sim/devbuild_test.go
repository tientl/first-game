package sim

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

// newFreeBuildWorld is a dev world in dev mode with free build on for its
// player, who carries nothing.
func newFreeBuildWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, id := newDevWorld(t)
	w.SetDevMode(true)
	if !w.DevSetFreeBuild(id, true) || !w.DevFreeBuild(id) {
		t.Fatal("free build would not turn on in dev mode")
	}
	p := w.players[id]
	p.Inv = Inventory{}
	return w, p
}

// freeTiles are tiles near p, nearest first, where anything (soft ground
// too) can be built: soft, empty, in reach, not p's own.
func freeTiles(w *World, p *Player, n int) [][2]int {
	var out [][2]int
	px, py := int(math.Floor(p.X)), int(math.Floor(p.Y))
	for r := 1; r <= 6 && len(out) < n; r++ {
		for ty := py - r; ty <= py+r && len(out) < n; ty++ {
			for tx := px - r; tx <= px+r && len(out) < n; tx++ {
				if max(abs(tx-px), abs(ty-py)) != r || !w.canPlace(tx, ty) || !w.TileAt(tx, ty).Soft() {
					continue
				}
				if PlayerRadius+0.5 > math.Abs(float64(tx)+0.5-p.X) && PlayerRadius+0.5 > math.Abs(float64(ty)+0.5-p.Y) {
					continue // p stands on it
				}
				out = append(out, [2]int{tx, ty})
			}
		}
	}
	return out
}

func devBuildAt(w *World, p *Player, piece int, tx, ty int) {
	w.Apply(p.ID, Command{Kind: CmdDevBuild, Slot: uint8(piece), At: true, TX: int16(tx), TY: int16(ty)})
}

func pieceOf(t *testing.T, k EntityKind, v uint8) int {
	t.Helper()
	for i, pc := range DevPieces() {
		if pc.Kind == k && pc.Var == v {
			return i
		}
	}
	t.Fatalf("no piece %v/%d in the palette", k, v)
	return 0
}

// Outside dev mode free build cannot be turned on, and its commands do
// nothing: crafting still takes what it takes.
func TestFreeBuildOnlyInDevMode(t *testing.T) {
	w, id := newDevWorld(t)
	p := w.players[id]
	p.Inv = Inventory{}
	if w.DevSetFreeBuild(id, true) || w.DevFreeBuild(id) || w.DevFreeBuildChosen(id) {
		t.Fatal("free build turned on in a world not in dev mode")
	}
	if r := devRun(t, w, id, "freebuild on"); w.DevFreeBuild(id) || !strings.Contains(strings.Join(r.Lines, " "), "chỉ có trong chế độ dev") {
		t.Errorf("freebuild on outside dev mode: %v", r.Lines)
	}
	tiles := freeTiles(w, p, 1)
	if len(tiles) == 0 {
		t.Fatal("no free tile near the player")
	}
	n := w.EntityCount()
	devBuildAt(w, p, pieceOf(t, EntityWall, 0), tiles[0][0], tiles[0][1])
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeAxe})
	if w.EntityCount() != n || p.Inv.Count(ItemAxe) != 0 {
		t.Error("a dev build or a craft from nothing went through outside dev mode")
	}
	if p.Notice != NoticeNotEnough {
		t.Errorf("crafting with nothing: notice %v, want NotEnough", p.Notice)
	}

	// Dev mode alone is not free build: it is turned on by the player (the
	// server does it as a dev player joins).
	w.SetDevMode(true)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeAxe})
	if p.Inv.Count(ItemAxe) != 0 {
		t.Error("crafted from nothing in dev mode with free build never turned on")
	}
	if r := devRun(t, w, id, "freebuild on"); !w.DevFreeBuild(id) {
		t.Errorf("freebuild on in dev mode: %v", r.Lines)
	}
	devRun(t, w, id, "freebuild")
	if w.DevFreeBuild(id) {
		t.Error("freebuild with no word should turn it over")
	}
	devRun(t, w, id, "xây tự do bật")
	if !w.DevFreeBuild(id) {
		t.Error("freebuild should be known by its Vietnamese name")
	}
	devRun(t, w, id, "freebuild off")
	if w.DevFreeBuild(id) || !w.DevFreeBuildChosen(id) {
		t.Error("freebuild off")
	}
	w.SetDevMode(false)
	devRun(t, w, id, "freebuild on")
	if w.DevFreeBuild(id) {
		t.Error("free build on in a world taken out of dev mode")
	}
}

// Free build makes every recipe out of nothing, away from any fire or
// furnace, and uses up nothing the player carries.
func TestFreeCraftNeedsNothingAndUsesNothing(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	w.DevGive(p.ID, ItemWood, 7)
	w.DevGive(p.ID, ItemMeat, 2)
	if w.nearFire(p.X, p.Y) || w.nearFurnace(p.X, p.Y) {
		t.Fatal("the player starts by a fire")
	}
	for _, c := range []struct {
		id   RecipeID
		gets ItemKind
	}{
		{RecipeAxe, ItemAxe},             // sticks and a pebble, none carried
		{RecipeCookMeat, ItemCookedMeat}, // by a fire, none near
		{RecipeSmeltIron, ItemIronBar},   // by a furnace, none near
		{RecipeClayPot, ItemClayPot},     // clay, by a furnace
		{RecipeBattleAxe, ItemBattleAxe}, // wood (carried), iron and hide
		{RecipeSailboat, ItemSailboat},   // 20 wood: more than carried
		{TierToolRecipe(ToolPickaxe, TierDiamond), ToolItem(ToolPickaxe, TierDiamond)},
	} {
		before := p.Inv.Count(c.gets)
		w.Apply(p.ID, Command{Kind: CmdCraft, Recipe: c.id})
		if p.Inv.Count(c.gets) != before+1 {
			r, _ := RecipeByID(c.id)
			t.Errorf("%s: not made (notice %v)", r.Name, p.Notice)
		}
	}
	if p.Inv.Count(ItemWood) != 7 || p.Inv.Count(ItemMeat) != 2 {
		t.Errorf("free crafting used up what was carried: %d wood, %d meat", p.Inv.Count(ItemWood), p.Inv.Count(ItemMeat))
	}

	// A structure from a recipe goes on the tile chosen; nothing is used.
	tiles := freeTiles(w, p, 2)
	if len(tiles) < 2 {
		t.Fatal("no free tiles near the player")
	}
	for i, id := range []RecipeID{RecipeWall, RecipeFurnace} {
		w.Apply(p.ID, Command{Kind: CmdCraft, Recipe: id, At: true, TX: int16(tiles[i][0]), TY: int16(tiles[i][1])})
		r, _ := RecipeByID(id)
		if e := w.entityAt(tiles[i][0], tiles[i][1]); e == nil || e.Kind != r.Places {
			t.Errorf("%s was not put down (notice %v)", r.Name, p.Notice)
		}
	}
	if p.Inv.Count(ItemWood) != 7 {
		t.Errorf("building used up wood: %d left", p.Inv.Count(ItemWood))
	}

	// A torch set down from the hand stays in the hand.
	p.Selected, p.Inv[0] = 0, Slot{Item: ItemTorch, Count: 1}
	if tt := freeTiles(w, p, 1); len(tt) > 0 {
		w.Apply(p.ID, Command{Kind: CmdPlace, At: true, TX: int16(tt[0][0]), TY: int16(tt[0][1])})
		if e := w.entityAt(tt[0][0], tt[0][1]); e == nil || e.Kind != EntityTorchStand || p.Inv.Count(ItemTorch) != 1 {
			t.Error("a torch set down in free build should stand there and stay in hand")
		}
	}
}

// Every piece of the palette goes down from nothing, as what it is, and
// uses nothing up; cooking gear goes onto a bare fire too.
func TestFreeBuildPlacesEveryPiece(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	pieces := DevPieces()
	tiles := freeTiles(w, p, len(pieces))
	if len(tiles) < len(pieces) {
		t.Fatalf("only %d free tiles for %d pieces", len(tiles), len(pieces))
	}
	for i, pc := range pieces {
		tx, ty := tiles[i][0], tiles[i][1]
		devBuildAt(w, p, i, tx, ty)
		e := w.entityAt(tx, ty)
		if e == nil || e.Kind != pc.Kind || e.Var != pc.Var {
			t.Errorf("%s: got %+v (notice %v)", pc.Name, e, p.Notice)
			continue
		}
		switch pc.Kind {
		case EntitySapling:
			if e.Ready || e.RegrowAt <= w.Tick() || w.active[e.ID] == nil {
				t.Errorf("%s: a seedling should be growing", pc.Name)
			}
		case EntityTorchStand:
			if e.Fuel != TorchStandTicks {
				t.Errorf("%s: fuel %d", pc.Name, e.Fuel)
			}
		}
		if pc.OnFire && w.stations[e.ID] == nil {
			t.Errorf("%s: no station on the fire", pc.Name)
		}
	}
	if p.Inv != (Inventory{}) {
		t.Errorf("free build put something in the bag or took it: %v", p.Inv)
	}

	// The pot onto a bare fire already there.
	fire, pot := pieceOf(t, EntityCampfire, 0), pieceOf(t, EntityCampfire, uint8(GearPot))
	tt := freeTiles(w, p, 1)
	if len(tt) == 0 {
		t.Fatal("no tile left")
	}
	devBuildAt(w, p, fire, tt[0][0], tt[0][1])
	e := w.entityAt(tt[0][0], tt[0][1])
	devBuildAt(w, p, pot, tt[0][0], tt[0][1])
	if e2 := w.entityAt(tt[0][0], tt[0][1]); e2 != e || e.Var != uint8(GearPot) {
		t.Error("the pot should go onto the fire there")
	}
	// Placing again keeps nothing back: the pot on a fire with a pot is refused.
	devBuildAt(w, p, pot, tt[0][0], tt[0][1])
	if p.Notice != NoticeNoSpace {
		t.Errorf("a pot onto a fire with one: notice %v", p.Notice)
	}
}

// Free build keeps the ground's rules: not on water, rock or a tile built
// on, not on the builder's own tile, not out of reach.
func TestFreeBuildKeepsTheGroundRules(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	wall := pieceOf(t, EntityWall, 0)
	try := func(what string, tx, ty int, why NoticeKind) {
		t.Helper()
		n := w.EntityCount()
		devBuildAt(w, p, wall, tx, ty)
		if w.EntityCount() != n || p.Notice != why {
			t.Errorf("%s: built (%d entities, was %d), notice %v, want %v", what, w.EntityCount(), n, p.Notice, why)
		}
	}
	px, py := int(math.Floor(p.X)), int(math.Floor(p.Y))
	try("on the builder's tile", px, py, NoticeNoSpace)
	try("out of reach", px+int(DevReach)+2, py, NoticeTooFar)
	try("off the map", -1, py, NoticeTooFar)
	found := 0
	for ty := py - 6; ty <= py+6 && found < 3; ty++ {
		for tx := px - 6; tx <= px+6; tx++ {
			if math.Hypot(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y) > DevReach {
				continue
			}
			switch {
			case !w.TileAt(tx, ty).Walkable() && found&1 == 0:
				try(fmt.Sprintf("on tile %v", w.TileAt(tx, ty)), tx, ty, NoticeNoSpace)
				found |= 1
			case w.entityAt(tx, ty) != nil && found&2 == 0:
				try("on a "+w.entityAt(tx, ty).Kind.Name(), tx, ty, NoticeNoSpace)
				found |= 2
			}
		}
	}
	if found&2 == 0 {
		t.Error("no built-on tile near the player to try")
	}
	// The sea and rock, wherever the nearest are: the builder stands by them.
	for _, want := range []func(Tile) bool{Tile.Sea, func(t Tile) bool { return t == TileMountain }} {
		best, bx, by := math.Inf(1), 0, 0
		for ty := 1; ty < w.height-1; ty++ {
			for tx := 1; tx < w.width-1; tx++ {
				if want(w.TileAt(tx, ty)) && w.standable(float64(tx)+0.5, float64(ty)+2.5) {
					if d := math.Hypot(float64(tx)-p.X, float64(ty)-p.Y); d < best {
						best, bx, by = d, tx, ty
					}
				}
			}
		}
		if math.IsInf(best, 1) {
			t.Error("no sea or rock with ground beside it")
			continue
		}
		w.DevTeleport(p.ID, float64(bx)+0.5, float64(by)+2.5)
		try(fmt.Sprintf("on tile %v", w.TileAt(bx, by)), bx, by, NoticeNoSpace)
	}
	w.DevTeleport(p.ID, float64(px)+0.5, float64(py)+0.5)
	// A seed only on soft ground.
	seed := pieceOf(t, EntitySapling, uint8(SeedTree))
	for ty := py - 6; ty <= py+6; ty++ {
		for tx := px - 6; tx <= px+6; tx++ {
			if w.canPlace(tx, ty) && !w.TileAt(tx, ty).Soft() && math.Hypot(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y) < DevReach-1 {
				devBuildAt(w, p, seed, tx, ty)
				if w.entityAt(tx, ty) != nil {
					t.Errorf("a seedling went on tile %v", w.TileAt(tx, ty))
				}
				return
			}
		}
	}
}

// Shift and the right button take a structure away at once, for nothing:
// what a chest holds spills, the wall and the pot do not come back.
func TestFreeBuildRemoves(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	tiles := freeTiles(w, p, 3)
	if len(tiles) < 3 {
		t.Fatal("no free tiles near the player")
	}
	devBuildAt(w, p, pieceOf(t, EntityWall, 0), tiles[0][0], tiles[0][1])
	devBuildAt(w, p, pieceOf(t, EntityChest, 0), tiles[1][0], tiles[1][1])
	devBuildAt(w, p, pieceOf(t, EntityCampfire, uint8(GearPot)), tiles[2][0], tiles[2][1])
	chest := w.entityAt(tiles[1][0], tiles[1][1])
	w.chests[chest.ID].Add(ItemCoal, 5)
	drops := len(w.drops)
	for _, tl := range tiles {
		w.Apply(p.ID, Command{Kind: CmdDevRemove, At: true, TX: int16(tl[0]), TY: int16(tl[1])})
		if e := w.entityAt(tl[0], tl[1]); e != nil {
			t.Errorf("%s still stands", e.Kind.Name())
		}
	}
	var spilt []ItemKind
	for _, d := range w.drops[drops:] {
		spilt = append(spilt, d.Item)
	}
	if len(spilt) != 1 || spilt[0] != ItemCoal || p.Inv != (Inventory{}) {
		t.Errorf("taking away gave %v and %v; want only the chest's coal on the ground", spilt, p.Inv)
	}
	if w.chests[chest.ID] != nil {
		t.Error("the chest's store was kept")
	}

	// Nothing there, out of reach, a sight of the world: refused.
	w.Apply(p.ID, Command{Kind: CmdDevRemove, At: true, TX: int16(tiles[0][0]), TY: int16(tiles[0][1])})
	if p.Notice != NoticeNothingNear {
		t.Errorf("taking away nothing: notice %v", p.Notice)
	}
	for _, k := range []EntityKind{EntityPyramid, EntityCaveMouth, EntityGiantTree, EntityGrave, EntityTreasure} {
		if DevRemovable(k) {
			t.Errorf("%s may be taken away", k.Name())
		}
	}
	far := 0
	w.EachEntityIn(int(p.X)-20, int(p.Y)-20, int(p.X)+20, int(p.Y)+20, func(e Entity) {
		if far == 0 && DevRemovable(e.Kind) && math.Hypot(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y) > DevReach+1 {
			far = int(e.ID)
			w.Apply(p.ID, Command{Kind: CmdDevRemove, At: true, TX: int16(e.TX), TY: int16(e.TY)})
		}
	})
	if far == 0 || w.entities[EntityID(far)] == nil || p.Notice != NoticeTooFar {
		t.Errorf("taking away out of reach: notice %v", p.Notice)
	}

	// Outside free build it does nothing.
	w.DevSetFreeBuild(p.ID, false)
	devBuildAt(w, p, pieceOf(t, EntityWall, 0), tiles[0][0], tiles[0][1])
	w.addEntity(EntityWall, tiles[0][0], tiles[0][1])
	w.Apply(p.ID, Command{Kind: CmdDevRemove, At: true, TX: int16(tiles[0][0]), TY: int16(tiles[0][1])})
	if w.entityAt(tiles[0][0], tiles[0][1]) == nil {
		t.Error("taken away with free build off")
	}
}

// In free build one blow fells a tree, breaks a boulder, mines an ore and
// takes down a wall, with no tool in hand.
func TestFreeBuildOneBlow(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	tiles := freeTiles(w, p, 3)
	if len(tiles) < 3 {
		t.Fatal("no free tiles near the player")
	}
	tree := w.addEntity(EntityTree, tiles[0][0], tiles[0][1])
	tree.Big, tree.HP = true, maxHP(EntityTree, true)
	ore := w.addEntity(EntityIronOre, tiles[1][0], tiles[1][1])
	wall := w.addEntity(EntityWall, tiles[2][0], tiles[2][1])
	if !w.interact(p, tree) || tree.Ready {
		t.Error("a big tree should fall to one blow, axe or not")
	}
	if !w.interact(p, ore) || w.entities[ore.ID] != nil || p.Inv.Count(ItemIronOre) == 0 {
		t.Error("an ore should be mined with one blow, pickaxe or not")
	}
	if !w.interact(p, wall) || w.entities[wall.ID] != nil {
		t.Error("a wall should come down with one blow")
	}

	// Without free build the same is refused for want of a tool.
	w.DevSetFreeBuild(p.ID, false)
	ore2 := w.addEntity(EntityIronOre, tiles[1][0], tiles[1][1])
	if w.interact(p, ore2) || w.entities[ore2.ID] == nil {
		t.Error("mined without a pickaxe outside free build")
	}
}

// The setting is kept in a dev world's save, and only there.
func TestFreeBuildSavedOnlyInDevSaves(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	w.DevSetFreeBuild(p.ID, false)
	d := w.Save()
	var got *DevSave
	for _, ps := range d.Players {
		if ps.Name == p.Name {
			got = ps.Dev
		}
	}
	if got == nil || got.FreeBuild {
		t.Fatalf("dev save keeps %+v, want free build off", got)
	}

	// Back in dev mode: off, as chosen.
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	w2.SetDevMode(true)
	id2 := w2.AddPlayer(p.Name)
	if w2.DevFreeBuild(id2) || !w2.DevFreeBuildChosen(id2) {
		t.Error("a dev save opened in dev mode lost the choice")
	}

	// Opened as a normal game: nothing of it, and saved again without it.
	w3, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	id3 := w3.AddPlayer(p.Name)
	if w3.DevFreeBuild(id3) || w3.DevFreeBuildChosen(id3) {
		t.Error("a normal game took up the dev save's free build")
	}
	w3.RemovePlayer(id3) // offline players are saved too
	for _, s := range []SaveData{w3.Save(), mustLoad(t, d).Save()} {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "FreeBuild") {
			t.Error("a normal save carries free build")
		}
	}

	// On, the default, also comes back.
	w.DevSetFreeBuild(p.ID, true)
	w4 := mustLoad(t, w.Save())
	w4.SetDevMode(true)
	if id4 := w4.AddPlayer(p.Name); !w4.DevFreeBuild(id4) {
		t.Error("free build on was not kept")
	}
}

func mustLoad(t *testing.T, d SaveData) *World {
	t.Helper()
	w, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// Every piece names a kind and fits a command; the console lists the
// command.
func TestDevPalette(t *testing.T) {
	pieces := DevPieces()
	if len(pieces) == 0 || len(pieces) > 255 {
		t.Fatalf("%d pieces", len(pieces))
	}
	groups := map[DevGroup]int{}
	for i, pc := range pieces {
		if pc.Name == "" || strings.Contains(pc.Name, "?") || pc.Kind == EntityNone || pc.Kind >= NumEntityKinds || pc.Group >= NumDevGroups {
			t.Errorf("piece %d: %+v", i, pc)
		}
		groups[pc.Group]++
	}
	for gr := range NumDevGroups {
		if groups[gr] == 0 {
			t.Errorf("section %s is empty", DevGroupNames[gr])
		}
	}
	if !strings.Contains(strings.Join(DevHelp, "\n"), "freebuild") {
		t.Error("help does not list freebuild")
	}
	if !IsDevCommand(CmdDevBuild) || !IsDevCommand(CmdDevRemove) || IsDevCommand(CmdCraft) {
		t.Error("IsDevCommand")
	}
}
