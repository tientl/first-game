package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The fights at the lairs of StyleLairs (sim/lairsite.go, temple.go,
// atoll.go) on screen: the temple's paved courtyard and black pool (baked
// in templeart.go), the wreck's deck on the reef, what the fights leave on
// the ground (venom, the flood, broken reef, the splash, the shark's blood),
// the birds wheeling over the landmarks, the bosses' poses through each of
// their moves and the effects as they land, the spears in the air and the
// line a drawn spear will fly. The art is in templeart.go and atollart.go.

// lairFight is the client's own state of those fights.
type lairFight struct {
	pave    *templePaving
	phases  map[sim.CreatureID]phaseSeen
	puddles map[sim.CreatureID][]float64 // where a spray's puddles were shown, x, y pairs
	globs   []venomGlob
	reefs   map[[2]int32]bool // broken footholds seen, for their splash
	splash  map[[2]int32]bool
	pull    float64 // the camera drawn back for a fight: 1 none, less is further back
	rate    float64 // the battle music's speed
	drawPull float64       // the pull this frame is drawn at
	up       *ebiten.Image // the world drawn back, shrunk to the screen
}

type phaseSeen struct {
	skill sim.SkillID
	phase sim.SkillPhase
}

// venomGlob is a gob of venom in the air, from the snake's jaws to where
// it lands.
type venomGlob struct {
	x0, y0, x1, y1 float64
	at             int
}

// --- the frame's update -----------------------------------------------------------

// lairFightUpdate moves the fights' scenes on a frame: the effects as the
// bosses' moves land, the gobs of venom, the camera drawn back.
func (g *Game) lairFightUpdate() {
	f := &g.lf
	if f.phases == nil {
		f.phases = map[sim.CreatureID]phaseSeen{}
		f.puddles = map[sim.CreatureID][]float64{}
		f.reefs = map[[2]int32]bool{}
		f.splash = map[[2]int32]bool{}
		f.pull, f.rate = 1, 1
	}
	for _, c := range g.curr.Creatures {
		if c.Kind != sim.CreatureTitanoboa && c.Kind != sim.CreatureMegalodon {
			continue
		}
		was := f.phases[c.ID]
		now := phaseSeen{c.Skill, c.Phase}
		if now.skill != sim.SkillNone && now.phase == sim.PhaseWindup {
			var pts []float64
			for _, z := range g.curr.Zones {
				if z.Creature == c.ID {
					pts = append(pts, float64(z.X), float64(z.Y))
				}
			}
			if len(pts) > 0 {
				f.puddles[c.ID] = pts
			}
		}
		if now != was && now.skill != sim.SkillNone {
			g.bossMoveFX(c, was, now)
		}
		f.phases[c.ID] = now
	}
	// What the ground shows breaking: a splash as a foothold goes.
	seen := map[[2]int32]bool{}
	for _, h := range g.curr.Hazards {
		key := [2]int32{int32(math.Floor(float64(h.X))), int32(math.Floor(float64(h.Y)))}
		switch h.Kind {
		case sim.HazardReef:
			seen[key] = true
			if !f.reefs[key] {
				g.spawnFX(fxInst{kind: fxSplash, x: float64(h.X), y: float64(h.Y), k: 1.1})
				g.spawnFX(fxInst{kind: fxDust, x: float64(h.X), y: float64(h.Y), k: 0.8, pal: 1})
			}
		case sim.HazardSplash:
			if !f.splash[key] {
				f.splash[key] = true
				g.spawnFX(fxInst{kind: fxSplash, x: float64(h.X), y: float64(h.Y), k: 2.6})
				addTrauma(0.3)
			}
		}
	}
	f.reefs = seen
	if len(f.splash) > 32 {
		clear(f.splash)
	}
	keep := f.globs[:0]
	for _, gl := range f.globs {
		if g.frame-gl.at < 26 {
			keep = append(keep, gl)
		}
	}
	f.globs = keep
	// The camera draws back a little in a fight, so the boss and the
	// player both fit; and the music is driven harder as the boss weakens.
	want, rate := 1.0, 1.0
	if b := g.curr.Boss; b.Kind.Boss() && !g.curr.Self.Dead {
		want = 0.8
		if l, ok := g.lairOf(b.Kind); ok && l.Layout != sim.LayoutHollow {
			switch {
			case b.Kind == sim.CreatureTitanoboa && b.Stage >= 2:
				rate = 1.1
			case b.Kind == sim.CreatureMegalodon && b.Stage >= 1:
				rate = 1.07
			case b.Enraged:
				rate = 1.04
			}
		} else if b.Enraged {
			rate = 1.05
		}
	}
	f.pull += (want - f.pull) * 0.025
	if math.Abs(f.pull-want) < 0.001 {
		f.pull = want
	}
	f.rate += (rate - f.rate) * 0.02
}

// bossMoveFX is what boss c's move shows as it passes from was to now: the
// slam's dust, the dive's splash, the lunge's spray, the venom spat.
func (g *Game) bossMoveFX(c protocol.CreatureView, was, now phaseSeen) {
	x, y := float64(c.X), float64(c.Y)
	switch {
	case now.phase == sim.PhaseActive && now.skill == sim.TitanSkills()[5]: // the slam comes down
		g.spawnFX(fxInst{kind: fxStomp, x: x, y: y, k: 3.2})
		g.spawnFX(fxInst{kind: fxDust, x: x, y: y + 0.3, k: 2.4})
		addTrauma(0.55)
		if g.lairFlooded(sim.CreatureTitanoboa) {
			g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 2.2})
		}
	case now.phase == sim.PhaseActive && now.skill == sim.TitanSkills()[4]: // up out of the pool
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 2.8})
		g.spawnFX(fxInst{kind: fxSplash, x: x + 0.8, y: y - 0.2, k: 1.6})
		addTrauma(0.4)
	case now.phase == sim.PhaseWindup && now.skill == sim.TitanSkills()[4]: // into the pool
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 1.4})
	case now.phase == sim.PhaseActive && now.skill == sim.TitanSkills()[1]: // the tail swept round
		g.spawnFX(fxInst{kind: fxDust, x: x - 1, y: y + 0.4, k: 2, ang: math.Pi})
		addTrauma(0.25)
	case now.phase == sim.PhaseActive && now.skill == sim.TitanSkills()[3]: // venom spat
		pts := g.lf.puddles[c.ID]
		hx := x + 2.2
		if c.FaceLeft {
			hx = x - 2.2
		}
		for i := 0; i+1 < len(pts); i += 2 {
			g.lf.globs = append(g.lf.globs, venomGlob{hx, y - 0.4, pts[i], pts[i+1], g.frame - i})
		}
	case now.phase == sim.PhaseActive && now.skill == sim.SharkSkills()[0]: // out over the edge
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 2})
		addTrauma(0.35)
	case now.phase == sim.PhaseRecover && now.skill == sim.SharkSkills()[0]:
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 1.2})
	case now.phase == sim.PhaseActive && (now.skill == sim.SharkSkills()[1] || now.skill == sim.SharkSkills()[2]): // the ram
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 2.2})
		addTrauma(0.4)
	case now.phase == sim.PhaseActive && (now.skill == sim.SharkSkills()[3] || now.skill == sim.SharkSkills()[4]):
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 1.8})
	case now.phase == sim.PhaseActive && now.skill == sim.SharkSkills()[5]: // the leap
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 2})
	case now.phase == sim.PhaseRecover && now.skill == sim.SharkSkills()[5]: // and down
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 3.2})
		addTrauma(0.5)
	}
}

// lairFlooded reports whether the lair of k is under water now.
func (g *Game) lairFlooded(k sim.CreatureKind) bool {
	l, ok := g.lairOf(k)
	return ok && l.Flooded
}

// --- the bosses' poses through their moves --------------------------------------

// phaseU is how far c is through the phase of its move, 0 to 1.
func phaseU(c protocol.CreatureView) float64 {
	if c.PhaseLen == 0 {
		return 1
	}
	return clamp01(float64(c.PhaseT) / float64(c.PhaseLen))
}

// bossSkillSprite is the frame of Titanoboa or the Megalodon making one of
// their moves of StyleLairs; ok is false for anything else.
func (g *Game) bossSkillSprite(c protocol.CreatureView, t int, left bool) (sprite, bool) {
	if c.Skill == sim.SkillNone {
		return sprite{}, false
	}
	u := phaseU(c)
	switch c.Kind {
	case sim.CreatureTitanoboa:
		hdWarm(titanSweep, titanSlam)
		ts := sim.TitanSkills()
		switch c.Skill {
		case ts[0], ts[3]: // the strike, the venom: drawn back, then the lunge
			age := 0.0
			switch c.Phase {
			case sim.PhaseWindup:
				age = 30 * smooth(u)
			case sim.PhaseActive:
				age = 31 + 6*u
			default:
				age = 40 + 14*u
			}
			return titanStrike.sprite(titanStrikeFrame(int(age)), left), true
		case ts[1]: // the tail
			i := 0
			switch c.Phase {
			case sim.PhaseWindup:
				i = int(u * 2.99)
			case sim.PhaseActive:
				i = 3 + int(u*4.99)
			default:
				i = 8 + int(u*1.99)
			}
			return titanSweep.sprite(i, left), true
		case ts[2]: // the coils
			switch {
			case c.Phase == sim.PhaseWindup:
				return titanRoar.sprite(t/6, left), true
			case c.Phase == sim.PhaseActive:
				return titanGrab.sprite(t/6, left), true
			}
			return titanSlither.sprite(t/7, left), true
		case ts[4]: // up out of the pool, reared
			return titanRoar.sprite(t/5, left), true
		case ts[5]: // the slam
			switch c.Phase {
			case sim.PhaseWindup:
				return titanRoar.sprite(t/5, left), true
			case sim.PhaseActive:
				return titanSlam.sprite(int(u*3.99), left), true
			}
			return titanSlam.sprite(4+int(u*3.99), left), true
		}
	case sim.CreatureMegalodon:
		hdWarm(megaBeach)
		ss := sim.SharkSkills()
		switch c.Skill {
		case ss[0]: // the lunge out over the edge, and beached
			if c.Phase == sim.PhaseActive {
				return megaBeach.sprite(int(u*3.99), left), true
			}
			if c.Phase == sim.PhaseRecover {
				return megaBeach.sprite(4+(t/5)%6, left), true
			}
		case ss[5]: // the breach
			switch c.Phase {
			case sim.PhaseActive:
				return megaBreach.sprite(megaFrameAt(megaBreachAges[:], int(u*68)), left), true
			case sim.PhaseRecover:
				return megaSlam.sprite(int(u*float64(megaSlamN-1)), left), true
			}
		default: // the ram, the overturn, the bite: up out of the water and the snap
			switch c.Phase {
			case sim.PhaseActive:
				return megaStrike.sprite(megaFrameAt(megaStrikeAges[:], 31+int(u*5)), left), true
			case sim.PhaseRecover:
				return megaStrike.sprite(megaFrameAt(megaStrikeAges[:], 37+int(u*15)), left), true
			}
		}
	}
	return sprite{}, false
}

// drawTitanUnder draws Titanoboa gone down into its pool to come up
// elsewhere: first its coils sliding under, then only a dark shape gliding
// below the black water, a wake of rings and bubbles over it.
func (g *Game) drawTitanUnder(dst drawTarget, cam camera, c protocol.CreatureView, alpha float64) {
	x, y, _ := g.lerpCreature(c, alpha)
	sx, sy := cam.toScreenFine(x, y)
	u := phaseU(c)
	if c.Phase == sim.PhaseWindup && u < 0.4 { // sliding under
		s := titanSlither.sprite(g.frame/4, c.FaceLeft)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-s.ax, sy-s.ay+u*14)
		v := float32(1 - u/0.4)
		op.ColorScale.Scale(0.5+0.5*v, 0.55+0.45*v, 0.6+0.4*v, v)
		drawSprite(dst, s, op)
	}
	// The shape under the water.
	for i := range 9 {
		k := float64(i) / 8
		bx := sx - float64(1-2*b2i(c.FaceLeft))*k*90
		by := sy + math.Sin(float64(g.frame)*0.08-k*5)*6
		r := float32(14 - 8*k)
		fillCircle(dst, float32(bx), float32(by), r*1.4, color.RGBA{0, 0, 0, 70}, true)
	}
	for k := range 3 { // rings where it goes
		ph := math.Mod(float64(g.frame)/40+float64(k)/3, 1)
		v := uint8(90 * (1 - ph))
		strokeCircleOval(dst, sx, sy, 8+ph*40, 0.42, color.RGBA{v, v, v, v})
	}
	for i := range 6 { // bubbles breaking
		ph := math.Mod(float64(g.frame)/30+float64(i)*0.37, 1)
		bx := sx + (hashf(i, g.frame/30, 1801)-0.5)*60
		by := sy + (hashf(i, g.frame/30, 1802)-0.5)*22 - ph*4
		fillCircle(dst, float32(bx), float32(by), float32(1+ph*1.5), color.RGBA{180, 200, 190, uint8(160 * (1 - ph))}, true)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- on the ground: the temple's floor, the wreck's deck, the fights' marks -------

// drawLairFloor draws what lies flat at the lairs of StyleLairs, before
// anything that stands: the paving and the pool, the wreck's deck, the
// flood, the venom, the broken reef, the blood in the lagoon.
func (g *Game) drawLairFloor(dst drawTarget, cam camera) {
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	if t, ok := g.lairOf(sim.CreatureTitanoboa); ok && t.Layout == sim.LayoutTemple {
		g.drawPaving(dst, cam, templeOf(t.X, t.Y, t.Radius, t.PoolX, t.PoolY, t.PoolRX, t.PoolRY), w, h)
		g.drawPool(dst, cam, t)
	}
	for _, e := range g.curr.Entities {
		if e.Kind == sim.EntityLairProp && e.Var == sim.LairWreck {
			sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
			drawAt(dst, atollSpritesOf().deck, sx, sy, 1, 1, 0, nil)
		}
	}
	for _, hz := range g.curr.Hazards {
		switch hz.Kind {
		case sim.HazardFlood:
			g.drawFlood(dst, cam, hz, w, h)
		case sim.HazardVenom:
			g.drawVenom(dst, cam, hz)
		case sim.HazardReef:
			g.drawBrokenReef(dst, cam, hz)
		case sim.HazardBlood:
			g.drawBlood(dst, cam, hz)
		case sim.HazardSplash:
			g.drawSplashRings(dst, cam, hz)
		}
	}
}

// drawPaving draws the baked blocks of the courtyard in view, baking at
// most two a frame (the rest the frames after).
func (g *Game) drawPaving(dst drawTarget, cam camera, t templeLook, w, h float64) {
	f := &g.lf
	if f.pave == nil || f.pave.look != t {
		f.pave = &templePaving{look: t, blocks: map[paveKey]*ebiten.Image{}, used: map[paveKey]int{}}
	}
	p := f.pave
	const bw, bh = paveBlockTiles * TileW, paveBlockTiles * TileH
	R := t.r + 1
	bx0, by0 := int(math.Floor((t.x-R)/paveBlockTiles)), int(math.Floor((t.y-R)/paveBlockTiles))
	bx1, by1 := int(math.Floor((t.x+R)/paveBlockTiles)), int(math.Floor((t.y+R)/paveBlockTiles))
	baked := 0
	for by := by0; by <= by1; by++ {
		for bx := bx0; bx <= bx1; bx++ {
			sx, sy := cam.toScreen(float64(bx*paveBlockTiles), float64(by*paveBlockTiles))
			if sx > w || sy > h || sx+bw < 0 || sy+bh < 0 {
				continue
			}
			k := paveKey{bx, by}
			img, ok := p.blocks[k]
			if !ok {
				if baked >= 2 {
					continue
				}
				img = t.paveBake(k)
				p.blocks[k] = img
				baked++
			}
			p.used[k] = g.frame
			if img != nil {
				drawSprite(dst, spriteHD(img, 0, 0), func() *ebiten.DrawImageOptions {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(sx, sy)
					return op
				}())
			}
		}
	}
	for k, at := range p.used { // let go of blocks long out of view
		if g.frame-at > 600 {
			if img := p.blocks[k]; img != nil {
				img.Deallocate()
			}
			delete(p.blocks, k)
			delete(p.used, k)
		}
	}
}

// drawPool draws the life of the black pool over its baked water: slow
// rings, mist drifting low over it, rings where the snake lies in it.
func (g *Game) drawPool(dst drawTarget, cam camera, l protocol.LairView) {
	px, py := float64(l.PoolX), float64(l.PoolY)
	rx, ry := float64(l.PoolRX), float64(l.PoolRY)
	for i := range 4 { // drops from the dead tree's limbs, rings spreading
		ph := math.Mod(float64(g.frame)/200+float64(i)*0.29, 1)
		cyc := (g.frame + i*53) / 200
		a := hashf(i, cyc, 1811) * 2 * math.Pi
		r := math.Sqrt(hashf(i, cyc, 1812)) * 0.7
		sx, sy := cam.toScreen(px+math.Cos(a)*rx*r, py+math.Sin(a)*ry*r)
		v := uint8(70 * (1 - ph))
		strokeCircleOval(dst, sx, sy, 2+ph*16, 0.45, color.RGBA{v, v, v, v})
	}
	for i := range 7 { // mist, low and slow
		t := float64(g.frame)*0.0025 + float64(i)*0.9
		mx := px + math.Sin(t+float64(i))*rx*0.7
		my := py + math.Cos(t*0.7+float64(i)*1.7)*ry*0.6
		sx, sy := cam.toScreen(mx, my)
		a := uint8(16 + 10*math.Sin(t*2))
		fillCircle(dst, float32(sx), float32(sy-6), float32(26+8*math.Sin(t)), color.RGBA{a, a, a, a}, true)
	}
	if c, ok := g.bossOf(sim.CreatureTitanoboa); ok && c.State != sim.CreatureLurk {
		if dx, dy := (float64(c.X)-px)/rx, (float64(c.Y)-py)/ry; dx*dx+dy*dy < 1.2 {
			sx, sy := cam.toScreen(float64(c.X), float64(c.Y))
			for k := range 3 {
				ph := math.Mod(float64(g.frame)/90+float64(k)/3, 1)
				v := uint8(80 * (1 - ph))
				strokeCircleOval(dst, sx, sy, 30+ph*60, 0.4, color.RGBA{v, v, v, v})
			}
		}
	}
}

// drawFlood draws the courtyard under shallow water: dark and clear over
// the paving, rising from the pool, ripples running over it, rings round
// whoever wades it.
func (g *Game) drawFlood(dst drawTarget, cam camera, hz protocol.HazardView, w, h float64) {
	rise := clamp01(float64(hz.Age) / (sim.FloodRise))
	fall := clamp01(float64(hz.Left) / 40)
	cx, cy, R := float64(hz.X), float64(hz.Y), float64(hz.R)*rise
	if R < 0.5 {
		return
	}
	shade := func(x, y float64) float32 {
		d := math.Hypot(x-cx, y-cy)
		return float32(0.5 * fall * smooth(clamp01((R-d)/1.6)) * (0.85 + 0.15*vnoise(x*0.6, y*0.6+float64(g.frame)*0.004, 1821)))
	}
	var vs []ebiten.Vertex
	var is []uint16
	for ty := int(cy - R - 1); ty <= int(cy+R+1); ty++ {
		for tx := int(cx - R - 1); tx <= int(cx+R+1); tx++ {
			if len(vs) > 60000 {
				break
			}
			sx, sy := cam.toScreen(float64(tx), float64(ty))
			if sx > w || sy > h || sx+TileW < 0 || sy+TileH < 0 {
				continue
			}
			n := uint16(len(vs))
			for _, c := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				x, y := float64(tx+c[0]), float64(ty+c[1])
				a := shade(x, y)
				px, py := cam.toScreen(x, y)
				vs = append(vs, ebiten.Vertex{DstX: float32(px), DstY: float32(py), SrcX: 1, SrcY: 1,
					ColorR: 0.05 * a, ColorG: 0.16 * a, ColorB: 0.2 * a, ColorA: a})
			}
			is = append(is, n, n+1, n+2, n+1, n+3, n+2)
		}
	}
	if len(vs) > 0 {
		dst.DrawTriangles(vs, is, whitePixel(), nil)
	}
	// Ripples running across it, catching the light.
	for i := range 40 {
		a := hashf(i, 0, 1822) * 2 * math.Pi
		r := math.Sqrt(hashf(i, 1, 1822)) * (R - 0.8)
		x, y := cx+math.Cos(a)*r, cy+math.Sin(a)*r
		ph := math.Mod(float64(g.frame)/80+hashf(i, 2, 1822), 1)
		sx, sy := cam.toScreen(x, y)
		v := uint8(70 * math.Sin(ph*math.Pi) * fall)
		strokeLine(dst, float32(sx-4-ph*3), float32(sy), float32(sx+4+ph*3), float32(sy), 0.8, color.RGBA{v, v, v, v}, true)
	}
	for _, p := range g.curr.Players { // rings round who wades
		if math.Hypot(float64(p.X)-cx, float64(p.Y)-cy) < R {
			sx, sy := cam.toScreen(float64(p.X), float64(p.Y))
			ph := math.Mod(float64(g.frame)/50, 1)
			v := uint8(110 * (1 - ph))
			strokeCircleOval(dst, sx, sy, 5+ph*9, 0.4, color.RGBA{v, v, v, v})
		}
	}
}

// venomPuddle is a puddle of Titanoboa's venom: a sickly gloss, darker at
// its rim, a sheen to the light.
var venomPuddle = newBHAnim(3, func(i int) (*canvas, float64, float64) {
	c := hdBossCanvas(40, 26)
	seed := uint32(1830 + i*7)
	vr := mkRamp(0x3f4328, 0x5f7132, 0x94ad39, 0xc2d64f, 0xeff37c)
	c.ellipse(20, 13, 17, 10, func(x, y int, nx, ny float64) color.RGBA {
		lx, ly := c.lx(x), c.ly(y)
		edge := math.Hypot(nx, ny) + (vnoise(lx*0.25, ly*0.3, seed)-0.5)*0.55
		if edge > 1 {
			return color.RGBA{}
		}
		l := 0.6 - 0.45*smooth(clamp01((edge-0.65)/0.35)) + (vnoise(lx*0.5, ly*0.6, seed+1)-0.5)*0.2
		if nx < -0.1 && ny < -0.2 && nx > -0.55 && ny > -0.6 {
			l += 0.3 // the sheen
		}
		return withAlpha(rq(vr, l, x, y), 230)
	})
	return hdCrop(c, 20, 13)
})

// drawVenom draws a puddle of venom: splashed down, lying, drying away;
// bubbles rising in it.
func (g *Game) drawVenom(dst drawTarget, cam camera, hz protocol.HazardView) {
	sx, sy := cam.toScreen(float64(hz.X), float64(hz.Y))
	in := clamp01(float64(hz.Age) / 4)
	out := clamp01(float64(hz.Left) / 40)
	k := float64(hz.R) / 0.7 * (0.6 + 0.4*in)
	i := int(sim.HashTile(int(hz.X*7), int(hz.Y*7))) % 3
	v := uint8(255 * out)
	drawAt(dst, venomPuddle.sprite(i, i == 1), sx, sy, k, k, 0, color.RGBA{v, v, v, v})
	for j := range 4 {
		ph := math.Mod(float64(g.frame)/45+float64(j)*0.27+float64(i)*0.1, 1)
		bx := sx + (hashf(j, int(hz.X*10), 1831)-0.5)*20*k
		by := sy + (hashf(j, int(hz.Y*10), 1832)-0.5)*10*k
		a := uint8(200 * (1 - ph) * out)
		strokeCircle(dst, float32(bx), float32(by-ph*2), float32(0.6+ph*1.6), 0.5, color.RGBA{a, a, uint8(float64(a) * 0.6), a}, true)
	}
}

// brokenReef is a foothold of reef broken by the shark: the sea in the gap,
// the rubble of the coral in it, foam round its edges.
var brokenReef = newBHAnim(3, func(i int) (*canvas, float64, float64) {
	c := hdBossCanvas(TileW+10, TileH+10)
	seed := uint32(1840 + i*11)
	c.ellipse(5+TileW/2, 5+TileH/2, TileW/2+4, TileH/2+4, func(x, y int, nx, ny float64) color.RGBA {
		lx, ly := c.lx(x), c.ly(y)
		e := math.Hypot(nx, ny) + (vnoise(lx*0.3, ly*0.35, seed)-0.5)*0.5
		switch {
		case e > 1:
			return color.RGBA{}
		case e > 0.86:
			return withAlpha(colFoam, uint8(220*(1-(e-0.86)/0.14)))
		}
		l := 0.25 + 0.5*e + (vnoise(lx*0.5, ly*0.8, seed+1)-0.5)*0.2
		return rq(reefPool, l, x, y)
	})
	rng := int(seed)
	for j := range 6 { // rubble lying in the water
		x := 5 + TileW/2 + (hashf(j, rng, 1841)-0.5)*TileW*0.8
		y := 5 + TileH/2 + (hashf(j, rng, 1842)-0.5)*TileH*0.7
		r := 1.6 + hashf(j, rng, 1843)*2
		c.ellipse(x, y, r, r*0.75, func(px, py int, nx, ny float64) color.RGBA {
			return rq(reefFlatRock, 0.35+0.5*lum(nx, ny), px, py)
		})
	}
	c.outline(0.5)
	return hdCrop(c, 5+TileW/2, 5+TileH/2)
})

// drawBrokenReef draws a broken foothold: the gap opened in a burst, the
// water in it, and as it settles the coral rising back out of the water.
func (g *Game) drawBrokenReef(dst drawTarget, cam camera, hz protocol.HazardView) {
	sx, sy := cam.toScreen(float64(hz.X), float64(hz.Y))
	i := int(sim.HashTile(int(hz.X), int(hz.Y))) % 3
	settle := clamp01(1 - float64(hz.Left)/sim.ReefSettle)
	open := clamp01(float64(hz.Age) / 6)
	a := uint8(255 * open * (1 - settle))
	k := 0.7 + 0.3*open
	drawAt(dst, brokenReef.sprite(i, false), sx, sy, k, k, 0, color.RGBA{a, a, a, a})
	if settle > 0 { // rising back: stones and foam
		for j := range 5 {
			bx := sx + (hashf(j, int(hz.X), 1844)-0.5)*TileW*0.8
			by := sy + (hashf(j, int(hz.Y), 1845)-0.5)*TileH*0.7
			v := uint8(255 * settle)
			fillCircle(dst, float32(bx), float32(by), float32(1.5+2*settle), color.RGBA{uint8(0.7 * float64(v)), uint8(0.66 * float64(v)), uint8(0.6 * float64(v)), v}, true)
		}
	}
	// Foam boiling round it, each a little different.
	ph := math.Mod(float64(g.frame)/35+float64(i)*0.3, 1)
	v := uint8(140 * (1 - ph) * (1 - settle))
	strokeCircleOval(dst, sx, sy, 8+ph*8, 0.6, color.RGBA{v, v, v, v})
}

// drawBlood draws the shark's blood clouding its lagoon in its frenzy.
func (g *Game) drawBlood(dst drawTarget, cam camera, hz protocol.HazardView) {
	cx, cy, R := float64(hz.X), float64(hz.Y), float64(hz.R)
	grow := clamp01(float64(hz.Age) / 120)
	for i := range 14 {
		t := float64(g.frame)*0.002 + float64(i)*1.3
		a := hashf(i, 0, 1850)*2*math.Pi + t*0.3
		r := (0.2 + 0.7*hashf(i, 1, 1850)) * (R - 3)
		x, y := cx+math.Cos(a)*r, cy+math.Sin(a)*r*0.95
		if g.tileAt(int(math.Floor(x)), int(math.Floor(y))) != sim.TileWater {
			continue
		}
		sx, sy := cam.toScreen(x, y)
		al := uint8(38 * grow)
		fillCircle(dst, float32(sx), float32(sy), float32((30+20*hashf(i, 2, 1850))*grow), color.RGBA{al * 2 / 3, 0, 0, al}, true)
	}
}

// drawSplashRings draws the sea still heaving where the shark came down.
func (g *Game) drawSplashRings(dst drawTarget, cam camera, hz protocol.HazardView) {
	sx, sy := cam.toScreen(float64(hz.X), float64(hz.Y))
	f := clamp01(float64(hz.Age) / float64(sim.SplashTicks))
	for k := range 3 {
		ph := clamp01(f*1.3 - float64(k)*0.2)
		if ph <= 0 || ph >= 1 {
			continue
		}
		v := uint8(160 * (1 - ph))
		strokeCircleOval(dst, sx, sy, (0.4+ph)*float64(hz.R)*TileW, 0.42, color.RGBA{v, v, v, v})
	}
}

// --- the lair props of StyleLairs ---------------------------------------------------

// lairSpriteV2 is the sprite of a lair prop of StyleLairs; ok is false for
// the older ones.
func (g *Game) lairSpriteV2(e protocol.EntityView) (sprite, bool) {
	h := int(sim.HashTile(int(e.TX), int(e.TY)) / 16)
	switch e.Var {
	case sim.LairPillar:
		a := templeSpritesOf()
		return a.pillar[h%len(a.pillar)], true
	case sim.LairColumnM:
		return templeSpritesOf().column[h%2], true
	case sim.LairDeadTree:
		return templeSpritesOf().tree, true
	case sim.LairSerpent:
		a := templeSpritesOf()
		if l, ok := g.lairOf(sim.CreatureTitanoboa); ok && float64(e.TX)+0.5 > float64(l.PoolX) {
			return a.snake[1], true // on the east side: looking west, over the water
		}
		return a.snake[0], true
	case sim.LairCampRing:
		return templeSpritesOf().ring, true
	case sim.LairCampLog:
		return templeSpritesOf().log, true
	case sim.LairMast:
		return atollSpritesOf().mast, true
	case sim.LairPalm:
		s, _ := g.fruitTreeSprite(protocol.EntityView{Kind: sim.EntityPalm, TX: e.TX, TY: e.TY, Ready: true})
		return s, true
	}
	return sprite{}, false
}

// lairHidden reports whether lair prop v is drawn as part of another (a
// fallen column's ends, the dead tree's foot, the wreck's deck) or flat in
// drawLairFloor.
func lairHidden(v uint8) bool {
	switch v {
	case sim.LairColumnL, sim.LairColumnR, sim.LairTreeRoot, sim.LairWreck, sim.LairWreckPart:
		return true
	}
	return false
}

// --- in the sky: the landmarks' birds, the spears, a spear's line ----------------

// drawLandmarkBirds draws the birds wheeling over the lairs' landmarks:
// vultures over the dead tree, gulls over the mast, their shadows below.
func (g *Game) drawLandmarkBirds(dst drawTarget, cam camera) {
	a := lairArtOf()
	for _, l := range g.curr.Lairs {
		if l.Layout == sim.LayoutHollow || l.Defeated && l.Kind == sim.CreatureTitanoboa {
			continue
		}
		mx, my := float64(l.MarkX), float64(l.MarkY)
		if mx == 0 && my == 0 {
			continue
		}
		n, alt, r, tint := 5, 190.0, 3.4, color.RGBA{50, 44, 44, 255}
		if l.Layout == sim.LayoutAtoll {
			n, alt, r, tint = 7, 150, 2.8, color.RGBA{240, 240, 236, 255}
		}
		for i := range n {
			sp := 0.006 + 0.002*hashf(i, 0, 1860)
			ang := float64(g.frame)*sp*float64(1-2*(i%2)) + float64(i)*2*math.Pi/float64(n)
			rr := r * (0.7 + 0.6*hashf(i, 1, 1860))
			x, y := mx+math.Cos(ang)*rr*1.3, my+math.Sin(ang)*rr
			sx, sy := cam.toScreenFine(x, y)
			h := alt + 30*math.Sin(float64(g.frame)*0.01+float64(i))
			fillCircle(dst, float32(sx), float32(sy), 3, color.RGBA{0, 0, 0, 40}, true) // its shadow far below
			k := 1.5
			if l.Layout == sim.LayoutAtoll {
				k = 1.2
			}
			drawAt(dst, a.bird[(g.frame/6+i)%2], sx, sy-h, k, k, 0, tint)
		}
	}
}

// drawSpears draws the spears in the air: the shaft along its flight, its
// stone point, its shadow on the ground below.
func (g *Game) drawSpears(dst drawTarget, cam camera, alpha float64) {
	for _, s := range g.curr.Spears {
		x, y := float64(s.X)+float64(s.VX)*alpha, float64(s.Y)+float64(s.VY)*alpha
		sx, sy := cam.toScreenFine(x, y)
		dx, dy := float64(s.VX)*TileW, float64(s.VY)*TileH
		l := math.Hypot(dx, dy)
		if l < 1e-6 {
			continue
		}
		dx, dy = dx/l, dy/l
		const lift, ln = 12.0, 20.0
		fillCircle(dst, float32(sx), float32(sy), 2, color.RGBA{0, 0, 0, 60}, true)
		strokeLine(dst, float32(sx-dx*ln*0.6), float32(sy+1), float32(sx+dx*ln*0.4), float32(sy+1), 1.4, color.RGBA{0, 0, 0, 50}, true)
		tx, ty := sx+dx*ln*0.5, sy-lift+dy*ln*0.5
		bx, by := sx-dx*ln*0.5, sy-lift-dy*ln*0.5
		strokeLine(dst, float32(bx), float32(by), float32(tx), float32(ty), 1.5, color.RGBA{88, 60, 36, 255}, true)
		strokeLine(dst, float32(bx), float32(by-0.5), float32(tx), float32(ty-0.5), 0.6, color.RGBA{150, 112, 70, 255}, true)
		pt := color.RGBA{105, 111, 128, 255}
		if s.Item == sim.ItemIvorySpear || s.Item == sim.ItemSpinoSpear {
			pt = color.RGBA{228, 222, 196, 255}
		}
		strokeLine(dst, float32(tx), float32(ty), float32(tx+dx*5), float32(ty+dy*5), 2.2, pt, true)
		strokeLine(dst, float32(bx-dx*2), float32(by-dy*2), float32(bx), float32(by), 2, color.RGBA{180, 70, 60, 255}, true) // the fletching of hide
	}
}

// drawThrowLine draws where a drawn spear will fly: a dotted line from
// the player to the pointer, as far as a throw goes, and a ring there.
func (g *Game) drawThrowLine(dst drawTarget, cam camera, alpha float64) {
	if !g.drawing {
		return
	}
	me, ok := g.me()
	if !ok {
		return
	}
	ax, ay, ok := g.aimPoint()
	if !ok {
		return
	}
	px, py, _ := g.lerpPlayer(me, alpha)
	dx, dy := ax-px, ay-py
	d := math.Hypot(dx, dy)
	if d < 0.3 {
		return
	}
	reach := min(d, sim.ThrowRange)
	ux, uy := dx/d, dy/d
	for t := 0.6; t <= reach; t += 0.45 {
		sx, sy := cam.toScreenFine(px+ux*t, py+uy*t)
		lift := math.Sin(t/reach*math.Pi) * 10
		fillCircle(dst, float32(sx), float32(sy-lift-8), 1.1, color.RGBA{255, 236, 190, 200}, true)
	}
	ex, ey := cam.toScreenFine(px+ux*reach, py+uy*reach)
	strokeCircleOval(dst, ex, ey, 8, 0.5, color.RGBA{255, 220, 150, 210})
}

// drawVenomGlobs draws the gobs of venom in the air.
func (g *Game) drawVenomGlobs(dst drawTarget, cam camera) {
	for _, gl := range g.lf.globs {
		u := clamp01(float64(g.frame-gl.at) / 22)
		if u <= 0 || u >= 1 {
			continue
		}
		x, y := lerpf(gl.x0, gl.x1, u), lerpf(gl.y0, gl.y1, u)
		sx, sy := cam.toScreenFine(x, y)
		lift := math.Sin(u*math.Pi)*40 + (1-u)*30
		for k := range 4 {
			back := float64(k) * 0.04
			tx, ty := cam.toScreenFine(lerpf(gl.x0, gl.x1, u-back), lerpf(gl.y0, gl.y1, u-back))
			tl := math.Sin((u-back)*math.Pi)*40 + (1-u+back)*30
			fillCircle(dst, float32(tx), float32(ty-tl), float32(3-0.6*float64(k)), color.RGBA{148, 173, 57, uint8(200 - 40*k)}, true)
		}
		fillCircle(dst, float32(sx), float32(sy-lift), 3.4, color.RGBA{194, 214, 79, 255}, true)
		fillCircle(dst, float32(sx-1), float32(sy-lift-1), 1.2, color.RGBA{239, 243, 124, 255}, true)
	}
}

// drawSharkWake draws the wake of the shark charging in under the water
// through the wind-up of its lunge or its ram: a churned line behind its
// fin, so the run at the edge is seen coming.
func (g *Game) drawSharkWake(dst drawTarget, cam camera, alpha float64) {
	c, ok := g.bossOf(sim.CreatureMegalodon)
	if !ok || c.Phase != sim.PhaseWindup {
		return
	}
	ss := sim.SharkSkills()
	if c.Skill != ss[0] && c.Skill != ss[1] && c.Skill != ss[2] {
		return
	}
	x, y, _ := g.lerpCreature(c, alpha)
	var ang float64
	for _, z := range g.curr.Zones {
		if z.Creature == c.ID {
			ang = float64(z.Angle)
			if c.Skill != ss[0] {
				ang = math.Atan2(float64(z.Y)-y, float64(z.X)-x)
			}
		}
	}
	ux, uy := math.Cos(ang), math.Sin(ang)
	for i := range 26 {
		t := float64(i) * 0.25
		wx, wy := x-ux*t, y-uy*t
		sx, sy := cam.toScreenFine(wx, wy)
		spread := 2 + t*3
		v := uint8(200 * (1 - float64(i)/26))
		px, py := -uy*spread, ux*spread*0.6
		fillCircle(dst, float32(sx+px), float32(sy+py), 1.3, color.RGBA{v, v, v, v}, true)
		fillCircle(dst, float32(sx-px), float32(sy-py), 1.3, color.RGBA{v, v, v, v}, true)
	}
}

// throwDrawFrames is how long the left button is held with a spear in hand
// before it is drawn back to throw (a click is a thrust).
const throwDrawFrames = 18

// throwInput works the left button with a spear in hand: a click thrusts
// (action as given), holding it draws the spear back (draw: the player
// turns to the pointer, the line it will fly shows) and letting it go
// throws it. It gives Action and Draw to send.
func (g *Game) throwInput(action bool) (act, draw bool) {
	held := sim.ItemNone
	if s := g.curr.Self.Inv[g.curr.Self.Selected]; !s.Empty() {
		held = s.Item
	}
	left := g.leftButton()
	if !sim.Throwable(held) || g.build.on || g.curr.Self.Dead {
		g.throwHold, g.drawing = 0, false
		return action, false
	}
	if g.drawing && !left { // let go: away it flies
		g.drawing, g.throwHold = false, 0
		_ = g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientCommand, Command: sim.Command{Kind: sim.CmdThrow}})
		return false, false
	}
	if !left {
		g.throwHold = 0
		return action, false
	}
	if g.throwHold++; g.throwHold >= throwDrawFrames {
		g.drawing = true
		return false, true
	}
	return action, false
}

// aimPoint is where the pointer is in the world, for a drawn spear.
func (g *Game) aimPoint() (float64, float64, bool) { return g.pointerWorld() }

// lairQuiet is how much of the jungle's birdsong and insects is heard: none
// within 15 tiles of Titanoboa's temple while it lives, coming back by 25.
func (g *Game) lairQuiet() float64 {
	l, ok := g.lairOf(sim.CreatureTitanoboa)
	if !ok || l.Defeated || l.Layout == sim.LayoutHollow {
		return 1
	}
	me, ok := g.me()
	if !ok {
		return 1
	}
	d := math.Hypot(float64(me.X-l.X), float64(me.Y-l.Y))
	return smooth(clamp01((d - 15) / 10))
}

// bossBarMarks are where the boss bar of k is notched, as shares of its
// health: the stages of its fight at its lair of StyleLairs, else half.
func (g *Game) bossBarMarks(k sim.CreatureKind) []float64 {
	if l, ok := g.lairOf(k); ok && l.Layout != sim.LayoutHollow {
		switch k {
		case sim.CreatureTitanoboa:
			return []float64{sim.TitanStage2, sim.TitanStage3}
		case sim.CreatureMegalodon:
			return []float64{sim.SharkFrenzy}
		}
	}
	return []float64{0.5}
}

// QuietZoom takes the zoom set by SetZoom without the notice that tells
// of it: for the screenshot tool.
func (g *Game) QuietZoom() { g.zoomNoticed = viewZoom }
