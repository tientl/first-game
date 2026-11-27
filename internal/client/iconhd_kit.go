package client

import (
	"image/color"
	"math"
)

// Shared brushes of the HD icons (iconpaint.go): faceted stone, polished
// stone, locks of fur, scales, threads. Each paints in design units, lit from
// the top left, in clean bands of a ramp.

// facetFan paints a lump cut in flat faces: the polygon pts (x, y pairs)
// divided into triangles from (cx, cy), each lit as a face leaning out
// towards its edge by tilt (0 flat, 1 on its side). The first face is a new
// part, the rest more of it. speck varies the tone over a face.
func facetFan(p *icv, r ramp, cx, cy, tilt, bias, speck float64, seed uint32, pts ...float64) {
	n := len(pts) / 2
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		ax, ay, bx, by := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
		mx, my := (ax+bx)/2-cx, (ay+by)/2-cy
		if m := math.Hypot(mx, my); m > 0 {
			mx, my = mx/m, my/m
		}
		v := litOf(mx*tilt, my*tilt) + bias + (hashf(i, 3, seed)-0.5)*0.08
		tri := sPoly(cx, cy, ax, ay, bx, by).grow(0.02)
		sh := shFacet(r, v, speck, seed+uint32(i))
		if i == 0 {
			p.fill(tri, sh)
		} else {
			p.add(tri, sh)
		}
	}
}

// shPolish shades ground and polished stone about (cx, cy), rx by ry: lit
// as a low dome, a cloud in the stone, a bright rim where it turns to the
// light and one sharp glint.
func shPolish(r ramp, cx, cy, rx, ry, bias float64, glint color.RGBA, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := litOf(nx*0.75, ny*0.75) + bias + (fbm2(f.x*0.7, f.y*0.7, seed, 3)-0.5)*0.2
		if -f.d < 0.45 && toward(f.gx, f.gy) > 0.3 {
			l += 0.2
		}
		if gx, gy := nx+0.42, ny+0.46; gx*gx*1.6+gy*gy < 0.03 {
			return glint
		}
		return pick(r, l)
	}
}

// lock is a lock of fur or hair from (x0, y0) bent through (cx, cy) to its
// tip (x1, y1), w thick at the root: lit as a strand, a dark parting down
// its middle.
func lock(p *icv, r ramp, x0, y0, cx, cy, x1, y1, w, bias float64, mode int) {
	s := sLock(x0, y0, cx, cy, x1, y1, w, 0.05)
	a := mkAxis(x0, y0, x1, y1)
	p.paint(s, func(f *ifrag) color.RGBA {
		_, v := a.uv(f.x, f.y)
		l := bulge(f, w*0.9) + bias
		if math.Abs(v) < 0.09 {
			l -= 0.18
		}
		return pick(r, l)
	}, mode)
}

// shScales shades a scaly skin: rows of round scales cell units across,
// each lit as a small dome, the gaps between them dark.
func shScales(r ramp, cell, bias float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		gx, gy := f.x/cell, f.y/(cell*0.8)
		row := math.Floor(gy)
		off := 0.5 * math.Mod(row, 2)
		cx := math.Floor(gx+off) + 0.5 - off
		cy := row + 0.5
		dx, dy := gx-cx, (gy-cy)*1.1
		d := math.Hypot(dx, dy)
		l := litOf(dx*1.1, dy*1.1) + bias + (hashf(int(cx*2), int(row), seed)-0.5)*0.12
		if d > 0.5 {
			l -= 0.24
		}
		return pick(r, l)
	}
}

// shPebbly shades a reptile's pebbly skin: scales of uneven size and shape
// (cells about cell units across), each a low dome lit from the top left,
// a dark seam between them.
func shPebbly(r ramp, cell, bias float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		gx, gy := f.x/cell, f.y/cell
		ix, iy := math.Floor(gx), math.Floor(gy)
		best, second := 9.0, 9.0
		var bx, by float64
		var bi, bj int
		for dy := -1.0; dy <= 1; dy++ {
			for dx := -1.0; dx <= 1; dx++ {
				i, j := int(ix+dx), int(iy+dy)
				cx := ix + dx + 0.12 + 0.76*hashf(i, j, seed)
				cy := iy + dy + 0.12 + 0.76*hashf(i, j, seed+7)
				d := math.Hypot(gx-cx, gy-cy)
				if d < best {
					second, best, bx, by, bi, bj = best, d, cx, cy, i, j
				} else if d < second {
					second = d
				}
			}
		}
		l := litOf((gx-bx)*1.3, (gy-by)*1.3) + bias + (hashf(bi, bj, seed+11)-0.5)*0.2
		if second-best < 0.14 {
			l -= 0.3
		}
		return pick(r, l)
	}
}

// leafShape is a leaf from its stalk at (x, y) out along ang, length long
// and width wide at its widest: its half width at t (0 at the stalk, 1 at
// the tip) is form(t) times width.
func leafShape(x, y, length, width, ang float64, form func(t float64) float64) shape {
	const n = 14
	ex, ey := math.Cos(ang), math.Sin(ang)
	px, py := -ey, ex
	pts := make([]float64, 0, 4*(n+1))
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		w := form(t) * width
		pts = append(pts, x+ex*t*length+px*w, y+ey*t*length+py*w)
	}
	for i := n; i >= 0; i-- {
		t := float64(i) / n
		w := form(t) * width
		pts = append(pts, x+ex*t*length-px*w, y+ey*t*length-py*w)
	}
	return sPoly(pts...)
}

// Leaf outlines for leafShape.
var (
	formLance = func(t float64) float64 { return math.Pow(math.Sin(math.Pi*math.Min(1, t*1.05)), 0.75) * (1 - 0.2*t) }
	formOval  = func(t float64) float64 { return math.Sqrt(math.Max(0, math.Sin(math.Pi*t))) }
	formSaw   = func(t float64) float64 {
		return math.Pow(math.Sin(math.Pi*math.Min(1, t*1.05)), 0.7) * (0.82 + 0.18*math.Abs(math.Sin(t*math.Pi*6)))
	}
)

// leaf paints a leaf of ramp r (see leafShape): lit as a shallow fold along
// its midrib, the far half darker, the midrib a line, veins off it if veins.
func leaf(p *icv, r ramp, x, y, length, width, ang float64, form func(t float64) float64, veins bool, bias float64) {
	s := leafShape(x, y, length, width, ang, form)
	a := mkAxis(x, y, x+math.Cos(ang)*length, y+math.Sin(ang)*length)
	cx, cy := a.across()
	side := toward(cx, cy) // which half faces the light
	p.fill(s, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := 0.6 + bias + 0.1*clamp01(1+f.d/0.5)*toward(f.gx, f.gy)
		if v*side > 0 {
			l += 0.12
		} else {
			l -= 0.1
		}
		if math.Abs(v) < 0.09+0.04*(1-u/length) {
			l -= 0.2
		} else if veins && math.Abs(math.Mod(u-math.Abs(v)*0.9+20, 1.3)) < 0.12 {
			l -= 0.12
		}
		return pick(r, l)
	})
}

// shWet adds a wet sheen to sh: a band of light along the edges that face
// the light.
func shWet(sh ishader, r ramp, w float64) ishader {
	return func(f *ifrag) color.RGBA {
		c := sh(f)
		if -f.d < w && toward(f.gx, f.gy) > 0.55 {
			return mix(c, r[len(r)-1], 0.6)
		}
		return c
	}
}

// glintAt puts a small sharp glint, a lens rx by ry, on what is painted.
func glintAt(p *icv, x, y, rx, ry float64) {
	p.on(iEll(x, y, rx, ry, -0.6), solidC(icGlint))
}

// tilted is a shape turned by ang about (cx, cy).
func tilted(s shape, cx, cy, ang float64) shape {
	cs, sn := math.Cos(ang), math.Sin(ang)
	r := math.Max(math.Max(math.Abs(s.x0-cx), math.Abs(s.x1-cx)), math.Max(math.Abs(s.y0-cy), math.Abs(s.y1-cy))) * 1.42
	return shape{cx - r, cy - r, cx + r, cy + r, func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		return s.d(cx+dx*cs+dy*sn, cy-dx*sn+dy*cs)
	}}
}
