package client

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The relief of the newest worlds on the world map (M) and the minimap:
// clean blue rivers and lakes, cliffs as dark contour lines along their low
// side, ramps as paths, the shallows of the sea in their own clear colours.
// The minimap is cut from pieces baked at exactly its own scale, so it is
// drawn one map pixel to one minimap pixel, crisp, rather than the whole
// map scaled up.

var (
	mapRiver    = hex(0x3d8fd0)
	mapRiverLit = hex(0x67b0e0)
	mapCliff    = hex(0x2a211d)
	mapFall     = hex(0xdff2f8)
	mapLake     = hex(0x2f78b4)
	mapCoral    = hex(0xd98e9a)
	mapLagoon   = hex(0x72d6cc)
)

// mapRelief lays rivers, cliffs, ramps and the levels over map land pixel
// c at (u, v).
func (t *terrain) mapRelief(c color.RGBA, u, v float64) color.RGBA {
	if t.relief == nil {
		return c
	}
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	fx, fy := u-float64(tx), v-float64(ty)
	k := t.tile(tx, ty)
	switch {
	case k == sim.TileCliff:
		// A dark line along the side the ground drops to; the rest of it is
		// the high ground's.
		l := t.rel(tx, ty).Level()
		low := func(dx, dy int) bool {
			n := t.tile(tx+dx, ty+dy)
			return n.Sea() || n != sim.TileCliff && t.rel(tx+dx, ty+dy).Level() < l
		}
		if low(0, 1) && fy >= 0.45 || low(0, -1) && fy <= 0.55 || low(1, 0) && fx >= 0.45 || low(-1, 0) && fx <= 0.55 ||
			t.rel(tx, ty).Variant() == sim.VarStack {
			return mapCliff
		}
		return scaleCol(c, 1+0.06*float64(l))
	case k == sim.TileWaterfall:
		return mapFall
	case runningWater(k):
		// Lighter down the middle of a wide river.
		if runningWater(t.tile(tx-1, ty)) && runningWater(t.tile(tx+1, ty)) || runningWater(t.tile(tx, ty-1)) && runningWater(t.tile(tx, ty+1)) {
			return mapRiverLit
		}
		return mapRiver
	case k == sim.TileWater && t.rel(tx, ty).Variant() == sim.VarLake:
		return mapLake
	case t.rel(tx, ty).Variant() == sim.VarRamp && k.Walkable():
		c = mix(c, pathRamp.at(3), 0.55)
	}
	return scaleCol(c, 1+0.06*float64(t.rel(tx, ty).Level()))
}

// mapShallow colours map sea pixel c at (u, v) over the shallows, lakes and
// sea stacks: the sea by its depth, as the ground draws it (see
// terrainsea.go), eased into c, the open sea, out where it is deepest.
func (t *terrain) mapShallow(c color.RGBA, u, v float64) color.RGBA {
	if t.relief == nil {
		return c
	}
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	switch {
	case t.tile(tx, ty) == sim.TileCliff:
		return mapCliff // a sea stack
	case t.tile(tx, ty) == sim.TileWater && t.rel(tx, ty).Variant() == sim.VarLake:
		return mapLake
	}
	d := t.seaDepthAt(u, v)
	if d >= 0.995 {
		return c
	}
	w := seaColor(d)
	a := float64(w.A) / 255
	col := mix(colWaterBG, withAlpha(w, 255), a)
	col.A = 255
	col = mix(col, c, smooth(clamp01((d-0.8)/0.2)))
	if t.tile(tx, ty) == sim.TileShallow {
		switch t.rel(tx, ty).Variant() {
		case sim.VarReef:
			if hashf(int(u*4), int(v*4), 1450) < 0.14 {
				return mapCoral
			}
		case sim.VarLagoon:
			col = mix(col, mapLagoon, 0.35)
		}
	}
	return col
}

// --- the minimap's pieces -----------------------------------------------------------

const (
	miniPx    = 5  // minimap pixels to a tile across (and 3.75 down)
	miniPiece = 32 // tiles to a side of a piece of the minimap
)

type miniKey struct{ cx, cy int }

type miniResult struct {
	key  miniKey
	gen  int
	data []byte
}

// miniPieces are the minimap's pieces baked so far, the rest baked off the
// game loop as they come into view.
type miniPieces struct {
	terr    *terrain
	gen     int
	imgs    map[miniKey]*ebiten.Image
	pending map[miniKey]bool
	used    map[miniKey]int
	results chan miniResult
	frame   int
}

func (p *miniPieces) reset(terr *terrain) {
	p.terr = terr
	p.gen++
	for k, img := range p.imgs {
		img.Deallocate()
		delete(p.imgs, k)
	}
	clear(p.pending)
	clear(p.used)
}

// pieceSize is a piece's size in minimap pixels.
func pieceSize() (w, h int) { return miniPiece * miniPx, miniPiece * miniPx * TileH / TileW }

// get returns piece k if it is baked, and starts baking it if not.
func (p *miniPieces) get(k miniKey) *ebiten.Image {
	if p.imgs == nil {
		p.imgs, p.pending, p.used = map[miniKey]*ebiten.Image{}, map[miniKey]bool{}, map[miniKey]int{}
		p.results = make(chan miniResult, 16)
	}
	p.collect()
	p.used[k] = p.frame
	if img := p.imgs[k]; img != nil || p.pending[k] || p.terr == nil {
		return img
	}
	p.pending[k] = true
	terr, gen, out := p.terr, p.gen, p.results
	go func() { out <- miniResult{k, gen, terr.bakeMiniPiece(k)} }()
	return nil
}

func (p *miniPieces) collect() {
	for {
		select {
		case r := <-p.results:
			delete(p.pending, r.key)
			if r.gen != p.gen {
				continue
			}
			w, h := pieceSize()
			img := ebiten.NewImage(w, h)
			img.WritePixels(r.data)
			p.imgs[r.key] = img
		default:
			// Keep the pieces seen lately; forget the rest.
			for len(p.imgs) > 24 {
				var old miniKey
				of := math.MaxInt
				for k := range p.imgs {
					if p.used[k] < of {
						old, of = k, p.used[k]
					}
				}
				p.imgs[old].Deallocate()
				delete(p.imgs, old)
				delete(p.used, old)
			}
			return
		}
	}
}

// bakeMiniPiece draws piece k of the minimap, as the world map draws it.
func (t *terrain) bakeMiniPiece(k miniKey) []byte {
	w, h := pieceSize()
	c := newCanvas(w, h)
	sx, sy := float64(miniPx), float64(miniPx)*TileH/TileW
	for y := 0; y < h; y++ {
		gy := k.cy*h + y
		for x := 0; x < w; x++ {
			gx := k.cx*w + x
			c.px[y*w+x] = t.mapPixel((float64(gx)+0.5)/sx, (float64(gy)+0.5)/sy, gx, gy)
		}
	}
	return c.bytes()
}

// drawMiniPieces draws the pieces of the minimap that cover canvas c, map
// tile (0, 0) landing at (ox, oy), and reports whether all were ready.
func (p *miniPieces) draw(c *ebiten.Image, ox, oy float64, frame int) bool {
	p.frame = frame
	w, h := pieceSize()
	cw, ch := float64(c.Bounds().Dx()), float64(c.Bounds().Dy())
	all := true
	for cy := int(math.Floor(-oy / float64(h))); float64(cy*h)+oy < ch; cy++ {
		for cx := int(math.Floor(-ox / float64(w))); float64(cx*w)+ox < cw; cx++ {
			if cx < 0 || cy < 0 || p.terr == nil || cx*miniPiece >= p.terr.w || cy*miniPiece >= p.terr.h {
				continue
			}
			img := p.get(miniKey{cx, cy})
			if img == nil {
				all = false
				continue
			}
			// Only the part on the map: off it the minimap shows fog.
			pw := min(w, int(math.Ceil(float64((p.terr.w-cx*miniPiece)*miniPx))))
			ph := min(h, int(math.Ceil(float64(p.terr.h-cy*miniPiece)*miniPx*TileH/TileW)))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(math.Round(float64(cx*w)+ox), math.Round(float64(cy*h)+oy))
			c.DrawImage(img.SubImage(image.Rect(0, 0, pw, ph)).(*ebiten.Image), op)
		}
	}
	return all
}
