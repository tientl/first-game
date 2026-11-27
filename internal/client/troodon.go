package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The night hunter (sim/nightstalker.go): Troodon, a small, long-legged,
// feathered dinosaur the colour of charcoal, with a pale throat, a long
// stiff tail barred to its dark point, and huge amber eyes. It is drawn from
// dinos/troodon.png by the rig in dinorigs.go. In the dark there is little
// of it to see but its eyes, which shine: drawEyeshine paints them over the
// night, after the lighting.

const (
	troodonStand = iota
	troodonWalkA
	troodonWalkB
	troodonRunA
	troodonRunB
	troodonStalk // crouched low, watching
	troodonHiss  // the same, jaws parted
	troodonLeap  // darting in, off the ground
	troodonBite  // the snout thrown forward, jaws wide
	troodonPeer  // head up, looking about
	troodonHurt
	troodonPoses
)

// The two ambers of the eye in dinos/troodon.png, which nothing else in the
// drawing uses: the eyeshine finds the eye of each pose by them.
var troodonEye = [2]color.RGBA{hex(0xffb71e), hex(0xffdd62)}

// troodonArt holds its sprites; the first index is 0 facing right, 1 left.
type troodonArt struct {
	s [2][troodonPoses]sprite
	// eye is where the middle of the eye is in each sprite, from its anchor.
	eye [2][troodonPoses][2]float64
}

func troodonCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("troodon", pose); ok {
		if ex, ey, ok := eyeOf(c); ok {
			redrawEye(c, int(math.Floor(ex)), int(math.Floor(ey)))
		}
		return c, ax, ay
	}
	return newCanvas(1, 1), 0, 1 // the drawing is embedded; this is never reached
}

// / redrawEye draws the eye afresh about pixel (cx, cy): turning and crouching
// the drawing leaves so small a thing ragged.
func redrawEye(c *canvas, cx, cy int) {
	ink := hex(0x17120f)
	for i, p := range c.px {
		if p == troodonEye[0] || p == troodonEye[1] {
			c.px[i] = ink
		}
	}
	if c.dens > 1 { // room for a round eye: socket, iris, a slit of a pupil and a glint
		k := float64(c.dens)
		for dy := -6; dy <= 6; dy++ {
			for dx := -6; dx <= 6; dx++ {
				switch d := math.Hypot(float64(dx), float64(dy)) / k; {
				case d <= 1.4 && dx <= 0 && dy <= 0 && d > 0.7:
					c.put(cx+dx, cy+dy, troodonEye[1]) // the lit side of the iris
				case d <= 1.4:
					c.put(cx+dx, cy+dy, troodonEye[0])
				case d <= 1.85 && c.at(cx+dx, cy+dy).A >= 128:
					c.put(cx+dx, cy+dy, ink)
				}
			}
		}
		for dy := -3; dy <= 3; dy++ {
			c.put(cx, cy+dy, hex(0x120a04))
		}
		c.put(cx+1, cy, hex(0x120a04))
		c.put(cx-1, cy, hex(0x120a04))
		c.put(cx-2, cy-2, hex(0xfff6d8))
		return
	}
	for dy := -2; dy <= 1; dy++ {
		for dx := -2; dx <= 1; dx++ {
			if dx == -2 && dy == -2 {
				continue
			}
			c.put(cx+dx, cy+dy, ink) // the brow, and the dark feathers behind
		}
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			c.put(cx+dx, cy+dy, troodonEye[0])
		}
	}
	c.put(cx-1, cy-1, troodonEye[1])
	c.put(cx, cy, hex(0x120a04)) // the pupil
}

// eyeOf is the middle of the eye on a Troodon's canvas; ok is false if the
// pose hides it.
func eyeOf(c *canvas) (x, y float64, ok bool) {
	n := 0.0
	for i, p := range c.px {
		if p == troodonEye[0] || p == troodonEye[1] {
			x += float64(i%c.w) + 0.5
			y += float64(i/c.w) + 0.5
			n++
		}
	}
	if n == 0 {
		return 0, 0, false
	}
	return x / n, y / n, true
}

// eyeOffsets is where the eye at (ex, ey) of canvas c is from its anchor
// (ax, ay), in art pixels, facing right and facing left.
func eyeOffsets(c *canvas, ex, ey, ax, ay float64) (r, l [2]float64) {
	k := float64(max(1, c.dens))
	return [2]float64{(ex - ax) / k, (ey - ay) / k}, [2]float64{(ax - ex) / k, (ey - ay) / k}
}

func genTroodonArt() troodonArt {
	var a troodonArt
	for p := range troodonPoses {
		c, ax, ay := troodonCanvas(p)
		a.s[0][p], a.s[1][p] = sidedPair(mk(c, ax, ay))
		if ex, ey, ok := eyeOf(c); ok {
			a.eye[0][p], a.eye[1][p] = eyeOffsets(c, ex, ey, ax, ay)
		}
	}
	return a
}

// troodonStrikes is the frame at which each Troodon now striking began to.
var troodonStrikes = map[sim.CreatureID]int{}

// / troodonHeld are the Troodon shown held in mid-leap, for screenshots.
var troodonHeld = map[sim.CreatureID]bool{}

// HoldLeap shows the striking Troodon id held in its leap, for screenshots.
func (g *Game) HoldLeap(id sim.CreatureID) { troodonHeld[id] = true }

// troodonLeapFrames is how long a strike is the leap in, before the bite.
const troodonLeapFrames = 9

// troodonWindFrames is how long a strike is the crouch in, before the leap.
const troodonWindFrames = 4

// troodonFrame is the frame a Troodon is drawn in (with where its eye is),
// and how far off the ground.
func (g *Game) troodonFrame(c protocol.CreatureView, moving bool) (rigFrame, float64) {
	a := &g.art.beasts.troodon
	side := 0
	if c.FaceLeft {
		side = 1
	}
	held := func(p int) rigFrame {
		return rigFrame{s: a.s[side][p], eye: a.eye[side][p], hasEye: a.eye[side][p] != ([2]float64{})}
	}
	t := g.frame + int(c.ID)*7
	if c.State != sim.CreatureStrike {
		delete(troodonStrikes, c.ID)
	}
	if f, lift, ok := g.skillFrame(c, side); ok {
		return f, lift // one of its moves: see dinoskills.go
	}
	switch {
	case c.State == sim.CreatureStrike && troodonHeld[c.ID]:
		return held(troodonLeap), 4
	case c.State == sim.CreatureStrike:
		at, ok := troodonStrikes[c.ID]
		if !ok || g.frame < at {
			at = g.frame
			troodonStrikes[c.ID] = at
		}
		age := g.frame - at
		if age < troodonWindFrames { // gathered low to spring
			return held(troodonStalk), 0
		}
		if age -= troodonWindFrames; age < troodonLeapFrames {
			return held(troodonLeap), 2 + math.Round(3*math.Sin(float64(age+3)/float64(troodonLeapFrames+3)*math.Pi))
		}
		return held(troodonBite), 0
	case c.State == sim.CreatureHurt:
		return held(troodonHurt), 0
	case moving && (c.State == sim.CreatureChase || c.State == sim.CreatureFlee):
		return rigFrameOf("troodon", side, stridePose(rigPose{tilt: 0.08, head: 0.1, tail: 0.06}, t*gaitFrames/8, 1.4)), 0
	case c.State == sim.CreatureChase: // waiting at the edge of the light
		if (t/45)%6 == 2 {
			return held(troodonHiss), 0
		}
		return held(troodonStalk), 0
	case moving:
		return rigFrameOf("troodon", side, stridePose(rigPose{}, t*gaitFrames/18, 0.8)), 0
	case (t/70)%4 == 1:
		return held(troodonPeer), 0
	}
	return held(troodonStand), 0
}

// troodonSprite is creatureSprite for the night hunter; ok is false for any
// other kind.
func (g *Game) troodonSprite(c protocol.CreatureView, moving bool) (sprite, float64, bool) {
	if c.Kind != sim.CreatureTroodon {
		return sprite{}, 0, false
	}
	f, bob := g.troodonFrame(c, moving)
	return f.s, bob, true
}

// troodonLabel is creatureLabel for the night hunter.
func troodonLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	if c.Kind != sim.CreatureTroodon {
		return "", color.RGBA{}, false
	}
	return c.Kind.Name() + " — đánh (săn đêm theo bầy, sợ ánh lửa!)", colBad, true
}

// drawEyeshine paints the eyes of the night hunters over the dark: two
// small points of amber-gold, the near eye and the far one beyond it,
// brighter the darker it is, and blinking now and then. It is drawn after
// the lighting, so in the black of night the eyes are all that shows.
func (g *Game) drawEyeshine(dst drawTarget, cam camera, alpha, dark float64) {
	k := (dark - 0.2) / 0.45
	if k <= 0 {
		return
	}
	k = math.Min(1, k)
	if len(troodonStrikes) > 256 {
		clear(troodonStrikes) // long gone
	}
	for _, c := range g.curr.Creatures {
		if c.Kind != sim.CreatureTroodon {
			continue
		}
		x, y, moving := g.lerpCreature(c, alpha)
		if g.hiddenInCave(x, y) {
			continue
		}
		t := g.frame + int(c.ID)*37
		if t%(170+int(c.ID%5)*23) < 7 {
			continue // a blink
		}
		dir := 1.0
		if c.FaceLeft {
			dir = -1
		}
		f, bob := g.troodonFrame(c, moving)
		if !f.hasEye {
			continue
		}
		eye := f.eye
		sx, sy := cam.toScreenFine(x, y)
		align(dst, sx, sy) // on the beast's own grid, as drawSprites drew it
		if c.State == sim.CreatureHurt {
			sx += math.Round(math.Sin(float64(g.frame) * 1.3)) // as drawCreature flinches it
		}
		ex := float32(sx + math.Floor(eye[0])) // whole art pixels from the beast's own point; see align
		ey := float32(sy + math.Floor(eye[1]-bob))
		shimmer := 0.88 + 0.12*math.Sin(float64(t)*0.11)
		if c.Skill == sim.SkillTroodonAmbush && c.Phase == sim.PhaseWindup {
			shimmer = 1.6 + 0.4*math.Sin(float64(t)*0.5) // about to spring: its eyes catch the light
		}
		glint(dst, ex, ey, k*shimmer)
		glint(dst, ex+float32(dir*4), ey, k*shimmer*0.75) // the far eye, past the snout
	}
}

// glint draws one shining eye at pixel (x, y): a bright point in a faint
// cross of light.
func glint(dst drawTarget, x, y float32, k float64) {
	px := func(dx, dy float32, c color.RGBA, a float64) {
		a = math.Max(0, math.Min(1, a))
		fillRect(dst, x+dx, y+dy, 1, 1, color.RGBA{
			uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a),
		}, false)
	}
	halo := color.RGBA{0xe8, 0xc2, 0x30, 0xff}
	for _, d := range [][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		px(d[0], d[1], halo, 0.6*k)
	}
	for _, d := range [][2]float32{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		px(d[0], d[1], halo, 0.24*k)
	}
	px(0, 0, color.RGBA{0xff, 0xf2, 0x9a, 0xff}, 0.5+0.5*k)
}
