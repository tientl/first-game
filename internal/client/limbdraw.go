package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Showing the jointed figure of limbs.go: its frames, drawn the first time
// they are shown and kept with the look's other art; how the halves are put
// together; and which frame a player is on (the stride from how far they
// have gone, a blow from how long ago it began).

// handInfo is where a hand of a frame is, in art pixels from the figure's
// point, and how its forearm is turned.
type handInfo struct {
	x, y, swing float64
	over        bool    // drawn over everything
	size        float64 // the fist's, in art pixels
}

// upperArt is the upper half of a frame as sprites.
type upperArt struct {
	layer [numUpLayers]sprite
	hand  [2]handInfo // the hand that holds things, and the other
}

// lowerArt is the legs of a frame, how far they lift the body and how far
// it leans.
type lowerArt struct {
	legs      sprite
	bob, lean float64
}

// limbArt is the frames of one look drawn so far.
type limbArt struct {
	up    map[upKey]*upperArt
	lo    map[loKey]*lowerArt
	heads headCache
}

func (la *limbArt) init() {
	if la.up == nil {
		la.up, la.lo, la.heads = make(map[upKey]*upperArt), make(map[loKey]*lowerArt), make(headCache)
	}
}

// upper returns the upper half of b for k, drawing it on first use. Like
// art.body it must be called inside the game loop.
func (b *bodyArt) upper(k upKey) *upperArt {
	b.limbs.init()
	if u, ok := b.limbs.up[k]; ok {
		return u
	}
	u := &upperArt{}
	if k.dir == sim.DirLeft {
		k.dir = sim.DirRight
		r := b.upper(k)
		k.dir = sim.DirLeft
		for i, s := range r.layer {
			if s.img != nil {
				u.layer[i] = mirrorSprite(s)
			}
		}
		u.hand = r.hand
		for i := range u.hand {
			u.hand[i].x, u.hand[i].swing = -u.hand[i].x, -u.hand[i].swing
		}
	} else {
		c := bakeUpper(b.look, b.wear, k, b.limbs.heads)
		for i, l := range c.layer {
			u.layer[i] = limbSprite(l)
		}
		for i, a := range c.arm {
			u.hand[i] = handInfo{
				x:     a.hx/HDDensity - 9,
				y:     a.hy/HDDensity - limbFeet,
				swing: a.swing,
				over:  a.layer == armOver,
				size:  a.size / HDDensity,
			}
		}
	}
	b.limbs.up[k] = u
	return u
}

// lower returns the legs of b for k, drawing them on first use.
func (b *bodyArt) lower(k loKey) *lowerArt {
	b.limbs.init()
	if l, ok := b.limbs.lo[k]; ok {
		return l
	}
	l := &lowerArt{}
	if k.dir == sim.DirLeft {
		k.dir = sim.DirRight
		*l = *b.lower(k)
		k.dir = sim.DirLeft
		l.legs = mirrorSprite(l.legs)
	} else {
		c, p := bakeLower(b.look, b.wear, k)
		l.legs, l.bob, l.lean = limbSprite(c), -p.drop/HDDensity, p.lean
		if k.dir != sim.DirRight {
			l.lean = 0
		}
	}
	b.limbs.lo[k] = l
	return l
}

// figDraw is a jointed figure to draw: its halves, what it wears and holds.
type figDraw struct {
	b        *bodyArt
	armor    *armorArt // nil for none
	up       upKey
	lo       loKey
	tint     color.Color
	s        float64 // scale; 0 is 1
	rot      float64
	push     float64 // how far the body leans into a blow, in art pixels
	abed     bool    // under a blanket: the legs are not drawn
	legsOnly bool    // only the legs: a rider's far leg, drawn behind the mount
	// item draws what the hand holds, the hand at (hx, hy), its forearm
	// turned swing towards the facing.
	item func(dst drawTarget, hx, hy, swing float64)
	// pair, if set, is drawn with item: what is held in both hands (a bowl),
	// or in the other one, the hands at (hx, hy) and (ox, oy).
	pair func(dst drawTarget, hx, hy, ox, oy float64)
	// tool is set when item is a tool gripped in the fist: raised beside
	// the head, seen from the side, it goes behind the head with the hand.
	tool bool
	// offHand puts item in the other hand: the hand that holds things is
	// busy (reaching up for fruit).
	offHand bool
	// itemBack draws item behind the body: a tool carried along the side of
	// a figure seen from the front or the back (carriedBehind).
	itemBack bool
	// shield draws the shield (shieldart.go) at each of the layers it may
	// go in: behind the figure, over the trunk under the near arm, or over
	// everything; it draws only at the one it belongs in.
	shield func(dst drawTarget, at shieldLayer, ux, uy, skew float64, u *upperArt)
}

// playerTop is how far above a player's feet their name goes, in art
// pixels: just over the head.
const playerTop = 35

// hipHeight is how far above the feet the body leans from, in art pixels.
const hipHeight = (limbGround - hipY) / HDDensity

// drawFig draws f standing at (x, y), the feet, and returns where the upper
// half's point ended up and how it leans (see handAt).
func drawFig(dst drawTarget, f figDraw, x, y float64) (ux, uy, skew float64, u *upperArt) {
	s := f.s
	if s == 0 {
		s = 1
	}
	dir := f.up.dir
	f.lo.dir = dir
	if f.lo.anim == loSit {
		f.up.frame |= upSit
	}
	u, lo := f.b.upper(f.up), f.b.lower(f.lo)
	side := facingSide(dir)
	dx, dy := dir.Delta()
	ux, uy = x+float64(dx)*f.push*s, y+(float64(dy)*f.push-lo.bob)*s
	if dir == sim.DirLeft || dir == sim.DirRight {
		skew = math.Tan(lo.lean+f.push*0.04) * side
	}
	if f.rot != 0 { // lying down: the offsets turn with the figure
		ux, uy = x, y
	}
	put := func(sp sprite, tint color.Color, upper bool) {
		if sp.img == nil {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-sp.ax, -sp.ay)
		px, py := x, y
		if upper {
			px, py = ux, uy
			if skew != 0 {
				op.GeoM.Translate(0, hipHeight)
				op.GeoM.Skew(-math.Atan(skew), 0)
				op.GeoM.Translate(0, -hipHeight)
			}
		}
		op.GeoM.Scale(s, s)
		if f.rot != 0 {
			if upper { // the hips' drop goes with the figure as it turns
				op.GeoM.Translate(0, -lo.bob*s)
			}
			op.GeoM.Rotate(f.rot)
		}
		op.GeoM.Translate(px, py)
		if tint != nil {
			op.ColorScale.ScaleWithColor(tint)
		}
		drawSprite(dst, sp, op)
	}
	layer := func(i int) {
		var tint color.Color
		if upperTinted(i) {
			tint = f.tint
		}
		put(u.layer[i], tint, true)
	}
	h := u.hand[0]
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	back := heldBehind(dir) || f.itemBack // drawn before the body
	if f.offHand {
		h = u.hand[1]
		back = back || sideways // the far hand
	}
	// A hand raised sideways is behind the head (armGeo), and so is the tool
	// in it, from the grip up; from behind, a tool raised over the head is
	// beyond it.
	behindHead := f.tool && ((sideways && !h.over && h.y < toolBehindHeadY && !f.offHand) || (dir == sim.DirUp && h.over))
	item := func() {
		if f.item != nil {
			hx, hy := handPoint(h, ux, uy, skew)
			f.item(dst, hx, hy, h.swing*side)
		}
		if f.pair != nil {
			hx, hy := handPoint(u.hand[0], ux, uy, skew)
			ox, oy := handPoint(u.hand[1], ux, uy, skew)
			f.pair(dst, hx, hy, ox, oy)
		}
	}
	if !h.over && back {
		item()
	}
	shieldAt := func(at shieldLayer) {
		if f.shield != nil {
			f.shield(dst, at, ux, uy, skew, u)
		}
	}
	shieldAt(shieldBehind)
	layer(upFar)
	layer(upBack)
	if f.legsOnly {
		put(lo.legs, nil, false)
		return ux, uy, skew, u
	}
	if !f.abed {
		put(lo.legs, nil, false)
	}
	layer(upBody)
	layer(upShirt)
	if f.armor != nil {
		put(f.armor.plate(dir, f.up), nil, true)
	}
	shieldAt(shieldOnBody)
	layer(upArms)
	if behindHead {
		item()
	}
	layer(upTop)
	if f.armor != nil {
		put(f.armor.helm(dir, f.up), nil, true)
	}
	if !h.over && !back && !behindHead {
		item()
	}
	layer(upArmsOver)
	if h.over && !behindHead {
		item()
	}
	layer(upTopOver)
	shieldAt(shieldOver)
	return ux, uy, skew, u
}

// carried reports whether figure f, its legs lo, carries what it holds
// (drawCarried): standing, idling, walking or running, not astride a
// mount, nor in a blow or a deed.
func carried(f figDraw, lo loKey) bool {
	if lo.anim == loSit || f.offHand {
		return false
	}
	switch f.up.anim {
	case upStand, upWalk, upRun, upWalkHold, upRunHold, upMotion + upAnim(moIdle):
		return f.up.frame&upSit == 0
	}
	return false
}

// toolBehindHeadY is how high a hand seen from the side has to be, art
// pixels above the feet, for the tool in it to go behind the head: above
// the shoulders, where the chop and the slash are wound up.
const toolBehindHeadY = -21.5

// handPoint is where hand h of a figure whose upper half is at (ux, uy),
// leaning by skew, is on the screen.
func handPoint(h handInfo, ux, uy, skew float64) (float64, float64) {
	return ux + h.x - skew*(h.y+hipHeight), uy + h.y
}

// drawBodyJ is drawBody for the jointed figure: frame 0 stands, the others
// are a walking stride.
func drawBodyJ(dst drawTarget, b *bodyArt, dir sim.Dir, frame int, x, y, s float64, tint color.Color) {
	f := figDraw{b: b, up: upKey{dir: dir}, tint: tint, s: s}
	if frame != 0 {
		f.up = upKey{dir, upWalk, uint8(frame % upFrames)}
		f.lo = loKey{dir, loWalk, uint8(frame * loFrames / upFrames % loFrames)}
	}
	drawFig(dst, f, x, y)
}

// --- which frame -------------------------------------------------------------

// gaitState is how far through a stride a player is.
type gaitState struct {
	x, y  float64 // where they were last drawn, in tiles
	frame int     // and on which frame
	ph    float64 // through the stride, 0 to 1
	speed float64 // art pixels of ground a frame, smoothed
	run   bool
}

// Faster than runAbove art pixels a frame a player runs, until slower than
// walkBelow: on open ground they run, through snow and marsh they walk.
const (
	runAbove  = 1.2
	walkBelow = 1.0
)

// stride moves p's gait on to where they are now, (wx, wy) in tiles, and
// returns the frames of their legs and arms. The stride goes by the ground
// covered, so a foot that is down stays where it is.
func (g *Game) stride(p protocol.PlayerView, dir sim.Dir, wx, wy float64, moving bool) (loKey, upKey) {
	if g.gaits == nil {
		g.gaits = make(map[sim.PlayerID]*gaitState)
	}
	st := g.gaits[p.ID]
	if st == nil {
		st = &gaitState{x: wx, y: wy, frame: g.frame, speed: sim.PlayerSpeed * TileW * sim.TicksPerSecond / 60}
		g.gaits[p.ID] = st
	}
	if g.frame-st.frame > 2 { // out of sight for a while
		st.x, st.y = wx, wy
	}
	if g.frame != st.frame {
		dx, dy := (wx-st.x)*TileW, (wy-st.y)*TileW
		fx, fy := dir.Delta()
		dist := math.Hypot(dx, dy)
		along := dx*float64(fx) + dy*float64(fy)
		if math.Abs(along) >= 0.5*dist {
			dist = along // forwards, or backing away
		}
		if n := g.frame - st.frame; n > 0 && dist != 0 {
			st.speed += (math.Abs(dist)/float64(n) - st.speed) * 0.25
		}
		switch {
		case st.speed > runAbove:
			st.run = true
		case st.speed < walkBelow:
			st.run = false
		}
		gd := stepGait
		if st.run {
			gd = runGait
		}
		st.ph += dist / gd.cycle
		st.ph -= math.Floor(st.ph)
		st.x, st.y, st.frame = wx, wy, g.frame
	}
	if !moving {
		st.ph = 0
		return loKey{dir: dir}, upKey{dir: dir}
	}
	la, ua := loWalk, upWalk
	if st.run {
		la, ua = loRun, upRun
	}
	return loKey{dir, la, uint8(int(st.ph*loFrames) % loFrames)}, upKey{dir, ua, uint8(int(st.ph*upFrames) % upFrames)}
}

// limbGrip is where the hand is while a tool is swung, for the swings of
// swing.go, slash.go and battleaxe.go: they turn the tool about it.
var limbGrip struct {
	on   bool
	x, y float64
}

// heldTurn turns what drawHeldAt draws, in radians towards the facing: a
// carried tool goes with the forearm.
var heldTurn float64

// atkFrame is the frame of a blow whose tool is at angle a between raised
// and landed.
func atkFrame(a, raised, landed float64) uint8 {
	return uint8(math.Round(clamp01((a-raised)/(landed-raised)) * (atkFrames - 1)))
}

// drawFigureJ draws p standing at (x, y) with legs lo and arms up, but for
// what a blow or a start does to the arms, and with what it holds.
func (g *Game) drawFigureJ(dst drawTarget, p protocol.PlayerView, dir sim.Dir, lo loKey, up upKey, x, y float64, armor sim.Tier) {
	l := p.Look
	if armor >= sim.TierDiamond && (l.Hair == sim.HairSpiky || l.Hair == sim.HairBun) {
		l.Hair = sim.HairShort // under the skull cap; see drawArmored
	}
	f := figDraw{b: g.art.bodyIn(l, p.Wear), up: up, lo: lo, tint: playerColors[int(p.ID)%len(playerColors)]}
	if lo.anim != loStand && p.Held.Valid() { // striding with something in hand: that arm holds it
		switch up.anim {
		case upWalk:
			f.up.anim = upWalkHold
		case upRun:
			f.up.anim = upRunHold
		}
	}
	if armor > sim.TierStone {
		f.armor = g.art.armor(armor, l.Female)
	}
	follow := 0.6 // how far a carried tool turns with the forearm
	if up.anim == upRun {
		follow = 0.35 // the arms pump: held steadier
	}
	carry := func(turn float64) func(drawTarget, float64, float64, float64) {
		return func(dst drawTarget, hx, hy, swing float64) {
			heldTurn = turn - follow*swing
			g.drawHeldAt(dst, p.Held, dir, hx, hy)
			heldTurn = 0
		}
	}
	gripped := func(draw func(d drawable)) func(drawTarget, float64, float64, float64) {
		f.tool = true
		return func(dst drawTarget, hx, hy, _ float64) {
			limbGrip.on, limbGrip.x, limbGrip.y = true, hx, hy
			draw(drawable{sx: x, sy: y})
			limbGrip.on = false
		}
	}
	f.item = carry(0)
	f.tool = heldDefOf(p.Held) != nil
	age := float64(g.frame - g.swingAt[p.ID])
	w := g.swingWhat[p.ID]
	fp, punching := g.punchPose(p)
	jolt, jolted := g.jolting(p.ID)
	fist := 0
	// A blow or a motion sets the arms; standing, the legs too (a stance,
	// a lunge, a crouch). Striding, the legs go on striding.
	still := lo.anim == loStand
	pose := func(m motionID, fr int) {
		l2, u2 := m.keys(dir, fr)
		f.up = u2
		if still {
			f.lo = l2
		}
	}
	gm, gf, gathering := g.gathering(p)
	switch {
	case jolted: // struck: LPC's hurt, a flinch; the flash and the hop are drawJolted's
		pose(moHurt, moHurt.frameAt(float64(jolt)))
	case p.Guard && p.Mount == 0 && !p.Swimming: // behind a raised shield; see shieldart.go
		pose(moGuard, moGuard.frameAt(float64(g.frame)))
	case gathering:
		pose(gm, gf)
		if gm == moReachUp && p.Held.Valid() { // the free hand takes it while the other reaches
			f.offHand = true
			f.item = func(dst drawTarget, hx, hy, swing float64) {
				heldTurn = -0.6 * swing
				if dir == sim.DirDown || dir == sim.DirUp {
					heldFlip = -1 // in the hand on the left of the screen
				}
				g.drawHeldAt(dst, p.Held, dir, hx, hy)
				heldTurn, heldFlip = 0, 1
			}
		}
	case punching:
		pose, anim := fp.hand, upPunch
		if fp.off != poseRest {
			pose, anim, fist = fp.off, upPunchOff, 1
		}
		f.up = upKey{dir, anim, map[armPose]uint8{poseWind: 0, poseHalf: 3, poseFull: atkFrames - 1}[pose]}
		f.push = fp.lean
	case g.cleaving(p): // outlasts the working pose; see battleaxe.go
		ps := cleavePoseFor(dir)
		pose(moChop, int(atkFrame(cleaveAngle(ps, age), ps.back, ps.down)))
		f.item = gripped(func(d drawable) { g.drawCleave(dst, p, d, 0) })
	case p.Acting && isSword(p.Held):
		k := min(1, age/slashFrames)
		if g.backhand(p) {
			pose(moBackslash, int(atkFrame(backslashAngle(k), backslashFrom, backslashTo)))
		} else {
			a := slashAngle(k)
			pose(moSlash, int(atkFrame(a, slashFrom, slashTo)))
			f.push = 0.5 * clamp01((a-0.4)/1.2)
		}
		f.item = gripped(func(d drawable) { g.drawSlash(dst, p, d, 0) })
	case p.Acting && (w.kind == swingAxe || w.kind == swingPick || w.kind == swingHoe):
		a := chopAngle(age)
		pose(moChop, int(atkFrame(a, chopRaised, chopImpact)))
		f.push = 0.5 * clamp01((a-0.4)/1.2)
		f.item = gripped(func(d drawable) { g.drawSwing(dst, p, d, 0, 0) })
	case p.Acting && w.kind == swingThrust:
		// LPC's thrust on the server's clock: drawn back for two frames, out
		// in the third (the blow lands), held, and back.
		fr := thrustFrame(age)
		pose(moThrust, fr)
		out := clamp01(reach(age))
		turn := 0.0
		if dir == sim.DirLeft || dir == sim.DirRight {
			turn = 0.3 * out // levelled at the beast
		}
		f.push = 0.5 * out
		f.item = func(dst drawTarget, hx, hy, swing float64) {
			heldTurn = turn
			g.drawHeldAt(dst, p.Held, dir, hx, hy)
			heldTurn = 0
		}
	case p.Acting && w.kind == swingHand:
		out := reach(age)
		f.up = upKey{dir, upThrust, uint8(math.Round(lerpf(2, atkFrames-1, out)))}
	case p.Mount == 0:
		if lvl, ok := g.rising(p); ok && !still { // up from the ground as they step off; see motionpick.go
			f.lo, f.up = moRise.keys(dir, lvl*loFrames+int(lo.frame)%loFrames)
			break
		}
		if fr, dx, dy, ok := g.dismounting(p); ok { // down off a beast; see ridehd.go
			f.lo, f.up = moDismount.keys(dir, fr)
			x, y = x+dx, y+dy
			break
		}
		if m, fr, ok := g.motionOf(p, !still); ok {
			l2, u2 := m.keys(dir, fr)
			f.up = u2
			if still || m == moStandUp {
				f.lo = l2
			}
			g.deedItems(dst, &f, p, m, fr)
		}
	}
	// Walking, running or standing, a tool is carried as it would be: see
	// drawCarried. A torch is held up, its arm raised whatever the other does.
	if carried(f, lo) && heldDefOf(p.Held) != nil {
		if p.Held == sim.ItemTorch {
			f.up.frame |= upTorch
		}
		f.tool, f.itemBack = false, carriedBehind(p.Held, dir)
		f.item = func(dst drawTarget, hx, hy, swing float64) {
			g.drawCarried(dst, p.Held, dir, hx, hy, swing)
		}
	}
	if p.Shield {
		f.shield = shieldDrawer(p, dir) // on the back, or raised; see shieldart.go
	}
	_, onWorld := dst.(*worldDst) // where a torch's flame and a rod's tip go: see heldhd.go
	heldBy.on, heldBy.id = onWorld, p.ID
	ux, uy, skew, u := drawFig(dst, f, x, y)
	heldBy.on = false
	if punching && fp.spark >= 0 {
		h := u.hand[fist]
		hx, hy := handPoint(h, 0, 0, skew)
		dx, dy := dir.Delta()
		r := h.size/2 + 2
		drawSpark(dst, ux+math.Floor(hx+float64(dx)*r), uy+math.Floor(hy+float64(dy)*r), fp.spark) // whole art pixels from the figure's point; see align
	}
}
