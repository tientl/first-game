package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Bare hands. A blow with nothing in hand, or with something that is no
// tool, is thrown by the arm itself: the body is drawn without that arm
// (playerLayersPose in look.go) and the arm is drawn over it in a pose of
// its own, a sleeve in the shirt's greys and a clenched fist. At a beast it
// is a punch: drawn back, jabbed out as the body leans in, a spark where it
// lands, and back; the next one is thrown with the other hand. At anything
// else (picking, gathering, stoking) it is a slower reach, with whatever is
// held in the fist.
//
// The same arms are flung out when a player is hurt: see jolt.go.

// armSide names an arm of the figure.
type armSide uint8

const (
	armNone armSide = iota
	armHand         // the hand that holds things: on the right of the screen facing down or up, the near arm sideways
	armOff          // the other one
	armBoth
	numArmSides
)

// armPose is how an arm drawn by itself is held.
type armPose uint8

const (
	poseRest  armPose = iota // on the body, as genPlayer draws it
	poseWind                 // drawn back, the fist by the chest
	poseHalf                 // half way out
	poseFull                 // straight out at the target
	poseFling                // flung out in a start; see jolt.go
	numArmPoses
)

// A punch, in frames at 60 a second from the blow: it lands when the tools
// do (chopStrike), and the arm is back well before the working pose ends.
const (
	punchWind = 2  // drawn back until here
	punchHit  = 3  // out straight from here: the blow lands
	punchHold = 8  // and held until here
	punchEnd  = 12 // the arm is back at the side
	sparkLife = 4  // frames the spark of a landed punch shows
)

// figPose is what a figure's arms and body do on one frame.
type figPose struct {
	hand, off armPose
	lean      float64 // how far the body leans towards the facing, in art pixels
	spark     int     // frames since a punch landed; negative for none
}

// bareArt is a body's shirt and top layers drawn without an arm or two.
type bareArt struct {
	shirt, top [4][walkFrames]sprite
}

// armArt is one arm in one pose: the sleeve, tinted like the shirt, and the
// fist. hx, hy is where the hand is, as handAt gives it; none means the arm
// is out of sight.
type armArt struct {
	sleeve, fist sprite
	hx, hy       float64
	size         int // the fist's width
	none         bool
}

// fistArt is every arm of one look, by facing, arm and pose.
type fistArt [4][numArmSides][numArmPoses]armArt

// bareBody returns b's shirt and top without the arms hide names, drawing
// them on first use. Like art.body it must be called inside the game loop.
func (b *bodyArt) bareBody(hide armSide) *bareArt {
	if hide == armNone {
		return &bareArt{b.shirt, b.top}
	}
	if b.bare[hide] != nil {
		return b.bare[hide]
	}
	a := &bareArt{}
	for d := sim.DirDown; d <= sim.DirUp; d++ {
		for f := range walkFrames {
			_, _, S, T := playerLayersPose(b.look, d, f, false, hide)
			a.shirt[d][f], a.top[d][f] = mk(S, 9, bodyFeet), mk(T, 9, bodyFeet)
		}
	}
	// Sideways only the near arm can go: the far one is behind the body.
	a.shirt[sim.DirRight], a.top[sim.DirRight] = b.shirt[sim.DirRight], b.top[sim.DirRight]
	a.shirt[sim.DirLeft], a.top[sim.DirLeft] = b.shirt[sim.DirLeft], b.top[sim.DirLeft]
	if hide != armOff {
		for f := range walkFrames {
			_, _, S, T := playerLayersPose(b.look, sim.DirRight, f, false, armHand)
			a.shirt[sim.DirRight][f], a.top[sim.DirRight][f] = mk(S, 9, bodyFeet), mk(T, 9, bodyFeet)
			a.shirt[sim.DirLeft][f], a.top[sim.DirLeft][f] = mirrorSprite(a.shirt[sim.DirRight][f]), mirrorSprite(a.top[sim.DirRight][f])
		}
	}
	b.bare[hide] = a
	return a
}

// arms returns b's arms, drawing them on first use.
func (b *bodyArt) arms() *fistArt {
	if b.fists != nil {
		return b.fists
	}
	fa := &fistArt{}
	for d := sim.DirDown; d <= sim.DirRight; d++ {
		for _, side := range []armSide{armHand, armOff} {
			for pose := poseWind; pose < numArmPoses; pose++ {
				fa[d][side][pose] = genArm(b.look, d, side, pose)
			}
		}
	}
	b.fists = fa
	return fa
}

// armPad is how far the arm sprites reach past the body's 18 columns on each
// side: a fist thrown sideways goes well beyond them.
const armPad = 6

// armShape is an arm in the body sprite's pixels: the sleeve is a 2x2 brush
// drawn through pts (its top-left corner), the fist a square of size pixels
// with its top-left at (fx, fy).
type armShape struct {
	pts    [][2]int
	fx, fy int
	size   int
	near   bool // thrown at the viewer: the fist is in the light
	none   bool
}

// armShapeOf is the shape of the arm on the hand side of a figure facing d
// (never DirLeft), or of the far arm sideways. Facing down or up the other
// arm is its mirror image.
func armShapeOf(female bool, d sim.Dir, side armSide, pose armPose) armShape {
	rx := 13 // the arm's left column, as in playerLayersPose
	if female {
		rx = 12
	}
	switch d {
	case sim.DirDown:
		switch pose {
		case poseWind: // cocked: the elbow down, the fist up by the shoulder
			return armShape{pts: [][2]int{{rx, 16}, {rx + 1, 17}}, fx: rx + 1, fy: 15, size: 2}
		case poseHalf:
			return armShape{pts: [][2]int{{rx, 16}, {rx, 20}}, fx: rx, fy: 22, size: 2}
		case poseFull: // down at the viewer: the fist comes nearer and bigger
			return armShape{pts: [][2]int{{rx, 16}, {rx, 18}, {rx - 1, 20}, {rx - 1, 21}}, fx: rx - 3, fy: 23, size: 3, near: true}
		case poseFling:
			return armShape{pts: [][2]int{{rx, 16}, {rx + 1, 16}, {rx + 2, 15}}, fx: rx + 3, fy: 13, size: 2}
		}
	case sim.DirUp:
		switch pose {
		case poseWind:
			return armShape{pts: [][2]int{{rx, 16}, {rx + 1, 17}, {rx + 1, 18}}, fx: rx, fy: 20, size: 2}
		case poseHalf:
			return armShape{pts: [][2]int{{rx, 16}, {rx + 1, 16}, {rx + 2, 15}}, fx: rx + 2, fy: 13, size: 2}
		case poseFull: // up and away, past the head
			return armShape{pts: [][2]int{{rx, 16}, {rx + 1, 15}, {rx + 2, 14}, {rx + 2, 13}}, fx: rx + 2, fy: 11, size: 2}
		case poseFling:
			return armShape{pts: [][2]int{{rx, 16}, {rx + 1, 16}, {rx + 2, 15}}, fx: rx + 3, fy: 13, size: 2}
		}
	case sim.DirRight:
		if side == armOff { // the far arm comes out from behind the body
			switch pose {
			case poseHalf:
				return armShape{pts: [][2]int{{10, 18}, {11, 18}}, fx: 13, fy: 18, size: 2}
			case poseFull:
				return armShape{pts: [][2]int{{10, 18}, {14, 18}}, fx: 16, fy: 17, size: 3}
			}
			return armShape{none: true}
		}
		switch pose {
		case poseWind: // the elbow back, the fist at the hip
			return armShape{pts: [][2]int{{8, 16}, {7, 17}, {6, 18}}, fx: 8, fy: 19, size: 2}
		case poseHalf:
			return armShape{pts: [][2]int{{8, 16}, {9, 17}, {10, 18}}, fx: 12, fy: 18, size: 2}
		case poseFull: // a straight jab
			return armShape{pts: [][2]int{{8, 16}, {8, 17}, {9, 18}, {13, 18}}, fx: 15, fy: 17, size: 3}
		case poseFling: // thrown up behind
			return armShape{pts: [][2]int{{8, 16}, {7, 16}, {6, 15}}, fx: 4, fy: 13, size: 2}
		}
	}
	return armShape{none: true}
}

// armCanvases draws one arm of look l facing d (never DirLeft): the sleeve
// and the fist, on canvases armPad wider than the body on each side.
func armCanvases(l sim.Look, d sim.Dir, side armSide, pose armPose) (sleeve, fist *canvas, sh armShape) {
	l = l.Sanitized()
	sh = armShapeOf(l.Female, d, side, pose)
	w, h := bodyW+2*armPad, bodyH+2
	sleeve, fist = newCanvas(w, h), newCanvas(w, h)
	if sh.none {
		return sleeve, fist, sh
	}
	sideways := d == sim.DirRight
	if !sideways && side == armOff {
		// The mirror image, about the middle of the body.
		for i := range sh.pts {
			sh.pts[i][0] = bodyW - 2 - sh.pts[i][0]
		}
		sh.fx = bodyW - sh.size - sh.fx
	}
	arm := make(map[[2]int]bool)
	stamp := func(x, y int) {
		for _, o := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
			arm[[2]int{x + o[0], y + o[1]}] = true
		}
	}
	for i, p := range sh.pts {
		stamp(p[0], p[1])
		if i == 0 {
			continue
		}
		q := sh.pts[i-1]
		n := max(absInt(p[0]-q[0]), absInt(p[1]-q[1]))
		for k := 1; k < n; k++ {
			stamp(q[0]+(p[0]-q[0])*k/n, q[1]+(p[1]-q[1])*k/n)
		}
	}
	inFist := func(x, y int) bool {
		return x >= sh.fx && x < sh.fx+sh.size && y >= sh.fy && y < sh.fy+sh.size
	}
	// The sleeve: lit from the top left; the arm on the right of the screen
	// and the far arm are in the body's shade.
	hi, lo := 3, 1
	if l.Female {
		hi, lo = 3, 2
	}
	shaded := sideways && side == armOff || !sideways && side == armHand
	switch {
	case sideways && !shaded:
		hi, lo = 3, 2
	case shaded && l.Female:
		hi, lo = 2, 1
	case shaded:
		hi, lo = 1, 0
	}
	for p := range arm {
		x, y := p[0], p[1]
		if inFist(x, y) {
			continue
		}
		tone := lo
		lit := !arm[[2]int{x - 1, y}] || !arm[[2]int{x, y - 1}]
		if shaded && !sideways {
			lit = arm[[2]int{x - 1, y}] // its inner edge is the dark one
		}
		if lit {
			tone = hi
		}
		sleeve.put(x+armPad, y, shirtRamp.at(tone))
	}
	// The fist: a lit knuckle, a shaded heel.
	sk := skinTones[l.Skin].ramp
	lit := 2
	if shaded && !sh.near {
		lit = 1
	}
	for y := sh.fy; y < sh.fy+sh.size; y++ {
		for x := sh.fx; x < sh.fx+sh.size; x++ {
			tone := lit
			switch {
			case x == sh.fx && y == sh.fy:
				tone = lit + 1
			case x == sh.fx+sh.size-1 && y == sh.fy+sh.size-1:
				tone = lit - 1
			case sh.size > 2 && (x == sh.fx+sh.size-1 || y == sh.fy+sh.size-1):
				tone = lit - 1
			}
			fist.put(x+armPad, y, sk.at(tone))
		}
	}
	rimLayers(sleeve, fist)
	return sleeve, fist, sh
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// genArm draws one arm of look l as sprites anchored like the body's.
func genArm(l sim.Look, d sim.Dir, side armSide, pose armPose) armArt {
	if d == sim.DirLeft {
		a := genArm(l, sim.DirRight, side, pose)
		if a.none {
			return a
		}
		a.sleeve, a.fist, a.hx = mirrorSprite(a.sleeve), mirrorSprite(a.fist), -a.hx
		return a
	}
	S, F, sh := armCanvases(l, d, side, pose)
	if sh.none {
		return armArt{none: true}
	}
	return armArt{
		sleeve: mk(S, 9+armPad, bodyFeet),
		fist:   mk(F, 9+armPad, bodyFeet),
		hx:     float64(sh.fx) + float64(sh.size-1)/2 - 9,
		hy:     float64(sh.fy) + float64(sh.size+1)/2 - bodyFeet,
		size:   sh.size,
	}
}

// fistHolds reports whether item k is held in a closed fist, so that the arm
// can be thrown out with it: nothing at all, or anything that is not a tool,
// a weapon, a rod or a torch, which have swings of their own.
func fistHolds(k sim.ItemKind) bool {
	if !k.Valid() {
		return true
	}
	if class, _ := k.ToolKind(); class != sim.NoTool {
		return false
	}
	if isSword(k) {
		return false
	}
	switch k {
	case sim.ItemSpear, sim.ItemIvorySpear, sim.ItemRod, sim.ItemTorch:
		return false
	}
	return true
}

// HoldBlows stops every punch and jolt in view age frames in, for
// screenshots; a negative age lets them run again.
func (g *Game) HoldBlows(age int) { g.heldAge = age + 1 }

// since is how many frames ago frame at was, for a punch or a jolt.
func (g *Game) since(at int) int {
	if g.heldAge > 0 {
		return g.heldAge - 1
	}
	return g.frame - at
}

// punchPose is what p's arms do on this frame of a blow by hand, if one is
// under way.
func (g *Game) punchPose(p protocol.PlayerView) (figPose, bool) {
	w := g.swingWhat[p.ID]
	if !p.Acting || w.kind != swingHand || !fistHolds(p.Held) {
		return figPose{}, false
	}
	at, ok := g.swingAt[p.ID]
	age := g.since(at)
	if !ok || age < 0 || age >= punchEnd {
		return figPose{}, false
	}
	fp := figPose{spark: -1}
	arm := poseHalf
	if w.beast { // a punch
		switch {
		case age < punchWind:
			arm = poseWind
		case age >= punchHit && age < punchHold:
			arm, fp.lean = poseFull, 1
		}
		if s := age - punchHit; s >= 0 && s < sparkLife {
			fp.spark = s
		}
	} else if age >= punchHit && age < punchHold+1 { // a reach: no wind-up, no lean
		arm = poseFull
	}
	if w.cross && !p.Held.Valid() {
		fp.off = arm
	} else {
		fp.hand = arm
	}
	return fp, true
}

// drawFigure draws p standing at (x, y) like drawBody or drawArmored, with
// its arms as fp has them and whatever it holds in its fist.
func (g *Game) drawFigure(dst *ebiten.Image, p protocol.PlayerView, dir sim.Dir, frame int, x, y, bob float64, tint color.Color, armor sim.Tier, fp figPose) {
	l := p.Look
	if armor >= sim.TierDiamond && (l.Hair == sim.HairSpiky || l.Hair == sim.HairBun) {
		l.Hair = sim.HairShort // under the skull cap; see drawArmored
	}
	b := g.art.body(l)
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	hide := armNone
	switch {
	case fp.hand != poseRest && fp.off != poseRest && !sideways:
		hide = armBoth
	case fp.hand != poseRest:
		hide = armHand
	case fp.off != poseRest && !sideways:
		hide = armOff
	}
	if sideways && fp.hand == poseRest && (fp.off == poseHalf || fp.off == poseFull) {
		fp.hand = poseWind // the near fist is drawn back as the far one goes out
		hide = armHand
	}
	bare, arms := b.bareBody(hide), b.arms()
	dx, dy := dir.Delta()
	ux, uy := x+float64(dx)*fp.lean, y-bob+float64(dy)*fp.lean
	arm := func(side armSide, pose armPose) (armArt, bool) {
		if pose == poseRest {
			return armArt{}, false
		}
		a := arms[dir][side][pose]
		if a.none {
			return a, false
		}
		drawAt(dst, a.sleeve, ux, uy, 1, 1, 0, tint)
		drawAt(dst, a.fist, ux, uy, 1, 1, 0, nil)
		return a, true
	}
	held := func(a armArt) {
		if p.Held.Valid() {
			g.drawHeldAt(dst, p.Held, dir, ux+a.hx, uy+a.hy)
		}
	}
	var fist armArt // the one thrown out: where the spark goes
	note := func(a armArt, ok bool, pose armPose) {
		if ok && (fist.size == 0 || pose == poseFull) {
			fist = a
		}
	}

	if sideways { // the far arm, behind the body
		a, ok := arm(armOff, fp.off)
		note(a, ok, fp.off)
	}
	drawAt(dst, b.back[dir][frame], ux, uy, 1, 1, 0, nil)
	drawAt(dst, b.legs[dir][frame], x, y, 1, 1, 0, nil)
	drawAt(dst, bare.shirt[dir][frame], ux, uy, 1, 1, 0, tint)
	var a *armorArt
	if armor > sim.TierStone {
		a = g.art.armor(armor, l.Female)
		plate := a.body[dir][frame]
		if hide == armHand || hide == armBoth {
			plate = a.whole[dir][frame]
		}
		drawAt(dst, plate, ux, uy, 1, 1, 0, nil)
	}
	// Facing the viewer the arms hang from under the mantle and the head;
	// seen from behind or from the side they pass in front.
	hangs := func(p armPose) bool { return p == poseRest || p == poseHalf || p == poseFull }
	under := dir == sim.DirDown && hangs(fp.hand) && hangs(fp.off)
	var late func() // what is held, drawn last
	both := func() {
		if !sideways {
			a, ok := arm(armOff, fp.off)
			note(a, ok, fp.off)
		}
		h, ok := arm(armHand, fp.hand)
		note(h, ok, fp.hand)
		if ok && !under {
			if dir == sim.DirUp {
				// What is held is behind the fist.
				held(h)
				drawAt(dst, h.fist, ux, uy, 1, 1, 0, nil)
			} else {
				late = func() { held(h) }
			}
		}
	}
	if under {
		both()
	}
	drawAt(dst, bare.top[dir][frame], ux, uy, 1, 1, 0, nil)
	if !under {
		both()
	}
	if a != nil {
		drawAt(dst, a.helmet[dir][frame], ux, uy, 1, 1, 0, nil)
	}
	if under && fp.hand != poseRest {
		held(arms[dir][armHand][fp.hand])
	}
	if late != nil {
		late()
	}
	if fp.spark >= 0 && fist.size > 0 {
		reach := float64(fist.size)/2 + 2
		cx := ux + fist.hx + float64(dx)*reach
		cy := uy + fist.hy - 1 + float64(dy)*reach
		drawSpark(dst, math.Floor(cx), math.Floor(cy), fp.spark)
	}
}

var (
	sparkWhite  = hex(0xfffbe8)
	sparkYellow = hex(0xffd84a)
)

// drawSpark draws the little star of a landed punch round pixel (x, y), n
// frames after it landed: a flash, a star, then four flecks flying apart.
func drawSpark(dst *ebiten.Image, x, y float64, n int) {
	dot := func(dx, dy float64, c color.RGBA, a float64) {
		vector.FillRect(dst, float32(x+dx), float32(y+dy), 1, 1, premul(c, a), false)
	}
	arms := func(r float64, c color.RGBA, a float64) {
		dot(r, 0, c, a)
		dot(-r, 0, c, a)
		dot(0, r, c, a)
		dot(0, -r, c, a)
	}
	diag := func(r float64, c color.RGBA, a float64) {
		dot(r, r, c, a)
		dot(-r, r, c, a)
		dot(r, -r, c, a)
		dot(-r, -r, c, a)
	}
	switch n {
	case 0:
		dot(0, 0, sparkWhite, 1)
		arms(1, sparkWhite, 1)
	case 1:
		dot(0, 0, sparkWhite, 1)
		arms(1, sparkWhite, 1)
		arms(2, sparkYellow, 1)
		diag(1, sparkYellow, 1)
	case 2:
		arms(3, sparkYellow, 1)
		diag(2, sparkWhite, 1)
	default:
		arms(4, sparkYellow, 0.6)
		diag(3, sparkYellow, 0.45)
	}
}
