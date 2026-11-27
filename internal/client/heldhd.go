package client

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// HD held items: the tools and weapons in a player's hand, painted at the
// figure's own density (HDDensity), in the look of the HD item icons
// (iconpaint.go, iconhd_pilot.go): light from the top left, the selective
// outline, the dark line where one part lies over another, knapped stone,
// flint gloss, lashings and wood grain, each tier in its own stone.
//
// A held item is painted in its own frame: art pixels, the grip (where the
// fist closes on it) at (0, 0), the working end straight up (-y), as the
// old held sprites stood before they were turned. Tools turn through their
// swings, so each item is painted anew at heldTurns angles, the first time
// it is shown, every shape turned before it is painted: the edges are
// clean at every angle, the light stays in the top left, the outline stays
// one pixel. Each angle is painted heldSS times finer and shrunk, and the
// item's angles are painted ahead on another goroutine the first time it
// is shown, so the first swing does not stall a frame. Facing left, the
// image is mirrored, as the figure is.
//
// Where the fist closes on the haft, the haft is left out (heldDef.grip):
// the fist, drawn before it, shows through, so the hand is round the haft
// rather than under it.
//
// An item without a held painter is shown as its bag icon (drawIcon): the
// HD icon if it has one, else the old icon redrawn three times as fine by
// upscale3 (iconFine).

// heldHD turns the HD held items on; off, the old density-1 sprites.
var heldHD = true

// SetHeldHD turns the HD held items on or off (shot -heldhd=false).
func SetHeldHD(on bool) { heldHD = on }

// heldDef is an item as held, in its own frame (see above).
type heldDef struct {
	paint          func(h *hpv)
	x0, y0, x1, y1 float64 // what it covers in its frame, art pixels
	grip           float64 // the radius of the gap the fist fills, art pixels; 0 for none
}

// heldDefs holds the held painter of every item that has one.
var heldDefs [sim.NumItems]*heldDef

// regHeld gives item k a held painter. Call it from an init function.
func regHeld(k sim.ItemKind, d *heldDef) {
	if k <= sim.ItemNone || k >= sim.NumItems {
		panic("regHeld: no such item")
	}
	if heldDefs[k] != nil {
		panic("regHeld: " + k.Def().Name + " has a held painter already")
	}
	heldDefs[k] = d
}

// heldDefOf is k's held painter, or nil.
func heldDefOf(k sim.ItemKind) *heldDef {
	if !heldHD || k <= sim.ItemNone || k >= sim.NumItems {
		return nil
	}
	return heldDefs[k]
}

// --- painting in the item's frame ------------------------------------------------

// hpv is a held item being painted at one angle: an icon canvas (icv) in
// art pixels, and the turn from the item's frame (u across, v along, the
// head at -v) to it. Shapes are made in the item's frame and turned by
// fill and the like, so the shaders see the canvas's own normals (f.gx,
// f.gy) and the light stays where the world's is; loc gives a pixel's place
// in the item's frame, for grain and scars that must stay on the item as it
// turns.
type hpv struct {
	*icv
	cs, sn float64 // the turn
	ox, oy float64 // the grip on the canvas, art pixels
	flip   bool    // mirrored across its axis before it is turned: an icon's edge to the light
	icon   bool    // painted as a bag icon (iconhd_tools.go): deeper tones, glints only on edges
}

// at is item point (u, v) on the canvas.
func (h *hpv) at(u, v float64) (float64, float64) {
	if h.flip {
		u = -u
	}
	return h.ox + u*h.cs - v*h.sn, h.oy + u*h.sn + v*h.cs
}

// loc is canvas point (x, y) in the item's frame.
func (h *hpv) loc(x, y float64) (u, v float64) {
	dx, dy := x-h.ox, y-h.oy
	u, v = dx*h.cs+dy*h.sn, -dx*h.sn+dy*h.cs
	if h.flip {
		u = -u
	}
	return u, v
}

// dir turns a direction of the item's frame onto the canvas.
func (h *hpv) dir(u, v float64) (float64, float64) {
	if h.flip {
		u = -u
	}
	return u*h.cs - v*h.sn, u*h.sn + v*h.cs
}

// lit is the light on a surface whose normal leans (u, v) in the item's
// frame out of the picture's plane, as litOf.
func (h *hpv) lit(u, v float64) float64 {
	x, y := h.dir(u, v)
	return litOf(x, y)
}

// rod is the light at u across a round rod r thick lying along v.
func (h *hpv) rod(u, r float64) float64 { return h.lit(max(-1, min(1, u/r)), 0) }

// xf is s, a shape in the item's frame, on the canvas.
func (h *hpv) xf(s shape) shape {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]float64{{s.x0, s.y0}, {s.x1, s.y0}, {s.x0, s.y1}, {s.x1, s.y1}} {
		x, y := h.at(c[0], c[1])
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	return shape{x0, y0, x1, y1, func(x, y float64) float64 {
		u, v := h.loc(x, y)
		return s.d(u, v)
	}}
}

// fill, add, on, under and soft are icv's, for shapes in the item's frame.
func (h *hpv) fill(s shape, sh ishader)  { h.icv.fill(h.xf(s), sh) }
func (h *hpv) add(s shape, sh ishader)   { h.icv.add(h.xf(s), sh) }
func (h *hpv) on(s shape, sh ishader)    { h.icv.on(h.xf(s), sh) }
func (h *hpv) under(s shape, sh ishader) { h.icv.under(h.xf(s), sh) }

// --- the bake ------------------------------------------------------------------------

const (
	heldTurns = 64 // angles painted, all the way round
	heldSS    = 3  // each painted this many times finer, then shrunk
	heldEdge  = 2  // image pixels kept round the item's box for its outline
)

// heldPix is one angle of a held item, painted: straight alpha, w by h,
// the grip gx, gy image pixels from the top left.
type heldPix struct {
	px     []color.RGBA
	w, h   int
	gx, gy float64
}

// turnAngle is the angle of turn i, radians.
func turnAngle(i int) float64 { return 2 * math.Pi * float64(i) / heldTurns }

// turnIndex is the turn nearest th radians.
func turnIndex(th float64) int {
	i := int(math.Round(th/(2*math.Pi)*heldTurns)) % heldTurns
	if i < 0 {
		i += heldTurns
	}
	return i
}

// bakeHeld paints d at turn i. It touches nothing shared, so it may run on
// any goroutine.
func bakeHeld(d *heldDef, i int) *heldPix {
	th := turnAngle(i)
	cs, sn := math.Cos(th), math.Sin(th)
	bx0, by0, bx1, by1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]float64{{d.x0, d.y0}, {d.x1, d.y0}, {d.x0, d.y1}, {d.x1, d.y1}} {
		x, y := c[0]*cs-c[1]*sn, c[0]*sn+c[1]*cs
		bx0, by0, bx1, by1 = min(bx0, x), min(by0, y), max(bx1, x), max(by1, y)
	}
	const K = HDDensity
	// The grip on a pixel corner, so it lands on the hand to the pixel.
	gx := heldEdge + int(math.Ceil(-bx0*K))
	gy := heldEdge + int(math.Ceil(-by0*K))
	n := max(gx+int(math.Ceil(bx1*K))+heldEdge, gy+int(math.Ceil(by1*K))+heldEdge)
	const m = heldSS
	p := &icv{n: n * m, k: K * m, px: make([]color.RGBA, n*m*n*m), part: make([]uint8, n*m*n*m)}
	h := &hpv{icv: p, cs: cs, sn: sn, ox: float64(gx) / K, oy: float64(gy) / K}
	d.paint(h)
	p.contact(m)
	px := shrink(p.px, n*m, m)
	px = outlineIcon(px, n, 1, p.edgeBare(m))
	if p.glow != nil {
		g := shrinkPM(p.glow, n*m, m)
		for j := range px {
			px[j] = icUnPM(overPM(g[j], icPM(px[j])))
		}
	}
	if d.grip > 0 { // the gap the fist fills
		r := d.grip * K
		for y := max(0, gy-int(r)-2); y < min(n, gy+int(r)+2); y++ {
			for x := max(0, gx-int(r)-2); x < min(n, gx+int(r)+2); x++ {
				dd := math.Hypot(float64(x)+0.5-float64(gx), float64(y)+0.5-float64(gy))
				if k := clamp01(dd - r + 0.5); k < 1 {
					c := &px[y*n+x]
					c.A = uint8(math.Round(float64(c.A) * k))
				}
			}
		}
	}
	// Trimmed to what is painted.
	x0, y0, x1, y1 := n, n, -1, -1
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if px[y*n+x].A != 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < x0 {
		return &heldPix{px: make([]color.RGBA, 1), w: 1, h: 1, gx: float64(gx), gy: float64(gy)}
	}
	w, hh := x1-x0+1, y1-y0+1
	out := &heldPix{px: make([]color.RGBA, w*hh), w: w, h: hh, gx: float64(gx - x0), gy: float64(gy - y0)}
	for y := 0; y < hh; y++ {
		copy(out.px[y*w:(y+1)*w], px[(y0+y)*n+x0:(y0+y)*n+x1+1])
	}
	return out
}

// heldFrame is one angle of a held item as an image.
type heldFrame struct {
	img    *ebiten.Image
	gx, gy float64 // the grip, image pixels
}

// heldSet is the angles of one held item: made into images on the game
// thread as they are first drawn, painted ahead by warm.
type heldSet struct {
	def   *heldDef
	frame [heldTurns]*heldFrame // game thread only

	mu    sync.Mutex
	ready [heldTurns]*heldPix // painted ahead, waiting to be made images
	made  [heldTurns]bool
}

// heldSets holds every held item shown so far. Game thread only.
var heldSets = map[*heldDef]*heldSet{}

// heldSetOf is the angles of d, its painting ahead begun on first use.
func heldSetOf(d *heldDef) *heldSet {
	s := heldSets[d]
	if s == nil {
		s = &heldSet{def: d}
		heldSets[d] = s
		if heldWarmAhead {
			go s.warm()
		}
	}
	return s
}

// heldWarmAhead paints a held item's angles on another goroutine when it
// is first shown; tests turn it off to time the painting alone.
var heldWarmAhead = true

// warm paints every angle of s not painted yet.
func (s *heldSet) warm() {
	for i := range heldTurns {
		s.mu.Lock()
		skip := s.made[i] || s.ready[i] != nil
		s.mu.Unlock()
		if skip {
			continue
		}
		px := bakeHeld(s.def, i)
		s.mu.Lock()
		if !s.made[i] {
			s.ready[i] = px
		}
		s.mu.Unlock()
	}
}

// at is turn i of s as an image, painted now if it has not been yet.
func (s *heldSet) at(i int) *heldFrame {
	if f := s.frame[i]; f != nil {
		return f
	}
	s.mu.Lock()
	px := s.ready[i]
	s.ready[i] = nil
	s.made[i] = true
	s.mu.Unlock()
	if px == nil {
		px = bakeHeld(s.def, i)
	}
	img := ebiten.NewImage(px.w, px.h)
	img.WritePixels(straightBytes(px.px))
	artBytes.Add(int64(4 * px.w * px.h))
	f := &heldFrame{img: img, gx: px.gx, gy: px.gy}
	s.frame[i] = f
	return f
}

// straightBytes is px premultiplied, as WritePixels takes it.
func straightBytes(px []color.RGBA) []byte {
	b := make([]byte, 4*len(px))
	for i, c := range px {
		c = icPM(c)
		b[4*i], b[4*i+1], b[4*i+2], b[4*i+3] = c.R, c.G, c.B, c.A
	}
	return b
}

// drawHeldDef draws d with its grip at (x, y), art pixels, turned th
// radians from upright towards the facing, mirrored for side -1, at
// opacity alpha.
func drawHeldDef(dst drawTarget, d *heldDef, x, y, th, side float64, alpha float32) {
	f := heldSetOf(d).at(turnIndex(th))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-f.gx/HDDensity, -f.gy/HDDensity) // drawImageD takes the image to art pixels first
	op.GeoM.Scale(side, 1)
	op.GeoM.Translate(x, y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(alpha)
	}
	drawImageD(dst, f.img, op, HDDensity)
}

// drawHeldHD draws item k as drawHeldDef does and reports true, or false
// if k has no held painter.
func drawHeldHD(dst drawTarget, k sim.ItemKind, x, y, th, side float64, alpha float32) bool {
	d := heldDefOf(k)
	if d == nil {
		return false
	}
	drawHeldDef(dst, d, x, y, th, side, alpha)
	return true
}

// heldPoint is where item point (u, v) of an item gripped at (x, y),
// turned th towards side, is: the flame on a torch, the tip of a rod.
func heldPoint(x, y, th, side, u, v float64) (float64, float64) {
	th = turnAngle(turnIndex(th)) // as drawn
	cs, sn := math.Cos(th), math.Sin(th)
	return x + side*(u*cs-v*sn), y + u*sn + v*cs
}

// --- the torch's flame -----------------------------------------------------------

// The flame on a held torch: torchFlameN frames of a flame licking up,
// painted at HD the first time a torch is shown. It stands upright
// whichever way the torch leans.
const torchFlameN = 8

// torchFlameBox is a flame's frame, art pixels: its root at (0, 0).
const (
	tfX0, tfY0, tfX1, tfY1 = -4.5, -11.0, 4.5, 3.0
)

var torchFlames [torchFlameN]*heldFrame

// torchFlame is flame frame i, painted on first use.
func torchFlame(i int) *heldFrame {
	if f := torchFlames[i]; f != nil {
		return f
	}
	const K = HDDensity
	const m = heldSS
	w, h := int(math.Ceil((tfX1-tfX0)*K)), int(math.Ceil((tfY1-tfY0)*K))
	n := max(w, h)
	p := &icv{n: n * m, k: K * m, px: make([]color.RGBA, n*n*m*m), part: make([]uint8, n*n*m*m)}
	ox, oy := -tfX0, -tfY0 // the root on the canvas, art pixels
	paintTorchFlame(p, ox, oy, float64(i)/torchFlameN)
	px := shrink(p.px, n*m, m)
	g := shrinkPM(p.glow, n*m, m)
	for j := range px {
		px[j] = icUnPM(overPM(g[j], icPM(px[j])))
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(straightBytes(px))
	artBytes.Add(int64(4 * n * n))
	f := &heldFrame{img: img, gx: ox * K, gy: oy * K}
	torchFlames[i] = f
	return f
}

// paintTorchFlame paints a flame rooted at (x, y), ph of the way through its
// flicker: tongues licking up from a round root, hottest low in the middle,
// a red rim, and a glow about it.
func paintTorchFlame(p *icv, x, y, ph float64) {
	a := ph * 2 * math.Pi
	sway := 0.45 * math.Sin(a)
	tall := 1 + 0.12*math.Sin(2*a+1)
	flame := iBlend(1.2,
		sEll(x, y-1.0, 2.2, 2.0),
		sTaper(1.9, 1.35, 0.06, x, y-1.2, x+sway*0.5, y-4.0*tall, x+sway-0.6, y-7.6*tall),
		sTaper(1.0, 0.7, 0.05, x+1.1, y-1.4, x+1.8+0.4*math.Sin(a+2), y-3.6, x+1.6+0.6*math.Sin(a+2), y-5.4*tall),
		sTaper(0.9, 0.6, 0.05, x-1.2, y-1.2, x-2.0+0.4*math.Sin(a+4), y-2.8, x-2.3+0.5*math.Sin(a+4), y-4.4))
	t := ph * 4
	lick := shape{flame.x0 - 0.8, flame.y0 - 0.8, flame.x1 + 0.8, flame.y1 + 0.8, func(px, py float64) float64 {
		up := clamp01((y - py) / 7)
		// Loopable noise: two samples of the cycle mixed.
		n := lerpf(fbm2(px*0.9, py*0.55+t, 7, 2), fbm2(px*0.9, py*0.55+t-4, 7, 2), ph)
		return flame.d(px, py) + (n-0.5)*1.5*up
	}}
	p.fill(lick, func(f *ifrag) color.RGBA {
		dx, dy := (f.x-x-sway*0.3)/1.9, (f.y-y+1.6)/3.2
		l := 1.1 - math.Hypot(dx, dy)*0.5 + (fbm2(f.x*0.8, f.y*0.5-t, 8, 2)-0.5)*0.25
		if -f.d < 0.35 {
			l = min(l, 0.3) // a red rim
		}
		return pick(iiFire, l)
	})
	p.bareLast()
	p.soft(sEll(x, y-2.6, 4.4, 5.6), func(f *ifrag) color.RGBA {
		d := math.Hypot((f.x-x)/4.4, (f.y-y+2.6)/5.6)
		return color.RGBA{0xff, 0xa8, 0x40, uint8(80 * clamp01(1-d) * clamp01(1-d))}
	})
}

// drawTorchFlame draws flame frame i rooted at (x, y), art pixels.
func drawTorchFlame(dst drawTarget, i int, x, y float64) {
	if fireHD { // the fire of the torch on its stand; see fireart.go
		drawFire(dst, fireTorch, i, x, y+0.5)
		return
	}
	i %= torchFlameN
	f := torchFlame(i % torchFlameN)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-f.gx/HDDensity, -f.gy/HDDensity)
	op.GeoM.Translate(x, y)
	drawImageD(dst, f.img, op, HDDensity)
}

// --- marks for what is drawn after the figures -------------------------------------

// heldBy is the player whose figure is being drawn on the world (not on a
// scratch image), so the flame of their torch and the tip of their rod can
// be noted for the light and the line, which are drawn after the figures.
var heldBy struct {
	on bool
	id sim.PlayerID
}

// heldMark is where a player's torch flame and rod tip were last drawn,
// art pixels on the world, and on which frame.
type heldMark struct {
	frame              int
	flame, tip         bool
	fx, fy, tipX, tipY float64
}

// markHeld notes, for the player being drawn, where something in their
// hand is.
func (g *Game) markHeld(set func(m *heldMark)) {
	if !heldBy.on {
		return
	}
	m := heldMarks[heldBy.id]
	if m == nil {
		m = &heldMark{}
		heldMarks[heldBy.id] = m
	}
	if m.frame != g.frame {
		*m = heldMark{frame: g.frame}
	}
	set(m)
}

// heldMarks holds the marks of every player drawn. Game thread only.
var heldMarks = map[sim.PlayerID]*heldMark{}

// heldMarkOf is what was noted of p's hand this frame, if anything.
func (g *Game) heldMarkOf(id sim.PlayerID) *heldMark {
	if m := heldMarks[id]; m != nil && m.frame == g.frame {
		return m
	}
	return nil
}

// --- icons in the world -----------------------------------------------------------

// iconsFine holds the old icons redrawn by iconFine. Game thread only.
var iconsFine = map[sim.ItemKind]*ebiten.Image{}

// iconFine is the old icon of k redrawn three times as fine (upscale3):
// for an item with no HD icon, shown in the world (in a hand, on the
// ground), where its 16 pixels would be blocks of the art-pixel grid.
// Nil if k has no icon. Game thread only: it reads the old icon back.
func (g *Game) iconFine(k sim.ItemKind) *ebiten.Image {
	if !k.Valid() || k >= sim.NumItems {
		return nil
	}
	if img, ok := iconsFine[k]; ok {
		return img
	}
	old := g.art.icons[k]
	if old == nil {
		iconsFine[k] = nil
		return nil
	}
	b := old.Bounds()
	w, h := b.Dx(), b.Dy()
	raw := make([]byte, 4*w*h)
	old.ReadPixels(raw)
	c := newCanvas(w, h)
	for i := range c.px {
		c.px[i] = icUnPM(color.RGBA{raw[4*i], raw[4*i+1], raw[4*i+2], raw[4*i+3]})
	}
	// Its outline: the opaque pixels on the edge, as canvas.outline drew them.
	c.ol = make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if c.px[y*w+x].A < 128 {
				continue
			}
			for _, d := range [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
				if !c.in(x+d[0], y+d[1]) || c.px[(y+d[1])*w+x+d[0]].A < 128 {
					c.ol[y*w+x] = true
					break
				}
			}
		}
	}
	u := upscale3(c)
	img := ebiten.NewImage(u.w, u.h)
	img.WritePixels(u.bytes())
	artBytes.Add(int64(4 * u.w * u.h))
	iconsFine[k] = img
	return img
}

// --- swing trails ---------------------------------------------------------------

// heldWhite is a white pixel to colour triangles with.
var heldWhite *ebiten.Image

func heldWhitePx() *ebiten.Image {
	if heldWhite == nil {
		img := ebiten.NewImage(3, 3)
		img.Fill(colWhite)
		heldWhite = img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	}
	return heldWhite
}

// drawTrianglesFine draws vs, in art pixels, at the full resolution of dst
// (the world's canvas, a fineBuf's image), coloured by their vertices.
func drawTrianglesFine(dst drawTarget, vs []ebiten.Vertex, is []uint16) {
	var target *ebiten.Image
	k := float32(1)
	switch d := dst.(type) {
	case *worldDst:
		d.flush()
		target, k = d.canvas, float32(d.r)
	case *fineBuf:
		target, k = d.img, HDDensity
	case *ebiten.Image:
		target = d
	default:
		return
	}
	for i := range vs {
		vs[i].DstX *= k
		vs[i].DstY *= k
		vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
	}
	target.DrawTriangles(vs, is, heldWhitePx(), &ebiten.DrawTrianglesOptions{AntiAlias: true, ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha})
}

// swingArc draws the trail an edge leaves as it swings about the grip
// (gx, gy): the stretch swept from angle a back tail radians (going way,
// +1 or -1, and never past from), between radius r0 and r1, fading along;
// fade scales it all. Angles are from upright towards the facing (side), as
// the held items turn. It is drawn as pixelfx's melee/sword-slash (see
// fx.go): a crescent thickest just behind the edge and thinning to its
// tail, in flat bands, the tool's tone inside, a pale band, a white edge.
func swingArc(dst drawTarget, gx, gy, side, a, from, way, tail, r0, r1 float64, c color.RGBA, fade float64) {
	if fade <= 0 || tail <= 0 {
		return
	}
	const steps = 30
	type bnd struct {
		u0, u1, a float64
		c         color.RGBA
	}
	bands := [...]bnd{{0, 0.5, 0.42, c}, {0.5, 0.84, 0.78, mix(c, colWhite, 0.55)}, {0.84, 1, 0.95, colWhite}}
	var b fxBatch
	for _, bd := range bands {
		var p uint16
		for i := 0; i <= steps; i++ {
			s := float64(i) / steps
			ang := a - way*tail*s
			last := false
			if way*(ang-from) < 0 {
				ang, last = from, true
			}
			inner := r0 + (r1-r0)*(0.05+0.85*math.Pow(s, 0.8)) // thinning to the tail
			ru0, ru1 := lerpf(inner, r1, bd.u0), lerpf(inner, r1, bd.u1)
			al := bd.a * math.Pow(1-s, 1.2) * fade
			sn, cs := math.Sin(ang), math.Cos(ang)
			q := b.vert(gx+sn*ru0*side, gy-cs*ru0, bd.c, al, 0.2)
			b.vert(gx+sn*ru1*side, gy-cs*ru1, bd.c, al, 0.2)
			if i > 0 {
				b.is = append(b.is, p, p+1, q, p+1, q+1, q)
			}
			p = q
			if last {
				break
			}
		}
	}
	b.flush(dst)
}

// dropPx is the size, image pixels, of the HD icon a pile on the ground is
// drawn with: 13 art pixels, about the old icon at 0.8.
const dropPx = 13 * HDDensity

// drawDropHD draws pile d with its icon's bottom middle about (x, y), art
// pixels, as drawDrop places the old icon: the HD icon if the item has one,
// else the old icon redrawn finer (iconFine), and a second one behind for a
// stack. It reports false, drawing nothing, with the HD items off.
func (g *Game) drawDropHD(dst drawTarget, d protocol.DropView, x, y float64) bool {
	if !heldHD {
		return false
	}
	img, sc := hdIcon(d.Item, dropPx), 1.0
	if img == nil {
		if img = g.iconFine(d.Item); img == nil {
			return false
		}
		sc = float64(dropPx) / float64(img.Bounds().Dx())
	}
	put := func(dx, dy float64, dim float32) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(x-6.5+dx, y-11.3+dy)
		if sc != 1 {
			op.Filter = ebiten.FilterLinear
		}
		if dim < 1 {
			op.ColorScale.Scale(dim, dim, dim, 1)
		}
		drawImageD(dst, img, op, HDDensity)
	}
	if d.Count > 1 { // a little stack: a second icon behind, offset
		put(2, -1, 0.7)
	}
	put(0, 0, 1)
	return true
}

// fineStrip draws a line w art pixels thick through pts (x, y pairs, art
// pixels) at the full resolution of dst, in c (straight alpha): a fishing
// line from an HD rod.
func fineStrip(dst drawTarget, pts []float64, w float64, c color.RGBA) {
	n := len(pts) / 2
	if n < 2 {
		return
	}
	a := float32(c.A) / 255
	col := [4]float32{float32(c.R) / 255 * a, float32(c.G) / 255 * a, float32(c.B) / 255 * a, a}
	vs := make([]ebiten.Vertex, 0, 2*n)
	is := make([]uint16, 0, 6*(n-1))
	for i := 0; i < n; i++ {
		j, k := max(0, i-1), min(n-1, i+1)
		dx, dy := pts[2*k]-pts[2*j], pts[2*k+1]-pts[2*j+1]
		l := math.Hypot(dx, dy)
		if l == 0 {
			l = 1
		}
		ox, oy := -dy/l*w/2, dx/l*w/2
		for _, s := range [2]float64{-1, 1} {
			vs = append(vs, ebiten.Vertex{DstX: float32(pts[2*i] + s*ox), DstY: float32(pts[2*i+1] + s*oy),
				ColorR: col[0], ColorG: col[1], ColorB: col[2], ColorA: col[3]})
		}
		if i > 0 {
			p := uint16(2 * (i - 1))
			is = append(is, p, p+1, p+2, p+1, p+3, p+2)
		}
	}
	drawTrianglesFine(dst, vs, is)
}
