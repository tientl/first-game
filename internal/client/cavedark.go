package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The dark of the natural caves (see cavebake.go and cavelights.go). Away
// from any light they are near black, a little blue, at every hour of the
// day: the player makes out their own shape and a tile or two round them,
// and needs a torch to see further.
//
// The night above is a dark veil that the lights cut holes in (lighting.go).
// The caves are lit with a light map instead: the dim ambient, then every
// light added on top of it with a soft edge. The lights are a held torch, a
// torch on its stand, a campfire, crystals, glowing mushrooms and plants,
// lava, and daylight down a shaft. The map is drawn at art-pixel resolution
// and the scene is multiplied by it. A torch's light is warm on what it
// falls on and a crystal's is green, and lights that overlap add up. A
// little of each light's glow is then added over the scene, as light
// caught in the air, and the eyes of the beasts lurking in the dark catch
// the light: drawCaveEyes.

// caveAmbient is the light of the natural caves away from any light, by
// level (the deep caves are darker still), before the cave brightness
// setting: near black, a little blue.
var caveAmbient = [sim.CaveLevels][3]float32{{0.068, 0.078, 0.125}, {0.052, 0.062, 0.102}}

// caveBrightness is how much each choice of the cave brightness setting
// (Settings.CaveLight: dark, dim, light) raises the dark.
var caveBrightness = [...]float32{1, 1.9, 2.9}

// The reach of the lights in the caves, in tiles. They are wider than the
// warmth the server gives (sim.TorchRadius and the like): light carries
// further than heat.
const (
	caveTorchReach = 6.5 // a torch in the hand
	caveStandReach = 5.5 // a torch on its stand
	caveOwnReach   = 2.0 // the player's own faint sight, with no light at all
)

// The colours of the caves' lights at full strength.
var (
	colTorchLight = [3]float32{1.3, 1.02, 0.7}
	colFireLight  = [3]float32{1.35, 0.95, 0.55}
	colDayLight   = [3]float32{0.78, 0.88, 1}
	colOwnLight   = [3]float32{0.2, 0.19, 0.2}
	colLavaLight  = [3]float32{1.15, 0.52, 0.2}
)

// caveLamp is one light in the caves this frame.
type caveLamp struct {
	sx, sy float64    // its centre, in screen art pixels
	r      float64    // its reach in tiles: r*TileW across, r*TileH up and down
	col    [3]float32 // its colour and strength at the centre
	haze   float32    // how much of it is added over the scene, as light in the air
}

// caveDarkState is what the caves' light keeps from frame to frame.
type caveDarkState struct {
	on    bool       // the caves' light is drawn this frame
	amb   [3]float32 // the light away from any lamp this frame
	lamps []caveLamp // the lights in view this frame

	lmap   *ebiten.Image // the light map: the view in art pixels, with a pixel round it
	lava   *ebiten.Image // the glow of the lava, one pixel per tile
	lavaPx []byte
}

// lampSize is the side of the lamp image.
const lampSize = 128

var lampImg *ebiten.Image

// lampFall is how a lamp's light falls off, from 1 at its centre to 0 at
// distance 1: softly, with no hard edge.
func lampFall(d float64) float64 {
	if d >= 1 {
		return 0
	}
	t := 1 - d*d
	return t * t
}

// lampImage is a lamp's light, white, falling off as lampFall.
func lampImage() *ebiten.Image {
	if lampImg != nil {
		return lampImg
	}
	px := make([]byte, 4*lampSize*lampSize)
	h := float64(lampSize) / 2
	for y := range lampSize {
		for x := range lampSize {
			v := uint8(math.Round(255 * lampFall(math.Hypot(float64(x)+0.5-h, float64(y)+0.5-h)/h)))
			i := 4 * (y*lampSize + x)
			px[i], px[i+1], px[i+2], px[i+3] = v, v, v, v
		}
	}
	lampImg = ebiten.NewImage(lampSize, lampSize)
	lampImg.WritePixels(px)
	return lampImg
}

// lampOp places lamp l, its reach scaled by k, shifted by (dx, dy).
func lampOp(l caveLamp, k, dx, dy float64) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	h := float64(lampSize) / 2
	op.GeoM.Translate(-h, -h)
	op.GeoM.Scale(l.r*k*TileW/h, l.r*k*TileH/h)
	op.GeoM.Translate(l.sx+dx, l.sy+dy)
	return op
}

// blendMultiply multiplies what is drawn on by the colour drawn.
var blendMultiply = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorZero,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorSourceColor,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

// caveDarkOn reports whether the caves' own light is drawn: whenever this
// player is in the natural caves.
func (g *Game) caveDarkOn() bool {
	_, ok := g.caveLevel()
	return ok
}

// caveLightSetting is the cave brightness setting, within its choices.
func caveLightSetting() int {
	return max(0, min(len(caveBrightness)-1, settings.CaveLight))
}

// drawCaveDark lights the natural caves: see the top of this file.
func (g *Game) drawCaveDark(dst drawTarget, cam camera) {
	cd := &g.caveDark
	cd.on = true
	level, _ := g.caveLevel()
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()

	// Coming down, the dark closes in as the cave fades in.
	s := float32(smooth(g.caveIn()))
	k := caveBrightness[caveLightSetting()]
	for i, a := range caveAmbient[level] {
		cd.amb[i] = 1 + (min(1, a*k)-1)*s
	}
	cd.lamps = g.caveLamps(cam, w, h, cd.lamps[:0])

	lm := sizedImage(&cd.lmap, w+2, h+2)
	lm.Fill(color.RGBA{unit8(cd.amb[0]), unit8(cd.amb[1]), unit8(cd.amb[2]), 255})
	img := lampImage()
	for _, l := range cd.lamps {
		op := lampOp(l, 1, 1, 1)
		op.ColorScale.Scale(l.col[0], l.col[1], l.col[2], 1)
		op.Blend = ebiten.BlendLighter
		lm.DrawImage(img, op)
	}
	lx0, ly0, lava := g.tileMask(&g.lavaMask, cam, w, h, isLava)
	if lava {
		g.caveLavaLight(lm, cam, lx0, ly0)
	}
	multiplyOver(dst, lm)

	// Light caught in the air round the flames, the crystals, the shafts.
	for _, l := range cd.lamps {
		if l.haze <= 0 {
			continue
		}
		op := lampOp(l, 0.45, 0, 0)
		a := l.haze * s
		op.ColorScale.Scale(l.col[0]*a, l.col[1]*a, l.col[2]*a, a)
		op.Blend = ebiten.BlendLighter
		dst.DrawImage(img, op)
	}
	if lava {
		g.lavaLight(dst, cam, lx0, ly0, false, 0.07*s)
	}
	g.drawCaveEyes(dst, cam, s)
}

// unit8 is v, 0 to 1, as a byte.
func unit8(v float32) uint8 { return uint8(math.Round(float64(max(0, min(1, v))) * 255)) }

// multiplyOver multiplies the scene on dst by the light map lm, which
// covers dst's view in art pixels with a pixel to spare all round. On the
// world pass it goes straight onto the canvas, smoothly scaled up.
func multiplyOver(dst drawTarget, lm *ebiten.Image) {
	op := &ebiten.DrawImageOptions{Blend: blendMultiply, Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-1, -1)
	if wd, ok := dst.(*worldDst); ok {
		wd.flush()
		op.GeoM.Scale(wd.r, wd.r)
		wd.canvas.DrawImage(lm, op)
		return
	}
	dst.DrawImage(lm, op)
}

// caveLamps lists the lights in view, w by h art pixels.
func (g *Game) caveLamps(cam camera, w, h int, out []caveLamp) []caveLamp {
	add := func(sx, sy, r float64, k float32, col [3]float32, haze float32) {
		if sx+r*TileW < 0 || sx-r*TileW > float64(w) || sy+r*TileH < 0 || sy-r*TileH > float64(h) {
			return // out of view
		}
		out = append(out, caveLamp{sx, sy, r, [3]float32{col[0] * k, col[1] * k, col[2] * k}, haze})
	}
	at := func(tx, ty int16) (float64, float64) {
		return cam.toScreen(float64(tx)+0.5, float64(ty)+0.5)
	}
	day := float32(0.3 + 0.7*sim.Ambient(g.curr.Tick)) // by night, only the moon's
	for _, e := range g.curr.Entities {
		switch e.Kind {
		case sim.EntityCaveExit: // daylight down the steps of the way out: bright at their foot, soon gone
			sx, sy := at(e.TX, e.TY-1)
			add(sx, sy, 3, 0.9*day, colDayLight, 0.1) // the steps and the opening over them
			sx, sy = at(e.TX, e.TY+2)
			add(sx, sy, 2.8, 0.7*day, colDayLight, 0)
			add(sx, sy, 6.5, 0.2*day, colDayLight, 0)
			continue
		case sim.EntityCaveUp: // a little of it comes down the rope's shaft to the deep caves
			sx, sy := at(e.TX, e.TY+1)
			add(sx, sy, 2.2, 0.6*day, colDayLight, 0.08)
			add(sx, sy, 4.5, 0.14*day, colDayLight, 0)
			continue
		case sim.EntityGlowPlant:
			if !e.Ready {
				continue
			}
			sx, sy := at(e.TX, e.TY)
			pulse := 1 + 0.08*math.Sin(float64(g.frame)*0.03+float64(e.ID))
			col := [3]float32{0.25, 0.85, 1}
			if e.Var%2 != 0 {
				col = [3]float32{0.7, 0.35, 1}
			}
			add(sx, sy, sim.GlowRadius(e.Var)*pulse, 0.42, col, 0.12)
			continue
		}
		if r, col, ok := caveGlow(e); ok { // crystals and glowing mushrooms: see caveprops.go
			sx, sy := at(e.TX, e.TY)
			pulse := 1 + 0.07*math.Sin(float64(g.frame)*0.04+float64(e.ID%97))
			core := [3]float32{0.55 + 0.45*col[0], 0.55 + 0.45*col[1], 0.55 + 0.45*col[2]}
			add(sx, sy-4, 1.1, 0.8, core, 0)                         // the stone or the cap itself, aglow
			add(sx, sy, r*1.3*pulse, 0.46*float32(pulse), col, 0.12) // a dim pool round it, no further
			continue
		}
		if r, ok := sim.FireReach(e.Kind, int(e.Fuel)); ok { // campfires, torches on stands, furnaces
			sx, sy := at(e.TX, e.TY)
			f := g.flicker(uint32(e.ID))
			reach, col := 1.3*r+1.5, colFireLight
			if e.Kind == sim.EntityTorchStand {
				reach, col = caveStandReach, colTorchLight
				sy -= 8 // the flame is up on its pole
			}
			add(sx, sy, reach*f, float32(0.94+0.06*f), col, 0.1)
		}
	}
	for _, t := range g.torches(cam) {
		f := g.flicker(t.id)
		add(t.sx, t.sy+6, caveTorchReach*f, float32(0.94+0.06*f), colTorchLight, 0.1)
	}
	for _, p := range g.curr.Players {
		if p.ID != g.myID || p.Dead {
			continue
		}
		x, y, _ := g.lerpPlayer(p, g.alpha())
		sx, sy := cam.toScreenFine(x, y)
		add(sx, sy-6, caveOwnReach, 1, colOwnLight, 0)
	}
	return out
}

// caveLavaLight lets the lava in view light the caves: the lava itself as
// bright as it is drawn, and a strong orange glow a few tiles round it.
// (lx0, ly0) is the first tile of g.lavaMask (see tileMask).
func (g *Game) caveLavaLight(lm *ebiten.Image, cam camera, lx0, ly0 int) {
	op := maskOp(cam, lx0, ly0)
	op.GeoM.Translate(1, 1)
	op.Blend = ebiten.BlendLighter
	lm.DrawImage(g.lavaMask, op) // the lava glows: drawn as it is

	// The glow round it: each tile lit by the nearest lava, out to
	// lavaReach tiles, worked out a tile at a time and smoothed between
	// tiles as it is drawn.
	const lavaReach = 3.5
	const m = 4 // tiles searched round each tile: lavaReach and a little
	cd := &g.caveDark
	b := g.lavaMask.Bounds()
	nw, nh := b.Dx(), b.Dy()
	ex, ey := lx0-m, ly0-m
	gw, gh := nw+2*m, nh+2*m
	if len(cd.lavaPx) != 4*gw*gh {
		cd.lavaPx = make([]byte, 4*gw*gh)
	}
	px := cd.lavaPx
	clear(px)
	for y := range gh {
		for x := range gw {
			if !isLava(g.tileAt(ex+x, ey+y)) {
				continue
			}
			for dy := -m; dy <= m; dy++ {
				for dx := -m; dx <= m; dx++ {
					X, Y := x+dx, y+dy
					if X < 0 || Y < 0 || X >= gw || Y >= gh {
						continue
					}
					v := uint8(math.Round(255 * lampFall(math.Hypot(float64(dx), float64(dy))/lavaReach)))
					if i := 4 * (Y*gw + X); v > px[i] {
						px[i], px[i+1], px[i+2], px[i+3] = v, v, v, v
					}
				}
			}
		}
	}
	img := sizedImage(&cd.lava, gw, gh)
	img.WritePixels(px)
	op = maskOp(cam, ex, ey)
	op.GeoM.Translate(1, 1)
	c := colLavaLight
	op.ColorScale.Scale(c[0], c[1], c[2], 1)
	op.Blend = ebiten.BlendLighter
	lm.DrawImage(img, op)
}

// caveLightAt is roughly how bright the caves are at (sx, sy), in screen
// art pixels, this frame: the ambient and the lamps (not the lava), 0 to 1.
func (g *Game) caveLightAt(sx, sy float64) float64 {
	cd := &g.caveDark
	var c [3]float64
	for i := range c {
		c[i] = float64(cd.amb[i])
	}
	for _, l := range cd.lamps {
		f := lampFall(math.Hypot((sx-l.sx)/(l.r*TileW), (sy-l.sy)/(l.r*TileH)))
		if f <= 0 {
			continue
		}
		for i := range c {
			c[i] += f * float64(l.col[i])
		}
	}
	return min(1, 0.3*c[0]+0.55*c[1]+0.15*c[2])
}

// caveEyes is where a beast's eyes are, from its point on the ground
// facing right, in art pixels, and the colour they shine.
type caveEyes struct {
	eyes [][2]float64
	core color.RGBA // the bright point
	halo color.RGBA // the faint cross round it
}

var (
	spiderEyes = caveEyes{ // a cluster: two big eyes in front, two small above
		eyes: [][2]float64{{7, -6}, {5, -6}, {6, -8}},
		core: color.RGBA{0xd8, 0xff, 0xe8, 0xff}, halo: color.RGBA{0x40, 0xc8, 0x70, 0xff},
	}
	bearEyes = caveEyes{
		eyes: [][2]float64{{22, -16}, {19, -16}},
		core: color.RGBA{0xff, 0xe8, 0xb0, 0xff}, halo: color.RGBA{0xd8, 0x80, 0x28, 0xff},
	}
	batEyes = caveEyes{
		eyes: [][2]float64{{-1, -1}, {1, -1}},
		core: color.RGBA{0xff, 0x90, 0x80, 0xff}, halo: color.RGBA{0xa0, 0x20, 0x20, 0xff},
	}
)

// caveEyesDark is how dark it must be for eyes to catch the light: they
// shine fully below the first, not at all above the second.
const caveEyesFull, caveEyesNone = 0.16, 0.34

// caveEyeShine is how strongly eyes shine at a place this bright.
func caveEyeShine(light float64) float64 {
	return clamp01((caveEyesNone - light) / (caveEyesNone - caveEyesFull))
}

// drawCaveEyes lets the eyes of the beasts of the caves (the spiders, the
// bears) shine in the dark, so they can be seen coming without a torch.
// The night hunters have their own: see troodon.go.
func (g *Game) drawCaveEyes(dst drawTarget, cam camera, s float32) {
	alpha := g.alpha()
	for _, c := range g.curr.Creatures {
		var e *caveEyes
		switch c.Kind {
		case sim.CreatureSpider:
			e = &spiderEyes
		case sim.CreatureBear:
			e = &bearEyes
		default:
			continue
		}
		t := g.frame + int(c.ID)*53
		if t%(190+int(c.ID%7)*17) < 8 {
			continue // a blink
		}
		x, y, _ := g.lerpCreature(c, alpha)
		sx, sy := cam.toScreenFine(x, y)
		k := caveEyeShine(g.caveLightAt(sx, sy-8)) * float64(s)
		if c.State == sim.CreatureChase || c.State == sim.CreatureStrike || c.State == sim.CreatureClaw || c.State == sim.CreatureSpit {
			k *= 1.15 // hunting: wide awake
		} else {
			k *= 0.85 + 0.15*math.Sin(float64(t)*0.05)
		}
		if k <= 0.02 {
			continue
		}
		align(dst, sx, sy) // on the beast's own grid, as drawSprites drew it
		dir := 1.0
		if c.FaceLeft {
			dir = -1
		}
		if c.State == sim.CreatureHurt {
			sx += math.Round(math.Sin(float64(g.frame) * 1.3)) // as drawCreature flinches it
		}
		for i, p := range e.eyes {
			kk := k
			if i >= 2 {
				kk *= 0.6 // the small eyes
			}
			eyeGlint(dst, float32(sx+math.Floor(dir*p[0])), float32(sy+math.Floor(p[1])), kk, *e)
		}
	}
	alignCamera(dst)
}

// eyeGlint draws one shining eye at art pixel (x, y): a bright point in a
// faint cross of its colour.
func eyeGlint(dst drawTarget, x, y float32, k float64, e caveEyes) {
	px := func(dx, dy float32, c color.RGBA, a float64) {
		a = clamp01(a)
		if a <= 0 {
			return
		}
		fillRect(dst, x+dx, y+dy, 1, 1, color.RGBA{
			uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a),
		}, false)
	}
	for _, d := range [][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		px(d[0], d[1], e.halo, 0.45*k)
	}
	px(0, 0, e.core, 0.35+0.65*k)
}

// drawBatEyes lets the eyes of bat k flitting at (x, y) shine in the dark:
// two tiny red points, now and then turned away.
func (g *Game) drawBatEyes(dst drawTarget, x, y float64, k int) {
	if !g.caveDark.on || (g.frame/23+k*5)%7 == 0 {
		return
	}
	s := caveEyeShine(g.caveLightAt(x, y)) * smooth(g.caveIn())
	if s <= 0.02 {
		return
	}
	for _, p := range batEyes.eyes {
		c := batEyes.core
		a := clamp01(0.3 + 0.6*s)
		fillRect(dst, float32(math.Round(x+p[0])), float32(math.Round(y+p[1])), 1, 1, color.RGBA{
			uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a),
		}, false)
	}
}
