package client

import (
	"image/color"
	"math"
)

// The critters' painter: a part drawn with body keeps, for each of its
// pixels, the way its surface faces, how much it is shadowed and what it is
// made of; done lights them all at the end as the painted dinosaurs are lit
// (internal/dinopaint): from the top left, a little towards the viewer, the
// far side of a form deep in a cool shadow, a rim of light along the edges
// turned away, a contact shadow where a part lies over another, a glint on
// what shines, and an outline in the part's own darkest colour, lighter on
// the lit side. Each material has its own grain: fur in locks, scales that
// are each a little dome, a shell's hard gloss, a wet skin's soft one.
//
// Parts drawn the old way (fill with a shader) keep their colours, so a
// drawing can be moved over a part at a time; eyes, teeth and claws are
// still drawn flat over the lit parts.

// surfKind is what a surface is made of.
type surfKind uint8

const (
	sMatte surfKind = iota // smooth hide or skin
	sFur                   // locks of hair lying along flow
	sScale                 // rows of scales, each a little dome
	sShell                 // a crab's or a scorpion's armour: hard, finely pitted, shining
	sWet                   // moist skin: mottled, softly shining
	sPlate                 // big plates edge to edge: a belly's scutes, a shell's segments
)

// surf is a material.
type surf struct {
	r     ramp // its tones from deep shadow to the highlight (richRamp)
	kind  surfKind
	gloss float64 // how bright its glint is (0: none)
	size  float64 // the grain: a scale's width, a lock's length, image pixels (0: a default)
	flow  float64 // fur: the way it lies, radians from +x (0: back, along -x... see furFlow)
	grain float64 // how strongly the grain shows (0: 1)
	seed  uint32
	// uv maps an image point to where on the surface it lies (image pixels),
	// so scales follow a tube round its bends; nil: as drawn
	uv func(x, y float64) (float64, float64)
}

// furFlow is fur combed back from the head of a beast facing right, and a
// little down.
const furFlow = math.Pi - 0.35

var (
	shadowHue = hex(0x2a2246) // what the shadows lean to: a cool violet
	lightHue  = hex(0xfff0c8) // and the highlights: warm sunlight
)

// richRamp makes nine tones of a ramp: a little deeper than its darkest and
// lighter than its lightest, the dark end shifted towards a cool violet and
// the light end towards warm sunlight, as a painter's would be.
func richRamp(r ramp) ramp {
	const n = 9
	out := make(ramp, n)
	m := len(r) - 1
	for i := range out {
		t := float64(i)/float64(n-1)*1.22 - 0.13
		var c color.RGBA
		switch {
		case t < 0:
			c = mix(r[0], colInk, -t*3.4)
		case t > 1:
			c = mix(r[m], colWhite, (t-1)*1.3)
		default:
			u := t * float64(m)
			k := min(m-1, int(u))
			c = mix(r[k], r[k+1], u-float64(k))
		}
		if t < 0.46 {
			c = mix(c, shadowHue, (0.46-t)*0.5)
		}
		if t > 0.62 {
			c = mix(c, lightHue, (t-0.62)*0.32)
		}
		out[i] = withAlpha(c, 255)
	}
	return out
}

// newSurf is a surface of kind k in the tones of r.
func newSurf(r ramp, k surfKind, gloss float64, seed uint32) *surf {
	return &surf{r: richRamp(r), kind: k, gloss: gloss, seed: seed}
}

// with is a copy of s with its grain size and flow set.
func (s *surf) with(size, flow float64) *surf {
	c := *s
	c.size, c.flow = size, flow
	return &c
}

// look is how a part is lit.
type look struct {
	round float64 // how far in from its edge it rounds over, art pixels (0: 2)
	bias  float64 // lighter (+) or darker
	far   bool    // on the far side of the body: in its shadow, no glint
	flat  float64 // 0 to 1: how much flatter than round its face is (a plate, a shell)
	soft  bool    // casts no shadow on what is behind it, and has no crease along its edge
}

// gbuf is what the painter keeps of each pixel of a critter's canvas.
type gbuf struct {
	nx, ny, nz []float32
	ao         []float32 // shadowed by what lies over it, 0 to 1
	tv         []float32 // its grain's lightness, and the part's bias
	sf         []*surf
	far        []bool
}

func newGbuf(n int) *gbuf {
	return &gbuf{nx: make([]float32, n), ny: make([]float32, n), nz: make([]float32, n),
		ao: make([]float32, n), tv: make([]float32, n), sf: make([]*surf, n), far: make([]bool, n)}
}

var paintLight = norm3f(-0.45, -0.8, 0.45)

func norm3f(x, y, z float64) [3]float64 {
	l := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / l, y / l, z / l}
}

// body draws part s of surface sf as a form, to be lit when the drawing is
// done. Where it lies over what is drawn already its edge is a crease and it
// casts a soft shadow down and to the right on it.
func (q *crit) body(s shape, sf *surf, o look) {
	c := q.c
	if q.g == nil {
		q.g = newGbuf(len(c.px))
	}
	g := q.g
	z := q.zoom
	round := o.round
	if round == 0 {
		round = 2
	}
	R := round * cK
	shx, shy := 0.55*cK, 1.1*cK // how far the shadow falls, image pixels
	if o.soft {
		shx, shy = 0, 0
	}
	x0, y0 := max(0, int(math.Floor(s.x0*z))-1+q.ox), max(0, int(math.Floor(s.y0*z))-1+q.oy)
	x1, y1 := min(c.w-1, int(math.Ceil((s.x1+shx)*z))+1+q.ox), min(c.h-1, int(math.Ceil((s.y1+shy)*z))+1+q.oy)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*c.w + x
			has := c.px[i].A >= 128
			lx, ly := (float64(x-q.ox)+0.5)/z, (float64(y-q.oy)+0.5)/z
			d := s.d(lx, ly)
			if d > 0 {
				if has && g.sf[i] != nil && shy > 0 {
					if ds := s.d(lx-shx, ly-shy); ds < 0 {
						a := float32(0.62 * smooth(clamp01(-ds/(1.4*cK))))
						g.ao[i] = max(g.ao[i], a)
					}
				}
				continue
			}
			// The form: the surface turns from facing the viewer deep inside
			// to facing out at the edge.
			gx := s.d(lx+0.75, ly) - s.d(lx-0.75, ly)
			gy := s.d(lx, ly+0.75) - s.d(lx, ly-0.75)
			gn := math.Hypot(gx, gy)
			e := 1 - clamp01(-d/R)
			e = e * (1 - 0.55*o.flat)
			nx, ny := 0.0, 0.0
			if gn > 1e-9 {
				nx, ny = gx/gn*e, gy/gn*e
			}
			nz := math.Sqrt(math.Max(0, 1-nx*nx-ny*ny))
			bx, by, tv := sf.grainAt(lx, ly, o.far)
			nx, ny = nx+bx, ny+by
			l := math.Sqrt(nx*nx + ny*ny + nz*nz)
			g.nx[i], g.ny[i], g.nz[i] = float32(nx/l), float32(ny/l), float32(nz/l)
			g.ao[i] = 0
			if has && !o.soft && d*z > -1.05 {
				g.ao[i] = 0.7 // the crease where it lies over another part
			}
			g.tv[i] = float32(tv + o.bias)
			g.sf[i], g.far[i] = sf, o.far
			c.px[i] = sf.r[len(sf.r)/2]
		}
	}
}

// coat gives what of s is painted already the surface sf, keeping its form:
// a pale belly, a stripe, a patch.
func (q *crit) coat(s shape, sf *surf) {
	if q.g == nil {
		return
	}
	c, g, z := q.c, q.g, q.zoom
	x0, y0 := max(0, int(math.Floor(s.x0*z))-1+q.ox), max(0, int(math.Floor(s.y0*z))-1+q.oy)
	x1, y1 := min(c.w-1, int(math.Ceil(s.x1*z))+1+q.ox), min(c.h-1, int(math.Ceil(s.y1*z))+1+q.oy)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*c.w + x
			if g.sf[i] == nil {
				continue
			}
			lx, ly := (float64(x-q.ox)+0.5)/z, (float64(y-q.oy)+0.5)/z
			if s.d(lx, ly) > 0 {
				continue
			}
			old := g.sf[i]
			_, _, ot := old.grainAt(lx, ly, g.far[i])
			_, _, nt := sf.grainAt(lx, ly, g.far[i])
			g.tv[i] += float32(nt - ot)
			g.sf[i] = sf
		}
	}
}

// shade darkens (by < 0) or lightens what of s is painted already, keeping
// its form: a groove, the shadow under a brow, a worn patch.
func (q *crit) shade(s shape, by float64) {
	if q.g == nil {
		return
	}
	c, g, z := q.c, q.g, q.zoom
	x0, y0 := max(0, int(math.Floor(s.x0*z))-1+q.ox), max(0, int(math.Floor(s.y0*z))-1+q.oy)
	x1, y1 := min(c.w-1, int(math.Ceil(s.x1*z))+1+q.ox), min(c.h-1, int(math.Ceil(s.y1*z))+1+q.oy)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*c.w + x
			if g.sf[i] == nil {
				continue
			}
			lx, ly := (float64(x-q.ox)+0.5)/z, (float64(y-q.oy)+0.5)/z
			if s.d(lx, ly) <= 0 {
				g.tv[i] += float32(by)
			}
		}
	}
}

// grainAt is what surface sf's grain does at image point (x, y) of the
// drawing: how it tilts the surface (bx, by) and lightens it (tv).
func (sf *surf) grainAt(x, y float64, far bool) (bx, by, tv float64) {
	k := sf.grain
	if k == 0 {
		k = 1
	}
	switch sf.kind {
	case sFur:
		size := sf.size
		if size == 0 {
			size = 7
		}
		flow := sf.flow
		if flow == 0 {
			flow = furFlow
		}
		di, u := lockAt(x, y, size*0.62, size, flow, sf.seed)
		tv = 0.075 * float64(di) * k
		// each lock rounds over its middle, across the way it lies
		cs, sn := math.Cos(flow), math.Sin(flow)
		bx, by = -sn*u*0.3*k, cs*u*0.3*k
	case sScale:
		size := sf.size
		if size == 0 {
			size = 4
		}
		ch := size * 0.8
		if sf.uv != nil {
			x, y = sf.uv(x, y)
		}
		row := math.Floor(y / ch)
		u := x/size + 0.5*math.Mod(row, 2)
		col := math.Floor(u)
		fx, fy := u-col-0.5, y/ch-row-0.45
		h := hashf(int(col), int(row), sf.seed)
		r := math.Hypot(fx*1.1, fy*1.25)
		// each scale a low dome; the seam between them sunk and dark
		bx, by = fx*0.32*k, fy*0.32*k
		tv = (h - 0.5) * 0.08 * k
		if r > 0.52 {
			tv -= 0.08 * k
		}
	case sShell:
		tv = (vnoise(x/2.2, y/2.2, sf.seed) - 0.5) * 0.08 * k
		if hashf(int(x/1.6), int(y/1.6), sf.seed+3) < 0.05 {
			tv -= 0.12 * k // a pit
		}
	case sWet:
		m := vnoise(x/9, y/9, sf.seed)
		tv = (m - 0.5) * 0.22 * k
		n := vnoise(x/2.5, y/2.5, sf.seed+7) - 0.5
		bx, by = n*0.2*k, -n*0.2*k
	case sPlate:
		tv = (vnoise(x/3, y/3, sf.seed) - 0.5) * 0.06 * k
	default:
		tv = (vnoise(x/3.5, y/3.5, sf.seed) - 0.5) * 0.07 * k
	}
	return bx, by, tv
}

// lockAt is the lock of hair over image point (x, y) in a coat of locks w
// wide and about l long, lying along flow: di is how much lighter it is there
// (-1 between the locks and at their tips, up to 2 down their lit side), u
// where across the lock the point is (-1 to 1).
func lockAt(x, y, w, l, flow float64, seed uint32) (di int, u float64) {
	cs, sn := math.Cos(flow), math.Sin(flow)
	// along the lock (fy grows towards its tip) and across it
	fy := x*cs + y*sn
	fx := -x*sn + y*cs
	r0 := int(math.Floor(fy / l))
	for row := r0 - 2; row <= r0; row++ {
		st := 0.5*float64(row&1) + hashf(row, 1, seed)
		j0 := int(math.Floor(fx/w - st))
		for j := j0 - 1; j <= j0+1; j++ {
			h1, h2, h3 := hashf(j, row, seed), hashf(j, row, seed+1), hashf(j, row, seed+2)
			t := (fy - (float64(row)+0.6*h2)*l) / (l * (1.5 + 0.9*h3))
			if t < 0 || t >= 1 {
				continue
			}
			cx := (float64(j) + st + 0.5 + 0.5*(h1-0.5) + 0.7*(h3-0.5)*t*t) * w
			u = (fx - cx) / (w * 0.7 * (1 - t*t*t))
			if u < -1 || u > 1 {
				continue
			}
			switch {
			case u > 0.55 || t > 0.9:
				di = -1
			case u < -0.15 && t > 0.12 && t < 0.8:
				di = 1
				if h1 > 0.78 && u < -0.45 {
					di = 2
				}
			case row == r0 && t < 0.2:
				di = -1
			}
			if h2 > 0.86 {
				di--
			}
			return di, u
		}
	}
	return -1, 0
}

// light lights every pixel a part was drawn on (body), in the tones of its
// surface.
func (q *crit) light() {
	g, c := q.g, q.c
	if g == nil {
		return
	}
	L := paintLight
	hx, hy, hz := L[0], L[1], L[2]+1
	hl := math.Sqrt(hx*hx + hy*hy + hz*hz)
	hx, hy, hz = hx/hl, hy/hl, hz/hl
	step := make([]int8, len(c.px))
	for i, sf := range g.sf {
		if sf == nil {
			continue
		}
		nx, ny, nz := float64(g.nx[i]), float64(g.ny[i]), float64(g.nz[i])
		v := 0.07 + 0.88*clamp01(0.28+0.72*(nx*L[0]+ny*L[1]+nz*L[2]))
		v -= 0.38 * float64(g.ao[i])
		rim := smoothstep(0.4, 0.95, 1-nz) * clamp01(nx*0.55+ny*0.83)
		v += 0.24 * rim
		v += float64(g.tv[i])
		if g.far[i] {
			v = v*0.8 - 0.06
		}
		n := len(sf.r)
		k := int(math.Floor(clamp01(v)*float64(n) - 1e-9))
		if sf.gloss > 0 && !g.far[i] && g.ao[i] < 0.3 {
			sp := math.Pow(clamp01(nx*hx+ny*hy+nz*hz), 28) * sf.gloss
			switch {
			case sp > 0.6:
				k = n + 1 // the glint itself
			case sp > 0.3:
				k = max(k, n-1)
			case sp > 0.15:
				k++
			}
		}
		step[i] = int8(max(0, min(n+1, k)))
	}
	// A lone pixel of a tone, none of its neighbours on the same surface
	// sharing it, takes their tone: clusters, not noise.
	clean := append([]int8(nil), step...)
	for y := 1; y < c.h-1; y++ {
		for x := 1; x < c.w-1; x++ {
			i := y*c.w + x
			if g.sf[i] == nil || g.sf[i].kind == sFur {
				continue
			}
			var count [12]int
			alone, nb := true, 0
			for _, j := range [4]int{i - 1, i + 1, i - c.w, i + c.w} {
				if g.sf[j] != g.sf[i] {
					continue
				}
				nb++
				if step[j] == step[i] {
					alone = false
				}
				count[min(11, int(step[j]))]++
			}
			if !alone || nb < 3 {
				continue
			}
			best := int(step[i])
			for s, n := range count {
				if n > count[min(11, best)] {
					best = s
				}
			}
			clean[i] = int8(best)
		}
	}
	for i, sf := range g.sf {
		if sf == nil {
			continue
		}
		n := len(sf.r)
		switch k := int(clean[i]); {
		case k >= n+1:
			c.px[i] = mix(sf.r[n-1], colWhite, 0.55)
		case k >= n:
			c.px[i] = mix(sf.r[n-1], colWhite, 0.25)
		default:
			c.px[i] = sf.r[k]
		}
	}
}

// outlineLit is the outline round a painted critter: each pixel round it
// takes the colour of the part it borders, deep on the side away from the
// light, lighter on the lit side (a selective outline), darkest under it.
func (q *crit) outlineLit(strength float64) {
	c := q.c
	src := make([]color.RGBA, len(c.px))
	copy(src, c.px)
	var sfs []*surf
	if q.g != nil {
		sfs = append(sfs, q.g.sf...)
	}
	get := func(x, y int) (color.RGBA, *surf) {
		if !c.in(x, y) {
			return color.RGBA{}, nil
		}
		i := y*c.w + x
		if sfs != nil {
			return src[i], sfs[i]
		}
		return src[i], nil
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if p, _ := get(x, y); p.A >= 128 {
				continue
			}
			// The side of the shape this pixel is on: from the solid
			// neighbour towards it.
			best, sx, sy := -1, 0, 0
			var bc color.RGBA
			var bs *surf
			for k, d := range [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}} {
				if n, s := get(x+d[0], y+d[1]); n.A >= 128 {
					if best < 0 {
						best, sx, sy, bc, bs = k, -d[0], -d[1], n, s
					}
				}
			}
			if best < 0 {
				continue
			}
			// lit: the pixel lies up or to the left of the shape
			lit := clamp01(0.5 + 0.5*(float64(sx)*paintLight[0]+float64(sy)*paintLight[1])/0.92)
			var col color.RGBA
			if bs != nil {
				deep := mix(bs.r[0], colInk, 0.45)
				col = mix(deep, bs.r[2], lit*0.85)
			} else {
				col = mix(withAlpha(bc, 255), colInk, strength*(1-0.35*lit))
			}
			c.px[y*c.w+x] = col
			if c.ol == nil {
				c.ol = make([]bool, len(c.px))
			}
			c.ol[y*c.w+x] = true
		}
	}
}

// glassEye is a round, wet eye at art point (x, y), r art pixels across
// its ball: dark round its rim, the iris lighter below where the light
// comes through it, the pupil (slit: a tall narrow one), a bright glint
// up on the lit side and a faint one below.
func (q *crit) glassEye(x, y, r float64, iris color.RGBA, slit bool) {
	q.flat(aEll(x, y, r+0.34, r+0.34), plain(mix(iris, colInk, 0.82)))
	ir := aEll(x, y, r, r)
	q.flat(ir, func(px, py int, _ float64) color.RGBA {
		ay := (float64(py)+0.5)/cK - y
		t := clamp01(0.5 + ay/(2*r))
		return mix(mix(iris, colInk, 0.45), mix(iris, colWhite, 0.18), t)
	})
	if slit {
		q.flat(aEll(x+0.05, y, r*0.26, r*0.86), plain(colInk))
	} else if r > 0.45 {
		q.flat(aEll(x+0.08, y+0.04, r*0.5, r*0.54), plain(colInk))
	}
	q.dot(x-r*0.38, y-r*0.4, 0, 0, colWhite)
	if r > 0.7 {
		q.dot(x-r*0.38, y-r*0.4, 1, 0, mix(colWhite, iris, 0.3))
		q.dot(x+r*0.4, y+r*0.42, 0, 0, mix(iris, colWhite, 0.5))
	}
}

// coatBy gives each painted pixel of s the surface pick says (nil: keeps
// its own), keeping its form: a pattern that follows the body, read off a
// tube's length and girth.
func (q *crit) coatBy(s shape, pick func(x, y float64) *surf) {
	if q.g == nil {
		return
	}
	c, g, z := q.c, q.g, q.zoom
	x0, y0 := max(0, int(math.Floor(s.x0*z))-1+q.ox), max(0, int(math.Floor(s.y0*z))-1+q.oy)
	x1, y1 := min(c.w-1, int(math.Ceil(s.x1*z))+1+q.ox), min(c.h-1, int(math.Ceil(s.y1*z))+1+q.oy)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*c.w + x
			if g.sf[i] == nil {
				continue
			}
			lx, ly := (float64(x-q.ox)+0.5)/z, (float64(y-q.oy)+0.5)/z
			if s.d(lx, ly) > 0 {
				continue
			}
			sf := pick(lx/cK, ly/cK)
			if sf == nil || sf == g.sf[i] {
				continue
			}
			_, _, ot := g.sf[i].grainAt(lx, ly, g.far[i])
			_, _, nt := sf.grainAt(lx, ly, g.far[i])
			g.tv[i] += float32(nt - ot)
			g.sf[i] = sf
		}
	}
}

// furry is a with the ragged edge of a coat: tufts about period art
// pixels apart standing up to amp out of it, every way round.
func (a shape) furry(amp, period float64, seed uint32) shape {
	amp, period = amp*cK, period*cK
	tri := func(u float64) float64 { return math.Abs(u-math.Floor(u)-0.5) * 2 }
	return shape{a.x0 - amp, a.y0 - amp, a.x1 + amp, a.y1 + amp, func(x, y float64) float64 {
		d := a.d(x, y)
		if d < -2*amp || d > 2*amp {
			return d
		}
		u, v := (x*0.8+y*0.6)/period, (-x*0.6+y*0.8)/period
		h := hashf(int(math.Floor(u)), int(math.Floor(v)), seed)
		return d - amp*(1-0.5*tri(u)-0.5*tri(v))*(0.5+0.5*h) + amp*0.3
	}}
}
