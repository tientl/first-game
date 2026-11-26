package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"survivalgame/internal/sim"
)

// The rocks: the boulders of every ground, the ore veins, the coal seams and
// the pebbles, drawn the way a pixel artist draws a rock (after "Cách mình
// Pixel Art: QUẶNG KIM LOẠI", @pt.game91):
//
//  1. a palette: every rock is coloured from COMFY52 (BESS, lospec.com);
//  2. a rugged silhouette of straight runs that break and turn, with a chip
//     or two lying apart from the main mass;
//  3. light: the faces turned up and to the top left are the lightest, in
//     large clear planes, the rock built of several blocks;
//  4. dark: the faces turned away are dark, and so are the hollows and
//     where one block meets another; every block shows its three faces;
//  5. details: an outline darker or lighter with the light, fine cracks,
//     and small bright points on the edges where a lit face turns into a
//     dark one;
//  6. the ore: the same again for the pieces of ore on and in the stone, in
//     their own colours, some standing proud of it.
//
// The blocks are convex solids (each the inside of a few planes), lit from
// the top left and seen from the game's angle: each pixel looks down into
// the scene and takes the face it meets first, so faces come out flat,
// blocks overlap and the ore sinks into the stone of itself. Everything is
// measured in art pixels and drawn k image pixels to one (HDDensity at HD).

// comfy52 is the COMFY52 palette by BESS (lospec.com).
var comfy52 = [...]uint32{
	0x3f4328, 0x5f7132, 0x94ad39, 0xc2d64f, 0xeff37c, 0xe3e6ac, 0xa5c67c, 0x739a70, 0x4d6659, 0x343f41,
	0x282e3b, 0x1a1f2e, 0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xe5efef, 0xbac9cd, 0x8d99a4,
	0x696f80, 0x414453, 0xb8a1c2, 0x7e659b, 0x5c3a6f, 0x39275e, 0x2f193e, 0x4e1a49, 0x7b234c, 0xb23657,
	0xd16974, 0xedaaa3, 0xeecb90, 0xe1a845, 0xc57835, 0x8d4830, 0xe47259, 0xc33c40, 0x8d3649, 0x5c2b34,
	0x3c252b, 0x684039, 0x825646, 0xb77862, 0x7d595d, 0x533b41, 0x3f333b, 0x2b222a, 0x6d4e4b, 0x867066,
	0xb49d7e, 0xc4c6b8,
}

// rkMat is what a block is made of: its faces from dark to light, the
// outline on its shaded side (ink) and on its lit side (rim), and the
// brightest points of its edges.
type rkMat struct {
	tone            ramp
	ink, rim, glint color.RGBA
	bias            float64 // added to the light before a tone is picked
}

var (
	rkGraniteMat = &rkMat{tone: mkRamp(0x282e3b, 0x414453, 0x696f80, 0x8d99a4, 0xbac9cd), ink: hex(0x1a1f2e), rim: hex(0x414453), glint: hex(0xe5efef)}
	rkCaveMat    = &rkMat{tone: mkRamp(0x1a1f2e, 0x282e3b, 0x414453, 0x696f80, 0x8d99a4), ink: hex(0x1a1f2e), rim: hex(0x282e3b), glint: hex(0xbac9cd)}
	rkSandMat    = &rkMat{tone: mkRamp(0x3f333b, 0x533b41, 0x6d4e4b, 0x867066, 0xb49d7e), ink: hex(0x2b222a), rim: hex(0x533b41), glint: hex(0xeecb90), bias: 0.1}
	rkBasaltMat  = &rkMat{tone: mkRamp(0x1a1f2e, 0x282e3b, 0x343f41, 0x414453, 0x696f80), ink: hex(0x1a1f2e), rim: hex(0x282e3b), glint: hex(0x8d99a4), bias: 0.06}
	rkFlintMat   = &rkMat{tone: mkRamp(0x2b222a, 0x3f333b, 0x533b41, 0x7d595d), ink: hex(0x1a1f2e), rim: hex(0x2b222a), glint: hex(0xe5efef), bias: -0.18}
	rkJadeMat    = &rkMat{tone: mkRamp(0x4d6659, 0x739a70, 0xa5c67c, 0x9ae5d5), ink: hex(0x343f41), rim: hex(0x4d6659), glint: hex(0xe5efef), bias: 0.02}
	rkAgateMat   = &rkMat{tone: mkRamp(0x5c2b34, 0x8d4830, 0xc57835, 0xe1a845, 0xeecb90), ink: hex(0x3c252b), rim: hex(0x5c2b34), glint: hex(0xe3e6ac)}
	rkGlassMat   = &rkMat{tone: mkRamp(0x1a1f2e, 0x2f193e, 0x39275e, 0x5c3a6f, 0x7e659b, 0xb8a1c2), ink: hex(0x1a1f2e), rim: hex(0x2f193e), glint: hex(0xe5efef)}
	rkCoalMat    = &rkMat{tone: mkRamp(0x1a1f2e, 0x282e3b, 0x343f41, 0x414453), ink: hex(0x1a1f2e), rim: hex(0x1a1f2e), glint: hex(0x696f80)}
	rkCoalHost   = &rkMat{tone: mkRamp(0x282e3b, 0x414453, 0x696f80, 0x8d99a4, 0xbac9cd), ink: hex(0x1a1f2e), rim: hex(0x414453), glint: hex(0xbac9cd), bias: -0.2}

	rkMossRamp = mkRamp(0x3f4328, 0x5f7132, 0x94ad39, 0xc2d64f)
	rkSnowRamp = mkRamp(0x696f80, 0x8d99a4, 0xbac9cd, 0xe5efef)
	rkJadeVein = mkRamp(0x4d6659, 0x739a70, 0xa5c67c, 0x9ae5d5)
	rkCoalSeam = mkRamp(0x1a1f2e, 0x1a1f2e, 0x282e3b, 0x343f41)
)

// rkVec is a point or a direction: x east, y north (away from the viewer),
// z up, in art pixels.
type rkVec [3]float64

func (a rkVec) dot(b rkVec) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a rkVec) add(b rkVec) rkVec   { return rkVec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a rkVec) mul(s float64) rkVec { return rkVec{a[0] * s, a[1] * s, a[2] * s} }
func (a rkVec) cross(b rkVec) rkVec {
	return rkVec{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func (a rkVec) unit() rkVec {
	if l := math.Sqrt(a.dot(a)); l > 0 {
		return a.mul(1 / l)
	}
	return a
}

// The view: the camera looks north and down at about 38°, so the depth of
// the ground shows at rkSin of its length and heights at rkCos of theirs.
const (
	rkSin = 0.62
	rkCos = 0.785
)

var (
	rkView = rkVec{0, rkCos, -rkSin}          // the way a pixel looks
	rkUp   = rkVec{0, rkSin, rkCos}           // up the screen
	rkSun  = rkVec{-0.5, 0.15, 0.85}.unit()   // towards the light: the top left
	rkHalf = rkSun.add(rkView.mul(-1)).unit() // half way to the eye: where gloss shows
	rkDown = rkVec{0, 0, -1}                  // the bottom of a block on the ground
	rkEast = rkVec{1, 0, 0}                   // for frames about an axis
	rkTop  = rkVec{0, 0, 1}                   // straight up
)

// rkLight is the light a face turned to n gets, 0 to 1, wrapped round so a
// face turned a little away is not yet black.
func rkLight(n rkVec) float64 {
	const wrap = 0.5
	return clamp01((n.dot(rkSun) + wrap) / (1 + wrap))
}

// rkPlane bounds a block: the block is where n·p <= d.
type rkPlane struct {
	n rkVec
	d float64
}

func rkPlaneAt(n, p rkVec) rkPlane { n = n.unit(); return rkPlane{n, n.dot(p)} }

// The looks of a block.
const (
	rkStone  = iota // host rock
	rkGloss         // a glossy nodule: flint
	rkGem           // a translucent lump: jade
	rkBanded        // banded stone: agate
	rkGlass         // volcanic glass: obsidian
	rkCoal          // coal: black with a dull sheen
)

// rkDecor may colour a point p of a host block's face turned to n, whose
// light is l: moss, snow, the layers of sandstone, veins, seams. It gives
// the ramp and the light to pick from it.
type rkDecor func(p, n rkVec, l float64, pt *rkPart) (ramp, float64, bool)

// rkPart is one block of a rock, or one piece of ore.
type rkPart struct {
	pl    []rkPlane
	mat   *rkMat
	look  int
	ore   bool    // ore: drawn in the last step
	c     rkVec   // its middle, where an agate's bands centre
	r     float64 // its size
	band  rkVec   // across the bands of an agate
	nudge float64 // a little more or less light, so blocks side by side part
	crack float64 // fine cracks per hundred pixels of its faces
	decor rkDecor
	verts []rkVec // its corners, worked out by corners()
}

// corners works out the corners of p: every point where three of its
// planes meet that is inside the rest.
func (p *rkPart) corners() {
	p.verts = p.verts[:0]
	n := len(p.pl)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			ab := p.pl[i].n.cross(p.pl[j].n)
			for k := j + 1; k < n; k++ {
				a, b, c := p.pl[i], p.pl[j], p.pl[k]
				bc, ca := b.n.cross(c.n), c.n.cross(a.n)
				det := a.n.dot(bc)
				if math.Abs(det) < 1e-9 {
					continue
				}
				v := bc.mul(a.d).add(ca.mul(b.d)).add(ab.mul(c.d)).mul(1 / det)
				in := true
				for _, q := range p.pl {
					if q.n.dot(v) > q.d+1e-6 {
						in = false
						break
					}
				}
				if in {
					p.verts = append(p.verts, v)
				}
			}
		}
	}
}

// move scales p by f about the ground point and then shifts it by s.
func (p *rkPart) move(f float64, s rkVec) {
	for i, q := range p.pl {
		p.pl[i].d = q.d*f + q.n.dot(s)
	}
	p.c = p.c.mul(f).add(s)
	p.r *= f
	for i, v := range p.verts {
		p.verts[i] = v.mul(f).add(s)
	}
}

// rkScene is a rock: its blocks and its pieces of ore.
type rkScene struct {
	parts []rkPart
	seed  uint32
	f     float64 // how fit has scaled the scene (0: not yet)...
	sh    rkVec   // ...and shifted it
}

func (s *rkScene) add(p rkPart) *rkPart {
	p.corners()
	if len(p.verts) < 4 {
		return nil // cut away to nothing
	}
	s.parts = append(s.parts, p)
	return &s.parts[len(s.parts)-1]
}

// extent is how far the scene reaches on the screen from its ground point,
// in art pixels: up (top), down (a negative bot), left and right.
func (s *rkScene) extent() (top, bot, left, right float64) {
	top, bot, left, right = math.Inf(-1), math.Inf(1), math.Inf(1), math.Inf(-1)
	for _, p := range s.parts {
		for _, v := range p.verts {
			u := v.dot(rkUp)
			top, bot = max(top, u), min(bot, u)
			left, right = min(left, v[0]), max(right, v[0])
		}
	}
	return
}

// fit scales and shifts the scene so that it stands height art pixels
// above its ground point and its front reaches foot below it.
func (s *rkScene) fit(height, foot float64) {
	top, bot, _, _ := s.extent()
	f := (height + foot) / (top - bot)
	sh := rkVec{0, (height - f*top) / rkSin, 0}
	for i := range s.parts {
		s.parts[i].move(f, sh)
	}
	if s.f == 0 {
		s.f = 1
	}
	s.f, s.sh = s.f*f, s.sh.mul(f).add(sh)
}

// design is where point p of the fitted scene was as the scene was laid
// out, before fit moved it.
func (s *rkScene) design(p rkVec) rkVec { return p.add(s.sh.mul(-1)).mul(1 / s.f) }

// rkPix is what one pixel sees.
type rkPix struct {
	part, face int // part -1: nothing
	t          float64
	p, n       rkVec
	rp         ramp
	idx        int
	col        color.RGBA
	lip        bool
}

// cast looks into the scene through every pixel of a w by h image whose
// ground point is at (ox, oy), k image pixels to an art pixel. With
// hostOnly, the ore is left out.
func (s *rkScene) cast(w, h int, ox, oy, k float64, hostOnly bool) []rkPix {
	px := make([]rkPix, w*h)
	for i := range px {
		px[i].part = -1
	}
	for i := range s.parts {
		pt := &s.parts[i]
		if hostOnly && pt.ore {
			continue
		}
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, v := range pt.verts {
			sx, sy := ox+v[0]*k, oy-v.dot(rkUp)*k
			x0, x1 = min(x0, sx), max(x1, sx)
			y0, y1 = min(y0, sy), max(y1, sy)
		}
		nv := make([]float64, len(pt.pl))
		for f, pl := range pt.pl {
			nv[f] = pl.n.dot(rkView)
		}
		for y := max(0, int(y0)-1); y < min(h, int(y1)+2); y++ {
			up := (oy - float64(y) - 0.5) / k
			for x := max(0, int(x0)-1); x < min(w, int(x1)+2); x++ {
				p0 := rkVec{(float64(x) + 0.5 - ox) / k, up * rkUp[1], up * rkUp[2]}
				in, out, face := math.Inf(-1), math.Inf(1), -1
				for f, pl := range pt.pl {
					r := pl.d - pl.n.dot(p0)
					switch v := nv[f]; {
					case v < -1e-9:
						if t := r / v; t > in {
							in, face = t, f
						}
					case v > 1e-9:
						out = min(out, r/v)
					case r < 0:
						out = math.Inf(-1)
					}
				}
				q := &px[y*w+x]
				if face < 0 || in > out || q.part >= 0 && in >= q.t {
					continue
				}
				q.part, q.face, q.t = i, face, in
				q.p, q.n = p0.add(rkView.mul(in)), pt.pl[face].n
			}
		}
	}
	return px
}

// shade picks the colour of what q sees at step stage.
func (s *rkScene) shade(q *rkPix, stage int) {
	pt := &s.parts[q.part]
	m := pt.mat
	q.rp = m.tone
	n := len(q.rp)
	l := rkLight(q.n) + pt.nudge + m.bias
	switch stage {
	case 1: // the silhouette, in one colour
		q.idx = 1
		q.col = q.rp[q.idx]
		return
	case 2: // the light: lit faces on the base colour
		q.idx = 1
		switch {
		case l >= 0.8:
			q.idx = n - 1
		case l >= 0.6:
			q.idx = n - 2
		}
		q.col = q.rp[q.idx]
		return
	}
	spec := q.n.dot(rkHalf)
	var glint bool
	switch pt.look {
	case rkStone:
		if pt.decor != nil && stage >= 4 {
			if rp, dl, ok := pt.decor(q.p, q.n, l, pt); ok {
				q.rp, l, n = rp, dl, len(rp)
			}
		}
	case rkGloss: // flint: glossy, a wet point of light where its round back turns to the sun
		if spec > 0.9 {
			l = 1
		}
		sn := q.p.add(pt.c.mul(-1)).unit()
		glint = sn.dot(rkHalf) > 0.93
	case rkGem: // jade: light comes through, it is never black
		l = 0.18 + 0.68*l
		glint = spec > 0.975
	case rkBanded: // agate: wavy bands laid through it, lit or shaded
		q0 := q.p.add(pt.c.mul(-1)).dot(pt.band) + (vnoise(q.p[0]*0.7, q.p[2]*0.7+q.p[1]*0.3, s.seed+7)-0.5)*1.1
		bands := [...]int{4, 2, 3, 1, 3, 4, 2, 0, 2, 3}
		b := bands[int(math.Floor(q0/0.62)+100)%len(bands)]
		switch {
		case l < 0.38:
			b -= 2
		case l < 0.6:
			b--
		case l > 0.85:
			b++
		}
		q.idx = max(0, min(n-1, b))
		q.col = q.rp[q.idx]
		if spec > 0.97 {
			q.col = m.glint
		}
		return
	case rkGlass: // obsidian: black glass, a purple sheen where it faces the light
		l = l * l
		if l > 0.3 && math.Mod(q.p[0]*0.55+q.p[2]*0.35+40, 2.2) < 0.45 {
			l += 0.25 // a streak of sheen
		}
		if spec > 0.9 {
			l = 1
		}
		glint = spec > 0.97
	case rkCoal: // coal: dull
		l *= 0.95
		glint = spec > 0.95
	}
	q.idx = max(0, min(n-1, int(l*float64(n))))
	q.col = q.rp[q.idx]
	if glint {
		q.col = m.glint
	}
}

// draw paints the scene up to step stage (1 the silhouette, 2 the light,
// 3 the dark, 4 the details, 5 the ore; 5 for the game) at k image pixels
// to an art pixel. It gives the canvas and its ground point in its pixels.
func (s *rkScene) draw(k float64, stage int) (*canvas, float64, float64) {
	top, bot, left, right := s.extent()
	l, r := math.Floor(left-1), math.Ceil(right+1)
	t, b := math.Ceil(top+1), math.Ceil(-bot+1)
	w, h := int((r-l)*k), int((t+b)*k)
	ox, oy := -l*k, t*k
	c := newCanvas(w, h)
	if k == HDDensity {
		c.dens = HDDensity
	}
	px := s.cast(w, h, ox, oy, k, stage < 5)
	s.paint(c, px, k, stage)
	return c, ox, oy
}

// paint colours what the pixels px see onto c.
func (s *rkScene) paint(c *canvas, px []rkPix, k float64, stage int) {
	w, h := c.w, c.h
	at := func(x, y int) *rkPix {
		if x < 0 || y < 0 || x >= w || y >= h {
			return nil
		}
		return &px[y*w+x]
	}
	hit := func(x, y int) bool { q := at(x, y); return q != nil && q.part >= 0 }
	dirs := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	for i := range px {
		if px[i].part >= 0 {
			s.shade(&px[i], stage)
		}
	}
	// A pixel alone on its face, among another face's, takes theirs: no
	// specks where faces meet edge on.
	if stage >= 3 {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				q := at(x, y)
				if q.part < 0 {
					continue
				}
				var o *rkPix
				same := 0
				for _, d := range dirs {
					r := at(x+d[0], y+d[1])
					switch {
					case r == nil || r.part < 0:
						same = 9
					case r.part == q.part && r.face == q.face:
						same++
					default:
						o = r
					}
				}
				if same == 0 && o != nil && o.part == q.part {
					q.face, q.n, q.rp, q.idx, q.col = o.face, o.n, o.rp, o.idx, o.col
				}
			}
		}
	}
	rng := rand.New(rand.NewPCG(uint64(s.seed), 0x70c4))
	if stage >= 4 {
		// The edges where a lit face turns down or away into a darker one
		// catch the light; where three faces meet, a glint.
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				q := at(x, y)
				if q.part < 0 {
					continue
				}
				for _, d := range [2][2]int{{0, 1}, {1, 0}} {
					r := at(x+d[0], y+d[1])
					if r != nil && r.part == q.part && r.face != q.face && r.idx < q.idx && q.col == q.rp[q.idx] {
						q.lip = true
					}
				}
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				q := at(x, y)
				if !q.lip {
					continue
				}
				pt := &s.parts[q.part]
				up := 1
				if pt.look == rkGlass {
					up = 2
				}
				q.col = q.rp[min(len(q.rp)-1, q.idx+up)]
				faces := map[int]bool{}
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if r := at(x+dx, y+dy); r != nil && r.part == q.part {
							faces[r.face] = true
						}
					}
				}
				if len(faces) >= 3 && q.idx >= len(q.rp)-2 || q.idx == len(q.rp)-1 && hashf(x, y, s.seed+3) < 0.12 {
					q.col = pt.mat.glint
				}
			}
		}
		s.cracks(px, w, h, k, rng)
		// The foot of the rock, in the shade of the ground.
		for i := range px {
			q := &px[i]
			if q.part >= 0 && !s.parts[q.part].ore && q.p[2] < 0.6 && q.n[2] < 0.6 && q.col == q.rp[q.idx] {
				q.idx = max(0, q.idx-1)
				q.col = q.rp[q.idx]
			}
		}
	}
	if stage >= 3 {
		// Where one block meets another the one behind is dark, and the
		// stone round a piece of ore sunk in it is dark too.
		dark := make([]bool, len(px))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				q := at(x, y)
				if q.part < 0 {
					continue
				}
				for _, d := range dirs {
					r := at(x+d[0], y+d[1])
					if r == nil || r.part < 0 || r.part == q.part {
						continue
					}
					// The one behind; where they meet flush, the stone
					// round the ore, or else the later block.
					if r.t < q.t-0.3 || r.t < q.t+0.3 && (s.parts[r.part].ore && !s.parts[q.part].ore || s.parts[r.part].ore == s.parts[q.part].ore && q.part > r.part) {
						dark[y*w+x] = true
					}
				}
			}
		}
		for i, d := range dark {
			if !d {
				continue
			}
			q := &px[i]
			m := s.parts[q.part].mat
			if q.idx >= 2 && q.rp[0] != m.ink {
				q.col = q.rp[0]
			} else {
				q.col = m.ink
			}
		}
	}
	for i, q := range px {
		if q.part >= 0 {
			c.px[i] = q.col
		}
	}
	if stage < 2 {
		return
	}
	// The outline, dark below and on the right, lighter on the lit faces at
	// the top and on the left.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if hit(x, y) {
				continue
			}
			var src *rkPix
			lit := false
			switch {
			case hit(x, y-1):
				src = at(x, y-1)
			case hit(x-1, y):
				src = at(x-1, y)
			case hit(x+1, y):
				src, lit = at(x+1, y), true
			case hit(x, y+1):
				src, lit = at(x, y+1), true
			default:
				continue
			}
			m := s.parts[src.part].mat
			col := m.ink
			if lit && stage >= 4 && src.idx >= 2 {
				col = m.rim
			}
			c.px[y*w+x] = col
		}
	}
}

// cracks draws fine cracks over the faces of the host blocks: a dark line
// with a lit lip on its lower side here and there.
func (s *rkScene) cracks(px []rkPix, w, h int, k float64, rng *rand.Rand) {
	at := func(x, y int) *rkPix {
		if x < 0 || y < 0 || x >= w || y >= h {
			return nil
		}
		return &px[y*w+x]
	}
	inner := func(x, y int, q *rkPix) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				r := at(x+dx, y+dy)
				if r == nil || r.part != q.part || r.face != q.face {
					return false
				}
			}
		}
		return true
	}
	for pi := range s.parts {
		pt := &s.parts[pi]
		if pt.crack <= 0 {
			continue
		}
		var cand []int
		for i, q := range px {
			if q.part == pi && q.col == q.rp[q.idx] && inner(i%w, i/w, &px[i]) {
				cand = append(cand, i)
			}
		}
		n := int(float64(len(cand))*pt.crack/100 + rng.Float64())
		for j := 0; j < n && len(cand) > 0; j++ {
			i := cand[rng.IntN(len(cand))]
			face := px[i].face
			fx, fy := float64(i%w)+0.5, float64(i/w)+0.5
			a := rng.Float64() * 2 * math.Pi
			if px[i].n[2] < 0.5 { // down a side
				a = math.Pi/2 + (rng.Float64()-0.5)*1.6
			}
			ln := int((3 + rng.Float64()*5) * k / HDDensity)
			for step := 0; step < max(2, ln); step++ {
				x, y := int(fx), int(fy)
				q := at(x, y)
				if q == nil || q.part != pi || q.face != face || !inner(x, y, q) && step > 0 {
					break
				}
				if q.idx >= 2 {
					q.col = q.rp[q.idx-2]
				} else {
					q.col = pt.mat.ink
				}
				if r := at(x, y+1); r != nil && r.part == pi && r.face == face && step%2 == 0 && r.col == r.rp[r.idx] {
					r.col = r.rp[min(len(r.rp)-1, r.idx+1)]
				}
				a += (rng.Float64() - 0.5) * 0.9
				fx, fy = fx+math.Cos(a), fy+math.Sin(a)
			}
		}
	}
}

// --- blocks -------------------------------------------------------------------

// rkChunk is a block of stone standing on z0: its sides lean in as they rise,
// its top tilts a little and some of its upper edges are cut off at a slant.
func rkChunk(rng *rand.Rand, cx, cy, z0, rx, ry, h float64, sides, cuts int, tilt, lean float64) rkPart {
	pl := []rkPlane{{rkDown, -z0}}
	pl = append(pl, rkPlaneAt(rkVec{(rng.Float64()*2 - 1) * tilt, (rng.Float64()*2 - 1) * tilt, 1}, rkVec{cx, cy, z0 + h}))
	support := func(a float64) float64 { return math.Hypot(rx*math.Cos(a), ry*math.Sin(a)) }
	a0 := rng.Float64() * 2 * math.Pi
	for i := 0; i < sides; i++ {
		a := a0 + (float64(i)+(rng.Float64()-0.5)*0.6)*2*math.Pi/float64(sides)
		u := rkVec{math.Cos(a), math.Sin(a), 0}
		s := support(a) * (0.9 + rng.Float64()*0.2)
		pl = append(pl, rkPlaneAt(rkVec{u[0], u[1], lean * (0.5 + rng.Float64())}, rkVec{cx + u[0]*s, cy + u[1]*s, z0}))
	}
	for i := 0; i < cuts; i++ {
		a := -math.Pi/2 + (rng.Float64()*2-1)*math.Pi*0.8 // mostly the edges seen
		u := rkVec{math.Cos(a), math.Sin(a), 0}
		s := support(a) * (0.35 + rng.Float64()*0.4)
		pl = append(pl, rkPlaneAt(rkVec{u[0], u[1], 0.6 + rng.Float64()*1.1}, rkVec{cx + u[0]*s, cy + u[1]*s, z0 + h}))
	}
	return rkPart{pl: pl, c: rkVec{cx, cy, z0 + h/2}, r: math.Max(rx, h)}
}

// rkColumn is a column of basalt: six sides, its top tilted or broken.
func rkColumn(rng *rand.Rand, cx, cy, r, h, rot, tilt float64) rkPart {
	pl := []rkPlane{{rkDown, 0}}
	pl = append(pl, rkPlaneAt(rkVec{(rng.Float64()*2 - 1) * tilt, (rng.Float64()*2 - 1) * tilt, 1}, rkVec{cx, cy, h}))
	for i := 0; i < 6; i++ {
		a := rot + float64(i)*math.Pi/3 + (rng.Float64()-0.5)*0.15
		rr := r * (0.92 + rng.Float64()*0.16)
		pl = append(pl, rkPlaneAt(rkVec{math.Cos(a), math.Sin(a), 0.05}, rkVec{cx + math.Cos(a)*rr, cy + math.Sin(a)*rr, 0}))
	}
	if rng.Float64() < 0.6 { // a corner broken off the top
		a := -math.Pi/2 + (rng.Float64()*2-1)*1.8
		pl = append(pl, rkPlaneAt(rkVec{math.Cos(a), math.Sin(a), 0.9 + rng.Float64()*0.6}, rkVec{cx + math.Cos(a)*r*0.35, cy + math.Sin(a)*r*0.35, h}))
	}
	return rkPart{pl: pl, c: rkVec{cx, cy, h / 2}, r: r}
}

// rkFallen is a column of basalt lying on its side along x, ln long.
func rkFallen(rng *rand.Rand, cx, cy, r, ln float64) rkPart {
	c := rkVec{cx, cy, r}
	var pl []rkPlane
	for i := 0; i < 6; i++ {
		a := math.Pi/6 + float64(i)*math.Pi/3
		n := rkVec{0, math.Cos(a), math.Sin(a)}
		pl = append(pl, rkPlaneAt(n, c.add(n.mul(r*(0.94+rng.Float64()*0.12)))))
	}
	tw := (rng.Float64() - 0.5) * 0.5
	pl = append(pl, rkPlaneAt(rkVec{1, tw, 0.1}, c.add(rkVec{ln / 2, 0, 0})), rkPlaneAt(rkVec{-1, -tw, 0.25}, c.add(rkVec{-ln / 2, 0, 0})))
	return rkPart{pl: pl, c: c, r: r}
}

// rkNodule is a rounded lump about c, rx by ry by rz, of facets flat faces;
// jag roughens it.
func rkNodule(rng *rand.Rand, c rkVec, rx, ry, rz float64, facets int, jag float64) rkPart {
	var pl []rkPlane
	off := rng.Float64() * 2 * math.Pi
	for i := 0; i < facets; i++ {
		z := 1 - 2*(float64(i)+0.5)/float64(facets)
		r := math.Sqrt(1 - z*z)
		a := off + float64(i)*2.39996
		n := rkVec{r*math.Cos(a) + (rng.Float64()-0.5)*0.35, r*math.Sin(a) + (rng.Float64()-0.5)*0.35, z}.unit()
		d := math.Sqrt(rx*rx*n[0]*n[0]+ry*ry*n[1]*n[1]+rz*rz*n[2]*n[2]) * (1 - jag*rng.Float64())
		pl = append(pl, rkPlane{n, n.dot(c) + d})
	}
	return rkPart{pl: pl, c: c, r: max(rx, ry, rz)}
}

// rkShard is a blade of glass from its foot b along a, ln long and w wide
// at the foot, with sides faces.
func rkShard(rng *rand.Rand, b, a rkVec, ln, w float64, sides int) rkPart {
	a = a.unit()
	e1 := rkEast.add(a.mul(-a[0])).unit()
	e2 := a.cross(e1)
	var pl []rkPlane
	rot := rng.Float64() * 2 * math.Pi
	for i := 0; i < sides; i++ {
		th := rot + (float64(i)+(rng.Float64()-0.5)*0.4)*2*math.Pi/float64(sides)
		dir := e1.mul(math.Cos(th)).add(e2.mul(math.Sin(th)))
		ww := w * (0.8 + rng.Float64()*0.4)
		pl = append(pl, rkPlaneAt(dir.add(a.mul(ww/ln)), b.add(dir.mul(ww))))
	}
	pl = append(pl, rkPlaneAt(a.mul(-1), b.add(a.mul(-w))))
	if rng.Float64() < 0.5 { // a slanting break near the tip
		dir := e1.mul(math.Cos(rot)).add(e2.mul(math.Sin(rot)))
		pl = append(pl, rkPlaneAt(a.add(dir.mul(0.9)), b.add(a.mul(ln*(0.72+rng.Float64()*0.15)))))
	}
	return rkPart{pl: pl, c: b.add(a.mul(ln / 3)), r: w}
}

// --- rocks --------------------------------------------------------------------

// The grounds a boulder lies on, each with its own stone.
const (
	rkGranite = iota // grass, rocky hills, woods: grey granite, moss on some
	rkSand           // the desert: warm sandstone in layers
	rkBasalt         // the volcano's ash: dark basalt in columns
	rkSnow           // the snow: cold grey under caps of snow
	rkCavern         // the caves: dark, damp stone
	numRockGrounds
)

// rockGroundOf is the stone of the rocks lying on tile t.
func rockGroundOf(t sim.Tile) int {
	switch t {
	case sim.TileDesert, sim.TileSand:
		return rkSand
	case sim.TileAsh, sim.TileLava:
		return rkBasalt
	case sim.TileSnow, sim.TileIce:
		return rkSnow
	case sim.TileCave:
		return rkCavern
	}
	return rkGranite
}

// rkMoss is moss in patches on the faces turned up.
func rkMoss(seed uint32) rkDecor {
	return func(p, n rkVec, l float64, _ *rkPart) (ramp, float64, bool) {
		m := vnoise(p[0]*0.32, p[1]*0.5+p[2]*0.3, seed) + (vnoise(p[0]*1.1, p[1]*1.1+p[2], seed+1)-0.5)*0.35
		if n[2] > 0.6 && m > 0.6 {
			return rkMossRamp, l*0.8 - 0.08, true
		}
		return nil, 0, false
	}
}

// rkSnowCap is snow lying on the faces turned up, hanging a little over the
// edges.
func rkSnowCap(seed uint32) rkDecor {
	return func(p, n rkVec, l float64, pt *rkPart) (ramp, float64, bool) {
		hang := 0.15 + 0.8*vnoise(p[0]*0.7, p[1]*0.7, seed)
		if n[2] > 0.7-0.25*vnoise(p[0]*0.5, p[1]*0.5+p[2]*0.4, seed+1) || n[2] > -0.1 && pt.below(p) < hang {
			return rkSnowRamp, l*0.85 + 0.22, true
		}
		return nil, 0, false
	}
}

// below is how far p lies under the top face of pt (the face most turned up).
func (pt *rkPart) below(p rkVec) float64 {
	best, bn := -1.0, rkPlane{}
	for _, q := range pt.pl {
		if q.n[2] > best {
			best, bn = q.n[2], q
		}
	}
	if best <= 0 {
		return math.Inf(1)
	}
	return (bn.d - bn.n.dot(p)) / bn.n[2]
}

// rkLayers are the beds of sandstone across its sides: a dark groove and a
// lit lip under it.
func rkLayers(seed uint32) rkDecor {
	return func(p, n rkVec, l float64, pt *rkPart) (ramp, float64, bool) {
		if n[2] > 0.7 {
			return nil, 0, false
		}
		q := p[2]*0.9 + p[0]*0.05 + (vnoise(p[0]*0.25, p[1]*0.25, seed)-0.5)*1.4
		switch f := math.Mod(q+100, 2.6); {
		case f < 0.4:
			return pt.mat.tone, l - 0.22, true
		case f < 0.75:
			return pt.mat.tone, l + 0.1, true
		}
		return nil, 0, false
	}
}

// rkVeins are veins of jade winding through the stone.
func rkVeins(seed uint32) rkDecor {
	return func(p, n rkVec, l float64, _ *rkPart) (ramp, float64, bool) {
		v := vnoise(p[0]*0.22+p[2]*0.17, p[1]*0.22-p[2]*0.15, seed)
		u := vnoise(p[0]*0.2-p[2]*0.2, p[1]*0.2+p[2]*0.1+7, seed+5)
		if math.Abs(v-0.5) < 0.05 || math.Abs(u-0.5) < 0.035 {
			return rkJadeVein, 0.12 + l*0.75, true
		}
		return nil, 0, false
	}
}

// rkSeams are seams of coal laid through the stone, tilted, ragged at the
// edges, a dull sheen along the top of each.
func rkSeams(seed uint32, tilt float64) rkDecor {
	return func(p, n rkVec, l float64, _ *rkPart) (ramp, float64, bool) {
		q := p[2] + p[0]*tilt + p[1]*0.2 + (vnoise(p[0]*0.35, p[1]*0.35+p[2]*0.2, seed)-0.5)*1.6
		switch f := math.Mod(q+100, 4.4); {
		case f < 2.1:
			return rkCoalSeam, l * 0.8, true
		case f < 2.35 && l > 0.45:
			return rkCoalSeam, 1, true
		}
		return nil, 0, false
	}
}

// rkSplit parts block p in two along an upright plane through its middle
// turned to a (about the vertical), the second half with a top of its own,
// drop of its height lower, so the block reads as two stones.
func rkSplit(rng *rand.Rand, p rkPart, a, drop, tilt float64) (rkPart, rkPart) {
	n := rkVec{math.Cos(a), math.Sin(a), (rng.Float64() - 0.5) * 0.3}.unit()
	d := n.dot(p.c) + (rng.Float64()-0.5)*p.r*0.3
	top := p.pl[1]
	topZ := (top.d - top.n[0]*p.c[0] - top.n[1]*p.c[1]) / top.n[2]
	a1, b1 := p, p
	a1.pl = append(append([]rkPlane{}, p.pl...), rkPlane{n, d})
	b1.pl = append(append([]rkPlane{}, p.pl...), rkPlane{n.mul(-1), -d})
	bc := p.c.add(rkVec{n[0], n[1], 0}.mul(p.r * 0.5))
	b1.pl[1] = rkPlaneAt(rkVec{(rng.Float64()*2 - 1) * tilt, (rng.Float64()*2 - 1) * tilt, 1}, rkVec{bc[0], bc[1], topZ * (1 - drop)})
	b1.c = bc
	return a1, b1
}

// rkLump is a rounded stone sitting on the ground, worn smooth: the caves'.
func rkLump(rng *rand.Rand, cx, cy, rx, ry, h float64, facets int) rkPart {
	p := rkNodule(rng, rkVec{cx, cy, h * 0.42}, rx, ry, h*0.58, facets, 0.08)
	p.pl = append(p.pl, rkPlane{rkDown, 0})
	return p
}

// rkHost lays out the blocks of a rock of ground g, about one unit across,
// mirrored when flip is -1; small rocks have fewer blocks.
func rkHost(s *rkScene, rng *rand.Rand, g int, small bool, flip float64) {
	j := func(a float64) float64 { return (rng.Float64()*2 - 1) * a }
	sz := func() float64 { return 0.88 + rng.Float64()*0.24 }
	put := func(p rkPart, m *rkMat, crack float64) {
		p.mat, p.crack = m, crack
		p.nudge = j(0.04)
		s.add(p)
	}
	x := func(v float64) float64 { return v * flip }
	split := func(p rkPart, m *rkMat, crack float64) {
		ang := j(0.5)
		if flip < 0 {
			ang += math.Pi
		}
		a, b := rkSplit(rng, p, ang, 0.12+rng.Float64()*0.15, 0.25)
		put(a, m, crack)
		put(b, m, crack)
	}
	switch g {
	case rkSand: // a wide slab, sometimes with a cap of harder stone on it
		m := rkSandMat
		h := 0.26 * sz()
		put(rkChunk(rng, x(j(0.03)), 0.12, 0, 0.36*sz(), 0.2, h, 7, 2, 0.05, 0.1), m, 0.25)
		if !small || rng.Float64() < 0.5 {
			put(rkChunk(rng, x(-0.06+j(0.06)), 0.14+j(0.02), h, 0.22*sz(), 0.14, 0.17*sz(), 6, 2, 0.12, 0.15), m, 0.3)
		}
		put(rkChunk(rng, x(0.31+j(0.03)), 0.02, 0, 0.14*sz(), 0.12, 0.16*sz(), 6, 1, 0.15, 0.25), m, 0.3)
		if !small {
			put(rkChunk(rng, x(-0.3+j(0.03)), 0, 0, 0.12*sz(), 0.1, 0.11*sz(), 6, 1, 0.2, 0.3), m, 0.3)
		}
		put(rkChunk(rng, x(0.47+j(0.03)), -0.1, 0, 0.045, 0.04, 0.045, 5, 1, 0.3, 0.4), m, 0)
	case rkBasalt: // columns, packed six-sided, some broken, one fallen
		m := rkBasaltMat
		r := 0.105
		type col struct{ x, y, h float64 }
		cols := []col{{0, 0.16, 0.5}, {-0.18, 0.13, 0.36}, {0.18, 0.14, 0.42}, {-0.09, -0.01, 0.26}, {0.09, 0, 0.3}, {0.27, 0.0, 0.18}, {-0.27, -0.02, 0.14}}
		if small { // fewer, stouter
			r = 0.15
			cols = []col{{-0.13, 0.12, 0.42}, {0.13, 0.13, 0.5}, {0, -0.06, 0.26}, {0.27, -0.04, 0.16}}
		}
		for _, c := range cols {
			put(rkColumn(rng, x(c.x+j(0.012)), c.y+j(0.012), r*sz(), c.h*sz(), j(0.2), 0.22), m, 0.12)
		}
		if !small && rng.Float64() < 0.6 {
			put(rkFallen(rng, x(0.05+j(0.08)), -0.17, 0.06, 0.32*sz()), m, 0.1)
		}
		put(rkChunk(rng, x(-0.42+j(0.03)), -0.12, 0, 0.05, 0.04, 0.05, 5, 1, 0.3, 0.4), m, 0)
	case rkCavern: // stones worn round by the damp
		m := rkCaveMat
		put(rkLump(rng, x(j(0.04)), 0.1, 0.3*sz(), 0.2, 0.44*sz(), 13), m, 0.25)
		put(rkLump(rng, x(0.27+j(0.03)), 0.1, 0.18*sz(), 0.15, 0.28*sz(), 11), m, 0.25)
		if !small || rng.Float64() < 0.5 {
			put(rkLump(rng, x(-0.26+j(0.03)), 0.02, 0.15*sz(), 0.12, 0.2*sz(), 10), m, 0.25)
		}
		if !small {
			put(rkLump(rng, x(0.06+j(0.05)), -0.08, 0.14*sz(), 0.09, 0.14*sz(), 9), m, 0.2)
		}
		put(rkChunk(rng, x(0.42+j(0.03)), -0.12+j(0.02), 0, 0.05, 0.04, 0.05, 6, 1, 0.3, 0.6), m, 0)
	default: // granite, and the snow's stone: blocks, the big ones split
		m := rkGraniteMat
		main := rkChunk(rng, x(j(0.04)), 0.1+j(0.02), 0, 0.3*sz(), 0.2, 0.42*sz(), 6+rng.IntN(2), 3, 0.15, 0.25)
		split(main, m, 0.2)
		right := rkChunk(rng, x(0.25+j(0.03)), 0.12+j(0.02), 0, 0.2*sz(), 0.16, 0.3*sz(), 5+rng.IntN(2), 2, 0.2, 0.3)
		if small {
			put(right, m, 0.2)
		} else {
			split(right, m, 0.2)
		}
		if !small || rng.Float64() < 0.5 {
			put(rkChunk(rng, x(-0.27+j(0.03)), 0.01, 0, 0.15*sz(), 0.13, 0.2*sz(), 5, 2, 0.25, 0.35), m, 0.2)
		}
		if !small {
			put(rkChunk(rng, x(0.04+j(0.05)), -0.1, 0, 0.16*sz(), 0.09, 0.13*sz(), 5, 1, 0.3, 0.4), m, 0.15)
			put(rkChunk(rng, x(-0.14+j(0.04)), -0.11, 0, 0.09*sz(), 0.07, 0.08*sz(), 5, 1, 0.3, 0.45), m, 0.1)
		}
		put(rkChunk(rng, x(0.42+j(0.03)), -0.12+j(0.02), 0, 0.05, 0.04, 0.05, 5, 1, 0.3, 0.5), m, 0)
		if !small && rng.Float64() < 0.5 {
			put(rkChunk(rng, x(-0.44+j(0.02)), -0.08, 0, 0.035, 0.03, 0.035, 5, 1, 0.3, 0.5), m, 0)
		}
	}
}

// rkDecorate lays the ground's look over the host blocks.
func rkDecorate(s *rkScene, g int, moss bool) {
	var d rkDecor
	switch {
	case g == rkSnow:
		d = rkSnowCap(s.seed + 11)
	case g == rkSand:
		d = rkLayers(s.seed + 12)
	case g == rkGranite && moss:
		d = rkMoss(s.seed + 13)
	}
	for i := range s.parts {
		if !s.parts[i].ore {
			s.parts[i].decor = d
		}
	}
}

// boulderScene is boulder variant v on ground g, small or big: as high as
// the boulders have stood beside the player, 0.42 and 0.65 of him.
func boulderScene(g, v int, big bool) *rkScene {
	s := &rkScene{seed: uint32(g*97+v*13) + 400}
	if big {
		s.seed += 50
	}
	rng := rand.New(rand.NewPCG(uint64(s.seed), 0xb01d))
	flip := 1.0
	if v%2 == 1 {
		flip = -1
	}
	rkHost(s, rng, g, !big, flip)
	rkDecorate(s, g, v != 1)
	h := 13 - 0.33 + float64(v%3)*0.34 // 39 to 41 image pixels at HD
	if big {
		h = 20 - 0.33 - float64(v%3)*0.67 // 60 down to 56
	}
	s.fit(h, 0.6)
	return s
}

// boulderCanvas draws boulder variant v of ground g, small or big, and gives
// its ground point.
func boulderCanvas(g, v int, big bool) (*canvas, float64, float64) {
	return boulderScene(g, v, big).draw(hdK(), 5)
}

// The ores, as oreScene numbers them.
const (
	rkFlint = iota
	rkJade
	rkAgate
	rkObsidian
	rkCoalOre
	numRockOres
)

// How high an ore vein and a coal seam stand, in art pixels (the outline
// adds a third): 0.6 and 0.63 of the player.
const (
	oreHeight  = 18.67 - 0.33
	coalHeight = 19.67 - 0.33
)

// rkSites picks up to n points on the faces of the host blocks seen in px,
// at least gap image pixels apart, where ok says.
func rkSites(px []rkPix, w int, rng *rand.Rand, n int, gap float64, ok func(q *rkPix) bool) []rkPix {
	h := len(px) / w
	var cand []int
	for i, q := range px {
		x, y := i%w, i/w
		if q.part < 0 || !ok(&px[i]) || x < 2 || y < 2 || x >= w-2 || y >= h-2 {
			continue
		}
		inner := true
		for _, d := range [4]int{-2, 2, -2 * w, 2 * w} {
			if px[i+d].part < 0 {
				inner = false
			}
		}
		if inner {
			cand = append(cand, i)
		}
	}
	var got []rkPix
	var at [][2]float64
	for _, k := range rng.Perm(len(cand)) {
		i := cand[k]
		x, y := float64(i%w), float64(i/w)
		far := true
		for _, a := range at {
			if math.Hypot(a[0]-x, a[1]-y) < gap {
				far = false
				break
			}
		}
		if far {
			got = append(got, px[i])
			at = append(at, [2]float64{x, y})
			if len(got) == n {
				break
			}
		}
	}
	return got
}

// oreScene is variant v of ore o: flint nodules in granite, jade veins and
// lumps in the caves' stone, banded agate in sandstone, blades of obsidian
// out of basalt, or a coal seam.
func oreScene(o, v int) *rkScene {
	s := &rkScene{seed: uint32(o*131+v*17) + 900}
	rng := rand.New(rand.NewPCG(uint64(s.seed), 0x0e0e))
	flip := 1.0
	if v%2 == 1 {
		flip = -1
	}
	host := [...]int{rkGranite, rkCavern, rkSand, rkBasalt, rkGranite}[o]
	rkHost(s, rng, host, false, flip)
	switch o {
	case rkAgate:
		rkDecorate(s, rkSand, false)
	case rkJade:
		for i := range s.parts {
			s.parts[i].decor = rkVeins(s.seed + 21)
		}
	case rkCoalOre:
		for i := range s.parts {
			s.parts[i].decor, s.parts[i].mat = rkSeams(s.seed+22, 0.22*flip), rkCoalHost
		}
	}
	stand, hostH := oreHeight, oreHeight
	switch o {
	case rkCoalOre:
		stand, hostH = coalHeight, coalHeight
	case rkObsidian: // the blades stand above the basalt
		hostH = oreHeight * 0.62
	}
	s.fit(hostH, 1)

	// Where the ore lies: points on the faces of the host seen from here.
	const k = HDDensity
	top, bot, left, right := s.extent()
	w, h := int((right-left+4)*k), int((top-bot+4)*k)
	ox, oy := (-left+2)*k, (top+2)*k
	px := s.cast(w, h, ox, oy, k, true)
	face := func(q *rkPix) bool { return q.n[2] > -0.1 }
	ore := func(p rkPart, m *rkMat, look int) {
		p.mat, p.look, p.ore = m, look, true
		p.nudge = (rng.Float64() - 0.5) * 0.06
		s.add(p)
	}
	switch o {
	case rkFlint: // dark glossy nodules, knobbly, some well out of the stone
		for i, q := range rkSites(px, w, rng, 5, 22, face) {
			r := 1.5 + rng.Float64()*0.9
			out := 0.1 + rng.Float64()*0.3
			if i < 2 {
				r, out = r+0.7, 0.6
			}
			c := q.p.add(q.n.mul(r * out))
			ore(rkNodule(rng, c, r*(1.05+rng.Float64()*0.3), r, r*0.9, 16, 0.12), rkFlintMat, rkGloss)
			if i < 3 { // a knob grown on it
				k := r * (0.45 + rng.Float64()*0.2)
				a := rng.Float64() * 2 * math.Pi
				kc := c.add(rkVec{math.Cos(a) * r * 0.8, math.Sin(a) * r * 0.5, r * 0.35})
				ore(rkNodule(rng, kc, k*1.1, k, k, 12, 0.1), rkFlintMat, rkGloss)
			}
		}
	case rkJade: // green lumps, faceted, and the veins they come from
		for i, q := range rkSites(px, w, rng, 6, 15, face) {
			r := 1.4 + rng.Float64()*1.5
			out := 0.1 + rng.Float64()*0.3
			if i%3 == 0 {
				out = 0.55
			}
			ore(rkNodule(rng, q.p.add(q.n.mul(r*out)), r*1.1, r, r, 9, 0.2), rkJadeMat, rkGem)
		}
	case rkAgate: // banded nodules broken open
		for i, q := range rkSites(px, w, rng, 4, 20, face) {
			r := 2.4 + rng.Float64()*1.3
			out := rng.Float64() * 0.3
			if i%3 == 0 {
				out = 0.5
			}
			p := rkNodule(rng, q.p.add(q.n.mul(r*out)), r*1.1, r, r*0.95, 12, 0.15)
			ba := rng.Float64() * math.Pi
			p.band = rkVec{math.Cos(ba) * 0.6, math.Sin(ba) * 0.4, 1}.unit()
			ore(p, rkAgateMat, rkBanded)
		}
	case rkObsidian: // blades of black glass, standing out of the basalt
		up := func(q *rkPix) bool { return q.n[2] > 0.5 }
		for i, q := range rkSites(px, w, rng, 7, 11, up) {
			ln := 5 + rng.Float64()*5
			if i < 2 {
				ln += 3
			}
			a := rkVec{(rng.Float64() - 0.5) * 0.9, (rng.Float64() - 0.6) * 0.6, 1}
			if q.p[0] < 0 {
				a[0] -= 0.3
			} else {
				a[0] += 0.3
			}
			ore(rkShard(rng, q.p.add(rkTop.mul(-0.6)), a, ln, 1+rng.Float64()*1.1, 4+rng.IntN(2)), rkGlassMat, rkGlass)
		}
	case rkCoalOre: // lumps of coal broken out of the seam
		for _, q := range rkSites(px, w, rng, 4, 18, face) {
			r := 1.4 + rng.Float64()*1.3
			ore(rkNodule(rng, q.p.add(q.n.mul(r*0.3)), r*1.2, r, r*0.9, 7, 0.25), rkCoalMat, rkCoal)
		}
	}
	// A piece fallen at the foot.
	if o != rkObsidian {
		mats := [...]*rkMat{rkFlintMat, rkJadeMat, rkAgateMat, rkGlassMat, rkCoalMat}
		looks := [...]int{rkGloss, rkGem, rkBanded, rkGlass, rkCoal}
		r := 1.1 + rng.Float64()*0.5
		_, bot, left, right := s.extent()
		fx := right - 2.5
		if rng.Float64() < 0.5 {
			fx = left + 2.5
		}
		ore(rkNodule(rng, rkVec{fx, (bot + 1.2) / rkSin, r * 0.7}, r*1.2, r, r*0.8, 9, 0.15), mats[o], looks[o])
	}
	s.fit(stand-float64(v%3)*0.34, 1)
	return s
}

// oreCanvas draws variant v of ore o and gives its ground point.
func oreCanvas(o, v int) (*canvas, float64, float64) {
	return oreScene(o, v).draw(hdK(), 5)
}

// pebblesScene is a few small stones of ground g lying together.
func pebblesScene(g int) *rkScene {
	s := &rkScene{seed: uint32(g)*7 + 1300}
	rng := rand.New(rand.NewPCG(uint64(s.seed), 0x9eb))
	m := [...]*rkMat{rkGraniteMat, rkSandMat, rkBasaltMat, rkGraniteMat, rkCaveMat}[g]
	for _, p := range [][4]float64{{0, 0.12, 0.2, 0.17}, {-0.32, 0, 0.26, 0.2}, {0.3, -0.02, 0.22, 0.15}, {0.62, 0.08, 0.1, 0.08}} {
		pt := rkChunk(rng, p[0], p[1], 0, p[2], p[2]*0.8, p[3], 5+rng.IntN(2), 2, 0.25, 0.6)
		pt.mat = m
		pt.nudge = (rng.Float64() - 0.5) * 0.08
		s.add(pt)
	}
	rkDecorate(s, g, false)
	s.fit(3.3, 0.8)
	return s
}

// rkBlock is a cut stone about c lying in the frame e1, e2, e3 (unit and
// square to each other; e3 faces out), r1 by r2 by r3, its faces a little
// askew and the edges of its outer face worn off by bevel.
func rkBlock(rng *rand.Rand, c, e1, e2, e3 rkVec, r1, r2, r3, bevel float64) rkPart {
	j := func() float64 { return (rng.Float64() - 0.5) * 0.22 }
	var pl []rkPlane
	face := func(n rkVec, r float64) {
		n = n.add(e1.mul(j())).add(e2.mul(j())).add(e3.mul(j())).unit()
		pl = append(pl, rkPlane{n, n.dot(c) + r})
	}
	face(e1, r1)
	face(e1.mul(-1), r1)
	face(e2, r2)
	face(e2.mul(-1), r2)
	face(e3, r3)
	face(e3.mul(-1), r3)
	for _, s := range [4][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		n := e1.mul(s[0]).add(e2.mul(s[1])).add(e3).unit()
		r := (math.Abs(s[0])*r1 + math.Abs(s[1])*r2 + r3) / math.Sqrt2 * (1 - bevel*(0.5+rng.Float64()))
		pl = append(pl, rkPlane{n, n.dot(c) + r})
	}
	return rkPart{pl: pl, c: c, r: max(r1, r2, r3)}
}

var (
	rkClayMat  = &rkMat{tone: mkRamp(0x3f333b, 0x533b41, 0x6d4e4b, 0x867066), ink: hex(0x2b222a), rim: hex(0x533b41), glint: hex(0xb49d7e)}
	rkSootMat  = &rkMat{tone: mkRamp(0x1a1f2e, 0x1a1f2e, 0x282e3b), ink: hex(0x1a1f2e), rim: hex(0x1a1f2e), glint: hex(0x282e3b)}
	rkFireRamp = mkRamp(0x5c2b34, 0x8d3649, 0xc33c40, 0xe47259, 0xe1a845, 0xeecb90, 0xeff37c)
	rkHoleRamp = mkRamp(0x1a1f2e, 0x1a1f2e, 0x282e3b, 0x343f41)
	rkWarmRamp = mkRamp(0x3c252b, 0x684039, 0x8d4830, 0xc57835, 0xe1a845)
)

// furnaceScene is the furnace: a dome of fieldstones bedded in clay, an
// arched mouth to the front, dark or full of fire (lit), a lintel over it,
// a hearth stone before it and a short chimney of stones on its crown.
func furnaceScene(lit bool) *rkScene {
	s := &rkScene{seed: 1500}
	rng := rand.New(rand.NewPCG(1500, 0xf0))
	const a, b, c, cy = 9.5, 8.0, 12.0, 2.0
	const mw, mh = 3.4, 6.4 // the mouth: half its width, its height
	inMouth := func(x, z float64) bool {
		if math.Abs(x) >= mw {
			return false
		}
		return z < mh-2.4+2.4*math.Sqrt(1-(x/mw)*(x/mw))
	}
	front := func(z float64) float64 { // y of the dome's face at height z, in front
		return cy - b*math.Sqrt(max(0, 1-(z/c)*(z/c)))
	}
	// Firelight on the stones round the mouth.
	glow := func(p, n rkVec, l float64, pt *rkPart) (ramp, float64, bool) {
		p = s.design(p)
		d := math.Hypot(p[0]/1.3, p[2]-mh*0.4)
		if !lit || n[1] > 0.1 || d > mw+3.5 {
			return nil, 0, false
		}
		return rkWarmRamp, l*0.6 + 0.55*(1-d/(mw+3.5)), true
	}
	// The clay inside, which shows between the stones and in the mouth.
	core := rkNodule(rng, rkVec{0, cy, 0}, a-1, b-1, c-1, 40, 0)
	core.pl = append(core.pl, rkPlane{rkDown, 0})
	core.mat = rkClayMat
	core.decor = func(p, n rkVec, l float64, pt *rkPart) (ramp, float64, bool) {
		p = s.design(p)
		if p[1] > cy || !inMouth(p[0], p[2]) {
			return nil, 0, false
		}
		if lit {
			t := 1 - p[2]/mh*0.8 - math.Abs(p[0])/mw*0.35 + (vnoise(p[0]*1.3, p[2]*1.1, 1501)-0.5)*0.35
			return rkFireRamp, t, true
		}
		t := 0.15 + p[2]/mh*0.2
		if p[2] < 1.2 {
			t = 0.8 // ash on the floor
		}
		return rkHoleRamp, t, true
	}
	s.add(core)
	mats := [...]*rkMat{rkGraniteMat, rkGraniteMat, rkCaveMat, rkSandMat}
	stone := func(p rkPart) {
		p.mat = mats[rng.IntN(len(mats))]
		p.nudge = (rng.Float64() - 0.5) * 0.08
		p.crack = 0.15
		p.decor = glow
		s.add(p)
	}
	// The dome, in rings, each laid over the joints of the one below.
	const rows = 6
	for k := 0; k < rows; k++ {
		phi := (float64(k) + 0.45) / rows * 1.3
		circ := 2 * math.Pi * math.Sqrt((a*a+b*b)/2) * math.Cos(phi)
		n := max(5, int(circ/3.6))
		hgt := 1.15 * (1 - float64(k)*0.06)
		for i := 0; i < n; i++ {
			th := (float64(i) + 0.5*float64(k%2) + (rng.Float64()-0.5)*0.2) * 2 * math.Pi / float64(n)
			if math.Sin(th) > 0.6 {
				continue // round the back, unseen
			}
			p := rkVec{a * math.Cos(phi) * math.Cos(th), cy + b*math.Cos(phi)*math.Sin(th), c * math.Sin(phi)}
			if p[1] < cy && inMouth(p[0]*0.9, p[2]-0.7) {
				continue
			}
			nrm := rkVec{math.Cos(phi) * math.Cos(th) / a, math.Cos(phi) * math.Sin(th) / b, math.Sin(phi) / c}.unit()
			e1 := rkVec{-math.Sin(th) * a, math.Cos(th) * b, 0}.unit()
			e2 := nrm.cross(e1)
			ln := circ / float64(n) / 2 * (0.82 + rng.Float64()*0.12)
			stone(rkBlock(rng, p.add(nrm.mul(0.25)), e1, e2, nrm, ln, hgt*(0.9+rng.Float64()*0.15), 0.9, 0.4))
		}
	}
	// The lintel over the mouth, and the hearth stone before it.
	lz := mh + 0.5
	ln := rkVec{0, -math.Sqrt(max(0, 1-(lz/c)*(lz/c))) / b, lz / c / c}.unit()
	stone(rkBlock(rng, rkVec{0, front(lz) - 0.5, lz}, rkEast, ln.cross(rkEast), ln, mw+1.4, 1.05, 1.3, 0.3))
	hearth := rkChunk(rng, 0, front(0)-1.4, 0, mw+1.6, 1.7, 0.7, 7, 1, 0.04, 0.3)
	hearth.mat, hearth.crack, hearth.decor = rkGraniteMat, 0.2, glow
	s.add(hearth)
	// The chimney: two rings of stones round a sooty flue.
	top := rkVec{0, cy + 1.2, c - 0.8}
	flue := rkColumn(rng, top[0], top[1], 1.6, 0, 0, 0)
	flue.pl[0] = rkPlane{rkDown, -(c - 3)}
	flue.pl[1] = rkPlane{rkTop, c + 2.9}
	flue.mat = rkSootMat
	s.add(flue)
	for k := 0; k < 2; k++ {
		for i := 0; i < 6; i++ {
			th := (float64(i) + 0.5*float64(k)) * math.Pi / 3
			out := rkVec{math.Cos(th), math.Sin(th), 0}
			e1 := rkVec{-math.Sin(th), math.Cos(th), 0}
			p := top.add(out.mul(2.3)).add(rkVec{0, 0, 0.9 + float64(k)*1.7})
			stone(rkBlock(rng, p, e1, rkTop, out, 1.25, 0.82, 0.75, 0.35))
		}
	}
	s.fit(24.33-0.33, 0.5)
	return s
}
