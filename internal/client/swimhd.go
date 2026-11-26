package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// A player in the water. Swimming, the jointed figure does a breaststroke
// (moSwim) timed by the water it covers, as a stride is by the ground, or
// treads water standing still (moTread); wading through a ford it walks.
// Either way it is cut at the surface: what is above is drawn as it is,
// what is under it is drawn faint and blue-green, wavering, and the surface
// is marked round it: a ring of foam at the waterline, rings spreading, and
// on the move a bow wave at the head, a wake of foam behind and a splash
// where each hand goes in. Near the shore the water is shallower, so a
// swimmer rises out of it as they come in, and sinks into it going out.

const (
	strokeCycle = 24.5 // art pixels of water to a stroke: at full speed one every swimCycle frames, with its sound
	deepSink    = 21.0 // art pixels of the figure under the water out of one's depth
	wadeSink    = 6.0  // and at the water's edge
	fordCut     = 7.0  // how high a ford's water comes on a figure walking through it
)

// swimGait keeps the stroke of a swimmer: how far through it, and where they
// were last seen.
type swimGait struct {
	x, y  float64
	frame int
	ph    float64
}

// strokeOf moves p's stroke on to where they are now and returns its frame.
func (g *Game) strokeOf(p protocol.PlayerView, st *moState, wx, wy float64) int {
	s := &st.swim
	if g.frame-s.frame > 2 {
		s.x, s.y = wx, wy
	}
	if g.frame != s.frame {
		s.ph += math.Hypot(wx-s.x, wy-s.y) * TileW / strokeCycle
		s.ph -= math.Floor(s.ph)
		s.x, s.y, s.frame = wx, wy, g.frame
	}
	return int(s.ph*float64(moSwim.frames())) % moSwim.frames()
}

// swimDepth is how deep the water is where a swimmer is, 0 at the edge of
// the land to 1 out of their depth, half a tile or so out.
// sheetDepth, when not negative, is the depth swimDepth gives everywhere:
// for the motion sheets.
var sheetDepth = -1.0

func (g *Game) swimDepth(x, y float64) float64 {
	if sheetDepth >= 0 {
		return sheetDepth
	}
	best := 2.0
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - 1; ty <= cy+1; ty++ {
		for tx := cx - 1; tx <= cx+1; tx++ {
			if t := g.tileAt(tx, ty); t.Walkable() && t != sim.TileFord {
				// How far from that tile's square.
				dx := math.Max(0, math.Max(float64(tx)-x, x-float64(tx+1)))
				dy := math.Max(0, math.Max(float64(ty)-y, y-float64(ty+1)))
				best = math.Min(best, math.Hypot(dx, dy))
			}
		}
	}
	return clamp01((best - 0.08) / 0.6)
}

// drawSwimmerHD draws p swimming at d.
func (g *Game) drawSwimmerHD(dst drawTarget, p protocol.PlayerView, d drawable) {
	dir := p.Dir & 3
	st := g.moveState(p.ID)
	depth := g.swimDepth(d.wx, d.wy)
	m, fr := moTread, moTread.frameAt(float64(g.frame+int(p.ID)*37))
	if d.moving {
		m, fr = moSwim, g.strokeOf(p, st, d.wx, d.wy)
	}
	if depth < 0.35 && d.moving { // wading in or out: walking, not swimming yet
		lo, up := g.stride(p, dir, d.wx, d.wy, true)
		g.drawInWater(dst, p, d, lerpf(wadeSink*0.4, wadeSink*1.6, depth/0.35), 0, func(buf drawTarget, x, y float64) {
			g.drawFigureJ(buf, p, dir, lo, up, x, y, p.Armor)
		})
		return
	}
	sink := lerpf(wadeSink, deepSink, smooth(depth))
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	if m == moSwim && sideways {
		sink -= smooth(depth) // lying forward on the water: the shoulders at the surface, the arm over it
	}
	if m == moSwim {
		g.drawWake(dst, p, d)
	}
	lo, up := m.keys(dir, fr)
	// The hair floats behind the head: under it, but over the swimmer
	// swimming away.
	hx, hy := g.headOf(p, up, lo, d.sx, d.sy+sink)
	wave := float64(g.frame) / 70
	if m == moSwim {
		wave = st.swim.ph
	}
	float := func() {
		switch {
		case sideways:
			g.drawFloatHair(dst, p.Look, dir, hx-facingSide(dir)*2.5, d.sy-0.5, wave)
		case dir == sim.DirDown: // behind the head on the water, mostly hidden by it
			g.drawFloatHair(dst, p.Look, dir, hx, d.sy-2, wave)
		default:
			g.drawFloatHair(dst, p.Look, dir, hx, math.Max(hy+1, d.sy-2), wave)
		}
	}
	if dir != sim.DirUp {
		float()
	}
	g.drawInWater(dst, p, d, sink, 0, func(buf drawTarget, x, y float64) {
		var armor *armorArt
		if p.Armor > sim.TierStone {
			armor = g.art.armor(p.Armor, p.Look.Female)
		}
		drawFig(buf, figDraw{b: g.art.bodyIn(p.Look, p.Wear), armor: armor, up: up, lo: lo, tint: playerColors[int(p.ID)%len(playerColors)]}, x, y)
	})
	g.drawStrokeFX(dst, p, d, m, fr, depth)
	if dir == sim.DirUp {
		float()
	}
	if m == moSwim {
		g.drawEntry(dst, p, up, lo, d.sx, d.sy+sink, d.sy, fr, st.swim.ph)
	}
}

// The scratch image a figure in the water is drawn in, and where its feet
// go in it.
const (
	waterBufW, waterBufH = 64, 64
	waterBufX, waterBufY = 32, 56
)

// drawInWater draws a figure standing sink art pixels into the water at d
// (its feet that far below the surface, which is at d.sy), cut off at the
// surface, with what is under it drawn faint and wavering. lift raises the
// surface off d.sy. draw draws the figure with its feet at (x, y) of the
// scratch image it is given.
func (g *Game) drawInWater(dst drawTarget, p protocol.PlayerView, d drawable, sink, lift float64, draw func(buf drawTarget, x, y float64)) {
	if g.swimBuf == nil || g.swimBuf.w != waterBufW {
		g.swimBuf = newFineBuf(waterBufW, waterBufH)
	}
	buf := g.swimBuf
	buf.img.Clear()
	gx, gy := gridOf(dst)
	fx, fy := d.sx-gx-math.Floor(d.sx-gx), d.sy-gy-math.Floor(d.sy-gy)
	draw(buf, waterBufX+fx, waterBufY+fy)
	const k = HDDensity
	t := float64(g.frame) + float64(p.ID)*17
	ox, oy := d.sx-fx-waterBufX, d.sy+sink-fy-waterBufY
	// The surface, in the scratch image's rows (image pixels): it rises and
	// falls a little with the swell.
	surf := int(math.Round((waterBufY - sink - lift + math.Sin(t*0.07)*0.4) * k))
	surf = max(0, min(waterBufH*k, surf))
	put := func(y0, y1 int, dx float64, under bool) {
		if y1 <= y0 {
			return
		}
		sub := buf.img.SubImage(imageRect(0, y0, waterBufW*k, y1)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ox+dx, oy+float64(y0)/k)
		if under {
			op.ColorScale.Scale(0.42, 0.6, 0.66, 1)
			op.ColorScale.ScaleAlpha(0.42)
		}
		drawImageD(dst, sub, op, k)
	}
	// Under the water: in bands, each pushed a little aside by the swell,
	// fading with depth.
	for y := surf; y < waterBufH*k; y += 4 {
		dx := math.Round(math.Sin(t*0.09+float64(y)*0.35)*0.9) / k
		if float64(y-surf) > 10*k {
			break // deeper than that, the water hides it
		}
		put(y, min(y+4, waterBufH*k), dx, true)
	}
	put(0, surf, 0, false)
	// The waterline: a ring of foam round the figure where it meets the
	// surface, its near half over it.
	wy := oy + float64(surf)/k
	ring(dst, d.sx, wy, 6.5, 1.8, t, 0.75)
}

// ring draws a ring of foam of radii rx, ry about (x, y), brightest in
// front, broken here and there.
func ring(dst drawTarget, x, y, rx, ry, t, alpha float64) {
	for s := 0; s < 28; s++ {
		ang := 2 * math.Pi * float64(s) / 28
		if math.Sin(ang*3+t*0.05) > 0.75 {
			continue // a break in the foam
		}
		front := 0.5 + 0.5*math.Sin(ang) // 1 at the front, 0 behind
		a := alpha * (0.35 + 0.65*front)
		fillRect(dst, float32(math.Round(x+rx*math.Cos(ang))), float32(math.Round(y+ry*math.Sin(ang))), 1, 1,
			premul(color.RGBA{236, 246, 250, 255}, a), false)
	}
}

// drawStrokeFX draws what a swimmer does to the water: rings spreading
// round them; on the move a bow wave at the head, a wake behind and a
// splash where the hands go in on each stroke.
func (g *Game) drawStrokeFX(dst drawTarget, p protocol.PlayerView, d drawable, m motionID, fr int, depth float64) {
	t := float64(g.frame) + float64(p.ID)*17
	foam := color.RGBA{232, 244, 250, 255}
	dot := func(x, y, a float64, w float32) {
		fillRect(dst, float32(math.Round(x)), float32(math.Round(y)), w, 1, premul(foam, a), false)
	}
	// Rings, slower and wider when still.
	period := 80.0
	if m == moSwim {
		period = 50
	}
	for k := 0; k < 2; k++ {
		ph := math.Mod(t/period+float64(k)*0.5, 1)
		r := 6 + ph*9
		for s := 0; s < 24; s++ {
			ang := 2 * math.Pi * float64(s) / 24
			dot(d.sx+r*math.Cos(ang), d.sy+1+r*0.38*math.Sin(ang), 0.5*(1-ph)*depth, 1)
		}
	}
	if m != moSwim {
		return
	}
	fx, fy := p.Dir.Delta()
	// The bow wave: two short arcs pushed out ahead of the head.
	for s := -1.0; s <= 1; s += 2 {
		for i := 0.0; i < 4; i++ {
			ax := d.sx + float64(fx)*(5+i*0.3) + s*float64(fy)*i*1.5 + s*float64(fx)*0*i
			ay := d.sy + float64(fy)*(2+i*0.3) + s*float64(-fx)*i*0.4 + 0.5*i
			if fx == 0 {
				ax = d.sx + s*(3+i*1.4)
				ay = d.sy + float64(fy)*(3-i*0.5)
			}
			dot(ax, ay, 0.8-0.15*i, 1)
		}
	}
}

// fordWading reports whether p walks through water at d: a ford, or the
// water's edge.
// sheetFord puts everyone in a ford: for the motion sheets.
var sheetFord bool

func (g *Game) fordWading(d drawable) bool {
	if sheetFord {
		return true
	}
	return g.tileAt(int(math.Floor(d.wx)), int(math.Floor(d.wy))) == sim.TileFord
}

// drawWader draws p walking or standing in a ford: the water to the knees,
// a ring round them, a wake when they move.
func (g *Game) drawWader(dst drawTarget, p protocol.PlayerView, d drawable) {
	dir := p.Dir & 3
	lo, up := g.stride(p, dir, d.wx, d.wy, d.moving)
	g.drawInWater(dst, p, d, 0, fordCut, func(buf drawTarget, x, y float64) {
		g.drawFigureJ(buf, p, dir, lo, up, x, y, p.Armor)
	})
	t := float64(g.frame) + float64(p.ID)*17
	if d.moving {
		fx, fy := p.Dir.Delta()
		for i := 1; i <= 4; i++ {
			ph := math.Mod(t*0.05+float64(i)*0.27, 1)
			x := d.sx - float64(fx)*(float64(i)*3+ph*2)
			y := d.sy - fordCut + 1 - float64(fy)*float64(i)*2
			fillRect(dst, float32(math.Round(x-2)), float32(math.Round(y)), 4, 1, premul(color.RGBA{232, 244, 250, 255}, 0.5*(1-float64(i)/5)), false)
		}
	}
}

// waterReels are the motion sheets' scenes in the water: swimming, treading
// water, wading in from the shore, and through a ford.
func (g *Game) waterReels() []reel {
	swim := func(speed, d0, d1 float64) func(*Game, int, *protocol.PlayerView, *drawable) {
		return func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			dx, dy := p.Dir.Delta()
			along := float32(speed * float64(t) / TileW)
			p.X += float32(dx) * along
			p.Y += float32(dy) * along
			p.Swimming = true
			d.moving = speed > 0
			sheetDepth = lerpf(d0, d1, float64(t)/90)
		}
	}
	inWater := func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		g.drawSwimmerHD(dst, p, d)
		sheetDepth = -1
	}
	return []reel{
		{name: "swim", n: 68, dirs: allDirs, at: swim(0.72, 1, 1), draw: inWater, bg: reelWater},
		{name: "tread water", n: 80, dirs: allDirs, at: swim(0, 1, 1), draw: inWater, bg: reelWater},
		{name: "wade in", n: 90, dirs: allDirs, at: swim(0.72, 0, 1), draw: inWater, bg: reelWater},
		{name: "hair afloat", n: 140, dirs: allDirs, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			swim(0.72, 1, 1)(g, t, p, d)
			p.Look.Hair, p.Look.HairColor = sim.HairStyle(t/20%int(sim.NumHairStyles)), uint8(t/20%int(sim.NumHairColors))
			p.Look.Female = t/20%2 == 1
		}, draw: inWater, bg: reelWater},
		{name: "ford", n: 48, dirs: allDirs, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			dx, dy := p.Dir.Delta()
			along := float32(0.9 * float64(t) / TileW)
			p.X += float32(dx) * along
			p.Y += float32(dy) * along
			d.moving = true
		}, draw: func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
			sheetFord = true
			g.drawWader(dst, p, d)
			sheetFord = false
		}, bg: reelWater},
	}
}

// drawWake draws the V of foam a swimmer leaves behind, broken and fading:
// before the swimmer, as it lies behind them.
func (g *Game) drawWake(dst drawTarget, p protocol.PlayerView, d drawable) {
	t := float64(g.frame) + float64(p.ID)*17
	fx, fy := p.Dir.Delta()
	for i := 1; i <= 6; i++ {
		ph := math.Mod(t*0.03+float64(i)*0.31, 1)
		back := float64(i)*2.6 + ph*2
		spread := float64(i) * 1.0
		a := 0.55 * (1 - float64(i)/7) * (1 - ph*0.5)
		for s := -1.0; s <= 1; s += 2 {
			x := d.sx - float64(fx)*back
			y := d.sy + 1 - float64(fy)*back*0.75 + s*spread*0.42*math.Abs(float64(fx))
			if fx == 0 {
				x = d.sx + s*(2.5+spread)
			}
			fillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 2, 1, premul(color.RGBA{232, 244, 250, 255}, a), false)
		}
	}
}

// headOf is where the middle of the head of p's figure in frame (up, lo) is
// when its feet are at (x, y).
func (g *Game) headOf(p protocol.PlayerView, up upKey, lo loKey, x, y float64) (float64, float64) {
	d := up.dir
	if d == sim.DirLeft {
		up.dir = sim.DirRight
	}
	pose := upperPoseOf(up)
	var f figHD
	f.xf = bendOf(pose).head(up.dir, p.Look.Female)
	hx, hy := f.fwd(27.5, 35)
	ax, ay := hx/HDDensity-9, hy/HDDensity-limbFeet
	if d == sim.DirLeft {
		ax = -ax
	}
	return x + ax, y + ay + lowerPoseOf(lo).drop/HDDensity
}

// drawEntry draws the splash where a swimmer's hand goes into the water
// ahead, as it comes over it: the near hand seen from the side (frame 0),
// each in turn from the front or the back (0 and 4). The surface is at sy;
// the figure's feet at (x, y).
func (g *Game) drawEntry(dst drawTarget, p protocol.PlayerView, up upKey, lo loKey, x, y, sy float64, fr int, ph float64) {
	sideways := up.dir == sim.DirLeft || up.dir == sim.DirRight
	u := math.Mod(ph*float64(moSwim.frames()), 1) // through the frame
	hand := -1
	switch {
	case fr == 0:
		hand = 0
	case fr == 4 && !sideways:
		hand = 1
	case fr == 1 && u < 0.5: // the spray still falling
		hand, u = 0, u+1
	case fr == 5 && u < 0.5 && !sideways:
		hand, u = 1, u+1
	}
	if hand < 0 {
		return
	}
	b := g.art.bodyIn(p.Look, p.Wear)
	h := b.upper(up).hand[hand]
	hx := x + h.x
	u /= 1.5
	foam := color.RGBA{236, 246, 252, 255}
	for i := 0; i < 7; i++ {
		ang := -math.Pi/2 + (float64(i)-3)*0.42
		r := 1 + 5*u*(0.6+0.4*hashf(i, 5, uint32(p.ID)))
		px := hx + math.Cos(ang)*r
		py := sy - 1 + math.Sin(ang)*r*0.9 + 6*u*u // flung up, falling back
		fillRect(dst, float32(math.Round(px)), float32(math.Round(py)), 1, 1, premul(foam, 0.9*(1-u)), false)
	}
	// And a ring where it went in.
	ring(dst, hx, sy, 2+4*u, 0.8+1.4*u, float64(g.frame), 0.7*(1-u))
}
