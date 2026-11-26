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
// body) and covered from the shoulders with a fur blanket that follows the
// shape under it: over the knees, into the hollow of the waist. It fits the
// bed: on the back the knees are up, on the side it curls. It breathes,
// slowly, the blanket rising with the chest, and turns over now and then:
// on its back, onto one side, back, onto the other. Going to bed it sits on
// the bed's edge, lies back and pulls the blanket up; waking (moWake) it
// sits up, stands, and stretches with a yawn.
//
// Each drawing is made the first time it is shown and kept: a few poses, a
// breath in and out, the blanket up or half way.

// sleepKey names one drawing of a sleeper.
type sleepKey struct {
	look   sim.Look
	wear   sim.Wear
	tint   color.RGBA
	pose   uint8 // 0 on the back, 1 on the side facing out, 2 on the side facing in, 3 curled on the ground, 4 on the back, stirring
	breath uint8 // 0 out, 1 in
	cover  uint8 // the blanket: 0 none, 1 to the hips, 2 to the shoulders
}

var sleepCache = map[sleepKey]sprite{}

// The art pixel of a sleeper's drawing that goes on its bed's anchor is
// where the middle of the head is, put on the pillow.
const (
	pillowX, pillowY = -8.0, -8.0 // the pillow's middle from the bed's anchor, art pixels
)

// sleepFrame is the frame of moSleep, the facing it is drawn in, and
// whether it is mirrored, for pose.
func sleepFrame(pose uint8) (f int, d sim.Dir, mirror bool) {
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

// sleeperSprite returns the drawing k, making it the first time.
func sleeperSprite(k sleepKey) sprite {
	if s, ok := sleepCache[k]; ok {
		return s
	}
	f, d, mirror := sleepFrame(k.pose)
	f += int(k.breath)
	lo, up := moSleep.keys(d, f)
	c, drop := standingCanvas(k.look, k.wear, k.tint, up, lo, mirror)
	lay := layFlat(c)
	if k.cover > 0 {
		blanket(lay, shoulderY-2+drop, float64(k.cover)/2, k.pose == 3)
	}
	// The head's middle, laid down: it goes on the pillow.
	hx, hy := 35+drop, float64(c.w-1)-(27+limbPad*HDDensity)
	s := cropSprite(lay, hx/HDDensity-pillowX, hy/HDDensity-pillowY)
	sleepCache[k] = s
	return s
}

// standingCanvas draws frame (up, lo) of look l wearing w whole on one
// canvas, standing, in the player's colour tint, mirrored if asked; and how
// far its hips dropped.
func standingCanvas(l sim.Look, w sim.Wear, tint color.RGBA, up upKey, lo loKey, mirror bool) (*canvas, float64) {
	uc := bakeUpper(l, w, up, headCache{})
	lc, lp := bakeLower(l, w, lo)
	c := newCanvas(limbW, limbH)
	drop := int(math.Round(lp.drop))
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
	put(uc.layer[upFar], true, drop)
	put(uc.layer[upBack], false, drop)
	put(lc, false, 0)
	put(uc.layer[upBody], false, drop)
	put(uc.layer[upShirt], true, drop)
	put(uc.layer[upArms], true, drop)
	put(uc.layer[upTop], false, drop)
	put(uc.layer[upArmsOver], true, drop)
	put(uc.layer[upTopOver], false, drop)
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

var blanketFur = mkRamp(0x3a2618, 0x553823, 0x744e31, 0x946a43, 0xb48a5c, 0xd2ad7d)

// blanket lays a fur over the laid-down figure on c from column x0 (the
// shoulders) to the feet, as far as cover (0 to 1) of the way up from the
// feet: over everything there, a little wider, its gaps closed, so it drapes
// over the knees and down to the waist. Lit from above, the fur's lie
// along it, a turned-down hem at the top. thin is for a sleeper on the
// bare ground: a hide just over the legs and hips.
func blanket(c *canvas, x0, cover float64, thin bool) {
	in := make([]bool, len(c.px))
	maxX := 0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A > 0 {
				in[y*c.w+x] = true
				maxX = max(maxX, x)
			}
		}
	}
	grow := func(m []bool, r int, keep bool) []bool {
		out := make([]bool, len(m))
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				v := keep
				for dy := -r; dy <= r && v == keep; dy++ {
					for dx := -r; dx <= r; dx++ {
						if dx*dx+dy*dy > r*r+r {
							continue
						}
						xx, yy := x+dx, y+dy
						on := xx >= 0 && yy >= 0 && xx < c.w && yy < c.h && m[yy*c.w+xx]
						if on != keep {
							v = !keep
							break
						}
					}
				}
				out[y*c.w+x] = v
			}
		}
		return out
	}
	mask := grow(grow(in, 4, false), 2, true) // closed: the gaps between the limbs filled
	from := float64(maxX) + 2 - cover*(float64(maxX)+2-x0)
	if thin {
		from = math.Max(from, x0+22)
	}
	for i := range mask {
		x := i % c.w
		mask[i] = mask[i] && float64(x) >= from && x <= maxX+2
	}
	// How deep inside the blanket each pixel is, for the light.
	depth := make([]int, len(mask))
	cur := mask
	for k := 1; k <= 6; k++ {
		cur = grow(cur, 1, true)
		for i, v := range cur {
			if v {
				depth[i] = k
			}
		}
	}
	top := make([]int, c.w)
	bot := make([]int, c.w)
	for x := 0; x < c.w; x++ {
		top[x], bot[x] = c.h, -1
		for y := 0; y < c.h; y++ {
			if mask[y*c.w+x] {
				top[x], bot[x] = min(top[x], y), max(bot[x], y)
			}
		}
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			i := y*c.w + x
			if !mask[i] {
				continue
			}
			v := 0.0
			if bot[x] > top[x] {
				v = float64(y-top[x]) / float64(bot[x]-top[x]) // 0 on top
			}
			l := 0.78 - 0.5*v + 0.05*float64(min(depth[i], 4))
			// The fur lies along the body: streaks, broken.
			if hashf(x/4, y, 811) < 0.22 {
				l -= 0.12
			} else if hashf(x/3, y, 812) < 0.1 {
				l += 0.1
			}
			hem := float64(x) - from
			switch {
			case hem < 3:
				l = 0.82 - 0.06*hem // the hem, turned down, lit
				if hashf(x, y/2, 813) < 0.3 {
					l -= 0.15
				}
			case hem < 4:
				l = 0.25 // the shadow under it
			}
			col := toneOf(blanketFur, l)
			if depth[i] <= 1 { // the edge
				col = mix(col, colInk, 0.45)
			}
			c.px[i] = col
		}
	}
}

// drawSleeperHD draws p asleep at d, in their bed (the bed is drawn under
// them as an entity): lying down first if they have only just gone to bed,
// then asleep, breathing, turning over now and then.
func (g *Game) drawSleeperHD(dst drawTarget, p protocol.PlayerView, d drawable) {
	st := g.moveState(p.ID)
	age := float64(g.frame - st.asleepAt)
	if st.asleepAt == never {
		age = 1e9
	}
	tint := playerColors[int(p.ID)%len(playerColors)]
	k := sleepKey{look: p.Look.Sanitized(), wear: p.Wear, tint: tint, cover: 2}
	b := g.art.bodyIn(p.Look, p.Wear)
	switch {
	case age < 14: // down onto the bed
		lo, up := moSitDown.keys(sim.DirDown, 1)
		drawFig(dst, figDraw{b: b, up: up, lo: lo, tint: tint}, d.sx-1, d.sy-4)
		return
	case age < 28: // sat on its edge
		lo, up := moSit.keys(sim.DirDown, 0)
		drawFig(dst, figDraw{b: b, up: up, lo: lo, tint: tint}, d.sx-3, d.sy-4)
		return
	case age < 36:
		k.cover = 0
	case age < 46:
		k.cover = 1
	}
	// Asleep: a breath every three seconds or so; over on one side, back,
	// over on the other, every nine.
	t := age + float64(p.ID)*211
	if math.Sin(t*2*math.Pi/190) > 0 {
		k.breath = 1
	}
	k.pose = [4]uint8{0, 1, 0, 2}[int(t/540)%4]
	if k.cover < 2 {
		k.pose, k.breath = 0, 0
	}
	g.drawSleeping(dst, k, d.sx, d.sy, p.ID)
}

// drawSleeping draws sleeper k with its bed's anchor at (x, y), and the z's
// floating up from the head.
func (g *Game) drawSleeping(dst drawTarget, k sleepKey, x, y float64, id sim.PlayerID) {
	s := sleeperSprite(k)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x)-s.ax, math.Round(y)-s.ay)
	drawSprite(dst, s, op)
	if k.cover == 2 {
		g.drawZzz(dst, x+pillowX+2, y+pillowY-8, float64(id))
	}
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

// sleepReels are the motion sheets' scenes of sleep: going to bed, asleep
// and turning over, waking; and curled up asleep on the bare ground.
func (g *Game) sleepReels() []reel {
	inBed := func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
		st := g.moveState(p.ID)
		// Down at 0, asleep, turned over on the clock, up at 260.
		const up = 260
		if t < up {
			p.Asleep = true
			st.asleepAt = g.frame - t
			if t >= 60 {
				st.asleepAt = g.frame - t*9 // asleep a while: turned over now and then
			}
			st.wokeAt = never
		} else {
			st.wokeAt = g.frame - (t - up)
			st.movedAt = g.frame - 1000
		}
	}
	bed := func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		drawAt(dst, g.art.bed, d.sx, d.sy, 1, 1, 0, nil)
		g.drawPlayer(dst, p, d)
	}
	ground := func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		k := sleepKey{look: p.Look.Sanitized(), wear: p.Wear, tint: playerColors[int(p.ID)%len(playerColors)], pose: 3, cover: 2}
		if math.Sin(float64(g.frame)*2*math.Pi/190) > 0 {
			k.breath = 1
		}
		g.drawSleeping(dst, k, d.sx, d.sy+4, p.ID)
	}
	return []reel{
		{name: "sleep", n: 360, dirs: []sim.Dir{sim.DirDown}, at: inBed, draw: bed},
		{name: "sleep on the ground", n: 200, dirs: []sim.Dir{sim.DirDown}, draw: ground},
	}
}
