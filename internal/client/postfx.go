package client

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Post-processing: the world canvas goes through shaders before it goes
// on the screen. Bright things (fire, torches, glowing plants,
// lava, the sun on sand and foam) bleed a soft glow around them (bloom),
// and the colours are graded like a film: a little more contrast and
// colour, shadows leaning cool, highlights warm, and the whole picture
// tinted by the hour, golden at dawn and dusk, blue by night.
//
// It is off unless the settings (Settings.PostFX) or SetPostFX turn it on.

var postFX bool

// SetPostFX turns the shaders on or off for this session, without saving it.
func SetPostFX(on bool) { OverrideSettings(func(s *Settings) { s.PostFX = on }) }

// brightBlurSrc keeps what is bright and blurs it along Dir.
const brightBlurSrc = `//kage:unit pixels

package main

var Dir vec2
var Threshold float

func bright(p vec2) vec3 {
	c := imageSrc0At(p).rgb
	l := dot(c, vec3(0.2126, 0.7152, 0.0722))
	k := clamp((l-Threshold)/(1.0-Threshold), 0.0, 1.0)
	return c * k * k
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	w0 := 0.227
	w1 := 0.194
	w2 := 0.122
	w3 := 0.054
	w4 := 0.016
	s := bright(srcPos) * w0
	s += (bright(srcPos+Dir*1.5) + bright(srcPos-Dir*1.5)) * w1
	s += (bright(srcPos+Dir*3.0) + bright(srcPos-Dir*3.0)) * w2
	s += (bright(srcPos+Dir*4.5) + bright(srcPos-Dir*4.5)) * w3
	s += (bright(srcPos+Dir*6.0) + bright(srcPos-Dir*6.0)) * w4
	return vec4(s, 1)
}
`

// blurSrc blurs image 0 along Dir.
const blurSrc = `//kage:unit pixels

package main

var Dir vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	s := imageSrc0At(srcPos).rgb * 0.227
	s += (imageSrc0At(srcPos+Dir*1.5).rgb + imageSrc0At(srcPos-Dir*1.5).rgb) * 0.194
	s += (imageSrc0At(srcPos+Dir*3.0).rgb + imageSrc0At(srcPos-Dir*3.0).rgb) * 0.122
	s += (imageSrc0At(srcPos+Dir*4.5).rgb + imageSrc0At(srcPos-Dir*4.5).rgb) * 0.054
	s += (imageSrc0At(srcPos+Dir*6.0).rgb + imageSrc0At(srcPos-Dir*6.0).rgb) * 0.016
	return vec4(s, 1)
}
`

// gradeSrc grades image 0 and adds the glow in image 1.
const gradeSrc = `//kage:unit pixels

package main

var Tint vec3    // the hour's colour, multiplied in
var Bloom float  // how strong the glow is
var Size vec2    // the canvas, for the vignette

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0At(srcPos).rgb
	glow := imageSrc1At(srcPos).rgb
	// Contrast and colour.
	c = (c-0.5)*1.1 + 0.5
	l := dot(c, vec3(0.2126, 0.7152, 0.0722))
	c = mix(vec3(l), c, 1.18)
	// Split toning: cool shadows, warm highlights.
	c += (1.0-l) * vec3(-0.015, 0.005, 0.03)
	c += l * vec3(0.035, 0.015, -0.025)
	c *= Tint
	// The glow, added on top, a little warmer than what made it.
	c += glow * Bloom * vec3(1.05, 0.95, 0.85)
	// A soft darkening towards the corners.
	uv := srcPos/Size - 0.5
	c *= 1.0 - dot(uv, uv)*0.45
	return vec4(clamp(c, 0.0, 1.0), 1)
}
`

type postShaders struct {
	bright, blur, grade *ebiten.Shader
	small, a, b, c      *ebiten.Image // art-pixel size, for the glow
	glow, out           *ebiten.Image // canvas size
}

var post *postShaders

func getPost() *postShaders {
	if post != nil {
		return post
	}
	p := &postShaders{}
	for _, s := range []struct {
		dst **ebiten.Shader
		src string
	}{{&p.bright, brightBlurSrc}, {&p.blur, blurSrc}, {&p.grade, gradeSrc}} {
		sh, err := ebiten.NewShader([]byte(s.src))
		if err != nil {
			log.Printf("post shader: %v", err)
			postFX = false
			return nil
		}
		*s.dst = sh
	}
	post = p
	return p
}

// hourTint is the colour the light has at tick: golden near sunrise and
// sunset, a little blue by night, white at noon.
func hourTint(tick uint64) [3]float32 {
	a := sim.Ambient(tick) // 0 night .. 1 full day
	gold := 1 - min(1, abs64(a-0.55)/0.35)
	r := 1 + 0.08*gold - 0.06*(1-a)
	g := 1 + 0.01*gold - 0.03*(1-a)
	b := 1 - 0.1*gold + 0.05*(1-a)
	return [3]float32{float32(r), float32(g), float32(b)}
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// applyPost runs the full-resolution world canvas through the shaders and
// returns what to draw on the screen. The glow is worked out at art-pixel
// size (the canvas shrunk scale times), as it was when the world was drawn
// small, then spread back over the canvas for the grade.
func (g *Game) applyPost(canvas *ebiten.Image, scale int) *ebiten.Image {
	p := getPost()
	if p == nil {
		return canvas
	}
	cb := canvas.Bounds()
	w, h := (cb.Dx()+scale-1)/scale, (cb.Dy()+scale-1)/scale
	if p.out == nil || p.out.Bounds() != cb || p.a.Bounds().Dx() != w || p.a.Bounds().Dy() != h {
		for _, im := range []*ebiten.Image{p.small, p.a, p.b, p.c, p.glow, p.out} {
			if im != nil {
				im.Deallocate()
			}
		}
		p.small, p.a, p.b, p.c = ebiten.NewImage(w, h), ebiten.NewImage(w, h), ebiten.NewImage(w, h), ebiten.NewImage(w, h)
		p.glow, p.out = ebiten.NewImage(cb.Dx(), cb.Dy()), ebiten.NewImage(cb.Dx(), cb.Dy())
	}
	// The canvas at art-pixel size: each art pixel's colour, for the glow.
	sop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sop.GeoM.Scale(1/float64(scale), 1/float64(scale))
	p.small.Clear()
	p.small.DrawImage(canvas, sop)
	pass := func(dst *ebiten.Image, sh *ebiten.Shader, u map[string]any, srcs ...*ebiten.Image) {
		op := &ebiten.DrawRectShaderOptions{Uniforms: u}
		copy(op.Images[:], srcs)
		b := dst.Bounds()
		dst.DrawRectShader(b.Dx(), b.Dy(), sh, op)
	}
	night := 1 - sim.Ambient(g.curr.Tick)
	threshold := float32(0.66 - 0.04*night)
	pass(p.a, p.bright, map[string]any{"Dir": []float32{1, 0}, "Threshold": threshold}, p.small)
	pass(p.b, p.blur, map[string]any{"Dir": []float32{0, 1}}, p.a)
	pass(p.a, p.bright, map[string]any{"Dir": []float32{2, 0}, "Threshold": threshold}, p.small) // a wider halo
	pass(p.c, p.blur, map[string]any{"Dir": []float32{0, 2}}, p.a)
	p.b.DrawImage(p.c, &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter})
	gop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	gop.GeoM.Scale(float64(scale), float64(scale))
	p.glow.Clear()
	p.glow.DrawImage(p.b, gop)
	t := hourTint(g.curr.Tick)
	pass(p.out, p.grade, map[string]any{
		"Tint":  []float32{t[0], t[1], t[2]},
		"Bloom": float32(0.5 + 0.2*night),
		"Size":  []float32{float32(cb.Dx()), float32(cb.Dy())},
	}, canvas, p.glow)
	return p.out
}
