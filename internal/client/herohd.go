package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// A trial of a bigger, more detailed hero (see SetHeroHD in herohddraw.go):
// the default man drawn about 32 by 48 pixels, with a real run cycle, an
// idle and a sword blow, for three facings (left is right mirrored). Every
// frame is drawn here in code from a small posed figure: limbs and locks are
// shaded capsules, the face and the trimmings are placed by hand on top.
// Beside the colours each frame keeps a normal per pixel, which the trial's
// point lighting uses.

const (
	hdW, hdH   = 96, 80 // a frame: room for the sword's arc
	hdOX, hdOY = 48, 68 // the point between the feet: the row hdOY is the ground

	hdHip      = 14.0 // heights above the ground, in art pixels
	hdShoulder = 28.5
	hdHeadI    = 38 // the pixel row through the middle of the head
)

type hdAnim int

const (
	hdIdle hdAnim = iota
	hdRun
	hdAttack
	numHDAnims
)

// hdFrames is how many frames each animation has. The idle's last frame is
// a blink, shown now and then in place of the first.
var hdFrames = [numHDAnims]int{5, 8, 5}

// --- the buffer ---------------------------------------------------------------

type hdBuf struct {
	col  []color.RGBA
	nrm  [][3]float32 // x right, y up, z towards the viewer; zero: unlit (the arc)
	tone []int8
}

func newHDBuf() *hdBuf {
	return &hdBuf{make([]color.RGBA, hdW*hdH), make([][3]float32, hdW*hdH), make([]int8, hdW*hdH)}
}

// hdFig draws a figure part by part: shapes go to the layer l, and commit
// lays it on m, darkening what it borders so the part stands off.
type hdFig struct {
	m, l *hdBuf
	bias float64 // added to the light on what is drawn: below zero for the far limbs
}

func newHDFig() *hdFig { return &hdFig{m: newHDBuf(), l: newHDBuf()} }

var hdLight = func() [3]float64 {
	x, y, z := -0.52, 0.62, 0.6
	n := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / n, y / n, z / n}
}()

// hdTone picks one of four tones for a surface with normal (nx, ny, nz):
// clean bands, no dithering.
func hdTone(nx, ny, nz, bias float64) int {
	l := nx*hdLight[0] + ny*hdLight[1] + nz*hdLight[2] + bias
	switch {
	case l > 0.9:
		return 3
	case l > 0.44:
		return 2
	case l > 0.02:
		return 1
	}
	return 0
}

func hdIn(px, py int) bool { return px >= 0 && py >= 0 && px < hdW && py < hdH }

func (f *hdFig) put(px, py int, c color.RGBA, tone int, nx, ny, nz float64) {
	if !hdIn(px, py) {
		return
	}
	i := py*hdW + px
	f.l.col[i] = c
	f.l.tone[i] = int8(tone)
	f.l.nrm[i] = [3]float32{float32(nx), float32(ny), float32(nz)}
}

func (f *hdFig) shade(px, py int, r ramp, nx, ny float64) {
	d := nx*nx + ny*ny
	if d > 1 {
		s := math.Sqrt(d)
		nx, ny, d = nx/s, ny/s, 1
	}
	nz := math.Sqrt(1 - d)
	t := hdTone(nx, ny, nz, f.bias)
	f.put(px, py, r.at(t), t, nx, ny, nz)
}

// fx and fy are the figure coordinates of the centre of pixel (px, py).
func hdFX(px int) float64 { return float64(px-hdOX) + 0.5 }
func hdFY(py int) float64 { return float64(hdOY-py) - 0.5 }

// capPat draws a tapered capsule from (x0, y0), radius r0, to (x1, y1),
// radius r1. pat, when given, may recolour a pixel from how far along the
// axis it is (a, in pixels), how far off it (c) and its tone.
func (f *hdFig) capPat(x0, y0, r0, x1, y1, r1 float64, r ramp, pat func(a, c float64, tone int) (color.RGBA, bool)) {
	rm := math.Max(r0, r1)
	pxa, pxb := int(math.Floor(math.Min(x0, x1)-rm))+hdOX-1, int(math.Ceil(math.Max(x0, x1)+rm))+hdOX+1
	pya, pyb := hdOY-int(math.Ceil(math.Max(y0, y1)+rm))-1, hdOY-int(math.Floor(math.Min(y0, y1)-rm))+1
	dx, dy := x1-x0, y1-y0
	l2 := dx*dx + dy*dy
	ln := math.Sqrt(l2)
	for py := pya; py <= pyb; py++ {
		for px := pxa; px <= pxb; px++ {
			x, y := hdFX(px), hdFY(py)
			t := 0.0
			if l2 > 1e-9 {
				t = clamp01(((x-x0)*dx + (y-y0)*dy) / l2)
			}
			rr := lerpf(r0, r1, t)
			qx, qy := x0+dx*t, y0+dy*t
			ox, oy := x-qx, y-qy
			d := math.Sqrt(ox*ox + oy*oy)
			if d > rr || rr <= 0 {
				continue
			}
			f.shade(px, py, r, ox/rr, oy/rr)
			if pat != nil {
				cr := 0.0
				if ln > 1e-9 {
					cr = (ox*dy - oy*dx) / ln
				}
				i := py*hdW + px
				if c, ok := pat(t*ln, cr, int(f.l.tone[i])); ok {
					f.l.col[i] = c
				}
			}
		}
	}
}

func (f *hdFig) cap(x0, y0, r0, x1, y1, r1 float64, r ramp) {
	f.capPat(x0, y0, r0, x1, y1, r1, r, nil)
}

// ell draws an ellipse shaded as a ball; keep, when given, clips it.
func (f *hdFig) ell(cx, cy, rx, ry float64, r ramp, keep func(x, y float64) bool) {
	for py := hdOY - int(math.Ceil(cy+ry)) - 1; py <= hdOY-int(math.Floor(cy-ry))+1; py++ {
		for px := int(math.Floor(cx-rx)) + hdOX - 1; px <= int(math.Ceil(cx+rx))+hdOX+1; px++ {
			x, y := hdFX(px), hdFY(py)
			nx, ny := (x-cx)/rx, (y-cy)/ry
			if nx*nx+ny*ny > 1 || (keep != nil && !keep(x, y)) {
				continue
			}
			f.shade(px, py, r, nx, ny)
		}
	}
}

// poly fills a polygon (x, y pairs) shaded as a cylinder standing about
// x = cx with half-width hw, tipped a little by tilt (the normal's y).
func (f *hdFig) poly(pts []float64, r ramp, cx, hw, tilt float64) {
	n := len(pts) / 2
	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for i := 0; i < n; i++ {
		minX, maxX = math.Min(minX, pts[2*i]), math.Max(maxX, pts[2*i])
		minY, maxY = math.Min(minY, pts[2*i+1]), math.Max(maxY, pts[2*i+1])
	}
	for py := hdOY - int(math.Ceil(maxY)) - 1; py <= hdOY-int(math.Floor(minY))+1; py++ {
		for px := int(math.Floor(minX)) + hdOX - 1; px <= int(math.Ceil(maxX))+hdOX+1; px++ {
			x, y := hdFX(px), hdFY(py)
			in := false
			for i, j := 0, n-1; i < n; j, i = i, i+1 {
				xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
				if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
					in = !in
				}
			}
			if in {
				f.shade(px, py, r, math.Max(-0.98, math.Min(0.98, (x-cx)/hw)), tilt)
			}
		}
	}
}

// dot sets the pixel of column ix, row ih (the pixel whose centre is at
// (ix+0.5, ih+0.5)) on the layer, keeping the normal of what is under it.
func (f *hdFig) dot(ix, ih int, c color.RGBA) {
	px, py := ix+hdOX, hdOY-1-ih
	if !hdIn(px, py) {
		return
	}
	i := py*hdW + px
	n := f.l.nrm[i]
	if f.l.col[i].A == 0 {
		n = f.m.nrm[i]
		if f.m.col[i].A == 0 {
			n = [3]float32{0, 0, 1}
		}
	}
	f.l.col[i], f.l.nrm[i] = c, n
}

// dots sets a run of pixels along row ih from column ix.
func (f *hdFig) dots(ix, ih int, cs ...color.RGBA) {
	for i, c := range cs {
		if c.A != 0 {
			f.dot(ix+i, ih, c)
		}
	}
}

// has reports whether the layer has a pixel at column ix, row ih.
func (f *hdFig) has(ix, ih int) bool {
	px, py := ix+hdOX, hdOY-1-ih
	return hdIn(px, py) && f.l.col[py*hdW+px].A != 0
}

// band recolours the layer's pixels in rows h0..h1 with ramp r, tone kept.
func (f *hdFig) band(h0, h1 int, r ramp) {
	for ih := h0; ih <= h1; ih++ {
		py := hdOY - 1 - ih
		if py < 0 || py >= hdH {
			continue
		}
		for px := 0; px < hdW; px++ {
			if i := py*hdW + px; f.l.col[i].A != 0 {
				f.l.col[i] = r.at(int(f.l.tone[i]))
			}
		}
	}
}

// retone runs fn over the layer's pixels to move their tone (fur, folds).
func (f *hdFig) retone(r ramp, fn func(ix, ih, tone int) int) {
	for py := 0; py < hdH; py++ {
		for px := 0; px < hdW; px++ {
			i := py*hdW + px
			if f.l.col[i].A == 0 {
				continue
			}
			t := max(0, min(len(r)-1, fn(px-hdOX, hdOY-1-py, int(f.l.tone[i]))))
			f.l.tone[i] = int8(t)
			f.l.col[i] = r.at(t)
		}
	}
}

var hdInk = hex(0x1a1226)

// commit lays the layer on the figure. edge darkens the pixels already
// there that the part borders: all round it, or only below and to the
// right, where its shadow would fall.
func (f *hdFig) commit(edge float64, all bool) {
	if edge > 0 {
		lay := func(px, py int) bool { return hdIn(px, py) && f.l.col[py*hdW+px].A != 0 }
		for py := 0; py < hdH; py++ {
			for px := 0; px < hdW; px++ {
				i := py*hdW + px
				if f.l.col[i].A != 0 || f.m.col[i].A == 0 {
					continue
				}
				near := lay(px-1, py) || lay(px, py-1)
				if all {
					near = near || lay(px+1, py) || lay(px, py+1)
				}
				if near {
					a := f.m.col[i].A
					f.m.col[i] = mix(withAlpha(f.m.col[i], 255), hdInk, edge)
					f.m.col[i].A = a
				}
			}
		}
	}
	for i, c := range f.l.col {
		if c.A != 0 {
			f.m.col[i], f.m.nrm[i], f.m.tone[i] = c, f.l.nrm[i], f.l.tone[i]
			f.l.col[i] = color.RGBA{}
		}
	}
}

// outline rims the whole figure: a dark line tinted by what it borders,
// lighter along the top and left, where the light falls.
func (b *hdBuf) outline() {
	src := make([]color.RGBA, len(b.col))
	copy(src, b.col)
	solid := func(px, py int) (color.RGBA, bool) {
		if !hdIn(px, py) {
			return color.RGBA{}, false
		}
		c := src[py*hdW+px]
		return c, c.A == 255 && b.nrm[py*hdW+px] != [3]float32{}
	}
	for py := 0; py < hdH; py++ {
		for px := 0; px < hdW; px++ {
			i := py*hdW + px
			if src[i].A == 255 {
				continue
			}
			// The neighbour below or to the right makes this a lit edge.
			for k, d := range [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}} {
				if n, ok := solid(px+d[0], py+d[1]); ok {
					s := 0.78
					if k < 2 {
						s = 0.62
					}
					b.col[i] = mix(n, hdInk, s)
					b.nrm[i] = b.nrm[(py+d[1])*hdW+px+d[0]]
					break
				}
			}
		}
	}
}

// --- colours ------------------------------------------------------------------

var (
	hdSkin    = mkRamp(0x9a5148, 0xd08a68, 0xf2b98e, 0xffdcb4)
	hdHair    = mkRamp(0x2c1614, 0x55301c, 0x86562c, 0xc08a48)
	hdBeard   = mkRamp(0x20100f, 0x3d2218, 0x623b22, 0x8a5a30)
	hdFur     = mkRamp(0x3a3442, 0x625a62, 0x8f8780, 0xbdb4a2, 0xe2dbc6)
	hdTrouser = mkRamp(0x2a1c1e, 0x46302a, 0x664a38, 0x84664a)
	hdWrap    = mkRamp(0x77603f, 0xa58c62, 0xd0bc8e, 0xeee0b8)
	hdLeather = mkRamp(0x351d12, 0x5a341a, 0x84522a, 0xb07a40)
	hdBoot    = mkRamp(0x24140f, 0x402617, 0x5e3b22, 0x805632)
	hdGold    = mkRamp(0x8a5a14, 0xd09a26, 0xf6cf52, 0xfff6c0)
	hdSteel   = mkRamp(0x4a5068, 0x8a94ac, 0xc6d0de, 0xffffff)
	hdIris    = hex(0x58a0e8)
	hdIrisDk  = hex(0x2f5fa8)
	hdEyeDark = hex(0x1d1626)
	hdEyeWht  = hex(0xf4f6fa)
	hdMouth   = hex(0x7a2c2c)
	hdBlush   = hex(0xea9480)
)

// hdTunic is the tunic's ramp in a player's colour: shadows pulled towards
// violet, lights towards a warm white.
func hdTunic(t color.RGBA) ramp {
	t.A = 255
	return ramp{
		mix(mix(t, hex(0x1c1438), 0.68), hex(0x000000), 0.1),
		mix(t, hex(0x241a4a), 0.4),
		mix(t, hex(0xffffff), 0.04),
		mix(t, hex(0xfff4d8), 0.42),
	}
}

// --- the pose -----------------------------------------------------------------

// hdLimb is a bent limb: its middle joint and its end.
type hdLimb struct{ jx, jy, ex, ey float64 }

type hdPose struct {
	bob   float64 // the hips and all above them, up
	chest float64 // the shoulders and the head, up again: breathing
	lean  float64 // side view: how far forward the shoulders are of the hips
	headX float64
	headY float64

	leg    [2]hdLimb  // knee and ankle from the ground point below the hip; 0 is the near (or the viewer's left) leg
	foot   [2]float64 // side view: the toe's pitch, radians up; front views: how far towards the viewer the foot is, -1 to 1
	arm    [2]hdLimb  // elbow and hand from the shoulder
	armLow [2]bool    // front views: the arm is behind the body

	hairT, hairY   float64 // the locks' tips, back and up
	mantT, mantY   float64 // the mantle's tail and fringe
	braidX, braidY float64
	blink          bool

	sword   bool
	swAng   float64 // the blade: radians from upright, clockwise on screen
	smear   bool
	smA0    float64 // the arc the tip has just swept, the same angles, and
	smA1    float64 // where it is now
	smCX    float64
	smCY    float64
	smR     float64
	smFade  float64 // 1: full, towards 0: thinning out
	swUnder bool    // the sword is behind the body
}

// hdArmSide is an arm in the side view: the upper arm a1 radians forward of
// hanging, the forearm bent a2 more.
func hdArmSide(a1, a2 float64) hdLimb {
	ex, ey := math.Sin(a1)*5.2, -math.Cos(a1)*5.2
	return hdLimb{ex, ey, ex + math.Sin(a1+a2)*5.2, ey - math.Cos(a1+a2)*5.2}
}

// hdLegIK bends a leg from the hip (hx, hy) to the ankle (ax, ay), the knee
// forward by dir.
func hdLegIK(hx, hy, ax, ay, dir float64) hdLimb {
	const l1, l2 = 6.2, 7.4
	dx, dy := ax-hx, ay-hy
	d := math.Sqrt(dx*dx + dy*dy)
	if d > l1+l2-0.01 {
		k := (l1 + l2 - 0.01) / d
		dx, dy, d = dx*k, dy*k, l1+l2-0.01
		ax, ay = hx+dx, hy+dy
	}
	a := (l1*l1 - l2*l2 + d*d) / (2 * d)
	h := math.Sqrt(math.Max(0, l1*l1-a*a))
	mx, my := hx+dx*a/d, hy+dy*a/d
	// perpendicular pointing forward (+x) for a leg hanging down
	px, py := -dy/d, dx/d
	return hdLimb{mx + px*h*dir, my + py*h*dir, ax, ay}
}

var (
	hdRunBob = [8]float64{0, -1, 0, 1, 0, -1, 0, 1}
	// the near foot through a stride, side view: forward and up
	hdRunFoot = [8][3]float64{
		{7, 0.6, 0.5}, {3.5, 0, 0}, {-1, 0, 0}, {-6, 1.2, -0.7},
		{-8.5, 5.5, -1.2}, {-4.5, 7.5, -0.9}, {1.5, 6, -0.2}, {6.5, 3.2, 0.4},
	}
	// the same stride seen from the front: forward (towards the viewer) and lift
	hdRunFront = [8][2]float64{
		{1, 0}, {0.5, 0}, {0, 0}, {-0.7, 0.2},
		{-1, 0.8}, {-0.3, 1}, {0.5, 0.9}, {0.9, 0.4},
	}
)

func hdLerpLimb(a, b hdLimb, t float64) hdLimb {
	return hdLimb{lerpf(a.jx, b.jx, t), lerpf(a.jy, b.jy, t), lerpf(a.ex, b.ex, t), lerpf(a.ey, b.ey, t)}
}

// hdFrontArm is an arm seen from the front: s is its swing, 1 towards the
// viewer and -1 away; side is -1 for the viewer's left arm.
func hdFrontArm(s, side float64) hdLimb {
	rest := hdLimb{1.2, -5, 1.4, -10.2}
	fwd := hdLimb{1.6, -4.4, -1.2, -4.8}
	back := hdLimb{2.4, -4.2, 3, -7.4}
	l := rest
	if s > 0 {
		l = hdLerpLimb(rest, fwd, s)
	} else {
		l = hdLerpLimb(rest, back, -s)
	}
	l.jx *= side
	l.ex *= side
	return l
}

// hdPoseOf is the pose of frame i of an animation, facing d (down, up or
// right).
func hdPoseOf(an hdAnim, d sim.Dir, i int) hdPose {
	var p hdPose
	side := d == sim.DirRight
	stand := func() {
		if side {
			p.leg[0] = hdLegIK(0.5, hdHip, 1.2, 1.6, 1)
			p.leg[1] = hdLegIK(-0.5, hdHip, -1.6, 1.6, 1)
			p.arm[0] = hdArmSide(0.05, 0.25)
			p.arm[1] = hdArmSide(-0.1, 0.3)
		} else {
			for k, s := range [2]float64{-1, 1} {
				p.leg[k] = hdLimb{s * 3.1, 8.6, s * 3.3, 1.6}
				p.arm[k] = hdFrontArm(0, s)
			}
		}
	}
	switch an {
	case hdIdle:
		stand()
		p.chest = [5]float64{0, 1, 1, 0, 0}[i]
		p.mantY = [5]float64{0, 0, -1, -1, 0}[i]
		p.hairY = [5]float64{0, 0, 1, 0, 0}[i]
		p.braidY = [5]float64{0, -1, 0, 0, 0}[i]
		p.blink = i == 4
		if side {
			p.lean = 0.5
		}
	case hdRun:
		p.bob = hdRunBob[i]
		vel := hdRunBob[i] - hdRunBob[(i+7)%8]
		vel1 := hdRunBob[(i+7)%8] - hdRunBob[(i+6)%8]
		p.hairY, p.mantY, p.braidY = -vel*0.9, -vel1*1.2, -vel1
		if side {
			p.lean = 3.6
			p.headX = 0.4
			p.hairT, p.mantT, p.braidX = 2.2+0.5*vel1, 3.2+0.6*vel, -2.5
			for k := range 2 {
				ft := hdRunFoot[(i+4*k)%8]
				hx := 0.6 - 1.2*float64(k)
				p.leg[k] = hdLegIK(hx, hdHip+p.bob, hx+ft[0], ft[1]+1.6, 1)
				p.foot[k] = ft[2]
				sw := -ft[0] / 8.5
				p.arm[k] = hdArmSide(sw*1.05+0.1, 1.45+0.25*sw)
			}
		} else {
			p.hairT, p.mantT = 1, 1.5
			for k, s := range [2]float64{-1, 1} {
				ft := hdRunFront[(i+4*k)%8]
				fwd, lift := ft[0], ft[1]
				if d == sim.DirUp {
					fwd = -fwd
				}
				ay := 1.6 + lift*5.6 - fwd*0.9
				p.leg[k] = hdLimb{s * (3.1 + 0.5*lift), (hdHip+p.bob+ay)/2 + 1, s * 3.3, ay}
				p.foot[k] = fwd
				as := -ft[0]
				if d == sim.DirUp {
					as = -as
				}
				p.arm[k] = hdFrontArm(as*0.95, s)
				p.armLow[k] = as < -0.5
			}
		}
	case hdAttack:
		hdAttackPose(&p, d, i)
	}
	return p
}

// hdAttackPose is the sword blow: two frames of wind-up, the slash, its
// follow-through and the recovery.
func hdAttackPose(p *hdPose, d sim.Dir, i int) {
	p.sword = true
	if d == sim.DirRight {
		p.lean = [5]float64{-0.6, -1.6, 4.4, 4.6, 2}[i]
		p.bob = [5]float64{0, 0, -1.5, -1.5, -0.5}[i]
		front := [5]float64{3.5, 3.5, 7, 7, 5}[i]
		back := [5]float64{-3.5, -4, -6, -6, -4.5}[i]
		p.leg[0] = hdLegIK(0.6, hdHip+p.bob, front, 1.6, 1)
		p.leg[1] = hdLegIK(-0.6, hdHip+p.bob, back, 1.6, 1)
		p.foot[1] = [5]float64{0, 0, -0.5, -0.5, -0.2}[i]
		p.arm[0] = hdArmSide([5]float64{-1.9, -2.7, 1.45, 0.75, 0.5}[i], [5]float64{-0.9, -0.7, 0.15, 0.15, 0.6}[i])
		p.arm[1] = hdArmSide([5]float64{0.9, 1.1, -0.9, -0.8, -0.3}[i], [5]float64{1.2, 1.2, 0.5, 0.5, 0.6}[i])
		p.swAng = [5]float64{-0.75, -1.9, 1.75, 2.45, 1.25}[i]
		p.hairT = [5]float64{0, -1, 2.5, 1.5, 0.5}[i]
		p.mantT = [5]float64{0, -1, 3, 3.5, 1.5}[i]
		p.braidX = [5]float64{0.5, 1, -2, -1, 0}[i]
		if i == 2 || i == 3 {
			p.smear = true
			p.smA0 = [5]float64{0, 0, -1.3, 0.9, 0}[i]
			p.smA1 = p.swAng
			p.smFade = [5]float64{0, 0, 1, 0.5, 0}[i]
		}
		return
	}
	// Facing the viewer the sword hand is on the viewer's left; facing away
	// it is on the right, and the blow lands beyond the body.
	s := -1.0
	k, o := 0, 1
	if d == sim.DirUp {
		s, k, o = 1, 1, 0
	}
	p.bob = [5]float64{0, 0.5, -1.5, -1.5, -0.5}[i]
	for q, sg := range [2]float64{-1, 1} {
		w := [5]float64{0.5, 0.8, 1.6, 1.6, 0.8}[i]
		p.leg[q] = hdLimb{sg * (3.1 + w*0.5), 8.6 + p.bob/2, sg * (3.3 + w), 1.6}
	}
	// the hand from the shoulder, written for the viewer's-left hand
	hx := [5]float64{-4.5, -1.5, 8, 10.5, 6}[i]
	hy := [5]float64{3, 8, -7.5, -6.5, -8}[i]
	ex := [5]float64{-3.5, -3, 3.5, 4.5, 2}[i]
	ey := [5]float64{-2.5, 2.5, -4.5, -4.5, -5}[i]
	p.arm[k] = hdLimb{-s * ex, ey, -s * hx, hy}
	p.arm[o] = hdFrontArm([5]float64{-0.3, -0.6, 0.5, 0.5, 0.2}[i], -s)
	p.armLow[o] = i < 2
	ang := [5]float64{-0.55, -1.15, -4.05, -4.6, -3.9}[i]
	p.swAng = -s * ang
	p.hairY = [5]float64{0, 1, -1, 0, 0}[i]
	p.mantY = [5]float64{0, 0.5, -1, -1, 0}[i]
	p.mantT = [5]float64{0, 0, 1.5, 1.5, 0.5}[i]
	p.swUnder = d == sim.DirUp && i >= 2
	if i == 2 || i == 3 {
		p.smear = true
		p.smA0 = -s * [5]float64{0, 0, -1.2, -3.2, 0}[i]
		p.smA1 = p.swAng
		p.smFade = [5]float64{0, 0, 1, 0.5, 0}[i]
	}
}

// --- drawing ------------------------------------------------------------------

// hdFrame is one drawn frame: colours and normals, hdW by hdH.
type hdFrame struct {
	col *canvas
	nrm *canvas // the normal as a colour: (n+1)/2, y up; mid grey with no blue: unlit
}

// genHeroHD draws frame i of an animation for facing d (down, up or right)
// in a tunic of colour tint, with or without a sword in hand.
func genHeroHD(tint color.RGBA, an hdAnim, d sim.Dir, i int, armed bool) hdFrame {
	p := hdPoseOf(an, d, i)
	switch {
	case an == hdAttack:
		p.sword = armed
		p.smear = p.smear && armed
	case armed: // carried low, the point to the ground
		p.sword = true
		switch d {
		case sim.DirRight:
			p.swAng = 2.25 + 0.25*p.arm[0].ex/5
		case sim.DirDown:
			p.swAng = -2.75
		default:
			p.swAng = 2.75
		}
	}
	f := newHDFig()
	h := &hdHero{f: f, p: p, tunic: hdTunic(tint)}
	if d == sim.DirRight {
		h.side()
	} else {
		h.front(d == sim.DirUp)
	}
	f.m.outline()
	col, nrm := newCanvas(hdW, hdH), newCanvas(hdW, hdH)
	copy(col.px, f.m.col)
	for k, n := range f.m.nrm {
		if f.m.col[k].A == 0 {
			continue
		}
		if n == [3]float32{} {
			nrm.px[k] = color.RGBA{128, 128, 0, 255}
			continue
		}
		e := func(v float32) uint8 { return uint8(math.Round(float64(v*0.5+0.5) * 255)) }
		nrm.px[k] = color.RGBA{e(n[0]), e(n[1]), e(n[2]), 255}
	}
	return hdFrame{col, nrm}
}

type hdHero struct {
	f     *hdFig
	p     hdPose
	tunic ramp
}

// furTex roughens the layer into fur: slanting strokes, a tone down and a
// tone up.
func (h *hdHero) furTex() {
	h.f.retone(hdFur, func(ix, ih, t int) int {
		t++ // the fur ramp has a tone more
		switch k := ((ix+ih/2*3)%6 + 6) % 6; {
		case k == 0 && ih&1 == 0:
			return t - 1
		case k == 3 && ih&1 == 1 && t < 4:
			return t + 1
		}
		return t
	})
}

// leg draws leg k from hip (hx, hy): trousers, criss-crossed wraps on the
// shin, a fur cuff and the boot. In the side view toe is the foot's pitch;
// from the front it is how far towards the viewer the foot is.
func (h *hdHero) leg(k int, hx, hy float64, side, back bool) {
	f, l := h.f, h.p.leg[k]
	kx, ky := l.jx, l.jy
	ax, ay := l.ex, l.ey
	// the foot
	if side {
		a := h.p.foot[k]
		tx, ty := ax+math.Cos(a)*3.6, ay-1.0+math.Sin(a)*3.6
		f.cap(ax-0.6, ay-0.5, 1.9, tx, ty, 1.5, hdBoot)
	} else {
		fw := h.p.foot[k]
		if back {
			fw = -fw
		}
		f.ell(ax+0.2*sign(ax), ay-0.6-0.5*fw, 2.7+0.3*fw, 1.9+0.3*fw, hdBoot, nil)
	}
	f.commit(0.35, true)
	ux0, uy0 := hdUnit(ax-kx, ay-ky)
	// thigh
	f.cap(hx, hy, 2.7, kx, ky, 2.4, hdTrouser)
	// shin, wrapped
	f.capPat(kx, ky, 2.4, ax-ux0*1.4, ay-uy0*1.4, 2.2, hdTrouser, func(a, c float64, tone int) (color.RGBA, bool) {
		if a < 0.6 {
			return color.RGBA{}, false
		}
		ph := math.Mod(a+math.Abs(c)*0.9+40, 2.6)
		if ph < 0.9 {
			return hdWrap.at(tone - 1), true
		}
		return hdWrap.at(tone), true
	})
	// boot shaft and its fur cuff
	dx, dy := ax-kx, ay-ky
	n := math.Hypot(dx, dy)
	if n < 0.01 {
		dx, dy, n = 0, -1, 1
	}
	ux, uy := dx/n, dy/n
	f.cap(ax-ux*1.2, ay-uy*1.2, 2.3, ax, ay, 2.2, hdBoot)
	f.commit(0.3, true)
	f.cap(ax-ux*2.3-2.5*uy, ay-uy*2.3+2.5*ux, 1.1, ax-ux*2.3+2.5*uy, ay-uy*2.3-2.5*ux, 1.1, hdFur)
	h.furTex()
	f.commit(0.3, false)
}

func hdUnit(dx, dy float64) (float64, float64) {
	n := math.Hypot(dx, dy)
	if n < 0.01 {
		return 0, -1
	}
	return dx / n, dy / n
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// arm draws arm k from shoulder (sx, sy): a tunic sleeve, a leather bracer
// and the hand.
func (h *hdHero) arm(k int, sx, sy float64) {
	f, l := h.f, h.p.arm[k]
	ex, ey := sx+l.jx, sy+l.jy
	hx, hy := sx+l.ex, sy+l.ey
	f.cap(sx, sy, 2.5, ex, ey, 2.2, h.tunic)
	mx, my := lerpf(ex, hx, 0.3), lerpf(ey, hy, 0.3)
	f.cap(ex, ey, 2.2, mx, my, 2.0, h.tunic)
	f.capPat(mx, my, 2.1, lerpf(ex, hx, 0.8), lerpf(ey, hy, 0.8), 2.0, hdLeather, func(a, c float64, tone int) (color.RGBA, bool) {
		if a < 0.9 {
			return hdLeather.at(tone + 1), true // the bracer's bright rim
		}
		return color.RGBA{}, false
	})
	f.ell(hx, hy, 2.1, 2.1, hdSkin, nil)
	f.commit(0.42, true)
}

// swordAt draws the sword gripped at (gx, gy), and the arc its tip has
// swept.
func (h *hdHero) swordAt(gx, gy float64) {
	f, p := h.f, h.p
	if !p.sword {
		return
	}
	dx, dy := math.Sin(p.swAng), math.Cos(p.swAng)
	const blade = 15.0
	if p.smear {
		// the arc: a bright crescent about the grip's path, its tail thinning
		cx, cy := gx-dx*1.0, gy-dy*1.0
		r1 := blade + 4.5
		a0, a1 := p.smA0, p.smA1
		for py := 0; py < hdH; py++ {
			for px := 0; px < hdW; px++ {
				x, y := hdFX(px)-cx, hdFY(py)-cy
				r := math.Hypot(x, y)
				if r > r1 || r < 4 {
					continue
				}
				a := math.Atan2(x, y)
				// bring a into the sweep's range
				lo, hi := math.Min(a0, a1), math.Max(a0, a1)
				for a < lo {
					a += 2 * math.Pi
				}
				for a > hi {
					a -= 2 * math.Pi
				}
				if a < lo {
					continue
				}
				t := (a - a0) / (a1 - a0) // 0 at the tail, 1 at the blade
				// the crescent: thick at the blade, tapering to a point at the tail
				in := r1 - (0.6+8.5*t*t)*p.smFade - 1.2
				if r < in {
					continue
				}
				u := (r - in) / (r1 - in) // 0 inner edge, 1 outer
				var c color.RGBA
				switch {
				case u > 0.55 && t > 0.3:
					c = hex(0xffffff)
				case u > 0.25 || t > 0.75:
					c = hex(0xcfeeff)
				default:
					c = hex(0x8cc8f4)
				}
				if p.smFade < 1 || t < 0.35 {
					// thin the tail and the fading arc with a checker
					if (px+py)&1 == 0 && (t < 0.35*p.smFade+0.2 || u < 0.3) {
						continue
					}
				}
				i := py*hdW + px
				f.l.col[i] = c
				f.l.nrm[i] = [3]float32{}
			}
		}
		f.commit(0, false)
	}
	// blade: two pixels wide, a bright edge and a mid tone, a point
	nx, ny := dy, -dx
	bx, by := gx+dx*2.2, gy+dy*2.2
	f.cap(bx, by, 1.25, gx+dx*(blade), gy+dy*(blade), 1.0, hdSteel)
	f.cap(gx+dx*blade, gy+dy*blade, 1.0, gx+dx*(blade+2.2), gy+dy*(blade+2.2), 0.3, hdSteel)
	f.retone(hdSteel, func(ix, ih, t int) int {
		x, y := float64(ix)+0.5-gx, float64(ih)+0.5-gy
		c := x*nx + y*ny
		if c > 0.15 {
			return 3
		}
		if c > -0.7 {
			return 2
		}
		return 1
	})
	f.commit(0.3, true)
	// guard, grip, pommel
	f.cap(gx+dx*1.6+nx*2.6, gy+dy*1.6+ny*2.6, 1.0, gx+dx*1.6-nx*2.6, gy+dy*1.6-ny*2.6, 1.0, hdGold)
	f.commit(0.3, true)
	f.cap(gx-dx*2.2, gy-dy*2.2, 1.1, gx-dx*2.4, gy-dy*2.4, 1.1, hdGold)
	f.commit(0, false)
}

// mantleFringe hangs clumps of fur along a curve from (x0, y0) to (x1, y1),
// sagging by sag: the mantle's lower edge.
func (h *hdHero) mantleFringe(x0, y0, x1, y1, sag float64, n int, lenBase float64) {
	for k := 0; k <= n; k++ {
		t := float64(k) / float64(n)
		x := lerpf(x0, x1, t)
		y := lerpf(y0, y1, t) - sag*4*t*(1-t)
		ln := lenBase + float64(k%2)*0.8 - h.p.mantY*float64((k+1)%2)
		h.f.cap(x, y+0.8, 1.9, x+(t-0.5)*1.6, y-ln, 1.1, hdFur)
	}
}

// front draws the figure facing the viewer, or with its back turned.
func (h *hdHero) front(back bool) {
	f, p := h.f, h.p
	hipY := hdHip + p.bob
	shY := hdShoulder + p.bob + p.chest
	hy := float64(hdHeadI) + 0.5 + p.bob + p.chest + p.headY
	hyI := int(math.Round(hy - 0.5))
	hy = float64(hyI) + 0.5
	handOf := func(k int) (float64, float64) {
		s := float64(2*k - 1)
		return s*7.2 + p.arm[k].ex, shY + p.arm[k].ey
	}
	swHand := 0
	if back {
		swHand = 1
	}
	if p.sword && p.swUnder {
		h.swordAt(handOf(swHand))
	}

	// arms swung behind the body
	for k, s := range [2]float64{-1, 1} {
		if p.armLow[k] {
			f.bias = -0.25
			h.arm(k, s*7.2, shY)
			f.bias = 0
		}
	}
	// legs: the one further from the viewer first
	order := [2]int{0, 1}
	if (p.foot[0] > p.foot[1]) != back {
		order = [2]int{1, 0}
	}
	for n, k := range order {
		s := float64(2*k - 1)
		if n == 0 && p.foot[0] != p.foot[1] {
			f.bias = -0.2
		}
		h.leg(k, s*2.9, hipY, false, back)
		f.bias = 0
	}

	// the tunic: chest, waist and a skirt to mid thigh
	sk := p.mantT * 0.5
	f.poly([]float64{
		-6.6, shY + 1.5, 6.6, shY + 1.5,
		5.6, hipY + 5.2, 6.8 + sk, hipY + 0.4, 4, hipY - 0.3, 0, hipY + 0.3, -4, hipY - 0.3, -6.8 - sk, hipY + 0.4, -5.6, hipY + 5.2,
	}, h.tunic, 0, 7.4, 0.1)
	bh := int(math.Round(hipY)) + 4 // the belt's upper row
	// folds: the skirt's pleats and a crease under the chest
	f.retone(h.tunic, func(ix, ih, t int) int {
		if ih < bh-1 {
			if ix == -4 || ix == 1 || ix == 4 {
				return t - 1
			}
			if (ix == -3 || ix == 2) && t < 3 {
				return t + 1
			}
		}
		if ih > bh+1 && ih < bh+5 && (ix == -3-(ih-bh)/2 || ix == 3+(ih-bh)/3) {
			return t - 1
		}
		return t
	})
	f.band(bh-1, bh, hdLeather)
	gh, gs, gd := hdGold[3], hdGold[2], hdGold[1]
	if !back {
		// the buckle, the strap's loose end, and a pouch on the hip
		f.dots(-1, bh, gs, gh)
		f.dots(-1, bh-1, gd, gs)
		f.dot(1, bh-2, hdLeather[2])
		f.dot(1, bh-3, hdLeather[1])
	}
	f.commit(0.4, true)
	px := 3
	if back {
		px = -6
	}
	for r := 0; r < 4; r++ {
		for c := 0; c < 3; c++ {
			col := hdLeather[2]
			if r == 0 {
				col = hdLeather[3]
			}
			if c == 2 || r == 3 {
				col = hdLeather[1]
			}
			f.dot(px+c, bh-1-r, col)
		}
	}
	f.dot(px+1, bh-2, gs)
	f.commit(0.4, true)

	if back {
		h.neckFront(0, hy)
	}
	// the arms in front
	for k, s := range [2]float64{-1, 1} {
		if !p.armLow[k] && !(p.sword && k == swHand && !p.swUnder) {
			h.arm(k, s*7.2, shY)
		}
	}

	// the mantle: fur over the shoulders, its lower edge in clumps
	my := shY + 0.3
	if back {
		// from behind it is a short cape
		h.mantleFringe(-8.6-p.mantT*0.6, my-4.5, 8.6+p.mantT*0.6, my-4.5, 2.2, 8, 2.4)
		f.ell(0, my-0.6, 9.3+p.mantT*0.3, 5.2, hdFur, nil)
	} else {
		h.mantleFringe(-8.4-p.mantT*0.6, my-2.2, 8.4+p.mantT*0.6, my-2.2, 2.6, 8, 1.8)
		f.ell(0, my+0.2, 9.2+p.mantT*0.3, 4.2, hdFur, nil)
	}
	h.furTex()
	f.commit(0.45, true)
	if !back {
		// the brooch: a gold disc on the wearer's right shoulder
		bx, by := -7, int(math.Round(my))-3
		f.dots(bx, by+1, gs, gh, gs)
		f.dots(bx-1+1, by, gs, gs, gd)
		f.dots(bx, by-1, gd, gd, hdGold[0])
		f.dot(bx+1, by+1, hex(0xffffff))
		f.commit(0.35, true)
	}

	if back {
		h.headBack(0, hy, hyI)
	} else {
		h.headFront(0, hy, hyI)
	}
	if p.sword && !p.swUnder {
		h.arm(swHand, float64(2*swHand-1)*7.2, shY)
		h.swordAt(handOf(swHand))
		// the fist over the grip
		gx, gy := handOf(swHand)
		f.ell(gx, gy, 2.1, 2.1, hdSkin, nil)
		f.commit(0.3, true)
	}
}

func (h *hdHero) neckFront(x, hy float64) {
	h.f.cap(x, hy-7, 2.6, x, hy-9, 2.6, hdSkin)
	h.f.commit(0, false)
}

// lock draws one lock of hair from its root to its tip.
func (h *hdHero) lock(r ramp, bx, by, tx, ty, w float64) {
	h.f.cap(bx, by, w, tx, ty, 0.55, r)
}

// The heads are drawn by hand, a letter a pixel, 18 wide, the top row 11
// above the middle of the head:
//
//	H h d D  hair, light to dark    * its shine
//	S s e E  skin                   p a flushed cheek
//	B b q Q  beard                  m the mouth
//	K W I i  the eye: dark, shine, iris and its shadow
var hdHeadFront = []string{
	"......hh...h......",
	"...hh.hHHh.hHh.hh.",
	"..hHHhHHHHhHHHhHHd",
	".hHH*HhH**HdHHHHhd",
	".hH*HHdHHHHdHH*Hhd",
	"hHHHHhdHHHhDhHHHhd",
	"hHHhhDhHHhdDhHHhdd",
	"hHhhDdhHhdDshHhDdd",
	".hhdDshhdDsSdhdDd.",
	".hdDsSSdDSSSSdsDd.",
	".hdsDDDSSSSDDDsdd.",
	".dsSKKKSSSSKKKSsd.",
	".esSWKKSSSSWKKSse.",
	".eeSIIKSSsSIIKsee.",
	"..qsppSSSeSSppsq..",
	"..qbbBBBsSBBBbbq..",
	"..QbBBbqmmqbBbqQ..",
	"..QqbbbqeeqbbbqQ..",
	"...QbbBbbbbbbqQ...",
	"...QqbBbqbbqbqQ...",
	"....QqbBbbbbqQ....",
	".....QqbBbbqQ.....",
	"......QqbbqQ......",
	".......QqqQ.......",
}

var hdHeadSide = []string{
	".....hh...hh......",
	"..hh.hHHh.hHHh.h..",
	".hHHhHHHHhHHHHhHh.",
	"hHH*HhH**HdHHHHHh.",
	"hH*HHdHHHHdHH*HHhd",
	"hHHHhdHHHhDhHHHHhd",
	"hHHhDhHHhdDhHHdHhd",
	"hHhhDhHhdDhHhDshd.",
	"hhhdDhhdDhhdDsSSd.",
	".hhdhhdDhdDsSSSSS.",
	"hhdDhdDhdsSSSDDDD.",
	".hdDhdDhdsSSSKKKS.",
	"..hdhdDdeSsSSWKKS.",
	"..dhdDddeesSSIIKSs",
	"...dDdqqssppSSSSe.",
	"...dDqbbbsSBBBBs..",
	"....QqbbbBbbbqmm..",
	".....QqbbbbbbbqQ..",
	"......QqbbBbbbbQ..",
	".......QqbbBbbbQ..",
	".........QqbBbqQ..",
	"..........QqbbQ...",
	"...........QqQ....",
}

var hdHeadBack = []string{
	"......hh...h......",
	"...hh.hHHh.hHh.hh.",
	"..hHHhHHHHhHHHhHHd",
	".hHH*HhH**HdHHHHhd",
	".hH*HHdHHHHdHH*Hhd",
	"hHHHHhdHHHhDhHHHhd",
	"hHHhhDhHHhdDhHHhdd",
	"hHhhDdhHhhDdhHhDdd",
	".hHhDdhHhhDdhHhDd.",
	".hHhDdhHhhDdhhhDd.",
	".hhhdDhHhdDhhhdDd.",
	".hhhdDhhhdDhhhDhd.",
	".ehhdDhhhdDhhdDde.",
	".eehdDhhhdDhhdDee.",
	"..dhdDhhhdDhhdDd..",
	"..dhdDhhdDdhhdDd..",
	"...ddDhhdDDhdDd...",
	"...DdDhhdD.hdD....",
	"....DDdhD..dD.....",
	".......dD.........",
}

// stamp draws a head map with its middle column at xi and its middle row
// at hyI. Rows above the brow shift by lift (the hair bouncing), and each
// pixel takes the normal of a ball about the head.
func (h *hdHero) stamp(m []string, xi, hyI int, lift int) {
	f := h.f
	shine := mix(hdHair[3], hex(0xffe9b0), 0.5)
	for r, row := range m {
		ih := hyI + 11 - r
		if r < 3 {
			ih += lift
		}
		for c := 0; c < len(row); c++ {
			ch := row[c]
			var col color.RGBA
			switch ch {
			case 'H', 'h', 'd', 'D':
				col = hdHair[3-indexOf("HhdD", ch)]
			case '*':
				col = shine
			case 'S', 's', 'e', 'E':
				col = hdSkin[3-indexOf("SseE", ch)]
			case 'B', 'b', 'q', 'Q':
				col = hdBeard[3-indexOf("BbqQ", ch)]
			case 'p':
				col = hdBlush
			case 'm':
				col = hdMouth
			case 'K', 'W', 'I', 'i':
				switch {
				case h.p.blink && r == 12:
					col = hdEyeDark
				case h.p.blink:
					col = hdSkin[3]
				case ch == 'K':
					col = hdEyeDark
				case ch == 'W':
					col = hdEyeWht
				case ch == 'I':
					col = hdIris
				default:
					col = hdIrisDk
				}
			default:
				continue
			}
			ix := xi - 9 + c
			nx, ny := (float64(c)-8.5)/10, (float64(11-r)+0.5)/10.5
			if d := nx*nx + ny*ny; d > 0.96 {
				s := math.Sqrt(d / 0.96)
				nx, ny = nx/s, ny/s
			}
			px, py := ix+hdOX, hdOY-1-ih
			f.put(px, py, col, 2, nx, ny, math.Sqrt(math.Max(0, 1-nx*nx-ny*ny)))
		}
	}
}

func indexOf(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
	}
	return 0
}

// braid hangs the beard's braid from (bx, by) to (tx, ty), a gold bead at
// its end.
func (h *hdHero) braid(bx, by, tx, ty float64) {
	f, bd := h.f, hdBeard
	f.capPat(bx, by, 1.4, tx, ty, 1.2, bd, func(a, c float64, tone int) (color.RGBA, bool) {
		if int(a+0.5)%2 == 0 {
			return bd.at(tone + 1), true
		}
		return bd.at(tone - 1), true
	})
	f.commit(0.35, true)
	bi, bj := int(math.Floor(tx-1)), int(math.Round(ty-1.5))
	f.dots(bi, bj, hdGold[3], hdGold[2])
	f.dots(bi, bj-1, hdGold[2], hdGold[1])
	f.dot(bi, bj-2, bd[1])
	f.dot(bi+1, bj-2, bd[2])
	f.commit(0.35, true)
}

// headFront draws the head facing the viewer, centred on column x.
func (h *hdHero) headFront(x, hy float64, hyI int) {
	p := h.p
	h.stamp(hdHeadFront, int(x), hyI, int(math.Round(p.hairY)))
	h.f.commit(0.3, true)
	h.braid(x+p.braidX*0.3, hy-12.4, x+p.braidX, hy-15.4+p.braidY)
}

// headBack draws the head from behind: all hair, in locks to the nape.
func (h *hdHero) headBack(x, hy float64, hyI int) {
	h.stamp(hdHeadBack, int(x), hyI, int(math.Round(h.p.hairY)))
	h.f.commit(0.3, true)
}

// side draws the figure facing right.
func (h *hdHero) side() {
	f, p := h.f, h.p
	hipY := hdHip + p.bob
	shY := hdShoulder + p.bob + p.chest
	shear := func(y float64) float64 { return p.lean * math.Max(0, y-hipY) / (hdShoulder - hdHip) }
	shX := shear(shY)
	hx := math.Round(shear(shY+9.5)*0.9 + p.headX + 0.8)
	hy := float64(hdHeadI) + 0.5 + p.bob + p.chest + p.headY
	hyI := int(math.Round(hy - 0.5))
	hy = float64(hyI) + 0.5

	// the far arm and leg, a tone down
	f.bias = -0.3
	h.arm(1, shX+0.8, shY-0.2)
	h.leg(1, -0.5, hipY, true, false)
	f.bias = 0

	// the mantle's tail, streaming behind
	tx, ty := shX-5.4-p.mantT*1.5, shY-6.6+p.mantY+p.mantT*1.2
	f.cap(shX-3, shY-0.4, 3.8, tx, ty, 2.7, hdFur)
	f.cap(tx+0.2, ty-0.6, 2.0, tx-1.4-p.mantT*0.4, ty-3-p.mantY*0.5+p.mantT*0.3, 1.2, hdFur)
	f.cap(tx+2.4, ty-1.4, 1.8, tx+1.8-p.mantT*0.2, ty-3.6, 1.2, hdFur)
	h.furTex()
	f.commit(0.4, true)

	h.leg(0, 0.5, hipY, true, false)

	// the tunic
	sk := p.mantT * 0.55
	f.poly([]float64{
		shX - 4.6, shY + 1.5, shX + 4.4, shY + 1.5, shX + 5, shY - 3,
		shear(hipY+5) + 4.2, hipY + 5, 5.2, hipY + 0.2, 1, hipY - 0.4, -3, hipY + 0.4, -5.4 - sk, hipY + 0.8 + sk*0.4,
		shear(hipY+5) - 4.2, hipY + 5,
	}, h.tunic, shX*0.5, 5.6, 0.1)
	bh := int(math.Round(hipY)) + 4
	f.retone(h.tunic, func(ix, ih, t int) int {
		if ih < bh-1 && (ix == -2 || ix == 2) {
			return t - 1
		}
		if ih < bh-1 && ix == 0 && t < 3 {
			return t + 1
		}
		return t
	})
	f.band(bh-1, bh, hdLeather)
	bxf := 0
	for ix := 12; ix > -12; ix-- {
		if f.has(ix, bh) {
			bxf = ix
			break
		}
	}
	f.dots(bxf-1, bh, hdGold[2], hdGold[3])
	f.dots(bxf-1, bh-1, hdGold[1], hdGold[2])
	f.commit(0.4, true)
	// the pouch, behind the hip
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			col := hdLeather[2]
			if r == 0 {
				col = hdLeather[3]
			}
			if c == 3 || r == 3 {
				col = hdLeather[1]
			}
			f.dot(-4+c, bh-1-r, col)
		}
	}
	f.dot(-3, bh-2, hdGold[2])
	f.commit(0.4, true)

	sword := func() {
		if p.sword {
			h.swordAt(shX-0.6+p.arm[0].ex, shY+p.arm[0].ey)
		}
	}
	if p.sword && p.swAng < -0.3 {
		sword() // raised behind: under the arm and the head
	}

	// the neck, and the mantle over the shoulders
	f.cap(hx-1.5, hy-6.5, 2.4, shX, shY+1, 2.6, hdSkin)
	f.commit(0, false)
	h.mantleFringe(shX-6.2, shY-2.2, shX+5.6, shY-2.4, 1.6, 6, 1.7)
	f.ell(shX-0.4, shY+0.6, 6.9, 3.9, hdFur, nil)
	h.furTex()
	f.commit(0.45, true)
	// the brooch at the breast
	bx, by := int(math.Round(shX))+3, int(math.Round(shY))-2
	f.dots(bx, by+1, hdGold[2], hdGold[3])
	f.dots(bx, by, hdGold[1], hdGold[2])
	f.dot(bx, by+1, hex(0xffffff))
	f.commit(0.35, true)

	lowArm := p.sword && p.swAng < -0.3
	if !lowArm {
		h.headSide(hx, hy, hyI)
	}
	// the near arm
	h.arm(0, shX-0.6, shY)
	if p.sword && !(p.swAng < -0.3) {
		sword()
		gx, gy := shX-0.6+p.arm[0].ex, shY+p.arm[0].ey
		f.ell(gx, gy, 2.1, 2.1, hdSkin, nil)
		f.commit(0.3, true)
	}
	if lowArm {
		h.headSide(hx, hy, hyI)
	}
}

// headSide draws the head in profile, facing right, centred near column x.
func (h *hdHero) headSide(x, hy float64, hyI int) {
	f, p := h.f, h.p
	hr := hdHair
	t, ty := p.hairT, p.hairY
	// locks streaming off the back of the head, behind it
	if t > 0.5 {
		f.cap(x-6.5, hy-1, 2.3, x-8.6-t*1.1, hy-2.6+ty*0.6, 1.0, hr)
		f.commit(0.3, true)
		f.cap(x-6, hy+3.6, 2.5, x-9.4-t*1.4, hy+3.8+ty, 1.0, hr)
		f.commit(0.3, true)
	}
	h.stamp(hdHeadSide, int(x), hyI, int(math.Round(ty)))
	f.commit(0.3, true)
	h.braid(x+4.6+p.braidX*0.3, hy-11.4, x+4.8+p.braidX, hy-14.6+p.braidY)
}
