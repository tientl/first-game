package client

import (
	"cmp"
	"fmt"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

var playerColors = []color.RGBA{
	{226, 92, 78, 255}, {84, 144, 232, 255}, {240, 198, 72, 255},
	{124, 204, 118, 255}, {182, 122, 216, 255}, {240, 150, 82, 255},
}

// viewHeight is how many art pixels of world the window shows vertically;
// the integer pixel scale is chosen to get close to it at any window size.
const viewHeight = 360

func pixelScale(sh int) int { return max(1, int(math.Round(float64(sh)/viewHeight))) }

// Zoom steps: how many art-pixel sizes the view is from the default.
const (
	minZoom = -3
	maxZoom = 3
)

// viewZoom is the player's zoom, minZoom to maxZoom; kept in the settings.
var viewZoom int

// worldScale is the pixel scale of the world on a screen sh pixels high, at
// the player's zoom. It never goes below 1: one art pixel to a screen pixel
// already shows the most the server sends.
func worldScale(sh int) int { return max(1, pixelScale(sh)+viewZoom) }

// camera maps world coordinates (tiles) to art pixels on screen, rounded to
// whole canvas pixels (1/r art pixel; see hd.go) so nothing shimmers while
// scrolling. r is 0 or 1 for whole art pixels.
type camera struct{ ox, oy, r float64 }

func newCamera(cx, cy float64, w, h, r int) camera {
	c := camera{r: float64(r)}
	c.ox = c.round(cx*TileW) - float64(w/2)
	c.oy = c.round(cy*TileH) - float64(h/2)
	return c
}

// round rounds v, in art pixels, to the camera's canvas pixels.
func (c camera) round(v float64) float64 {
	if c.r <= 1 {
		return math.Round(v)
	}
	return math.Round(v*c.r) / c.r
}

// toScreen maps a world point to art pixels on screen, on the ground's
// art-pixel grid: whole art pixels from the world's origin, as before HD
// art, for everything that stands still (and for anything whose place is
// worked out from noise, so it lands where it always did).
func (c camera) toScreen(x, y float64) (float64, float64) {
	return math.Round(x*TileW) - c.ox, math.Round(y*TileH) - c.oy
}

// toScreenFine is toScreen for things on the move (players, beasts,
// boats, piles in the air): rounded to canvas pixels, so they glide in
// steps of 1/R art pixel. Draw them after align(dst, sx, sy); see hd.go.
func (c camera) toScreenFine(x, y float64) (float64, float64) {
	return c.round(x*TileW) - c.ox, c.round(y*TileH) - c.oy
}

// drawable is one upright thing in the depth-sorted pass.
type drawable struct {
	y        float64 // world y of its ground point: larger draws later (in front)
	id       uint32
	isPlayer bool
	creature bool
	drop     bool    // a pile on the ground
	boat     bool    // a boat, with its rider
	lift     float64 // how high a falling pile is
	idx      int     // index into curr.Players, curr.Creatures, curr.Drops or curr.Entities
	sx, sy   float64
	moving   bool
	wx, wy   float64 // a player's place in tiles, for its stride; see limbdraw.go
	extra    bool    // part of a boss's lair that is not an entity: idx is into lair.extras; see lairrender.go
	wall     bool    // the foot of a cliff's face, over the figures by it: idx is into ground.strips; see terraincliff.go
}

// layer orders things standing on the same row: entities, then animals,
// then players in front.
func (d drawable) layer() int {
	switch {
	case d.isPlayer:
		return 2
	case d.creature:
		return 1
	default:
		return 0
	}
}

// Draw renders one frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{14, 18, 28, 255})
	if g.art == nil {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	uiScaleNow = uiScaleFor(sh) // world labels' text grows with the interface
	if !g.ready || !g.hasCurr {
		msg := "Đang tải…"
		if g.disconnected {
			msg = "Mất kết nối tới máy chủ"
		}
		g.text(screen, msg, float64(sw/2), float64(sh/2-16), colWhite, text.AlignCenter, 2)
		return
	}

	// The world is laid out in art pixels and drawn on a canvas the size of
	// the screen, scale canvas pixels to an art pixel: see hd.go.
	scale := pixelScale(sh)
	if !g.demo {
		scale = worldScale(sh)
	}
	cw, ch := (sw+scale-1)/scale, (sh+scale-1)/scale
	// The render quality may draw it on a smaller canvas, rr canvas pixels
	// to an art pixel, blown up to the screen after: see display.go.
	rr := renderScale(scale)
	canvas := screen
	frozen := g.frozen() && !g.demo // a hit-stop: the world as it was at the blow; see feel.go
	if rr < scale || (postFX || g.stop.capture) && !g.demo {
		w, h := renderSize(sw, sh, scale, rr)
		canvas = sizedImage(&g.worldHi, w, h)
		canvas.Fill(color.RGBA{14, 18, 28, 255})
	}
	if g.world == nil {
		g.world = &worldDst{}
	}

	alpha := g.alpha()
	cx, cy := float64(g.width)/2, float64(g.height)/2
	if g.demo {
		t := float64(g.frame) / 60
		cx = g.demoX + math.Sin(t*0.05)*9
		cy = g.demoY + math.Sin(t*0.037+1)*6
	}
	for _, p := range g.curr.Players {
		if p.ID == g.myID && !g.demo {
			cx, cy, _ = g.lerpPlayer(p, alpha)
		}
	}
	if !g.demo {
		cx, cy = g.introCamera(cx, cy) // a boss waking: see bossintro.go
		sx, sy := shake(g.frame)       // heavy blows, roars, stomps; see feel.go
		cx, cy = cx+sx, cy+sy
	}
	sub := 1
	if subPixel {
		sub = rr
	}
	cam := newCamera(cx, cy, cw, ch, sub)
	if !g.demo {
		g.ptr.cam, g.ptr.scale, g.ptr.sw, g.ptr.sh, g.ptr.drawn = cam, float64(scale), sw, sh, true // see mouse.go
	}
	g.updateCave() // before anything hides under, or shows from, a cave roof
	target, far := g.target()
	g.hudTarget, g.pick.view = target, g.pickTarget() // see pickup.go

	if frozen {
		g.drawStopped(screen, scale)
	} else {
		g.drawWorld(screen, canvas, cam, scale, rr, cw, ch, alpha, target)
	}
	g.drawAfterWorld(screen, cam, scale, sw, sh, alpha, target, far)
}

// drawWorld draws the world on canvas, rr canvas pixels to an art pixel, and
// that on screen (at scale) if it is not.
func (g *Game) drawWorld(screen, canvas *ebiten.Image, cam camera, scale, rr, cw, ch int, alpha float64, target targetView) {
	w := g.world
	w.begin(canvas, rr, cw, ch, cam)
	for _, stage := range []func(){
		func() { g.drawGround(w, cam) },
		func() { g.drawLairWater(w, cam, alpha) }, // the shark's deep: see lairrender.go
		func() { g.drawFish(w, cam, alpha) },
		func() { g.drawUndergrowth(w, cam) },
		func() { g.drawFX(w, cam, true, false) }, // dust on the ground; see fx.go
		func() { g.drawHDTest(w, cam) },          // the HD trial stones; see hdtest.go
		func() { g.drawSprites(w, cam, alpha, target) },
		func() { g.drawLines(w, cam, alpha) },
		func() { g.drawFX(w, cam, false, false) }, // splashes, dust, leaves
		func() { g.drawCookFX(w, cam) },           // steam and smoke; see cookrender.go
		func() { g.drawFireSmoke(w, cam) },        // see fireart.go
		func() { g.drawSky(w, cam, alpha) },       // flying reptiles overhead
		func() { g.drawLairSky(w, cam) },          // birds startled near a lair
		func() { g.drawHearts(w, cam) },           // beasts being fed; see ride.go
		func() { g.drawSnowfall(w, cam) },         // over the frozen north; see iceart.go
		func() { g.drawWeather(w, cam, alpha) },   // rain, snow, wind, tornadoes, webs; see tribalrender.go
		func() { g.drawClouds(w, cam) },
		func() { g.drawLighting(w, cam) },
		func() { g.drawGlowMasks(w, cam, g.darknessNow()) },
		func() { g.drawToadGlow(w, cam, g.darknessNow()) },        // see toadart.go
		func() { g.drawEyeshine(w, cam, alpha, g.darknessNow()) }, // the night hunters' eyes; see troodon.go
		func() { g.drawCaveBats(w, cam) },                         // bats flitting over their roost; see caveprops.go
		func() { g.drawLightning(w, cam) },
		func() { g.drawEmbers(w, cam) },
		func() { g.drawFX(w, cam, false, true) }, // sparks and slashes, bright at night
		func() { g.drawFireflies(w, cam) },
		func() {
			if !g.demo {
				g.drawGhost(w, cam) // what a left click would build; see mouse.go
			}
		},
	} {
		alignCamera(w)
		stage()
	}
	w.end()
	if canvas != screen {
		img := canvas
		if postFX && !g.demo {
			img = g.applyPost(canvas, rr) // see postfx.go
		}
		if rr < scale && g.stop.capture {
			// The picture a hit-stop holds is kept at the screen's size.
			img, rr = upscaleWorld(&g.worldUp, img, screen.Bounds().Dx(), screen.Bounds().Dy(), scale, rr), scale
		}
		presentWorld(screen, img, scale, rr) // see display.go
		g.keepStop(img)                      // the picture a hit-stop holds; see feel.go
	}
}

// drawAfterWorld draws what goes over the world: the vignette, labels and
// the interface.
func (g *Game) drawAfterWorld(screen *ebiten.Image, cam camera, scale, sw, sh int, alpha float64, target, far targetView) {

	vop := &ebiten.DrawImageOptions{}
	stretchOver(&vop.GeoM, g.art.vignette.Bounds(), sw, sh) // to the very edge: see overlay.go
	vop.ColorScale.ScaleAlpha(0.38)
	vop.Filter = ebiten.FilterLinear
	screen.DrawImage(g.art.vignette, vop)

	if g.demo {
		return
	}
	g.drawCombat(screen, cam, float64(scale), alpha) // see combatui.go
	g.drawWorldLabels(screen, cam, float64(scale), target)
	g.drawAimCues(screen, cam, float64(scale), target, far) // see mouse.go
	g.drawDigLabel(screen, cam, float64(scale))
	g.drawFishingLabels(screen, cam, float64(scale))
	g.drawBossIntro(screen, sw, sh) // letterbox and title card; see bossintro.go

	g.drawInterface(screen)
}

// drawInterface draws the interface over the world, at the screen's own
// resolution in UI pixels; see uiscale.go.
func (g *Game) drawInterface(screen *ebiten.Image) {
	ui, uw, uh := beginUI(screen)
	g.lastScreenH, g.lastScreenW = uh, uw
	g.drawFishingMessage(ui, uw, uh)
	if _, intro := g.introT(); !intro {
		g.drawHUD(ui, uw, uh) // hidden while a boss's title card is up
	}
	g.drawBossBar(ui, uw) // see bossui.go
	switch g.overlay {
	case overlayMap:
		g.drawMap(ui, uw, uh)
	case overlayBag:
		g.drawBag(ui, uw, uh)
	case overlayChest:
		g.drawChest(ui, uw, uh)
	case overlayCook:
		g.drawCook(ui, uw, uh) // see cookui.go
	case overlayCraft:
		g.drawCraft(ui, uw, uh) // see craftui.go
	case overlayPalette:
		g.drawPalette(ui, uw, uh) // see buildpalette.go
	}
	g.drawNotice(ui, uw)      // above the bag, so a failed craft or meal says why
	g.drawVictory(ui, uw, uh) // every boss has fallen; see bossui.go
	g.drawDev(ui, uw, uh)     // see devconsole.go

	if g.disconnected && !g.paused {
		g.dim(ui, uw, uh, 0.6)
		g.text(ui, "Mất kết nối tới máy chủ", float64(uw/2), float64(uh/2-16), colWhite, text.AlignCenter, 2)
		g.text(ui, "Nhấn Esc để về menu", float64(uw/2), float64(uh/2+20), colMuted, text.AlignCenter, 1)
	}
	if g.paused {
		g.drawPause(ui, uw, uh)
	}
	endUI(ui)
}

func (g *Game) drawSprites(dst drawTarget, cam camera, alpha float64, target targetView) {
	g.drawCam = cam
	list := g.drawList[:0]
	for i, e := range g.curr.Entities {
		wx, wy := float64(e.TX)+0.5, float64(e.TY)+0.5
		if g.hiddenInCave(wx, wy) {
			continue // under the rock: seen only from inside the cave
		}
		sx, sy := cam.toScreen(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(e.ID), idx: i, sx: sx, sy: sy})
	}
	for i, b := range g.curr.Boats {
		wx, wy := g.lerpBoat(b, alpha)
		sx, sy := cam.toScreenFine(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(b.ID), boat: true, idx: i, sx: sx, sy: sy})
	}
	for i, d := range g.curr.Drops {
		wx, wy, lift := g.dropPos(d)
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreenFine(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(d.ID), drop: true, idx: i, sx: sx, sy: sy, lift: lift})
	}
	for i, c := range g.curr.Creatures {
		if c.Kind == sim.CreatureFish || c.Kind == sim.CreaturePlesio || c.State == sim.CreatureHidden || inTheSky(c) || g.underIce(c) {
			continue // in the water: drawn by drawFish; down its burrow or den; or high up: drawSky
		}
		wx, wy, moving := g.lerpCreature(c, alpha)
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreenFine(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(c.ID), creature: true, idx: i, sx: sx, sy: sy, moving: moving})
	}
	riders := g.riders[:0] // players astride a beast: drawn on its back, see ride.go
	for i, p := range g.curr.Players {
		if p.Mount != 0 && !p.Dead {
			riders = append(riders, i)
			continue
		}
		if p.Boat != 0 && !p.Dead && g.boatInView(p.Boat) {
			continue // drawn in its boat
		}
		wx, wy, moving := g.lerpPlayer(p, alpha)
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreenFine(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(p.ID), isPlayer: true, idx: i, sx: sx, sy: sy, moving: moving, wx: wx, wy: wy})
	}
	list = g.lairDrawables(list, cam, float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())) // see lairrender.go
	list = g.wallDrawables(list, cam)
	slices.SortFunc(list, func(a, b drawable) int {
		if c := cmp.Compare(a.y, b.y); c != 0 {
			return c
		}
		if c := cmp.Compare(a.layer(), b.layer()); c != 0 {
			return c
		}
		return cmp.Compare(a.id, b.id)
	})
	g.drawList = list
	g.noteFigures(list) // see occlude.go
	g.prefetchArt()     // see lazyart.go

	if heroHD {
		g.noteHeroLights(cam) // see herohddraw.go
	}
	g.drawSoil(dst, list)
	g.drawFloors(dst, cam) // see tribalrender.go
	g.drawLairGround(dst, cam)
	g.drawCaveGround(dst, cam) // see caveprops.go
	g.drawDigSpot(dst, cam)
	g.drawZones(dst, cam, alpha) // red warnings where beasts' moves will land: see telegraphzone.go
	g.drawHazards(dst, cam, alpha) // clouds, webs, collapses: see cavecreatures.go
	// Each thing is drawn on the art-pixel grid through its own ground
	// point, so a moving one keeps its pixel art whole between art pixels;
	// see align in hd.go.
	for _, d := range list {
		align(dst, d.sx, d.sy)
		var w float64
		switch {
		case d.drop:
			w = 7
		case d.boat, d.extra, d.wall:
			w = 0
		case d.isPlayer && g.curr.Players[d.idx].Swimming:
			w = 0
		case d.isPlayer:
			w = 13
		case d.creature:
			w = creatureShadow(g.curr.Creatures[d.idx].Kind)
		default:
			w = g.shadowWidth(g.curr.Entities[d.idx])
		}
		if heroHD && d.isPlayer && w > 0 && g.heroStands(g.curr.Players[d.idx]) {
			g.drawHeroShadow(dst, d.sx, d.sy)
			continue
		}
		if w > 0 {
			g.drawShadow(dst, d.sx, d.sy, w)
		}
	}

	// The thing being worked on shakes while the player swings at it.
	shaking := sim.EntityID(0)
	if target.ok && !target.creature && !target.drop {
		for _, p := range g.curr.Players {
			if p.ID == g.myID && p.Acting {
				shaking = target.e.ID
			}
		}
	}
	beginCanopy(dst) // see canopyshare.go
	for _, d := range list {
		align(dst, d.sx, d.sy)
		if d.drop {
			g.drawDrop(dst, g.curr.Drops[d.idx], d.sx, d.sy, d.lift)
			continue
		}
		if d.boat {
			g.drawBoat(dst, g.curr.Boats[d.idx], d.sx, d.sy)
			continue
		}
		if d.wall {
			g.drawWallFront(dst, d)
			continue
		}
		if d.extra {
			g.drawLairExtra(dst, d)
			continue
		}
		if d.isPlayer {
			p := g.curr.Players[d.idx]
			g.drawPlayer(dst, p, d)
			if !p.Dead && !p.Swimming && !p.Asleep {
				vx := 0.0
				if q, ok := g.prevPlayers[p.ID]; ok {
					vx = float64(p.X - q.X)
				}
				g.drawFeetInGrass(dst, d.wx, d.wy, d.sx, d.sy, 22, vx) // see coverlife.go
			}
			g.drawSwarm(dst, p, d.sx, d.sy) // see jungleart.go
			g.drawDazed(dst, p, d.sx, d.sy) // stunned: see telegraphzone.go
			continue
		}
		if d.creature {
			c := g.curr.Creatures[d.idx]
			if c.Ridden { // the rider's far leg, behind it; see ridehd.go
				for _, i := range riders {
					if p := g.curr.Players[i]; p.Mount == c.ID {
						g.drawRiderBehind(dst, p, c, d)
					}
				}
			}
			g.drawCreature(dst, c, d)
			if c.Kind != sim.CreatureWorm && c.State != sim.CreatureLurk {
				wx, wy, _ := g.lerpCreature(c, g.alpha())
				vx := 0.0
				if q, ok := g.prevCreatures[c.ID]; ok {
					vx = float64(c.X - q.X)
				}
				g.drawFeetInGrass(dst, wx, wy, d.sx, d.sy, max(14, creatureShadow(c.Kind)*0.9), vx) // see coverlife.go
			}
			if c.Ridden {
				for _, i := range riders {
					if p := g.curr.Players[i]; p.Mount == c.ID {
						g.drawRider(dst, p, c, d)
					}
				}
			} else if c.Tamed {
				s, bob := g.creatureSprite(c, d.moving)
				g.drawPetMark(dst, c, d.sx, d.sy-bob-s.ay)
			}
			continue
		}
		e := g.curr.Entities[d.idx]
		if e.ID == shaking {
			d.sx += math.Round(math.Sin(float64(g.frame) * 1.3))
		}
		g.drawEntity(dst, e, d)
	}
	endCanopy()
	g.drawHiddenSelf(dst) // the player's own figure, if the leaves hide it: see hiddenself.go
	alignCamera(dst)
}

func (g *Game) shadowWidth(e protocol.EntityView) float64 {
	if sim.IsFruitStump(e.Kind, e.Var) {
		return 14 * stumpScale[fruitStumpSize(e.Kind)]
	}
	if w, ok := woodsShadow(e); ok {
		return w
	}
	if w, ok := coverShadow(e); ok { // see coverlife.go
		return w
	}
	if w, ok := lairShadow(e); ok {
		return w
	}
	if w, ok := caveShadow(e); ok { // see cavethings.go
		return w
	}
	if w, ok := tribalShadow(e); ok {
		return w
	}
	if w, ok := graveShadow(e); ok {
		return w
	}
	if w, ok := cookShadow(e); ok { // see cookrender.go
		return w
	}
	size := sizeIdx(e.Big)
	switch e.Kind {
	case sim.EntityTree:
		if e.Ready {
			return 17 * treeScale[size]
		}
		return 14 * stumpScale[size]
	case sim.EntityBoulder:
		return 28 * boulderScale[size]
	case sim.EntityBerryBush:
		return 26
	case sim.EntityFigTree, sim.EntityGinkgoTree, sim.EntityPalm:
		return 26
	case sim.EntityGrassTuft:
		return 12
	case sim.EntityTwig:
		if e.Ready {
			return 16
		}
		return 0
	case sim.EntityWall, sim.EntityStoneWall:
		return 26
	case sim.EntityChest:
		return 20
	case sim.EntityBed:
		return 30
	case sim.EntityPebble:
		if e.Ready {
			return 13
		}
		return 0
	case sim.EntityCrop:
		return 8 + 4*float64(e.Stage)
	case sim.EntityFlora:
		return g.floraShadow(e)
	case sim.EntityDen:
		return 28
	case sim.EntityCactus:
		return 16
	case sim.EntityReed:
		return 14
	case sim.EntityMushroom:
		if e.Ready {
			return 10
		}
		return 0
	case sim.EntityCoal, sim.EntityFurnace:
		return 26
	case sim.EntityCopperOre, sim.EntityIronOre, sim.EntityPlatinumOre, sim.EntityDiamondOre:
		return 28
	default:
		return 0 // burrows are dug into the ground
	}
}

// drawShadow draws a soft ground shadow, pushed a little away from the sun.
func (g *Game) drawShadow(dst drawTarget, sx, sy, w float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-shadowSize/2, -shadowSize/2)
	op.GeoM.Scale(w/shadowSize, w*0.42/shadowSize)
	op.GeoM.Translate(sx+w*0.08, sy)
	op.ColorScale.ScaleAlpha(0.34)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(g.art.shadow, op)
}

// drawAt draws s with its anchor at (x, y), scaled and rotated about the anchor.
func drawAt(dst drawTarget, s sprite, x, y, scaleX, scaleY, rot float64, tint color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Scale(scaleX, scaleY)
	if rot != 0 {
		op.GeoM.Rotate(rot)
	}
	op.GeoM.Translate(x, y)
	if tint != nil {
		op.ColorScale.ScaleWithColor(tint)
	}
	drawSprite(dst, s, op)
}

// drawSwaying draws s leaning sideways about its anchor, for wind.
func drawSwaying(dst drawTarget, s sprite, x, y, skew float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Skew(skew, 0)
	op.GeoM.Translate(x, y)
	drawSprite(dst, s, op)
}

// entitySprite returns the sprite for e and how much it sways in the wind.
func (g *Game) entitySprite(e protocol.EntityView) (sprite, float64) {
	a := g.art
	if sim.IsFruitStump(e.Kind, e.Var) {
		return a.stumps[fruitStumpSize(e.Kind)], 0 // a felled fruit tree; see fruitart.go
	}
	if s, sway, ok := g.woodsEntitySprite(e); ok {
		return s, sway // see jungleart.go
	}
	if s, sway, ok := g.tribalEntitySprite(e); ok {
		return s, sway // see tribalrender.go
	}
	h := sim.HashTile(int(e.TX), int(e.TY))
	size := sizeIdx(e.Big)
	switch e.Kind {
	case sim.EntityTree:
		if !e.Ready {
			return a.stumps[size], 0
		}
		// Skew moves the top more on a taller sprite: big trunks are stiffer.
		stiff := 1 / treeScale[size]
		tile := g.tileAt(int(e.TX), int(e.TY))
		forest := tile == sim.TileForest
		// The look follows from the tile (see sim.TreeLookAt, which the
		// woods are spaced by): prehistoric trees by terrain, giant redwoods
		// and monkey-puzzles in the woods, cycads and ginkgos in the open,
		// tree ferns in both.
		v := int(h / 8)
		switch sim.TreeLookAt(int(e.TX), int(e.TY), tile) {
		case sim.LookSnowPine: // frosted pines on the snow
			pines := a.snowPinesOf(size)
			return pines[v%len(pines)], 0.02 * stiff
		case sim.LookSwamp: // bald cypresses and dead snags; see swampart.go
			return swampTree(size, h), 0.02 * stiff
		case sim.LookRedwood:
			anc := a.ancientOf(size)
			return anc.redwoods[v%len(anc.redwoods)], 0.008 * stiff
		case sim.LookAraucaria:
			anc := a.ancientOf(size)
			return anc.araucarias[v%len(anc.araucarias)], 0.015 * stiff
		case sim.LookTreeFern:
			anc := a.ancientOf(size)
			return anc.ferns[v%len(anc.ferns)], 0.05 * stiff
		case sim.LookCycad:
			anc := a.ancientOf(size)
			return anc.cycads[v%len(anc.cycads)], 0.02 * stiff
		case sim.LookGinkgo:
			anc := a.ancientOf(size)
			return anc.ginkgos[v%len(anc.ginkgos)], 0.035 * stiff
		case sim.LookPine:
			pines := a.treesOf(size).pines
			return pines[v%len(pines)], 0.02 * stiff
		}
		set := a.treesOf(size)
		// Vines hang in the woods; out in the open only now and then.
		trees := set.trees
		if v := (h >> 12) % 10; forest && v < 6 || v == 0 {
			trees = set.vined
		}
		return trees[int(h/8)%len(trees)], 0.035 * stiff
	case sim.EntityBoulder:
		boulders := a.boulders[rockGroundOf(g.tileAt(int(e.TX), int(e.TY)))][size] // granite, sandstone, basalt...
		return boulders[int(h/8)%len(boulders)], 0
	case sim.EntityFigTree, sim.EntityGinkgoTree, sim.EntityPalm:
		return g.fruitTreeSprite(e)
	case sim.EntityBerryBush:
		if e.Ready {
			return a.bushFull, 0.03
		}
		return a.bushPicked, 0.03
	case sim.EntityGrassTuft:
		if e.Ready {
			return a.tuftFull, 0.12
		}
		return a.tuftPicked, 0
	case sim.EntityTwig:
		if g.tileAt(int(e.TX), int(e.TY)) == sim.TileSand {
			return a.driftwood, 0 // washed up on a beach
		}
		return a.twig, 0
	case sim.EntityWall:
		return a.wall, 0
	case sim.EntityStoneWall:
		return a.gear.stoneWall, 0
	case sim.EntityChest:
		return g.chestSprite(e), 0
	case sim.EntityBed:
		return a.bed, 0
	case sim.EntityLeafTent:
		return g.tentSprite(e), 0 // see tentart.go
	case sim.EntityGrave: // a fallen player's body and their things; see sim/grave.go
		return graveSprite(e), 0 // see bodyart.go
	case sim.EntityPebble:
		return a.farm.pebbles[rockGroundOf(g.tileAt(int(e.TX), int(e.TY)))], 0
	case sim.EntityFarmland:
		return a.farm.farmland, 0
	case sim.EntityCrop:
		if s, ok := newCropSprite(e); ok {
			return s, 0.05 // rice, corn, squash: see floracrop.go
		}
		return g.cropSprite(e), 0.05
	case sim.EntityFlora:
		return g.floraEntitySprite(e) // see florarender.go
	case sim.EntityBurrow:
		return a.beasts.burrow, 0
	case sim.EntityDen:
		return a.beasts.den, 0
	case sim.EntityCactus:
		if e.Ready {
			return a.open.cactus, 0
		}
		return a.open.cactusCut, 0
	case sim.EntityReed:
		if e.Ready {
			return a.open.reed, 0.1
		}
		return a.open.reedCut, 0
	case sim.EntityMushroom:
		if g.onLog(e) {
			return mushroomLog(e.Ready), 0 // the log stays when they are picked
		}
		return a.open.mushroom, 0
	case sim.EntityCoal:
		return g.coalSeam(e), 0
	case sim.EntityClam:
		return a.open.clams, 0
	case sim.EntityFurnace:
		return g.furnaceSprite(e), 0
	case sim.EntityClay, sim.EntityMortar, sim.EntityOven, sim.EntityDryRack:
		return g.cookSprite(e), 0 // see cookrender.go
	default:
		if tier, ok := sim.OreTier(e.Kind); ok {
			return g.oreRock(e, tier), 0
		}
		return a.campBack, 0
	}
}

func (g *Game) drawEntity(dst drawTarget, e protocol.EntityView, d drawable) {
	switch {
	case (e.Kind == sim.EntityTwig || e.Kind == sim.EntityPebble || e.Kind == sim.EntityMushroom && !g.onLog(e) || e.Kind == sim.EntityClam) && !e.Ready:
		return // picked up; another will turn up here in time
	case e.Kind == sim.EntityFarmland:
		return // flat: drawn with the soil, under everything
	case e.Kind == sim.EntityGiantRoot:
		return // part of the giant, drawn whole from its front
	case e.Kind == sim.EntityGrave && g.graveFalling(e):
		return // its player is still falling; see motiondraw.go
	case e.Kind == sim.EntityLairProp:
		g.drawLairProp(dst, e, d) // see lairrender.go
		return
	case g.drawCaveThing(dst, e, d): // see cavethings.go
		return
	case g.floraHidden(e):
		return // under the ice
	}
	g.drawFloraUnder(dst, e, d)                                       // see florarender.go
	if g.drawTribalEntity(dst, e, d) || g.drawCookEntity(dst, e, d) { // a pot on the fire; see cookrender.go
		return
	}
	if g.drawTentEntity(dst, e, d) { // see tentart.go
		return
	}
	if g.drawCoverEntity(dst, e, d) { // tall grass, bushes, flowers: see coverlife.go
		return
	}
	s, sway := g.entitySprite(e)
	if sway != 0 {
		// A slow gust travelling across the map, plus each plant's own rhythm.
		t := float64(g.frame)
		gust := 0.55 + 0.45*math.Sin(t*0.011+float64(e.TX)*0.21+float64(e.TY)*0.13)
		sway *= gust * math.Sin(t*0.045+float64(sim.HashTile(int(e.TX), int(e.TY))%628)/100)
	}
	tint := g.treeTint()
	if tint != nil && !((e.Kind == sim.EntityTree || e.Kind == sim.EntityGiantTree) && e.Ready && g.tileAt(int(e.TX), int(e.TY)) != sim.TileSnow) {
		tint = nil
	}
	solid := g.seeThrough(e, s, d) // fading while it hides someone; see occlude.go
	noteCanopy(s, d.sx, d.sy)      // see canopyshare.go
	if tint != nil || solid < 1 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.ax, -s.ay)
		op.GeoM.Skew(sway, 0)
		op.GeoM.Translate(d.sx, d.sy)
		if tint != nil {
			op.ColorScale.ScaleWithColor(tint)
		}
		op.ColorScale.ScaleAlpha(solid)
		drawSprite(dst, s, op)
	} else {
		drawSwaying(dst, s, d.sx, d.sy, sway)
	}
	if e.Kind == sim.EntityCampfire {
		if e.Fuel > 0 && !g.drawFireOf(dst, e, campFire(e.Fuel), d.sx, d.sy+1) { // see fireart.go
			size := 0.6 + 0.4*min(1, float64(e.Fuel)/sim.FireStartFuel)
			f := g.art.flames[(g.frame/5+int(e.ID))%flameFrames]
			drawAt(dst, f, d.sx, d.sy+1, size, size, 0, nil)
		}
		drawAt(dst, g.art.campFront, d.sx, d.sy, 1, 1, 0, nil)
	}
	if e.Kind == sim.EntityFurnace && e.Fuel > 0 && !g.drawFireOf(dst, e, fireMouth, d.sx, d.sy-1) {
		f := g.art.flames[(g.frame/4+int(e.ID))%flameFrames]
		drawAt(dst, f, d.sx, d.sy-1, 0.45, 0.4, 0, nil) // in its mouth
	}
	if hasHive(e) {
		g.drawHive(dst, e, s, d.sx, d.sy, solid) // see jungleart.go
	}
	g.drawOvenFire(dst, e, d) // see cookrender.go
}

func (g *Game) drawPlayer(dst drawTarget, p protocol.PlayerView, d drawable) {
	if p.Dead {
		if g.drawDying(dst, p, d) { // the fall first; see motiondraw.go
			return
		}
		drawAt(dst, g.art.tombstone, d.sx, d.sy, 1, 1, 0, nil)
		return
	}
	if p.Asleep {
		g.drawSleeper(dst, p, d)
		return
	}
	if age, ok := g.jolting(p.ID); ok { // struck: thrown back, flashing; see jolt.go
		g.drawJolted(dst, age, p.Dir&3, d.sx, d.sy, func(buf drawTarget, x, y float64) {
			at := d
			at.sx, at.sy = x, y
			g.drawStanding(buf, p, at)
		})
		return
	}
	g.drawStanding(dst, p, d)
}

// backpedalling reports whether p walks against the way they face, as
// they may now that they face the pointer.
func (g *Game) backpedalling(p protocol.PlayerView) bool {
	q, ok := g.prevPlayers[p.ID]
	if !ok {
		return false
	}
	fx, fy := p.Dir.Delta()
	return float64(p.X-q.X)*float64(fx)+float64(p.Y-q.Y)*float64(fy) < -1e-4
}

// drawStanding draws a player who is up: swimming, standing or walking.
func (g *Game) drawStanding(dst drawTarget, p protocol.PlayerView, d drawable) {
	if p.Swimming {
		g.drawSwimmer(dst, p, d) // see searender.go
		return
	}
	if heroHD {
		g.drawHeroHD(dst, p, d) // the trial hero; see herohddraw.go
		return
	}
	if playerHD && playerJointed && g.fordWading(d) { // knee-deep in a ford; see swimhd.go
		g.drawWader(dst, p, d)
		g.drawAilments(dst, p, d)
		return
	}
	if playerHD && playerJointed { // see limbdraw.go
		dir := p.Dir & 3
		lo, up := g.stride(p, dir, d.wx, d.wy, d.moving)
		g.drawFigureJ(dst, p, dir, lo, up, d.sx, d.sy, p.Armor)
		g.drawAilments(dst, p, d)
		return
	}
	frame, bob := 0, 0.0
	if d.moving {
		frame = walkFrameAt(g.frame)
		if g.backpedalling(p) {
			frame = (walkFrames - frame) % walkFrames // facing the pointer and walking away from it: the steps run backwards
		}
		bob = walkBob(frame)
	} else if (g.frame/40)%2 == 0 {
		bob = 0 // idle: breathing would go here
	}
	dir := p.Dir & 3
	tint := playerColors[int(p.ID)%len(playerColors)]
	step := 0.0
	if d.moving {
		step = walkSwing(frame)
	}

	// The hand: while working, the tool the job takes, swung in time with the
	// hit (see swing.go); else what is held. When facing away it is behind
	// the body.
	hand := func() {
		switch {
		case g.cleaving(p): // outlasts the working pose; see battleaxe.go
			g.drawCleave(dst, p, d, bob)
		case p.Acting && isSword(p.Held):
			g.drawSlash(dst, p, d, bob)
		case p.Acting && g.swingWhat[p.ID].kind != swingNone:
			g.drawSwing(dst, p, d, bob, step)
		default:
			g.drawHeld(dst, p.Held, dir, d.sx, d.sy, bob, step)
		}
	}
	// A blow by hand throws the arm itself out, and a struck player flings
	// both: see punch.go and jolt.go.
	fp, arms := g.punchPose(p)
	if age, ok := g.jolting(p.ID); ok && !arms && age < joltFling {
		fp, arms = flingPose(p), true
	}
	inFist := arms && fp.hand != poseRest // drawFigure draws what the fist holds
	if heldBehind(dir) && !inFist {
		hand()
	}
	switch {
	case arms:
		g.drawFigure(dst, p, dir, frame, d.sx, d.sy, bob, tint, p.Armor, fp)
	case p.Armor > sim.TierStone:
		g.drawArmored(dst, p, dir, frame, d.sx, d.sy, bob, tint)
	default:
		drawBody(dst, g.art.bodyIn(p.Look, p.Wear), dir, frame, d.sx, d.sy, 1, bob, tint)
	}
	if !heldBehind(dir) && !inFist {
		hand()
	}
	g.drawAilments(dst, p, d)
}

// drawAilments draws what clings to my own player: a web, the bubbles of
// poison.
func (g *Game) drawAilments(dst drawTarget, p protocol.PlayerView, d drawable) {
	if p.ID == g.myID {
		if g.curr.Self.Webbed {
			drawAt(dst, g.art.tribal.webbed, d.sx, d.sy, 1, 1, 0, nil)
		}
		if g.curr.Self.Poisoned && (g.frame/8)%3 == 0 {
			for k := range 3 {
				ph := math.Mod(float64(g.frame)/40+float64(k)*0.33, 1)
				fillRect(dst, float32(math.Round(d.sx-5+float64(k)*5)), float32(math.Round(d.sy-playerTop-ph*10)), 2, 2, color.RGBA{120, 220, 80, uint8(220 * (1 - ph))}, false)
			}
		}
	}
}

// drawSleeper draws p lying in bed, head on the pillow, under a blanket.
// The server snaps a sleeper onto the bed's tile, so the bed is at (sx, sy).
func (g *Game) drawSleeper(dst drawTarget, p protocol.PlayerView, d drawable) {
	if playerHD && playerJointed { // laid down under a fur, breathing; see sleephd.go
		g.drawSleeperHD(dst, p, d)
		return
	}
	if e, ok := g.tentUnder(p); ok { // in a tent: lying further in, its shell over them; see tentart.go
		defer g.drawTentShell(dst, e, math.Round(d.sx), math.Round(d.sy))
		d.sx, d.sy = d.sx+tentSleepDX, d.sy+tentSleepDY
	}
	tint := playerColors[int(p.ID)%len(playerColors)]
	// Turned a quarter left: the feet anchor goes to the foot of the bed and
	// the head to the pillow. Breathing lifts the blanket a pixel.
	fx, fy := d.sx+10, d.sy-6
	b := g.art.bodyIn(p.Look, p.Wear)
	if playerHD && playerJointed { // see limbdraw.go
		// The tall figure is longer than the bed: its legs are under the
		// blanket, and only the rest of it is drawn.
		drawFig(dst, figDraw{b: b, up: upKey{dir: sim.DirDown, anim: upAsleep}, tint: tint, rot: -math.Pi / 2, abed: true}, fx+9, fy)
		b = nil
	}
	if b != nil {
		drawAt(dst, b.back[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, nil)
		drawAt(dst, b.legs[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, nil)
		drawAt(dst, b.shirt[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, tint)
		drawAt(dst, b.asleep, fx, fy, 1, 1, -math.Pi/2, nil)
	}
	breath := 0.0
	if (g.frame/50)%2 == 0 {
		breath = 1
	}
	drawAt(dst, g.art.blanket, d.sx+5, d.sy-1-breath, 1, 1, 0, nil)
}

// drawTool draws the tool s swinging in p's hand.

// drawClouds drifts soft cloud shadows over the land by day.
func (g *Game) drawClouds(dst drawTarget, cam camera) {
	amb := sim.Ambient(g.curr.Tick)
	if amb < 0.3 {
		return
	}
	const cellW, cellH, cloudScale = 420.0, 300.0, 2.0
	driftX, driftY := float64(g.frame)*0.11, float64(g.frame)*0.04
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	cb := g.art.cloud.Bounds()
	cwid, chei := float64(cb.Dx())*cloudScale, float64(cb.Dy())*cloudScale
	x0 := int(math.Floor((cam.ox - driftX - cwid) / cellW))
	x1 := int(math.Floor((cam.ox - driftX + w) / cellW))
	y0 := int(math.Floor((cam.oy - driftY - chei) / cellH))
	y1 := int(math.Floor((cam.oy - driftY + h) / cellH))
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			if hashf(cx, cy, 400) > 0.6 {
				continue
			}
			x := float64(cx)*cellW + hashf(cx, cy, 401)*(cellW-cwid*0.5) + driftX - cam.ox
			y := float64(cy)*cellH + hashf(cx, cy, 402)*(cellH-chei*0.5) + driftY - cam.oy
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(cloudScale, cloudScale)
			op.GeoM.Translate(math.Round(x), math.Round(y))
			op.ColorScale.ScaleAlpha(float32(0.2 * amb))
			op.Filter = ebiten.FilterLinear
			dst.DrawImage(g.art.cloud, op)
		}
	}
}

// drawEmbers draws sparks rising from every burning campfire. They are a
// pure function of time, so there is no particle state to keep. At HD the
// fires have their own (drawFireFX in fireart.go).
func (g *Game) drawEmbers(dst drawTarget, cam camera) {
	if fireHD {
		g.drawFireFX(dst, cam)
		return
	}
	t := float64(g.frame)
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityCampfire || e.Fuel <= 0 {
			continue
		}
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		const n = 7
		for i := 0; i < n; i++ {
			seed := float64(e.ID)*0.37 + float64(i)*1.618
			ph := math.Mod(t/(70+float64(i)*9)+seed, 1)
			x := sx + math.Sin(seed*7)*4 + math.Sin(ph*5+seed)*3*ph
			y := sy - 8 - ph*30
			col := fireRamp[3]
			if ph > 0.5 {
				col = fireRamp[2]
			}
			a := uint8(255 * (1 - ph) * (1 - ph))
			fillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 1, 1,
				color.RGBA{uint8(int(col.R) * int(a) / 255), uint8(int(col.G) * int(a) / 255), uint8(int(col.B) * int(a) / 255), a}, false)
		}
	}
}

// target returns the entity or animal the left button (or Space) would
// work on right now, using the same rule as the server (sim/aim.go), and
// far: what is under the pointer but out of reach, for the red cue.
func (g *Game) target() (tg, far targetView) {
	if g.curr.Self.Dead {
		return targetView{}, targetView{}
	}
	p, ok := g.me()
	if !ok || p.Cast != sim.CastNone {
		return targetView{}, targetView{} // fishing: the button strikes, it works on nothing else
	}
	ax, ay, aim := g.aim()
	if aim && g.overlay == overlayNone {
		if g.build.on {
			return targetView{}, targetView{} // building: the left button places
		}
		u, in := g.underPointer(p, ax, ay)
		if u.ok && in {
			return u, targetView{}
		}
		if u.ok {
			far = u
		}
		if g.ghost().ok {
			return targetView{}, far // a left click sets down what is held
		}
	}
	if _, _, ok := g.digSpot(); ok {
		return targetView{}, far // a held hoe tills the tile in front: marked by drawDigSpot
	}
	x, y := float64(p.X), float64(p.Y)
	// Never a pile or a loose thing: E picks those up (pickTarget, pickup.go).
	t, ok := sim.ResolveTarget(x, y, aim, ax, ay, g.curr.Self.Inv.HasFuel(), g.candidateAt,
		creatureCandidates(g.curr.Creatures, x, y), sim.ReachBonusFor(p.Held)+sim.RideReach(p.Mount != 0))
	if !ok {
		return targetView{}, far
	}
	tg, _ = g.viewOf(t)
	return tg, far
}

func (g *Game) candidateAt(tx, ty int) (sim.TargetCandidate, bool) {
	i, ok := g.entAt[[2]int{tx, ty}]
	if !ok {
		return sim.TargetCandidate{}, false
	}
	e := g.curr.Entities[i]
	return sim.TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready, Var: e.Var}, true
}

// labelLift is the highest a thing's name floats above its foot, in art
// pixels: trees stand several times the player's height, and a name over the
// top of the crown would be far from where the player is looking.
const labelLift = 2.5 * playerArtH

// drawWorldLabels draws, at full screen resolution, what must stay readable
// at night: the name of what the pointer is on (see uilabels.go) and other
// players' names.
func (g *Game) drawWorldLabels(screen *ebiten.Image, cam camera, scale float64, tg targetView) {
	alpha := g.alpha()
	var cur floatLabel
	tg = g.labelled(tg) // or what E picks up there; see pickup.go
	if tg.ok && tg.aimed {
		var sx, sy, top float64
		var label string
		col := colWhite
		if tg.drop {
			x, y, lift := g.dropPos(tg.d)
			sx, sy = cam.toScreenFine(x, y)
			top = sy - lift - 14
			label = "E: nhặt " + tg.d.Item.Def().Name
			if tg.d.Count > 1 {
				label += fmt.Sprintf(" ×%d", tg.d.Count)
			}
		} else if tg.creature {
			x, y, moving := g.lerpCreature(tg.c, alpha)
			sx, sy = cam.toScreenFine(x, y)
			s, bob := g.creatureSprite(tg.c, moving)
			top = sy - bob - s.ay
			label, col = g.creatureLabel(tg.c)
		} else {
			e := tg.e
			sx, sy = cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
			s, _ := g.entitySprite(e)
			top = sy - min(s.ay, labelLift) // a tree is named across its crown, not above it
			label = sim.EntityLabel(e.Kind, e.Var)
			if e.Kind.Sized() {
				label += [2]string{" nhỏ", " lớn"}[sizeIdx(e.Big)]
			}
			switch {
			case sim.LooseItem(e.Kind): // picked up with E, not the left button; see pickup.go
				label = "E: nhặt " + label
			case hasHive(e) && e.Fuel == 0: // see jungleart.go
				label = "Tổ ong — lấy mật (cầm đuốc cho ong khỏi đốt)"
				if g.myHeld() != sim.ItemTorch {
					col = colBad
				}
			case g.cookLabel(e) != "": // see cookrender.go
				label = g.cookLabel(e)
			case e.Kind == sim.EntityGiantTree && !g.has(sim.ToolAxe):
				label, col = label+" — cần rìu, chặt rất lâu", colBad
			case e.Kind == sim.EntityGiantTree:
				label += " — chặt (rất lâu)"
			case e.Kind == sim.EntityCampfire:
				label += " — thêm củi"
				if g.hasRawFood() {
					label += " · chuột phải: nướng"
				}
			case e.Kind == sim.EntityTree && !g.has(sim.ToolAxe):
				label, col = label+" — cần rìu (C: chế tạo)", colBad
			case e.Kind == sim.EntityWall:
				if !g.has(sim.ToolAxe) {
					label, col = label+" — cần rìu để dỡ", colBad
				} else {
					label += " — dỡ"
				}
			case e.Kind == sim.EntityGrave:
				label = graveLabel(e) // see bodyart.go
			case e.Kind == sim.EntityChest:
				label += " — mở"
			case e.Kind == sim.EntityStoneWall:
				if !g.has(sim.ToolPickaxe) {
					label, col = label+" — cần cúp để dỡ", colBad
				} else {
					label += " — dỡ"
				}
			case sim.IsBedLike(e.Kind):
				label = g.bedLabel(label) // see tentart.go
			case (e.Kind == sim.EntityBoulder || e.Kind == sim.EntityCoal) && !g.has(sim.ToolPickaxe):
				label, col = label+" — cần cúp đá", colBad
			case e.Kind == sim.EntityCactus && !g.has(sim.ToolAxe):
				label, col = label+" — cần rìu", colBad
			case e.Kind == sim.EntityFurnace:
				label += " — thêm than, củi"
				if e.Fuel > 0 {
					label += " · đang cháy: nung ở bảng chế tạo (C)"
				}
			case !g.canMine(e.Kind):
				need, _ := orePick(e.Kind)
				label, col = label+" — cần "+sim.ToolItem(sim.ToolPickaxe, need).Def().Name, colBad
			case e.Kind == sim.EntityFarmland, e.Kind == sim.EntityCrop:
				label, col = g.farmLabel(e, label, col) // see florarender.go
			case e.Kind == sim.EntityFlora:
				label, col = g.floraLabel(e, label, col)
			case sim.IsFruitTree(e.Kind) && e.Ready:
				label += " — hái quả"
			case sim.IsFruitTree(e.Kind) && !g.has(sim.ToolAxe):
				label, col = label+" — hết quả, cần rìu (C: chế tạo)", colBad
			case sim.IsFruitTree(e.Kind):
				label += " — hết quả, chặt"
			default:
				label, col = g.tribalEntityLabel(e, label, col)
			}
		}
		// Over the thing's top, but on the screen however tall it is.
		topArt := max(top-5, (6+18*uiK())/scale)
		cur = floatLabel{ok: true, key: targetKey(tg), text: label, col: labelColor(col), x: sx * scale, y: topArt * scale}
	}
	g.showLabel(screen, cur)
	for _, p := range g.curr.Players {
		wx, wy, _ := g.lerpPlayer(p, alpha)
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreenFine(wx, wy)
		if p.Asleep {
			// Two z's drifting up from the pillow.
			for i := range 2 {
				ph := math.Mod(float64(g.frame)/90+float64(i)*0.5, 1)
				zx, zy := (sx-4+ph*6)*scale, (sy-14-ph*12)*scale
				g.text(screen, "z", zx, zy, color.RGBA{220, 226, 255, uint8(255 * (1 - ph))}, text.AlignCenter, (1+float64(i))*uiK())
			}
		}
		if p.ID == g.myID {
			continue
		}
		nameTag(screen, p.Name, sx*scale, (sy-playerTop)*scale-5*uiK())
	}
}
