package client

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Full HD art. The world is laid out in art pixels (TileW by TileH to a
// tile) and drawn on a canvas as big as the screen, R canvas pixels to an
// art pixel (R is the integer world scale: 2 at 720p, 3 at 1080p, plus the
// zoom; above 3 kept to a multiple of HDDensity, the canvas then smaller
// than the screen and blown up to it: see crispScale and presentWorld in
// display.go). An image can hold its art at a density: how many of its
// pixels make one art pixel. Density 1 is the original pixel art;
// HDDensity is the redrawn art, three times as fine, which at 1080p lands
// one image pixel on one screen pixel. See HD_ART.md.
//
// Density-1 drawing looks exactly as it did when the world was drawn on a
// small art-pixel canvas and scaled up: it still is. Such draws go to an
// art-pixel layer (worldDst.layer) that is copied up onto the canvas, R
// times bigger with nearest filtering, whenever something has to go on top
// of it at full resolution (an HD sprite) or the layer must move to a new
// sub-pixel grid (a moving thing; see align). So shapes, rotations, linear
// shadows and shaders of the old art keep their art-pixel grain, while HD
// sprites are drawn straight onto the canvas at full resolution.

// HDDensity is the pixel density of redrawn art: image pixels per art pixel.
const HDDensity = 3

// subPixel lets the camera and moving things sit between art pixels, on
// canvas pixels, so they glide in steps of 1/R art pixel. Off, everything
// snaps to whole art pixels as it did before HD art.
var subPixel = true

// SetSubPixel turns sub-art-pixel motion on or off.
func SetSubPixel(on bool) { subPixel = on }

// drawTarget is what world drawing code draws on: the world pass (a
// *worldDst, in art pixels) or any plain image (the screen, a scratch
// image), which *ebiten.Image satisfies. Bounds is in art pixels.
//
// Vector shapes go through fillRect, strokeLine and the like below, colour
// matrices through drawColorM, and sprites through drawSprite or drawAt, all
// of which take a drawTarget.
type drawTarget interface {
	Bounds() image.Rectangle
	DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions)
	DrawRectShader(w, h int, s *ebiten.Shader, op *ebiten.DrawRectShaderOptions)
	DrawTriangles(vs []ebiten.Vertex, is []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions)
}

// density is s's pixel density: 1 for the original art.
func (s sprite) density() int {
	if s.dens <= 1 {
		return 1
	}
	return s.dens
}

// w and h are s's size in art pixels, whatever its density.
func (s sprite) w() float64 { return float64(s.img.Bounds().Dx()) / float64(s.density()) }
func (s sprite) h() float64 { return float64(s.img.Bounds().Dy()) / float64(s.density()) }

// mkHD makes an HD sprite from a canvas drawn HDDensity times finer than
// art pixels (newScaledCanvas(w, h, HDDensity)), anchored at design point
// (ax, ay), in art pixels like every coordinate the drawing code uses.
func mkHD(c *canvas, ax, ay float64) sprite {
	return spriteHD(c.image(), (ax+c.tx)*c.k()/HDDensity, (ay+c.ty)*c.k()/HDDensity)
}

// spriteHD wraps an image drawn at HDDensity (say, loaded from a PNG) as a
// sprite anchored at (ax, ay) art pixels: image pixel (3*ax, 3*ay).
func spriteHD(img *ebiten.Image, ax, ay float64) sprite {
	return sprite{img: img, ax: ax, ay: ay, dens: HDDensity}
}

// drawSprite draws s with op, whose GeoM places s in art pixels as if s were
// density-1 art of the same size (so GeoM.Translate(-s.ax, -s.ay) puts its
// anchor at the origin, whatever s's density).
func drawSprite(dst drawTarget, s sprite, op *ebiten.DrawImageOptions) {
	drawImageD(dst, s.img, op, s.density())
}

// drawImageD draws img, whose art is at the given density, with op in art
// pixels as for drawSprite.
func drawImageD(dst drawTarget, img *ebiten.Image, op *ebiten.DrawImageOptions, density int) {
	if density <= 1 {
		dst.DrawImage(img, op)
		return
	}
	o := *op
	var m ebiten.GeoM
	m.Scale(1/float64(density), 1/float64(density))
	m.Concat(op.GeoM)
	o.GeoM = m
	switch d := dst.(type) {
	case *worldDst:
		d.drawFine(img, &o)
		return
	case *fineBuf:
		d.drawFine(img, &o)
		return
	case *uiDst:
		d.drawFine(img, &o)
		return
	}
	pickFilter(&o, op.Filter)
	dst.DrawImage(img, &o)
}

// pickFilter keeps an HD image crisp when it lands on whole pixels at a
// whole scale (nearest, translation rounded) and smooth otherwise.
func pickFilter(o *ebiten.DrawImageOptions, asked ebiten.Filter) {
	a, b, c, d := o.GeoM.Element(0, 0), o.GeoM.Element(0, 1), o.GeoM.Element(1, 0), o.GeoM.Element(1, 1)
	if asked == ebiten.FilterNearest && b == 0 && c == 0 && whole(a) && whole(d) && a != 0 && d != 0 {
		o.Filter = ebiten.FilterNearest
		tx, ty := o.GeoM.Element(0, 2), o.GeoM.Element(1, 2)
		o.GeoM.Translate(math.Round(tx)-tx, math.Round(ty)-ty)
		return
	}
	o.Filter = ebiten.FilterLinear
}

func whole(v float64) bool { return math.Abs(v-math.Round(v)) < 1e-6 }

// worldDst is the world pass: a canvas of screen pixels seen as art pixels,
// R to one. See the comment at the top of this file.
type worldDst struct {
	canvas  *ebiten.Image // the full-resolution target
	r       float64       // canvas pixels per art pixel, a whole number
	w, h    int           // the view in art pixels
	layer   *ebiten.Image // density-1 drawing not yet on the canvas: (w+2) by (h+2) art pixels
	scratch *ebiten.Image // the same size, for a draw with an unusual blend

	camX, camY float64           // the camera's sub-pixel grid
	gx, gy     float64           // the grid the next density-1 draw lands on, in [0, 1) art pixels
	lx, ly     float64           // the grid of what is in the layer
	lblend     ebiten.Blend      // how the layer goes on the canvas: source-over or lighter
	dirty      []image.Rectangle // where the layer has been drawn on, in layer pixels

	flushes int // layer copies this frame, for profiling
}

// begin starts a frame: the world, w by h art pixels, drawn r times bigger
// on canvas, with the camera at cam.
func (wd *worldDst) begin(canvas *ebiten.Image, r, w, h int, cam camera) {
	wd.canvas, wd.r, wd.w, wd.h = canvas, float64(r), w, h
	if wd.layer == nil || wd.layer.Bounds().Dx() != w+2 || wd.layer.Bounds().Dy() != h+2 {
		for _, im := range []*ebiten.Image{wd.layer, wd.scratch} {
			if im != nil {
				im.Deallocate()
			}
		}
		wd.layer, wd.scratch = ebiten.NewImage(w+2, h+2), ebiten.NewImage(w+2, h+2)
	}
	wd.layer.Clear()
	wd.dirty = wd.dirty[:0]
	wd.flushes = 0
	wd.camX, wd.camY = wd.frac(-cam.ox), wd.frac(-cam.oy)
	wd.gx, wd.gy = wd.camX, wd.camY
	wd.lx, wd.ly = wd.gx, wd.gy
	wd.lblend = ebiten.BlendSourceOver
}

// end puts whatever is left in the layer on the canvas.
func (wd *worldDst) end() { wd.flush() }

// frac is v's place between art pixels, to the nearest canvas pixel.
func (wd *worldDst) frac(v float64) float64 {
	f := math.Round((v-math.Floor(v))*wd.r) / wd.r
	if f >= 1 {
		f = 0
	}
	return f
}

// Bounds is the view in art pixels.
func (wd *worldDst) Bounds() image.Rectangle { return image.Rect(0, 0, wd.w, wd.h) }

// align puts the density-1 drawing that follows on the art-pixel grid
// through (x, y), a point in art pixels: the ground point of a moving thing,
// so all of it keeps its old pixel art shape wherever it stands between
// art pixels. alignCamera goes back to the grid of the ground.
func align(dst drawTarget, x, y float64) {
	if wd, ok := dst.(*worldDst); ok {
		wd.gx, wd.gy = wd.frac(x), wd.frac(y)
	}
}

// gridOf is where dst's art-pixel grid lies now: (0, 0) for a plain image.
func gridOf(dst drawTarget) (gx, gy float64) {
	if wd, ok := dst.(*worldDst); ok {
		return wd.gx, wd.gy
	}
	return 0, 0
}

func alignCamera(dst drawTarget) {
	if wd, ok := dst.(*worldDst); ok {
		wd.gx, wd.gy = wd.camX, wd.camY
	}
}

// layerShift is how far the layer sits from art pixels for a grid at g: on the
// whole-pixel grid not at all (the very same numbers the old canvas drew
// with), else a pixel less g, so the part of the view between 0 and g
// still lands in the layer.
func layerShift(g float64) float64 {
	if g == 0 {
		return 0
	}
	return 1 - g
}

// into readies the layer for a density-1 draw with the given blend and
// returns the shift from art pixels to layer pixels, or ok=false when the
// blend cannot be gathered in a layer.
func (wd *worldDst) into(blend ebiten.Blend) (dx, dy float64, ok bool) {
	if blend == (ebiten.Blend{}) {
		blend = ebiten.BlendSourceOver
	}
	if blend != ebiten.BlendSourceOver && blend != ebiten.BlendLighter {
		return 0, 0, false
	}
	if wd.lx != wd.gx || wd.ly != wd.gy || wd.lblend != blend {
		wd.flush()
		wd.lx, wd.ly, wd.lblend = wd.gx, wd.gy, blend
	}
	return layerShift(wd.gx), layerShift(wd.gy), true
}

// mark notes that the layer has been drawn on inside r (layer pixels), and
// reports whether any of r is in the layer at all.
func (wd *worldDst) mark(r image.Rectangle) bool {
	r = r.Intersect(wd.layer.Bounds())
	if r.Empty() {
		return false
	}
	// The dirty rectangles never overlap (a pixel copied up twice would be
	// blended twice), and there are only a few: r joins any it touches, and
	// when there are too many, the one it grows least.
	for {
		j := -1
		for i, d := range wd.dirty {
			if d.Overlaps(r) {
				j = i
				break
			}
		}
		if j < 0 && len(wd.dirty) >= maxDirty {
			best := math.MaxInt
			for i, d := range wd.dirty {
				u := d.Union(r)
				if g := u.Dx()*u.Dy() - d.Dx()*d.Dy(); g < best {
					best, j = g, i
				}
			}
		}
		if j < 0 {
			wd.dirty = append(wd.dirty, r)
			return true
		}
		r = r.Union(wd.dirty[j])
		wd.dirty = append(wd.dirty[:j], wd.dirty[j+1:]...)
	}
}

// maxDirty is how many separate rectangles of the layer flush copies up.
const maxDirty = 12

// flush copies the layer up onto the canvas and clears it.
func (wd *worldDst) flush() {
	if len(wd.dirty) == 0 {
		return
	}
	wd.flushes++
	for _, d := range wd.dirty {
		wd.copyUp(wd.layer, d, wd.lx, wd.ly, wd.lblend)
	}
	for _, d := range wd.dirty {
		wd.layer.SubImage(d).(*ebiten.Image).Clear()
	}
	wd.dirty = wd.dirty[:0]
}

// copyUp draws region rect of an art-pixel buffer whose grid is at (gx,
// gy) onto the canvas, r times bigger.
func (wd *worldDst) copyUp(buf *ebiten.Image, rect image.Rectangle, gx, gy float64, blend ebiten.Blend) {
	op := &ebiten.DrawImageOptions{Blend: blend}
	op.GeoM.Scale(wd.r, wd.r)
	op.GeoM.Translate(math.Round((float64(rect.Min.X)-layerShift(gx))*wd.r), math.Round((float64(rect.Min.Y)-layerShift(gy))*wd.r))
	wd.canvas.DrawImage(buf.SubImage(rect).(*ebiten.Image), op)
}

// boxOf is the layer pixels a w by h rectangle covers under m, shifted by
// (dx, dy), with a pixel to spare for antialiasing.
func boxOf(m ebiten.GeoM, w, h, dx, dy float64) image.Rectangle {
	x0, y0 := math.Inf(1), math.Inf(1)
	x1, y1 := math.Inf(-1), math.Inf(-1)
	for _, p := range [4][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		x, y := m.Apply(p[0], p[1])
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	return image.Rect(int(math.Floor(x0+dx))-1, int(math.Floor(y0+dy))-1, int(math.Ceil(x1+dx))+1, int(math.Ceil(y1+dy))+1)
}

// snapGeoM rounds a plain shift of a density-1 image to whole layer pixels,
// as the old canvas did by sampling, so it never lands half way. Halves
// round up, as the old canvas's nearest sampling settled them (many
// anchors are at half pixels).
func snapGeoM(m *ebiten.GeoM, f ebiten.Filter) {
	if f != ebiten.FilterNearest {
		return
	}
	a, b, c, d := m.Element(0, 0), m.Element(0, 1), m.Element(1, 0), m.Element(1, 1)
	if b != 0 || c != 0 || !whole(a) || !whole(d) {
		return
	}
	tx, ty := m.Element(0, 2), m.Element(1, 2)
	m.Translate(roundHalfUp(tx)-tx, roundHalfUp(ty)-ty)
}

func roundHalfUp(v float64) float64 { return math.Floor(v + 0.5 + 1e-6) }

// DrawImage draws density-1 art, in art pixels.
func (wd *worldDst) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	var o ebiten.DrawImageOptions
	if op != nil {
		o = *op
	}
	b := img.Bounds()
	dx, dy, ok := wd.into(o.Blend)
	if !ok {
		wd.odd(o.Blend, boxOf(o.GeoM, float64(b.Dx()), float64(b.Dy()), layerShift(wd.gx), layerShift(wd.gy)), func(buf *ebiten.Image, dx, dy float64) {
			p := o
			p.Blend = ebiten.Blend{}
			p.GeoM.Translate(dx, dy)
			snapGeoM(&p.GeoM, p.Filter)
			buf.DrawImage(img, &p)
		})
		return
	}
	if !wd.mark(boxOf(o.GeoM, float64(b.Dx()), float64(b.Dy()), dx, dy)) {
		return
	}
	o.GeoM.Translate(dx, dy)
	snapGeoM(&o.GeoM, o.Filter)
	o.Blend = wd.lblend
	wd.layer.DrawImage(img, &o)
}

// DrawRectShader runs a shader over a w by h rectangle at art-pixel
// resolution, as before.
func (wd *worldDst) DrawRectShader(w, h int, s *ebiten.Shader, op *ebiten.DrawRectShaderOptions) {
	var o ebiten.DrawRectShaderOptions
	if op != nil {
		o = *op
	}
	box := boxOf(o.GeoM, float64(w), float64(h), layerShift(wd.gx), layerShift(wd.gy))
	dx, dy, ok := wd.into(o.Blend)
	if !ok {
		wd.odd(o.Blend, box, func(buf *ebiten.Image, dx, dy float64) {
			p := o
			p.Blend = ebiten.Blend{}
			p.GeoM.Translate(dx, dy)
			snapGeoM(&p.GeoM, ebiten.FilterNearest)
			buf.DrawRectShader(w, h, s, &p)
		})
		return
	}
	if !wd.mark(box) {
		return
	}
	o.GeoM.Translate(dx, dy)
	snapGeoM(&o.GeoM, ebiten.FilterNearest)
	o.Blend = wd.lblend
	wd.layer.DrawRectShader(w, h, s, &o)
}

// DrawTriangles draws triangles with their Dst in art pixels, at art-pixel
// resolution.
func (wd *worldDst) DrawTriangles(vs []ebiten.Vertex, is []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	var o ebiten.DrawTrianglesOptions
	if op != nil {
		o = *op
	}
	if len(vs) == 0 {
		return
	}
	moved := func(dx, dy float64) []ebiten.Vertex {
		out := make([]ebiten.Vertex, len(vs))
		copy(out, vs)
		for i := range out {
			out[i].DstX += float32(dx)
			out[i].DstY += float32(dy)
		}
		return out
	}
	box := vertexBox(vs, layerShift(wd.gx), layerShift(wd.gy))
	dx, dy, ok := wd.into(o.Blend)
	if !ok {
		wd.odd(o.Blend, box, func(buf *ebiten.Image, dx, dy float64) {
			p := o
			p.Blend = ebiten.Blend{}
			buf.DrawTriangles(moved(dx, dy), is, img, &p)
		})
		return
	}
	if !wd.mark(box) {
		return
	}
	o.Blend = wd.lblend
	wd.layer.DrawTriangles(moved(dx, dy), is, img, &o)
}

func vertexBox(vs []ebiten.Vertex, dx, dy float64) image.Rectangle {
	x0, y0 := math.Inf(1), math.Inf(1)
	x1, y1 := math.Inf(-1), math.Inf(-1)
	for _, v := range vs {
		x, y := float64(v.DstX), float64(v.DstY)
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	return image.Rect(int(math.Floor(x0+dx))-1, int(math.Floor(y0+dy))-1, int(math.Ceil(x1+dx))+1, int(math.Ceil(y1+dy))+1)
}

// odd draws density-1 art with a blend other than source-over or lighter:
// drawn alone at art-pixel resolution in the scratch image, which then goes
// on the canvas with that blend.
func (wd *worldDst) odd(blend ebiten.Blend, box image.Rectangle, draw func(buf *ebiten.Image, dx, dy float64)) {
	box = box.Intersect(wd.scratch.Bounds())
	if box.Empty() {
		return
	}
	wd.flush()
	draw(wd.scratch, layerShift(wd.gx), layerShift(wd.gy))
	wd.copyUp(wd.scratch, box, wd.gx, wd.gy, blend)
	wd.scratch.SubImage(box).(*ebiten.Image).Clear()
}

// vec runs a vector drawing call on the layer: draw gets the image and the
// shift from art pixels to its pixels. box is in art pixels.
func (wd *worldDst) vec(x0, y0, x1, y1 float64, draw func(img *ebiten.Image, dx, dy float32)) {
	dx, dy, _ := wd.into(ebiten.BlendSourceOver) // vector draws are source-over
	if !wd.mark(image.Rect(int(math.Floor(x0+dx))-1, int(math.Floor(y0+dy))-1, int(math.Ceil(x1+dx))+1, int(math.Ceil(y1+dy))+1)) {
		return
	}
	draw(wd.layer, float32(dx), float32(dy))
}

// drawFine draws an HD image straight onto the canvas: op is in art pixels
// (the density already folded in).
func (wd *worldDst) drawFine(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	wd.flush()
	o := *op
	o.GeoM.Scale(wd.r, wd.r)
	pickFilter(&o, op.Filter)
	wd.canvas.DrawImage(img, &o)
}

// fineBuf is a scratch drawTarget in art pixels on an image HDDensity times
// finer, so a figure drawn in it and put on the world afterwards (thrown
// back by a blow, cut off at the water) keeps its HD detail. Put it on
// another target with drawImageD(dst, b.img, op, HDDensity).
type fineBuf struct {
	img  *ebiten.Image
	w, h int // in art pixels
}

func newFineBuf(w, h int) *fineBuf {
	return &fineBuf{ebiten.NewImage(w*HDDensity, h*HDDensity), w, h}
}

func (b *fineBuf) Bounds() image.Rectangle { return image.Rect(0, 0, b.w, b.h) }

// DrawImage draws density-1 art, in art pixels.
func (b *fineBuf) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	var o ebiten.DrawImageOptions
	if op != nil {
		o = *op
	}
	snapGeoM(&o.GeoM, o.Filter)
	o.GeoM.Scale(HDDensity, HDDensity)
	b.img.DrawImage(img, &o)
}

func (b *fineBuf) drawFine(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	o := *op
	o.GeoM.Scale(HDDensity, HDDensity)
	pickFilter(&o, op.Filter)
	b.img.DrawImage(img, &o)
}

func (b *fineBuf) DrawRectShader(w, h int, s *ebiten.Shader, op *ebiten.DrawRectShaderOptions) {
	o := *op
	o.GeoM.Scale(HDDensity, HDDensity)
	b.img.DrawRectShader(w, h, s, &o)
}

func (b *fineBuf) DrawTriangles(vs []ebiten.Vertex, is []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	scaled := make([]ebiten.Vertex, len(vs))
	for i, v := range vs {
		v.DstX, v.DstY = v.DstX*HDDensity, v.DstY*HDDensity
		scaled[i] = v
	}
	b.img.DrawTriangles(scaled, is, img, op)
}

// vec runs a vector drawing call on the buffer, scaled up.
func (b *fineBuf) vec(draw func(img *ebiten.Image, k float32)) { draw(b.img, HDDensity) }

// drawColorMD is drawColorM for an image at the given density, such as a
// fineBuf's.
func drawColorMD(dst drawTarget, img *ebiten.Image, cm colorm.ColorM, op *colorm.DrawImageOptions, density int) {
	if density <= 1 {
		drawColorM(dst, img, cm, op)
		return
	}
	o := *op
	var m ebiten.GeoM
	m.Scale(1/float64(density), 1/float64(density))
	m.Concat(op.GeoM)
	o.GeoM = m
	o.Filter = ebiten.FilterLinear
	switch d := dst.(type) {
	case *ebiten.Image:
		colorm.DrawImage(d, img, cm, &o)
	case *fineBuf:
		o.GeoM.Scale(HDDensity, HDDensity)
		colorm.DrawImage(d.img, img, cm, &o)
	case *worldDst:
		d.flush()
		o.GeoM.Scale(d.r, d.r)
		if whole(o.GeoM.Element(0, 0)) && whole(o.GeoM.Element(1, 1)) && o.GeoM.Element(0, 1) == 0 && o.GeoM.Element(1, 0) == 0 {
			o.Filter = ebiten.FilterNearest
			tx, ty := o.GeoM.Element(0, 2), o.GeoM.Element(1, 2)
			o.GeoM.Translate(math.Round(tx)-tx, math.Round(ty)-ty)
		}
		colorm.DrawImage(d.canvas, img, cm, &o)
	}
}

// The vector calls the world code makes, on any drawTarget.

func fillRect(dst drawTarget, x, y, w, h float32, clr color.Color, aa bool) {
	switch d := dst.(type) {
	case *ebiten.Image:
		vector.FillRect(d, x, y, w, h, clr, aa)
	case *uiDst:
		k := float32(d.k)
		vector.FillRect(d.img, float32(d.X(float64(x))), float32(d.Y(float64(y))), w*k, h*k, clr, aa)
	case *fineBuf:
		X, Y := snapRect(x, y, aa)
		d.vec(func(img *ebiten.Image, k float32) { vector.FillRect(img, X*k, Y*k, w*k, h*k, clr, aa) })
	case *worldDst:
		d.vec(float64(x), float64(y), float64(x+w), float64(y+h), func(img *ebiten.Image, dx, dy float32) {
			X, Y := snapRect(x+dx, y+dy, aa)
			vector.FillRect(img, X, Y, w, h, clr, aa)
		})
	}
}

func strokeRect(dst drawTarget, x, y, w, h, sw float32, clr color.Color, aa bool) {
	switch d := dst.(type) {
	case *ebiten.Image:
		vector.StrokeRect(d, x, y, w, h, sw, clr, aa)
	case *uiDst:
		k := float32(d.k)
		vector.StrokeRect(d.img, float32(d.X(float64(x))), float32(d.Y(float64(y))), w*k, h*k, sw*k, clr, aa)
	case *fineBuf:
		X, Y := snapRect(x, y, aa)
		d.vec(func(img *ebiten.Image, k float32) { vector.StrokeRect(img, X*k, Y*k, w*k, h*k, sw*k, clr, aa) })
	case *worldDst:
		d.vec(float64(x-sw), float64(y-sw), float64(x+w+sw), float64(y+h+sw), func(img *ebiten.Image, dx, dy float32) {
			X, Y := snapRect(x+dx, y+dy, aa)
			vector.StrokeRect(img, X, Y, w, h, sw, clr, aa)
		})
	}
}

func strokeLine(dst drawTarget, x0, y0, x1, y1, sw float32, clr color.Color, aa bool) {
	switch d := dst.(type) {
	case *ebiten.Image:
		vector.StrokeLine(d, x0, y0, x1, y1, sw, clr, aa)
	case *uiDst:
		vector.StrokeLine(d.img, float32(d.X(float64(x0))), float32(d.Y(float64(y0))), float32(d.X(float64(x1))), float32(d.Y(float64(y1))), sw*float32(d.k), clr, aa)
	case *fineBuf:
		d.vec(func(img *ebiten.Image, k float32) { vector.StrokeLine(img, x0*k, y0*k, x1*k, y1*k, sw*k, clr, aa) })
	case *worldDst:
		d.vec(float64(min(x0, x1)-sw), float64(min(y0, y1)-sw), float64(max(x0, x1)+sw), float64(max(y0, y1)+sw), func(img *ebiten.Image, dx, dy float32) {
			vector.StrokeLine(img, x0+dx, y0+dy, x1+dx, y1+dy, sw, clr, aa)
		})
	}
}

func fillCircle(dst drawTarget, cx, cy, r float32, clr color.Color, aa bool) {
	switch d := dst.(type) {
	case *ebiten.Image:
		vector.FillCircle(d, cx, cy, r, clr, aa)
	case *uiDst:
		vector.FillCircle(d.img, float32(d.X(float64(cx))), float32(d.Y(float64(cy))), r*float32(d.k), clr, aa)
	case *fineBuf:
		d.vec(func(img *ebiten.Image, k float32) { vector.FillCircle(img, cx*k, cy*k, r*k, clr, aa) })
	case *worldDst:
		d.vec(float64(cx-r), float64(cy-r), float64(cx+r), float64(cy+r), func(img *ebiten.Image, dx, dy float32) {
			vector.FillCircle(img, cx+dx, cy+dy, r, clr, aa)
		})
	}
}

func strokeCircle(dst drawTarget, cx, cy, r, sw float32, clr color.Color, aa bool) {
	switch d := dst.(type) {
	case *ebiten.Image:
		vector.StrokeCircle(d, cx, cy, r, sw, clr, aa)
	case *uiDst:
		k := float32(d.k)
		vector.StrokeCircle(d.img, float32(d.X(float64(cx))), float32(d.Y(float64(cy))), r*k, sw*k, clr, aa)
	case *fineBuf:
		d.vec(func(img *ebiten.Image, k float32) { vector.StrokeCircle(img, cx*k, cy*k, r*k, sw*k, clr, aa) })
	case *worldDst:
		d.vec(float64(cx-r-sw), float64(cy-r-sw), float64(cx+r+sw), float64(cy+r+sw), func(img *ebiten.Image, dx, dy float32) {
			vector.StrokeCircle(img, cx+dx, cy+dy, r, sw, clr, aa)
		})
	}
}

// snapRect puts a crisp rectangle's corner on whole layer pixels, as
// snapGeoM does for images.
func snapRect(x, y float32, aa bool) (float32, float32) {
	if aa {
		return x, y
	}
	return float32(roundHalfUp(float64(x))), float32(roundHalfUp(float64(y)))
}

// drawColorM draws img through a colour matrix: density-1 art.
func drawColorM(dst drawTarget, img *ebiten.Image, cm colorm.ColorM, op *colorm.DrawImageOptions) {
	switch d := dst.(type) {
	case *ebiten.Image:
		colorm.DrawImage(d, img, cm, op)
	case *uiDst:
		o := *op
		o.GeoM.Concat(d.geoM())
		colorm.DrawImage(d.img, img, cm, &o)
	case *fineBuf:
		o := *op
		snapGeoM(&o.GeoM, o.Filter)
		o.GeoM.Scale(HDDensity, HDDensity)
		colorm.DrawImage(d.img, img, cm, &o)
	case *worldDst:
		o := *op
		b := img.Bounds()
		dx, dy, ok := d.into(o.Blend)
		if !ok {
			d.odd(o.Blend, boxOf(o.GeoM, float64(b.Dx()), float64(b.Dy()), layerShift(d.gx), layerShift(d.gy)), func(buf *ebiten.Image, dx, dy float64) {
				p := o
				p.Blend = ebiten.Blend{}
				p.GeoM.Translate(dx, dy)
				snapGeoM(&p.GeoM, p.Filter)
				colorm.DrawImage(buf, img, cm, &p)
			})
			return
		}
		if !d.mark(boxOf(o.GeoM, float64(b.Dx()), float64(b.Dy()), dx, dy)) {
			return
		}
		o.GeoM.Translate(dx, dy)
		snapGeoM(&o.GeoM, o.Filter)
		o.Blend = d.lblend
		colorm.DrawImage(d.layer, img, cm, &o)
	}
}

// drawShaderFine runs a shader over a w by h rectangle placed by op in art
// pixels, at the full resolution of the canvas when dst is the world pass
// (one fragment to a canvas pixel), so fine moving detail such as the water
// of terrainanim.go keeps the grain of the HD art.
func drawShaderFine(dst drawTarget, w, h int, s *ebiten.Shader, op *ebiten.DrawRectShaderOptions) {
	if wd, ok := dst.(*worldDst); ok {
		wd.flush()
		o := *op
		o.GeoM.Scale(wd.r, wd.r)
		wd.canvas.DrawRectShader(w, h, s, &o)
		return
	}
	dst.DrawRectShader(w, h, s, op)
}
