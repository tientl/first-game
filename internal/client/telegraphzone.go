package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The red warnings on the ground (sim/skills.go: Zones): where a beast's
// special move will land, from the start of its wind-up till it lands. The
// whole of the zone is laid faint, with its edge; a deeper red fills it from
// the beast outwards (a circle from its centre) as the time runs out, so the
// moment it is full is the moment it lands. A placeholder look, drawn on the
// ground under everything standing; the combat effects may restyle it.

var (
	zoneFaint = color.RGBA{190, 16, 12, 255}
	zoneDeep  = color.RGBA{240, 26, 20, 255}
	zoneEdge  = color.RGBA{255, 84, 58, 255}
)

// drawZones draws the warnings in the snapshot, eased between ticks by alpha.
func (g *Game) drawZones(dst drawTarget, cam camera, alpha float64) {
	if len(g.curr.Zones) == 0 && !g.anyDust() {
		return
	}
	alignCamera(dst)
	for _, z := range g.curr.Zones {
		total := max(1, float64(z.Total))
		left := max(0, float64(z.Left)-alpha)
		k := 1 - left/total // 0 as it begins, 1 as it lands
		if z.Left == 0 {
			k = 1 // a charge under way: its line stays full till it is run
		}
		pulse := 0.5 + 0.5*math.Sin(float64(g.frame)*(0.15+0.35*k))
		zoneShape(dst, cam, z, 1, scaleAlpha(zoneFaint, 0.24+0.1*k))
		zoneShape(dst, cam, z, easeInOut(k), scaleAlpha(zoneDeep, 0.3+0.2*k))
		zoneEdgeLine(dst, cam, z, scaleAlpha(zoneEdge, 0.45+0.4*pulse*k+0.15))
	}
	g.drawDust(dst, cam)
}

// scaleAlpha is c at a share a of full, premultiplied.
func scaleAlpha(c color.RGBA, a float64) color.RGBA {
	a = clamp01(a)
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
}

// zonePoints is the outline of zone z, grown to a share k of its reach, in
// tiles.
func zonePoints(z protocol.ZoneView, k float64) [][2]float64 {
	x, y, a := float64(z.X), float64(z.Y), float64(z.Angle)
	r, w := float64(z.Reach)*k, float64(z.Width)
	var pts [][2]float64
	switch z.Shape {
	case sim.ZoneCone:
		pts = append(pts, [2]float64{x, y})
		for i := 0; i <= 18; i++ {
			t := a - w + 2*w*float64(i)/18
			pts = append(pts, [2]float64{x + r*math.Cos(t), y + r*math.Sin(t)})
		}
	case sim.ZoneCircle:
		for i := 0; i < 40; i++ {
			t := 2 * math.Pi * float64(i) / 40
			pts = append(pts, [2]float64{x + r*math.Cos(t), y + r*math.Sin(t)})
		}
	case sim.ZoneLine:
		ux, uy := math.Cos(a), math.Sin(a)
		nx, ny := -uy*w, ux*w
		pts = append(pts, [2]float64{x + nx, y + ny}, [2]float64{x + ux*r + nx, y + uy*r + ny},
			[2]float64{x + ux*r - nx, y + uy*r - ny}, [2]float64{x - nx, y - ny})
	}
	return pts
}

// zonePath is pts (tiles) as a path in art pixels on dst, shifted by (dx, dy).
func zonePath(cam camera, pts [][2]float64, dx, dy float32) *vector.Path {
	var p vector.Path
	for i, q := range pts {
		sx := float32(q[0]*TileW-cam.ox) + dx
		sy := float32(q[1]*TileH-cam.oy) + dy
		if i == 0 {
			p.MoveTo(sx, sy)
		} else {
			p.LineTo(sx, sy)
		}
	}
	p.Close()
	return &p
}

// zoneBox is the box round pts in art pixels.
func zoneBox(cam camera, pts [][2]float64) (x0, y0, x1, y1 float64) {
	x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, q := range pts {
		sx, sy := q[0]*TileW-cam.ox, q[1]*TileH-cam.oy
		x0, y0, x1, y1 = min(x0, sx), min(y0, sy), max(x1, sx), max(y1, sy)
	}
	return x0 - 2, y0 - 2, x1 + 2, y1 + 2
}

// zoneShape fills zone z, grown to a share k of its reach.
func zoneShape(dst drawTarget, cam camera, z protocol.ZoneView, k float64, clr color.RGBA) {
	if k <= 0.01 {
		return
	}
	pts := zonePoints(z, k)
	fill := func(img *ebiten.Image, dx, dy float32, scale float32) {
		p := zonePath(cam, pts, dx, dy)
		if scale != 1 {
			var q vector.Path
			q.AddPath(p, &vector.AddPathOptions{GeoM: zoneScale(scale)})
			p = &q
		}
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(clr)
		vector.FillPath(img, p, nil, op)
	}
	vecDraw(dst, cam, pts, fill)
}

// zoneEdgeLine strokes the edge of zone z.
func zoneEdgeLine(dst drawTarget, cam camera, z protocol.ZoneView, clr color.RGBA) {
	pts := zonePoints(z, 1)
	stroke := func(img *ebiten.Image, dx, dy float32, scale float32) {
		p := zonePath(cam, pts, dx, dy)
		var out vector.Path
		g := zoneScale(scale)
		out.AddStroke(p, &vector.AddStrokeOptions{StrokeOptions: vector.StrokeOptions{Width: 1.2, LineJoin: vector.LineJoinRound}, GeoM: g})
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(clr)
		vector.FillPath(img, &out, nil, op)
	}
	vecDraw(dst, cam, pts, stroke)
}

func zoneScale(k float32) ebiten.GeoM {
	var g ebiten.GeoM
	g.Scale(float64(k), float64(k))
	return g
}

// vecDraw runs a path drawing on any drawTarget, as fillCircle does: draw
// gets the image, the shift to it in art pixels, and how many of its pixels
// make an art pixel.
func vecDraw(dst drawTarget, cam camera, pts [][2]float64, draw func(img *ebiten.Image, dx, dy, scale float32)) {
	switch d := dst.(type) {
	case *ebiten.Image:
		draw(d, 0, 0, 1)
	case *fineBuf:
		d.vec(func(img *ebiten.Image, k float32) { draw(img, 0, 0, k) })
	case *worldDst:
		x0, y0, x1, y1 := zoneBox(cam, pts)
		d.vec(x0, y0, x1, y1, func(img *ebiten.Image, dx, dy float32) { draw(img, dx, dy, 1) })
	}
}

// anyDust reports whether some beast is scraping the ground to charge.
func (g *Game) anyDust() bool {
	for _, c := range g.curr.Creatures {
		if look, ok := skillLooks[c.Skill]; ok && look.dust && c.Phase == sim.PhaseWindup {
			return true
		}
		if c.Skill == sim.SkillMammothCharge && c.Phase == sim.PhaseWindup {
			return true
		}
	}
	return false
}

// drawDust puffs dust from under a beast scraping the ground before its
// charge: a placeholder, a few brown puffs rising and fading.
func (g *Game) drawDust(dst drawTarget, cam camera) {
	for _, c := range g.curr.Creatures {
		look := skillLooks[c.Skill]
		if !(look.dust || c.Skill == sim.SkillMammothCharge) || c.Phase != sim.PhaseWindup {
			continue
		}
		sx, sy := cam.toScreen(float64(c.X), float64(c.Y))
		dir := 1.0
		if c.FaceLeft {
			dir = -1
		}
		w := creatureShadow(c.Kind) / 2
		for i := range 5 {
			ph := math.Mod(float64(g.frame)/24+float64(i)*0.21+float64(c.ID)*0.13, 1)
			x := sx + dir*(w*0.4+float64(i%3)*3-ph*8*dir*dir) - dir*ph*6
			y := sy - 1 - ph*7 + float64(i%2)
			a := 0.45 * (1 - ph)
			fillCircle(dst, float32(x), float32(y), float32(1.5+ph*3.5), scaleAlpha(color.RGBA{150, 120, 84, 255}, a), true)
		}
	}
}

var dazedStarS sprite

// dazedStar is a little four-pointed star, at HD.
func dazedStar() sprite {
	if dazedStarS.img != nil {
		return dazedStarS
	}
	c := newCanvas(15, 15)
	c.dens = HDDensity
	for y := range 15 {
		for x := range 15 {
			dx, dy := math.Abs(float64(x)-7), math.Abs(float64(y)-7)
			if d := dx*dy*0.9 + (dx+dy)*0.55; d < 4.2 {
				l := 1 - d/4.2
				c.px[y*15+x] = color.RGBA{uint8(220 + 35*l), uint8(170 + 80*l), uint8(40 + 160*l*l), 255}
			} else if d < 5 {
				c.px[y*15+x] = color.RGBA{90, 60, 10, 255}
			}
		}
	}
	dazedStarS = mk(c, 7.5, 7.5)
	return dazedStarS
}

// drawDazed circles a stunned player's head with little stars.
func (g *Game) drawDazed(dst drawTarget, p protocol.PlayerView, sx, sy float64) {
	if !p.Stunned || p.Dead {
		return
	}
	for i := range 3 {
		a := float64(g.frame)*0.12 + 2*math.Pi*float64(i)/3
		x, y := sx+math.Cos(a)*7, sy-34+math.Sin(a)*2.2
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(x*3)/3-dazedStar().ax, math.Round(y*3)/3-dazedStar().ay)
		if math.Sin(a) < 0 {
			op.ColorScale.Scale(0.75, 0.7, 0.55, 1) // behind the head: dimmer
		}
		drawSprite(dst, dazedStar(), op)
	}
}
