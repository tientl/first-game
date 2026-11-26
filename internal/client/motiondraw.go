package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Drawing the motions that are more than a frame of the standing figure:
// the fall when a player dies.

// drawDying draws p falling, if they died a moment ago: LPC's hurt row,
// frame by frame (moDie): the blow, the knees going, a kneel, a slump, over
// (the slump turned), and down on the ground, laid as the body lies
// (bodyart.go), till the body the server left takes over. It reports
// whether it drew anything.
func (g *Game) drawDying(dst drawTarget, p protocol.PlayerView, d drawable) bool {
	st := g.moveState(p.ID)
	age := float64(g.frame - st.deadAt)
	if st.deadAt == never || age < 0 || age >= moDie.span() {
		return false
	}
	fr := moDie.frameAt(age)
	dir := st.deadDir
	b := g.art.bodyIn(p.Look, p.Wear)
	tint := playerColors[int(p.ID)%len(playerColors)]
	var armor *armorArt
	if p.Armor > sim.TierStone {
		armor = g.art.armor(p.Armor, p.Look.Female)
	}
	x, y := d.sx, d.sy
	switch fr {
	case 4: // going over: the slump, turned about the knees towards the back
		lo, up := moDie.keys(dir, 3)
		way := -1.0
		if dir == sim.DirLeft {
			way = 1
		}
		drawFig(dst, figDraw{b: b, armor: armor, up: up, lo: lo, tint: tint, rot: way * 0.75}, x, y)
	case 5: // down, on the back, as the body lies
		dust(dst, x, y, age-moDie.startOf(5), uint32(p.ID))
		g.drawLaidBody(dst, p, d)
	default:
		lo, up := moDie.keys(dir, fr)
		drawFig(dst, figDraw{b: b, armor: armor, up: up, lo: lo, tint: tint}, x, y)
	}
	return true
}

// drawLaidBody draws p lying on their back where they fell, as their body
// lies (bodyart.go): head to the left, an arm thrown up past it.
func (g *Game) drawLaidBody(dst drawTarget, p protocol.PlayerView, d drawable) {
	var armor *armorArt
	if p.Armor > sim.TierStone {
		armor = g.art.armor(p.Armor, p.Look.Female)
	}
	drawFig(dst, figDraw{b: g.art.bodyIn(p.Look, p.Wear), armor: armor, up: upKey{dir: sim.DirDown, anim: upFallen},
		tint: playerColors[int(p.ID)%len(playerColors)], rot: -math.Pi / 2}, d.sx+14, d.sy-3)
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
