package client

import (
	"embed"
	"image"
	"image/color"
	_ "image/png" // the drawings in dinos/
	"math"
	"sync"
)

// The dinosaurs are drawn from hand-made pixel art: one side-on drawing per
// species in dinos/, made at the size the game shows it. Every other pose is
// worked from that drawing by a simple rig, so the art stays the artist's:
//
//   - legs step: each leg's pixels lean forward or back, more at the foot;
//   - the head and neck turn about the base of the neck (to eat, call or
//     strike), with the joint filled so it never comes apart;
//   - the jaw opens about the corner of the mouth, showing the mouth and,
//     for meat-eaters, teeth;
//   - a tail or a wing turns about its root; and the whole body can crouch,
//     tilt or leave the ground.
//
// A pose that reaches past the drawing's edges gets a larger canvas, with its
// anchor (the ground point) moved to match.

//go:embed dinos/*.png
var dinoFiles embed.FS

type rigPt struct{ x, y float64 }

// rigLimb is a leg: columns x0 to x1, from row top down to the feet.
type rigLimb struct{ x0, x1, top int }

// rigPart is a piece that turns about a joint: the pixels on the far side of
// the line through pivot whose normal points along dir (radians), and above
// row maxY.
type rigPart struct {
	pivot rigPt
	dir   float64
	maxY  float64
	all   bool // the whole drawing
}

func (p rigPart) has(x, y float64) bool {
	if p.all {
		return true
	}
	if y >= p.maxY {
		return false
	}
	return (x-p.pivot.x)*math.Cos(p.dir)+(y-p.pivot.y)*math.Sin(p.dir) > 0
}

// dinoRig is how one drawing moves.
type dinoRig struct {
	file   string
	ax, ay float64 // the ground point, in pixels of the drawing; 0, 0: under the feet
	legs   []rigLimb
	stride float64 // how far a foot moves in a step, in pixels
	head   rigPart
	neck   rigPart  // the root of a long neck, which bends there too; the zero value when none
	jaw    [2]rigPt // the corner of the mouth and the tip of the snout
	teeth  bool
	tail   rigPart // the zero value when it has none to swing
	wings  [2]rigPart
}

// rigPose is one pose worked from a rig.
type rigPose struct {
	step   float64 // legs: -1 back, +1 forward, 0 as drawn
	head   float64 // radians, + = nose down
	neck   float64 // radians, + = the whole neck down, after the head
	open   float64 // the jaw, radians
	tail   float64 // radians, + = tip up
	wings  float64 // radians, + = tips up
	squash float64 // crouch to this share of the height; 0 = standing
	tilt   float64 // the whole body, radians, + = nose down
	lift   float64 // pixels off the ground
}

var (
	dinoMouth  = mkRamp(0x2a0c0c, 0x5a1a1a, 0x8a3030)
	dinoImages = map[string]*canvas{}
	dinoMu     sync.Mutex
)

// dinoImage loads drawing name from dinos/, or nil if it is not there.
func dinoImage(name string) *canvas {
	dinoMu.Lock()
	defer dinoMu.Unlock()
	if c, ok := dinoImages[name]; ok {
		return c
	}
	var c *canvas
	if f, err := dinoFiles.Open("dinos/" + name + ".png"); err == nil {
		if im, _, err := image.Decode(f); err == nil {
			b := im.Bounds()
			c = newCanvas(b.Dx(), b.Dy())
			for y := 0; y < b.Dy(); y++ {
				for x := 0; x < b.Dx(); x++ {
					n := color.NRGBAModel.Convert(im.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
					c.px[y*c.w+x] = color.RGBA{n.R, n.G, n.B, n.A} // the canvas keeps straight alpha
				}
			}
		}
		f.Close()
	}
	dinoImages[name] = c
	return c
}

// rigCanvas draws pose p of rig r, with its anchor in pixels. ok is false
// when the drawing is missing, so the caller can draw its own.
func rigCanvas(r dinoRig, p rigPose) (c *canvas, ax, ay float64, ok bool) {
	base := dinoImage(r.file)
	if base == nil {
		return nil, 0, 0, false
	}
	const pad = 48 // room for a pose to reach past the drawing
	w := newCanvas(base.w+2*pad, base.h+2*pad)
	w.over(base, pad, pad)
	off := func(q rigPt) rigPt { return rigPt{q.x + pad, q.y + pad} }
	shift := func(pr rigPart) rigPart { pr.pivot = off(pr.pivot); pr.maxY += pad; return pr }
	ax0, ay0 := r.ax, r.ay
	if ax0 == 0 && ay0 == 0 {
		ax0, ay0 = underFeet(base)
	}
	gx, gy := ax0+pad, ay0+pad
	joint := max(4, float64(base.h)/12) // how deep a band of the neck fills its joint

	if p.step != 0 {
		for i, l := range r.legs {
			sign := 1.0
			if i%2 == 1 {
				sign = -1
			}
			stepLeg(w, l.x0+pad, l.x1+pad, l.top+pad, pad+base.h-1, p.step*sign*r.stride)
		}
	}
	if p.open != 0 && r.jaw[0] != r.jaw[1] {
		openJaw(w, off(r.jaw[0]), off(r.jaw[1]), shift(r.head), p.open, r.teeth)
	}
	if p.head != 0 {
		turn(w, shift(r.head), p.head, joint)
	}
	if p.neck != 0 && r.neck.maxY != 0 {
		turn(w, shift(r.neck), p.neck, joint)
	}
	if p.tail != 0 && r.tail.maxY != 0 {
		turn(w, shift(r.tail), p.tail, 4) // + lifts a tail that points back
	}
	if p.wings != 0 {
		for i, wg := range r.wings {
			if wg.maxY == 0 {
				continue
			}
			a := p.wings // the left wing's tip rises turning one way...
			if i == 1 {
				a = -p.wings // ...the right wing's the other
			}
			turn(w, shift(wg), a, 3)
		}
	}
	if p.squash > 0 {
		crouch(w, gy, p.squash)
	}
	if p.tilt != 0 {
		turn(w, rigPart{pivot: rigPt{gx, gy - float64(base.h)/3}, all: true}, p.tilt, 0)
	}
	if p.lift != 0 {
		raise(w, int(math.Round(p.lift)))
	}
	return cropTo(w, pad, base.w, base.h, gx, gy)
}

// underFeet is the ground point of a drawing: the middle of what touches
// its bottom rows, on its bottom row.
func underFeet(c *canvas) (float64, float64) {
	bottom := -1
	for y := c.h - 1; y >= 0 && bottom < 0; y-- {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A >= 128 {
				bottom = y
				break
			}
		}
	}
	sum, n := 0.0, 0
	for y := max(0, bottom-2); y <= bottom; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A >= 128 {
				sum += float64(x) + 0.5
				n++
			}
		}
	}
	if n == 0 {
		return float64(c.w) / 2, float64(c.h)
	}
	return sum / float64(n), float64(bottom) + 0.5
}

// stepLeg leans the pixels of columns x0..x1, from row top to row bottom, by
// up to dx at the foot.
func stepLeg(c *canvas, x0, x1, top, bottom int, dx float64) {
	src := append([]color.RGBA(nil), c.px...)
	for y := top; y <= bottom; y++ {
		for x := x0; x <= x1; x++ {
			c.px[y*c.w+x] = color.RGBA{}
		}
	}
	for y := top; y <= bottom; y++ {
		d := int(math.Round(dx * float64(y-top) / float64(max(1, bottom-top))))
		for x := x0; x <= x1; x++ {
			if p := src[y*c.w+x]; p.A >= 128 {
				c.set(x+d, y, p)
			}
		}
	}
}

// turn rotates the pixels of rigPart pr by a radians about its pivot. Near the
// pivot it also paints the in-between angles, so the joint stays whole.
func turn(c *canvas, pr rigPart, a float64, joint float64) {
	src := append([]color.RGBA(nil), c.px...)
	in := func(x, y int) bool {
		return src[y*c.w+x].A >= 128 && pr.has(float64(x)+0.5, float64(y)+0.5)
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if in(x, y) {
				c.px[y*c.w+x] = color.RGBA{}
			}
		}
	}
	ux, uy := math.Cos(pr.dir), math.Sin(pr.dir)
	paint := func(b float64, near float64) {
		sn, cs := math.Sin(-b), math.Cos(-b)
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				dx, dy := float64(x)+0.5-pr.pivot.x, float64(y)+0.5-pr.pivot.y
				fx, fy := pr.pivot.x+dx*cs-dy*sn, pr.pivot.y+dx*sn+dy*cs
				if near > 0 && (fx-pr.pivot.x)*ux+(fy-pr.pivot.y)*uy > near {
					continue // only the band along the cut fills the joint
				}
				sx, sy := int(math.Floor(fx)), int(math.Floor(fy))
				if sx >= 0 && sy >= 0 && sx < c.w && sy < c.h && in(sx, sy) {
					c.px[y*c.w+x] = src[sy*c.w+sx]
				}
			}
		}
	}
	if joint > 0 {
		const steps = 12
		for k := 1; k < steps; k++ { // the joint, at the angles between
			paint(a*float64(k)/steps, joint)
		}
	}
	paint(a, 0)
}

// openJaw drops the lower jaw: the head's pixels just below the line from
// the corner of the mouth to the tip of the snout, turned a radians about the
// corner. The wedge it opens is the mouth, with teeth along both jaws for a
// meat-eater.
func openJaw(c *canvas, corner, tip rigPt, head rigPart, a float64, teeth bool) {
	ux, uy := tip.x-corner.x, tip.y-corner.y
	l := math.Hypot(ux, uy)
	ux, uy = ux/l, uy/l
	// along runs from the corner to the tip; across is how far below the line.
	frame := func(x, y float64) (along, across float64) {
		return (x-corner.x)*ux + (y-corner.y)*uy, (x-corner.x)*-uy + (y-corner.y)*ux
	}
	isJaw := func(x, y float64) bool {
		along, across := frame(x, y)
		return head.has(x, y) && along > -1 && along < l+2 && across >= 0 && across < 6
	}
	src := append([]color.RGBA(nil), c.px...)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if src[y*c.w+x].A >= 128 && isJaw(float64(x)+0.5, float64(y)+0.5) {
				c.px[y*c.w+x] = color.RGBA{}
			}
		}
	}
	gape := math.Tan(a)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			along, across := frame(float64(x)+0.5, float64(y)+0.5)
			if along <= 0.5 || along > l || across < 0 || across > along*gape+0.5 {
				continue
			}
			col := dinoMouth.pick(1-across/(along*gape+1), x, y)
			switch {
			case teeth && along > 2 && int(along)%3 == 1 && across < 1.2:
				col = colTooth // along the upper jaw
			case teeth && along > 3 && int(along)%3 == 2 && across > along*gape-1:
				col = colTooth // along the lower
			}
			c.px[y*c.w+x] = col
		}
	}
	sn, cs := math.Sin(-a), math.Cos(-a)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := float64(x)+0.5-corner.x, float64(y)+0.5-corner.y
			sx := int(math.Floor(corner.x + dx*cs - dy*sn))
			sy := int(math.Floor(corner.y + dx*sn + dy*cs))
			if sx >= 0 && sy >= 0 && sx < c.w && sy < c.h && src[sy*c.w+sx].A >= 128 &&
				isJaw(float64(sx)+0.5, float64(sy)+0.5) {
				c.px[y*c.w+x] = src[sy*c.w+sx]
			}
		}
	}
}

// crouch squashes the drawing towards the ground row gy.
func crouch(c *canvas, gy, k float64) {
	src := append([]color.RGBA(nil), c.px...)
	clear(c.px)
	for y := 0; y < c.h; y++ {
		sy := int(math.Floor(gy - (gy-(float64(y)+0.5))/k))
		if sy < 0 || sy >= c.h {
			continue
		}
		copy(c.px[y*c.w:(y+1)*c.w], src[sy*c.w:(sy+1)*c.w])
	}
}

// raise moves the whole drawing up by n pixels.
func raise(c *canvas, n int) {
	src := append([]color.RGBA(nil), c.px...)
	clear(c.px)
	for y := 0; y+n < c.h; y++ {
		if y+n >= 0 {
			copy(c.px[y*c.w:(y+1)*c.w], src[(y+n)*c.w:(y+n+1)*c.w])
		}
	}
}

// cropTo cuts the working canvas back to the drawing's own w x h, grown
// to take in anything the pose moved outside it, and returns the anchor.
func cropTo(c *canvas, pad, w, h int, gx, gy float64) (*canvas, float64, float64, bool) {
	x0, y0, x1, y1 := pad, pad, pad+w, pad+h
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A >= 128 {
				x0, y0 = min(x0, x), min(y0, y)
				x1, y1 = max(x1, x+1), max(y1, y+1)
			}
		}
	}
	out := newCanvas(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		copy(out.px[(y-y0)*out.w:(y-y0+1)*out.w], c.px[y*c.w+x0:y*c.w+x1])
	}
	return out, gx - float64(x0), gy - float64(y0), true
}
