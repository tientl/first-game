package client

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Mouse-first controls; the rules the server keeps are in sim/aim.go.
//
//   - Where the pointer is goes to the server as the aim: it picks what the
//     left button works on. The player faces the way they walk, and turns
//     to the pointer only while aiming something aimed (sim.AimedItem) with
//     the button held. A pointer left alone for a few seconds stops aiming.
//   - The left button works on what is under the pointer, bracketed on
//     screen (red when out of reach). With nothing there in reach it works
//     on the nearest target, preferring those towards the pointer, as Space
//     does. A click on the hotbar picks that slot.
//   - The right button uses what is held: see rightClickUse.
//   - Building: a structure chosen in the crafting window (C), or a torch,
//     a head or a seed in the hand, shows a ghost on the tile under the
//     pointer: green where it can go, red where it cannot. A left click puts
//     it there. A right click or Esc stops building a structure.

const (
	aimIdle    = 5 * time.Second      // a pointer left alone this long no longer aims
	aimStep    = 16                   // the aim is sent in steps of 1/aimStep tile
	ghostRange = sim.PlaceReach + 2.5 // a torch or seed in hand shows its ghost this close
)

// pointer is the mouse in play.
type pointer struct {
	cam          camera // the last frame's view, to map the pointer into the world
	scale        float64
	sw, sh       int
	drawn        bool
	lastX, lastY int
	movedAt      time.Time
	block        bool // the left press went to something else (the hotbar, a building): no Action till it is let go

	fixed      bool // SetAim: a pointer held at an offset from the player, for screenshots
	fixX, fixY float64
}

// buildMode is a structure chosen in the crafting window, waiting for a
// tile to stand on.
type buildMode struct {
	on     bool
	recipe sim.RecipeID
}

// SetAim holds the pointer (dx, dy) tiles from the player, as if the mouse
// were there: for the screenshot tool.
func (g *Game) SetAim(dx, dy float64) { g.ptr.fixed, g.ptr.fixX, g.ptr.fixY = true, dx, dy }

// StartBuild chooses recipe id to build, as a click in the crafting window
// does: for the screenshot tool.
func (g *Game) StartBuild(id sim.RecipeID) { g.build = buildMode{on: true, recipe: id} }

// toWorld maps a point on the world canvas (art pixels) to tiles.
func (c camera) toWorld(x, y float64) (float64, float64) {
	return (x + c.ox) / TileW, (y + c.oy) / TileH
}

// notePointer remembers when the mouse last moved.
func (g *Game) notePointer() {
	mx, my := ebiten.CursorPosition()
	if mx != g.ptr.lastX || my != g.ptr.lastY {
		g.ptr.lastX, g.ptr.lastY, g.ptr.movedAt = mx, my, time.Now()
	}
}

// pointerWorld is the point of the world under the pointer, in tiles.
func (g *Game) pointerWorld() (x, y float64, ok bool) {
	if g.ptr.fixed {
		me, ok := g.me()
		return float64(me.X) + g.ptr.fixX, float64(me.Y) + g.ptr.fixY, ok
	}
	if !g.ptr.drawn || g.ptr.scale <= 0 {
		return 0, 0, false
	}
	mx, my := ebiten.CursorPosition()
	if mx < 0 || my < 0 || mx >= g.ptr.sw || my >= g.ptr.sh {
		return 0, 0, false
	}
	x, y = g.ptr.cam.toWorld(float64(mx)/g.ptr.scale, float64(my)/g.ptr.scale)
	return x, y, true
}

func anyMouseButton() bool {
	return ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
}

// aim is where the player aims, if they do: at the pointer while it is in
// use, or held, or a structure is being placed.
func (g *Game) aim() (x, y float64, ok bool) {
	x, y, ok = g.pointerWorld()
	if !ok || g.ptr.fixed || g.build.on || time.Since(g.ptr.movedAt) < aimIdle || anyMouseButton() {
		return x, y, ok
	}
	return 0, 0, false
}

// quantAim rounds an aim to what is sent, so a still pointer sends nothing.
func quantAim(v float64) float32 { return float32(math.Round(v*aimStep) / aimStep) }

// --- what the pointer is on ---------------------------------------------------

// underPointer is the target or pile under the pointer at (ax, ay) and
// whether the player reaches it; ok is false when there is none.
func (g *Game) underPointer(me protocol.PlayerView, ax, ay float64) (tg targetView, inReach bool) {
	x, y := float64(me.X), float64(me.Y)
	var drops []sim.DropCandidate
	for _, d := range g.curr.Drops {
		drops = append(drops, sim.DropCandidate{ID: d.ID, X: float64(d.X), Y: float64(d.Y)})
	}
	if id, ok := sim.AimedDrop(x, y, ax, ay, drops); ok {
		for _, d := range g.curr.Drops {
			if d.ID == id {
				return targetView{ok: true, drop: true, d: d, aimed: true}, true
			}
		}
	}
	t, in, ok := sim.AimedTarget(x, y, ax, ay, g.curr.Self.Inv.HasFuel(), g.candidateAt,
		creatureCandidates(g.curr.Creatures, ax, ay), sim.ReachBonusFor(me.Held)+sim.RideReach(me.Mount != 0))
	if !ok {
		return targetView{}, false
	}
	if tv, ok := g.viewOf(t); ok {
		tv.aimed = true
		return tv, in
	}
	return targetView{}, false
}

// viewOf finds what t names in the snapshot; things hidden under a cave
// roof are not targets.
func (g *Game) viewOf(t sim.Target) (targetView, bool) {
	if t.Creature != 0 {
		for _, c := range g.curr.Creatures {
			if c.ID == t.Creature && !g.hiddenInCave(float64(c.X), float64(c.Y)) {
				return targetView{ok: true, creature: true, c: c}, true
			}
		}
		return targetView{}, false
	}
	for _, e := range g.curr.Entities {
		if e.ID == t.Entity && !g.hiddenInCave(float64(e.TX)+0.5, float64(e.TY)+0.5) {
			return targetView{ok: true, e: e}, true
		}
	}
	return targetView{}, false
}

// ghostView is something about to be built, drawn on the tile under the
// pointer.
type ghostView struct {
	ok     bool
	kind   sim.EntityKind
	item   sim.ItemKind // what is held, when it comes from the hand
	recipe bool         // a structure from a recipe (build mode)
	tx, ty int
	valid  bool
	why    sim.NoticeKind
}

// ghost is what a left click would build, and where: in build mode the
// structure chosen; else a torch, head or seed in the hand, when the
// pointer is near and on no target.
func (g *Game) ghost() ghostView {
	me, ok := g.me()
	if !ok || me.Dead || g.curr.Self.Dead || g.overlay != overlayNone || me.Mount != 0 || me.Boat != 0 || me.Swimming {
		return ghostView{}
	}
	ax, ay, aim := g.aim()
	if !aim {
		return ghostView{}
	}
	gv := ghostView{tx: int(math.Floor(ax)), ty: int(math.Floor(ay))}
	soft := false
	if g.build.on {
		r, ok := sim.RecipeByID(g.build.recipe)
		if !ok || r.Places == sim.EntityNone {
			return ghostView{}
		}
		gv.kind, gv.recipe = r.Places, true
	} else {
		k, s, ok := sim.PlacedBy(me.Held)
		if !ok || math.Hypot(ax-float64(me.X), ay-float64(me.Y)) > ghostRange || me.Cast != sim.CastNone {
			return ghostView{}
		}
		if u, _ := g.underPointer(me, ax, ay); u.ok {
			return ghostView{} // a target under the pointer is worked on instead
		}
		gv.kind, gv.item, soft = k, me.Held, s
	}
	gv.ok = true
	gv.valid, gv.why = sim.BuildSpot(float64(me.X), float64(me.Y), gv.tx, gv.ty, gv.kind, soft, g.tileAt, g.taken, g.standers())
	return gv
}

// taken reports whether a tile is built on, as far as this client sees.
func (g *Game) taken(tx, ty int) bool {
	_, ok := g.entAt[[2]int{tx, ty}]
	return ok
}

// standers are where the players in view stand.
func (g *Game) standers() [][2]float64 {
	var out [][2]float64
	for _, p := range g.curr.Players {
		if !p.Dead {
			out = append(out, [2]float64{float64(p.X), float64(p.Y)})
		}
	}
	return out
}

// --- the buttons -------------------------------------------------------------------

// leftButton handles the left button in play and reports whether it is
// held down for Action.
func (g *Game) leftButton() bool {
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || g.lastScreenH == 0 {
		g.ptr.block = false
		return false
	}
	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.ptr.block = g.leftClick(mx, my)
	}
	return !g.ptr.block && my < g.lastScreenH-slotPx-32 // not on the hotbar
}

// leftClick is a fresh left press at (mx, my) in play; it reports whether
// it was used up, so it does no Action.
func (g *Game) leftClick(mx, my int) bool {
	if i, ok := hotbarSlotAt(mx, my, g.lastScreenW, g.lastScreenH); ok {
		g.selectSlot(i)
		return true
	}
	gv := g.ghost()
	if gv.ok {
		g.placeGhost(gv)
		return true
	}
	return g.build.on // building: the left button builds, never strikes
}

// placeGhost asks the server to build gv where it stands.
func (g *Game) placeGhost(gv ghostView) {
	if !gv.valid {
		g.showNotice(buildRefusal(gv.why))
		return
	}
	c := sim.Command{Kind: sim.CmdPlace, At: true, TX: int16(gv.tx), TY: int16(gv.ty)}
	if gv.recipe {
		c.Kind, c.Recipe = sim.CmdCraft, g.build.recipe
	}
	g.command(c)
}

// buildRefusal says why the ghost is red.
func buildRefusal(why sim.NoticeKind) string {
	if why == sim.NoticeTooFar {
		return "Xa quá — lại gần hơn rồi đặt"
	}
	return "Không đặt được ở đây — cần chỗ đất trống, không có ai đứng"
}

// hotbarSlotAt is the hotbar slot under screen point (mx, my), as
// drawHotbar lays it out on a sw by sh screen.
func hotbarSlotAt(mx, my, sw, sh int) (int, bool) {
	const n = sim.HotbarSlots
	total := n*slotPx + (n-1)*slotGap
	x0 := (sw - total) / 2
	y := sh - slotPx - 20
	if sw <= 0 || my < y || my >= y+slotPx || mx < x0 {
		return 0, false
	}
	i := (mx - x0) / (slotPx + slotGap)
	if i >= n || (mx-x0)%(slotPx+slotGap) >= slotPx {
		return 0, false
	}
	return i, true
}

// useContext is what the right button needs to know about the world.
type useContext struct {
	byFire bool      // in a burning campfire's light: raw food is cooked
	feed   bool      // the pointer is on a beast in reach that eats what is held
	spot   ghostView // the tile under the pointer, for a torch, head or seed
	aim    bool      // the pointer is in the world: dung goes on what it is on
	aimTX  int
	aimTY  int
}

// cookRecipeFor is the fire recipe that cooks item on its own, if any.
func cookRecipeFor(item sim.ItemKind) (sim.Recipe, bool) {
	for _, r := range cookRecipes {
		if len(r.Inputs) == 1 && r.Inputs[0].Item == item {
			return r, true
		}
	}
	return sim.Recipe{}, false
}

// isBoat reports whether k is a boat to launch.
func isBoat(k sim.ItemKind) bool { return k == sim.ItemRaft || k == sim.ItemSailboat }

// rightClickUse is what the right button does with held, in hotbar slot
// slot: feed the beast pointed at its food; cook raw food by a fire; eat
// food or medicine; launch a boat; set down a torch, a head or a seed on the
// tile pointed at (in front, towards the pointer, when that tile will not
// do); spread dung on what is pointed at. Anything else does nothing, and
// hint says so.
func rightClickUse(held sim.ItemKind, slot int, ctx useContext) (cmd sim.Command, hint string) {
	switch {
	case !held.Valid():
		return sim.Command{}, "Tay không — chuột phải dùng món đang cầm: đồ ăn, đuốc, hạt giống, thuyền…"
	case ctx.feed:
		return sim.Command{Kind: sim.CmdFeed}, ""
	}
	if r, ok := cookRecipeFor(held); ok && ctx.byFire {
		return sim.Command{Kind: sim.CmdCraft, Recipe: r.ID}, ""
	}
	switch {
	case held.Edible():
		return sim.Command{Kind: sim.CmdEat, Slot: uint8(slot)}, ""
	case isBoat(held):
		return sim.Command{Kind: sim.CmdPlace}, ""
	case held == sim.ItemDung:
		c := sim.Command{Kind: sim.CmdPlace}
		if ctx.aim {
			c.At, c.TX, c.TY = true, int16(ctx.aimTX), int16(ctx.aimTY)
		}
		return c, ""
	}
	if _, _, ok := sim.PlacedBy(held); ok {
		c := sim.Command{Kind: sim.CmdPlace}
		if ctx.spot.ok && ctx.spot.valid {
			c.At, c.TX, c.TY = true, int16(ctx.spot.tx), int16(ctx.spot.ty)
		}
		return c, ""
	}
	if held.Def().Tool || sim.WeaponDamage(held) > sim.WeaponDamage(sim.ItemNone) {
		return sim.Command{}, held.Def().Name + ": dùng bằng chuột trái"
	}
	return sim.Command{}, held.Def().Name + " không dùng bằng chuột phải được — chế tạo ở bảng chế tạo (C)"
}

// useContextNow is the right button's context in this frame.
func (g *Game) useContextNow() useContext {
	ctx := useContext{byFire: g.byFire()}
	me, ok := g.me()
	if !ok {
		return ctx
	}
	ax, ay, aim := g.aim()
	if !aim {
		return ctx
	}
	ctx.aim, ctx.aimTX, ctx.aimTY = true, int(math.Floor(ax)), int(math.Floor(ay))
	if u, in := g.underPointer(me, ax, ay); u.ok && in && u.creature {
		c := u.c
		ctx.feed = c.Kind.TameFood() == me.Held && me.Held.Valid() && !(c.Tamed && c.HP >= c.MaxHP)
	}
	if k, s, ok := sim.PlacedBy(me.Held); ok {
		gv := ghostView{ok: true, kind: k, tx: ctx.aimTX, ty: ctx.aimTY}
		gv.valid, gv.why = sim.BuildSpot(float64(me.X), float64(me.Y), gv.tx, gv.ty, k, s, g.tileAt, g.taken, g.standers())
		ctx.spot = gv
	}
	return ctx
}

// rightClick is the right button in play.
func (g *Game) rightClick() {
	if g.build.on {
		g.build.on = false
		g.showNotice("Đã thôi xây")
		snd.ui("ui_move")
		return
	}
	cmd, hint := rightClickUse(g.myHeld(), g.hotSel, g.useContextNow())
	if cmd.Kind == sim.CmdNone {
		g.showNotice(hint)
		return
	}
	g.command(cmd)
}

// useLabel names what the right button does with held, for the help.
func useLabel(cmd sim.Command, held sim.ItemKind) string {
	name := strings.ToLower(held.Def().Name)
	switch cmd.Kind {
	case sim.CmdFeed:
		return "cho thú ăn " + name
	case sim.CmdEat:
		return "ăn " + name
	case sim.CmdCraft:
		return "nướng " + name + " (đứng gần lửa)"
	case sim.CmdPlace:
		switch {
		case isBoat(held):
			return "thả thuyền xuống nước, về phía con trỏ"
		case held == sim.ItemDung:
			return "bón phân cho cây non, luống đất dưới con trỏ"
		case held == sim.ItemTorch:
			return "cắm đuốc xuống chỗ con trỏ"
		case sim.IsSeed(held):
			return "gieo hạt ở chỗ con trỏ"
		}
		return "dựng đầu thú lên cọc ở chỗ con trỏ"
	}
	return ""
}

// craftRecipe is a click on recipe r, in the crafting window or the bag: a
// structure waits for a tile to stand on (build mode), anything else is
// made at once.
func (g *Game) craftRecipe(r sim.Recipe) {
	if r.Gives.Valid() || r.Places == sim.EntityNone {
		g.command(sim.Command{Kind: sim.CmdCraft, Recipe: r.ID})
		return
	}
	if !g.curr.Self.Inv.HasAll(r.Inputs) {
		g.showNotice("Thiếu nguyên liệu — cần " + ingredientsText(r))
		return
	}
	g.build = buildMode{on: true, recipe: r.ID}
	g.setOverlay(overlayNone)
	g.showNotice("Chọn chỗ dựng " + r.Name + ": chuột trái đặt · chuột phải hoặc Esc thôi")
}

// keepBuild leaves build mode once the ingredients run out, or on death.
func (g *Game) keepBuild() {
	if !g.build.on {
		return
	}
	r, ok := sim.RecipeByID(g.build.recipe)
	switch {
	case !ok || g.curr.Self.Dead:
		g.build.on = false
	case !g.curr.Self.Inv.HasAll(r.Inputs):
		g.build.on = false
		g.showNotice("Hết nguyên liệu để dựng " + r.Name)
	}
}

// --- drawing ---------------------------------------------------------------------

var (
	colGhostOK  = color.RGBA{110, 230, 120, 255}
	colGhostBad = color.RGBA{236, 90, 90, 255}
)

// drawGhost draws what a left click would build on the world canvas: the
// tile outlined and the thing itself, see-through, green or red.
func (g *Game) drawGhost(dst drawTarget, cam camera) {
	gv := g.ghost()
	if !gv.ok {
		return
	}
	x, y := cam.toScreen(float64(gv.tx), float64(gv.ty))
	col, tint := colGhostOK, [3]float32{0.75, 1.35, 0.75}
	if !gv.valid {
		col, tint = colGhostBad, [3]float32{1.4, 0.6, 0.6}
	}
	pulse := uint8(150 + 70*math.Sin(float64(g.frame)*0.15))
	fillRect(dst, float32(x)+1, float32(y)+1, TileW-2, TileH-2, color.RGBA{col.R / 3, col.G / 3, col.B / 3, 90}, false)
	strokeRect(dst, float32(x)+1.5, float32(y)+1.5, TileW-3, TileH-3, 1, color.RGBA{uint8(int(col.R) * int(pulse) / 255), uint8(int(col.G) * int(pulse) / 255), uint8(int(col.B) * int(pulse) / 255), pulse}, false)
	ev := protocol.EntityView{Kind: gv.kind, TX: int16(gv.tx), TY: int16(gv.ty), Ready: true}
	switch gv.kind {
	case sim.EntityTorchStand:
		ev.Fuel = sim.TorchStandTicks
	case sim.EntityTrophy:
		c, _ := sim.HeadOf(gv.item)
		ev.Var = uint8(c)
	case sim.EntitySapling:
		ev.Ready = false
	}
	// See-through and tinted: the colour scale is premultiplied, so the
	// tint is scaled by the alpha too.
	const a = 0.72
	draw := func(s sprite, x, y float64) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(x-s.ax), math.Round(y-s.ay))
		op.ColorScale.Scale(tint[0]*a, tint[1]*a, tint[2]*a, a)
		drawSprite(dst, s, op)
	}
	cx, cy := x+TileW/2, y+TileH/2
	s, _ := g.entitySprite(ev)
	draw(s, cx, cy)
	if gv.kind == sim.EntityCampfire { // the flames and the front stones, as on its recipe card
		draw(g.art.flames[(g.frame/5)%flameFrames], cx, cy+1)
		draw(g.art.campFront, cx, cy)
	}
}

// drawAimCues brackets what the pointer is on, at screen resolution: white
// for the target the left button will work on, red with "Xa quá" for one
// out of reach; and names what a left click would build.
func (g *Game) drawAimCues(screen *ebiten.Image, cam camera, scale float64, tg, far targetView) {
	if tg.ok && tg.aimed {
		if x, y, w, h, ok := g.targetBox(tg, cam); ok {
			drawBrackets(screen, x*scale, y*scale, w*scale, h*scale, colWhite)
		}
	}
	if far.ok {
		if x, y, w, h, ok := g.targetBox(far, cam); ok {
			drawBrackets(screen, x*scale, y*scale, w*scale, h*scale, colGhostBad)
			g.text(screen, "Xa quá — lại gần hơn", (x+w/2)*scale, y*scale-22, colGhostBad, text.AlignCenter, 1)
		}
	}
	if gv := g.ghost(); gv.ok {
		x, y := cam.toScreen(float64(gv.tx)+0.5, float64(gv.ty))
		name := gv.kind.Name()
		if gv.recipe {
			if r, ok := sim.RecipeByID(g.build.recipe); ok {
				name = r.Name
			}
		} else if gv.item.Valid() {
			name = gv.item.Def().Name
		}
		label, col := "Đặt "+name, colGhostOK
		switch {
		case !gv.valid && gv.why == sim.NoticeTooFar:
			label, col = name+" — xa quá", colGhostBad
		case !gv.valid:
			label, col = name+" — không đặt được ở đây", colGhostBad
		}
		g.text(screen, label, x*scale, (y-18)*scale-18, col, text.AlignCenter, 1)
	}
}

// targetBox is where tg is drawn on the world canvas: its sprite's box.
func (g *Game) targetBox(tg targetView, cam camera) (x, y, w, h float64, ok bool) {
	var s sprite
	var sx, sy float64
	switch {
	case tg.drop:
		dx, dy, lift := g.dropPos(tg.d)
		sx, sy = cam.toScreenFine(dx, dy)
		return sx - 7, sy - lift - 9, 14, 12, true
	case tg.creature:
		cx, cy, moving := g.lerpCreature(tg.c, g.alpha())
		sx, sy = cam.toScreenFine(cx, cy)
		var bob float64
		s, bob = g.creatureSprite(tg.c, moving)
		sy -= bob
	default:
		sx, sy = cam.toScreen(float64(tg.e.TX)+0.5, float64(tg.e.TY)+0.5)
		s, _ = g.entitySprite(tg.e)
	}
	if s.img == nil {
		return 0, 0, 0, 0, false
	}
	return sx - s.ax - 1, sy - s.ay - 1, s.w() + 2, s.h() + 2, true
}

// drawBrackets draws the four corners of a box, with a dark edge so they
// show on any ground.
func drawBrackets(dst *ebiten.Image, x, y, w, h float64, col color.RGBA) {
	l := max(6, min(16, math.Min(w, h)/3))
	corner := func(cx, cy, dx, dy float64, c color.RGBA, width float32) {
		vector.StrokeLine(dst, float32(cx), float32(cy), float32(cx+dx*l), float32(cy), width, c, true)
		vector.StrokeLine(dst, float32(cx), float32(cy), float32(cx), float32(cy+dy*l), width, c, true)
	}
	shadow := color.RGBA{0, 0, 0, 150}
	for _, pass := range []struct {
		c color.RGBA
		w float32
	}{{shadow, 4}, {col, 2}} {
		corner(x, y, 1, 1, pass.c, pass.w)
		corner(x+w, y, -1, 1, pass.c, pass.w)
		corner(x, y+h, 1, -1, pass.c, pass.w)
		corner(x+w, y+h, -1, -1, pass.c, pass.w)
	}
}
