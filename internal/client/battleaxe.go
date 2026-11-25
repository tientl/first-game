package client

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The war axe (sim.ItemBattleAxe): how it sits in the hand, and its blow at
// a beast. It is the slowest weapon and the hardest hitting, so the blow is
// neither the knife's quick slash nor the spear's thrust: the axe is heaved
// back over the head, crashes down in front, bites, and is hauled up again.
// The whole of it outlasts the server's short working pose, so it runs on
// its own clock from the frame the blow began (see cleaving).

var battleAxeHeld sprite // drawn on first use

// battleAxeSprite is the axe as carried: a double-bitted jade head lashed
// with hide to a long haft, gripped low.
func battleAxeSprite() sprite {
	if battleAxeHeld.img == nil {
		c := newCanvas(15, 20)
		iiPut(c, 1, 1, iiRs(iiWood, iiJade, iiHide),
			"......32.....",
			"..e...32...c.",
			".ed...32...cb",
			".edd..DB..cbb",
			"edddcbDBbccba",
			"edddcbCAbccba",
			"edddcbDBbccba",
			".edd..DB..cba",
			".ed...32...ba",
			"..e...32...a.",
			"......32.....",
			"......32.....",
			"......32.....",
			"......32.....",
			"......32.....",
			"......DB.....",
			"......CA.....",
			"......31.....")
		c.outline(0.8)
		battleAxeHeld = mk(c, 8, 17)
	}
	return battleAxeHeld
}

// battleAxeHead is how far up the haft from the grip the head sits, in art
// pixels.
const battleAxeHead = 11.0

// drawBattleAxeHeld draws the axe at rest in the hand at (hx, hy), tipped
// well out to the side or ahead, so the head stays clear of the face
// whichever way the player looks.
func drawBattleAxeHeld(dst drawTarget, dir sim.Dir, hx, hy, side float64) {
	drawAt(dst, battleAxeSprite(), math.Round(hx), math.Round(hy)+1, side, 1, (cleaveCarry+heldTurn)*side, nil)
}

// A blow, in frames at 60 a second.
const (
	cleaveRaise  = 4.0  // heaved up over the head
	cleaveFall   = 4.0  // brought down
	cleaveStick  = 10.0 // it stays where it bit
	cleaveHeave  = 20.0 // and is hauled back up
	cleaveFrames = cleaveRaise + cleaveFall + cleaveStick + cleaveHeave

	cleaveCarry = 1.2 // radians from upright, as carried
)

// cleavePose is a blow as seen from one facing: the angles the axe swings
// between, and where the grip goes, in art pixels from the player's anchor
// with x towards the hand side. Raised, the axe is high and clear of the
// face: behind the head when the player is seen from the side, up beside it
// when seen from the front or the back.
type cleavePose struct {
	back, down float64 // radians from upright: raised, and landed
	upX, upY   float64 // the grip when raised
	outX, outY float64 // and when landed
}

func cleavePoseFor(dir sim.Dir) cleavePose {
	switch dir {
	case sim.DirLeft, sim.DirRight:
		return cleavePose{back: -0.75, down: 1.95, upX: -2, upY: -19, outX: 7, outY: -5}
	case sim.DirUp: // the target is beyond the player: it lands high
		return cleavePose{back: 0.25, down: 1.35, upX: 9, upY: -15, outX: 7, outY: -8}
	}
	return cleavePose{back: 0.25, down: 2.45, upX: 9, upY: -15, outX: 7, outY: -6}
}

// cleaveAngle is the axe's angle age frames into a blow of pose ps.
func cleaveAngle(ps cleavePose, age float64) float64 {
	switch {
	case age < cleaveRaise: // snatched up, slowing at the top
		u := age / cleaveRaise
		return cleaveCarry + (ps.back-cleaveCarry)*(1-(1-u)*(1-u))
	case age < cleaveRaise+cleaveFall: // gathering speed all the way down
		u := (age - cleaveRaise) / cleaveFall
		return ps.back + (ps.down-ps.back)*u*u*u
	case age < cleaveRaise+cleaveFall+cleaveStick: // a judder as it bites
		u := (age - cleaveRaise - cleaveFall) / cleaveStick
		return ps.down - 0.12*math.Sin(u*math.Pi)*(1-u)
	case age < cleaveFrames:
		u := (age - cleaveRaise - cleaveFall - cleaveStick) / cleaveHeave
		return ps.down + (cleaveCarry-ps.down)*smooth(u)
	}
	return cleaveCarry
}

// cleaving reports whether p is in the middle of a blow of the war axe.
func (g *Game) cleaving(p protocol.PlayerView) bool {
	if p.Held != sim.ItemBattleAxe || g.swingWhat[p.ID].kind != swingCleave {
		return false
	}
	at, ok := g.swingAt[p.ID]
	return ok && float64(g.frame-at) < cleaveFrames
}

// drawCleave draws p's war axe in the middle of a blow: the axe, the broad
// sweep its head leaves on the way down, and the burst where it lands.
func (g *Game) drawCleave(dst drawTarget, p protocol.PlayerView, d drawable, bob float64) {
	dir := p.Dir & 3
	side := facingSide(dir)
	ps := cleavePoseFor(dir)
	age := float64(g.frame - g.swingAt[p.ID])
	a := cleaveAngle(ps, age)
	// The grip starts and ends at the hand, where the axe is carried; it is
	// lifted with the axe, and pushed out and down as it lands.
	rx, ry := handAt(dir, 0)
	rx, ry = rx*side, ry+1 // handAt is in screen pixels: fold it to the hand side
	up := clamp01((cleaveCarry - a) / (cleaveCarry - ps.back))
	out := clamp01((a - cleaveCarry) / (ps.down - cleaveCarry))
	gx := d.sx + math.Round(rx+(ps.upX-rx)*up+(ps.outX-rx)*out)*side
	gy := d.sy - bob + math.Round(ry+(ps.upY-ry)*up+(ps.outY-ry)*out)
	if limbGrip.on { // the jointed figure's hand; see limbdraw.go
		gx, gy = limbGrip.x, limbGrip.y
	}
	at := func(b, r float64) (float64, float64) { return gx + math.Sin(b)*r*side, gy - math.Cos(b)*r }
	dot := func(x, y, w, white, alpha float64) {
		fillRect(dst, float32(math.Round(x)), float32(math.Round(y)), float32(w), float32(w),
			premul(mix(iiJade[4], colWhite, white), alpha), false)
	}

	// The sweep: everything the head has passed through since the top,
	// brightest just behind it, as wide as the head.
	landed := age - cleaveRaise - cleaveFall
	if age >= cleaveRaise && landed < 8 {
		fade := 1.0
		if landed > 0 {
			fade = 1 - landed/8
		}
		const tail = 2.4 // radians of sweep behind the head
		for s := 0.0; s <= 1; s += 1.0 / 72 {
			b := a - tail*s
			if b < ps.back {
				break
			}
			for r := battleAxeHead - 4; r <= battleAxeHead+4.5; r++ {
				edge := 1 - math.Abs(r-battleAxeHead-1.5)/6 // strongest at the bits
				x, y := at(b, r)
				dot(x, y, 1, 0.4, (1-s)*math.Sqrt(1-s)*fade*0.9*edge)
			}
		}
	}

	drawAt(dst, battleAxeSprite(), gx, gy, side, 1, a*side, nil)

	// Where it lands: a flash, chips flying up and out, dust along the ground.
	if landed >= 0 && landed < 9 {
		u := landed / 9
		hx, hy := at(ps.down, battleAxeHead+1)
		if landed < 2 {
			for i := -3.0; i <= 3; i++ {
				dot(hx+i*side, hy+math.Abs(i)*0.4-1, 2, 0.85, 0.9)
			}
		}
		for i := 0; i < 7; i++ {
			th := -math.Pi/2 + (float64(i)-3)*0.42 // a fan upwards
			reach := (2.5 + 7*hashf(i, 3, uint32(p.ID))) * (1 - (1-u)*(1-u))
			x := hx + math.Cos(th)*reach*1.3 + side*2*u
			y := hy + math.Sin(th)*reach + 6*u*u // flung up, falling back
			dot(x, y, 1, 0.4+0.15*float64(i%3), 1-u)
		}
		for i := -1.0; i <= 1; i += 2 { // dust, low and wide
			fillRect(dst, float32(math.Round(hx+i*(3+7*u))), float32(math.Round(hy+2-2*u)), 2, 2,
				premul(iiShell[3], 0.6*(1-u)), false)
		}
	}
}
