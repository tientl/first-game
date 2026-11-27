package sim

import (
	"math"
	"math/rand/v2"
)

// Digging a planned cave system into the map: see cavegen.go.

// caveDig is one level of a system being dug, on the system's grid.
type caveDig struct {
	s     *caveSystem
	lv    int
	open  []bool
	owner []int16 // per cell: the chamber it is the floor of, plus one; 0 for none
}

// digSystem digs system s into the map: each level on its grid, smoothed
// and cleaned, then into the tiles, with the ways up and down, the mouths
// on the surface and what each tile of the caves is.
func (w *World) digSystem(s *caveSystem) {
	seed := uint32(w.seed) ^ uint32(s.island+1)*0x9e3779b1
	digs := make([]*caveDig, s.levels)
	for lv := range digs {
		d := &caveDig{s: s, lv: lv, open: make([]bool, s.gw*s.gh), owner: make([]int16, s.gw*s.gh)}
		digs[lv] = d
		for i := range s.chambers {
			switch c := &s.chambers[i]; {
			case c.level != lv:
			case c.junction:
				d.widen(c.x, c.y, c.r)
			default:
				d.chamber(i, seed)
			}
		}
		for i := range s.chambers {
			if c := &s.chambers[i]; c.level == lv {
				for _, p := range c.pillars {
					d.pillar(p, seed)
				}
			}
		}
		for i := range s.tunnels {
			if s.tunnels[i].level == lv {
				d.tunnel(s.tunnels[i], seed, false)
			}
		}
		d.smooth()
		for i := range s.tunnels {
			if s.tunnels[i].level == lv {
				d.tunnel(s.tunnels[i], seed, true) // the cores again: nothing narrows them
			}
		}
	}
	// The doors in the north walls of the halls, where the ways up are.
	for _, m := range s.mouths {
		digs[0].exitDoor(m[0]-s.x0, m[1]-s.y0)
	}
	// Where the hole down and the rope up are: off the tunnels, open all
	// round on both levels.
	if s.levels == 2 {
		hx, hy := s.holeSpot(digs, s.descent)
		s.down = [2]int{hx, CaveBandTop(w.surfaceH, 0) + hy}
		s.up = [2]int{hx, CaveBandTop(w.surfaceH, 1) + hy}
		for i := range s.shafts {
			sh := &s.shafts[i]
			hx, hy := s.holeSpot(digs, sh.descent)
			sh.down = [2]int{hx, CaveBandTop(w.surfaceH, 0) + hy}
			sh.up = [2]int{hx, CaveBandTop(w.surfaceH, 1) + hy}
		}
	}
	for lv, d := range digs {
		d.open3()
		var seeds []int
		if lv == 0 {
			for _, m := range s.mouths {
				seeds = append(seeds, (m[1]-s.y0)*s.gw+m[0]-s.x0)
			}
		} else {
			seeds = append(seeds, (s.up[1]-CaveBandTop(w.surfaceH, 1)-s.y0)*s.gw+s.up[0]-s.x0)
			for _, sh := range s.shafts {
				seeds = append(seeds, (sh.up[1]-CaveBandTop(w.surfaceH, 1)-s.y0)*s.gw+sh.up[0]-s.x0)
			}
		}
		d.reach(seeds)
		s.owner[lv], s.open[lv] = d.owner, d.open
	}
	w.writeSystem(s, digs)
}

// chamber opens chamber i: a blob round its middle, its edge pushed in and
// out by noise.
func (d *caveDig) chamber(i int, seed uint32) {
	s, c := d.s, &d.s.chambers[i]
	R := c.r * 1.5 * max(c.aspect, 1/c.aspect)
	for ty := int(math.Floor(c.y - R)); ty <= int(math.Ceil(c.y+R)); ty++ {
		for tx := int(math.Floor(c.x - R)); tx <= int(math.Ceil(c.x+R)); tx++ {
			cell, ok := s.gridAt(tx, ty)
			if !ok || d.border(cell) {
				continue
			}
			if s.inBlob(c, float64(tx)+0.5, float64(ty)+0.5, seed) {
				d.open[cell] = true
				if d.owner[cell] == 0 {
					d.owner[cell] = int16(i + 1)
				}
			}
		}
	}
}

// inBlob reports whether (x, y) lies in the floor of chamber c: its blob
// round its middle, the edge pushed in and out by noise (less for a hall
// and a side pocket).
func (s *caveSystem) inBlob(c *caveChamber, x, y float64, seed uint32) bool {
	cs, sn := math.Cos(c.rot), math.Sin(c.rot)
	dx, dy := x-c.x, y-c.y
	u, v := (dx*cs+dy*sn)/c.aspect, (-dx*sn+dy*cs)*c.aspect
	warp := 1 + 1.25*(fbm(x/9+c.x, y/9, seed+11, 3)-0.5) + 0.14*(fbm(x/3, y/3, seed+12, 2)-0.5)
	if c.hall || c.pocket != 0 {
		warp = 1 + 0.5*(fbm(x/4, y/4, seed+13, 2)-0.5)
	}
	return math.Hypot(u, v)/c.r < warp
}

// widen opens a round widening of radius r at (x, y), part of the tunnels:
// where a passage forks.
func (d *caveDig) widen(x, y, r float64) {
	for ty := int(math.Floor(y - r)); ty <= int(math.Ceil(y+r)); ty++ {
		for tx := int(math.Floor(x - r)); tx <= int(math.Ceil(x+r)); tx++ {
			if math.Hypot(float64(tx)+0.5-x, float64(ty)+0.5-y) > r {
				continue
			}
			if cell, ok := d.s.gridAt(tx, ty); ok && !d.border(cell) {
				d.open[cell] = true
			}
		}
	}
}

// pillar fills in a pillar of rock at p (x, y, radius).
func (d *caveDig) pillar(p [3]float64, seed uint32) {
	s := d.s
	R := p[2] * 1.4
	for ty := int(math.Floor(p[1] - R)); ty <= int(math.Ceil(p[1]+R)); ty++ {
		for tx := int(math.Floor(p[0] - R)); tx <= int(math.Ceil(p[0]+R)); tx++ {
			cell, ok := s.gridAt(tx, ty)
			if !ok {
				continue
			}
			x, y := float64(tx)+0.5, float64(ty)+0.5
			warp := 1 + 0.6*(fbm(x/2.2, y/2.2, seed+21, 2)-0.5)
			if math.Hypot(x-p[0], y-p[1])/p[2] < warp {
				d.open[cell] = false
				d.owner[cell] = 0
			}
		}
	}
}

// tunnel opens tunnel t along its path: with core, only the run of
// three-by-three squares round its tiles, that keeps it three wide at
// least; else a round brush as wide as the tunnel, wavering along it.
func (d *caveDig) tunnel(t caveTunnel, seed uint32, core bool) {
	s := d.s
	along := 0.0
	for j := 0; j+1 < len(t.path); j++ {
		p, q := t.path[j], t.path[j+1]
		l := math.Hypot(q[0]-p[0], q[1]-p[1])
		steps := max(1, int(math.Ceil(l/0.35)))
		for k := 0; k <= steps; k++ {
			f := float64(k) / float64(steps)
			x, y := p[0]+(q[0]-p[0])*f, p[1]+(q[1]-p[1])*f
			a := along + l*f
			if core {
				cx, cy := int(math.Floor(x)), int(math.Floor(y))
				for yy := cy - 1; yy <= cy+1; yy++ {
					for xx := cx - 1; xx <= cx+1; xx++ {
						if cell, ok := s.gridAt(xx, yy); ok && !d.border(cell) {
							d.open[cell] = true
						}
					}
				}
				continue
			}
			wd := t.width + (caveTunnelMax-t.width)*clamp((fbm(a/9, float64(t.a*31+t.b), seed+31, 2)-0.3)/0.4, 0, 1)
			r := wd / 2
			for yy := int(math.Floor(y - r)); yy <= int(math.Ceil(y+r)); yy++ {
				for xx := int(math.Floor(x - r)); xx <= int(math.Ceil(x+r)); xx++ {
					if dx, dy := float64(xx)+0.5-x, float64(yy)+0.5-y; dx*dx+dy*dy > r*r {
						continue
					}
					if cell, ok := s.gridAt(xx, yy); ok && !d.border(cell) {
						d.open[cell] = true
					}
				}
			}
		}
		along += l
	}
}

// border reports whether cell lies on the grid's edge, which stays rock.
func (d *caveDig) border(cell int) bool {
	x, y := cell%d.s.gw, cell/d.s.gw
	return x < 2 || y < 2 || x >= d.s.gw-2 || y >= d.s.gh-2
}

// smooth rounds the walls once: a tooth of rock with open ground nearly all
// round it is worn away, a spur of floor nearly walled in is filled.
func (d *caveDig) smooth() {
	gw, gh := d.s.gw, d.s.gh
	out := make([]bool, len(d.open))
	copy(out, d.open)
	for y := 1; y < gh-1; y++ {
		for x := 1; x < gw-1; x++ {
			n := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if (dx != 0 || dy != 0) && d.open[(y+dy)*gw+x+dx] {
						n++
					}
				}
			}
			i := y*gw + x
			switch {
			case d.open[i] && n <= 2:
				out[i] = false
			case !d.open[i] && n >= 6 && !d.border(i):
				out[i] = true
			}
		}
	}
	d.open = out
}

// exitDoor makes the door of a way up at cell (x, y): open in front, so it
// is reached from the hall, and rock behind it, where no tunnel passes.
func (d *caveDig) exitDoor(x, y int) {
	gw := d.s.gw
	for yy := y; yy <= y+2; yy++ {
		for xx := x - 1; xx <= x+1; xx++ {
			d.open[yy*gw+xx] = true
		}
	}
	for yy := y - 2; yy <= y-1; yy++ {
		for xx := x - 2; xx <= x+2; xx++ {
			if !d.s.nearTunnel(0, float64(d.s.x0+xx)+0.5, float64(d.s.y0+yy)+0.5, 2.4) {
				d.open[yy*gw+xx] = false
			}
		}
	}
}

// open3 fills in every floor tile that is not part of some open
// three-by-three square: nothing in the caves is narrower than three tiles.
func (d *caveDig) open3() {
	gw, gh := d.s.gw, d.s.gh
	full := make([]bool, len(d.open))
	for y := 1; y < gh-1; y++ {
		for x := 1; x < gw-1; x++ {
			ok := true
			for dy := -1; dy <= 1 && ok; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if !d.open[(y+dy)*gw+x+dx] {
						ok = false
						break
					}
				}
			}
			full[y*gw+x] = ok
		}
	}
	for y := 0; y < gh; y++ {
		for x := 0; x < gw; x++ {
			i := y*gw + x
			if !d.open[i] {
				continue
			}
			keep := false
			for dy := -1; dy <= 1 && !keep; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if nx, ny := x+dx, y+dy; nx >= 0 && ny >= 0 && nx < gw && ny < gh && full[ny*gw+nx] {
						keep = true
						break
					}
				}
			}
			if !keep {
				d.open[i], d.owner[i] = false, 0
			}
		}
	}
}

// reach fills in whatever floor cannot be walked to from seeds.
func (d *caveDig) reach(seeds []int) {
	gw, gh := d.s.gw, d.s.gh
	seen := make([]bool, len(d.open))
	var q []int
	for _, c := range seeds {
		if c >= 0 && c < len(d.open) && d.open[c] && !seen[c] {
			seen[c] = true
			q = append(q, c)
		}
	}
	for n := 0; n < len(q); n++ {
		x, y := q[n]%gw, q[n]/gw
		for _, dd := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+dd[0], y+dd[1]
			if nx < 0 || ny < 0 || nx >= gw || ny >= gh {
				continue
			}
			if j := ny*gw + nx; d.open[j] && !seen[j] {
				seen[j] = true
				q = append(q, j)
			}
		}
	}
	for i := range d.open {
		if d.open[i] && !seen[i] {
			d.open[i], d.owner[i] = false, 0
		}
	}
	for i := range d.owner {
		if !d.open[i] {
			d.owner[i] = 0
		}
	}
}

// holeSpot is where in descent chamber ci the hole down is (and under it,
// in the arrival chamber, the rope up), in tiles of the surface: open
// round it on both levels, off every tunnel, as near a third of the way out
// from the chamber's middle as can be.
func (s *caveSystem) holeSpot(digs []*caveDig, ci int) (int, int) {
	c := s.chambers[ci]
	R := int(math.Ceil(c.r))
	bx, by, best := int(c.x), int(c.y), math.Inf(1)
	for ty := int(c.y) - R; ty <= int(c.y)+R; ty++ {
		for tx := int(c.x) - R; tx <= int(c.x)+R; tx++ {
			ok := true
			for _, d := range digs {
				for dy := -2; dy <= 2 && ok; dy++ {
					for dx := -2; dx <= 2; dx++ {
						cell, in := s.gridAt(tx+dx, ty+dy)
						if !in || !d.open[cell] {
							ok = false
							break
						}
					}
				}
			}
			x, y := float64(tx)+0.5, float64(ty)+0.5
			if !ok || s.nearTunnel(0, x, y, 2.6) || s.nearTunnel(1, x, y, 2.6) {
				continue
			}
			if v := math.Abs(math.Hypot(x-c.x, y-c.y) - c.r*0.35); v < best {
				bx, by, best = tx, ty, v
			}
		}
	}
	if math.IsInf(best, 1) {
		// Nowhere off the tunnels: by the middle, opened round on both levels.
		for _, d := range digs {
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if cell, in := s.gridAt(bx+dx, by+dy); in && !d.border(cell) {
						d.open[cell] = true
					}
				}
			}
		}
	}
	return bx, by
}

// writeSystem puts the dug levels of s into the map: their floor, what each
// tile is, the ways up and down and the mouths above.
func (w *World) writeSystem(s *caveSystem, digs []*caveDig) {
	for lv, d := range digs {
		top := CaveBandTop(w.surfaceH, lv)
		rocks := s.cellRocks(d)
		route := s.routeCells(d)
		for i, o := range d.open {
			if !o {
				continue
			}
			tx, ty := s.x0+i%s.gw, top+s.y0+i/s.gw
			w.tiles[ty*w.width+tx] = TileCave
			z := ZoneTunnel
			if c := d.owner[i]; c > 0 {
				z = s.chambers[c-1].zone
			}
			w.setCaveCell(tx, ty, makeCaveCell(z, rocks[i], route[i]))
			s.floor[lv]++
		}
	}
	// The mouths, and the ways up under them.
	top0 := CaveBandTop(w.surfaceH, 0)
	for _, m := range s.mouths {
		ux, uy := m[0], top0+m[1]
		w.standMouth(m[0], m[1])
		w.addEntity(EntityCaveExit, ux, uy)
		w.links[[2]int{m[0], m[1]}] = caveLink{float64(ux) + 0.5, float64(uy) + 1.5}
		w.links[[2]int{ux, uy}] = caveLink{float64(m[0]) + 0.5, float64(m[1]) + 1.5}
		s.exits = append(s.exits, [2]int{ux, uy})
	}
	if s.levels == 2 {
		w.addEntity(EntityCaveDown, s.down[0], s.down[1])
		w.addEntity(EntityCaveUp, s.up[0], s.up[1])
		w.links[s.down] = caveLink{float64(s.up[0]) + 0.5, float64(s.up[1]) + 1.5}
		w.links[s.up] = caveLink{float64(s.down[0]) + 0.5, float64(s.down[1]) + 1.5}
		for _, sh := range s.shafts {
			w.addEntity(EntityCaveDown, sh.down[0], sh.down[1])
			w.addEntity(EntityCaveUp, sh.up[0], sh.up[1])
			w.links[sh.down] = caveLink{float64(sh.up[0]) + 0.5, float64(sh.up[1]) + 1.5}
			w.links[sh.up] = caveLink{float64(sh.down[0]) + 0.5, float64(sh.down[1]) + 1.5}
		}
	}
}

// standMouth sets a cave mouth on surface tile (tx, ty): a doorway in the
// south face of an outcrop, rock beside and behind it.
func (w *World) standMouth(tx, ty int) {
	for y := ty - 1; y <= ty+1; y++ {
		for x := tx - 2; x <= tx+2; x++ {
			if e := w.entityAt(x, y); e != nil {
				w.removeEntity(e)
			}
		}
	}
	w.addEntity(EntityCaveMouth, tx, ty)
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {-1, -1}, {0, -1}, {1, -1}} {
		w.addEntity(EntityCaveRock, tx+d[0], ty+d[1])
	}
}

// cellRocks is the rock of each open cell of d: its chamber's, or in a
// tunnel the nearest chamber's.
func (s *caveSystem) cellRocks(d *caveDig) []CaveRock {
	gw, gh := s.gw, s.gh
	rock := make([]CaveRock, len(d.open))
	seen := make([]bool, len(d.open))
	var q []int
	for i, o := range d.owner {
		if o > 0 {
			rock[i], seen[i] = s.chambers[o-1].rock, true
			q = append(q, i)
		}
	}
	for n := 0; n < len(q); n++ {
		x, y := q[n]%gw, q[n]/gw
		for _, dd := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+dd[0], y+dd[1]
			if nx < 0 || ny < 0 || nx >= gw || ny >= gh {
				continue
			}
			if j := ny*gw + nx; d.open[j] && !seen[j] {
				rock[j], seen[j] = rock[q[n]], true
				q = append(q, j)
			}
		}
	}
	return rock
}

// routeCells marks the open cells of d along the main route: a trodden way
// a little over two tiles wide down the middle of its tunnels and across its
// chambers.
func (s *caveSystem) routeCells(d *caveDig) []bool {
	out := make([]bool, len(d.open))
	for _, t := range s.tunnels {
		if t.level != d.lv || !t.route {
			continue
		}
		for j := 0; j+1 < len(t.path); j++ {
			p, q := t.path[j], t.path[j+1]
			x0, x1 := int(math.Floor(min(p[0], q[0])-2)), int(math.Ceil(max(p[0], q[0])+2))
			y0, y1 := int(math.Floor(min(p[1], q[1])-2)), int(math.Ceil(max(p[1], q[1])+2))
			for ty := y0; ty <= y1; ty++ {
				for tx := x0; tx <= x1; tx++ {
					cell, ok := s.gridAt(tx, ty)
					if !ok || !d.open[cell] {
						continue
					}
					if _, dd := segDist(float64(tx)+0.5, float64(ty)+0.5, p[0], p[1], q[0], q[1]); dd <= 1.15 {
						out[cell] = true
					}
				}
			}
		}
	}
	return out
}

// digSecretPassages joins a few systems' lower levels to a neighbour's by a
// long tunnel under the sea: the nearest pairs, each system in one at most.
func (w *World) digSecretPassages(rng *rand.Rand) {
	type cand struct {
		a, b, ca, cb int
		d            float64
	}
	var cands []cand
	deep := 0
	ss := w.caves.systems
	for a, sa := range ss {
		if sa.levels < 2 {
			continue
		}
		deep++
		for b := a + 1; b < len(ss); b++ {
			sb := ss[b]
			if sb.levels < 2 {
				continue
			}
			best := cand{d: math.Inf(1)}
			for i, ci := range sa.chambers {
				for j, cj := range sb.chambers {
					if ci.level != 1 || cj.level != 1 || ci.junction || cj.junction || ci.pocket != 0 || cj.pocket != 0 {
						continue
					}
					if dd := math.Hypot(ci.x-cj.x, ci.y-cj.y); dd < best.d {
						best = cand{a, b, i, j, dd}
					}
				}
			}
			if best.d <= caveSecretReach {
				cands = append(cands, best)
			}
		}
	}
	want := deep / caveSecretPer
	used := make(map[int]bool)
	for len(w.caves.secrets) < want {
		k, best := -1, math.Inf(1)
		for i, c := range cands {
			if !used[c.a] && !used[c.b] && c.d < best {
				k, best = i, c.d
			}
		}
		if k < 0 {
			break
		}
		c := cands[k]
		used[c.a], used[c.b] = true, true
		w.digSecret(rng, c.a, c.b, c.ca, c.cb)
	}
}

// digSecret digs a secret passage from chamber ca of system a to chamber cb
// of system b, on the lower level.
func (w *World) digSecret(rng *rand.Rand, a, b, ca, cb int) {
	sa, sb := w.caves.systems[a], w.caves.systems[b]
	p, q := sa.chambers[ca], sb.chambers[cb]
	t := caveTunnel{a: ca, b: cb, level: 1}
	pts := [][2]float64{{p.x, p.y}, {q.x, q.y}}
	for range 5 {
		var out [][2]float64
		for j := 0; j+1 < len(pts); j++ {
			u, v := pts[j], pts[j+1]
			out = append(out, u)
			dx, dy := v[0]-u[0], v[1]-u[1]
			l := math.Hypot(dx, dy)
			if l < 6 {
				continue
			}
			off := (rng.Float64()*2 - 1) * l * 0.18
			out = append(out, [2]float64{(u[0]+v[0])/2 - dy/l*off, (u[1]+v[1])/2 + dx/l*off})
		}
		pts = append(out, pts[len(pts)-1])
	}
	t.path = pts
	top := CaveBandTop(w.surfaceH, 1)
	half := len(pts) / 2
	for j := 0; j+1 < len(pts); j++ {
		u, v := pts[j], pts[j+1]
		rock := p.rock
		if j >= half {
			rock = q.rock
		}
		l := math.Hypot(v[0]-u[0], v[1]-u[1])
		steps := max(1, int(math.Ceil(l/0.35)))
		for k := 0; k <= steps; k++ {
			f := float64(k) / float64(steps)
			x, y := u[0]+(v[0]-u[0])*f, u[1]+(v[1]-u[1])*f
			cx, cy := int(math.Floor(x)), int(math.Floor(y))
			for yy := cy - 1; yy <= cy+1; yy++ {
				for xx := cx - 1; xx <= cx+1; xx++ {
					if xx < 2 || yy < 2 || xx >= w.width-2 || yy >= w.surfaceH-2 {
						continue
					}
					i := (top+yy)*w.width + xx
					if w.tiles[i] == TileMountain {
						w.tiles[i] = TileCave
						w.setCaveCell(xx, top+yy, makeCaveCell(ZoneTunnel, rock, false))
					}
				}
			}
		}
	}
	w.caves.secrets = append(w.caves.secrets, caveSecret{a: a, b: b, ca: ca, cb: cb, path: pts})
}
