package client

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Drawing the open sea: boats with their riders and sails, swimmers, and
// the sea monsters, under the water and out of it. The art is in seaart.go.

// boatLeft reports whether a boat heading h is drawn facing left.
func boatLeft(h float64) bool { return math.Cos(h) < -0.05 }

// lerpBoat returns where to draw b this frame.
func (g *Game) lerpBoat(b protocol.BoatView, alpha float64) (x, y float64) {
	x, y = float64(b.X), float64(b.Y)
	if q, ok := g.prevBoats[b.ID]; ok && math.Abs(float64(q.X)-x) < teleportTiles && math.Abs(float64(q.Y)-y) < teleportTiles {
		x, y = float64(q.X)+(x-float64(q.X))*alpha, float64(q.Y)+(y-float64(q.Y))*alpha
	}
	return x, y
}

// boatSprites are the hull's back and front and the seat offset for b.
func (g *Game) boatSprites(b protocol.BoatView) (back, front sprite, seatX, seatY float64) {
	a := &g.art.sea
	back, front, seatX, seatY = a.raftBack, a.raftFront, raftSeatX, raftSeatY
	if b.Kind == sim.BoatSail {
		back, front, seatX, seatY = a.sailBack, a.sailFront, sailSeatX, sailSeatY
	}
	if boatLeft(float64(b.Heading)) {
		back, front, seatX = flipped(back), flipped(front), -seatX
	}
	return back, front, seatX, seatY
}

// drawBoat draws b at (sx, sy) with its rider and sail.
func (g *Game) drawBoat(dst drawTarget, b protocol.BoatView, sx, sy float64) {
	t := float64(g.frame) + float64(b.ID)*13
	bob := math.Round(math.Sin(t*0.06) * 1)
	if b.Held {
		sx += math.Round(math.Sin(t * 0.9)) // shaking in the squid's arms
	}
	back, front, seatX, seatY := g.boatSprites(b)
	left := boatLeft(float64(b.Heading))

	// A wake of foam behind a boat under way.
	if b.Speed > 0.02 && envHD {
		drawBoatWake(dst, sx, sy, float64(b.Heading), float64(b.Speed), t/60, b.Kind != sim.BoatSail) // see boathd.go
	} else if b.Speed > 0.02 {
		hx, hy := math.Cos(float64(b.Heading)), math.Sin(float64(b.Heading))
		n := int(min(10, float64(b.Speed)*60))
		for k := 1; k <= n; k++ {
			ph := math.Mod(t*0.05+float64(k)*0.37, 1)
			d := float64(k)*3 + ph*2
			wx := sx - hx*d*TileW/TileH*0.9 + math.Sin(float64(k)*2.3)*float64(k)*0.4
			wy := sy - hy*d*0.7 + 2
			a := uint8(180 * (1 - float64(k)/float64(n+1)))
			fillRect(dst, float32(math.Round(wx)), float32(math.Round(wy)), 2, 1, color.RGBA{a, a, a, a}, false)
		}
	}
	drawAt(dst, back, sx, sy+bob, 1, 1, 0, nil)
	if b.Kind == sim.BoatSail {
		s := g.art.sea.sail[g.sailFrame(b)]
		mx := float64(mastX)
		if left {
			s, mx = flipped(s), -mx
		}
		drawAt(dst, s, sx+mx, sy+bob+mastY, 1, 1, 0, nil)
	}
	for i, p := range g.curr.Players {
		if p.Boat == b.ID && !p.Dead {
			d := drawable{sx: sx + seatX, sy: sy + bob + seatY, id: uint32(p.ID), isPlayer: true, idx: i}
			g.drawPlayer(dst, p, d)
		}
	}
	drawAt(dst, front, sx, sy+bob, 1, 1, 0, nil)
}

// sailFrame is how full the sail of b is with the wind: 0 slack to 2 full.
func (g *Game) sailFrame(b protocol.BoatView) int {
	if b.Speed < 0.01 {
		return 0
	}
	f := sim.SailFactor(float64(b.Heading), float64(g.curr.WindDir), float64(g.curr.Wind))
	switch {
	case f > 0.7:
		return 2
	case f > 0.45:
		return 1
	}
	return 0
}

// drawSwimmer draws p in the water at d: the head and shoulders above the
// surface, rings spreading round them.
func (g *Game) drawSwimmer(dst drawTarget, p protocol.PlayerView, d drawable) {
	if playerHD && playerJointed { // strokes, and the water round them; see swimhd.go
		g.drawSwimmerHD(dst, p, d)
		return
	}
	b := g.art.bodyIn(p.Look, p.Wear)
	dir := p.Dir & 3
	tint := playerColors[int(p.ID)%len(playerColors)]
	bob := math.Round(math.Sin(float64(g.frame)*0.12+float64(p.ID)) * 1)
	// Sunk to the chest: sink art pixels of the figure are under the water,
	// and its feet are at (ax, ay) of the strip it is drawn into.
	sink, ax, ay := 14.0, 24.0, 40.0
	if playerHD && playerJointed {
		sink, ax, ay = 21, 32, 56
	}
	y := d.sy + sink + bob
	// Only what is above the water: draw into a strip and cut it off.
	const above = 22
	if g.swimBuf == nil {
		g.swimBuf = newFineBuf(64, 64) // at HD density: see hd.go
	}
	g.swimBuf.img.Clear()
	if playerHD && playerJointed { // paddling; see limbdraw.go
		drawFig(g.swimBuf, figDraw{b: b, up: upKey{dir, upSwim, uint8(g.frame / 12 % 4)}, tint: tint}, ax, ay)
	} else {
		drawBody(g.swimBuf, b, dir, 0, ax, ay, 1, 0, tint)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(d.sx-ax, y-ay)
	const k = HDDensity
	top := int(ay - above - sink)
	sub := g.swimBuf.img.SubImage(imageRect(0, top*k, 64*k, int(ay-sink)*k)).(*ebiten.Image)
	op.GeoM.Translate(0, float64(top))
	drawImageD(dst, sub, op, HDDensity)
	y -= sink - 13 // the rings lie on the water
	for k := range 2 {
		ph := math.Mod(float64(g.frame)/70+float64(k)*0.5, 1)
		r, a := 5+ph*7, uint8(170*(1-ph))
		for s := range 20 {
			ang := 2 * math.Pi * float64(s) / 20
			fillRect(dst, float32(math.Round(d.sx+r*math.Cos(ang))), float32(math.Round(y-13+r*0.4*math.Sin(ang))), 1, 1, color.RGBA{a, a, a, a}, false)
		}
	}
}

// --- sea monsters ---------------------------------------------------------

// seaMonster reports whether k is a monster of the open sea.
func seaMonster(k sim.CreatureKind) bool { return k == sim.CreatureMosa || k == sim.CreatureKraken }

// seaSprite is creatureSprite for the sea monsters out of the water: frames
// of the runs in seahd.go.
func (g *Game) seaSprite(c protocol.CreatureView) (sprite, float64, bool) {
	t := g.frame + int(c.ID)*7
	if bossAgeShot >= 0 {
		t = bossAgeShot
	}
	switch c.Kind {
	case sim.CreatureMosa:
		hdWarm(mosaRise, mosaBite, mosaBack, mosaShadow)
		if c.State == sim.CreatureStrike { // thrown back, then the bite
			age, _ := g.stateAge(c)
			return mosaBite.sprite(megaFrameAt(mosaBiteAges[:], age), c.FaceLeft), 0, true
		}
		g.stateAge(c)
		return mosaRise.sprite(t/6, c.FaceLeft), 0, true
	case sim.CreatureKraken:
		hdWarm(krakenHead, krakenArm, krakenShadow)
		return krakenHead.sprite(t/7, c.FaceLeft), 0, true
	}
	return sprite{}, 0, false
}

// drawSeaShadow draws a sea monster swimming under the surface: its dark
// shape, and for the sea lizard now and then its back rolling through the
// water.
func (g *Game) drawSeaShadow(dst drawTarget, cam camera, c protocol.CreatureView, alpha float64) {
	x, y, moving := g.lerpCreature(c, alpha)
	sx, sy := cam.toScreenFine(x, y)
	t := g.frame + int(c.ID)*7
	if bossAgeShot >= 0 {
		t = bossAgeShot
	}
	per := 9
	if moving {
		per = 5
	}
	s := mosaShadow.sprite(t/per, c.FaceLeft)
	if c.Kind == sim.CreatureKraken {
		s = krakenShadow.sprite(t/(per+2), c.FaceLeft)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-s.ax, sy-s.ay)
	op.ColorScale.ScaleAlpha(0.5)
	drawSprite(dst, s, op)
	const roll = 60 // frames its back takes to roll through
	if u := (g.frame + int(c.ID)*37) % 170; c.Kind == sim.CreatureMosa && moving && u < roll {
		drawAt(dst, mosaBack.at(float64(u)/roll, c.FaceLeft), sx, sy+1, 1, 1, 0, nil)
	}
}

// drawKraken draws a giant squid holding a boat: its arms rising round it,
// each swaying in its own time.
func (g *Game) drawKrakenArms(dst drawTarget, c protocol.CreatureView, d drawable) {
	t := g.frame
	if bossAgeShot >= 0 {
		t = bossAgeShot
	}
	for k := range 5 {
		ang := float64(k)*2*math.Pi/5 + 0.4
		ax := d.sx + math.Cos(ang)*30
		ay := d.sy + math.Sin(ang)*11
		arm := krakenArm.sprite(t/5+k*5, math.Cos(ang) < 0)
		drawAt(dst, arm, ax, ay, 1, 1, 0, nil)
	}
}

// --- HUD -------------------------------------------------------------------------

// myBoat is the boat this player is aboard.
func (g *Game) myBoat() (protocol.BoatView, bool) {
	if id := g.curr.Self.Boat; id != 0 {
		for _, b := range g.curr.Boats {
			if b.ID == id {
				return b, true
			}
		}
	}
	return protocol.BoatView{}, false
}

// boatItem is the item a boat of kind k is made as, for its picture.
func boatItem(k sim.BoatKind) sim.ItemKind {
	if k == sim.BoatSail {
		return sim.ItemSailboat
	}
	return sim.ItemRaft
}

// seaVitals are the bars of the sea, under health and hunger (hud.go): the
// boat's while aboard, or a swimmer's strength.
func (g *Game) seaVitals() []vital {
	self := g.curr.Self
	if self.Dead {
		return nil
	}
	if b, ok := g.myBoat(); ok {
		return []vital{{id: vitalBoat, item: boatItem(b.Kind), cur: float64(max32i(b.HP)), max: float64(b.Kind.MaxHP()), fill: color.RGBA{180, 130, 70, 255}}}
	}
	swimming := false
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			swimming = p.Swimming
		}
	}
	if swimming || self.Breath > 0 && self.Breath < 0.999 {
		col := color.RGBA{110, 190, 240, 255}
		if self.Breath <= 0 {
			col = colBad
		}
		return []vital{{id: vitalBreath, cur: float64(max32(0, self.Breath)), max: 1, fill: col}}
	}
	return nil
}

// seaStatuses are the chips of the sea: the boat in a giant squid's arms,
// and the wind while sailing.
func (g *Game) seaStatuses() []status {
	b, ok := g.myBoat()
	if !ok || g.curr.Self.Dead {
		return nil
	}
	var out []status
	if b.Held {
		out = append(out, status{statusHeld, "Bị mực giữ", "Bị mực khổng lồ giữ chặt! Đánh nó để thoát", colBad})
	}
	if b.Kind.Sails() {
		word := "Gió nhẹ"
		switch {
		case g.curr.Wind > 0.66:
			word = "Gió mạnh"
		case g.curr.Wind > 0.33:
			word = "Gió vừa"
		}
		out = append(out, status{statusWind, word, "", colKey})
	}
	return out
}

// drawWindArrow draws the wind as an arrow the way it blows, centred on
// (cx, cy), longer the harder it blows, at most r from the middle.
func (g *Game) drawWindArrow(dst *uiDst, cx, cy, r float64) {
	wd := float64(g.curr.WindDir)
	l := r * (0.6 + 0.4*float64(g.curr.Wind))
	c, s := math.Cos(wd), math.Sin(wd)
	tx, ty := float32(cx+c*l), float32(cy+s*l)
	strokeLine(dst, float32(cx-c*l), float32(cy-s*l), tx, ty, 1.7, colKey, true)
	for _, a := range []float64{2.5, -2.5} {
		strokeLine(dst, tx, ty, tx+float32(math.Cos(wd+a)*4), ty+float32(math.Sin(wd+a)*4), 1.7, colKey, true)
	}
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func max32i(v int16) int { return max(0, int(v)) }

// seaLabel is the target label of a sea monster.
func seaLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	switch c.Kind {
	case sim.CreatureMosa:
		return c.Kind.Name() + " — đánh khi nó ngoi lên (cắn vỡ thuyền!)", colBad, true
	case sim.CreatureKraken:
		return c.Kind.Name() + " — đánh để nó buông thuyền!", colBad, true
	}
	return "", colWhite, false
}

func imageRect(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

// boatInView reports whether boat id is in the snapshot, so its rider is
// drawn in it.
func (g *Game) boatInView(id sim.BoatID) bool {
	for _, b := range g.curr.Boats {
		if b.ID == id {
			return true
		}
	}
	return false
}

// boatHint is the R key's line in the help while near a boat or aboard.
func (g *Game) boatHint() (string, bool) {
	if g.curr.Self.Boat != 0 {
		return "xuống thuyền lên bờ (chèo sát bờ)", true
	}
	for _, p := range g.curr.Players {
		if p.ID != g.myID {
			continue
		}
		for _, b := range g.curr.Boats {
			if b.Rider == 0 && math.Hypot(float64(b.X-p.X), float64(b.Y-p.Y)) <= sim.BoatReach {
				return "lên " + b.Kind.Name(), true
			}
		}
	}
	return "", false
}
