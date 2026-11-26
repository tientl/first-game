package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Sleeping. A sleeper is the jointed figure in a sleeping pose (moSleep),
// drawn standing, laid down (a row becomes a column, as bodyart.go lays the
// body) and covered from the shoulders with a fur that drapes over the shape
// under it: rounded over the body, up over the knees, down into the hollow
// of the waist, its lower edge falling past it over the side of the bed, a
// corner at the foot hanging, ragged with tufts, creased where it falls, the
// hair of the pelt lying along it. It fits the bed: on the back the knees
// are up, on the side it curls. It breathes, slowly, the fur rising with the
// chest, and turns over now and then in steps (moSleepTurn): on its back,
// rolling onto one side, curling up; back again; onto the other side. Going
// to bed it squats, sits on the edge and lies back, turned down onto the
// bed in four steps (moLieBack, drawn turned: see kp.spin), then pulls the
// fur up; waking (moWake) it sits up, stands, and stretches with a yawn.
// Long hair spreads over the pillow (hairfloat.go); armour is laid down by
// the bed.
//
// Each drawing is made the first time it is shown and kept.

// laidKey names one drawing of the figure drawn whole: laid down in bed, or
// turned part way (spun).
type laidKey struct {
	look   sim.Look
	wear   sim.Wear
	tint   color.RGBA
	m      motionID
	f      uint8
	dir    sim.Dir
	mirror bool
	lay    bool  // laid down: anchored so that put on a bed's anchor the head is on the pillow
	cover  uint8 // the fur over it: 0 none, 1 to the hips, 2 to the shoulders
	thin   bool  // a hide over the legs only, on the bare ground
}

var laidCache = map[laidKey]sprite{}

// The pillow's middle from the bed's anchor, art pixels: a sleeper's head
// goes there.
const (
	pillowX, pillowY = -8.0, -8.0
)

// sleepPose is the frame of moSleep for pose (0 on the back, 1 on the side
// facing out, 2 on the side facing in, 3 curled on the ground, 4 on the back
// stirring), the facing it is drawn in, and whether mirrored.
func sleepPose(pose uint8) (f int, d sim.Dir, mirror bool) {
	switch pose {
	case 1:
		return 2, sim.DirRight, true
	case 2:
		return 2, sim.DirRight, false
	case 3:
		return 4, sim.DirRight, true
	case 4:
		return 6, sim.DirDown, false
	}
	return 0, sim.DirDown, false
}

// laidSprite returns drawing k, making it the first time. A laid drawing is
// anchored at the pillow (see laidKey); a turned one at its hips.
func laidSprite(k laidKey) sprite {
	if s, ok := laidCache[k]; ok {
		return s
	}
	lo, up := k.m.keys(k.dir, int(k.f))
	c, drop := standingCanvas(k.look, k.wear, k.tint, up, lo, k.mirror)
	p := upperPoseOf(up)
	// The head's middle and the hips, on the canvas.
	var f figHD
	f.xf = bendOf(p).head(k.dir, k.look.Female)
	hx, hy := f.fwd(27.5, 35)
	hx += limbPad * HDDensity
	ix, iy := 27.0+limbPad*HDDensity, hipY+drop
	if p.spin == 0 {
		hy += drop
	}
	if k.mirror {
		hx, ix = float64(c.w-1)-hx, float64(c.w-1)-ix
	}
	var s sprite
	if k.lay {
		lay := layFlat(c)
		if k.cover > 0 {
			blanket(lay, shoulderY-2+drop, float64(k.cover)/2, k.thin, uint32(k.look.Hair)*13+uint32(k.f)+uint32(k.m)*7)
		}
		// Laid down, (x, y) went to (y, w-1-x).
		s = cropSprite(lay, hy/HDDensity-pillowX, (float64(c.w-1)-hx)/HDDensity-pillowY)
	} else {
		s = cropSprite(c, ix/HDDensity, iy/HDDensity)
	}
	laidCache[k] = s
	return s
}

// laidHips is where the hips of a sleeper laid on its back are from its
// bed's anchor: the turned drawings of lying down are put there.
func laidHips() (float64, float64) {
	return pillowX + (hipY-35)/HDDensity, pillowY
}

var blanketFur = mkRamp(0x3e2a1c, 0x5c412c, 0x7d5c3e, 0x9c7a55, 0xbb9a70, 0xd8bd92)

// blanket lays a fur over the laid-down figure on c from column x0 (the
// shoulders), as far as cover (0 to 1) of the way up from the feet. It
// drapes: rounded over the body (the body's own lie, smoothed along it),
// its lower edge falling past it over the side of the bed and a corner at
// the foot hanging further, the edges ragged with tufts; lit from above and
// behind, so the top of the roll is bright and what falls away is dark; the
// body's ridges (knees, hips) lit along their tops; creased where it falls;
// the pelt's hair lying along it, with darker markings; its top turned down
// in a hem. thin is a hide over the legs of a sleeper on the bare ground.
func blanket(c *canvas, x0, cover float64, thin bool, seed uint32) {
	W, H := c.w, c.h
	body := func(x, y int) bool { return x >= 0 && y >= 0 && x < W && y < H && c.px[y*W+x].A > 0 }
	top, bot := make([]float64, W), make([]float64, W)
	has := make([]bool, W)
	maxX := -1
	for x := 0; x < W; x++ {
		for y := 0; y < H; y++ {
			if body(x, y) {
				if !has[x] {
					top[x] = float64(y)
				}
				has[x], bot[x], maxX = true, float64(y), x
			}
		}
	}
	if maxX < 0 {
		return
	}
	end := float64(maxX) + 3
	from := math.Round(end - cover*(end-x0))
	if thin {
		from = math.Max(from, x0+22)
	}
	// The body's middle and half-height, smoothed along it.
	cy, hh := make([]float64, W), make([]float64, W)
	for x := 0; x < W; x++ {
		n, sc, sh := 0.0, 0.0, 0.0
		for d := -7; d <= 7; d++ {
			if xx := x + d; xx >= 0 && xx < W && has[xx] && float64(xx) >= from-4 {
				w := 1 - math.Abs(float64(d))/8
				sc += w * (top[xx] + bot[xx]) / 2
				sh += w * (bot[xx] - top[xx]) / 2
				n += w
			}
		}
		if n > 0 {
			cy[x], hh[x] = sc/n, sh/n+1.5
		}
	}
	type edge struct{ t, b float64 }
	edges := make([]edge, W)
	fur := func(x int, k uint32) float64 { // a ragged edge: tufts
		return 1.6*(vnoise(float64(x)/5, 0, seed+k)-0.5) + 0.9*(hashf(x/2, int(k), seed)-0.5)
	}
	for x := max(0, int(from)); x <= int(end) && x < W; x++ {
		if hh[x] == 0 {
			continue
		}
		droop := 2.5 + 2.5*vnoise(float64(x)/11, 1, seed)
		if thin {
			droop = 1.5
		}
		t := cy[x] - hh[x] - 0.5 + fur(x, 1)
		b := cy[x] + hh[x] + droop + fur(x, 2)
		// Rounded at the foot, its corner hanging over the side.
		if u := (float64(x) - (end - 9)) / 9; u > 0 {
			t += 6 * u * u
			b -= 2 * u * u
			if !thin {
				b += 8 * math.Max(0, 1-math.Abs(u-0.55)/0.4) // the corner
			}
		}
		edges[x] = edge{t, b}
	}
	// The hem runs across the body on a slant, pulled up further on the far
	// side, ragged.
	fi := max(0, min(W-1, int(from)+2))
	hemAt := func(y int) float64 {
		v := 0.0
		if hh[fi] > 0 {
			v = clamp01((float64(y) - cy[fi] + hh[fi]) / (2 * hh[fi]))
		}
		return from + 4*v + 1.2*(vnoise(float64(y)/4, 3, seed)-0.5)
	}
	inside := func(x, y int) bool {
		if float64(x) < from || x > int(end) || x < 0 || x >= W || y < 0 || y >= H || float64(x) < hemAt(y) {
			return false
		}
		e := edges[x]
		return e.b > e.t && float64(y) >= e.t && float64(y) <= e.b
	}
	// Creases from the body down to the edge, where the fur falls.
	type crease struct{ x0, y0, x1, y1 float64 }
	var creases []crease
	for i := 0; i < 5; i++ {
		x := from + 6 + (end-from-12)*(float64(i)+hashf(i, 3, seed))/5
		xi := int(x)
		if xi < 0 || xi >= W || hh[xi] == 0 {
			continue
		}
		creases = append(creases, crease{x, cy[xi] + hh[xi]*0.4, x + 3 + 4*hashf(i, 4, seed), edges[xi].b + 1})
	}
	out := make([]color.RGBA, len(c.px))
	copy(out, c.px)
	for y := 0; y < H; y++ {
		for x := max(0, int(from)); x <= int(end) && x < W; x++ {
			if !inside(x, y) {
				continue
			}
			e := edges[x]
			u := (float64(y) - cy[x]) / hh[x]
			var l float64
			if u <= 1 { // over the body: a roll, lit on top
				l = 0.62 - 0.3*u - 0.12*u*u
			} else { // falling away over the edge
				l = 0.3 - 0.12*(float64(y)-cy[x]-hh[x])/math.Max(1, e.b-cy[x]-hh[x])
			}
			// The body's own ridges under it: lit along their tops,
			// shaded under.
			if body(x, y) && !body(x, y-3) {
				l += 0.1
			}
			if body(x, y) && !body(x, y+3) {
				l -= 0.08
			}
			for _, cr := range creases {
				if float64(y) < cr.y0 {
					continue
				}
				ex, ey := cr.x1-cr.x0, cr.y1-cr.y0
				tt := clamp01(((float64(x)-cr.x0)*ex + (float64(y)-cr.y0)*ey) / (ex*ex + ey*ey))
				dx, dy := float64(x)-cr.x0-ex*tt, float64(y)-cr.y0-ey*tt
				switch dd := math.Hypot(dx, dy); {
				case dd < 1.1:
					l -= 0.2
				case dd < 2.4 && dx < 0:
					l += 0.09
				}
			}
			// The pelt: its hair lying along the body, darker markings.
			if hashf(x/4, y, seed+5) < 0.16 {
				l -= 0.09
			} else if hashf((x+1)/3, y, seed+6) < 0.08 {
				l += 0.08
			}
			if vnoise(float64(x)/13, float64(y)/8, seed+7) > 0.7 {
				l -= 0.1
			}
			// The hem, turned down at the top.
			switch hem := float64(x) - hemAt(y); {
			case hem < 3.5+fur(y, 9):
				l = 0.8 - 0.05*hem
				if hashf(x, y/2, seed+8) < 0.3 {
					l -= 0.14
				}
			case hem < 4.5+fur(y, 9):
				l = 0.24
			}
			col := toneOf(blanketFur, l)
			if !inside(x-1, y) || !inside(x+1, y) || !inside(x, y-1) || !inside(x, y+1) {
				col = mix(col, colInk, 0.42) // the edge
			}
			out[y*W+x] = col
		}
	}
	// Tufts standing off the top edge.
	for x := max(0, int(from)+5); x <= int(end) && x < W; x++ {
		e := edges[x]
		if e.b <= e.t || hashf(x, 11, seed) > 0.35 {
			continue
		}
		if y := int(math.Floor(e.t)) - 1; y >= 0 && y < H {
			out[y*W+x] = mix(toneOf(blanketFur, 0.7), colInk, 0.35)
		}
	}
	copy(c.px, out)
}

// standingCanvas draws frame (up, lo) of look l wearing w whole on one
// canvas, standing, in the player's colour tint, mirrored if asked; and how
// far its hips dropped. A frame turned whole (spin) is put together as it
// was drawn.
func standingCanvas(l sim.Look, w sim.Wear, tint color.RGBA, up upKey, lo loKey, mirror bool) (*canvas, float64) {
	uc := bakeUpper(l, w, up, headCache{})
	lc, lp := bakeLower(l, w, lo)
	c := newCanvas(limbW, limbH)
	drop := int(math.Round(lp.drop))
	udy := drop
	if lp.spin != 0 {
		udy = 0
	}
	put := func(src *canvas, tint1 bool, dy int) {
		for y := 0; y < src.h; y++ {
			for x := 0; x < src.w; x++ {
				p := src.px[y*src.w+x]
				if p.A < 128 || y+dy < 0 || y+dy >= c.h {
					continue
				}
				if tint1 {
					p = tinted(p, tint)
				}
				tx := x
				if mirror {
					tx = c.w - 1 - x
				}
				c.px[(y+dy)*c.w+tx] = p
			}
		}
	}
	put(uc.layer[upFar], true, udy)
	put(uc.layer[upBack], false, udy)
	put(lc, false, 0)
	put(uc.layer[upBody], false, udy)
	put(uc.layer[upShirt], true, udy)
	put(uc.layer[upArms], true, udy)
	put(uc.layer[upTop], false, udy)
	put(uc.layer[upArmsOver], true, udy)
	put(uc.layer[upTopOver], false, udy)
	return c, float64(drop)
}

// layFlat lays a standing figure down: a row becomes a column, the crown to
// the left and the figure's right side up.
func layFlat(c *canvas) *canvas {
	l := newCanvas(c.h, c.w)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			l.px[(c.w-1-x)*l.w+y] = c.px[y*c.w+x]
		}
	}
	return l
}

// cropSprite makes an HD sprite of what is drawn on c, anchored at (ax, ay)
// art pixels of c.
func cropSprite(c *canvas, ax, ay float64) sprite {
	x0, y0, x1, y1 := c.w, c.h, -1, -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A > 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return sprite{}
	}
	x0, y0 = x0/HDDensity*HDDensity, y0/HDDensity*HDDensity
	n := newCanvas(x1-x0+1, y1-y0+1)
	for y := 0; y < n.h; y++ {
		copy(n.px[y*n.w:(y+1)*n.w], c.px[(y+y0)*c.w+x0:(y+y0)*c.w+x0+n.w])
	}
	return spriteHD(n.image(), ax-float64(x0)/HDDensity, ay-float64(y0)/HDDensity)
}

// putSprite draws s with its anchor at (x, y).
func putSprite(dst drawTarget, s sprite, x, y float64) {
	if s.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-s.ax, y-s.ay)
	drawSprite(dst, s, op)
}

// Going to bed, in frames from lying down (asleepAt): squatting onto it,
// sat on its edge, lying back, then the fur pulled up.
const (
	bedSquat       = 14
	bedSit         = 26
	bedLie         = bedSit + 36 // four turned steps over 0.6 s
	bedFur         = bedLie + 18
	sleepTurnEvery = 540 // frames between turning over
	turnTime       = 36  // and how long a turn takes: two steps over 0.6 s
)

// drawSleeperHD draws p asleep at d, in their bed (the bed is drawn under
// them as an entity): going to bed if they have only just lain down, then
// asleep, breathing, turning over now and then.
func (g *Game) drawSleeperHD(dst drawTarget, p protocol.PlayerView, d drawable) {
	st := g.moveState(p.ID)
	age := float64(g.frame - st.asleepAt)
	if st.asleepAt == never {
		age = 1e9
	}
	look := p.Look.Sanitized()
	tint := playerColors[int(p.ID)%len(playerColors)]
	x, y := math.Round(d.sx), math.Round(d.sy)
	hx, hy := laidHips()
	b := g.art.bodyIn(p.Look, p.Wear)
	// The standing figure's feet, for its hips on the bed at (hx, hy).
	feet := func(m motionID, f int) (loKey, upKey, float64) {
		lo, up := m.keys(sim.DirDown, f)
		return lo, up, y + hy + hipHeight - motionPose(m, f, sim.DirDown).lo.drop/HDDensity
	}
	switch {
	case age < bedSquat: // down onto the bed
		lo, up, fy := feet(moSitDown, 1)
		drawFig(dst, figDraw{b: b, up: up, lo: lo, tint: tint}, x+hx, fy)
		return
	case age < bedSit: // sat on its edge
		lo, up, fy := feet(moSit, 0)
		drawFig(dst, figDraw{b: b, up: up, lo: lo, tint: tint}, x+hx, fy)
		return
	case age < bedLie: // lying back onto it, in four steps
		f := min(3, int((age-bedSit)/((bedLie-bedSit)/4)))
		putSprite(dst, laidSprite(laidKey{look: look, wear: p.Wear, tint: tint, m: moLieBack, f: uint8(f), dir: sim.DirDown}), x+hx, y+hy)
		return
	}
	k := laidKey{look: look, wear: p.Wear, tint: tint, m: moSleep, lay: true, cover: 2}
	switch {
	case age < bedLie+6:
		k.cover = 0
	case age < bedFur:
		k.cover = 1
	}
	// Asleep: a breath every three seconds or so; over on one side, back,
	// over on the other, every nine, each turn in two steps.
	t := age - bedLie + float64(p.ID)*211
	seg := int(t / sleepTurnEvery)
	seq := [4]uint8{0, 1, 0, 2}
	pose, prev := seq[seg%4], seq[(seg+3)%4]
	f, dir, mirror := sleepPose(pose)
	if math.Sin(t*2*math.Pi/190) > 0 && k.cover == 2 {
		f++
	}
	turning := false
	if within := t - float64(seg*sleepTurnEvery); seg > 0 && within < turnTime && k.cover == 2 && pose != prev {
		f, dir, mirror = turnStep(prev, pose, within < turnTime/2)
		k.m, turning = moSleepTurn, true
	}
	if k.cover < 2 {
		f, dir, mirror = 0, sim.DirDown, false
	}
	k.f, k.dir, k.mirror = uint8(f), dir, mirror
	g.drawPillowHair(dst, p.Look, pose, turning, x+pillowX, y+pillowY)
	putSprite(dst, laidSprite(k), x, y)
	if k.cover == 2 {
		g.drawZzz(dst, x+pillowX+2, y+pillowY-8, float64(p.ID))
	}
	g.armourByBed(dst, p, x, y)
}

// turnStep is the step of turning over from pose a to pose b: the first
// half of the turn or the second. Onto a side: rolling on the back, then on
// the side not yet curled up; back from a side, the same the other way.
func turnStep(a, b uint8, first bool) (f int, d sim.Dir, mirror bool) {
	side, toSide := b, true
	if b == 0 {
		side, toSide = a, false
	}
	if first == toSide {
		return 0, sim.DirDown, side == 2 // rolling: the other way for the other side
	}
	return 1, sim.DirRight, side == 1
}

// armourByBed lays p's armour on the ground beside their bed while they
// sleep, the helmet by it.
func (g *Game) armourByBed(dst drawTarget, p protocol.PlayerView, x, y float64) {
	if p.Armor <= sim.TierStone {
		return
	}
	a := g.art.armor(p.Armor, p.Look.Female)
	drawAt(dst, a.whole[sim.DirDown][0], x+40, y-3, 1, 1, -math.Pi/2, nil)
	drawAt(dst, a.helmet[sim.DirDown][0], x+43, y+3, 1, 1, -math.Pi/2, nil)
}

// drawSleeping draws p curled up asleep on the bare ground at (x, y) under
// a hide, breathing.
func (g *Game) drawSleeping(dst drawTarget, p protocol.PlayerView, x, y float64) {
	k := laidKey{look: p.Look.Sanitized(), wear: p.Wear, tint: playerColors[int(p.ID)%len(playerColors)], m: moSleep, lay: true, cover: 2, thin: true}
	f, dir, mirror := sleepPose(3)
	if math.Sin(float64(g.frame)*2*math.Pi/190) > 0 {
		f++
	}
	k.f, k.dir, k.mirror = uint8(f), dir, mirror
	g.drawPillowHair(dst, p.Look, 3, false, x+pillowX, y+pillowY)
	putSprite(dst, laidSprite(k), x, y)
	g.drawZzz(dst, x+pillowX+2, y+pillowY-8, float64(p.ID))
}

// waking reports the frame of moWake p is on, if they got up a moment ago
// and stand where they slept.
func (g *Game) waking(p protocol.PlayerView, st *moState) (int, bool) {
	age := float64(g.frame - st.wokeAt)
	if st.wokeAt == never || age < 0 || age >= moWake.span() {
		return 0, false
	}
	return moWake.frameAt(age), true
}

// sleepReels are the motion sheets' scenes of sleep: going to bed, turning
// over each way (the clock jumped on to just before each turn), waking; and
// curled up asleep on the bare ground.
func (g *Game) sleepReels() []reel {
	turnAt := func(k int) int { // the sleep age of turn k
		return k*sleepTurnEvery - 211*int(sheetID) + bedLie
	}
	inBed := func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
		st := g.moveState(p.ID)
		p.Asleep, p.Armor = true, sim.TierCopper
		age := t
		switch {
		case t < 100:
		case t < 170:
			age = turnAt(3) + t - 100 - 12 // onto the side, facing in
		case t < 240:
			age = turnAt(4) + t - 170 - 12 // back
		case t < 310:
			age = turnAt(5) + t - 240 - 12 // onto the other side
		}
		st.asleepAt, st.wokeAt = g.frame-age, never
		if t >= 310 {
			p.Asleep = false
			st.wokeAt, st.asleepAt = g.frame-(t-310), never
			st.movedAt = g.frame - 1000
		}
	}
	bed := func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		drawAt(dst, g.art.bed, d.sx, d.sy, 1, 1, 0, nil)
		g.drawPlayer(dst, p, d)
	}
	ground := func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		g.drawSleeping(dst, p, d.sx, d.sy+4)
	}
	return []reel{
		{name: "sleep", n: 390, dirs: []sim.Dir{sim.DirDown}, at: inBed, draw: bed},
		{name: "hair on the pillow", n: 140, dirs: []sim.Dir{sim.DirDown}, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Look.Hair, p.Look.HairColor = sim.HairStyle(t/20%int(sim.NumHairStyles)), uint8(t/20%int(sim.NumHairColors))
			p.Look.Female = t/20%2 == 1
			p.Asleep = true
			g.moveState(p.ID).asleepAt = g.frame - 200
		}, draw: bed},
		{name: "sleep on the ground", n: 200, dirs: []sim.Dir{sim.DirDown}, draw: ground},
	}
}
