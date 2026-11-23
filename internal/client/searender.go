package client

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

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
		back, front, seatX = mirrorSprite(back), mirrorSprite(front), -seatX
	}
	return back, front, seatX, seatY
}

// drawBoat draws b at (sx, sy) with its rider and sail.
func (g *Game) drawBoat(dst *ebiten.Image, b protocol.BoatView, sx, sy float64) {
	t := float64(g.frame) + float64(b.ID)*13
	bob := math.Round(math.Sin(t*0.06) * 1)
	if b.Held {
		sx += math.Round(math.Sin(t * 0.9)) // shaking in the squid's arms
	}
	back, front, seatX, seatY := g.boatSprites(b)
	left := boatLeft(float64(b.Heading))

	// A wake of foam behind a boat under way.
	if b.Speed > 0.02 {
		hx, hy := math.Cos(float64(b.Heading)), math.Sin(float64(b.Heading))
		n := int(min(10, float64(b.Speed)*60))
		for k := 1; k <= n; k++ {
			ph := math.Mod(t*0.05+float64(k)*0.37, 1)
			d := float64(k)*3 + ph*2
			wx := sx - hx*d*TileW/TileH*0.9 + math.Sin(float64(k)*2.3)*float64(k)*0.4
			wy := sy - hy*d*0.7 + 2
			a := uint8(180 * (1 - float64(k)/float64(n+1)))
			vector.FillRect(dst, float32(math.Round(wx)), float32(math.Round(wy)), 2, 1, color.RGBA{a, a, a, a}, false)
		}
	}
	drawAt(dst, back, sx, sy+bob, 1, 1, 0, nil)
	if b.Kind == sim.BoatSail {
		s := g.art.sea.sail[g.sailFrame(b)]
		mx := float64(mastX)
		if left {
			s, mx = mirrorSprite(s), -mx
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
func (g *Game) drawSwimmer(dst *ebiten.Image, p protocol.PlayerView, d drawable) {
	b := g.art.body(p.Look)
	dir := p.Dir & 3
	tint := playerColors[int(p.ID)%len(playerColors)]
	bob := math.Round(math.Sin(float64(g.frame)*0.12+float64(p.ID)) * 1)
	y := d.sy + 14 + bob // sunk to the chest
	// Only what is above the water: draw into a strip and cut it off.
	const above = 22
	if g.swimBuf == nil {
		g.swimBuf = ebiten.NewImage(48, 48)
	}
	g.swimBuf.Clear()
	drawBody(g.swimBuf, b, dir, 0, 24, 40, 1, 0, tint)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(d.sx-24, y-40)
	sub := g.swimBuf.SubImage(imageRect(0, 40-above-14, 48, 40-14)).(*ebiten.Image)
	op.GeoM.Translate(0, float64(40-above-14))
	dst.DrawImage(sub, op)
	for k := range 2 {
		ph := math.Mod(float64(g.frame)/70+float64(k)*0.5, 1)
		r, a := 5+ph*7, uint8(170*(1-ph))
		for s := range 20 {
			ang := 2 * math.Pi * float64(s) / 20
			vector.FillRect(dst, float32(math.Round(d.sx+r*math.Cos(ang))), float32(math.Round(y-13+r*0.4*math.Sin(ang))), 1, 1, color.RGBA{a, a, a, a}, false)
		}
	}
}

// --- sea monsters ---------------------------------------------------------

// seaMonster reports whether k is a monster of the open sea.
func seaMonster(k sim.CreatureKind) bool { return k == sim.CreatureMosa || k == sim.CreatureKraken }

// seaSprite is creatureSprite for the sea monsters out of the water.
func (g *Game) seaSprite(c protocol.CreatureView) (sprite, float64, bool) {
	a := &g.art.sea
	var s sprite
	switch c.Kind {
	case sim.CreatureMosa:
		s = a.mosaRise
		if c.State == sim.CreatureStrike {
			s = a.mosaBite
		}
	case sim.CreatureKraken:
		s = a.krakenHead
	default:
		return sprite{}, 0, false
	}
	if c.FaceLeft {
		s = mirrorSprite(s)
	}
	return s, 0, true
}

// drawSeaShadow draws a sea monster swimming under the surface: its dark
// shape, and for the sea lizard now and then a ridge of its back breaking
// the water.
func (g *Game) drawSeaShadow(dst *ebiten.Image, cam camera, c protocol.CreatureView, alpha float64) {
	x, y, moving := g.lerpCreature(c, alpha)
	sx, sy := cam.toScreen(x, y)
	a := &g.art.sea
	s := a.mosaShadow
	if c.Kind == sim.CreatureKraken {
		s = a.krakenShadow
	}
	if c.FaceLeft {
		s = mirrorSprite(s)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-s.ax, sy-s.ay)
	op.ColorScale.ScaleAlpha(0.42)
	dst.DrawImage(s.img, op)
	if c.Kind == sim.CreatureMosa && moving && (g.frame/50+int(c.ID))%3 == 0 {
		r := a.mosaBack
		if c.FaceLeft {
			r = mirrorSprite(r)
		}
		drawAt(dst, r, sx, sy+1, 1, 1, 0, nil)
	}
}

// drawKraken draws a giant squid holding a boat: its arms rising round it.
func (g *Game) drawKrakenArms(dst *ebiten.Image, c protocol.CreatureView, d drawable) {
	a := &g.art.sea
	for k := range 5 {
		ang := float64(k)*2*math.Pi/5 + 0.4
		ax := d.sx + math.Cos(ang)*26
		ay := d.sy + math.Sin(ang)*10
		arm := a.krakenArm[(g.frame/8+k)%len(a.krakenArm)]
		if math.Cos(ang) < 0 {
			arm = mirrorSprite(arm)
		}
		drawAt(dst, arm, ax, ay, 1, 1, 0, nil)
	}
}

// --- HUD -------------------------------------------------------------------------

// drawSeaHUD draws, under the health bars, the boat's state and the wind
// while aboard, or a swimmer's strength.
func (g *Game) drawSeaHUD(screen *ebiten.Image) {
	self := g.curr.Self
	y := float32(122)
	if self.Boat != 0 {
		for _, b := range g.curr.Boats {
			if b.ID != self.Boat {
				continue
			}
			max := b.Kind.MaxHP()
			panel(screen, 12, y, 230, 56)
			g.text(screen, b.Kind.Name(), 24, float64(y)+6, colWhite, text.AlignStart, 1)
			frac := float32(b.HP) / float32(max)
			fillRoundRect(screen, 24, y+30, 140, 10, 4, color.RGBA{0, 0, 0, 150})
			fillRoundRect(screen, 24, y+30, 140*max32(0, frac), 10, 4, color.RGBA{180, 130, 70, 255})
			g.text(screen, fmt.Sprintf("%d/%d", max32i(b.HP), max), 170, float64(y)+26, colMuted, text.AlignStart, 1)
			// The wind: an arrow the way it blows.
			cx, cy := float32(212), y+20
			vector.StrokeCircle(screen, cx, cy, 14, 1, color.RGBA{255, 255, 255, 60}, true)
			wd := float64(g.curr.WindDir)
			l := float32(8 + 5*g.curr.Wind)
			tx, ty := cx+float32(math.Cos(wd))*l, cy+float32(math.Sin(wd))*l
			vector.StrokeLine(screen, cx-float32(math.Cos(wd))*l, cy-float32(math.Sin(wd))*l, tx, ty, 2, colKey, true)
			for _, s := range []float64{2.6, -2.6} {
				vector.StrokeLine(screen, tx, ty, tx+float32(math.Cos(wd+s))*5, ty+float32(math.Sin(wd+s))*5, 2, colKey, true)
			}
			g.text(screen, "gió", float64(cx), float64(cy)+16, colMuted, text.AlignCenter, 1)
			if b.Held {
				g.text(screen, "Bị mực khổng lồ giữ chặt! Đánh nó để thoát", 24, float64(y)+62, colBad, text.AlignStart, 1)
			}
		}
		return
	}
	swimming := false
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			swimming = p.Swimming
		}
	}
	if (swimming || self.Breath > 0 && self.Breath < 0.999) && !self.Dead {
		panel(screen, 12, y, 230, 34)
		g.text(screen, "Sức bơi", 24, float64(y)+8, colWhite, text.AlignStart, 1)
		fillRoundRect(screen, 90, y+12, 140, 10, 4, color.RGBA{0, 0, 0, 150})
		col := color.RGBA{110, 190, 240, 255}
		if self.Breath <= 0 {
			col = colBad
		}
		fillRoundRect(screen, 90, y+12, 140*max32(0, self.Breath), 10, 4, col)
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
