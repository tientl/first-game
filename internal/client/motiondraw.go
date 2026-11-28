package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Drawing the motions that are more than a frame of the standing figure:
// the fall when a player dies.

// drawDying draws p falling, if they died a moment ago: LPC's hurt row
// frame by frame (moDie). The blow in the way they faced (0); then, turned
// to the viewer as they crumple, the knees going (1), down on them (2),
// slumping back (3); going over backwards, the body as it will lie turned
// part way down in three steps (4: moTopple, drawn turned whole, not
// turned at draw); and down (5): the body the server left, drawn exactly as
// it is drawn (bodyart.go), where it is, so it takes over without a jump.
// The server lays a body on the nearest free tile (sim/grave.go), so the
// fall slides there as it goes; without one (an empty bag) it lies where
// they fell. It reports whether it drew anything.
func (g *Game) drawDying(dst drawTarget, p protocol.PlayerView, d drawable) bool {
	st := g.moveState(p.ID)
	age := float64(g.frame - st.deadAt)
	if st.deadAt == never || age < 0 || age >= moDie.span() {
		return false
	}
	fr := moDie.frameAt(age)
	b := g.art.bodyIn(p.Look, p.Wear)
	tint := playerColors[int(p.ID)%len(playerColors)]
	var armor *armorArt
	if t := shownArmor(p); t > sim.TierStone {
		armor = g.art.armor(t, p.Look.Female)
	}
	// Where the body ends up: its anchor, and its hips.
	gx, gy, grave, onGrave := g.graveOf(p, st, d)
	bx, by := gx+fallenHipX, gy+fallenHipY
	kx, ky := d.sx, d.sy-hipHeight+kneelDrop/HDDensity // the hips, kneeling
	slide := [6]float64{0, 0.04, 0.12, 0.25, 0.6, 1}[fr]
	ox, oy := (bx-kx)*slide, (by-ky)*slide
	switch fr {
	case 0:
		lo, up := moDie.keys(st.deadDir, 0)
		drawFig(dst, figDraw{b: b, armor: armor, up: up, lo: lo, tint: tint}, d.sx, d.sy)
	case 4:
		sub := min(2, int((age-moDie.startOf(4))/((moDie.startOf(5)-moDie.startOf(4))/3)))
		t := lerpf(0.45, 0.85, float64(sub)/2)
		s := laidSprite(laidKey{look: p.Look.Sanitized(), wear: p.Wear, tint: tint, m: moTopple, f: uint8(sub), dir: sim.DirDown, mirror: true})
		putSprite(dst, s, kx+(bx-kx)*t, ky+(by-ky)*t)
	case 5:
		dust(dst, bx-2, gy, age-moDie.startOf(5), uint32(p.ID))
		if onGrave { // the very drawing the body will be drawn as, on the camera's grid
			alignCamera(dst)
			drawAt(dst, graveSprite(grave), gx, gy, 1, 1, 0, nil)
			align(dst, d.sx, d.sy)
			return true
		}
		putSprite(dst, fallenBodySprite(p.Look, p.Wear, tint), gx, gy)
	default:
		lo, up := moDie.keys(sim.DirDown, fr)
		drawFig(dst, figDraw{b: b, armor: armor, up: up, lo: lo, tint: tint}, d.sx+ox, d.sy+oy)
	}
	return true
}

// Where the hips of the body as it lies (bodyart.go) are from its anchor,
// art pixels: layDown carried the standing figure's hips there.
const (
	fallenHipX = (fallenOX*HDDensity+hipY-fallenTop)/HDDensity - fallenAX
	fallenHipY = (fallenOY*HDDensity+27.0)/HDDensity - fallenAY
)

// graveOf is where the body p left as they fell is drawn (the anchor of its
// sprite, as drawEntity puts it), and the body; or, if they left none, where
// they fell.
func (g *Game) graveOf(p protocol.PlayerView, st *moState, d drawable) (x, y float64, e protocol.EntityView, ok bool) {
	look := p.Look.Sanitized()
	best := 3.0
	for _, en := range g.curr.Entities {
		if en.Kind != sim.EntityGrave || en.Var == sim.GraveBones {
			continue
		}
		l, seed := sim.GraveLook(en.Fuel)
		if l != look || seed != int(p.ID)&15 {
			continue
		}
		if dd := math.Hypot(float64(en.TX)+0.5-float64(st.deadX), float64(en.TY)+0.5-float64(st.deadY)); dd < best {
			best, e, ok = dd, en, true
		}
	}
	if !ok {
		return math.Round(d.sx), math.Round(d.sy), e, false
	}
	x, y = g.drawCam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
	return x, y, e, true
}

// fallenCacheBody holds the bodies drawn alone, as they lie, for a fall that
// leaves none behind.
var fallenCacheBody = map[fallenKey]sprite{}

// fallenBodySprite is the body of look l wearing wear in colour tint laid
// as the body the server leaves is (bodyart.go), without the bag or the
// stain: where someone with nothing on them fell.
func fallenBodySprite(l sim.Look, wear sim.Wear, tint color.RGBA) sprite {
	k := fallenKey{look: l.Sanitized(), wear: wear}
	for i, c := range playerColors {
		if c == tint {
			k.col = i
		}
	}
	if s, ok := fallenCacheBody[k]; ok {
		return s
	}
	c := newCanvas(fallenW*HDDensity, fallenH*HDDensity)
	fallenBody(c, k.look, wear, tint)
	s := spriteHD(c.image(), fallenAX, fallenAY)
	fallenCacheBody[k] = s
	return s
}

// drawLaidBody draws p lying on their back where they fell, as their body
// lies (bodyart.go).
func (g *Game) drawLaidBody(dst drawTarget, p protocol.PlayerView, d drawable) {
	st := g.moveState(p.ID)
	gx, gy, grave, onGrave := g.graveOf(p, st, d)
	if onGrave {
		alignCamera(dst)
		drawAt(dst, graveSprite(grave), gx, gy, 1, 1, 0, nil)
		align(dst, d.sx, d.sy)
		return
	}
	putSprite(dst, fallenBodySprite(p.Look, p.Wear, playerColors[int(p.ID)%len(playerColors)]), gx, gy)
}

// graveFalling reports whether body e is where a player is still falling:
// it waits till they are down.
func (g *Game) graveFalling(e protocol.EntityView) bool {
	for id, st := range g.moves {
		if st.deadAt == never || float64(g.frame-st.deadAt) >= moDie.span() {
			continue
		}
		_ = id
		if math.Abs(float64(e.TX)+0.5-float64(st.deadX)) < 2.5 && math.Abs(float64(e.TY)+0.5-float64(st.deadY)) < 2.5 {
			return true
		}
	}
	return false
}

// dust draws the puff of dust a body throws up as it lands at (x, y), t
// frames after.
func dust(dst drawTarget, x, y, t float64, seed uint32) {
	if t < 0 || t > 24 {
		return
	}
	u := t / 24
	for i := 0; i < 9; i++ {
		ang := math.Pi * (0.1 + 0.8*hashf(i, 1, seed))
		r := (4 + 13*hashf(i, 2, seed)) * (1 - (1-u)*(1-u))
		px := x - 3 + math.Cos(ang)*r*(1-2*float64(i%2))
		py := y - 1 - math.Sin(ang)*r*0.35 - 2*u
		fillRect(dst, float32(math.Round(px)), float32(math.Round(py)), 2, 1, premul(color.RGBA{196, 178, 150, 255}, 0.55*(1-u)), false)
	}
}
