package client

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Waves rolling in to the shore. Every ground chunk with a coastline also
// bakes a "shore" image: for each pixel near the water's edge, its signed
// distance to the coast (red) and a mask (alpha). A shader turns that into
// lines of foam that travel in, break on the beach and wash up the sand,
// redrawn every frame on top of the baked ground.

const (
	surfSea  = 1.4  // tiles out to sea where waves are drawn
	surfLand = 0.45 // tiles up the beach the wash can reach
)

// surfShaderSrc is the wave shader. Red is (s + surfSea) / (surfSea +
// surfLand) for the signed coast distance s (negative at sea); alpha marks
// pixels where surf may show. It runs on the full-resolution canvas, on the
// grid of the baked ground (a third of an art pixel), the coast distance
// eased between the texels of the shore image, so the lines of foam are as
// smooth as the coast under them.
const surfShaderSrc = `//kage:unit pixels

package main

var Time float
var Origin vec2

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453)
}

func texel(p vec2) vec4 {
	q := clamp(p, vec2(0.5), imageSrc0Size()-vec2(0.5))
	return imageSrc0At(imageSrc0Origin() + q)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	lp := (floor((srcPos-imageSrc0Origin())*3.0) + 0.5) / 3.0
	d := texel(floor(lp) + 0.5)
	if d.a < 0.5 {
		return vec4(0)
	}
	s := d.r
	b := lp - 0.5
	i := floor(b)
	f := b - i
	t00 := texel(i + vec2(0.5, 0.5))
	t10 := texel(i + vec2(1.5, 0.5))
	t01 := texel(i + vec2(0.5, 1.5))
	t11 := texel(i + vec2(1.5, 1.5))
	if t00.a+t10.a+t01.a+t11.a > 3.99 {
		s = mix(mix(t00.r, t10.r, f.x), mix(t01.r, t11.r, f.x), f.y)
	}
	s = s*(1.4+0.45) - 1.4
	// lp is from the shore image's own corner, the same wherever the atlas
	// put it.
	w := Origin + lp
	// Each stretch of coast has its own rhythm, so the waves do not march
	// in lockstep along the whole beach.
	j := sin(w.x*0.021+w.y*0.013)*0.5 + sin(w.x*0.0071-w.y*0.017)*0.5
	t := Time*0.09 + j*0.35

	a := 0.0
	if s < 0 {
		// Crests travelling in: lines of foam, thicker close to the shore.
		ph := fract(t - s*0.62)
		near := 1.0 - (-s)/1.4
		band := 0.06 + 0.08*near
		if ph < band {
			// Faint where it rises out at sea, whiter as it comes in.
			a = (0.45 + 0.45*near) * smoothstep(0.0, 0.45, near)
		}
		// A trail of broken foam behind each crest.
		if ph >= band && ph < band+0.08*near && hash(floor(w*1.5)) < 0.55 {
			a = 0.3 * near
		}
	} else {
		// The wash: after each wave the water runs up the sand and back.
		ph := fract(t)
		run := 0.45 * max(0.0, sin(ph*6.2832)) // how far up the beach it is now
		if s < run {
			a = 0.22
			if s > run-0.06 {
				a = 0.75 // the foamy front of the wash
			}
		}
	}
	if a <= 0 {
		return vec4(0)
	}
	// Crisp pixel steps instead of smooth alpha.
	a = floor(a*4.0+0.5) / 4.0
	return vec4(0.93, 0.97, 0.98, 1) * a
}
`

var surfShader *ebiten.Shader

func getSurfShader() *ebiten.Shader {
	if surfShader == nil {
		s, err := ebiten.NewShader([]byte(surfShaderSrc))
		if err != nil {
			log.Printf("surf shader: %v", err)
			return nil
		}
		surfShader = s
	}
	return surfShader
}

// shoreByte encodes a coast distance for the shore image, or ok=false for
// pixels too far from the edge for any surf.
func shoreByte(s float64) (byte, bool) {
	if s < -surfSea || s > surfLand {
		return 0, false
	}
	return byte(math.Round((s + surfSea) / (surfSea + surfLand) * 255)), true
}

// drawSurf draws the waves over one chunk whose top-left is at (x, y) on
// dst and at world pixel (wx, wy).
func drawSurf(dst drawTarget, shore *ebiten.Image, x, y, wx, wy float64, frame int) {
	sh := getSurfShader()
	if sh == nil || shore == nil {
		return
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(x, y)
	op.Images[0] = shore
	op.Uniforms = map[string]any{
		"Time":   float32(float64(frame) / 60),
		"Origin": []float32{float32(wx), float32(wy)},
	}
	drawShaderFine(dst, chunkW, chunkH, sh, op)
}
