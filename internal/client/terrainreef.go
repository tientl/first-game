package client

import (
	"image/color"
	"math"
	"slices"

	"survivalgame/internal/sim"
)

// Coral reefs, as the ground bake draws them under the shallows (see
// reliefSea in terrainsea.go). A reef is not strewn with specks: its coral
// grows in formations, a cluster every couple of tiles where the reef is,
// with sand, sea grass and rock between them. Each cluster keeps to one
// palette, two or three related hues (coral reds with orange and cream,
// purples with pink and lavender, olive and lime with teal, amber and
// mustard, blues, rust and bone), and is made of a few heads of different
// kinds:
//
//   - brain coral, a dome with winding grooves;
//   - staghorn, branches fanning up from a base, pale at their tips;
//   - table coral, a flat plate on a short stem, standing over its shadow;
//   - sea fans, upright lattices you see the sand through;
//   - soft coral, a clump of round lobes dotted with polyps;
//
// and rocks of the reef among them. Every head is lit from the upper left
// and throws a soft shadow down to the right; then the whole is blurred a
// little and seen through the water: bluer and fainter the deeper it lies.
// The heads of a chunk (and of a margin round it, for those that reach in)
// are laid out before its pixels are coloured, in a layer of their own
// (reefField), the nearer drawn over the farther. The light rippling over
// them and the small fish darting among them are drawn by the water shader
// (terrainanim.go).

const (
	reefCellW  = 132 // pixels: the grid the clusters are laid on, about two tiles
	reefCellH  = 100
	reefSmallW = 46 // and the finer grid of the rocks and lone heads between them
	reefSmallH = 34
	reefMargin = 72 // pixels round a chunk whose heads may reach into it
)

// coralPal is a cluster's palette: three ramps of related hues, dark to
// light, the first for the big heads.
type coralPal [3]ramp

var coralPals = []coralPal{
	{ // coral red, orange, cream
		mkRamp(0x5c1f2a, 0x8a2f3a, 0xb84a4c, 0xdb6d5e, 0xf0977a, 0xf9c4a4),
		mkRamp(0x6a2c16, 0x9a4a1f, 0xc9702c, 0xe8963e, 0xf6ba62, 0xfde09a),
		mkRamp(0x6e5a4a, 0x9c8470, 0xc4ab92, 0xe0cab0, 0xf2e2cc, 0xfdf6ea),
	},
	{ // purple, magenta, lavender
		mkRamp(0x2e1a4a, 0x4a2a72, 0x6a3f98, 0x8d5cbc, 0xb187d8, 0xd6b8f0),
		mkRamp(0x4e1840, 0x7a2862, 0xa8448a, 0xcc6aac, 0xe698c8, 0xf6c8e4),
		mkRamp(0x2c2c5a, 0x434888, 0x6068b0, 0x8590d0, 0xaeb8e8, 0xd6defa),
	},
	{ // olive, lime, teal
		mkRamp(0x23361c, 0x3a5528, 0x557636, 0x759a48, 0x9cbe62, 0xc6de8e),
		mkRamp(0x3e4a14, 0x66761e, 0x92a42c, 0xbccc46, 0xdce872, 0xf2f8b0),
		mkRamp(0x123c3c, 0x1d5e5a, 0x2b827a, 0x45a698, 0x6ec8b6, 0xa6e6d4),
	},
	{ // amber, mustard, tan
		mkRamp(0x4a2a0e, 0x744216, 0xa0621e, 0xc88a2c, 0xe6b44a, 0xf8dc86),
		mkRamp(0x4a3e12, 0x76641c, 0xa48c28, 0xccb43a, 0xe6d462, 0xf8f0a6),
		mkRamp(0x4e3c2c, 0x76604a, 0x9e866a, 0xc4ac8c, 0xe2ceb0, 0xf6ead6),
	},
	{ // blue, aqua, pale violet
		mkRamp(0x142a4e, 0x1e4478, 0x2c62a2, 0x4686c4, 0x6eaade, 0xa4d0f2),
		mkRamp(0x103e4a, 0x1a6070, 0x288896, 0x40aeb8, 0x6ccfd4, 0xaaeeee),
		mkRamp(0x34305a, 0x504a84, 0x726caa, 0x9692c8, 0xbcbae0, 0xe0e0f6),
	},
	{ // rust, dusty pink, bone
		mkRamp(0x3c1810, 0x62281a, 0x8c3c24, 0xb45a34, 0xd47e4e, 0xeeaa7a),
		mkRamp(0x52303a, 0x7c4a56, 0xa66a76, 0xc88e98, 0xe2b4ba, 0xf6dcde),
		mkRamp(0x5a5446, 0x857c68, 0xaca28a, 0xccc4aa, 0xe4dec8, 0xf6f2e6),
	},
}

// reefRock is the reef's own rock, green-grey under the water.
var reefRock = mkRamp(0x26302e, 0x3a4644, 0x52605a, 0x6c7a70, 0x8a9688, 0xaab2a2)

// Kinds of head.
const (
	headBrain = iota
	headStag
	headTable
	headFan
	headSoft
	headRock
)

// coralHead is one head of coral (or a rock) of a reef.
type coralHead struct {
	kind   int
	x, y   float64 // its foot, in image pixels of the world
	size   float64 // pixels: its half-width, about
	rp     ramp
	seed   uint32
	sx, sy float64 // its shadow's middle
}

// reefField is the coral round a chunk being baked: its colour (straight,
// alpha how much of the pixel it covers) and how deep the shadow of the
// heads lies on the sand.
type reefField struct {
	col   []color.RGBA
	shade []uint8
}

// reefTile reports whether tile (tx, ty) is a reef.
func (t *terrain) reefTile(tx, ty int) bool {
	return t.tile(tx, ty) == sim.TileShallow && t.rel(tx, ty).Variant() == sim.VarReef
}

// reefShare is how much of sea point (u, v) is reef, through the same warp
// as the shallows' shapes.
func (t *terrain) reefShare(u, v float64) float64 {
	wu, wv := seaWarp(u, v)
	return cornersAt(wu, wv).share(t.reefTile)
}

// reefs lays out the coral round the chunk, or returns nil if no reef is
// near.
func (b *bake) reefs() *reefField {
	t := b.t
	tx0, ty0 := fdiv(b.x0-reefMargin, hdTW)-1, fdiv(b.y0-reefMargin, hdTH)-1
	tx1, ty1 := fdiv(b.x0+chunkPW+reefMargin, hdTW)+1, fdiv(b.y0+chunkPH+reefMargin, hdTH)+1
	any := false
	for ty := ty0; ty <= ty1 && !any; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			if t.reefTile(tx, ty) {
				any = true
				break
			}
		}
	}
	if !any {
		return nil
	}
	var heads []coralHead
	// The clusters.
	for cy := fdiv(b.y0-reefMargin, reefCellH); cy <= fdiv(b.y0+chunkPH+reefMargin, reefCellH); cy++ {
		for cx := fdiv(b.x0-reefMargin, reefCellW); cx <= fdiv(b.x0+chunkPW+reefMargin, reefCellW); cx++ {
			heads = t.reefCluster(heads, cx, cy)
		}
	}
	// Rocks and lone small heads between them.
	for cy := fdiv(b.y0-reefMargin, reefSmallH); cy <= fdiv(b.y0+chunkPH+reefMargin, reefSmallH); cy++ {
		for cx := fdiv(b.x0-reefMargin, reefSmallW); cx <= fdiv(b.x0+chunkPW+reefMargin, reefSmallW); cx++ {
			h := hash2(int32(cx), int32(cy), 1510)
			if float64(h&0xffff)/65536 > 0.11 {
				continue
			}
			x := float64(cx*reefSmallW) + 6 + float64(h>>16&0xff)/255*(reefSmallW-12)
			y := float64(cy*reefSmallH) + 6 + float64(h>>24)/255*(reefSmallH-12)
			u, v := x/hdTW, y/hdTH
			if t.reefShare(u, v) < 0.5 || t.seaDepthAt(u, v) > 0.6 {
				continue
			}
			hd := coralHead{kind: headRock, x: x, y: y, size: 4 + 4*hashf(cx, cy, 1511), rp: reefRock, seed: h}
			if hashf(cx, cy, 1512) < 0.6 {
				pal := coralPals[int(hashf(fdiv(cx, 3), fdiv(cy, 3), 1513)*float64(len(coralPals)))]
				hd.kind = []int{headBrain, headSoft, headStag}[h>>8%3]
				hd.rp = pal[1+int(h>>11&1)]
				hd.size = 5 + 5*hashf(cx, cy, 1514)
			}
			heads = append(heads, hd)
		}
	}
	if len(heads) == 0 {
		return nil
	}
	slices.SortStableFunc(heads, func(a, b coralHead) int {
		switch {
		case a.y < b.y:
			return -1
		case a.y > b.y:
			return 1
		}
		return 0
	})
	f := &reefField{col: make([]color.RGBA, chunkPW*chunkPH), shade: make([]uint8, chunkPW*chunkPH)}
	for i := range heads {
		f.shadow(b, &heads[i])
	}
	for i := range heads {
		f.draw(b, &heads[i])
	}
	f.soften()
	return f
}

// reefCluster adds the heads of the cluster of grid cell (cx, cy), if one
// grows there.
func (t *terrain) reefCluster(heads []coralHead, cx, cy int) []coralHead {
	h := hash2(int32(cx), int32(cy), 1500)
	x := (float64(cx) + 0.18 + 0.64*float64(h&0xff)/255) * reefCellW
	y := (float64(cy) + 0.18 + 0.64*float64(h>>8&0xff)/255) * reefCellH
	u, v := x/hdTW, y/hdTH
	share := t.reefShare(u, v)
	if share < 0.25 || float64(h>>16&0xff)/255 > 1.1*share {
		return heads
	}
	// Open sand here and there across a reef.
	if vnoise(x/260, y/200, 1501) < 0.22 {
		return heads
	}
	if t.seaDepthAt(u, v) > 0.62 {
		return heads
	}
	pal := coralPals[int(hashf(cx, cy, 1502)*float64(len(coralPals)))]
	r := 30 + 26*hashf(cx, cy, 1503) // the cluster's reach
	n := 5 + int(hashf(cx, cy, 1504)*6.99)
	for i := 0; i < n; i++ {
		g := hash2(int32(cx*16+i), int32(cy), 1505)
		a := float64(g&0xffff) / 65536 * 2 * math.Pi
		d := r * math.Sqrt(float64(g>>16&0xff)/255) * 0.9
		if i == 0 {
			d = 0 // the biggest head in the middle
		}
		hd := coralHead{x: x + d*math.Cos(a), y: y + 0.72*d*math.Sin(a), seed: g}
		big := 1 - 0.45*d/r
		k := float64(g>>24) / 255
		switch {
		case k < 0.28:
			hd.kind, hd.rp, hd.size = headBrain, pal[[4]int{0, 0, 0, 2}[g>>9&3]], (11+11*hashf(i, cx, 1506))*big
		case k < 0.5:
			hd.kind, hd.rp, hd.size = headStag, pal[[2]int{1, 0}[g>>9&1]], (13+11*hashf(i, cx, 1506))*big
		case k < 0.66:
			hd.kind, hd.rp, hd.size = headTable, pal[[2]int{0, 2}[g>>9&1]], (15+12*hashf(i, cx, 1506))*big
		case k < 0.8:
			hd.kind, hd.rp, hd.size = headFan, pal[1], (11+8*hashf(i, cx, 1506))*big
		case k < 0.93:
			hd.kind, hd.rp, hd.size = headSoft, pal[[2]int{1, 2}[g>>9&1]], (9+7*hashf(i, cx, 1506))*big
		default:
			hd.kind, hd.rp, hd.size = headRock, reefRock, (7+6*hashf(i, cx, 1506))*big
		}
		hd.size = max(hd.size, 5)
		heads = append(heads, hd)
	}
	return heads
}

// put lays colour c over chunk pixel (x, y) of the layer, covering a of it.
func (f *reefField) put(x, y int, c color.RGBA, a float64) {
	if x < 0 || y < 0 || x >= chunkPW || y >= chunkPH || a <= 0 {
		return
	}
	i := y*chunkPW + x
	o := f.col[i]
	a = clamp01(a)
	oa := float64(o.A) / 255
	na := a + oa*(1-a)
	if na <= 0 {
		return
	}
	mixc := func(n, old uint8) uint8 {
		return uint8(math.Round((float64(n)*a + float64(old)*oa*(1-a)) / na))
	}
	f.col[i] = color.RGBA{mixc(c.R, o.R), mixc(c.G, o.G), mixc(c.B, o.B), uint8(math.Round(na * 255))}
}

// box is the chunk pixels round (x, y) of the world, rx by ry either side.
func (b *bake) box(x, y, rx, ry float64) (x0, y0, x1, y1 int) {
	x0 = max(0, int(math.Floor(x-rx))-b.x0)
	x1 = min(chunkPW-1, int(math.Ceil(x+rx))-b.x0)
	y0 = max(0, int(math.Floor(y-ry))-b.y0)
	y1 = min(chunkPH-1, int(math.Ceil(y+ry))-b.y0)
	return
}

// shadow lays head h's shadow on the sand: soft, down and to the right.
func (f *reefField) shadow(b *bake, h *coralHead) {
	s := h.size
	rx, ry := s*1.05, s*0.55
	cx, cy := h.x+s*0.35, h.y+s*0.12
	if h.kind == headTable {
		cx, cy, rx, ry = h.x+s*0.3, h.y+2, s*1.1, s*0.5
	}
	x0, y0, x1, y1 := b.box(cx, cy, rx, ry)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := (float64(b.x0+x) + 0.5 - cx) / rx
			dy := (float64(b.y0+y) + 0.5 - cy) / ry
			if r := dx*dx + dy*dy; r < 1 {
				k := uint8((1 - r) * (1 - r) * 190)
				i := y*chunkPW + x
				f.shade[i] = max(f.shade[i], k)
			}
		}
	}
}

// reefLight is the light: up and to the left and towards the viewer.
var reefLight = func() [3]float64 {
	x, y, z := -0.5, -0.62, 0.6
	n := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / n, y / n, z / n}
}()

// lambert is how lit a surface facing (nx, ny, nz) is, 0 to 1.
func lambert(nx, ny, nz float64) float64 {
	n := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if n == 0 {
		return 0.5
	}
	return clamp01((nx*reefLight[0] + ny*reefLight[1] + nz*reefLight[2]) / n)
}

// draw paints head h into the layer.
func (f *reefField) draw(b *bake, h *coralHead) {
	switch h.kind {
	case headBrain:
		f.dome(b, h, true)
	case headRock:
		f.dome(b, h, false)
	case headStag:
		f.staghorn(b, h)
	case headTable:
		f.table(b, h)
	case headFan:
		f.fan(b, h)
	case headSoft:
		f.soft(b, h)
	}
}

// dome is a brain coral, its winding grooves lit and shaded, or a rock of
// the reef, rounded and speckled.
func (f *reefField) dome(b *bake, h *coralHead, brain bool) {
	rx := h.size
	ry := rx * 0.8
	cx, cy := h.x, h.y-ry*0.55 // it sits on the sand at its foot
	x0, y0, x1, y1 := b.box(cx, cy, rx+1, ry+1)
	sd := float64(h.seed & 0xff)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			wx, wy := float64(b.x0+x)+0.5, float64(b.y0+y)+0.5
			dx, dy := (wx-cx)/rx, (wy-cy)/ry
			// A lumpy edge.
			ang := math.Atan2(dy, dx)
			lump := 1 + 0.06*math.Sin(3*ang+sd) + 0.04*math.Sin(5*ang+sd*1.7)
			r2 := (dx*dx + dy*dy) / (lump * lump)
			if r2 >= 1 {
				continue
			}
			nz := math.Sqrt(1 - r2)
			l := lambert(dx, dy, nz*1.1)
			t := 0.12 + 0.82*l
			if !brain {
				t = 0.06 + 0.62*l // rock: duller than the living coral
			}
			if brain {
				// The grooves wind over it.
				g := math.Sin((wx*0.85+wy*0.45)/1.9 + 7*vnoise(wx/7+sd, wy/6, 1520) + 4*vnoise(wx/3.5, wy/3, 1521))
				switch {
				case g > 0.62:
					t -= 0.3
				case g > 0.3:
					t -= 0.1
				case g < -0.75:
					t += 0.08 // the ridge between, catching the light
				}
			} else if hashf(fdiv(b.x0+x, 2), fdiv(b.y0+y, 2), 1522) < 0.12 {
				t -= 0.18 // pitted
			}
			if r2 > 0.78 && dx+dy > 0 {
				t -= 0.12 // its shaded rim
			}
			a := clamp01((1 - r2) * rx * 0.45) // a soft edge
			f.put(x, y, h.rp.pick(t, b.x0+x, b.y0+y), a)
		}
	}
}

// staghorn is a fan of branches from a base, each a rod lit along its
// upper left, pale at its tip.
func (f *reefField) staghorn(b *bake, h *coralHead) {
	type rod struct{ x0, y0, x1, y1, w float64 }
	var rods []rod
	n := 5 + int(h.seed>>4%5)
	for i := 0; i < n; i++ {
		g := hash2(int32(h.seed), int32(i), 1530)
		a := -math.Pi/2 + (float64(i)/float64(n-1)-0.5)*2.4 + (float64(g&0xff)/255-0.5)*0.35
		l := h.size * (0.75 + 0.45*float64(g>>8&0xff)/255)
		bx, by := h.x+(float64(g>>16&0xff)/255-0.5)*h.size*0.5, h.y
		ex, ey := bx+math.Cos(a)*l, by+math.Sin(a)*l*0.85
		rods = append(rods, rod{bx, by, ex, ey, 2.1})
		if g>>24 < 150 { // a fork near the tip
			mx, my := bx+(ex-bx)*0.6, by+(ey-by)*0.6
			fa := a + 0.55*float64(int(g>>24&1)*2-1)
			rods = append(rods, rod{mx, my, mx + math.Cos(fa)*l*0.45, my + math.Sin(fa)*l*0.4, 1.6})
		}
	}
	// The farther (higher) tips first.
	slices.SortFunc(rods, func(a, b rod) int {
		switch {
		case math.Min(a.y0, a.y1) < math.Min(b.y0, b.y1):
			return -1
		case math.Min(a.y0, a.y1) > math.Min(b.y0, b.y1):
			return 1
		}
		return 0
	})
	for _, r := range rods {
		dx, dy := r.x1-r.x0, r.y1-r.y0
		ln := math.Hypot(dx, dy)
		if ln == 0 {
			continue
		}
		ux, uy := dx/ln, dy/ln
		x0, y0, x1, y1 := b.box((r.x0+r.x1)/2, (r.y0+r.y1)/2, math.Abs(dx)/2+r.w+1, math.Abs(dy)/2+r.w+1)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				px, py := float64(b.x0+x)+0.5-r.x0, float64(b.y0+y)+0.5-r.y0
				along := clamp01((px*ux + py*uy) / ln)
				qx, qy := px-ux*along*ln, py-uy*along*ln
				w := r.w * (1.15 - 0.35*along)
				d := math.Hypot(qx, qy)
				if d >= w {
					continue
				}
				// Across the rod: which side faces the light.
				side := (qx*reefLight[0] + qy*reefLight[1]) / w
				t := 0.4 + 0.45*side + 0.25*along
				if along > 0.8 {
					t += 0.25 * (along - 0.8) / 0.2 // the pale growing tip
				}
				if d > w*0.75 && side < 0 {
					t -= 0.12
				}
				f.put(x, y, h.rp.pick(t, b.x0+x, b.y0+y), clamp01((w-d)*1.2))
			}
		}
	}
}

// table is a plate coral: a flat disc on a short stem, lit on top, its
// thick edge in shade under the rim.
func (f *reefField) table(b *bake, h *coralHead) {
	rx := h.size
	ry := rx * 0.42
	cx, cy := h.x, h.y-5
	x0, y0, x1, y1 := b.box(cx, cy+2, rx+1, ry+4)
	sd := float64(h.seed & 0xff)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			wx, wy := float64(b.x0+x)+0.5, float64(b.y0+y)+0.5
			ang := math.Atan2((wy-cy)/ry, (wx-cx)/rx)
			lump := 1 + 0.07*math.Sin(5*ang+sd) + 0.03*math.Sin(9*ang+sd)
			dx, dy := (wx-cx)/rx/lump, (wy-cy)/ry/lump
			r2 := dx*dx + dy*dy
			if r2 < 1 {
				// The top: flat, faint rings of growth, its rim lit.
				r := math.Sqrt(r2)
				t := 0.58 - 0.12*dx - 0.1*dy
				if math.Sin(r*rx/1.7+sd) > 0.6 {
					t -= 0.08
				}
				if r > 0.86 {
					if dx+dy < 0 {
						t = 0.95
					} else {
						t -= 0.12
					}
				}
				f.put(x, y, h.rp.pick(t, b.x0+x, b.y0+y), clamp01((1-r2)*rx*0.5))
				continue
			}
			// Its edge, seen below the rim: the plate's thickness, dark.
			ey := (wy - 2.2 - cy) / ry / lump
			if dx*dx+ey*ey < 1 && dy > 0 {
				f.put(x, y, h.rp.at(1), 1)
				continue
			}
			// The stem under the middle.
			if math.Abs(wx-cx) < rx*0.16 && wy > cy && wy < h.y {
				f.put(x, y, h.rp.at(0), 1)
			}
		}
	}
}

// fan is a sea fan: an upright lattice on a stalk, the sand showing
// through its mesh.
func (f *reefField) fan(b *bake, h *coralHead) {
	w, ht := h.size, h.size*1.25
	bx, by := h.x, h.y
	cy := by - ht*0.55
	x0, y0, x1, y1 := b.box(bx, cy, w+1, ht*0.6+1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			wx, wy := float64(b.x0+x)+0.5, float64(b.y0+y)+0.5
			dx, dy := (wx-bx)/w, (wy-cy)/(ht*0.55)
			r2 := dx*dx + dy*dy
			if r2 >= 1 || wy > by-1 {
				if math.Abs(wx-bx) < 0.9 && wy >= by-ht*0.2 && wy <= by { // the stalk
					f.put(x, y, h.rp.at(1), 1)
				}
				continue
			}
			// Veins running out from the stalk, and across them.
			ang := math.Atan2(wy-by, wx-bx)
			rd := math.Hypot(wx-bx, (wy-by)*0.9)
			vein := math.Abs(math.Sin(ang*9+0.3*math.Sin(rd*0.4))) < 0.22
			cross := math.Abs(math.Sin(rd*0.95+ang*2)) < 0.25
			t := 0.55 - 0.25*dx - 0.15*dy
			switch {
			case r2 > 0.85:
				f.put(x, y, h.rp.pick(t-0.1, b.x0+x, b.y0+y), clamp01((1-r2)*w*0.6)) // its edge, thick
			case vein:
				f.put(x, y, h.rp.pick(t+0.1, b.x0+x, b.y0+y), 0.95)
			case cross:
				f.put(x, y, h.rp.pick(t, b.x0+x, b.y0+y), 0.8)
			default:
				f.put(x, y, h.rp.pick(t-0.15, b.x0+x, b.y0+y), 0.28) // the mesh
			}
		}
	}
}

// soft is a soft coral: a clump of round lobes, each lit, dotted with
// pale polyps.
func (f *reefField) soft(b *bake, h *coralHead) {
	n := 4 + int(h.seed>>5%4)
	type lobe struct{ x, y, r float64 }
	lobes := make([]lobe, 0, n)
	for i := 0; i < n; i++ {
		g := hash2(int32(h.seed), int32(i), 1540)
		a := float64(g&0xffff) / 65536 * 2 * math.Pi
		d := h.size * 0.55 * math.Sqrt(float64(g>>16&0xff)/255)
		r := h.size * (0.35 + 0.25*float64(g>>24)/255)
		lobes = append(lobes, lobe{h.x + d*math.Cos(a), h.y - r*0.8 + 0.6*d*math.Sin(a), r})
	}
	slices.SortFunc(lobes, func(a, b lobe) int {
		switch {
		case a.y < b.y:
			return -1
		case a.y > b.y:
			return 1
		}
		return 0
	})
	for _, l := range lobes {
		x0, y0, x1, y1 := b.box(l.x, l.y, l.r+1, l.r+1)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				wx, wy := float64(b.x0+x)+0.5, float64(b.y0+y)+0.5
				dx, dy := (wx-l.x)/l.r, (wy-l.y)/(l.r*0.9)
				r2 := dx*dx + dy*dy
				if r2 >= 1 {
					continue
				}
				t := 0.15 + 0.8*lambert(dx, dy, math.Sqrt(1-r2))
				if hashf(b.x0+x, b.y0+y, 1541) < 0.09 && r2 < 0.7 {
					t += 0.35 // a polyp
				}
				f.put(x, y, h.rp.pick(t, b.x0+x, b.y0+y), clamp01((1-r2)*l.r*0.5))
			}
		}
	}
}

// soften blurs the coral a little: it is seen through the water.
func (f *reefField) soften() {
	out := make([]color.RGBA, len(f.col))
	for y := 0; y < chunkPH; y++ {
		for x := 0; x < chunkPW; x++ {
			var r, g, bl, a float64
			for _, d := range [5][3]int{{0, 0, 4}, {1, 0, 1}, {-1, 0, 1}, {0, 1, 1}, {0, -1, 1}} {
				xx, yy := min(chunkPW-1, max(0, x+d[0])), min(chunkPH-1, max(0, y+d[1]))
				c := f.col[yy*chunkPW+xx]
				w := float64(d[2]) * float64(c.A) / 255
				r += float64(c.R) * w
				g += float64(c.G) * w
				bl += float64(c.B) * w
				a += w
			}
			if a <= 0 {
				continue
			}
			out[y*chunkPW+x] = color.RGBA{uint8(r / a), uint8(g / a), uint8(bl / a), uint8(math.Round(a / 8 * 255))}
		}
	}
	f.col = out
}

// reefPixel lays the reef over sea pixel c at chunk pixel (x, y), d deep,
// rf (0 to 1) how much of it is reef: the shadow of the heads on the sand,
// sea grass and rubble between them, and the coral, tinted by the water
// over it.
func (b *bake) reefPixel(c color.RGBA, x, y, hx, hy int, u, v, d, rf float64) color.RGBA {
	f := b.reef
	water := withAlpha(c, 255)
	// The floor between the heads: patches of pale sand, beds of sea grass,
	// a little rubble.
	if rf > 0.02 {
		if sp := fbm2(u*0.42+11, v*0.5, 1555, 2); sp > 0.55 {
			c = mix(c, withAlpha(shoalSand, c.A), 0.22*rf*smooth(clamp01((sp-0.55)/0.12)))
		}
		if g := fbm2(u*0.5+4, v*0.6, 1550, 2); g > 0.56 {
			k := smooth(clamp01((g - 0.56) / 0.1))
			c = mix(c, withAlpha(seaGrass, c.A), 0.12*k*rf)
			if bl, ok := grassBlade(hx, hy, k); ok {
				c = mix(c, withAlpha(mix(seaGrass, hex(0x7fbf7a), bl), c.A), 0.55*rf)
			}
		} else if p, l := pebble(hx, hy, 11, 9, 1554, 0.07*rf, 0.8, 1.8); p == pebStone {
			c = mix(c, withAlpha(reefRock.at(1+int(l*3)), c.A), 0.45) // rubble
		} else if p == pebShadow {
			c = scaleCol(c, 0.95)
		}
	}
	if f == nil {
		return c
	}
	i := y*chunkPW + x
	if s := f.shade[i]; s > 0 {
		c = mix(c, withAlpha(hex(0x0f3a4a), c.A), float64(s)/255*0.42)
	}
	k := f.col[i]
	if k.A == 0 {
		return c
	}
	// Seen through the water: bluer and fainter the deeper it lies.
	tint := clamp01(0.1 + 0.7*(d-0.3))
	cc := mix(withAlpha(k, 255), water, tint)
	a := float64(k.A) / 255
	out := mix(c, cc, a)
	out.A = uint8(max(float64(c.A), lerpf(float64(c.A), 238, a)))
	return out
}

// grassBlade reports whether (hx, hy) is on a blade of sea grass in a bed
// dens (0 to 1) thick, and how far up the blade it is (0 at its root, 1 at
// its tip): blades rooted at random in cells of a few pixels, each leaning
// its own way, the taller ones further apart.
func grassBlade(hx, hy int, dens float64) (float64, bool) {
	const cw, ch = 4, 6
	cx, cy := fdiv(hx, cw), fdiv(hy, ch)
	for oy := 0; oy <= 2; oy++ { // a blade rises into the cells above its root
		for ox := -1; ox <= 1; ox++ {
			h := hash2(int32(cx+ox), int32(cy+oy), 1556)
			if float64(h&0xff)/255 >= 0.7*dens {
				continue
			}
			bx := float64((cx+ox)*cw) + float64(h>>8&0xff)/255*cw
			by := float64((cy+oy)*ch) + float64(h>>16&0xff)/255*ch
			l := 5 + 8*float64(h>>24&0x7f)/127
			lean := (float64(h>>12&0xf)/15 - 0.5) * 0.7
			up := by - (float64(hy) + 0.5)
			if up < 0 || up > l {
				continue
			}
			f := up / l
			x := bx + lean*up + 0.8*f*f*lean*l*0.3
			if math.Abs(float64(hx)+0.5-x) < 0.75-0.3*f {
				return f, true
			}
		}
	}
	return 0, false
}

// near reports whether coral lies within a few pixels of chunk pixel (x,
// y): where the small fish keep (terrainanim.go).
func (f *reefField) near(x, y int) bool {
	for dy := -24; dy <= 24; dy += 12 {
		for dx := -30; dx <= 30; dx += 12 {
			xx, yy := x+dx, y+dy
			if xx >= 0 && yy >= 0 && xx < chunkPW && yy < chunkPH && f.col[yy*chunkPW+xx].A > 80 {
				return true
			}
		}
	}
	return false
}
