package client

import (
	"image/color"
	"math"
)

// The brushes of the HD environment: leaves, bark, fronds and stone, drawn
// in image pixels on a canvas marked HD (canvas.dens), a third of an art
// pixel fine. The tree generators work in pixels at a scale s; for HD they
// are run at three times the scale and these brushes put in the detail the
// extra pixels have room for. See hdup.go.

// markHD marks c as holding HD pixels when the HD environment is on, and
// returns it.
func markHD(c *canvas) *canvas {
	if envHD {
		c.dens = HDDensity
	}
	return c
}

// hd reports whether c is drawn in HD image pixels.
func (c *canvas) hd() bool { return c.dens == HDDensity }

// leafLook is how the leaves of a crown are drawn at HD: how far apart they
// sit and their half length and half width, all in image pixels.
type leafLook struct{ cell, a, b float64 }

var (
	leafBroad = leafLook{4.4, 3.9, 2.1} // a broadleaf tree
	leafSmall = leafLook{3.4, 2.9, 1.6} // bushes, small crowns
	leafFine  = leafLook{2.8, 2.5, 1.1} // needles in clumps
)

// leafClusterHD is leafCluster at HD: the same union of round blobs lit as
// one volume, but made of single leaves, each a pointed shape of one tone
// with a lit side, a shaded rim and a midrib, overlapping at random; the
// gaps between them are dark, and at the rim leaves stick out of the
// silhouette.
func leafClusterHD(c *canvas, blobs []blob, cx, cy, rx, ry float64, rp ramp, seed uint32, look leafLook) {
	pad := look.a + 1
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, b := range blobs {
		minX, maxX = math.Min(minX, b.x-b.r), math.Max(maxX, b.x+b.r)
		minY, maxY = math.Min(minY, b.y-b.r), math.Max(maxY, b.y+b.r)
	}
	minX, minY, maxX, maxY = minX-pad, minY-pad, maxX+pad, maxY+pad
	// at is the blob a point belongs to and its height in it (below 0:
	// outside every blob).
	at := func(px, py float64) (float64, int) {
		in, bi := math.Inf(-1), -1
		out, bo := math.Inf(-1), 0
		for i, b := range blobs {
			dx, dy := (px-b.x)/b.r, (py-b.y)/b.r
			h := 1 - (dx*dx + dy*dy)
			if h >= 0 && h+float64(i)*0.02 > in {
				in, bi = h+float64(i)*0.02, i
			} else if h < 0 && h > out {
				out, bo = h, i
			}
		}
		if bi < 0 {
			return out, bo
		}
		return in - float64(bi)*0.02, bi
	}
	light := func(px, py float64, b blob) float64 {
		gx, gy := (px-cx)/rx, (py-cy)/ry
		nx, ny := 0.45*(px-b.x)/b.r+0.65*gx, 0.45*(py-b.y)/b.r+0.65*gy
		l := lum(nx, ny) + (vnoise(px*0.15, py*0.15, seed)-0.5)*0.3
		if gy > 0.5 {
			l -= (gy - 0.5) * 0.4
		}
		return l
	}
	type leaf struct {
		x, y, cos, sin float64
		tone           int
		z              float64
		ok             bool
	}
	gx0, gy0 := int(math.Floor(minX/look.cell))-1, int(math.Floor(minY/look.cell))-1
	gw, gh := int(math.Ceil(maxX/look.cell))+2-gx0, int(math.Ceil(maxY/look.cell))+2-gy0
	leaves := make([]leaf, gw*gh)
	for j := range gh {
		for i := range gw {
			ix, iy := i+gx0, j+gy0
			lf := &leaves[j*gw+i]
			lf.x = (float64(ix) + hashf(ix, iy, seed+11)) * look.cell
			lf.y = (float64(iy) + hashf(ix, iy, seed+12)) * look.cell
			h, bi := at(lf.x, lf.y)
			if h < 0.04 {
				continue
			}
			lf.ok = true
			// Leaves point out from the middle of the crown and hang a little.
			a := math.Atan2((lf.y-cy)/ry*0.7+0.45, (lf.x-cx)/rx) + (hashf(ix, iy, seed+13)-0.5)*1.5
			lf.cos, lf.sin = math.Cos(a), math.Sin(a)
			l := light(lf.x, lf.y, blobs[bi]) + (hashf(ix, iy, seed+14)-0.5)*0.16
			if hashf(ix, iy, seed+15) < 0.08 {
				l += 0.14 // one catching the sun
			}
			lf.tone = rp.idx(l, ix, iy)
			lf.z = hashf(ix, iy, seed+16)
		}
	}
	for y := int(minY); y <= int(maxY); y++ {
		for x := int(minX); x <= int(maxX); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			ci, cj := int(math.Floor(px/look.cell))-gx0, int(math.Floor(py/look.cell))-gy0
			top, tone := -1.0, 0
			for dj := -1; dj <= 1; dj++ {
				for di := -1; di <= 1; di++ {
					i, j := ci+di, cj+dj
					if i < 0 || j < 0 || i >= gw || j >= gh {
						continue
					}
					lf := &leaves[j*gw+i]
					if !lf.ok || lf.z <= top {
						continue
					}
					dx, dy := px-lf.x, py-lf.y
					un := (dx*lf.cos + dy*lf.sin) / look.a
					if un < -1 || un > 1 {
						continue
					}
					v := math.Abs(-dx*lf.sin + dy*lf.cos)
					w := look.b * math.Sqrt(1-un*un) * (1 - 0.35*un)
					if v > w {
						continue
					}
					top, tone = lf.z, lf.tone
					side := -(dx*0.62 + dy*0.78) / look.a // towards the light
					switch {
					case v < 0.5 && un < 0.6 && un > -0.8:
						tone++ // the midrib
					case side > 0.3:
						tone++
					case v > w-1.1 && side < 0:
						tone-- // the rim away from the light
					}
				}
			}
			if top >= 0 {
				c.put(x, y, rp.at(tone))
				continue
			}
			if h, bi := at(px, py); h >= 0.16 { // a gap between leaves, deep in the crown
				c.put(x, y, rp.pick(light(px, py, blobs[bi])-0.34, x, y))
			}
		}
	}
}

// barkAt shades a point of a limb at HD: u across it (-1 at the left edge to
// 1) of a limb w pixels half wide, along it in pixels. The bark is lit from the left and furrowed along
// its length, with a ridge catching the light here and there.
func barkAt(rp ramp, u, w, along float64, x, y int, lit float64, seed uint32) color.RGBA {
	l := lit - (u+1)/2*0.7
	f := vnoise(u*max(w, 3)*0.42+float64(seed), along*0.085, seed)
	switch {
	case f < 0.36:
		l -= 0.22 // a furrow
	case f > 0.7:
		l += 0.1
	}
	l += (hashf(x, y, seed+3) - 0.5) * 0.1
	return rp.pick(l, x, y)
}

// drawLimbHD is drawLimbIn at HD.
func drawLimbHD(c *canvas, l limb, rp ramp, lit float64, seed uint32) {
	length := math.Hypot(l.x1-l.x0, l.y1-l.y0)
	n := int(length*2) + 2
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := l.at(t)
		w := lerpf(l.w0, l.w1, t)
		for py := int(math.Floor(y - w)); py <= int(math.Ceil(y+w)); py++ {
			for px := int(math.Floor(x - w)); px <= int(math.Ceil(x+w)); px++ {
				dx, dy := float64(px)+0.5-x, float64(py)+0.5-y
				if dx*dx+dy*dy > w*w+0.25 {
					continue
				}
				c.put(px, py, barkAt(rp, dx/max(w, 0.5), w, t*length, px, py, lit, seed))
			}
		}
	}
}

// knotHD is a knot in the bark at (x, y), r pixels across: a dark ring round
// a paler eye. Only pixels already of the wood are touched.
func knotHD(c *canvas, x, y, r float64, rp ramp) {
	for py := int(y - r*1.4); py <= int(y+r*1.4)+1; py++ {
		for px := int(x - r); px <= int(x+r)+1; px++ {
			dx, dy := (float64(px)+0.5-x)/r, (float64(py)+0.5-y)/(r*1.4)
			d := dx*dx + dy*dy
			if d > 1 || !c.opaque(px, py) {
				continue
			}
			switch {
			case d > 0.45:
				c.put(px, py, rp.at(0))
			case d > 0.12:
				c.put(px, py, rp.at(len(rp)-2))
			default:
				c.put(px, py, rp.at(1))
			}
		}
	}
}

// frondHD is frond at HD: the stem arches the same way, and every leaflet is
// a blade of its own, two pixels wide at the stem and tapering, its upper
// edge lit, with a little air between it and the next.
func frondHD(c *canvas, x0, y0, a, L float64, rp ramp, stiff bool, seed uint32) {
	droop, leaf, gap, sag := 0.95, 0.2, 2.7, 0.18
	if stiff {
		droop, leaf, gap, sag = -0.05, 0.16, 2.3, 0.02
	}
	ex, ey := x0+math.Cos(a)*L, y0+math.Sin(a)*L*0.75+L*droop
	cx, cy := x0+math.Cos(a)*L*0.6, y0+math.Sin(a)*L*0.95
	at := func(t float64) (float64, float64) {
		u := 1 - t
		return u*u*x0 + 2*u*t*cx + t*t*ex, u*u*y0 + 2*u*t*cy + t*t*ey
	}
	n := int(L*2.5) + 4
	next, walked := gap*1.5, 0.0
	lx, ly := x0, y0
	blade := 0
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := at(t)
		walked += math.Hypot(x-lx, y-ly)
		lx, ly = x, y
		x2, y2 := at(min(1, t+0.02))
		tx, ty := x2-x, y2-y
		d := math.Hypot(tx, ty)
		if d == 0 {
			continue
		}
		tx, ty = tx/d, ty/d
		nx, ny := -ty, tx // across the stem
		if walked >= next && t > 0.06 {
			next += gap
			blade++
			ln := leaf * L * (1 - t*0.85) * (0.6 + 0.4*math.Sin(t*math.Pi)) * (0.85 + 0.3*hashf(blade, 0, seed))
			for _, side := range []float64{-1, 1} {
				// The blade sweeps forward a little, and droops.
				dx, dy := nx*side+tx*0.6, ny*side+ty*0.6+sag
				dl := math.Hypot(dx, dy)
				dx, dy = dx/dl, dy/dl
				tone := -side*0.1 - t*0.15 + (hashf(blade, int(side)+2, seed)-0.5)*0.22
				for k := 0.0; k <= ln; k += 0.5 {
					px, py := x+dx*k, y+dy*k+k*k/ln*sag*0.6
					l := 0.78 - k/ln*0.22 + tone
					ix, iy := int(math.Floor(px)), int(math.Floor(py))
					if k < ln*0.75 { // the shaded under-edge of the blade
						if math.Abs(dx) > math.Abs(dy) {
							c.put(ix, iy+1, rp.pick(l-0.3, ix, iy+1))
						} else {
							c.put(ix+1, iy, rp.pick(l-0.3, ix+1, iy))
						}
					}
					c.put(ix, iy, rp.pick(l, ix, iy))
				}
			}
		}
		ix, iy := int(math.Floor(x)), int(math.Floor(y))
		c.put(ix, iy, rp.pick(0.3, ix, iy))
		if t < 0.6 {
			c.put(ix, iy+1, rp.pick(0.15, ix, iy+1))
		}
	}
}
