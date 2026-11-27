package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The interface's look, drawn at the screen's resolution: panels of dark
// weathered hide with a stitched leather rim, frames of wood with bone
// studs, parchment for notes, carved stone for the boss, sunken wells for
// slots and bars. Each is painted by one shader (panelSrc) into an image
// as big as it is on the screen, with its soft shadow round it, the first
// time it is drawn at that size; after that it is one image draw a frame
// (panelBakes). A bake not drawn for a few seconds is let go.

// panelStyle is a kind of panel.
type panelStyle uint8

const (
	psHide      panelStyle = iota // dark hide with a stitched leather rim: the HUD's panels
	psWood                        // a wooden frame with bone studs round dark hide: the bag, the hotbar
	psParchment                   // warm parchment with aged edges: notices, help
	psStone                       // carved stone: the boss
	psWell                        // a sunken well: slots, bar tracks
	psPlate                       // a soft dark plate: hints, labels, chips
	psLeather                     // a small leather tab: buttons, badges, cards
	psGlow                        // a soft glow round a rectangle
)

// panelLook is what a style is painted with: its colours (body, rim, the
// studs of a frame, the stitches' thread; straight colours, the alpha how
// much each shows) and the rim's width, in UI pixels.
type panelLook struct {
	body, rim, accent color.RGBA
	rimW              float64
	stitch            color.RGBA
}

var panelLooks = [...]panelLook{
	psHide:      {color.RGBA{40, 31, 25, 238}, color.RGBA{92, 64, 42, 255}, color.RGBA{}, 3, color.RGBA{196, 156, 102, 125}},
	psWood:      {color.RGBA{33, 26, 21, 242}, color.RGBA{128, 88, 54, 255}, color.RGBA{226, 214, 184, 255}, 5.5, color.RGBA{176, 136, 88, 95}},
	psParchment: {color.RGBA{232, 214, 172, 246}, color.RGBA{120, 84, 50, 255}, color.RGBA{160, 120, 70, 255}, 1.5, color.RGBA{}},
	psStone:     {color.RGBA{86, 82, 76, 250}, color.RGBA{132, 126, 116, 255}, color.RGBA{214, 200, 166, 255}, 4, color.RGBA{}},
	psWell:      {color.RGBA{20, 16, 13, 235}, color.RGBA{96, 78, 60, 255}, color.RGBA{0, 0, 0, 0}, 1, color.RGBA{}},
	psPlate:     {color.RGBA{22, 18, 15, 205}, color.RGBA{0, 0, 0, 0}, color.RGBA{0, 0, 0, 0}, 0, color.RGBA{}},
	psLeather:   {color.RGBA{58, 43, 31, 240}, color.RGBA{104, 74, 48, 255}, color.RGBA{}, 1.5, color.RGBA{196, 156, 102, 70}},
	psGlow:      {},
}

const panelSrc = `//kage:unit pixels

package main

var Size vec2    // the panel, in image pixels
var Margin float // room round it for the shadow
var Radius float
var K float      // image pixels to a UI pixel: the size of detail
var RimW float
var Style float
var Body vec4
var Rim vec4
var Accent vec4
var Stitch vec4
var Seed float
var Shadow float // how dark the shadow is

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))+Seed*13.7) * 43758.5453)
}

func vnoise(p vec2) float {
	i := floor(p)
	f := fract(p)
	u := f * f * (3.0 - 2.0*f)
	a := hash(i)
	b := hash(i + vec2(1.0, 0.0))
	c := hash(i + vec2(0.0, 1.0))
	d := hash(i + vec2(1.0, 1.0))
	return mix(mix(a, b, u.x), mix(c, d, u.x), u.y)
}

func fbm(p vec2) float {
	return vnoise(p)*0.55 + vnoise(p*2.03+vec2(5.2, 1.3))*0.3 + vnoise(p*4.11+vec2(2.7, 8.1))*0.15
}

func sdBox(p, b vec2, r float) float {
	q := abs(p) - b + r
	return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - r
}

// over puts colour c (straight alpha) over premultiplied dst.
func over(dst vec4, c vec4) vec4 {
	return vec4(c.rgb*c.a, c.a) + dst*(1.0-c.a)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := srcPos - Margin // srcPos runs over the rectangle drawn, from (0, 0)
	half := Size * 0.5
	p := pos - half
	r := min(Radius, min(half.x, half.y))
	d := sdBox(p, half, r)

	// The outward normal, from the distance's slope.
	e := 0.75
	n := vec2(sdBox(p+vec2(e, 0.0), half, r)-sdBox(p-vec2(e, 0.0), half, r), sdBox(p+vec2(0.0, e), half, r)-sdBox(p-vec2(0.0, e), half, r))
	if length(n) > 0.0 {
		n = normalize(n)
	}
	light := dot(n, normalize(vec2(-0.55, -1.0))) // 1 facing the light (top left)

	if Style > 6.5 {
		// A glow: the colour, fading out over the margin.
		t := 1.0 - smoothstep(-0.5*Margin, Margin, d)
		a := Body.a * t * t
		return vec4(Body.rgb*a, a)
	}

	// The soft shadow, down and a little right.
	ds := sdBox(p-vec2(0.6, 1.6)*K, half, r)
	sh := Shadow * (1.0 - smoothstep(-2.0*K, Margin, ds))
	out := vec4(0.0, 0.0, 0.0, sh*sh)
	if d > 0.5 {
		return out
	}
	cover := clamp(0.5-d, 0.0, 1.0) // anti-aliased edge

	// Along the edge, for grain and stitches: x on the top and bottom, y
	// on the sides.
	along := pos.x
	if abs(n.x) > abs(n.y) {
		along = pos.y
	}
	inCorner := abs(p.x) > half.x-r && abs(p.y) > half.y-r

	col := Body
	g := fbm(pos/(22.0*K)) - 0.5
	grain := hash(floor(pos)) - 0.5
	depth := -d // how far in from the edge

	if Style < 0.5 || (Style > 0.5 && Style < 1.5) || (Style > 5.5) {
		// Hide: mottled, a fine grain, lighter at the top, darker in.
		mott := g*0.32 + grain*0.07 + (vnoise(pos/(5.0*K))-0.5)*0.08
		col.rgb *= 1.0 + mott
		col.rgb *= 1.06 - 0.12*(pos.y/Size.y)
		col.rgb *= 1.0 - 0.28*(1.0-smoothstep(RimW, RimW+10.0*K, depth))
		if depth < RimW {
			// The rim: leather (hide) or wood (frame), lit from the top left.
			rimc := Rim
			if Style > 0.5 && Style < 1.5 {
				streak := vnoise(vec2(along/(26.0*K), depth/(1.3*K))+vec2(Seed, 0.0))
				rimc.rgb *= 0.78 + 0.42*streak + grain*0.08
				rimc.rgb *= 0.86 + 0.34*clamp(light, -1.0, 1.0)*(1.0-depth/RimW)
				// A bevel: lit outer lip, dark inner groove.
				rimc.rgb *= 1.0 + 0.22*(1.0-smoothstep(0.0, 1.2*K, depth))*light
				rimc.rgb *= 1.0 - 0.45*smoothstep(RimW-1.4*K, RimW, depth)
			} else {
				rimc.rgb *= 0.85 + 0.25*clamp(light, -1.0, 1.0) + g*0.2 + grain*0.06
				rimc.rgb *= 1.0 - 0.35*smoothstep(RimW-1.0*K, RimW, depth)
			}
			col = rimc
		}
		// A dark line at the very edge.
		col.rgb *= 1.0 - 0.55*(1.0-smoothstep(0.0, 1.0*max(1.0, K*0.6), depth))
		// Stitches just inside the rim, not round the corners.
		if Stitch.a > 0.0 && !inCorner {
			sd := abs(depth - (RimW + 2.4*K))
			dash := fract(along/(6.0*K) + 0.25)
			end := 1.0 - smoothstep(0.32, 0.46, abs(dash-0.23)*2.0) // soft ends
			if dash < 0.46 && sd < 0.6*K {
				a := Stitch.a * end * (1.0 - smoothstep(0.25*K, 0.6*K, sd))
				col.rgb = mix(col.rgb, Stitch.rgb, a)
			} else if sd < 1.1*K && depth > RimW+2.4*K && dash < 0.5 {
				col.rgb *= 1.0 - 0.1*Stitch.a // the thread's shadow
			}
		}
		// Bone studs at the corners of a frame.
		if Style > 0.5 && Style < 1.5 {
			c := half - vec2(RimW*0.5+0.6*K)
			q := abs(p) - c
			sr := 2.5 * K
			if length(q) < sr {
				z := sqrt(max(0.0, 1.0-dot(q, q)/(sr*sr)))
				nn := vec3(q/sr*sign(p), z)
				l := clamp(dot(nn, normalize(vec3(-0.55, -1.0, 1.1))), 0.0, 1.0)
				st := Accent.rgb * (0.45 + 0.7*l)
				st *= 1.0 - 0.5*smoothstep(sr-0.9*K, sr, length(q))
				col = vec4(st, 1.0)
			}
		}
	} else if Style < 2.5 {
		// Parchment: fibres, blotches, edges aged dark and uneven.
		fib := vnoise(vec2(pos.x/(2.0*K), pos.y/(9.0*K))) - 0.5
		blot := fbm(pos/(30.0*K)+vec2(3.0, 7.0)) - 0.5
		col.rgb *= 1.0 + fib*0.06 + blot*0.16 + grain*0.04
		ragged := depth + (vnoise(vec2(along/(7.0*K), Seed))-0.5)*3.0*K
		col.rgb = mix(col.rgb, Rim.rgb, 0.55*(1.0-smoothstep(0.0, 7.0*K, ragged)))
		col.rgb *= 1.0 - 0.5*(1.0-smoothstep(0.0, 1.0*K, depth))
	} else if Style < 3.5 {
		// Stone: flat faces, chisel marks, a bevel lit from the top left.
		face := vnoise(pos/(9.0*K)) - 0.5
		chip := step(0.82, vnoise(pos/(2.2*K)+vec2(9.0, 4.0)))
		col.rgb *= 1.0 + face*0.22 + grain*0.07 - chip*0.12
		if depth < RimW {
			t := 1.0 - depth/RimW
			col.rgb *= 1.0 + 0.45*light*t
		}
		col.rgb *= 1.0 - 0.6*(1.0-smoothstep(0.0, 1.0*K, depth))
	} else if Style < 4.5 {
		// A well: sunk in, shadowed at the top, a lit lip at the bottom.
		col.rgb *= 1.0 + grain*0.05 + g*0.12
		top := pos.y
		col.rgb *= 1.0 - 0.55*(1.0-smoothstep(0.0, 6.0*K, top))
		col.rgb *= 1.0 - 0.35*(1.0-smoothstep(0.0, 3.0*K, depth))
		lip := 1.0 - smoothstep(0.0, 1.2*K, depth)
		if n.y > 0.3 {
			col = mix(col, Rim, lip*n.y)
		} else {
			col.rgb *= 1.0 - 0.6*lip
		}
	} else {
		// A plate: soft, a little lighter at the top.
		col.rgb *= 1.08 - 0.16*(pos.y/Size.y)
		col.a *= smoothstep(0.0, 2.5*K, depth)*0.35 + 0.65
	}

	col.a *= cover
	return over(out, col)
}
`

var panelShader *ebiten.Shader

// panelKey is one bake: a style at a size in image pixels.
type panelKey struct {
	style     panelStyle
	w, h, r   int
	k         float32
	body, rim color.RGBA
	accent    color.RGBA
	stitch    color.RGBA
	shadow    uint8
}

type panelBake struct {
	img    *ebiten.Image
	margin float64
	used   int
}

var panelBakes = map[panelKey]*panelBake{}

// drawPanel draws a panel of style ps over the UI rectangle (x, y, w, h)
// with corner radius r, in its look's colours.
func drawPanel(d *uiDst, ps panelStyle, x, y, w, h, r float64) {
	l := panelLooks[ps]
	drawPanelLook(d, ps, l, x, y, w, h, r, 0.55)
}

// drawPanelLook is drawPanel in look l, its shadow shadow dark (0 to 1).
func drawPanelLook(d *uiDst, ps panelStyle, l panelLook, x, y, w, h, r, shadow float64) {
	if w <= 0 || h <= 0 {
		return
	}
	// Whole image pixels, so edges and stitches land crisp.
	X0, Y0 := math.Round(d.X(x)), math.Round(d.Y(y))
	X1, Y1 := math.Round(d.X(x+w)), math.Round(d.Y(y+h))
	pw, ph := int(X1-X0), int(Y1-Y0)
	if pw <= 0 || ph <= 0 {
		return
	}
	key := panelKey{ps, pw, ph, int(math.Round(r * d.k)), float32(d.k), l.body, l.rim, l.accent, l.stitch, uint8(shadow * 255)}
	b := panelBakes[key]
	if b == nil {
		b = bakePanel(key, l)
		panelBakes[key] = b
	}
	b.used = uiFrame
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(X0-b.margin, Y0-b.margin)
	d.img.DrawImage(b.img, op)
}

func bakePanel(key panelKey, l panelLook) *panelBake {
	if panelShader == nil {
		s, err := ebiten.NewShader([]byte(panelSrc))
		if err != nil {
			panic("uipanel: " + err.Error())
		}
		panelShader = s
	}
	k := float64(key.k)
	margin := math.Ceil(6 * k)
	switch {
	case key.style == psGlow:
		margin = math.Ceil(10 * k)
	case key.shadow == 0:
		margin = 2
	}
	iw, ih := key.w+2*int(margin), key.h+2*int(margin)
	img := ebiten.NewImage(iw, ih)
	vec := func(c color.RGBA) []float32 {
		if c.A == 0 {
			return []float32{0, 0, 0, 0}
		}
		// A look's colours are straight, not premultiplied: the alpha is how
		// much it shows.
		return []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255}
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Size":   []float32{float32(key.w), float32(key.h)},
		"Margin": float32(margin),
		"Radius": float32(key.r),
		"K":      float32(k),
		"RimW":   float32(l.rimW * k),
		"Style":  float32(key.style),
		"Body":   vec(key.body),
		"Rim":    vec(key.rim),
		"Accent": vec(key.accent),
		"Stitch": vec(key.stitch),
		"Seed":   float32((key.w*7+key.h*13)%97) / 9.7,
		"Shadow": float32(key.shadow) / 255,
	}
	img.DrawRectShader(iw, ih, panelShader, op)
	artBytes.Add(int64(4 * iw * ih))
	return &panelBake{img: img, margin: margin, used: uiFrame}
}

// sweepPanelBakes lets go of the panels not drawn for a few seconds.
func sweepPanelBakes() {
	if uiFrame%60 != 30 {
		return
	}
	for k, b := range panelBakes {
		if uiFrame-b.used > 240 {
			bw, bh := b.img.Bounds().Dx(), b.img.Bounds().Dy()
			b.img.Deallocate()
			artBytes.Add(-int64(4 * bw * bh))
			delete(panelBakes, k)
		}
	}
}
