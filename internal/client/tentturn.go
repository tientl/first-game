package client

import (
	"math"

	"survivalgame/internal/sim"
)

// The leaf tent turned (sim/facing.go): its door to the north, the east or
// the west. The tent is the same model as the one facing south
// (tentart.go), turned about the middle of its floor and seen again from
// the game's angle, so its light still falls from the upper left:
//
//   - North: the closed back end is to the viewer, a triangle thatched like
//     the slopes, with the back poles crossed and lashed over it and the end
//     of the ridge pole.
//   - East and west: turned not quite a quarter (tentSideYaw), so that the
//     long side and the open door both show. Through the door one sees
//     the floor of dry grass, the dark undersides of the thatch and the back
//     end from within, traced along the game's line of sight.
//
// Whoever sleeps in a turned tent is out of sight (tentShut); only their
// sleep floats up from it.

// tentSideYaw is how far a tent facing east or west is turned from facing
// south.
const tentSideYaw = 60 * math.Pi / 180

// tentGeo is how a turned tent stands: the way out of its door, east and
// north.
type tentGeo struct{ dx, dn float64 }

// tentGeoOf is the stance of a tent facing face; nil for south, drawn as it
// always was.
func tentGeoOf(face sim.Dir) *tentGeo {
	s, c := math.Sin(tentSideYaw), math.Cos(tentSideYaw)
	switch face & 3 {
	case sim.DirUp:
		return &tentGeo{0, 1}
	case sim.DirRight:
		return &tentGeo{s, -c}
	case sim.DirLeft:
		return &tentGeo{-s, -c}
	}
	return nil
}

// tv is a point or a way in the tent's own frame: x across it (to the
// right seen from before its door), y back from its front, z up; art
// pixels.
type tv struct{ x, y, z float64 }

// The middle of the tent's floor, in its frame, and where it is on the
// screen from the tile's middle as the tent faces south: it turns about it.
const (
	tentMidY  = tentL / 2
	tentMidSY = tentYF - tentDY/2
)

// world is way p of the tent's frame in the world: east, north, up.
func (g *tentGeo) world(p tv) tv {
	return tv{p.x*-g.dn + p.y*-g.dx, p.x*g.dx + p.y*-g.dn, p.z}
}

// projV is way v on the screen, art pixels.
func (g *tentGeo) projV(v tv) (x, y float64) {
	w := g.world(v)
	return w.x, -0.75*w.y - w.z
}

// proj is point p on the screen, art pixels from the tile's middle.
func (g *tentGeo) proj(p tv) (x, y float64) {
	x, y = g.projV(tv{p.x, p.y - tentMidY, p.z})
	return x, y + tentMidSY
}

// The planes of the tent besides its two slopes (sides -1 and 1), as a
// leaf's side names them.
const (
	tentGable  = 2.0 // the closed back end: a down from its apex, b across
	tentFront  = 3.0 // the open front, the same way
	tentGround = 4.0 // the ground: a across, b back
)

// tentPlane is plane side as a point and the two ways a and b go on it.
func tentPlane(side float64) (o, a, b tv) {
	switch side {
	case tentGable:
		return tv{0, tentL, tentH}, tv{0, 0, -1}, tv{1, 0, 0}
	case tentFront:
		return tv{0, 0, tentH}, tv{0, 0, -1}, tv{1, 0, 0}
	case tentGround:
		return tv{}, tv{1, 0, 0}, tv{0, 1, 0}
	}
	return tv{0, 0, tentH}, tv{side * tentB / slopeA, 0, -tentH / slopeA}, tv{0, 1, 0}
}

// planeXY is the art point at (a, b) on plane side.
func (g *tentGeo) planeXY(side, a, b float64) (x, y float64) {
	o, A, B := tentPlane(side)
	return g.proj(tv{o.x + a*A.x + b*B.x, o.y + a*A.y + b*B.y, o.z + a*A.z + b*B.z})
}

// planeAB is the point of plane side seen at art point (x, y); ok is false
// for a plane seen edge on.
func (g *tentGeo) planeAB(side, x, y float64) (a, b float64, ok bool) {
	o, A, B := tentPlane(side)
	ox, oy := g.proj(o)
	ax, ay := g.projV(A)
	bx, by := g.projV(B)
	det := ax*by - ay*bx
	if math.Abs(det) < 1e-3 {
		return 0, 0, false
	}
	rx, ry := x-ox, y-oy
	return (rx*by - ry*bx) / det, (ax*ry - ay*rx) / det, true
}

// tentNormal is the outward normal of plane side, in the tent's frame.
func tentNormal(side float64) tv {
	switch side {
	case tentGable:
		return tv{0, 1, 0}
	case tentFront:
		return tv{0, -1, 0}
	case tentGround:
		return tv{0, 0, 1}
	}
	return tv{side * tentH / slopeA, 0, tentB / slopeA}
}

// facesUs reports whether plane side turns towards the viewer, who looks
// north and down along (0, 1, -0.75).
func (g *tentGeo) facesUs(side float64) bool {
	n := g.world(tentNormal(side))
	return -0.8*n.y+0.6*n.z > 0.05
}

// tentLight is the way the light comes from, in the world: the upper left,
// a little from the front.
var tentLight = tv{-0.536, -0.15, 0.83}

// planeLit is the light on plane side, as the tent facing south has it:
// 0.7 on its left slope, 0.34 on its right.
func (g *tentGeo) planeLit(side float64) float64 {
	n := g.world(tentNormal(side))
	return 0.311 + 0.42*(n.x*tentLight.x+n.y*tentLight.y+n.z*tentLight.z)
}

// sXY is slopeXY for the tent as it stands: the art point at (a, b) on
// plane side.
func (t *tentPaint) sXY(side, a, b float64) (float64, float64) {
	if t.geo == nil {
		return slopeXY(side, a, b)
	}
	return t.geo.planeXY(side, a, b)
}

// sAB is the inverse of sXY: NaNs where the plane is seen edge on, which
// every test of where a point lies refuses.
func (t *tentPaint) sAB(side, x, y float64) (float64, float64) {
	if t.geo == nil {
		return slopeAB(side, x, y)
	}
	a, b, ok := t.geo.planeAB(side, x, y)
	if !ok {
		return math.NaN(), math.NaN()
	}
	return a, b
}

// foldSign is l.foldSign for the tent as it stands: which half of the leaf
// faces up the screen.
func (t *tentPaint) foldSign(l *tLeaf) float64 {
	if t.geo == nil || l.side == 0 {
		return l.foldSign()
	}
	_, A, B := tentPlane(l.side)
	_, ay := t.geo.projV(A)
	_, by := t.geo.projV(B)
	if -(l.b1-l.b0)*ay+(l.a1-l.a0)*by < 0 {
		return 1
	}
	return -1
}

// onGable reports whether image pixel (ix, iy) is on the closed end's
// triangle, a little ragged past its edges.
func (t *tentPaint) onGable(ix, iy int) bool {
	x, y := t.art(ix, iy)
	a, b := t.sAB(tentGable, x, y)
	return a >= -0.3 && a <= tentH+0.8 && math.Abs(b) <= tentB*a/tentH+0.8
}

// gnd is ground point (x, y), art pixels from the tile's middle as the
// tent faces south, where it is with the tent turned.
func (t *tentPaint) gnd(x, y float64) (float64, float64) {
	if t.geo == nil {
		return x, y
	}
	return t.geo.proj(tv{x, (tentYF - y) / 0.75, 0})
}

// groundLeaf lays leaf l on the ground (side 0, flat): its ends where they
// are with the tent turned.
func (t *tentPaint) groundLeaf(l tLeaf) {
	if t.geo != nil {
		f := l.flat
		if f == 0 {
			f = 0.6
		}
		x0, y0 := t.gnd(l.a0, l.b0*f)
		x1, y1 := t.gnd(l.a1, l.b1*f)
		l.a0, l.b0, l.a1, l.b1 = x0, y0/f, x1, y1/f
	}
	t.leaf(l)
}

// pole paints a pole of the frame from foot to top in the tent's frame.
func (t *tentPaint) pole(foot, top tv, r0, r1, lit float64, seed uint32) {
	x0, y0 := t.geo.proj(foot)
	x1, y1 := t.geo.proj(top)
	t.stick(x0, y0, x1, y1, r0, r1, lit, seed)
}

// framePoles paints the two poles of the end at y, crossed over its apex
// and lashed there, lit on the one to the left of the screen.
func (t *tentPaint) framePoles(y, out float64, seed uint32) {
	apex := tv{0, y, tentH}
	for _, side := range []float64{-1, 1} {
		foot := tv{side * (tentB + 0.6), y + out*1.6, 0}
		top := tv{foot.x + (apex.x-foot.x)*1.2 + side*0.4, foot.y + (apex.y-foot.y)*1.2, foot.z + (apex.z-foot.z)*1.2}
		fx, _ := t.geo.proj(foot)
		ax, _ := t.geo.proj(apex)
		lit := 0.5
		if fx < ax {
			lit = 0.7
		}
		t.pole(foot, top, 1.25, 0.9, lit, seed+uint32(10+side))
	}
	lx, ly := t.geo.proj(tv{0, y, tentH - 0.4})
	t.lash(lx, ly, 1.9, seed+20)
}

// shellTurned paints the shell of a turned tent: whatever of the ground it
// stands over first, so the tent hides it where it is behind; the slopes
// that face the viewer; the closed end or the open front with its frame.
func (t *tentPaint) shellTurned(v tentLeaf) {
	g := t.geo
	r := func(i, j int) float64 { return hashf(i, j, t.seed) }
	front, back := g.facesUs(tentFront), g.facesUs(tentGable)
	t.groundLeaves(v, r)
	stones := func(y, out float64) {
		for _, side := range []float64{-1, 1} {
			x, sy := g.proj(tv{side * (tentB + 1.6), y + out*2.13, 0})
			t.stone(x, sy, 2.1, 1.4, t.seed+uint32(30+side))
			x, sy = g.proj(tv{side * (tentB - 1.2), y + out*3.47, 0})
			t.stone(x, sy, 1.4, 1.0, t.seed+uint32(33+side))
		}
	}
	if front {
		stones(0, -1)
	}
	if back {
		stones(tentL, 1)
	}
	// The ridge pole, its end showing past the back.
	if !back {
		t.pole(tv{0, -2, tentH}, tv{0, tentL + 3.2, tentH}, 1.0, 0.9, 0.62, t.seed+3)
	}
	// The slopes the viewer sees, the darker first, so the lighter one's top
	// row lies over the ridge.
	var sides []float64
	for _, side := range []float64{1, -1} {
		if g.facesUs(side) {
			sides = append(sides, side)
		}
	}
	t.shown = sides
	if len(sides) == 2 && g.planeLit(sides[0]) > g.planeLit(sides[1]) {
		sides[0], sides[1] = sides[1], sides[0]
	}
	for _, side := range sides {
		lit := g.planeLit(side)
		t.clip = side
		t.thatchBed(side, lit-0.3)
		switch v {
		case tentBroad:
			t.broadSlope(side, lit)
		case tentPalm:
			t.palmSlope(side, lit)
		case tentFir:
			t.firSlope(side, lit)
		default:
			t.fernSlope(side, lit)
		}
		t.clip = 0
	}
	if v == tentFir {
		t.ridgeSnowTurned()
	}
	if back {
		t.gableThatch(v, g.planeLit(tentGable)+0.08)
		t.framePoles(tentL, 1, t.seed+50)
		// The end of the ridge pole, out over the lashing.
		t.pole(tv{0, tentL - 1, tentH + 0.2}, tv{0, tentL + 3.2, tentH + 0.2}, 1.0, 0.9, 0.62, t.seed+3)
	}
	if front {
		t.cutEdgeTurned()
		t.framePoles(0, -1, t.seed)
		t.fringe(v)
	}
}

// gableThatch thatches the closed end, lit: a bed of old leaves, and rows
// of new ones hanging from the apex down to the ground over it, each row
// over the one below, as on the slopes.
func (t *tentPaint) gableThatch(v tentLeaf, lit float64) {
	t.gable = true
	defer func() { t.gable = false }()
	x0, y0 := t.sXY(tentGable, 0, -tentB)
	x1, y1 := t.sXY(tentGable, tentH, tentB)
	bx0, by0, bx1, by1 := t.box(min(x0, x1)-tentB, min(y0, y1), max(x0, x1)+tentB, max(y0, y1), 2)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			x, y := t.art(ix, iy)
			a, b := t.sAB(tentGable, x, y)
			if !(a >= 0 && a <= tentH+0.3 && math.Abs(b) <= tentB*a/tentH+0.3) {
				continue
			}
			t.set(ix, iy, tmLeaf, lit-0.3+(fbm2(a*0.5, b*0.3, t.seed+79, 2)-0.5)*0.2)
		}
	}
	h := func(i, j int) float64 { return hashf(i, j, t.seed+601) }
	switch v {
	case tentBroad:
		for ri, tip := range []float64{34, 28, 22, 16, 10.5, 5.5} {
			off := 0.0
			if ri%2 == 1 {
				off = 3.6
			}
			for i, b := 0, -tentB-2+off; b <= tentB+2; i, b = i+1, b+7.2 {
				hb := func(j int) float64 { return h(ri*17+i, j) }
				ln := 11 + hb(1)*2.5
				mat := tmLeaf
				if hb(2) < 0.12 && ri < 2 {
					mat = tmDry
				}
				bb := b + (hb(0)-0.5)*2
				t.leaf(tLeaf{side: tentGable, a0: tip - ln, b0: bb + (hb(4)-0.5)*2, a1: tip + (hb(3)-0.5)*2, b1: bb + (hb(5)-0.5)*3,
					w: 4.2 + hb(6)*0.8, bend: (hb(7) - 0.5) * 2, mat: mat, lit: lit + (hb(8)-0.5)*0.12, fold: 0.14,
					veins: 7, tears: int(hb(9) * 1.6), shade: 0.15, aoEnd: 0.2, aoTip: 0.05, seed: t.seed + uint32(640+ri*31+i)})
			}
		}
	case tentFir:
		for i, b := range []float64{-19, -11, -3.5, 4, 11.5, 19} {
			t.bough(tentGable, -1+h(i, 1)*3, tentH+2, b+(h(i, 3)-0.5)*1.5, (h(i, 4)-0.5)*2, lit+(h(i, 5)-0.5)*0.1, t.seed+uint32(700+i*41))
		}
		for i, b := range []float64{-15, -7.5, 0, 7.5, 15} {
			t.bough(tentGable, 12+h(i, 6)*3, tentH+2, b+(h(i, 7)-0.5)*1.5, (h(i, 8)-0.5)*2, lit+0.04, t.seed+uint32(760+i*43))
		}
	case tentPalm:
		for i, b := range []float64{-18, -10.5, -3.5, 3.5, 10.5, 18} {
			mat := tmLeaf
			if h(i, 0) < 0.14 {
				mat = tmDry
			}
			t.frond(tentGable, -3+h(i, 1)*2, tentH+2, b+(h(i, 3)-0.5)*1.5, (h(i, 4)-0.5)*2, 1.55, 0.7,
				func(s float64) float64 { return (3 + 3.6*math.Sin(math.Pi*math.Min(1, s*1.15+0.1))) * (1 - 0.25*s) },
				mat, lit+(h(i, 5)-0.5)*0.12, 0.95, t.seed+uint32(800+i*53))
		}
	default:
		for ri, span := range [][2]float64{{15, tentH + 2}, {-3, 19}} {
			cols := []float64{-17, -11.5, -6, -0.5, 5, 10.5, 16}
			if ri == 1 {
				cols = []float64{-8, -2.5, 3, 8.5}
			}
			for i, b := range cols {
				mat := tmLeaf
				if h(ri*13+i, 0) < 0.07 {
					mat = tmDry
				}
				t.frond(tentGable, span[0]+(h(ri*13+i, 1)-0.5)*3, span[1], b+(h(ri*13+i, 3)-0.5)*1.5, (h(ri*13+i, 4)-0.5)*3, 1.25, 0.55,
					func(s float64) float64 { return 1.2 + 3.4*math.Pow(math.Sin(math.Pi*math.Min(1, s*0.95+0.05)), 0.8) },
					mat, lit+(h(ri*13+i, 5)-0.5)*0.12+0.04, 0.8, t.seed+uint32(900+ri*97+i*11))
			}
		}
	}
}

// cutEdgeTurned is cutEdge for the open front of a turned tent: the
// thickness of the thatch round the door, seen aslant.
func (t *tentPaint) cutEdgeTurned() {
	for iy := 0; iy < t.h; iy++ {
		for ix := 0; ix < t.w; ix++ {
			x, y := t.art(ix, iy)
			a, b, ok := t.geo.planeAB(tentFront, x, y)
			if !ok || a < 1 || a > tentH {
				continue
			}
			in := tentB*a/tentH - math.Abs(b)
			th := 2.2 + 1.4*vnoise(a*0.8, b*0.1, t.seed+88)
			if in < 0 || in > th {
				continue
			}
			if t.mat[iy*t.w+ix] != tmNone && in < 0.6 {
				continue
			}
			t.set(ix, iy, tmLeaf, 0.1+0.1*(1-in/th))
		}
	}
}

// ridgeSnowTurned is ridgeSnow along the ridge of a turned tent.
func (t *tentPaint) ridgeSnowTurned() {
	x0, y0 := t.geo.proj(tv{0, tentL - 0.8, tentH})
	x1, y1 := t.geo.proj(tv{0, 2, tentH})
	ex, ey := x1-x0, y1-y0
	L := math.Hypot(ex, ey)
	if L < 1 {
		return
	}
	ux, uy := ex/L, ey/L
	bx0, by0, bx1, by1 := t.box(x0, y0, x1, y1, 4)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			x, y := t.art(ix, iy)
			s := (x-x0)*ux + (y-y0)*uy
			if s < 0 || s > L {
				continue
			}
			d := -(x-x0)*uy + (y-y0)*ux
			hw := 1.5 + 0.9*vnoise(s*0.35, 3, t.seed+91) + 0.8*math.Max(0, vnoise(s*0.9, d*0.2, t.seed+92)-0.4)
			if f := s / 1.5; f < 1 {
				hw *= math.Sqrt(f) // rounded at the back
			}
			nx := d / hw
			if math.Abs(nx) > 1 {
				continue
			}
			v := lum(nx*0.75, -0.35) + 0.06
			if math.Abs(nx) > 0.82 {
				v -= 0.08
			}
			t.set(ix, iy, tmSnow, v)
		}
	}
}

// insideTurned paints what is seen of a turned tent under its shell: in
// through its door, if it shows, the floor of dry grass, the undersides of
// the thatch and the back end from within, each pixel traced along the
// line of sight from where it crosses the doorway; and the soft shadow on
// the ground round its foot.
func (t *tentPaint) insideTurned() {
	g := t.geo
	if g.facesUs(tentFront) {
		// The line of sight, (0, 1, -0.75) in the world, in the tent's frame.
		rx, ry, rz := g.dx, -g.dn, -0.75
		for iy := 0; iy < t.h; iy++ {
			for ix := 0; ix < t.w; ix++ {
				x, y := t.art(ix, iy)
				a, b, ok := g.planeAB(tentFront, x, y)
				if !ok {
					continue
				}
				z0, x0 := tentH-a, b
				if z0 < -0.3 || z0 > tentH-0.4 || math.Abs(x0) > tentB*(1-z0/tentH)+1.2 {
					continue
				}
				z0 = math.Max(z0, 0)
				const floor, wall, under = 0, 1, 2
				hit, tt := floor, z0/-rz
				if ry > 0 {
					if tb := tentL / ry; tb < tt {
						hit, tt = wall, tb
					}
				}
				s := math.Copysign(1, rx)
				if den := s*rx/tentB + rz/tentH; den > 0 {
					if ts := math.Max(0, (1-s*x0/tentB-z0/tentH)/den); ts < tt {
						hit, tt = under, ts
					}
				}
				px, py, pz := x0+rx*tt, ry*tt, z0+rz*tt
				edge := clamp01((tentB*(1-pz/tentH) - math.Abs(px)) / 7) // light falls off under the slopes
				switch hit {
				case floor:
					depth := clamp01(py / tentL)
					l := (0.44 - 0.38*depth) * (0.35 + 0.65*edge*edge)
					slant := 1.0
					if vnoise(px*0.3, py*0.4, t.seed+63) > 0.5 {
						slant = -1
					}
					sx, sy := float64(ix)+slant*float64(iy)*1.8, float64(iy)-slant*float64(ix)*0.25
					switch h := hashf(int(sx), int(sy/7), t.seed+64); {
					case h < 0.22:
						l += 0.14
					case h > 0.8:
						l -= 0.14
					}
					if py > tentL-1.2 {
						l -= 0.12 // where it meets the back end
					}
					t.set(ix, iy, tmStraw, l)
				case wall:
					up := pz / tentH
					n := fbm2(px*0.45, pz*0.7, t.seed+61, 3)
					l := 0.3 - 0.22*up + (n-0.5)*0.14
					if math.Mod(px*0.35+pz*0.6+n*3+20, 2.2) < 0.18 {
						l -= 0.08 // the edge of a leaf
					}
					l *= 0.55 + 0.45*edge
					if math.Abs(pz-9.5) < 0.55 { // the crossbar lashed across the back poles
						t.set(ix, iy, tmBark, 0.26)
						continue
					}
					t.set(ix, iy, tmInside, l)
				default: // the underside of the thatch, darker up under the ridge and further in
					n := fbm2(py*0.5, pz*0.6, t.seed+66, 2)
					l := 0.26 - 0.16*pz/tentH - 0.08*py/tentL + (n-0.5)*0.12
					if math.Mod(py*0.5+pz*0.4+n*3, 2.4) < 0.2 {
						l -= 0.07
					}
					t.set(ix, iy, tmInside, l)
				}
			}
		}
		// Dry grass spilling out over the doorstep.
		for i := 0; i < 26; i++ {
			x := -18 + 36*hashf(i, 1, t.seed+70)
			if math.Abs(x) > tentB-3 {
				continue
			}
			ln := 1.5 + 2*hashf(i, 2, t.seed+70)
			ang := (hashf(i, 3, t.seed+70) - 0.5) * 1.6
			for k := 0.0; k <= ln*tk; k++ {
				ax, ay := g.proj(tv{x + math.Sin(ang)*k/tk, 0.4 - math.Cos(ang)*k/tk, 0})
				ix, iy := t.px(ax, ay)
				t.set(int(ix), int(iy), tmStraw, 0.72-k/(ln*tk)*0.2)
			}
		}
	}
	// The soft shadow on the ground round the foot, falling right.
	for iy := 0; iy < t.h; iy++ {
		for ix := 0; ix < t.w; ix++ {
			x, y := t.art(ix, iy)
			gx, gy, ok := g.planeAB(tentGround, x-1.2, y)
			if !ok {
				continue
			}
			dx := math.Max(0, math.Abs(gx)-tentB)
			dy := math.Max(0, math.Max(-gy, gy-tentL)) * 0.75
			if d := math.Hypot(dx, dy); d < 3.5 {
				t.shadow[iy*t.w+ix] = max(t.shadow[iy*t.w+ix], float32(0.3*(1-smooth(d/3.5))*(1-0.5*clamp01(gy/tentL))))
			}
		}
	}
}

// onSlope reports whether image pixel (ix, iy) is on slope t.clip between
// its ridge and the ground: leaves that would hang past the foot into the
// ground are cut there, a little ragged, and past the ridge into the sky
// when it is the only slope seen.
func (t *tentPaint) onSlope(ix, iy int) bool {
	x, y := t.art(ix, iy)
	a, b := t.sAB(t.clip, x, y)
	if len(t.shown) == 1 && a < -1.2+0.8*hashf(int(b*2), 0, t.seed+17) {
		return false // the ridge, seen with the other slope behind it
	}
	return a <= slopeA+0.6+0.8*hashf(int(b*2), 1, t.seed+17)
}
