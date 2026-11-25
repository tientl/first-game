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
//   - legs step: each leg is cut into thigh, shin and foot, which turn about
//     its hip, knee and ankle so the foot goes where the stride puts it
//     (dinolegs.go); a leg with no joints measured leans whole instead;
//   - the head and neck turn about the base of the neck (to eat, call or
//     strike), with the joint filled so it never comes apart;
//   - the jaw opens about the corner of the mouth, showing the mouth and,
//     for meat-eaters, teeth;
//   - a tail or a wing turns about its root; and the whole body can crouch,
//     tilt or leave the ground.
//
// A pose that reaches past the drawing's edges gets a larger canvas, with its
// anchor (the ground point) moved to match.

//go:embed dinos/*.png dinos/hd/*.png
var dinoFiles embed.FS

// dinoHD draws the dinosaurs from the drawings of dinos/hd, redrawn
// HDDensity times as fine (see dinos/hd/gen.py); false brings back the
// density-1 drawings. A rig is measured in the pixels of the density-1
// drawing whichever is used, and so are its poses.
var dinoHD = true

type rigPt struct{ x, y float64 }

// rigLimb is a leg: columns x0 to x1, from row top down to the feet. j is
// its joints when it has them measured (see dinolegs.go): hip, knee, ankle,
// the ball of the foot for one that walks on its toes, and the tip of the
// toes. fore marks a foreleg, whose elbow bends back. r is how far around
// the thigh bone the flank above top moves with the thigh; 0: half the
// thigh's length.
type rigLimb struct {
	x0, x1, top int
	j           []rigPt
	fore        bool
	r           float64
}

// hind is a hind leg with its joints, fore a foreleg.
func hind(x0, x1, top int, j ...rigPt) rigLimb { return rigLimb{x0: x0, x1: x1, top: top, j: j} }
func fore(x0, x1, top int, j ...rigPt) rigLimb {
	return rigLimb{x0: x0, x1: x1, top: top, j: j, fore: true}
}

// rigPart is a piece that turns about a joint: the pixels on the far side of
// the line through pivot whose normal points along dir (radians), and above
// row maxY.
type rigPart struct {
	pivot rigPt
	dir   float64
	maxY  float64
	all   bool // the whole drawing
}

// tester is has with the trigonometry done once, for a loop over pixels.
func (p rigPart) tester() func(x, y float64) bool {
	if p.all {
		return func(x, y float64) bool { return true }
	}
	ux, uy := math.Cos(p.dir), math.Sin(p.dir)
	return func(x, y float64) bool {
		return y < p.maxY && (x-p.pivot.x)*ux+(y-p.pivot.y)*uy > 0
	}
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
	runner bool    // light enough to leave the ground between footfalls at a run
	// gait is when in a stride each leg steps, in strides; nil: each leg half
	// a stride after the one before. far marks the legs on the far side
	// (bit i for legs[i]), which are drawn behind the near ones.
	gait  []float64
	far   uint
	head  rigPart
	neck  rigPart  // the root of a long neck, which bends there too; the zero value when none
	jaw   [2]rigPt // the corner of the mouth and the tip of the snout
	teeth bool
	tail  rigPart // the zero value when it has none to swing
	wings [2]rigPart
}

// rigPose is one pose worked from a rig.
type rigPose struct {
	step   float64 // legs: -1 back, +1 forward, 0 as drawn
	cycle  float64 // 0: the legs held at step. Otherwise how far through a stride, in (0, 1]: the legs swing step wide, feet lifting, the body bobbing
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

// dinoImage loads drawing name from dinos/ (hd: from dinos/hd, HDDensity
// times as fine), or nil if it is not there.
func dinoImage(name string, hd bool) *canvas {
	dinoMu.Lock()
	defer dinoMu.Unlock()
	path := "dinos/" + name + ".png"
	if hd {
		path = "dinos/hd/" + name + ".png"
	}
	if c, ok := dinoImages[path]; ok {
		return c
	}
	var c *canvas
	if f, err := dinoFiles.Open(path); err == nil {
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
	dinoImages[path] = c
	return c
}

// rigCanvas draws pose p of rig r, with its anchor in the canvas's pixels.
// The canvas is at density c.dens (HDDensity when dinoHD), which mk turns
// into a sprite of that density. ok is false when the drawing is missing, so
// the caller can draw its own.
func rigCanvas(r dinoRig, p rigPose) (c *canvas, ax, ay float64, ok bool) {
	low := dinoImage(r.file, false)
	if low == nil {
		return nil, 0, 0, false
	}
	base, k := low, 1
	if dinoHD {
		if hd := dinoImage(r.file, true); hd != nil && hd.w == low.w*HDDensity && hd.h == low.h*HDDensity {
			base, k = hd, HDDensity
		}
	}
	kf := float64(k)
	pad := 48 * k // room for a pose to reach past the drawing
	w := newCanvas(base.w+2*pad, base.h+2*pad)
	w.over(base, pad, pad)
	ax0, ay0 := r.ax, r.ay
	if ax0 == 0 && ay0 == 0 {
		ax0, ay0 = underFeet(low) // from the density-1 drawing: the anchor is the same at any density
	}
	gx, gy := ax0*kf+float64(pad), ay0*kf+float64(pad)
	joint := max(4, float64(low.h)/12) * kf // how deep a band of the neck fills its joint

	bob := 0.0 // how far the body has sunk on its legs, in canvas pixels
	if (p.step != 0 || p.cycle != 0) && len(r.legs) > 0 {
		if r.jointed() {
			bob = stepJoints(w, r, p, pad, k)
		} else {
			bob = stepLegs(w, r, p, pad, base.h, k) // no joints measured: the legs lean whole
		}
	}
	off := func(q rigPt) rigPt { return rigPt{q.x*kf + float64(pad), q.y*kf + float64(pad) + bob} }
	shift := func(pr rigPart) rigPart {
		pr.pivot = off(pr.pivot)
		pr.maxY = pr.maxY*kf + float64(pad) + bob
		return pr
	}
	if p.open != 0 && r.jaw[0] != r.jaw[1] {
		openJaw(w, off(r.jaw[0]), off(r.jaw[1]), shift(r.head), p.open, r.teeth, kf)
	}
	if p.head != 0 {
		turn(w, shift(r.head), p.head, joint)
	}
	if p.neck != 0 && r.neck.maxY != 0 {
		turn(w, shift(r.neck), p.neck, joint)
	}
	if p.tail != 0 && r.tail.maxY != 0 {
		turn(w, shift(r.tail), p.tail, 4*kf) // + lifts a tail that points back
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
			turn(w, shift(wg), a, 3*kf)
		}
	}
	if p.squash > 0 && p.squash != 1 {
		crouch(w, gy, p.squash)
	}
	if p.tilt != 0 {
		turn(w, rigPart{pivot: rigPt{gx, gy - float64(base.h)/3}, all: true}, p.tilt, 0)
	}
	if p.lift != 0 {
		raise(w, int(math.Round(p.lift*kf)))
	}
	c, ax, ay, ok = cropTo(w, pad, base.w, base.h, gx, gy)
	c.dens = k
	return c, ax, ay, ok
}

// stepLegs moves the legs of a rig with no joints measured (stepJoints
// moves the others) on working canvas w for pose p, and
// returns how far the body sank between them, in canvas pixels. Held at a
// step, each leg leans forward or back in turn, as drawn. Through a stride
// (p.cycle) each swings in its own time: back along the ground, then lifted
// and forward, while the body sinks as the legs spread and rises over them.
// gaitReach is how much wider a leg swings through a stride than the lean
// of a standing step.
const gaitReach = 2.2

func stepLegs(w *canvas, r dinoRig, p rigPose, pad, h, k int) float64 {
	kf := float64(k)
	bottom := pad + h - 1
	type cut struct {
		x0, x1, top int
		dx, lift    float64
		px          []color.RGBA
	}
	cuts := make([]cut, len(r.legs))
	th := 2 * math.Pi * p.cycle
	for i, l := range r.legs {
		c := cut{x0: l.x0*k + pad, x1: (l.x1+1)*k - 1 + pad, top: l.top*k + pad}
		if p.cycle != 0 {
			g := float64(i) * 0.5
			if i < len(r.gait) {
				g = r.gait[i]
			}
			a := th + 2*math.Pi*g
			reach := p.step * r.stride * gaitReach * kf // a long, ground-eating stride
			c.dx = reach * math.Sin(a)
			c.lift = math.Max(0, math.Cos(a)) * math.Abs(reach) * 0.4
		} else {
			c.dx = p.step * r.stride * kf
			if i%2 == 1 {
				c.dx = -c.dx
			}
		}
		cw := c.x1 - c.x0 + 1
		c.px = make([]color.RGBA, cw*(bottom-c.top+1))
		for y := c.top; y <= bottom; y++ {
			row := w.px[y*w.w+c.x0 : y*w.w+c.x1+1]
			copy(c.px[(y-c.top)*cw:], row)
			clear(row)
		}
		cuts[i] = c
	}
	bob := 0.0
	if p.cycle != 0 {
		amp := 0.3
		if len(r.legs) > 2 {
			amp = 0.14 // four legs carry the body more evenly
		}
		bob = math.Round(math.Abs(p.step) * r.stride * kf * amp * (1 - math.Cos(2*th)) / 2)
		if bob > 0 {
			raise(w, -int(bob))
		}
	}
	paint := func(c cut) {
		cw := c.x1 - c.x0 + 1
		y0, y1 := c.top+int(bob), bottom-int(math.Round(c.lift))
		for y := y0; y <= y1; y++ {
			t := float64(y-y0) / float64(max(1, y1-y0))
			sy := int(math.Round(t * float64(bottom-c.top)))
			d := int(math.Round(c.dx * t))
			for x := 0; x < cw; x++ {
				if q := c.px[sy*cw+x]; q.A >= 128 {
					w.set(c.x0+x+d, y, q)
				}
			}
		}
	}
	for pass := range 2 { // the far legs, then the near ones over them
		for i, c := range cuts {
			if (r.far>>uint(i)&1 == 1) == (pass == 0) {
				paint(c)
			}
		}
	}
	return bob
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

// turn rotates the pixels of rigPart pr by a radians about its pivot. Near the
// pivot it also paints the in-between angles, so the joint stays whole.
func turn(c *canvas, pr rigPart, a float64, joint float64) {
	src := append([]color.RGBA(nil), c.px...)
	has := pr.tester()
	in := func(x, y int) bool {
		return src[y*c.w+x].A >= 128 && has(float64(x)+0.5, float64(y)+0.5)
	}
	// Only as far from the pivot as the part reaches can anything change.
	reach := 0.0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if in(x, y) {
				c.px[y*c.w+x] = color.RGBA{}
				reach = max(reach, math.Hypot(float64(x)+0.5-pr.pivot.x, float64(y)+0.5-pr.pivot.y))
			}
		}
	}
	x0, x1 := max(0, int(pr.pivot.x-reach)-1), min(c.w, int(pr.pivot.x+reach)+2)
	y0, y1 := max(0, int(pr.pivot.y-reach)-1), min(c.h, int(pr.pivot.y+reach)+2)
	ux, uy := math.Cos(pr.dir), math.Sin(pr.dir)
	const steps = 12
	var sn, cs [steps + 1]float64 // the angles between, for the joint; the last is the turn itself
	for k := 1; k <= steps; k++ {
		sn[k], cs[k] = math.Sincos(-a * float64(k) / steps)
	}
	first := 1
	if joint <= 0 {
		first = steps
	}
	r2 := (reach + 1) * (reach + 1)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dx, dy := float64(x)+0.5-pr.pivot.x, float64(y)+0.5-pr.pivot.y
			if dx*dx+dy*dy > r2 {
				continue
			}
			from := first
			if d, sweep := dx*ux+dy*uy, math.Sqrt(dx*dx+dy*dy)*math.Abs(a); d-sweep > joint || d+sweep < 0 {
				from = steps // too far from the cut to be in the joint at any angle between
			}
			for k := steps; k >= from; k-- { // the turn itself wins over the joint's filling
				fx, fy := dx*cs[k]-dy*sn[k], dx*sn[k]+dy*cs[k]
				if k < steps && fx*ux+fy*uy > joint {
					continue // only the band along the cut fills the joint
				}
				sx, sy := int(math.Floor(pr.pivot.x+fx)), int(math.Floor(pr.pivot.y+fy))
				if sx >= 0 && sy >= 0 && sx < c.w && sy < c.h && in(sx, sy) {
					c.px[y*c.w+x] = src[sy*c.w+sx]
					break
				}
			}
		}
	}
}

// openJaw drops the lower jaw: the head's pixels just below the line from
// the corner of the mouth to the tip of the snout, turned a radians about the
// corner. The wedge it opens is the mouth, with teeth along both jaws for a
// meat-eater.
func openJaw(c *canvas, corner, tip rigPt, head rigPart, a float64, teeth bool, kf float64) {
	ux, uy := tip.x-corner.x, tip.y-corner.y
	l := math.Hypot(ux, uy)
	ux, uy = ux/l, uy/l
	// along runs from the corner to the tip; across is how far below the line.
	frame := func(x, y float64) (along, across float64) {
		return (x-corner.x)*ux + (y-corner.y)*uy, (x-corner.x)*-uy + (y-corner.y)*ux
	}
	has := head.tester()
	// Nothing further from the corner than the jaw is long changes.
	reach := int(l + 8*kf)
	bx0, bx1 := max(0, int(corner.x)-reach), min(c.w, int(corner.x)+reach+1)
	by0, by1 := max(0, int(corner.y)-reach), min(c.h, int(corner.y)+reach+1)
	isJaw := func(x, y float64) bool {
		along, across := frame(x, y)
		return has(x, y) && along > -kf && along < l+2*kf && across >= 0 && across < 6*kf
	}
	src := append([]color.RGBA(nil), c.px...)
	for y := by0; y < by1; y++ {
		for x := bx0; x < bx1; x++ {
			if src[y*c.w+x].A >= 128 && isJaw(float64(x)+0.5, float64(y)+0.5) {
				c.px[y*c.w+x] = color.RGBA{}
			}
		}
	}
	gape := math.Tan(a)
	for y := by0; y < by1; y++ {
		for x := bx0; x < bx1; x++ {
			along, across := frame(float64(x)+0.5, float64(y)+0.5)
			if along <= 0.5 || along > l || across < 0 || across > along*gape+0.5 {
				continue
			}
			col := dinoMouth.pick(1-across/(along*gape+1), x, y)
			// Teeth every third pixel of the drawing: a point each when
			// there is room to draw one.
			u := along / kf
			point := func(at int) float64 { // how long a tooth is here, in pixels
				if int(u)%3 != at {
					return 0
				}
				if kf <= 1 {
					return 1.2
				}
				return kf * 1.5 * (1 - math.Abs(u-math.Floor(u)-0.5)*1.6)
			}
			switch {
			case teeth && u > 2 && across < point(1):
				col = colTooth // along the upper jaw
			case teeth && u > 3 && across > along*gape-point(2)+0.2:
				col = colTooth // along the lower
			}
			c.px[y*c.w+x] = col
		}
	}
	sn, cs := math.Sin(-a), math.Cos(-a)
	for y := by0; y < by1; y++ {
		for x := bx0; x < bx1; x++ {
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

// raise moves the whole drawing up by n pixels (down, when n < 0).
func raise(c *canvas, n int) {
	src := append([]color.RGBA(nil), c.px...)
	clear(c.px)
	for y := 0; y < c.h; y++ {
		if y+n >= 0 && y+n < c.h {
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
