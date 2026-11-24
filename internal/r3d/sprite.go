package r3d

import (
	"image"
	"image/color"
	"math"
)

// Sprite is a finished pixel-art frame and the pixel of it that stands on
// the model's origin (the ground under the creature).
type Sprite struct {
	Img    *image.NRGBA
	AX, AY float64
}

// Dot is a single pixel put on the finished sprite where a point of the
// model shows: an eye's glint, which would otherwise be averaged away.
type Dot struct {
	P    Vec3 // in model space, already posed
	C    RGB
	Bias float64 // how far behind the surface it may sit and still show, in units
}

// SpriteOpts is how a render is turned into a sprite.
type SpriteOpts struct {
	SS       int     // the render is this many times the sprite's size
	Cover    float64 // share of a pixel's samples that must be covered for it to be solid
	Palette  []RGB   // every pixel snaps to one of these; nil keeps the colours
	Ink      RGB     // the outline
	Outline  bool    // a one-pixel rim outside the shape
	Edge     float64 // a step in depth of this many units darkens the pixel behind it; 0: none
	EdgeDark float64 // by this factor
	Grain    float64 // each pixel's brightness is nudged by up to this share before it snaps: a hand-dithered look where two shades meet
}

// Resolve averages t down by o.SS, snaps the colours to the palette, draws
// the inner and outer lines and the dots, and crops the result. cam is the
// camera t was rendered with.
func Resolve(t *Target, cam Camera, o SpriteOpts, dots []Dot) Sprite {
	ss := max(1, o.SS)
	w, h := t.W/ss, t.H/ss
	col := make([]RGB, w*h)
	solid := make([]bool, w*h)
	depth := make([]float32, w*h) // the mean depth of what covers the pixel
	front := make([]float32, w*h) // the nearest
	need := int(math.Ceil(o.Cover * float64(ss*ss)))
	need = max(1, need)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, z float64
			n := 0
			zf := float32(math.Inf(-1))
			for sy := 0; sy < ss; sy++ {
				row := (y*ss+sy)*t.W + x*ss
				for sx := 0; sx < ss; sx++ {
					i := row + sx
					if zi := t.Z[i]; !math.IsInf(float64(zi), -1) {
						r += float64(t.Col[i*3])
						g += float64(t.Col[i*3+1])
						b += float64(t.Col[i*3+2])
						z += float64(zi)
						zf = max(zf, zi)
						n++
					}
				}
			}
			i := y*w + x
			front[i] = zf
			if n >= need {
				f := 1 / float64(n)
				col[i], depth[i], solid[i] = RGB{r * f, g * f, b * f}, float32(z*f), true
			}
		}
	}
	at := func(x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return -1
		}
		return y*w + x
	}
	nb := [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}}
	// Inner lines: where something much nearer overlaps this pixel.
	if o.Edge > 0 {
		dark := make([]bool, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := y*w + x
				if !solid[i] {
					continue
				}
				for _, d := range nb {
					if j := at(x+d[0], y+d[1]); j >= 0 && solid[j] && float64(depth[j]-depth[i]) > o.Edge {
						dark[i] = true
						break
					}
				}
			}
		}
		for i, d := range dark {
			if d {
				col[i] = col[i].Scale(o.EdgeDark)
			}
		}
	}
	if len(o.Palette) > 0 {
		for i := range col {
			if solid[i] {
				if o.Grain > 0 {
					col[i] = col[i].Scale(1 + o.Grain*(grain(i%w, i/w)*2-1))
				}
				col[i] = o.Palette[Nearest(o.Palette, col[i])]
			}
		}
	}
	view := cam.View()
	for _, d := range dots {
		p := view.Point(d.P)
		if i := at(int(math.Floor(p.X/float64(ss))), int(math.Floor(p.Y/float64(ss)))); i >= 0 && solid[i] && p.Z >= float64(front[i])-d.Bias {
			col[i] = d.C
		}
	}
	// The outer line, and the box of what is drawn.
	ink := make([]bool, w*h)
	x0, y0, x1, y1 := w, h, -1, -1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if !solid[i] && o.Outline {
				for _, d := range nb {
					if j := at(x+d[0], y+d[1]); j >= 0 && solid[j] {
						ink[i] = true
						break
					}
				}
			}
			if solid[i] || ink[i] {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < x0 {
		return Sprite{Img: image.NewNRGBA(image.Rect(0, 0, 1, 1))}
	}
	img := image.NewNRGBA(image.Rect(0, 0, x1-x0+1, y1-y0+1))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*w + x
			switch {
			case solid[i]:
				img.SetNRGBA(x-x0, y-y0, col[i].NRGBA())
			case ink[i]:
				img.SetNRGBA(x-x0, y-y0, o.Ink.NRGBA())
			}
		}
	}
	return Sprite{Img: img, AX: cam.OX/float64(ss) - float64(x0), AY: cam.OY/float64(ss) - float64(y0)}
}

// grain is a fixed scatter in [0, 1) over the pixels.
func grain(x, y int) float64 {
	h := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca6b
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	return float64(h&0xffff) / 65536
}

// NRGBA is the colour as an opaque pixel.
func (a RGB) NRGBA() color.NRGBA {
	q := func(v float64) uint8 { return uint8(clamp01(v)*255 + 0.5) }
	return color.NRGBA{q(a.R), q(a.G), q(a.B), 255}
}

// dist is how different two colours look, roughly.
func dist(a, b RGB) float64 {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return 2*dr*dr + 4*dg*dg + 3*db*db
}

// Nearest is the index of the palette colour closest to c.
func Nearest(pal []RGB, c RGB) int {
	best, bd := 0, math.Inf(1)
	for i, p := range pal {
		if d := dist(p, c); d < bd {
			best, bd = i, d
		}
	}
	return best
}

// ShadePalette builds a small palette for a model: each of its colours at
// levels steps of brightness under lt, then the closest pairs merged until
// only n are left. keep are colours that must survive as they are (teeth,
// an eye) and count towards n.
func ShadePalette(lt Light, mats []RGB, levels, n int, keep []RGB) []RGB {
	type entry struct {
		c RGB
		w float64
	}
	var es []entry
	for _, m := range mats {
		for i := 0; i < levels; i++ {
			es = append(es, entry{lt.Shade(m, float64(i)/float64(levels-1)), 1})
		}
	}
	for len(es)+len(keep) > n && len(es) > 1 {
		bi, bj, bd := 0, 1, math.Inf(1)
		for i := range es {
			for j := i + 1; j < len(es); j++ {
				if d := dist(es[i].c, es[j].c); d < bd {
					bi, bj, bd = i, j, d
				}
			}
		}
		a, b := es[bi], es[bj]
		es[bi] = entry{a.c.Mix(b.c, b.w/(a.w+b.w)), a.w + b.w}
		es = append(es[:bj], es[bj+1:]...)
	}
	pal := append([]RGB{}, keep...)
	for _, e := range es {
		pal = append(pal, e.c)
	}
	return pal
}
