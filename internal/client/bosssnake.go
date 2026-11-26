package client

import (
	"image/color"
	"math"
)

// Titanoboa, drawn at HDDensity: a monstrous serpent of the deep jungle,
// thicker than a man is tall and near thirty tiles long. Its body is laid
// out in the world (x right, y along the ground towards the eye, z up) and
// drawn as a chain of spheres into a depth buffer, so that coils, spines and
// fins hide one another as they should; every pixel of it knows where it
// lies round the body, and is coloured from that: olive and moss-green
// scales, saddles of dark bronze down the back, pale ridged plates under
// the belly, each scale lit on its own from the upper left, a cool rim on
// the far edges, and the shadows the coils and spines cast on what is
// below them. A crest of tall curved blades runs the length of its back
// and hooked fins fringe its flanks. Its head is a wedge of armoured
// plates under a heavy brow, crowned with a fan of horns that sweep back,
// its eyes burning in deep sockets, its jaws lined with rows of fangs.
//
// Its frames: under way in waves that run down its body from the head;
// asleep in a pile of coils; reared high to roar, its coils trailing far
// behind; drawn back and striking; and wound round its prey.

var (
	tbMoss   = mkRamp(0x090d08, 0x10170e, 0x172013, 0x1f2a17, 0x29361b, 0x344420, 0x405226, 0x4e612d, 0x5f7236, 0x768745, 0x93a05a)
	tbBronze = mkRamp(0x0b0705, 0x140d07, 0x1f140b, 0x2c1d0f, 0x3b2713, 0x4d3318, 0x62421e, 0x7a5326, 0x966731, 0xb4803f, 0xd29e55)
	tbBelly  = mkRamp(0x1c1a14, 0x2d2a1f, 0x423e2c, 0x5a553c, 0x746d4e, 0x8f8762, 0xa9a078, 0xc1b98f, 0xd8d1ab)
	tbBone   = mkRamp(0x110d09, 0x211a12, 0x34291d, 0x4b3d2a, 0x66553b, 0x84714f, 0xa18e67, 0xbcab83, 0xd5c7a1, 0xece2c4)
	tbRimCol = hex(0xa9d2bd) // the cool light of the forest on its far edges
	tbInk    = hex(0x0a0c08)
)

const (
	tbLen   = 660.0 // art pixels of body, from the tip of the tail to the back of the skull
	tbStep  = 0.5   // art pixels between the spheres it is drawn as
	tbRMax  = 19.5  // its half width at the thickest
	tbSq    = 0.6   // the ground seen from above: a depth along it is drawn this short...
	tbUp    = 0.8   // ...and a height this short
	tbHeadK = 1.4   // art pixels to a unit of the head's drawing
)

// vec3 is a direction or a point: in the world (x right, y along the
// ground towards the eye, z up) or in the picture (x right, y down, z out
// towards the eye).
type vec3 struct{ x, y, z float64 }

func (a vec3) add(b vec3) vec3    { return vec3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a vec3) sub(b vec3) vec3    { return vec3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec3) mul(k float64) vec3 { return vec3{a.x * k, a.y * k, a.z * k} }
func (a vec3) dot(b vec3) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec3) cross(b vec3) vec3 {
	return vec3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}
func (a vec3) norm() float64 { return math.Sqrt(a.dot(a)) }
func (a vec3) unit() vec3 {
	if l := a.norm(); l > 1e-12 {
		return a.mul(1 / l)
	}
	return a
}
func lerp3(a, b vec3, t float64) vec3 { return a.add(b.sub(a).mul(t)) }

// tbScr turns a direction in the world into the picture's.
func tbScr(v vec3) vec3 { return vec3{v.x, tbSq*v.y - tbUp*v.z, tbUp*v.y + tbSq*v.z} }

// tbProj is where world point p is drawn, in art pixels from the anchor.
func tbProj(p vec3) (float64, float64) { return p.x, tbSq*p.y - tbUp*p.z }

var (
	tbL   = vec3{lightDir[0], lightDir[1], lightDir[2]}
	tbHv  = vec3{lightDir[0], lightDir[1], lightDir[2] + 1}.unit() // half way to the eye: the glints
	tbUpS = tbScr(vec3{0, 0, 1})                                   // up, in the picture
)

// tbTone is ramp r at t (0..1), blended between its colours.
func tbTone(r ramp, t float64) color.RGBA {
	p := clamp01(t) * float64(len(r)-1)
	i := min(int(p), len(r)-2)
	return mix(r[i], r[i+1], p-float64(i))
}

// tbRadius is the body's half width at s: 0 the tip of the tail, 1 the neck.
func tbRadius(s float64) float64 {
	return 2.2 + 17.3*smooth(clamp01(s/0.3)) - 3.6*smooth(clamp01((s-0.8)/0.2))
}

// --- the body ---------------------------------------------------------------

// tbSample is one of the spheres the body is drawn as.
type tbSample struct {
	w        vec3    // its middle, in the world
	r, s     float64 // its half width; and 0 at the tip of the tail, 1 at the neck
	arc      float64 // art pixels from the tip of the tail
	lift     float64 // how far its underside is off the ground
	sx, sy   float64 // its middle in the picture, art pixels from the anchor
	z        float64 // how near the eye
	t2x, t2y float64 // the way it runs in the picture
	tan, n3  vec3    // in the picture: along it, and across it (to its right, flat in the picture)
	b3, dor  vec3    // in the picture: out of it towards the eye; and the way its back faces
	phiD     float64 // the angle of its back about it, from n3 towards b3
	su, sp   float64 // how many scales, and how many belly plates, from the tip of the tail
}

// tbBody lays the last tbLen art pixels of path (tail first, a point every
// art pixel or closer) out as the body; swell fattens it as it breathes.
func tbBody(path []vec3, swell float64) []tbSample {
	cum := make([]float64, len(path))
	for i := 1; i < len(path); i++ {
		cum[i] = cum[i-1] + path[i].sub(path[i-1]).norm()
	}
	total := cum[len(path)-1]
	n := int(tbLen / tbStep)
	pts := make([]tbSample, n+1)
	j := 0
	for i := range pts {
		at := total - tbLen + float64(i)*tbStep
		var p vec3
		if at <= 0 { // the path is too short: the tail goes on straight
			p = path[0].add(path[1].sub(path[0]).unit().mul(at))
		} else {
			for j < len(path)-2 && cum[j+1] < at {
				j++
			}
			p = lerp3(path[j], path[j+1], clamp01((at-cum[j])/max(1e-9, cum[j+1]-cum[j])))
		}
		s := float64(i) / float64(n)
		pts[i] = tbSample{w: p, s: s, arc: float64(i) * tbStep, r: tbRadius(s) * (1 + swell*math.Sin(s*math.Pi))}
		pts[i].w.z += pts[i].r
	}
	// Its back faces up where it starts, and is carried along it from there
	// without twisting: where it rears, its back turns away with it.
	var dor vec3
	for i := range pts {
		q := &pts[i]
		a, b := pts[max(0, i-2)].w, pts[min(n, i+2)].w
		T := b.sub(a).unit()
		if i == 0 {
			dor = vec3{0, 0, 1}
		}
		dor = dor.sub(T.mul(dor.dot(T))).unit()
		// where it lies level its back turns up again, as a snake's does
		if lv := 1 - math.Abs(T.z); lv > 0 {
			up := vec3{0, 0, 1}.sub(T.mul(T.z)).unit()
			dor = dor.add(up.sub(dor).mul(0.08 * lv * lv)).unit()
		}
		if i > 0 {
			sz := 0.6 + 0.4*q.r/tbRMax
			q.su = pts[i-1].su + tbStep/(6.2*sz)
			q.sp = pts[i-1].sp + tbStep/(4.6*sz)
		}
		q.lift = q.w.z - q.r
		q.sx, q.sy = tbProj(q.w)
		q.z = tbUp*q.w.y + tbSq*q.w.z
		q.tan = tbScr(T)
		ax, ay := tbProj(a)
		bx, by := tbProj(b)
		if l := math.Hypot(bx-ax, by-ay); l > 1e-6 {
			q.t2x, q.t2y = (bx-ax)/l, (by-ay)/l
		} else if i > 0 {
			q.t2x, q.t2y = pts[i-1].t2x, pts[i-1].t2y
		} else {
			q.t2x = 1
		}
		q.n3 = vec3{-q.t2y, q.t2x, 0}
		q.b3 = vec3{0, 0, 1}.sub(q.tan.mul(q.tan.z)).unit()
		q.dor = tbScr(dor)
		q.phiD = math.Atan2(q.dor.dot(q.b3), q.dor.dot(q.n3))
	}
	return pts
}

// tbSpline is a Catmull-Rom curve through ctrl, a point about every art
// pixel.
func tbSpline(ctrl []vec3) []vec3 {
	var out []vec3
	for i := 0; i+1 < len(ctrl); i++ {
		p0, p1, p2, p3 := ctrl[max(0, i-1)], ctrl[i], ctrl[i+1], ctrl[min(len(ctrl)-1, i+2)]
		m := max(2, int(p2.sub(p1).norm()))
		for k := 0; k < m; k++ {
			t := float64(k) / float64(m)
			t2, t3 := t*t, t*t*t
			w0 := -0.5*t3 + t2 - 0.5*t
			w1 := 1.5*t3 - 2.5*t2 + 1
			w2 := -1.5*t3 + 2*t2 + 0.5*t
			w3 := 0.5*t3 - 0.5*t2
			out = append(out, p0.mul(w0).add(p1.mul(w1)).add(p2.mul(w2)).add(p3.mul(w3)))
		}
	}
	return append(out, ctrl[len(ctrl)-1])
}

// --- blades: the spines, the fins and the horns -----------------------------

const tbBladeN = 20

type tbBladeKind int

const (
	tbSpine tbBladeKind = iota
	tbFin
	tbHornK
)

// tbBlade is a curved, tapering blade in the picture, in image pixels: its
// centre line from root to tip, and how it is lit on its two faces.
type tbBlade struct {
	x, y, dx, dy, w, seg [tbBladeN + 1]float64
	z0, z1               float64
	nL, nT               vec3 // the normals of the face towards the head and of the one towards the tail
	side                 float64
	kind                 tbBladeKind
	tone                 float64
}

// newBlade is a blade rooted at (rx, ry) (image pixels) at depth rz, along
// axis (in the picture's space) for h image pixels, curling back towards
// -fwd at its tip by curl, w0 half as wide at the root.
func newBlade(rx, ry, rz float64, axis, fwd vec3, h, w0, curl float64, kind tbBladeKind, tone float64) *tbBlade {
	b := &tbBlade{kind: kind, tone: tone}
	ex, ey := rx+axis.x*h-fwd.x*h*curl, ry+axis.y*h-fwd.y*h*curl
	mx, my := rx+axis.x*h*0.55, ry+axis.y*h*0.55
	for i := 0; i <= tbBladeN; i++ {
		u := float64(i) / tbBladeN
		a, c, d := (1-u)*(1-u), 2*u*(1-u), u*u
		b.x[i], b.y[i] = a*rx+c*mx+d*ex, a*ry+c*my+d*ey
		b.w[i] = w0 * math.Pow(1-u, 0.9) * (1 + 0.18*math.Sin(u*math.Pi))
	}
	for i := 0; i <= tbBladeN; i++ {
		a, c := max(0, i-1), min(tbBladeN, i+1)
		dx, dy := b.x[c]-b.x[a], b.y[c]-b.y[a]
		l := math.Max(1e-6, math.Hypot(dx, dy))
		b.dx[i], b.dy[i] = dx/l, dy/l
		if i < tbBladeN {
			b.seg[i] = math.Hypot(b.x[i+1]-b.x[i], b.y[i+1]-b.y[i])
		}
	}
	b.z0, b.z1 = rz, rz+axis.z*h-fwd.z*h*curl
	F := axis.cross(fwd).unit()
	if F.z < 0 {
		F = F.mul(-1)
	}
	G := fwd.sub(axis.mul(fwd.dot(axis))).unit()
	b.nL, b.nT = F.add(G.mul(1.3)).unit(), F.sub(G.mul(1.3)).unit()
	if (-b.dy[0])*fwd.x+b.dx[0]*fwd.y > 0 {
		b.side = 1
	} else {
		b.side = -1
	}
	return b
}

// raster calls f for each pixel of the blade with how far up it (0..1) and
// across it (-1..1) the pixel is.
func (b *tbBlade) raster(w, h int, f func(x, y int, u, sg float64)) {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i := 0; i <= tbBladeN; i++ {
		x0, x1 = min(x0, b.x[i]-b.w[i]), max(x1, b.x[i]+b.w[i])
		y0, y1 = min(y0, b.y[i]-b.w[i]), max(y1, b.y[i]+b.w[i])
	}
	for y := max(0, int(y0)-1); y <= min(h-1, int(y1)+1); y++ {
		for x := max(0, int(x0)-1); x <= min(w-1, int(x1)+1); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			best, bj := math.Inf(1), 0
			for i := 0; i <= tbBladeN; i++ {
				if d := (px-b.x[i])*(px-b.x[i]) + (py-b.y[i])*(py-b.y[i]); d < best {
					best, bj = d, i
				}
			}
			ex, ey := px-b.x[bj], py-b.y[bj]
			al := ex*b.dx[bj] + ey*b.dy[bj]
			seg := b.seg[min(bj, tbBladeN-1)]
			if al < 0 && bj > 0 {
				seg = b.seg[bj-1]
			}
			u := (float64(bj) + al/math.Max(seg, 1e-6)) / tbBladeN
			if u < 0 || u > 1 {
				continue
			}
			i := min(tbBladeN-1, int(u*tbBladeN))
			fu := u*tbBladeN - float64(i)
			wd := b.w[i]*(1-fu) + b.w[i+1]*fu
			sg := (-ex*b.dy[bj] + ey*b.dx[bj]) / math.Max(wd, 1e-6)
			if math.Abs(sg) <= 1 && wd > 0.35 {
				f(x, y, u, sg)
			}
		}
	}
}

// colour is a blade's pixel u up it and sg across it, lit; shade (0..1)
// is how much of it is in a shadow.
func (b *tbBlade) colour(x, y int, u, sg, shade float64) color.RGBA {
	lead := sg*b.side > 0
	n := b.nT
	if lead {
		n = b.nL
	}
	a := math.Abs(sg)
	// Each face is lit as it turns to the light in the picture: the one
	// towards the upper left bright, the other dark, the ridge between.
	j := min(tbBladeN, int(u*tbBladeN+0.5))
	px, py := -b.dy[j], b.dx[j]
	if sg < 0 {
		px, py = -px, -py
	}
	face := (px*tbL.x + py*tbL.y) / math.Hypot(tbL.x, tbL.y)
	l := (0.3*clamp01((n.dot(tbL)+0.3)/1.3) + 0.7*clamp01(0.42+0.5*face)) * (1 - 0.6*shade)
	l *= 0.5 + 0.5*smooth(clamp01(u/0.32)) // where it grows out of the hide
	if a > 0.78 {
		l -= 0.12 * (a - 0.78) / 0.22 // its edges turn away
	}
	if a < 0.09 { // the ridge between its faces catches the light
		l += 0.14 * (1 - a/0.09) * (1 - shade)
	}
	l += b.tone
	var col color.RGBA
	switch b.kind {
	case tbFin:
		col = tbTone(tbBronze, 0.04+0.62*l)
		if u > 0.5 {
			col = mix(col, tbTone(tbBone, 0.06+0.6*l), smooth((u-0.5)/0.4)*0.55)
		}
	default:
		// bronze where it grows, weathered bone up it, dark at the point
		bone := tbTone(tbBone, 0.08+0.8*l)
		base := tbTone(tbBronze, 0.04+0.75*l)
		col = mix(base, bone, smooth(clamp01((u-0.08)/0.4)))
		if u < 0.5 { // the rings it grew in
			if f := u * 15; f-math.Floor(f) < 0.2 {
				col = mix(col, tbInk, 0.22*(1-u*1.6))
			}
		}
		if u > 0.72 {
			col = mix(col, tbTone(tbBronze, 0.05+0.4*l), smooth((u-0.72)/0.28)*0.75)
		}
	}
	if sp := math.Pow(max(0, n.dot(tbHv)), 16) * (1 - shade); sp > 0.02 {
		col = mix(col, hex(0xf6ecd0), sp*0.5)
	}
	return col
}

// --- drawing a pose -------------------------------------------------------------

// tbPose is one frame of the snake.
type tbPose struct {
	path   []vec3  // its centre line in the world, tail first, at least tbLen long
	look   float64 // the head turned from the line of the neck (radians; + looks down)
	gape   float64 // the jaws open, 0..1
	tongue float64 // the tongue out, 0..1
	swell  float64 // fatter, breathing in
	lid    float64 // its eyes narrowed, 0..1
	spread float64 // its crown of horns raised, 0..1
}

// tbRender is a frame being drawn.
type tbRender struct {
	c      *canvas
	k      float64
	ox, oy float64 // the anchor on the canvas, art pixels
	pts    []tbSample
	zb     []float32
	own    []int32 // nearest the eye at each pixel: -1 nothing, a sphere of the body, or -2-i blade i
	pa, pb []float32
	blades []*tbBlade
}

func (p tbPose) canvas() (*canvas, float64, float64) {
	pts := tbBody(p.path, p.swell)
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, q := range pts {
		m := q.r*1.9 + 6
		x0, x1 = min(x0, q.sx-m), max(x1, q.sx+m)
		y0, y1 = min(y0, q.sy-m), max(y1, q.sy+m)
	}
	nk := pts[len(pts)-1]
	x0, x1 = min(x0, nk.sx-80), max(x1, nk.sx+80)
	y0, y1 = min(y0, nk.sy-80), max(y1, nk.sy+60)
	ox, oy := -math.Floor(x0), -math.Floor(y0)
	W, H := int(math.Ceil(x1-x0))+2, int(math.Ceil(y1-y0))+2
	rd := &tbRender{c: hdBossCanvas(W, H), k: float64(HDDensity), ox: ox, oy: oy, pts: pts}
	c := rd.c
	rd.zb = make([]float32, c.w*c.h)
	rd.own = make([]int32, c.w*c.h)
	rd.pa, rd.pb = make([]float32, c.w*c.h), make([]float32, c.w*c.h)
	for i := range rd.zb {
		rd.zb[i], rd.own[i] = -1e9, -1
	}
	hd := rd.headFrames(p)
	rd.stampBody()
	rd.addCrest()
	rd.addHorns(hd, p, false)
	for i, b := range rd.blades {
		rd.stampBlade(b, int32(-2-i))
	}
	rd.shade()
	rd.head(hd, p)
	rd.addHorns(hd, p, true) // the horns of its near side, over its head
	rd.eyeGlow(hd, p, false)
	c.outline(0.8)
	rd.eyeGlow(hd, p, true)
	sh := hdBossCanvas(W, H)
	rd.groundShadow(sh)
	for i, q := range c.px {
		if q.A > 0 {
			sh.px[i] = q
		}
	}
	return hdCrop(sh, ox, oy)
}

// stampBody draws the spheres of the body into the depth buffer.
func (rd *tbRender) stampBody() {
	c, k := rd.c, rd.k
	for i := range rd.pts {
		q := &rd.pts[i]
		cx, cy, R, zc := (q.sx+rd.ox)*k, (q.sy+rd.oy)*k, q.r*k, q.z*k
		for y := max(0, int(cy-R)); y <= min(c.h-1, int(cy+R)); y++ {
			dy := float64(y) + 0.5 - cy
			for x := max(0, int(cx-R)); x <= min(c.w-1, int(cx+R)); x++ {
				dx := float64(x) + 0.5 - cx
				d2 := dx*dx + dy*dy
				if d2 > R*R {
					continue
				}
				z := zc + math.Sqrt(R*R-d2)
				o := y*c.w + x
				if float32(z) > rd.zb[o] {
					rd.zb[o], rd.own[o] = float32(z), int32(i)
					rd.pa[o] = float32((dx*q.n3.x + dy*q.n3.y) / R)
					rd.pb[o] = float32((dx*q.t2x + dy*q.t2y) / k)
				}
			}
		}
	}
}

// stampBlade draws a blade into the depth buffer.
func (rd *tbRender) stampBlade(b *tbBlade, id int32) {
	c := rd.c
	b.raster(c.w, c.h, func(x, y int, u, sg float64) {
		o := y*c.w + x
		if z := float32(b.z0 + (b.z1-b.z0)*u); z > rd.zb[o] {
			rd.zb[o], rd.own[o], rd.pa[o], rd.pb[o] = z, id, float32(u), float32(sg)
		}
	})
}

// surf is the point of the body's surface at angle phi about sample q
// (from its back), in image pixels, and its depth and normal there.
func (rd *tbRender) surf(q *tbSample, phi, in float64) (float64, float64, float64, vec3) {
	a := q.phiD + phi
	n := q.n3.mul(math.Cos(a)).add(q.b3.mul(math.Sin(a)))
	r := q.r * in
	return (q.sx + rd.ox + n.x*r) * rd.k, (q.sy + rd.oy + n.y*r) * rd.k, (q.z + n.z*r) * rd.k, n
}

// addCrest sets the spines along its back and the fins down its flanks.
func (rd *tbRender) addCrest() {
	pts := rd.pts
	k := rd.k
	arc := 0.0
	for i := range pts {
		q := &pts[i]
		if q.s < 0.07 || q.s > 0.975 {
			continue
		}
		gap := 9 + 6.5*q.r/tbRMax
		if q.arc < arc {
			continue
		}
		arc = q.arc + gap
		h := q.r * 1.75 * smooth(clamp01((q.s-0.05)/0.3)) * (1 - 0.5*smooth(clamp01((q.s-0.86)/0.12))) * (0.68 + 0.32*smooth(clamp01((q.s-0.3)/0.4)))
		j := int(q.arc / gap)
		h *= 0.82 + 0.36*hashf(j, 0, 961)
		if h < 1.2 {
			continue
		}
		x, y, z, _ := rd.surf(q, 0, 0.82)
		sw := 0.5
		axis := q.dor.mul(math.Cos(sw)).sub(q.tan.unit().mul(math.Sin(sw))).unit()
		rd.blades = append(rd.blades, newBlade(x, y, z, axis, q.tan.unit(), h*k, gap*0.46*k, 0.5, tbSpine, (hashf(j, 1, 961)-0.5)*0.12))
	}
	// The fins: hooked spikes swept back along each flank.
	for _, side := range []float64{-1, 1} {
		arc = 0
		for i := range pts {
			q := &pts[i]
			if q.s < 0.14 || q.s > 0.93 || q.arc < arc {
				continue
			}
			gap := 5.5 + 3.5*q.r/tbRMax
			arc = q.arc + gap
			j := int(q.arc / gap)
			x, y, z, out := rd.surf(q, side*0.5*math.Pi, 0.92)
			h := q.r * 0.75 * (0.75 + 0.5*hashf(j, 2, 962)) * smooth(clamp01((q.s-0.12)/0.2))
			h *= 1 - smooth(clamp01((out.z-0.5)/0.35)) // one turned to the eye would lie flat along it
			if h < 1 {
				continue
			}
			axis := out.mul(0.55).sub(q.tan.unit().mul(0.82)).sub(q.dor.mul(0.12)).unit()
			rd.blades = append(rd.blades, newBlade(x, y, z, axis, q.tan.unit(), h*k, gap*0.55*k, 0.3, tbFin, (hashf(j, 3, 962)-0.5)*0.08))
		}
	}
}

// shadowAt is how much of a shadow pixel o lies in: what is nearer the
// eye between it and the light, not the body's own stretch at that point.
func (rd *tbRender) shadowAt(x, y int, z float32, arc float64) float64 {
	c := rd.c
	lx, ly := tbL.x, tbL.y
	l2 := math.Hypot(lx, ly)
	lx, ly = lx/l2, ly/l2
	rise := tbL.z / l2
	const reach = 42.0
	for s := 3.0; s < reach; s += 1.5 {
		px, py := int(float64(x)+0.5+lx*s), int(float64(y)+0.5+ly*s)
		if !c.in(px, py) {
			return 0
		}
		o := py*c.w + px
		if rd.zb[o] > z+float32(s*rise)+2 {
			if w := rd.own[o]; w >= 0 && math.Abs(rd.pts[w].arc-arc) < rd.pts[w].r*3.2 {
				continue // its own body, which its light already shades
			}
			return 1 - s/reach
		}
	}
	return 0
}

// shade colours every pixel of the body and of the blades in it.
func (rd *tbRender) shade() {
	c := rd.c
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			o := y*c.w + x
			w := rd.own[o]
			switch {
			case w >= 0:
				q := &rd.pts[w]
				sh := rd.shadowAt(x, y, rd.zb[o], q.arc+float64(rd.pb[o]))
				c.px[o] = rd.skin(x, y, q, float64(rd.pa[o]), float64(rd.pb[o]), sh)
			case w <= -2:
				b := rd.blades[-2-w]
				sh := rd.shadowAt(x, y, rd.zb[o], -1000)
				c.px[o] = b.colour(x, y, float64(rd.pa[o]), float64(rd.pb[o]), sh*0.8)
			}
		}
	}
}

// skin colours the body's pixel (x, y): t across the sphere q it belongs
// to, along art pixels ahead of its middle, sh in a shadow.
func (rd *tbRender) skin(x, y int, q *tbSample, t, along, sh float64) color.RGBA {
	t = max(-1, min(1, t))
	st := math.Sqrt(1 - t*t)
	n := q.n3.mul(t).add(q.b3.mul(st))
	round := q.b3.mul(t).sub(q.n3.mul(st)) // the way round the body
	tan := q.tan.unit()
	d := math.Atan2(st, t) - q.phiD
	d -= 2 * math.Pi * math.Floor((d+math.Pi)/(2*math.Pi))
	ad := math.Abs(d) / math.Pi // 0 along the ridge of its back, 1 down the middle of its belly
	b := d * q.r                // art pixels round from the ridge
	a := q.arc + along          // art pixels from the tip of the tail
	sz := 0.6 + 0.4*q.r/tbRMax

	var rp ramp
	tone, seam := 0.0, 0.0
	var bump vec3
	spec := hex(0xc9c27a)
	wob := (vnoise(a*0.2, b*0.2, 941) - 0.5) * 0.09
	switch {
	case ad > 0.68+wob: // the plates of its belly, ridged across
		f := q.sp + along/(4.6*sz)
		ff := f - math.Floor(f)
		bump = tan.mul(math.Cos(ff*math.Pi) * 0.32)
		if ff < 0.09 {
			seam = 0.3 * (1 - ff/0.09)
		} else if ff < 0.3 {
			tone += 0.06 * smooth((0.3-ff)/0.2) // the raised back edge of a plate
		}
		if ad < 0.76+wob { // their ends, where they meet the scales
			seam += 0.08
		}
		rp, spec = tbBelly, hex(0xfff6dc)
		tone = 0.17 + (hashf(int(math.Floor(f)), 0, 942)-0.5)*0.05
	default:
		// Scales: a diamond lattice wrapped round it, each scale a low dome
		// raised at its back edge, lit on its own.
		sc := q.su + along/(6.2*sz)
		pu, pv := sc+b/(5.2*sz), sc-b/(5.2*sz)
		iu, iv := math.Floor(pu), math.Floor(pv)
		fu, fv := pu-iu, pv-iv
		ca, cb := fu+fv-1, fu-fv
		kk := 0.26
		if ca < 0 {
			kk = 0.42
		}
		bump = tan.mul(ca * kk).add(round.mul(cb * 0.32))
		if e := min(fu, fv); e < 0.075 {
			seam = 0.3 * (1 - e/0.075)
		} else if max(fu, fv) > 0.95 {
			seam = 0.05
		}
		hs := hashf(int(iu), int(iv), 943)
		tone = (hs - 0.5) * 0.07
		rp = tbMoss
		// Saddles of bronze across the back, at their ends reaching down
		// the flanks; between them, bronze blotches low on each side.
		P := 30 * sz
		sd := q.su * 6.2 / 30 // saddles from the tail, one every 30 art pixels at its thickest
		sd += along / P
		ki := math.Floor(sd + 0.5)
		jit := hashf(int(ki), 0, 951)
		da := (sd-ki-(jit-0.5)*0.16)*P + (vnoise(a*0.2, b*0.2, 952)-0.5)*3.5
		// a diamond astride the ridge, rimmed dark and edged in pale gold
		ds := math.Abs(da)/(0.4*P*(0.9+0.2*jit)) + ad/0.44 + (vnoise(a*0.35, b*0.35, 954)-0.5)*0.1
		bronze := false
		switch {
		case ds < 1:
			bronze = true
			switch {
			case ds > 0.76:
				tone -= 0.27
			case ds < 0.42:
				tone += 0.07 // its middle, worn bright
			}
		case ds < 1.15:
			tone += 0.1
		default:
			db := math.Abs(math.Abs(d)/math.Pi - 0.56)
			if da2 := math.Abs(math.Abs(da) - P*0.5); math.Hypot(da2/(5.5*sz), db/0.07) < 1 {
				bronze, tone = true, tone-0.1
			}
		}
		// and a few scales of bronze among the green, near the saddles
		if !bronze && hs < 0.07*clamp01(2-ds) {
			bronze, tone = true, tone-0.04
		}
		if bronze {
			rp, spec = tbBronze, hex(0xf2cc7a)
		}
		// moss and lichen grown on the old scales of its back
		if ad < 0.36 {
			if m := vnoise(a*0.09, b*0.12, 953); m > 0.68 {
				tone += (m - 0.68) * 0.9
				if rp[0] == tbBronze[0] {
					rp = tbMoss
				}
			}
		}
	}
	ns := n.add(bump).unit()
	l := clamp01((ns.dot(tbL)+0.14)/1.14) + 0.08*n.dot(tbUpS) // the sun, and the sky above
	// What faces the ground near it is in the dark under the body.
	if low := clamp01(1 - q.lift/(q.r*1.4)); low > 0 {
		l *= 1 - 0.55*clamp01(-n.dot(tbUpS)*1.3)*low
	}
	l *= 1 - 0.62*sh
	col := tbTone(rp, 0.03+0.7*l+tone-seam)
	if g := math.Pow(max(0, ns.dot(tbHv)), 34) * (1 - sh) * (1 - seam*3); g > 0.01 {
		col = mix(col, spec, g*0.6)
	}
	rim := math.Pow(clamp01(1-n.z), 2) * clamp01(n.x*0.72+n.y*0.7) * (1 - sh*0.7)
	if rim > 0.01 {
		col = mix(col, tbRimCol, rim*0.62)
	}
	return col
}

// groundShadow lays the body's shadow on the ground under it: dark where
// it lies, fading under what is raised.
func (rd *tbRender) groundShadow(sh *canvas) {
	k := rd.k
	for i := 0; i < len(rd.pts); i += 4 {
		q := &rd.pts[i]
		h := q.w.z
		fade := clamp01(1 - h/120)
		if fade <= 0 {
			continue
		}
		gx, gy := q.w.x+0.3*h+1.5, tbSq*q.w.y+q.r*0.2
		cx, cy := (gx+rd.ox)*k, (gy+rd.oy)*k
		rx, ry := q.r*1.15*k*(1+h/80), q.r*0.62*k*(1+h/120)
		for y := int(cy - ry); y <= int(cy+ry); y++ {
			for x := int(cx - rx); x <= int(cx+rx); x++ {
				dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
				if d := dx*dx + dy*dy; d <= 1 && sh.in(x, y) {
					a := uint8((70 + 40*(1-d)) * fade)
					if sh.px[y*sh.w+x].A < a {
						sh.px[y*sh.w+x] = color.RGBA{8, 12, 8, a}
					}
				}
			}
		}
	}
}

// --- the head -----------------------------------------------------------------

// The head is drawn in two frames turned about the hinge of its jaws, the
// skull's (fr) and the jaw's (fj): u runs forward to the snout, v down.
var (
	tbSkull = []knob{{-15, -6, 11}, {-5, -8.2, 13}, {6, -8.8, 12.6}, {16, -7.2, 10.8}, {25, -5.4, 8.8}, {33, -4.2, 7.2}, {39.5, -3.2, 5.6}, {43.5, -2.4, 3.8}}
	tbJaw   = []knob{{-9, 5, 9.6}, {-1, 6.4, 7.6}, {10, 6.6, 6.5}, {21, 5.8, 5.3}, {30, 4.8, 4.3}, {37, 4, 3.3}, {41.5, 3.5, 2.4}}
	tbBrow  = []knob{{3.5, -17.6, 4.2}, {9.5, -18.8, 5}, {15.5, -17.8, 4.8}, {21, -15.2, 3.8}, {25, -12.4, 2.8}}
)

const tbEyeU, tbEyeV = 14.0, -10.4

// tbLipUp is where the skull's lip runs, in its frame; tbLipLow the jaw's.
func tbLipUp(u float64) float64  { return 2.3 - 1.6*smooth(clamp01((u-35)/9)) }
func tbLipLow(u float64) float64 { return 2.5 - 0.9*smooth(clamp01((u-35)/8)) }

type tbHeadPos struct {
	fr, fj bossFrame
	hx, hy float64 // the hinge, art pixels on the canvas
	a      float64
}

// headFrames places the head on the end of the neck.
func (rd *tbRender) headFrames(p tbPose) tbHeadPos {
	nk := rd.pts[len(rd.pts)-1]
	pr := rd.pts[len(rd.pts)-12]
	a := math.Atan2(nk.sy-pr.sy, nk.sx-pr.sx) + p.look
	ca, sa := math.Cos(a), math.Sin(a)
	hu, hv := 15.0*tbHeadK, 4.0*tbHeadK
	hx, hy := nk.sx+rd.ox+ca*hu-sa*hv, nk.sy+rd.oy+sa*hu+ca*hv
	return tbHeadPos{
		fr: hdFrameAt(rd.c, hx, hy, a-p.gape*0.36, tbHeadK),
		fj: hdFrameAt(rd.c, hx, hy, a+p.gape*0.7, tbHeadK),
		hx: hx, hy: hy, a: a,
	}
}

// tbPlates is the armour of the head: plates in a broken lattice, size
// big, with the distance to the nearest seam (in frame units) and the
// way to the middle of the plate.
func tbPlates(u, v, big float64, seed uint32) (edge, mx, my float64, id int) {
	gu, gv := u/big, v/big
	iu, iv := int(math.Floor(gu)), int(math.Floor(gv))
	d1, d2 := math.Inf(1), math.Inf(1)
	var cx, cy float64
	for j := iv - 1; j <= iv+1; j++ {
		for i := iu - 1; i <= iu+1; i++ {
			px := float64(i) + 0.15 + 0.7*hashf(i, j, seed)
			py := float64(j) + 0.15 + 0.7*hashf(i, j, seed+1)
			d := math.Hypot(gu-px, gv-py)
			if d < d1 {
				d2, d1, cx, cy, id = d1, d, px, py, i*7919+j
			} else if d < d2 {
				d2 = d
			}
		}
	}
	return (d2 - d1) * big * 0.5, cx*big - u, cy*big - v, id
}

// head paints the jaws, the mouth, the skull, the brow, the teeth and the eye.
func (rd *tbRender) head(hp tbHeadPos, p tbPose) {
	c := rd.c
	fr, fj := hp.fr, hp.fj
	// The mouth, between the jaws.
	if p.gape > 0.03 {
		var q [][3]float64
		for u := -9.0; u <= 42; u += 3 {
			q = append(q, [3]float64{u, tbLipUp(u) - 0.3, 0})
		}
		for u := 42.0; u >= -9; u -= 3 {
			q = append(q, [3]float64{u, tbLipLow(u) + 0.3, 1})
		}
		fr.poly2(c, fj, q, func(x, y int) color.RGBA {
			u, v := fr.local(x, y)
			_, v2 := fj.local(x, y)
			deep := smooth(clamp01((u + 4) / 44))
			col := mix(hex(0x14030a), hex(0x6a1f2a), deep)
			edge := min(v-tbLipUp(u), tbLipLow(u)-v2)
			if edge < 2.2 { // the gums
				col = mix(hex(0xa04c56), col, clamp01(edge/2.2))
			}
			if f := u / 2.6; f-math.Floor(f) < 0.18 && v-tbLipUp(u) < 5 && u > 4 {
				col = mix(col, hex(0x14030a), 0.3) // the ridges of its palate
			}
			return col
		})
		// the far row of teeth, in the dark of its mouth
		for u := 2.0; u < 38; u += 3.1 {
			l := 2 + 1.2*hashf(int(u), 5, 971)
			rd.tooth(fr, u-1.4, tbLipUp(u)+0.6, l*0.8, 0.8, 1, 0.45)
			rd.tooth(fj, u-0.6, tbLipLow(u)-0.6, l*0.7, 0.7, -1, 0.45)
		}
	}
	if p.tongue > 0 {
		f, tu, tv := fj, 30.0, 1.2
		if p.gape < 0.15 {
			f, tu, tv = fr, 38.0, 1.6
		}
		e := 14 * p.tongue
		frameQuad(c, f, tu-12, tv, tu+e, tv-0.3, 2, colTongue)
		frameQuad(c, f, tu-12, tv+0.4, tu+e-2, tv+0.2, 0.7, hex(0x7c1426))
		for _, dv := range []float64{-3, 2.4} {
			frameQuad(c, f, tu+e-0.4, tv-0.3, tu+e+5.5*p.tongue, tv-0.3+dv*p.tongue, 0.8, colTongue)
		}
	}
	// The jaw: pale scales along its lip, plates down its side, a pale throat.
	bossPaint(c, 0.3, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if u > -7 && v < tbLipLow(u) {
			return color.RGBA{}
		}
		l := 0.1 + q.l*0.86
		switch {
		case u > -7 && v < tbLipLow(u)+2.4: // the lip
			if f := u / 3.2; f-math.Floor(f) < 0.1 {
				l -= 0.12
			}
			return tbTone(tbBelly, l*0.9+0.04)
		case v > 9.2-0.1*u || q.ny > 0.55: // the throat
			if f := (u + v*0.3) / 2.2; f-math.Floor(f) < 0.18 {
				l -= 0.16
			}
			return tbTone(tbBelly, l*0.85)
		}
		e, mx, my, id := tbPlates(u, v, 3.6, 975)
		l += rd.plateLight(fj, e, mx, my) - 0.04
		rp := tbMoss
		if hashf(id, 0, 976) < 0.35 {
			rp = tbBronze
		}
		return tbTone(rp, l*0.92)
	}, fj.knobs(tbJaw))
	// Its jaw's own spikes, under the chin.
	for i, s := range [][4]float64{{-6, 10.5, 2.35, 11}, {3, 12, 2.15, 8}, {12, 11.8, 2.0, 6.5}, {20, 10.6, 1.9, 5}} {
		rd.hornAt(fj, s[0], s[1], s[2], s[3]*(0.9+0.1*p.spread), 1.6, 0.35, 0.25, true, 980+uint32(i))
	}
	// The skull: great keeled plates over the crown, small scales on the
	// face, a smooth pale lip, the eye sunk in a dark socket under the brow.
	bossPaint(c, 0.32, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > -9 && v > tbLipUp(u) {
			return color.RGBA{}
		}
		l := 0.06 + q.l*0.88
		if u > -8 && v > tbLipUp(u)-2.2 {
			if f := u / 3.2; f-math.Floor(f) < 0.1 {
				l -= 0.12
			}
			return tbTone(tbBelly, l*0.85+0.06)
		}
		if eye := math.Hypot((u-tbEyeU)/1.25, (v-tbEyeV)/0.95); eye < 5.2 {
			return tbTone(tbBronze, (l*0.3+0.01)*smooth(clamp01((eye-2.8)/2.4)))
		}
		crown := v < -12.5+0.14*u
		big := 2.5
		if crown {
			big = 5.8
		}
		e, mx, my, _ := tbPlates(u, v, big, 977)
		l += rd.plateLight(fr, e, mx, my) * (0.55 + 0.45*big/5.8)
		if crown && math.Abs(my) < 0.4 && e > 0.6 {
			l += 0.07 // the keel along a plate
		}
		// the shadow of the brow, over the eye
		if u > 0 && u < 28 && v > -15 && v < -5 {
			l -= 0.34 * smooth(clamp01((v+14)/3)) * smooth(clamp01((-6-v)/3)) * clamp01(1-math.Abs(u-tbEyeU)/15)
		}
		rp := tbMoss
		switch {
		case u < 12 && v > -7.6 && v < -3+0.12*u: // a dark band back from the eye
			rp, l = tbBronze, l*0.62
		case crown:
			rp, l = tbBronze, l*0.66
		}
		col := tbTone(rp, l*0.9)
		if g := math.Pow(clamp01(q.l), 7) * 0.35; e > 0.5 && g > 0.02 {
			col = mix(col, hex(0xe8d290), g)
		}
		return col
	}, fr.knobs(tbSkull))
	// The heavy brow over the eye: rough, ridged bone.
	bossPaint(c, 0.5, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		l := 0.03 + q.l*0.8 + (vnoise(u*0.8, v*0.8, 979)-0.5)*0.2
		if b := vnoise(u*1.5, v*1.5, 981); b < 0.32 {
			l -= 0.16 * (0.32 - b) / 0.32 * 2 // the pits and lumps of old bone
		}
		return tbTone(tbBronze, l*0.9+0.03)
	}, fr.knobs(tbBrow))
	// Nostril.
	nx, ny := fr.at(39.5, -4.6)
	for _, o := range [][3]float64{{0, 0, 1.1}, {0.4, 0.3, 0.7}} {
		rd.c.dotPx(nx+o[0]*fr.k1(), ny+o[1]*fr.k1(), o[2]*fr.k1(), tbInk)
	}
	// The teeth: rows of them, and a great fang at the front of each jaw.
	if p.gape > 0.12 {
		for u := 0.0; u < 37; u += 3.1 {
			l := 2.6 + 1.2*hashf(int(u), 0, 972)
			rd.tooth(fj, u+0.5, tbLipLow(u)+0.2, l, 1.05, -1, 1)
		}
		rd.tooth(fj, 33.5, tbLipLow(33.5)+0.2, 8*min(1, p.gape*1.6), 1.6, -1, 1)
	}
	for u := 1.0; u < 38; u += 2.9 {
		if math.Abs(u-29.7) < 2 {
			continue
		}
		l := (2.8 + 1.3*hashf(int(u), 1, 972)) * clamp01(p.gape*3)
		rd.tooth(fr, u, tbLipUp(u)-0.2, l, 1.05, 1, 1)
	}
	rd.tooth(fr, 29.8, tbLipUp(29.8)-0.2, 10, 1.9, 1, 1)
	rd.tooth(fr, 39.2, tbLipUp(39.2)-0.2, 7, 1.45, 1, 1)
	rd.eye(hp, p)
	// Spikes on the brow and a horn on the snout.
	rd.hornAt(fr, 9, -22.5, -2.1, 9*(0.8+0.2*p.spread), 1.9, 0.3, 0.3, true, 990)
	rd.hornAt(fr, 15.5, -21.8, -1.95, 7*(0.8+0.2*p.spread), 1.6, 0.3, 0.3, true, 991)
	rd.hornAt(fr, 21, -18.4, -1.8, 5.5, 1.4, 0.25, 0.3, true, 992)
	rd.hornAt(fr, 34.5, -9, -1.6, 6.5, 1.8, 0.35, 0.3, true, 993)
}

// plateLight is how much lighter (or darker) a pixel of a plate is: the
// bevel of its edge facing the light, the seam between plates in shadow.
func (rd *tbRender) plateLight(f bossFrame, edge, mx, my float64) float64 {
	if edge < 0.24 {
		return -0.24
	}
	bev := 1 - clamp01((edge-0.24)/1.2)
	if bev <= 0 {
		return 0
	}
	// the edge faces away from the plate's middle
	ex, ey := rotPt(f.a, -mx, -my)
	l := math.Hypot(ex, ey)
	if l < 1e-6 {
		return 0
	}
	return bev * 0.24 * (ex*tbL.x + ey*tbL.y) / l / math.Hypot(tbL.x, tbL.y)
}

// tooth sets a tooth on the lip at (u, v) of frame f, l long, w wide at its
// root, pointing down (dir 1) or up (-1); lit (0..1) dims the far row.
func (rd *tbRender) tooth(f bossFrame, u, v, l, w, dir, lit float64) {
	if l < 0.5 {
		return
	}
	tip := [2]float64{u - 0.28*l, v + dir*l}
	f.poly(rd.c, [][2]float64{{u - w, v}, {u + w, v}, {u + w*0.35, v + dir*l*0.55}, tip}, func(x, y int) color.RGBA {
		pu, pv := f.local(x, y)
		g := clamp01((pv - v) * dir / l) // 0 at the root, 1 at the tip
		ax := u - 0.28*l*g
		t := 0.75
		if pu > ax {
			t = 0.45 // the face turned from the light
		}
		col := tbTone(tbBone, (t+0.25*g)*lit)
		if g < 0.18 {
			col = mix(col, hex(0x9a4a50), (0.18-g)/0.18*0.6)
		}
		return col
	})
}

// eye paints an eye in its socket: a burning iris, a slit, a glint.
func (rd *tbRender) eye(hp tbHeadPos, p tbPose) {
	f := hp.fr
	cx, cy := f.at(tbEyeU, tbEyeV)
	r := 2.9 * f.k1()
	open := 1 - 0.82*p.lid
	for j := int(cy - r - 2); j <= int(cy+r+2); j++ {
		for i := int(cx - r - 2); i <= int(cx+r+2); i++ {
			du, dv := rotPt(-f.a, float64(i)+0.5-cx, float64(j)+0.5-cy)
			du, dv = du/r, dv/(r*open)
			d := math.Hypot(du, dv)
			switch {
			case d > 1.25:
			case d > 1:
				rd.c.put(i, j, hex(0x1a0c04))
			case math.Abs(du) < 0.2*(1.1-dv*dv) && open > 0.3:
				rd.c.put(i, j, hex(0x220a02))
			default:
				col := mix(hex(0xfff3b0), hex(0xffa830), clamp01(d*1.2))
				col = mix(col, hex(0xc8361a), clamp01((d-0.62)*2.4+dv*0.25))
				rd.c.put(i, j, col)
			}
		}
	}
	if open > 0.4 {
		gx, gy := f.at(tbEyeU-0.9, tbEyeV-0.9*open)
		rd.c.dotPx(gx, gy, 0.5*f.k1(), hex(0xfffef0))
	}
}

// eyeGlow lays the light of its eyes round them: on what is drawn, or
// outside, in the air about its head.
func (rd *tbRender) eyeGlow(hp tbHeadPos, p tbPose, outside bool) {
	c := rd.c
	cx, cy := hp.fr.at(tbEyeU, tbEyeV)
	R := 7 * hp.fr.k1()
	str := 1 - 0.5*p.lid
	for j := int(cy - R); j <= int(cy+R); j++ {
		for i := int(cx - R); i <= int(cx+R); i++ {
			if !c.in(i, j) {
				continue
			}
			d := math.Hypot(float64(i)+0.5-cx, float64(j)+0.5-cy) / R
			if d >= 1 {
				continue
			}
			g := (1 - d) * (1 - d) * str
			o := c.px[j*c.w+i]
			switch {
			case !outside && o.A > 0 && d > 0.62:
				add := func(a uint8, b float64) uint8 { return uint8(min(255, float64(a)+b)) }
				c.px[j*c.w+i] = color.RGBA{add(o.R, 150*g), add(o.G, 82*g), add(o.B, 14*g), o.A}
			case outside && o.A == 0 && d < 0.75:
				c.px[j*c.w+i] = color.RGBA{255, 170, 60, uint8(110 * g)}
			}
		}
	}
}

// hornAt draws a horn of the head on the canvas (or, for a far one, into
// the depth buffer): rooted at (u, v) in frame f, pointing along angle a
// (in the frame), l long and w wide at its root, curling back by curl.
func (rd *tbRender) hornAt(f bossFrame, u, v, a, l, w, curl, lift float64, near bool, seed uint32) {
	x, y := f.at(u, v)
	ang := f.a + a
	zs := lift
	if !near {
		zs = -lift
	}
	axis := vec3{math.Cos(ang), math.Sin(ang), zs}.unit()
	fwd := vec3{math.Cos(f.a), math.Sin(f.a), 0}
	b := newBlade(x, y, 0, axis, fwd, l*f.k1(), w*f.k1(), curl, tbHornK, (hashf(int(seed), 0, 994)-0.5)*0.08)
	if !near {
		nk := rd.pts[len(rd.pts)-1]
		b.z0, b.z1 = (nk.z-nk.r*0.4)*rd.k, (nk.z-nk.r*0.4)*rd.k
		rd.blades = append(rd.blades, b)
		return
	}
	b.raster(rd.c.w, rd.c.h, func(x, y int, uu, sg float64) {
		rd.c.put(x, y, b.colour(x, y, uu, sg, 0))
	})
}

// addHorns sets the crown of horns that fans back from its skull: the far
// ones (near false) behind the neck, the near ones over the head.
func (rd *tbRender) addHorns(hp tbHeadPos, p tbPose, near bool) {
	sp := p.spread
	for i, h := range [][6]float64{
		// u, v, angle, length, width, curl
		{7, -17, -2.42, 30, 3, 0.42},
		{0, -18.5, -2.62, 40, 3.4, 0.5},
		{-7, -17.5, -2.78, 38, 3.3, 0.45},
		{-13, -14, -2.92, 34, 3.1, 0.4},
		{-18, -8, -3.06, 29, 2.8, 0.35},
		{-19, -1, 3.05, 23, 2.5, 0.3},
		{-14, 5, 2.78, 17, 2.1, 0.28},
	} {
		isNear := i%2 == 0
		if isNear != near {
			continue
		}
		a := h[2] - sp*0.22*math.Sin(h[2]) // raised, they fan out wider
		l := h[3] * (0.92 + 0.08*sp)
		if !isNear {
			l *= 0.9
		}
		rd.hornAt(hp.fr, h[0], h[1], a, l, h[4], h[5], 0.3, isNear, 1000+uint32(i))
	}
}

// frameQuad fills a line from (u0, v0) to (u1, v1) of the frame, w thick.
func frameQuad(c *canvas, f bossFrame, u0, v0, u1, v1, w float64, col color.RGBA) {
	dx, dy := u1-u0, v1-v0
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	f.poly(c, [][2]float64{{u0 + nx, v0 + ny}, {u1 + nx, v1 + ny}, {u1 - nx, v1 - ny}, {u0 - nx, v0 - ny}}, solid(col))
}

// --- the poses ------------------------------------------------------------------

// tbSerpent is the path of the snake under way, its neck at (x, 0) and
// raised z above the ground: waves of the given phase run back down it from
// the head, growing towards the tail. n art pixels of it, tail first.
func tbSerpent(x, z, phase, amp, wave, n float64) []vec3 {
	const step = 0.5
	m := int(n / step)
	th := make([]float64, m+1)
	mean := 0.0
	for i := range th {
		sg := float64(i) * step
		env := amp * (0.16 + 0.84*smooth(clamp01(sg/130))) * (0.9 + 0.2*sg/n)
		th[i] = env * math.Sin(2*math.Pi*sg/wave+phase)
		mean += th[i]
	}
	mean /= float64(len(th))
	pts := make([]vec3, m+1)
	p := vec3{x, 0, 0}
	for i := range pts {
		sg := float64(i) * step
		p.z = z * smooth(clamp01(1-sg/90))
		pts[m-i] = p
		t := th[i] - mean
		p = p.add(vec3{-math.Cos(t), -math.Sin(t), 0}.mul(step))
	}
	return pts
}

const (
	titanSlitherN = 12
	titanIdleN    = 8
	titanRoarN    = 8
	titanGrabN    = 8
)

// titanStrikeAges is the age, in frames at 60 a second, each frame of the
// strike shows: drawing back, held there, the lunge (the blow lands 36
// frames in), the bite, and back.
var titanStrikeAges = [...]int{0, 3, 6, 9, 14, 20, 26, 31, 33, 35, 37, 41, 45, 49, 53}

// tbSlither is the path of the snake under way at phase ph of its waves,
// long enough to cut a strike's neck out of.
func tbSlither(ph float64) []vec3 { return tbSerpent(-30, 14, ph, 1.1, 300, tbLen+260) }

// tbFront replaces the last stretch of path (from cut art pixels short of
// its end) with a curve through ctrl: the body behind stays where it was,
// the front is posed.
func tbFront(path []vec3, cut float64, ctrl []vec3) []vec3 {
	at, i := 0.0, len(path)-1
	for ; i > 2 && at < cut; i-- {
		at += path[i].sub(path[i-1]).norm()
	}
	// the curve leaves the body behind along the way it was going
	c := append([]vec3{tbAt(path, cut+50), tbAt(path, cut+25), path[i]}, ctrl...)
	k := i
	for at2 := 0.0; k > 1 && at2 < 50; k-- {
		at2 += path[k].sub(path[k-1]).norm()
	}
	return append(append([]vec3{}, path[:k]...), tbSpline(c)...)
}

// tbAt is the point of path at art pixels from its end.
func tbAt(path []vec3, from float64) vec3 {
	at := 0.0
	for i := len(path) - 1; i > 0; i-- {
		d := path[i].sub(path[i-1]).norm()
		if at+d >= from {
			return lerp3(path[i], path[i-1], (from-at)/d)
		}
		at += d
	}
	return path[0]
}

// tbLerpPts blends two lists of control points.
func tbLerpPts(a, b []vec3, t float64) []vec3 {
	out := make([]vec3, len(a))
	for i := range a {
		out[i] = lerp3(a[i], b[i], t)
	}
	return out
}

const tbStrikeCut = 200.0 // the art pixels of neck a strike moves

var (
	tbRear  = []vec3{{-168, 4, 6}, {-128, 6, 38}, {-104, 6, 84}, {-110, 4, 118}, {-86, 2, 140}, {-58, 0, 128}}
	tbLunge = []vec3{{-150, 4, 2}, {-112, 6, 4}, {-72, 7, 8}, {-34, 8, 11}, {-4, 8, 12}, {24, 8, 10}}
)

// tbHold is pose to, its first points held near from, where the posed
// neck leaves the body.
func tbHold(from, to []vec3) []vec3 {
	out := make([]vec3, len(to))
	for i := range to {
		out[i] = lerp3(from[i], to[i], min(1, 0.35+0.33*float64(i)))
	}
	return out
}

// tbRest is the front of the slither at phase 0, as control points.
func tbRest() []vec3 {
	p := tbSlither(0)
	var out []vec3
	for _, d := range []float64{165, 132, 99, 66, 33, 0} {
		out = append(out, tbAt(p, d))
	}
	return out
}

// tbSpiral is a pile of coils about (cx, cy): from radius r0 at angle a0,
// turns times round and in to r1, rising from z0 to z1; a control point
// every twelfth of a turn.
func tbSpiral(cx, cy, r0, r1, a0, turns, z0, z1 float64) []vec3 {
	n := int(turns * 12)
	out := make([]vec3, 0, n+1)
	for i := 0; i <= n; i++ {
		u := float64(i) / float64(n)
		a := a0 - 2*math.Pi*turns*u
		r := lerpf(r0, r1, u)
		out = append(out, vec3{cx + r*math.Cos(a), cy + r*math.Sin(a), lerpf(z0, z1, math.Pow(u, 1.3))})
	}
	return out
}

var (
	titanSlither = newBHAnim(titanSlitherN, func(i int) (*canvas, float64, float64) {
		ph := -2 * math.Pi * float64(i) / titanSlitherN
		p := tbPose{path: tbSlither(ph), look: 0.1 + 0.05*math.Sin(ph+0.8)}
		if i == 5 || i == 6 {
			p.tongue = 0.6 + 0.4*float64(i-5)
		}
		return p.canvas()
	})
	titanStrike = newBHAnim(len(titanStrikeAges), func(i int) (*canvas, float64, float64) {
		age := float64(titanStrikeAges[i])
		rear := smooth(clamp01(age / 9))
		lunge := smooth(clamp01((age - 31) / 5))
		if age > 42 {
			lunge, rear = 1-smooth(clamp01((age-42)/12)), 0
		} else {
			rear *= 1 - lunge
		}
		shake := 0.0
		if age > 9 && age < 31 {
			shake = math.Sin(age * 1.9)
		}
		rest := tbRest()
		ctrl := tbLerpPts(rest, tbHold(rest, tbRear), rear)
		ctrl = tbLerpPts(ctrl, tbHold(rest, tbLunge), lunge)
		for j := range ctrl {
			ctrl[j].x += shake * 1.2 * float64(j) / 5
		}
		p := tbPose{path: tbFront(tbSlither(0), tbStrikeCut, ctrl)}
		p.look = 0.1*(1-rear-lunge) - 0.35*rear - 0.05*lunge
		p.gape = 0.15 + 0.85*smooth(clamp01(age/14))
		p.spread = rear
		switch {
		case age >= 42:
			p.gape = 0.25 * (1 - clamp01((age-42)/8))
		case age >= 36:
			p.gape = 0.25 // shut on its prey
		}
		if age >= 9 && age < 31 {
			p.tongue = 1
		}
		return p.canvas()
	})
	titanIdle = newBHAnim(titanIdleN, func(i int) (*canvas, float64, float64) {
		// Asleep: a pile of coils that swells as it breathes, the head laid on it.
		b := math.Sin(2 * math.Pi * float64(i) / titanIdleN)
		ctrl := tbSpiral(-10, -12, 58, 21, 2.9, 2.45, 0, 36)
		ctrl = append(ctrl, vec3{-2, -20, 48}, vec3{12, -10, 54 + b*0.8})
		p := tbPose{path: tbSpline(ctrl), look: 0.12, swell: 0.025 * b, lid: 0.55, spread: -0.8}
		if i == 2 || i == 3 {
			p.tongue = 0.5 + 0.5*float64(i-2)
		}
		return p.canvas()
	})
	titanRoar = newBHAnim(titanRoarN, func(i int) (*canvas, float64, float64) {
		// Reared high out of its coils, jaws wide, swaying as it roars; a
		// ripple runs back along the coils trailing behind.
		w := 2 * math.Pi * float64(i) / titanRoarN
		ctrl := []vec3{{-430, -10, 0}, {-360, 30, 0}, {-290, 40, 0}, {-225, 12, 0}, {-165, 18, 0}, {-118, 36, 0}}
		ctrl = append(ctrl, tbSpiral(-34, 2, 62, 36, math.Pi*1.02, 1.0, 0, 8)...)
		ctrl = append(ctrl, []vec3{{-22, 34, 30}, {-16, 26, 72}, {-8, 18, 112}, {6, 12, 138}, {24, 8, 146}, {40, 6, 138}}...)
		for j := range ctrl {
			q := &ctrl[j]
			if q.z == 0 {
				q.y += 5 * math.Sin(w-q.x*0.025)
			}
			q.x += 7 * math.Sin(w+0.6) * smooth(clamp01((q.z-40)/120))
			q.y += 3 * math.Cos(w) * smooth(clamp01((q.z-40)/120))
		}
		p := tbPose{path: tbSpline(ctrl), look: -0.62 + 0.08*math.Sin(w), gape: 0.92 + 0.08*math.Sin(2*w),
			tongue: 0.55 + 0.45*math.Sin(2*w), spread: 1}
		return p.canvas()
	})
	titanGrab = newBHAnim(titanGrabN, func(i int) (*canvas, float64, float64) {
		// Wound tight, ring on ring, round whatever it has caught: the rings
		// draw in and ease, and slide round as it tightens.
		w := 2 * math.Pi * float64(i) / titanGrabN
		sq := 0.5 - 0.5*math.Cos(w)
		ctrl := []vec3{{-340, 40, 0}, {-270, 10, 0}, {-200, 32, 0}, {-140, 46, 0}, {-88, 30, 0}}
		hel := tbSpiral(0, -4, 31-3*sq, 31-3*sq, math.Pi*0.92+w*0.06, 2.25, 0, 58)
		ctrl = append(ctrl, hel...)
		ctrl = append(ctrl, vec3{6, -12, 86}, vec3{16, -8, 98 + 2*sq})
		p := tbPose{path: tbSpline(ctrl), look: 0.55 + 0.06*math.Sin(w), gape: 0.45 + 0.3*sq, tongue: sq, spread: 0.6}
		return p.canvas()
	})
)

// titanStrikeFrame is the frame of the strike age frames into it.
func titanStrikeFrame(age int) int {
	f := 0
	for i, a := range titanStrikeAges {
		if age >= a {
			f = i
		}
	}
	return f
}
