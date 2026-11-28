package client

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

// The player's figure as it is painted (the look the user chose from a
// pilot of three, "A"): the jointed caveman of limbs.go with its rig, its
// poses, its layers and every motion, drawn with real anatomy. The painters
// are here and in charpilot_head.go (the head, the face, the hair and the
// beard), charpilot_body.go (the bare trunk, the loincloth, a woman's fur
// top) and charpilot_limbs.go (the arms, the hands, the legs and the feet)
// and charpilot_wear.go (the clothes: the tunic and the dress, the sleeves,
// the trousers and their wraps, the boots).
//
// The look: a bigger face with bold dark eyes that catch a glint, a brow,
// a nose; a beard that goes with the hair; muscles lit as forms; elbows,
// wrists, hands, knees, calves and feet; warm where the blood shows (the
// cheeks, the nose, the hands, the knees and the feet); fur and hair in
// soft locks and strands; hue-shifted OKLab skin with a cool rim light; a
// selective outline that is a deep shade of what it bounds.
//
// Every part is a surface (pHeight: a slab of its outline with each muscle
// a rise on it) lit from the top left and cut into an OKLab ramp, then
// tidied of specks as a pixel artist would (pClean); the face is set pixel
// by pixel over the lit skull.
//
// pilotStyle is a developer's switch to set the old figure (charCurrent)
// beside this one: every hook (in cavefig.go, tallhead.go, tallfig.go,
// limbs.go, look.go's rimLayers and pix.go's outline) falls through to the
// old painters there. cmd/shot takes -charstyle old.

type charStyle uint8

const (
	charCurrent charStyle = iota // the old figure, for comparison only
	charRefined                  // the figure as it is drawn
)

var pilotStyle = charRefined

// SetCharStyle picks the player figure's painters by name, for comparing:
// A (the default) or old. Art already drawn keeps its style: set it before
// the game draws a player.
func SetCharStyle(name string) error {
	switch strings.ToLower(name) {
	case "", "a", "refined", "new":
		pilotStyle = charRefined
	case "old", "current", "0":
		pilotStyle = charCurrent
	default:
		return fmt.Errorf("unknown character style %q: A or old", name)
	}
	return nil
}

func pilotOn() bool { return pilotStyle != charCurrent }

// pSpec is how the figure is built and shaded: proportions in image pixels,
// and the shading.
type pSpec struct {
	// The head: the skull's half width and half height about (27, headY),
	// a woman's narrower by fHead.
	headRX, headRY, headY, fHead float64
	// The trunk, a man's from the front: half widths at the shoulders (y
	// 47.5), under the arms (53), the waist (66) and the hips (75).
	shoulder, chest, waist, hips float64
	// How thick the limbs are drawn, about the rig's own: the upper arm,
	// the forearm, the thigh, the calf; the fist's size, the foot's length.
	arm, fore, thigh, calf, fist, foot float64
	// The loincloth's top and hem rows, a man's.
	loinTop, loinHem float64
	// Shading: tones in the skin's ramp; the strength of the cool rim
	// light; how dark the outline is below and right, and above and left.
	tones           int
	rim             float64
	inkDark, inkLit float64
	lineK           float64 // how much darker the fine line between two parts is
	contrast        float64
	muscleK         float64
	furTones        int
	hairTones       int
	shadeTurn       float64 // how far the skin's darks turn their hue, towards red
	shadeSat        float64 // and how saturated they are
	warmEdge        float64 // how much warmer and richer the skin is where it turns from the light
	warmZones       float64 // how much redder the hands, feet, elbows, knees and face are
	furK            float64 // how strongly the locks of fur stand out from each other
	furClump        float64 // how wide a lock of fur is
	hairClump       float64 // how wide a clump of hair is where it hangs
	hairGrain       float64 // the fine strands through the hair's clumps
}

var pSpecA = pSpec{
	headRX: 7.4, headRY: 9, headY: 35.2, fHead: 0.5,
	shoulder: 10.6, chest: 9.6, waist: 7.4, hips: 9,
	arm: 1.08, fore: 1.08, thigh: 1.05, calf: 1.1, fist: 6, foot: 1.12,
	loinTop: 68.5, loinHem: 84,
	tones: 6, rim: 1, inkDark: 0.62, inkLit: 0.42, lineK: 0.28, contrast: 1, muscleK: 0.85,
	furTones: 9, hairTones: 8, shadeTurn: -0.07, shadeSat: 0.92, warmEdge: 0.2, warmZones: 0.75,
	furK: 1.7, furClump: 2.6, hairClump: 2.2, hairGrain: 0.12,
}

func pspec() *pSpec { return &pSpecA }

// --- colour --------------------------------------------------------------------

// OKLab, for ramps that keep their lightness steps even while their hue
// shifts: warm and saturated into the shadow, towards yellow in the light.

func srgbToLin(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linToSrgb(v float64) uint8 {
	v = clamp01(v)
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(v * 255))
}

// okLCh is c in OKLab's lightness, chroma and hue (radians).
func okLCh(c color.RGBA) (L, C, h float64) {
	r, g, b := srgbToLin(c.R), srgbToLin(c.G), srgbToLin(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	L = 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
	A := 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	B := 0.0259040371*l + 0.7827717662*m - 0.8086757660*s
	return L, math.Hypot(A, B), math.Atan2(B, A)
}

// okColor is the colour of OKLCh (L, C, h), its chroma cut until it fits.
func okColor(L, C, h float64) color.RGBA {
	for k := 0; k < 24; k++ {
		A, B := C*math.Cos(h), C*math.Sin(h)
		l := L + 0.3963377774*A + 0.2158037573*B
		m := L - 0.1055613458*A - 0.0638541728*B
		s := L - 0.0894841775*A - 1.2914855480*B
		l, m, s = l*l*l, m*m*m, s*s*s
		r := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
		g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
		b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
		if (r >= -0.002 && r <= 1.002 && g >= -0.002 && g <= 1.002 && b >= -0.002 && b <= 1.002) || C < 0.002 {
			return color.RGBA{linToSrgb(r), linToSrgb(g), linToSrgb(b), 255}
		}
		C *= 0.9
	}
	return color.RGBA{linToSrgb(L), linToSrgb(L), linToSrgb(L), 255}
}

// hueRamp spreads anchors (dark to light) over n tones from lightness lo
// to hi, turning the hue by shadeTurn radians at the dark end and lightTurn
// at the light end (positive turns towards yellow from red), and scaling
// the chroma by shadeSat in the darks and lightSat in the lights.
func hueRamp(anchors ramp, n int, lo, hi, shadeTurn, lightTurn, shadeSat, lightSat float64) ramp {
	type lch struct{ L, C, h float64 }
	as := make([]lch, len(anchors))
	for i, a := range anchors {
		L, C, h := okLCh(a)
		as[i] = lch{L, C, h}
	}
	at := func(L float64) (float64, float64) { // the anchors' chroma and hue at lightness L
		if L <= as[0].L {
			return as[0].C, as[0].h
		}
		for i := 1; i < len(as); i++ {
			if L <= as[i].L {
				t := (L - as[i-1].L) / math.Max(as[i].L-as[i-1].L, 1e-6)
				dh := math.Remainder(as[i].h-as[i-1].h, 2*math.Pi)
				return lerpf(as[i-1].C, as[i].C, t), as[i-1].h + dh*t
			}
		}
		return as[len(as)-1].C, as[len(as)-1].h
	}
	r := make(ramp, n)
	for k := range r {
		t := float64(k) / float64(n-1)
		L := lerpf(lo, hi, t)
		C, h := at(L)
		turn := lerpf(shadeTurn, lightTurn, t)
		sat := lerpf(shadeSat, lightSat, smooth(t))
		r[k] = okColor(L, C*sat, h+turn)
	}
	return r
}

// okL is c's OKLab lightness.
func okL(c color.RGBA) float64 { L, _, _ := okLCh(c); return L }

// shiftLC is c with its lightness scaled by kL, its chroma by kC and its
// hue turned by turn radians.
func shiftLC(c color.RGBA, kL, kC, turn float64) color.RGBA {
	L, C, h := okLCh(c)
	return okColor(L*kL, C*kC, h+turn)
}

// pRamps is the colours of one look in a style: the skin, the lips, the
// eyes and the brows; the hair, the beard; the fur and the thong.
type pRamps struct {
	skin, hair, beard, fur, belt ramp
	lip, iris, brow              color.RGBA
}

// pRampsFor makes the ramps of skin tone skin and hair colour hairCol.
func pRampsFor(skin, hairCol int, female bool) pRamps {
	sp := pspec()
	st := skinTones[skin]
	a := st.ramp
	Ld, Ll := okL(a[0]), okL(a[len(a)-1])
	var r pRamps
	// The skin: from a deep warm shadow, red-brown going on purple, to a
	// highlight going on yellow.
	r.skin = hueRamp(a, sp.tones, Ld-0.1, Ll+0.04, sp.shadeTurn, 0.1, sp.shadeSat, 0.85)
	if sp.warmEdge > 0 { // the light through the skin where it turns from the sun: warmer, richer
		for k := range r.skin {
			t := float64(k) / float64(len(r.skin)-1)
			w := sp.warmEdge * math.Exp(-math.Pow((t-0.42)/0.16, 2))
			r.skin[k] = shiftLC(r.skin[k], 1, 1+w, 0.18*w)
		}
	}
	r.lip = shiftLC(mix(st.blush, a[1], 0.45), 0.82, 1.1, -0.1)
	r.iris = irisColors[skin]

	h := hairColors[hairCol].ramp
	Lh0, Lh1 := okL(h[0]), okL(h[len(h)-1])
	r.hair = hueRamp(h, sp.hairTones, Lh0-0.06, Lh1+0.06, -0.18, 0.14, 1.1, 0.85)
	r.beard = hueRamp(h, sp.hairTones, Lh0-0.08, lerpf(Lh0, Lh1, 0.8), -0.14, 0.1, 1.05, 0.8)
	r.brow = r.hair[1]
	if okL(r.brow) < 0.3 {
		r.brow = mix(r.hair[2], r.skin[1], 0.25)
	}
	if female {
		r.brow = mix(r.brow, r.skin[sp.tones/2], 0.25)
	}

	// A tawny hide with the fur on: a warm grey-brown, darker strands, pale
	// tips; and a dark leather thong.
	fur := mkRamp(0x3a2a20, 0x5c4532, 0x86694c, 0xae9070, 0xd2b893)
	r.fur = hueRamp(fur, sp.furTones, 0.26, 0.84, -0.22, 0.12, 1.15, 0.8)
	r.belt = hueRamp(mkRamp(0x24150d, 0x3e2616, 0x5e3b20, 0x80562e), 5, 0.2, 0.55, -0.2, 0.08, 1.2, 1)
	return r
}

// --- shading -------------------------------------------------------------------

// pTone picks the tone of r for light v, in clean bands: hair, fur and
// leather keep crisp clumps (only the skin's bands wander: pToneI).
func pTone(r ramp, v float64, _, _ int) color.RGBA {
	v = 0.5 + (v-0.5)*pspec().contrast
	return r.at(max(0, min(len(r)-1, int(math.Floor(v*float64(len(r)))))))
}

// pToneI is the index of the skin's tone for light v at pixel (x, y), in
// clean bands.
func pToneI(n int, v float64, _, _ int) int {
	v = 0.5 + (v-0.5)*pspec().contrast
	return max(0, min(n-1, int(math.Floor(v*float64(n)))))
}

// pHeight is a surface over the figure: its height at a point, in image
// pixels, out of the screen.
type pHeight func(x, y float64) float64

// pLight is the light on surface h at pixel (x, y): its slope gives the
// normal, lit from the top left as everything is (lum).
func pLight(h pHeight, x, y int) float64 {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	const e = 0.6
	hx := (h(fx+e, fy) - h(fx-e, fy)) / (2 * e)
	hy := (h(fx, fy+e) - h(fx, fy-e)) / (2 * e)
	n := math.Sqrt(1 + hx*hx + hy*hy)
	return lum(-hx/n, -hy/n)
}

// pCrease is how deep a crease of surface h is at (x, y): how far it lies
// below the mean of the ring round it, r out.
func pCrease(h pHeight, x, y int, r float64) float64 {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	c := h(fx, fy)
	m := 0.0
	const n = 4
	for k := 0; k < n; k++ {
		a := float64(k) * math.Pi * 2 / n
		m += h(fx+r*math.Cos(a), fy+r*math.Sin(a))
	}
	return m/n - c
}

// edgeH is the height of a rounded slab whose edge is d away (negative
// inside), rounding over r in from the edge: a quarter circle.
func edgeH(d, r float64) float64 {
	t := clamp01(-d / r)
	return r * math.Sqrt(1-(1-t)*(1-t))
}

// bumpH is a muscle's rise at q of its radius from its middle (0 to 1 and
// beyond), a high: smooth to its edge.
func bumpH(q, high float64) float64 {
	if q >= 1 {
		return 0
	}
	u := 1 - q*q
	return high * u * math.Sqrt(u)
}

// ellQ is how far (x, y) is from the middle of an ellipse about (cx, cy),
// ra along the unit (ux, uy) and rb across it: 1 on its edge.
func ellQ(x, y, cx, cy, ux, uy, ra, rb float64) float64 {
	dx, dy := x-cx, y-cy
	a, b := (dx*ux+dy*uy)/ra, (dx*uy-dy*ux)/rb
	return math.Sqrt(a*a + b*b)
}

// --- the fine lines and the outline -----------------------------------------

// pLines draws the fine lines between the parts laid on layers (one over
// the next), as innerLines does for the old figure: but each a deeper shade
// of the part it is drawn on rather than that part mixed with ink (a light
// skin went grey), and none round a shoulder (pBlendShoulder). Then the
// shared outline round them all (pilotOutline).
func (g *figGen) pLines(layers ...*figLayer) {
	c0 := layers[0].c
	type key struct {
		l int
		p uint16
	}
	top := func(x, y int) (key, bool) {
		if !c0.in(x, y) {
			return key{}, false
		}
		i := y*c0.w + x
		for k := len(layers) - 1; k >= 0; k-- {
			if layers[k].c.px[i].A >= 128 {
				return key{k, layers[k].part[i]}, true
			}
		}
		return key{}, false
	}
	type mark struct{ l, i, shade int } // shade: the pixel behind the line, in the shadow of the part over it (-1: none)
	var marks []mark
	isMark := make([]bool, c0.w*c0.h)
	for y := 0; y < c0.h; y++ {
		for x := 0; x < c0.w; x++ {
			a, ok := top(x, y)
			if !ok {
				continue
			}
			for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
				b, ok := top(x+d[0], y+d[1])
				if ok && (b.l > a.l || b.l == a.l && b.p > a.p) {
					sh := -1
					if c, ok := top(x-d[0], y-d[1]); ok && c == a && b.l > a.l {
						sh = (y-d[1])*c0.w + x - d[0]
					}
					marks = append(marks, mark{a.l, y*c0.w + x, sh})
					isMark[y*c0.w+x] = true
					break
				}
			}
		}
	}
	zones := pilotZonesOf == g
	k := pspec().lineK
	for _, m := range marks {
		p := &layers[m.l].c.px[m.i]
		if zones && inZone(m.i, layers[m.l].c.w) {
			continue
		}
		kk := k
		if _, C, _ := okLCh(*p); C < 0.025 {
			// Cloth under cloth: the greys the player's colour goes on want
			// a deeper line, and a shadow a pixel wide behind it.
			kk = 0.4
			if m.shade >= 0 && !isMark[m.shade] {
				q := &layers[m.l].c.px[m.shade]
				if _, C, _ := okLCh(*q); C < 0.025 {
					*q = pLineOf(*q, 0.14)
					isMark[m.shade] = true
				}
			}
		}
		*p = pLineOf(*p, kk)
	}
	cs := make([]*canvas, len(layers))
	for i, l := range layers {
		cs[i] = l.c
	}
	pilotOutline(cs...)
}

// pLineOf is the fine line's colour on colour c: c darker by k, a little
// more saturated, turned a little towards red.
func pLineOf(c color.RGBA, k float64) color.RGBA {
	L, C, h := okLCh(c)
	return okColor(L*(1-k), math.Min(C*1.02, 0.14), h-0.04)
}


// pInkOf is the outline's colour next to colour c: c much darker, its
// chroma kept and its hue turned a little towards purple, so the line is
// a deeper shade of what it bounds rather than black; k is how dark.
func pInkOf(c color.RGBA, k float64) color.RGBA {
	L, C, h := okLCh(c)
	// turn towards purple (about 5.4 rad in OKLab)
	dh := math.Remainder(5.3-h, 2*math.Pi)
	return okColor(L*(1-k)*0.92+0.03, math.Min(C*1.05+0.012, 0.12), h+dh*0.22)
}

// pilotOutline gives the layers laid one over the next a shared outline
// round their joint silhouette, each rim pixel in the layer it borders: a
// deep shade of the colour it bounds, lighter above and to the left, where
// the light comes from, darker below and to the right.
func pilotOutline(layers ...*canvas) {
	sp := pspec()
	type rim struct {
		c    *canvas
		x, y int
		col  color.RGBA
	}
	top := func(x, y int) (*canvas, color.RGBA) {
		for i := len(layers) - 1; i >= 0; i-- {
			if layers[i].opaque(x, y) {
				return layers[i], layers[i].at(x, y)
			}
		}
		return nil, color.RGBA{}
	}
	var rims []rim
	c0 := layers[0]
	// A cool light off the sky on the edge away from the sun: the pixel
	// just inside the figure's right edge, where it is in shade.
	if sp.rim > 0 {
		var lit []rim
		for y := 0; y < c0.h; y++ {
			for x := 1; x < c0.w-1; x++ {
				c, col := top(x, y)
				if c == nil {
					continue
				}
				if r, _ := top(x+1, y); r != nil {
					continue
				}
				if l, _ := top(x-1, y); l == nil {
					continue // a sliver: leave it
				}
				L, C, h := okLCh(col)
				if L > 0.66 {
					continue
				}
				k := 0.34 * sp.rim
				rc := okColor(math.Min(L+0.11, 0.8), C*0.55, h) // lighter and greyer: the sky's light, not the sun's
				lit = append(lit, rim{c, x, y, mix(col, rc, k)})
			}
		}
		for _, r := range lit {
			r.c.put(r.x, r.y, r.col)
		}
	}
	for y := 0; y < c0.h; y++ {
		for x := 0; x < c0.w; x++ {
			if c, _ := top(x, y); c != nil {
				continue
			}
			// below, right, left, above: the side of the figure the rim is on
			for i, d := range [4][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} {
				if c, col := top(x+d[0], y+d[1]); c != nil {
					k := [4]float64{sp.inkDark, sp.inkDark, sp.inkLit, sp.inkLit}[i]
					rims = append(rims, rim{c, x, y, pInkOf(col, k)})
					break
				}
			}
		}
	}
	for _, r := range rims {
		r.c.put(r.x, r.y, r.col)
	}
}

// The shoulders: where an arm's cap meets the trunk there is no line
// between them (the deltoid runs into the trapezius and the chest), so the
// fine lines are left out round each shoulder's joint. Canvas pixels, for
// the figure being drawn.
var (
	pilotZones   [][3]float64
	pilotZonesOf *figGen
)

// pBlendShoulder leaves the fine line out within r of the shoulder (x, y).
func (g *figGen) pBlendShoulder(x, y, r float64) {
	if pilotZonesOf != g {
		pilotZones, pilotZonesOf = nil, g
	}
	cx, cy := g.f.fwd(x, y)
	pilotZones = append(pilotZones, [3]float64{cx + g.f.ox, cy + g.f.oy, r})
}

// inZone reports whether canvas pixel i of a canvas w wide is round a
// shoulder.
func inZone(i, w int) bool {
	x, y := float64(i%w)+0.5, float64(i/w)+0.5
	for _, z := range pilotZones {
		if math.Hypot(x-z[0], y-z[1]) < z[2] {
			return true
		}
	}
	return false
}

// pClean tidies the tones of part p on layer l as a pixel artist would:
// a pixel with at most one neighbour of its own colour (a speck, or the
// tip of a one-pixel spur in a band's edge) takes the colour most of its
// neighbours of the part have. passes times.
func (f *figHD) pClean(l *figLayer, p uint16, passes int) {
	c := l.c
	for range passes {
		src := make([]color.RGBA, len(c.px))
		copy(src, c.px)
		for y := 1; y < c.h-1; y++ {
			for x := 1; x < c.w-1; x++ {
				i := y*c.w + x
				if l.part[i] != p || src[i].A < 128 {
					continue
				}
				same, n := 0, 0
				var cols [8]color.RGBA
				var counts [8]int
				k := 0
				for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}} {
					j := (y+d[1])*c.w + x + d[0]
					if l.part[j] != p || src[j].A < 128 {
						continue
					}
					n++
					q := src[j]
					if q == src[i] {
						same++
						continue
					}
					found := false
					for m := 0; m < k; m++ {
						if cols[m] == q {
							counts[m]++
							found = true
						}
					}
					if !found {
						cols[k], counts[k] = q, 1
						k++
					}
				}
				if n < 5 || same > 1 {
					continue // on the part's edge, or one of a cluster
				}
				best := 0
				for m := 1; m < k; m++ {
					if counts[m] > counts[best] {
						best = m
					}
				}
				if k > 0 && counts[best] >= 3 {
					c.px[i] = cols[best]
				}
			}
		}
	}
}
