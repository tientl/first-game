package client

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The surface of the open sea, moving. The ground bake leaves the sea
// see-through, the more the deeper it is (seaStops in terrainsea.go), over
// a flat floor of deep blue (groundCache.drawWater); over each chunk with
// sea in it a shader then draws, at the full resolution of the screen, what
// the wind does to the water: long soft swells rolling in, small wave
// crests that rise, catch the light and sink again, the sun glinting off
// them, and in a gale whitecaps and streaks of foam blown down the wind. It
// reads how deep the water is off the chunk's own alpha, so all of it fades
// out over the shallows, where the light rippling on the sand takes over
// (terrainanim.go). Nothing is baked again for it: the cost is one pass of
// a shader over the sea in view.

// seaSurfaceSrc is the shader. Its source image is the chunk, premultiplied,
// as baked: one pixel to an image pixel of the world (groundDens to an art
// pixel). Origin is the chunk's corner in image pixels of the world.
const seaSurfaceSrc = `//kage:unit pixels

package main

var Time float
var Origin vec2
var Rough float  // 0 calm to 1 a storm
var Sun float    // how brightly the sun (or the moon) glints: 0 to 1
var Legacy float // 1 in worlds without the sea's depths: the bake tints the sea otherwise
var Wind vec2    // the way the wind blows, a unit vector

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453)
}

func noise(p vec2) float {
	i := floor(p)
	f := fract(p)
	u := f * f * (3.0 - 2.0*f)
	a := hash(i)
	b := hash(i + vec2(1, 0))
	c := hash(i + vec2(0, 1))
	d := hash(i + vec2(1, 1))
	return mix(mix(a, b, u.x), mix(c, d, u.x), u.y)
}

// swell is the height of the long swells at p, -1 to 1: bands a few tiles
// apart rolling in from the north, a second train across them, gently bent.
func swell(p vec2, t float) float {
	w1 := noise(p*0.0016+vec2(3.0, 7.0)) * 1.6
	w2 := noise(p*0.0021+vec2(11.0, 2.0)) * 1.3
	a1 := dot(p, vec2(0.0042, 0.0215)) - t*0.5 + w1
	a2 := dot(p, vec2(-0.0098, 0.0152)) - t*0.37 + w2
	return (sin(a1) + 0.5*sin(a2)) / 1.5
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0At(srcPos)
	if c.a > 0.985 {
		return vec4(0) // land
	}
	// How deep it is: the bake leaves the deep sea the most see-through.
	k := clamp((0.885-c.a)/0.19, 0.0, 1.0)
	if Legacy > 0.5 {
		k = 1.0 - clamp((c.a-0.55)/0.3, 0.0, 1.0)
	}
	if k <= 0.0 {
		return vec4(0)
	}
	k = k * k * (3.0 - 2.0*k)
	p := Origin + (srcPos - imageSrc0Origin())
	t := Time
	r := Rough

	dark := 0.0  // how much darker
	light := 0.0 // how much light on top
	foam := 0.0  // how much white foam on top of that

	// The long swells: soft bands, lighter along their tops, darker in the
	// troughs between.
	sw := swell(p, t)
	amp := 0.6 + 0.8*r
	light += amp * (0.05*smoothstep(0.15, 1.0, sw) + 0.03*smoothstep(0.8, 0.97, sw))
	dark += amp * 0.06 * smoothstep(0.1, 1.0, -sw)

	// The wave crests: short curved lines that rise, catch the light and
	// sink again, each in a cell of its own, most along the swells' tops and
	// in a wind. The back of each, under it, is in shade.
	cs := vec2(46.0, 26.0)
	base := floor(p / cs)
	wind := Wind
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			cell := base + vec2(float(i), float(j))
			h := hash(cell + vec2(5.3, 1.7))
			sh := swell((cell+0.5)*cs, t)
			dens := 0.14 + 0.3*r + (0.45+0.2*r)*smoothstep(-0.2, 0.85, sh)
			if h < dens {
				per := 2.6 + 2.2*hash(cell+vec2(2.1, 8.4)) - 0.8*r
				ph := fract(t/per + hash(cell+vec2(9.7, 3.3)))
				life := smoothstep(0.0, 0.35, ph) * (1.0 - smoothstep(0.55, 1.0, ph))
				life *= 0.5 + 0.5*hash(cell+vec2(3.9, 7.1))
				ctr := (cell + vec2(0.1, 0.15) + vec2(hash(cell+vec2(1.1, 0.0)), hash(cell+vec2(0.0, 1.3)))*vec2(0.8, 0.7)) * cs
				ctr += wind * (ph * (5.0 + 6.0*r))
				hw := 6.0 + 10.0*hash(cell+vec2(4.4, 4.4)) + 6.0*r // half its length
				tilt := (hash(cell+vec2(6.6, 2.2)) - 0.5) * 0.5
				bow := 1.0 + 2.0*hash(cell+vec2(8.1, 5.5))
				d := p - ctr
				x := (d.x + tilt*d.y) / hw
				if abs(x) < 1.0 {
					y := d.y - tilt*d.x + bow*x*x // bowed: its ends trail back up
					e := 1.0 - x*x
					th := 0.6 + 1.0*e + 0.8*r
					if y > -th && y < th*0.45 {
						light = max(light, (0.32+0.25*r)*life*sqrt(e))
						if r > 0.55 && e > 0.4 && hash(floor(p*0.5)+cell) < (r-0.45)*1.4*life {
							foam = max(foam, 0.75*life) // a whitecap
						}
					} else if y >= th*0.45 && y < th*0.45+2.2 {
						dark = max(dark, 0.14*life*e)
					}
				}
			}
		}
	}

	// Glints: the sun off the water, a point here and there that flashes
	// and is gone, more on the swells' tops.
	gs := 22.0
	gc := floor(p / gs)
	gh := hash(gc + vec2(7.7, 3.1))
	if gh > 0.84-0.14*smoothstep(0.0, 1.0, sw) {
		gp := (gc + 0.2 + 0.6*vec2(hash(gc+vec2(0.3, 5.1)), hash(gc+vec2(2.9, 0.7)))) * gs
		tw := fract(t*(0.9+0.8*gh) + gh*17.0)
		if tw < 0.12 {
			q := abs(p - gp)
			f := sin(tw / 0.12 * 3.14159)
			if q.x < 0.6 && q.y < 0.6 {
				light = max(light, 0.95*f*Sun)
			} else if (q.x < 1.6 && q.y < 0.6) || (q.x < 0.6 && q.y < 1.6) {
				light = max(light, 0.45*f*Sun)
			}
		}
	}

	// In a gale: streaks of foam blown down the wind, broken and drifting.
	if r > 0.42 {
		a := p - wind*t*14.0
		along := dot(a, wind) / length(wind)
		across := dot(a, vec2(-wind.y, wind.x)) / length(wind)
		s := noise(vec2(along*0.006, across*0.11))
		s2 := noise(vec2(along*0.03, across*0.3) + 5.0)
		if s > 0.74 && s2 > 0.42 {
			foam = max(foam, (r-0.42)*1.1*min(1.0, (s-0.74)/0.08)*(0.6+0.4*s2))
		}
		dark += 0.14 * (r - 0.42) // a grey, heavy sea
	}

	dark = clamp(dark*k, 0.0, 0.6)
	light = clamp(light*k, 0.0, 1.0)
	foam = clamp(foam*k, 0.0, 1.0)
	// Light on the water is the sky's: a pale blue-white; foam is white.
	lc := vec3(0.84, 0.94, 1.0)
	al := max(light, foam)
	col := mix(lc, vec3(0.96, 0.98, 1.0), clamp(foam/max(al, 0.001), 0.0, 1.0))
	if al <= 0.002 && dark <= 0.002 {
		return vec4(0)
	}
	// Darken first, then the light over it: one source-over draw.
	return vec4(col*al, 1.0-(1.0-dark)*(1.0-al))
}
`

var seaSurfaceShader *ebiten.Shader

func getSeaSurfaceShader() *ebiten.Shader {
	if seaSurfaceShader == nil {
		s, err := ebiten.NewShader([]byte(seaSurfaceSrc))
		if err != nil {
			log.Printf("sea surface shader: %v", err)
			return nil
		}
		seaSurfaceShader = s
	}
	return seaSurfaceShader
}

// seaLook is how the sea looks now: how rough (0 to 1) and how brightly
// it glints (0 to 1).
type seaLook struct{ rough, sun, windX, windY float64 }

// seaLookOf is the sea's look in weather w with the scene darkness dark
// (see darknessNow) and the wind blowing towards windDir (radians, 0 east)
// with strength wind (0 to 1).
func seaLookOf(w sim.Weather, dark, windDir, wind float64) seaLook {
	rough := 0.12
	switch w {
	case sim.WeatherStorm:
		rough = 1
	case sim.WeatherTornado:
		rough = 0.85
	case sim.WeatherWind:
		rough = 0.6
	case sim.WeatherRain:
		rough = 0.35
	case sim.WeatherSnow:
		rough = 0.25
	}
	rough = max(rough, 0.1+0.4*clamp01(wind))
	sun := 1 - clamp01(dark/maxDarkness)
	return seaLook{rough: rough, sun: 0.2 + 0.8*sun*sun, windX: math.Cos(windDir), windY: math.Sin(windDir)}
}

// drawGround draws the ground, and the sea moving on it, as it looks now.
func (g *Game) drawGround(dst drawTarget, cam camera) {
	g.ground.sea = seaLookOf(g.curr.Weather, g.darknessNow(), float64(g.curr.WindDir), float64(g.curr.Wind))
	g.ground.draw(dst, cam.ox, cam.oy, g.frame)
}

// hasSea reports whether a chunk's pixels hold any sea: any not opaque.
func hasSea(px []byte) bool {
	for i := 3; i < len(px); i += 4 * 7 { // a pixel in seven is plenty: the sea comes in patches
		if px[i] < 255 {
			return true
		}
	}
	return false
}

// drawSeaSurface draws the moving surface of the sea over one chunk, whose
// top-left is at (x, y) on dst and at world pixel (wx, wy), from its baked
// image img.
func (gc *groundCache) drawSeaSurface(dst drawTarget, img *ebiten.Image, x, y, wx, wy float64, frame int) {
	sh := getSeaSurfaceShader()
	if sh == nil {
		return
	}
	wx0, wy0 := gc.sea.windX, gc.sea.windY
	if wx0 == 0 && wy0 == 0 {
		wx0 = 1
	}
	legacy := float32(0)
	if gc.terr == nil || gc.terr.depth == nil {
		legacy = 1
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Scale(1/float64(groundDens), 1/float64(groundDens))
	op.GeoM.Translate(x, y)
	op.Images[0] = img
	op.Uniforms = map[string]any{
		"Time":   float32(math.Mod(float64(frame)/60, 3600)),
		"Origin": []float32{float32(wx * groundDens), float32(wy * groundDens)},
		"Rough":  float32(gc.sea.rough),
		"Sun":    float32(gc.sea.sun),
		"Legacy": legacy,
		"Wind":   []float32{float32(wx0), float32(wy0)},
	}
	drawShaderFine(dst, chunkPW, chunkPH, sh, op)
}

// fineDot draws a w by h dot of colour c at alpha a, its corner at (x, y)
// in art pixels, at the full resolution of the canvas: the foam round a
// swimmer is drawn in dots a third of an art pixel, so it is as fine as the
// sea under it.
func fineDot(dst drawTarget, x, y, w, h float64, c color.RGBA, a float64) {
	if a <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w*HDDensity, h*HDDensity)
	// On the grid of the canvas at 1080p: thirds of an art pixel.
	op.GeoM.Translate(math.Round(x*HDDensity)/HDDensity, math.Round(y*HDDensity)/HDDensity)
	op.ColorScale.ScaleWithColor(premul(c, a))
	drawImageD(dst, whitePixel(), op, HDDensity)
}
