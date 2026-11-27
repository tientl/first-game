package client

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/sim"
)

// The maps' look at the screen's resolution: ground not yet walked shows as
// blank old hide-paper with faint hatching (fogSrc), not as black; the
// minimap is cut round at its corners (maskSrc) in a thin bronze rim with a
// tick for north, and the player is an arrow the way they face.

// fogSrc paints the fog: image 0 is the fog, one pixel to a tile, its alpha
// how thick; it is read smoothly and paints faded paper with hatching,
// drawn in the screen's own pixels so the pattern keeps its size however
// far the map is zoomed.
const fogSrc = `//kage:unit pixels

package main

var K float
var Paper vec3
var Ink vec3

func fogAt(p vec2) float {
	o := imageSrc0Origin()
	s := imageSrc0Size()
	q := clamp(p, o+0.5, o+s-0.5)
	return imageSrc0UnsafeAt(q).a
}

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

func vnoise(p vec2) float {
	i := floor(p)
	f := fract(p)
	u := f * f * (3.0 - 2.0*f)
	return mix(mix(hash(i), hash(i+vec2(1.0, 0.0)), u.x), mix(hash(i+vec2(0.0, 1.0)), hash(i+vec2(1.0, 1.0)), u.x), u.y)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// The fog, read between its pixels.
	p := srcPos - 0.5
	f := fract(p)
	b := floor(p) + 0.5
	a := mix(mix(fogAt(b), fogAt(b+vec2(1.0, 0.0)), f.x), mix(fogAt(b+vec2(0.0, 1.0)), fogAt(b+vec2(1.0, 1.0)), f.x), f.y)
	if a <= 0.0 {
		return vec4(0.0)
	}
	a = smoothstep(0.0, 0.85, a)
	q := dstPos.xy / K
	c := Paper
	c *= 0.9 + 0.16*vnoise(q/26.0) + 0.06*vnoise(q/6.0) + 0.04*(hash(floor(dstPos.xy))-0.5)
	// Faint hatching, as on an old map where nobody has been.
	h := abs(fract((q.x+q.y)/6.0)-0.5) * 2.0
	c = mix(c, Ink, 0.3*(1.0-smoothstep(0.0, 0.12, h))*smoothstep(0.25, 0.75, vnoise(q/34.0+3.0)))
	// Darker towards where the known land begins.
	c *= 0.82 + 0.18*a
	return vec4(c*a, a)
}
`

// maskSrc draws image 0 cut to a rounded rectangle, a little shadowed in
// from its edge.
const maskSrc = `//kage:unit pixels

package main

var Radius float
var K float

func sdBox(p, b vec2, r float) float {
	q := abs(p) - b + r
	return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - r
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	s := imageSrc0Size()
	p := srcPos - imageSrc0Origin()
	d := sdBox(p-s*0.5, s*0.5, Radius)
	cover := clamp(0.5-d, 0.0, 1.0)
	c := imageSrc0UnsafeAt(srcPos)
	c.rgb *= 1.0 - 0.38*(1.0-smoothstep(0.0, 7.0*K, -d))
	return c * cover
}
`

var (
	fogShader  *ebiten.Shader
	maskShader *ebiten.Shader
)

func mapShaders() {
	if fogShader != nil {
		return
	}
	var err error
	if fogShader, err = ebiten.NewShader([]byte(fogSrc)); err != nil {
		panic("uimap: " + err.Error())
	}
	if maskShader, err = ebiten.NewShader([]byte(maskSrc)); err != nil {
		panic("uimap: " + err.Error())
	}
}

// The paper and ink of unwalked ground.
var (
	colFogPaper = color.RGBA{150, 130, 100, 255}
	colFogInk   = color.RGBA{70, 52, 34, 255}
)

// drawFog lays fog (one pixel to a tile) over the image rectangle (x, y,
// w, h) of dst, k image pixels to a UI pixel.
func drawFog(dst *ebiten.Image, fog *ebiten.Image, x, y, w, h, k float64) {
	mapShaders()
	b := fog.Bounds()
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = fog
	op.GeoM.Scale(w/float64(b.Dx()), h/float64(b.Dy()))
	op.GeoM.Translate(x, y)
	op.Uniforms = map[string]any{
		"K":     float32(k),
		"Paper": rgb3(colFogPaper),
		"Ink":   rgb3(colFogInk),
	}
	dst.DrawRectShader(b.Dx(), b.Dy(), fogShader, op)
}

func rgb3(c color.RGBA) []float32 {
	return []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255}
}

// drawMasked draws img with its top-left at image pixel (x, y) of dst, cut
// to corners of radius r image pixels.
func drawMasked(dst, img *ebiten.Image, x, y, r, k float64) {
	mapShaders()
	b := img.Bounds()
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = img
	op.GeoM.Translate(x, y)
	op.Uniforms = map[string]any{"Radius": float32(r), "K": float32(k)}
	dst.DrawRectShader(b.Dx(), b.Dy(), maskShader, op)
}

// miniBase is the minimap's ground in its own pixels, before it is drawn up.
var miniBase *ebiten.Image

// drawMinimap draws the round minimap in the top right corner of a canvas
// sw wide: the ground around the player at the screen's resolution, cut
// round, in a thin bronze rim with north marked, the player an arrow.
func (g *Game) drawMinimap(screen *uiDst, sw int) {
	m := &g.wmap
	k := screen.k
	kv := k * miniZoom // screen pixels to a minimap pixel
	cw, ch := int(math.Round(miniD*k)), int(math.Round(miniD*k))
	if m.mini == nil || m.mini.Bounds().Dx() != cw || m.mini.Bounds().Dy() != ch {
		if m.mini != nil {
			m.mini.Deallocate()
		}
		m.mini = ebiten.NewImage(cw, ch)
	}
	c := m.mini
	view := &uiDst{img: c, k: kv} // the minimap's own canvas, in its own pixels

	alpha := g.alpha()
	cx, cy := float64(g.width)/2, float64(g.height)/2
	dir := sim.DirDown
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			cx, cy, _ = g.lerpPlayer(p, alpha)
			dir = p.Dir
		}
	}
	tw, th := miniScale()
	ox, oy := miniOrigin(cx, cy)
	at := func(x, y float64) (float32, float32) { return float32(ox + x*tw), float32(oy + y*th) }

	// The ground: its pieces laid one to one on a canvas in the minimap's own
	// pixels, where they meet on whole pixels without a seam, then drawn up
	// to the screen's resolution smoothly.
	c.Fill(colWaterBG)
	if miniBase == nil {
		miniBase = ebiten.NewImage(miniW, miniH)
	}
	miniBase.Fill(colWaterBG)
	mw, mh := float64(g.width)*tw, float64(g.height)*th
	fog := m.fogImage()
	if img := m.img; img != nil && fog != nil {
		if !m.minis.draw(miniBase, ox, oy, g.frame) {
			b := img.Bounds()
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(mw/float64(b.Dx()), mh/float64(b.Dy()))
			op.GeoM.Translate(ox, oy)
			op.Filter = ebiten.FilterLinear
			miniBase.DrawImage(img, op)
			m.minis.draw(miniBase, ox, oy, g.frame)
		}
	}
	view.DrawImage(miniBase, &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear})
	if fog != nil {
		drawFog(c, fog, ox*kv, oy*kv, mw*kv, mh*kv, k)
	}
	for _, f := range g.fires {
		x, y := at(float64(f.tx)+0.5, float64(f.ty)+0.5)
		drawFireMark(view, x, y)
	}
	g.drawCaveMarks(view, at, 0, float64(g.height)) // the ways up and down: see cavemap.go
	for _, p := range g.curr.Players {
		if p.ID == g.myID || p.Dead {
			continue
		}
		wx, wy, _ := g.lerpPlayer(p, alpha)
		x, y := at(wx, wy)
		g.drawHeadMark(view, x, y, 1, playerColors[int(p.ID)%len(playerColors)], p.Look)
	}

	// On the screen, cut round, in its rim.
	pcx, pcy := miniCentre(sw)
	mx, my := pcx-miniD/2, pcy-miniD/2
	drawMasked(screen.img, c, math.Round(screen.X(mx)), math.Round(screen.Y(my)), miniD/2*k, k)
	// The rim and north: a small bone tick inside it, top middle; and the M
	// that opens the map, on the corner. Drawn once, kept.
	drawBaked(screen, "minimap rim", mx, my, miniD, miniD, 8, func(b *uiDst) {
		g.drawMiniRim(b)
	})

	// The player: an arrow the way they face, a ring pulsing out from it.
	pulse := float64(g.frame%70) / 70
	strokeCircle(screen, float32(pcx), float32(pcy), float32(5+pulse*8), float32(1.1), color.RGBA{255, 246, 224, uint8(170 * (1 - pulse))}, true)
	drawMapArrow(screen, pcx, pcy, dirAngle(dir), 5.5, playerColors[int(g.myID)%len(playerColors)])
}

// drawMiniRim draws the round minimap's rim on b, the map's top-left at
// (0, 0): a thin bronze edge, north, and the key to the full map.
func (g *Game) drawMiniRim(b *uiDst) {
	const R = miniD / 2
	c := float32(R)
	strokeCircle(b, c, c, R+0.6, 1.4, color.RGBA{150, 110, 64, 255}, true)
	strokeCircle(b, c, c, R+1.3+float32(b.hair())/2, float32(b.hair()), color.RGBA{20, 14, 10, 220}, true)
	strokeCircle(b, c, c, R-float32(b.hair())/2, float32(b.hair()), color.RGBA{255, 226, 170, 70}, true)

	// North: a small bone tick just inside the rim, top middle, and its B.
	nx, ny := c, float32(1.5)
	var tick vector.Path
	tick.MoveTo(nx, ny)
	tick.LineTo(nx+3.4, ny+5)
	tick.LineTo(nx-3.4, ny+5)
	tick.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.RGBA{20, 14, 10, 230})
	var t2 vector.Path
	t2.AddPath(&tick, &vector.AddPathOptions{GeoM: func() ebiten.GeoM {
		var gm ebiten.GeoM
		gm.Translate(-float64(nx), -float64(ny)-2.5)
		gm.Scale(1.4, 1.4)
		gm.Translate(float64(nx), float64(ny)+2.5)
		return gm
	}()})
	fillPath(b, &t2, op)
	op2 := &vector.DrawPathOptions{AntiAlias: true}
	op2.ColorScale.ScaleWithColor(colBoneText)
	fillPath(b, &tick, op2)
	uiTextW(b, "B", float64(nx), float64(ny)+5, colBoneText, text.AlignCenter, 0.58, 900, fxOutline)

	// M opens the map: a little key on the corner, off the ring.
	const kw = 13.0
	kx, ky := R+54-kw/2, R+54-kw/2
	fillRoundRect(b, float32(kx)-0.5, float32(ky)-0.5, kw+1, kw+1, 4.5, color.RGBA{0, 0, 0, 120})
	fillRoundRect(b, float32(kx), float32(ky), kw, kw, 4, color.RGBA{120, 92, 62, 255})
	hair := float32(b.hair())
	fillRoundRect(b, float32(kx)+hair, float32(ky)+hair, kw-2*hair, kw-2*hair, 4-hair, color.RGBA{30, 23, 18, 240})
	noteBox(b, kx, ky, kw, kw)
	uiTextW(b, "M", kx+kw/2, ky+kw/2-lineUI*0.7/2, colKey, text.AlignCenter, 0.7, 900, fxPlain)
}

// drawMiniLevel says, underground, which level of the caves the minimap
// shows: a small plate centred on cx, its top at y.
func (g *Game) drawMiniLevel(screen *uiDst, cx, y float64) {
	if k, ok := g.caveLevel(); ok {
		const s = 0.76
		t := fmt.Sprintf("Hang · tầng %d", k+1)
		w := textWidthW(t, s, wHeavy) + 14
		x := math.Round(cx - w/2)
		plateA(screen, x, y, w, caveLineH, caveLineH/2, 0.9)
		uiTextW(screen, t, x+7, y+(caveLineH-lineUI*s)/2, colBoneText, text.AlignStart, s, wHeavy, fxShadow)
	}
}

// dirAngle is the screen angle a facing points at.
func dirAngle(d sim.Dir) float64 {
	switch d {
	case sim.DirUp:
		return -math.Pi / 2
	case sim.DirLeft:
		return math.Pi
	case sim.DirRight:
		return 0
	}
	return math.Pi / 2
}

// drawMapArrow draws a player's arrow on a map at (x, y), pointing at
// angle a, r long, in their colour with a dark rim and a light edge.
func drawMapArrow(d *uiDst, x, y, a, r float64, c color.RGBA) {
	pt := func(f, s float64) (float32, float32) {
		return float32(x + math.Cos(a)*f - math.Sin(a)*s), float32(y + math.Sin(a)*f + math.Cos(a)*s)
	}
	shape := func(grow float64) *vector.Path {
		var p vector.Path
		p.MoveTo(pt(r+grow*1.3, 0))
		p.LineTo(pt(-r*0.75-grow, r*0.8+grow))
		p.LineTo(pt(-r*0.35, 0))
		p.LineTo(pt(-r*0.75-grow, -r*0.8-grow))
		p.Close()
		return &p
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.RGBA{16, 12, 8, 235})
	fillPath(d, shape(1.6), op)
	op = &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.RGBA{255, 248, 230, 255})
	fillPath(d, shape(0.8), op)
	op = &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	fillPath(d, shape(0), op)
}
