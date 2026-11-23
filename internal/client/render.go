package client

import (
	"cmp"
	"fmt"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

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

// camera maps world coordinates (tiles) to whole art pixels, so nothing
// shimmers while scrolling.
type camera struct{ ox, oy float64 }

func newCamera(cx, cy float64, w, h int) camera {
	return camera{
		ox: math.Round(cx*TileW) - float64(w/2),
		oy: math.Round(cy*TileH) - float64(h/2),
	}
}

func (c camera) toScreen(x, y float64) (float64, float64) {
	return math.Round(x*TileW) - c.ox, math.Round(y*TileH) - c.oy
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
	g.lastScreenH = sh
	if !g.ready || !g.hasCurr {
		msg := "Đang tải…"
		if g.disconnected {
			msg = "Mất kết nối tới máy chủ"
		}
		g.text(screen, msg, float64(sw/2), float64(sh/2-16), colWhite, text.AlignCenter, 2)
		return
	}

	// The world is drawn in art pixels on its own canvas, then scaled up.
	scale := pixelScale(sh)
	if !g.demo {
		scale = worldScale(sh)
	}
	cw, ch := (sw+scale-1)/scale, (sh+scale-1)/scale
	if g.world == nil || g.world.Bounds().Dx() != cw || g.world.Bounds().Dy() != ch {
		if g.world != nil {
			g.world.Deallocate()
		}
		g.world = ebiten.NewImage(cw, ch)
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
	cam := newCamera(cx, cy, cw, ch)
	g.updateCave() // before anything hides under, or shows from, a cave roof
	target := g.target()

	g.ground.draw(g.world, cam.ox, cam.oy, g.frame)
	g.drawFish(g.world, cam, alpha)
	g.drawUndergrowth(g.world, cam)
	g.drawSprites(g.world, cam, alpha, target)
	g.drawLines(g.world, cam, alpha)
	g.drawSky(g.world, cam, alpha)     // flying reptiles overhead
	g.drawHearts(g.world, cam)         // beasts being fed; see ride.go
	g.drawSnowfall(g.world, cam)       // over the frozen north; see iceart.go
	g.drawWeather(g.world, cam, alpha) // rain, snow, wind, tornadoes, webs; see tribalrender.go
	g.drawClouds(g.world, cam)
	g.drawLighting(g.world, cam)
	g.drawGlowMasks(g.world, cam, g.darknessNow())
	g.drawLightning(g.world, cam)
	g.drawEmbers(g.world, cam)
	g.drawFireflies(g.world, cam)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(g.world, op)

	vop := &ebiten.DrawImageOptions{}
	vb := g.art.vignette.Bounds()
	vop.GeoM.Scale(float64(sw)/float64(vb.Dx()), float64(sh)/float64(vb.Dy()))
	vop.ColorScale.ScaleAlpha(0.38)
	vop.Filter = ebiten.FilterLinear
	screen.DrawImage(g.art.vignette, vop)

	if g.demo {
		return
	}
	g.drawWorldLabels(screen, cam, float64(scale), target)
	g.drawDigLabel(screen, cam, float64(scale))
	g.drawFishingLabels(screen, cam, float64(scale), sw, sh)
	g.drawHUD(screen, sw, sh)
	switch g.overlay {
	case overlayMap:
		g.drawMap(screen, sw, sh)
	case overlayBag:
		g.drawBag(screen, sw, sh)
	case overlayChest:
		g.drawChest(screen, sw, sh)
	}
	g.drawNotice(screen, sw) // above the bag, so a failed craft or meal says why

	if g.disconnected && !g.paused {
		g.dim(screen, sw, sh, 0.6)
		g.text(screen, "Mất kết nối tới máy chủ", float64(sw/2), float64(sh/2-16), colWhite, text.AlignCenter, 2)
		g.text(screen, "Nhấn Esc để về menu", float64(sw/2), float64(sh/2+20), colMuted, text.AlignCenter, 1)
	}
	if g.paused {
		g.drawPause(screen, sw, sh)
	}
}

func (g *Game) drawSprites(dst *ebiten.Image, cam camera, alpha float64, target targetView) {
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
		sx, sy := cam.toScreen(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(b.ID), boat: true, idx: i, sx: sx, sy: sy})
	}
	for i, d := range g.curr.Drops {
		wx, wy, lift := g.dropPos(d)
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreen(wx, wy)
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
		sx, sy := cam.toScreen(wx, wy)
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
		sx, sy := cam.toScreen(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(p.ID), isPlayer: true, idx: i, sx: sx, sy: sy, moving: moving})
	}
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

	g.drawSoil(dst, list)
	g.drawFloors(dst, cam) // see tribalrender.go
	g.drawDigSpot(dst, cam)
	for _, d := range list {
		var w float64
		switch {
		case d.drop:
			w = 7
		case d.boat:
			w = 0
		case d.isPlayer && g.curr.Players[d.idx].Swimming:
			w = 0
		case d.isPlayer:
			w = 13
		case d.creature:
			w = creatureShadow(g.curr.Creatures[d.idx].Kind)
		default:
			w = shadowWidth(g.curr.Entities[d.idx])
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
	for _, d := range list {
		if d.drop {
			g.drawDrop(dst, g.curr.Drops[d.idx], d.sx, d.sy, d.lift)
			continue
		}
		if d.boat {
			g.drawBoat(dst, g.curr.Boats[d.idx], d.sx, d.sy)
			continue
		}
		if d.isPlayer {
			g.drawPlayer(dst, g.curr.Players[d.idx], d)
			continue
		}
		if d.creature {
			c := g.curr.Creatures[d.idx]
			g.drawCreature(dst, c, d)
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
}

func shadowWidth(e protocol.EntityView) float64 {
	if w, ok := tribalShadow(e); ok {
		return w
	}
	size := sizeIdx(e.Big)
	switch e.Kind {
	case sim.EntityTree:
		if e.Ready {
			return 28 * treeScale[size]
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
func (g *Game) drawShadow(dst *ebiten.Image, sx, sy, w float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-shadowSize/2, -shadowSize/2)
	op.GeoM.Scale(w/shadowSize, w*0.42/shadowSize)
	op.GeoM.Translate(sx+w*0.08, sy)
	op.ColorScale.ScaleAlpha(0.34)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(g.art.shadow, op)
}

// drawAt draws s with its anchor at (x, y), scaled and rotated about the anchor.
func drawAt(dst *ebiten.Image, s sprite, x, y, scaleX, scaleY, rot float64, tint color.Color) {
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
	dst.DrawImage(s.img, op)
}

// drawSwaying draws s leaning sideways about its anchor, for wind.
func drawSwaying(dst *ebiten.Image, s sprite, x, y, skew float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Skew(skew, 0)
	op.GeoM.Translate(x, y)
	dst.DrawImage(s.img, op)
}

// entitySprite returns the sprite for e and how much it sways in the wind.
func (g *Game) entitySprite(e protocol.EntityView) (sprite, float64) {
	a := g.art
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
		if g.tileAt(int(e.TX), int(e.TY)) == sim.TileSnow { // frosted pines on the snow
			pines := a.snowPines[size]
			return pines[int(h/8)%len(pines)], 0.02 * stiff
		}
		tile := g.tileAt(int(e.TX), int(e.TY))
		if tile == sim.TileMarsh { // bald cypresses and dead snags; see swampart.go
			return swampTree(size, h), 0.02 * stiff
		}
		forest := tile == sim.TileForest
		// Prehistoric trees, by terrain: giant redwoods and monkey-puzzles
		// in the woods, cycads and ginkgos in the open, tree ferns in both.
		anc, v := &a.ancient[size], int(h/8)
		switch pick := (h >> 20) % 20; {
		case forest && pick < 3:
			return anc.redwoods[v%len(anc.redwoods)], 0.008 * stiff
		case forest && pick < 5, tile == sim.TileGrass && pick < 1:
			return anc.araucarias[v%len(anc.araucarias)], 0.015 * stiff
		case forest && pick < 7, tile == sim.TileGrass && pick < 4:
			return anc.ferns[v%len(anc.ferns)], 0.05 * stiff
		case tile == sim.TileGrass && pick < 8:
			return anc.cycads[v%len(anc.cycads)], 0.02 * stiff
		case tile == sim.TileGrass && pick < 11:
			return anc.ginkgos[v%len(anc.ginkgos)], 0.035 * stiff
		}
		if forest && h%5 < 1 {
			pines := a.pines[size]
			return pines[int(h/8)%len(pines)], 0.02 * stiff
		}
		// Vines hang in the woods; out in the open only now and then.
		trees := a.trees[size]
		if v := (h >> 12) % 10; forest && v < 6 || v == 0 {
			trees = a.vineTrees[size]
		}
		return trees[int(h/8)%len(trees)], 0.035 * stiff
	case sim.EntityBoulder:
		boulders := a.boulders[size]
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
		return a.twig, 0
	case sim.EntityWall:
		return a.wall, 0
	case sim.EntityStoneWall:
		return a.gear.stoneWall, 0
	case sim.EntityChest:
		return g.chestSprite(e), 0
	case sim.EntityBed:
		return a.bed, 0
	case sim.EntityPebble:
		return a.farm.pebbles, 0
	case sim.EntityFarmland:
		return a.farm.farmland, 0
	case sim.EntityCrop:
		return g.cropSprite(e), 0.05
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
		return a.open.coal, 0
	case sim.EntityClam:
		return a.open.clams, 0
	case sim.EntityFurnace:
		return g.furnaceSprite(e), 0
	default:
		if tier, ok := sim.OreTier(e.Kind); ok {
			return g.oreRock(e, tier), 0
		}
		return a.campBack, 0
	}
}

func (g *Game) drawEntity(dst *ebiten.Image, e protocol.EntityView, d drawable) {
	switch {
	case (e.Kind == sim.EntityTwig || e.Kind == sim.EntityPebble || e.Kind == sim.EntityMushroom && !g.onLog(e) || e.Kind == sim.EntityClam) && !e.Ready:
		return // picked up; another will turn up here in time
	case e.Kind == sim.EntityFarmland:
		return // flat: drawn with the soil, under everything
	}
	if g.drawTribalEntity(dst, e, d) {
		return
	}
	s, sway := g.entitySprite(e)
	if sway != 0 {
		// A slow gust travelling across the map, plus each plant's own rhythm.
		t := float64(g.frame)
		gust := 0.55 + 0.45*math.Sin(t*0.011+float64(e.TX)*0.21+float64(e.TY)*0.13)
		sway *= gust * math.Sin(t*0.045+float64(sim.HashTile(int(e.TX), int(e.TY))%628)/100)
	}
	if tint := g.treeTint(); tint != nil && e.Kind == sim.EntityTree && e.Ready && g.tileAt(int(e.TX), int(e.TY)) != sim.TileSnow {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.ax, -s.ay)
		op.GeoM.Skew(sway, 0)
		op.GeoM.Translate(d.sx, d.sy)
		op.ColorScale.ScaleWithColor(tint)
		dst.DrawImage(s.img, op)
	} else {
		drawSwaying(dst, s, d.sx, d.sy, sway)
	}
	if e.Kind == sim.EntityCampfire {
		if e.Fuel > 0 {
			size := 0.6 + 0.4*min(1, float64(e.Fuel)/sim.FireStartFuel)
			f := g.art.flames[(g.frame/5+int(e.ID))%flameFrames]
			drawAt(dst, f, d.sx, d.sy+1, size, size, 0, nil)
		}
		drawAt(dst, g.art.campFront, d.sx, d.sy, 1, 1, 0, nil)
	}
	if e.Kind == sim.EntityFurnace && e.Fuel > 0 {
		f := g.art.flames[(g.frame/4+int(e.ID))%flameFrames]
		drawAt(dst, f, d.sx, d.sy-1, 0.45, 0.4, 0, nil) // in its mouth
	}
}

func (g *Game) drawPlayer(dst *ebiten.Image, p protocol.PlayerView, d drawable) {
	if p.Dead {
		drawAt(dst, g.art.tombstone, d.sx, d.sy, 1, 1, 0, nil)
		return
	}
	if p.Asleep {
		g.drawSleeper(dst, p, d)
		return
	}
	if p.Swimming {
		g.drawSwimmer(dst, p, d) // see searender.go
		return
	}
	frame, bob := 0, 0.0
	if d.moving {
		frame = (g.frame / 7) % walkFrames
		if frame%2 == 1 {
			bob = 1
		}
	} else if (g.frame/40)%2 == 0 {
		bob = 0 // idle: breathing would go here
	}
	dir := p.Dir & 3
	tint := playerColors[int(p.ID)%len(playerColors)]
	step := 0
	if d.moving {
		step = walkStep(frame)
	}

	// The hand: while working, the tool the job takes, swung in time with the
	// hit (see swing.go); else what is held. When facing away it is behind
	// the body.
	hand := func() {
		switch {
		case p.Acting && isSword(p.Held):
			g.drawSlash(dst, p, d, bob)
		case p.Acting && g.swingWhat[p.ID].kind != swingNone:
			g.drawSwing(dst, p, d, bob, step)
		default:
			g.drawHeld(dst, p.Held, dir, d.sx, d.sy, bob, step)
		}
	}
	if heldBehind(dir) {
		hand()
	}
	if p.Armor > sim.TierStone {
		g.drawArmored(dst, p, dir, frame, d.sx, d.sy, bob, tint)
	} else {
		drawBody(dst, g.art.body(p.Look), dir, frame, d.sx, d.sy, 1, bob, tint)
	}
	if !heldBehind(dir) {
		hand()
	}
	if p.ID == g.myID {
		if g.curr.Self.Webbed {
			drawAt(dst, g.art.tribal.webbed, d.sx, d.sy, 1, 1, 0, nil)
		}
		if g.curr.Self.Poisoned && (g.frame/8)%3 == 0 {
			for k := range 3 {
				ph := math.Mod(float64(g.frame)/40+float64(k)*0.33, 1)
				vector.FillRect(dst, float32(math.Round(d.sx-5+float64(k)*5)), float32(math.Round(d.sy-30-ph*10)), 2, 2, color.RGBA{120, 220, 80, uint8(220 * (1 - ph))}, false)
			}
		}
	}
}

// drawSleeper draws p lying in bed, head on the pillow, under a blanket.
// The server snaps a sleeper onto the bed's tile, so the bed is at (sx, sy).
func (g *Game) drawSleeper(dst *ebiten.Image, p protocol.PlayerView, d drawable) {
	tint := playerColors[int(p.ID)%len(playerColors)]
	// Turned a quarter left: the feet anchor goes to the foot of the bed and
	// the head to the pillow. Breathing lifts the blanket a pixel.
	fx, fy := d.sx+10, d.sy-6
	b := g.art.body(p.Look)
	drawAt(dst, b.back[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, nil)
	drawAt(dst, b.legs[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, nil)
	drawAt(dst, b.shirt[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, tint)
	drawAt(dst, b.top[sim.DirDown][0], fx, fy, 1, 1, -math.Pi/2, nil)
	breath := 0.0
	if (g.frame/50)%2 == 0 {
		breath = 1
	}
	drawAt(dst, g.art.blanket, d.sx+5, d.sy-1-breath, 1, 1, 0, nil)
}

// drawTool draws the tool s swinging in p's hand.

// drawClouds drifts soft cloud shadows over the land by day.
func (g *Game) drawClouds(dst *ebiten.Image, cam camera) {
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
// pure function of time, so there is no particle state to keep.
func (g *Game) drawEmbers(dst *ebiten.Image, cam camera) {
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
			vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 1, 1,
				color.RGBA{uint8(int(col.R) * int(a) / 255), uint8(int(col.G) * int(a) / 255), uint8(int(col.B) * int(a) / 255), a}, false)
		}
	}
}

// target returns the entity or animal the Action key would work on right
// now, using the same rule as the server.
func (g *Game) target() targetView {
	if g.curr.Self.Dead {
		return targetView{}
	}
	for _, p := range g.curr.Players {
		if p.ID != g.myID {
			continue
		}
		if p.Cast != sim.CastNone {
			return targetView{} // fishing: Space strikes, it works on nothing else
		}
		if _, _, ok := g.digSpot(); ok {
			return targetView{} // a held hoe tills the tile in front: marked by drawDigSpot
		}
		x, y := float64(p.X), float64(p.Y)
		if d, ok := g.dropTarget(x, y); ok {
			return targetView{ok: true, drop: true, d: d} // a pile comes first, as on the server
		}
		hasWood := g.curr.Self.Inv.HasFuel()
		t, ok := sim.PickTarget(x, y, hasWood, g.candidateAt, creatureCandidates(g.curr.Creatures, x, y), sim.ReachBonusFor(p.Held)+sim.RideReach(p.Mount != 0))
		if !ok {
			return targetView{}
		}
		if t.Creature != 0 {
			for _, c := range g.curr.Creatures {
				if c.ID == t.Creature && !g.hiddenInCave(float64(c.X), float64(c.Y)) {
					return targetView{ok: true, creature: true, c: c}
				}
			}
		}
		for _, e := range g.curr.Entities {
			if e.ID == t.Entity && !g.hiddenInCave(float64(e.TX)+0.5, float64(e.TY)+0.5) {
				return targetView{ok: true, e: e}
			}
		}
	}
	return targetView{}
}

func (g *Game) candidateAt(tx, ty int) (sim.TargetCandidate, bool) {
	i, ok := g.entAt[[2]int{tx, ty}]
	if !ok {
		return sim.TargetCandidate{}, false
	}
	e := g.curr.Entities[i]
	return sim.TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready}, true
}

// drawWorldLabels draws, at full screen resolution, what must stay readable
// at night: the target marker and other players' names.
func (g *Game) drawWorldLabels(screen *ebiten.Image, cam camera, scale float64, tg targetView) {
	alpha := g.alpha()
	if tg.ok {
		var sx, sy, top float64
		var label string
		col := colWhite
		if tg.drop {
			x, y, lift := g.dropPos(tg.d)
			sx, sy = cam.toScreen(x, y)
			top = sy - lift - 14
			label = "Nhặt " + tg.d.Item.Def().Name
			if tg.d.Count > 1 {
				label += fmt.Sprintf(" ×%d", tg.d.Count)
			}
		} else if tg.creature {
			x, y, moving := g.lerpCreature(tg.c, alpha)
			sx, sy = cam.toScreen(x, y)
			s, bob := g.creatureSprite(tg.c, moving)
			top = sy - bob - s.ay
			label, col = g.creatureLabel(tg.c)
		} else {
			e := tg.e
			sx, sy = cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
			s, _ := g.entitySprite(e)
			top = sy - s.ay
			label = e.Kind.Name()
			if e.Kind.Sized() {
				label += [2]string{" nhỏ", " lớn"}[sizeIdx(e.Big)]
			}
			switch {
			case e.Kind == sim.EntityCampfire:
				label += " — thêm củi"
				if g.hasRawFood() {
					label += " · F nướng"
				}
			case e.Kind == sim.EntityTree && !g.has(sim.ToolAxe):
				label, col = label+" — cần rìu (Q)", colBad
			case e.Kind == sim.EntityWall:
				if !g.has(sim.ToolAxe) {
					label, col = label+" — cần rìu để dỡ", colBad
				} else {
					label += " — dỡ"
				}
			case e.Kind == sim.EntityChest:
				label += " — mở"
			case e.Kind == sim.EntityStoneWall:
				if !g.has(sim.ToolPickaxe) {
					label, col = label+" — cần cúp để dỡ", colBad
				} else {
					label += " — dỡ"
				}
			case e.Kind == sim.EntityBed:
				if sim.PhaseOf(g.curr.Tick) == sim.PhaseDay {
					label += " — ngủ khi trời tối"
				} else {
					label += " — ngủ"
				}
			case (e.Kind == sim.EntityBoulder || e.Kind == sim.EntityCoal) && !g.has(sim.ToolPickaxe):
				label, col = label+" — cần cúp đá", colBad
			case e.Kind == sim.EntityCactus && !g.has(sim.ToolAxe):
				label, col = label+" — cần rìu", colBad
			case e.Kind == sim.EntityFurnace:
				label += " — thêm than, củi"
				if e.Fuel > 0 {
					label += " · đang cháy: nung trong túi (B)"
				}
			case !g.canMine(e.Kind):
				need, _ := orePick(e.Kind)
				label, col = label+" — cần "+sim.ToolItem(sim.ToolPickaxe, need).Def().Name, colBad
			case e.Kind == sim.EntityFarmland:
				if g.curr.Self.Inv.Count(sim.ItemPotato) > 0 {
					label += " — trồng khoai"
				} else {
					label, col = label+" — cần khoai lang để trồng", colBad
				}
			case e.Kind == sim.EntityCrop:
				label += " — đào khoai"
			case sim.IsFruitTree(e.Kind):
				label += " — hái quả"
			default:
				label, col = g.tribalEntityLabel(e, label, col)
			}
		}
		topArt := top - 9 + math.Round(1.5*math.Sin(float64(g.frame)*0.12))
		mb := g.art.marker.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(mb.Dx())/2, 0)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(sx*scale, topArt*scale)
		screen.DrawImage(g.art.marker, op)
		g.text(screen, label, sx*scale, topArt*scale-18, col, text.AlignCenter, 1)
	}
	for _, p := range g.curr.Players {
		wx, wy, _ := g.lerpPlayer(p, alpha)
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreen(wx, wy)
		if p.Asleep {
			// Two z's drifting up from the pillow.
			for i := range 2 {
				ph := math.Mod(float64(g.frame)/90+float64(i)*0.5, 1)
				zx, zy := (sx-4+ph*6)*scale, (sy-14-ph*12)*scale
				g.text(screen, "z", zx, zy, color.RGBA{220, 226, 255, uint8(255 * (1 - ph))}, text.AlignCenter, 1+float64(i))
			}
		}
		if p.ID == g.myID {
			continue
		}
		g.text(screen, p.Name, sx*scale, (sy-30)*scale-14, colWhite, text.AlignCenter, 1)
	}
}
