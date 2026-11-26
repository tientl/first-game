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
		c := bakeUpper(b.look, k, b.limbs.heads)
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
		c, p := bakeLower(b.look, k)
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
	b     *bodyArt
	armor *armorArt // nil for none
	up    upKey
	lo    loKey
	tint  color.Color
	s     float64 // scale; 0 is 1
	rot   float64
	push  float64 // how far the body leans into a blow, in art pixels
	abed  bool    // under a blanket: the legs are not drawn
	// item draws what the hand holds, the hand at (hx, hy), its forearm
	// turned swing towards the facing.
	item func(dst drawTarget, hx, hy, swing float64)
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
	item := func() {
		if f.item != nil {
			hx, hy := handPoint(h, ux, uy, skew)
			f.item(dst, hx, hy, h.swing*side)
		}
	}
	if !h.over && heldBehind(dir) {
		item()
	}
	layer(upFar)
	layer(upBack)
	if !f.abed {
		put(lo.legs, nil, false)
	}
	layer(upShirt)
	if f.armor != nil {
		put(f.armor.whole[dir][0], nil, true)
	}
	layer(upArms)
	layer(upTop)
	if f.armor != nil {
		put(f.armor.helmet[dir][0], nil, true)
	}
	if !h.over && !heldBehind(dir) {
		item()
	}
	layer(upArmsOver)
	if h.over {
		item()
	}
	layer(upTopOver)
	return ux, uy, skew, u
}

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
	f := figDraw{b: g.art.body(l), up: up, lo: lo, tint: playerColors[int(p.ID)%len(playerColors)]}
	if armor > sim.TierStone {
		f.armor = g.art.armor(armor, l.Female)
	}
	carry := func(turn float64) func(drawTarget, float64, float64, float64) {
		return func(dst drawTarget, hx, hy, swing float64) {
			heldTurn = turn - 0.6*swing
			g.drawHeldAt(dst, p.Held, dir, hx, hy)
			heldTurn = 0
		}
	}
	gripped := func(draw func(d drawable)) func(drawTarget, float64, float64, float64) {
		return func(dst drawTarget, hx, hy, _ float64) {
			limbGrip.on, limbGrip.x, limbGrip.y = true, hx, hy
			draw(drawable{sx: x, sy: y})
			limbGrip.on = false
		}
	}
	f.item = carry(0)
	age := float64(g.frame - g.swingAt[p.ID])
	w := g.swingWhat[p.ID]
	fp, punching := g.punchPose(p)
	jolt, jolted := g.jolting(p.ID)
	fist := 0
	switch {
	case punching:
		pose, anim := fp.hand, upPunch
		if fp.off != poseRest {
			pose, anim, fist = fp.off, upPunchOff, 1
		}
		f.up = upKey{dir, anim, map[armPose]uint8{poseWind: 0, poseHalf: 3, poseFull: atkFrames - 1}[pose]}
		f.push = fp.lean
	case jolted && jolt < joltFling:
		f.up = upKey{dir: dir, anim: upFlingOff}
		if fistHolds(p.Held) {
			f.up.anim = upFling
		}
	case g.cleaving(p): // outlasts the working pose; see battleaxe.go
		ps := cleavePoseFor(dir)
		f.up = upKey{dir, upChop, atkFrame(cleaveAngle(ps, age), ps.back, ps.down)}
		f.item = gripped(func(d drawable) { g.drawCleave(dst, p, d, 0) })
	case p.Acting && isSword(p.Held):
		a := slashAngle(min(1, age/slashFrames))
		f.up = upKey{dir, upChop, atkFrame(a, slashFrom, slashTo)}
		f.push = clamp01((a - 0.4) / 1.2)
		f.item = gripped(func(d drawable) { g.drawSlash(dst, p, d, 0) })
	case p.Acting && (w.kind == swingAxe || w.kind == swingPick || w.kind == swingHoe):
		a := chopAngle(age)
		f.up = upKey{dir, upChop, atkFrame(a, chopRaised, chopImpact)}
		f.push = clamp01((a - 0.4) / 1.2)
		f.item = gripped(func(d drawable) { g.drawSwing(dst, p, d, 0, 0) })
	case p.Acting && (w.kind == swingThrust || w.kind == swingHand):
		out := reach(age)
		k := uint8(math.Round(lerpf(2, atkFrames-1, out)))
		turn := 0.0
		if w.kind == swingThrust {
			if age < 2 {
				k = 0 // drawn back first
			}
			f.push = out
			if dir == sim.DirLeft || dir == sim.DirRight {
				turn = 0.3 * out // levelled at the beast
			}
		}
		f.up = upKey{dir, upThrust, k}
		f.item = func(dst drawTarget, hx, hy, swing float64) {
			heldTurn = turn
			g.drawHeldAt(dst, p.Held, dir, hx, hy)
			heldTurn = 0
		}
	}
	ux, uy, skew, u := drawFig(dst, f, x, y)
	if punching && fp.spark >= 0 {
		h := u.hand[fist]
		hx, hy := handPoint(h, 0, 0, skew)
		dx, dy := dir.Delta()
		r := h.size/2 + 2
		drawSpark(dst, ux+math.Floor(hx+float64(dx)*r), uy+math.Floor(hy+float64(dy)*r), fp.spark) // whole art pixels from the figure's point; see align
	}
}
