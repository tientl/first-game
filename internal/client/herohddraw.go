package client

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The trial hero in the game (see herohd.go). SetHeroHD turns it on before
// the game is made: players who are up are then drawn with the big figure,
// standing, running or swinging a sword, lit by the fires and torches round
// them. With it off nothing here runs.

var heroHD bool

// SetHeroHD makes games made from now on draw players as the trial hero.
func SetHeroHD(on bool) { heroHD = on }

// hdSprite is a frame as two images of the same size: colours and normals.
type hdSprite struct{ col, nrm *ebiten.Image }

// hdSet is every frame for one tunic colour, with or without a sword in
// hand: [animation][down, up, right][frame].
type hdSet [numHDAnims][3][]hdSprite

type hdKey struct {
	tint  color.RGBA
	armed bool
}

type heroHDState struct {
	sets   map[hdKey]*hdSet
	shader *ebiten.Shader
	failed bool
	lights []hdLightSrc // this frame's fires and torches, on screen
	dark   float64
}

// hdLightSrc is a point light: where it stands on screen, in art pixels,
// how far it reaches and how strong it is.
type hdLightSrc struct {
	sx, sy, reach, k float64
}

const hdMaxLights = 4

// hdLitSrc lights a frame from up to four point lights. Each pixel has a
// place in the world: across the sprite, up from the feet, and no depth; a
// light is at LightPos from the feet (right, up, towards the viewer). The
// side of the figure turned to a light warms in bands, the pixels on its
// outline towards the light get a bright rim, and at night the side turned
// away sinks into a cool shade.
const hdLitSrc = `//kage:unit pixels

package main

var LightPos [4]vec3
var LightCol [4]vec3
var LightRad [4]float
var Feet vec2
var Dark float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(srcPos)
	if c.a == 0 {
		return vec4(0)
	}
	ne := imageSrc1UnsafeAt(srcPos).rgb
	if ne.b < 0.01 {
		return c // the sword's arc gives its own light
	}
	n := normalize(ne*2 - 1)
	p := srcPos - imageSrc0Origin()
	pos := vec3(p.x-Feet.x, Feet.y-p.y, 0)
	rgb := c.rgb / c.a
	warm := vec3(0)
	rim := vec3(0)
	near := 0.0
	lit := 0.0
	for i := 0; i < 4; i++ {
		d := LightPos[i] - pos
		dist := length(d)
		att := clamp(1-dist/LightRad[i], 0, 1)
		att = att * (2 - att)
		l := d / max(dist, 0.001)
		ndl := max(dot(n, l), 0)
		ndl = floor(ndl*3+0.35) / 3
		warm += LightCol[i] * att * ndl
		lit += att * ndl
		near = max(near, att)
		// the outline towards the light
		s := vec2(0)
		if abs(l.x) > 0.3 {
			s.x = sign(l.x)
		}
		if abs(l.y) > 0.45 {
			s.y = -sign(l.y)
		}
		e := 0.0
		if s.x != 0 || s.y != 0 {
			e = 1 - imageSrc0At(srcPos+s).a
			if s.x != 0 {
				e = max(e, 1-imageSrc0At(srcPos+vec2(s.x, 0)).a)
			}
		}
		rim += LightCol[i] * att * e
	}
	shade := mix(vec3(1), vec3(0.42, 0.41, 0.7), Dark*near*(1-clamp(lit, 0, 1)))
	out := rgb * shade
	out = out + (rgb*0.42+vec3(0.04, 0.02, 0))*warm
	out = mix(out, vec3(1, 0.88, 0.6), clamp(rim*0.42, vec3(0), vec3(0.7)))
	return vec4(clamp(out, vec3(0), vec3(1))*c.a, c.a)
}
`

func (g *Game) heroHD() *heroHDState {
	if g.hd == nil {
		g.hd = &heroHDState{sets: make(map[hdKey]*hdSet)}
		sh, err := ebiten.NewShader([]byte(hdLitSrc))
		if err != nil {
			log.Printf("hero shader: %v", err)
			g.hd.failed = true
		}
		g.hd.shader = sh
	}
	return g.hd
}

// hdDirs are the facings drawn; left is right mirrored.
var hdDirs = [3]sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight}

func (h *heroHDState) set(tint color.RGBA, armed bool) *hdSet {
	k := hdKey{tint, armed}
	if s, ok := h.sets[k]; ok {
		return s
	}
	s := &hdSet{}
	for an := range numHDAnims {
		for di, d := range hdDirs {
			for i := range hdFrames[an] {
				fr := genHeroHD(tint, an, d, i, armed)
				s[an][di] = append(s[an][di], hdSprite{fr.col.image(), fr.nrm.image()})
			}
		}
	}
	h.sets[k] = s
	return s
}

// noteHeroLights gathers the fires and torches in view for this frame's
// heroes.
func (g *Game) noteHeroLights(cam camera) {
	h := g.heroHD()
	h.lights = h.lights[:0]
	h.dark = (1 - sim.Ambient(g.curr.Tick)) * maxDarkness
	for _, e := range g.curr.Entities {
		r, ok := sim.FireReach(e.Kind, int(e.Fuel))
		if !ok || e.Kind == sim.EntityCaveExit {
			continue
		}
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		h.lights = append(h.lights, hdLightSrc{sx, sy, r * TileW * 1.15, g.flicker(uint32(e.ID))})
	}
	for _, t := range g.torches(cam) {
		h.lights = append(h.lights, hdLightSrc{t.sx, t.sy + 12, sim.TorchRadius * TileW * 1.15, g.flicker(t.id)})
	}
}

// heroLights picks the lights that reach a hero standing at (sx, sy), the
// strongest first, as the shader's uniforms; flip mirrors them for a figure
// drawn mirrored. It also says which way the strongest pushes a shadow.
func (g *Game) heroLights(sx, sy float64, flip bool) (pos, col, rad []float32, shx, shy float64) {
	h := g.hd
	pos, col, rad = make([]float32, 3*hdMaxLights), make([]float32, 3*hdMaxLights), make([]float32, hdMaxLights)
	for i := range rad {
		rad[i] = 1
	}
	type cand struct {
		l hdLightSrc
		w float64
	}
	var best [hdMaxLights]cand
	for _, l := range h.lights {
		dx, dz := l.sx-sx, (l.sy-sy)*TileW/TileH
		w := 1 - math.Hypot(dx, dz)/l.reach
		if w <= 0 {
			continue
		}
		c := cand{l, w}
		for i := range best {
			if c.w > best[i].w {
				c, best[i] = best[i], c
			}
		}
	}
	strength := 0.22 + h.dark
	for i, c := range best {
		if c.w <= 0 {
			break
		}
		dx, dz := c.l.sx-sx, (c.l.sy-sy)*TileW/TileH
		if i == 0 {
			n := math.Hypot(dx, dz) + 1e-6
			shx, shy = -dx/n*c.w*strength, -dz/n*c.w*strength
		}
		if flip {
			dx = -dx
		}
		k := float32(strength * c.l.k)
		pos[3*i], pos[3*i+1], pos[3*i+2] = float32(dx), 10, float32(dz)
		col[3*i], col[3*i+1], col[3*i+2] = 1.1*k, 0.7*k, 0.34*k
		rad[i] = float32(c.l.reach)
	}
	return pos, col, rad, shx, shy
}

// heroStands reports whether p is drawn as the trial hero: up, awake and
// on land.
func (g *Game) heroStands(p protocol.PlayerView) bool {
	return !p.Dead && !p.Asleep && !p.Swimming
}

// drawHeroShadow is the hero's contact shadow: a soft dark ellipse under
// the feet with a darker heart, pushed away from the nearest fire.
func (g *Game) drawHeroShadow(dst *ebiten.Image, sx, sy float64) {
	_, _, _, shx, shy := g.heroLights(sx, sy, false)
	for _, s := range [2]struct{ w, h, a, push float64 }{{27, 10, 0.3, 3.5}, {17, 6, 0.42, 1.5}} {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-shadowSize/2, -shadowSize/2)
		op.GeoM.Scale((s.w+math.Abs(shx)*4)/shadowSize, (s.h+math.Abs(shy)*1.5)/shadowSize)
		op.GeoM.Translate(sx+shx*s.push, sy-0.5+shy*s.push*0.5)
		op.ColorScale.ScaleAlpha(float32(s.a))
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(g.art.shadow, op)
	}
}

// drawHeroHD draws p, who is up and on land, as the trial hero with the
// feet at d's point.
func (g *Game) drawHeroHD(dst *ebiten.Image, p protocol.PlayerView, d drawable) {
	h := g.heroHD()
	tint := playerColors[int(p.ID)%len(playerColors)]
	armed := isSword(p.Held)
	set := h.set(tint, armed)
	an, fr := hdIdle, (g.frame/16)%4
	if fr == 0 && (g.frame/64)%3 == 2 && g.frame%64 < 8 {
		fr = 4 // a blink
	}
	blow := -1
	if armed && p.Acting {
		// the blow, then the guard held until the next one
		age := g.since(g.swingAt[p.ID]) % 34
		for i, end := range [5]int{5, 10, 13, 17, 24} {
			if age < end {
				blow = i
				break
			}
		}
		if blow < 0 {
			blow = 4
		}
	}
	switch {
	case blow >= 0:
		an, fr = hdAttack, blow
	case d.moving:
		an, fr = hdRun, (g.frame/5)%8
		if g.backpedalling(p) {
			fr = (8 - fr) % 8
		}
	}
	di, flip := 0, false
	switch p.Dir & 3 {
	case sim.DirUp:
		di = 1
	case sim.DirRight:
		di = 2
	case sim.DirLeft:
		di, flip = 2, true
	}
	sp := set[an][di][fr]
	geo := ebiten.GeoM{}
	geo.Translate(-hdOX, -hdOY)
	if flip {
		geo.Scale(-1, 1)
	}
	geo.Translate(math.Round(d.sx), math.Round(d.sy))
	if h.failed {
		dst.DrawImage(sp.col, &ebiten.DrawImageOptions{GeoM: geo})
		return
	}
	pos, col, rad, _, _ := g.heroLights(d.sx, d.sy, flip)
	op := &ebiten.DrawRectShaderOptions{GeoM: geo}
	op.Images[0], op.Images[1] = sp.col, sp.nrm
	op.Uniforms = map[string]any{
		"LightPos": pos, "LightCol": col, "LightRad": rad,
		"Feet": []float32{hdOX, hdOY}, "Dark": float32(h.dark),
	}
	dst.DrawRectShader(hdW, hdH, h.shader, op)
}
