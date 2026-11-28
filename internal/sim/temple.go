package sim

import "math"

// Titanoboa's temple, in worlds made since StyleLairs. Deep in the heart of
// the biggest jungle island an ancient temple lies in ruins round a black
// pool: a paved courtyard about thirty tiles across, open to the sky, ringed
// by what is left of a colonnade, with columns fallen across it to hide
// behind (they stop a sweep of its tail and its coils: coverBetween). The
// snake sleeps coiled in the pool. A giant dead tree stands over the
// temple's far side, seen from far off over the canopy, with birds circling
// it. A beaten trail runs in to the courtyard's gate from a small clearing
// by a stream thirty-odd tiles out, a safe camp with room for a fire and a
// tent; along it lie the signs of the snake (lair.go: bones, skins, broken
// trees, the track of its belly). Everything here is made when the world is,
// from the seed alone, so a save finds it again.

// Temple tuning.
const (
	TempleRX      = 20.0 // the courtyard's half widths, in tiles: 40 by 31 at full size, so the
	TempleRY      = 15.5 // whole of a screen (27 by 20 tiles at 1080p) fits in it with room over
	templeMinK    = 0.4  // its size, as a share of that, on a small world's jungle at the least
	templePoolRX  = 5.4  // the black pool's half widths
	templePoolRY  = 4.0
	templeRingQ   = 0.88 // the colonnade stands this far out to the courtyard's edge (1)
	templeTreeQ   = 0.84 // and so does the dead tree
	templeColumns = 18   // pillars round it, before the gate and the broken ones are left out
	templeFallen  = 6    // columns fallen across the courtyard
	campNear      = 30.0 // the camp lies this far from the lair at least...
	campFar       = 40.0 // ...and at most
	campRoom      = 4    // and has this much flat ground round it
)

// templeSite finds where Titanoboa's temple goes: on the biggest jungle
// island, in the deepest of its woods, where its courtyard (k times the
// full size) is flat ground at one level with nothing in the way. ok is
// false where no island has room for one even of templeMinK.
func (w *World) templeSite() (x, y, k float64, ok bool) {
	isl := w.bossIsland(CreatureTitanoboa)
	if isl < 0 {
		return 0, 0, 0, false
	}
	W, H := w.width, w.SurfaceHeight()
	im := w.islandIndex()
	bx0, by0, bx1, by1 := W, H, 0, 0 // the island's bounds: the room is worked out only there
	for i, v := range im {
		if v == int16(isl) {
			x, y := i%W, i/W
			bx0, by0, bx1, by1 = min(bx0, x), min(by0, y), max(bx1, x+1), max(by1, y+1)
		}
	}
	room := w.flatRoom(int(math.Ceil(TempleRY))+4, bx0-1, by0-1, bx1+1, by1+1)
	inland := w.landDistance(60)
	size := float64(max(W, H))
	step := 2
	if W*H > 600*600 {
		step = 3
	}
	// fits reports whether the courtyard of half widths (rx, ry) round
	// tile (tx, ty), and a tile more, is all flat ground with nothing kept
	// on it.
	fits := func(tx, ty int, rx, ry float64) bool {
		sx, sy := int(rx)+2, int(ry)+2
		for y := ty - sy; y <= ty+sy; y++ {
			for x := tx - sx; x <= tx+sx; x++ {
				if ArenaQ(LayoutTemple, rx+1.2, ry+1.2, float64(x-tx), float64(y-ty)) > 1 {
					continue
				}
				if room.at(x, y) < 1 || im[y*W+x] != int16(isl) {
					return false
				}
				if e := w.entityAt(x, y); e != nil && kept(e.Kind) {
					return false
				}
			}
		}
		return true
	}
	// The full courtyard where there is room; a smaller one on a small
	// world's jungle.
	for _, kk := range []float64{1, 0.92, 0.85, 0.75, 0.62, 0.5, templeMinK} {
		rx, ry := TempleRX*kk, TempleRY*kk
		need := int(ry) + 2
		best := math.Inf(-1)
		for ty := 6; ty < H-6; ty += step {
			for tx := 6; tx < W-6; tx += step {
				i := ty*W + tx
				cx, cy := float64(tx)+0.5, float64(ty)+0.5
				if im[i] != int16(isl) || room.at(tx, ty) < need || w.tiles[i] != TileForest ||
					math.Hypot(cx-w.spawnX, cy-w.spawnY) < size/5 || w.nearVillage(cx, cy, rx+8) {
					continue
				}
				score, _ := w.lairScore(CreatureTitanoboa, tx, ty, nil, 0, 0)
				score += 0.4*float64(min(int(inland[i]), 50)) + unit(hash2(int32(tx), int32(ty), uint32(w.seed)^0x7e3a1e))
				if score > best && fits(tx, ty, rx, ry) {
					best, x, y, k, ok = score, cx, cy, kk, true
				}
			}
		}
		if ok {
			return x, y, k, true
		}
	}
	return 0, 0, 0, false
}

// roomMap is flatRoom over a window of the map: how many steps (of eight
// ways) each tile of it lies from the nearest tile a courtyard cannot be
// laid over, up to most; the window's own edge counts as one.
type roomMap struct {
	x0, y0, w, h int
	d            []uint8
}

// at is the room on tile (tx, ty); 0 outside the window.
func (m *roomMap) at(tx, ty int) int {
	if tx < m.x0 || ty < m.y0 || tx >= m.x0+m.w || ty >= m.y0+m.h {
		return 0
	}
	return int(m.d[(ty-m.y0)*m.w+tx-m.x0])
}

// flatRoom is, for the tiles of the surface in [x0, x1) x [y0, y1), how
// many steps (of eight ways) each lies from the nearest tile a courtyard
// cannot be laid over: water, a river, a cliff, a ramp, a change of level,
// the window's edge. Up to most.
func (w *World) flatRoom(most, x0, y0, x1, y1 int) *roomMap {
	W, H := w.width, w.SurfaceHeight()
	x0, y0, x1, y1 = max(0, x0), max(0, y0), min(W, x1), min(H, y1)
	m := &roomMap{x0: x0, y0: y0, w: max(0, x1-x0), h: max(0, y1-y0)}
	m.d = make([]uint8, m.w*m.h)
	var q []int32
	bad := func(i int) bool {
		t := w.tiles[i]
		if !t.Walkable() || t == TileFord || t == TileSand {
			return true
		}
		if w.relief == nil {
			return false
		}
		r := w.relief[i]
		if r.Variant() == VarRamp {
			return true
		}
		x, y := i%W, i/W
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+n[0], y+n[1]
			if nx >= 0 && ny >= 0 && nx < W && ny < H && w.relief[ny*W+nx].Level() != r.Level() {
				return true
			}
		}
		return false
	}
	for j := range m.d {
		x, y := j%m.w, j/m.w
		if x == 0 || y == 0 || x == m.w-1 || y == m.h-1 || bad((y+y0)*W+x+x0) {
			q = append(q, int32(j))
		} else {
			m.d[j] = uint8(most)
		}
	}
	for k := 0; k < len(q); k++ {
		j := int(q[k])
		x, y := j%m.w, j/m.w
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= m.w || ny >= m.h {
					continue
				}
				if o := ny*m.w + nx; m.d[o] > m.d[j]+1 {
					m.d[o] = m.d[j] + 1
					q = append(q, int32(o))
				}
			}
		}
	}
	return m
}

// kept reports whether entity kind k is something a lair is never laid
// over: a way into the caves, a village's huts, the pyramid, a treasure.
func kept(k EntityKind) bool {
	switch k {
	case EntityCaveMouth, EntityCaveExit, EntityCaveRock, EntityCaveDown, EntityCaveUp, EntityHut, EntityTotem,
		EntityBonfire, EntityPyramid, EntityPyramidStone, EntityTreasure:
		return true
	}
	return false
}

// keptNear reports whether anything kept stands within r tiles of (tx, ty).
func (w *World) keptNear(tx, ty, r int) bool {
	for y := ty - r; y <= ty+r; y++ {
		for x := tx - r; x <= tx+r; x++ {
			if e := w.entityAt(x, y); e != nil && kept(e.Kind) {
				return true
			}
		}
	}
	return false
}

// clearFor takes away whatever grows or lies on tile (tx, ty): a giant
// tree goes whole. Nothing kept is touched.
func (w *World) clearFor(tx, ty int) {
	e := w.entityAt(tx, ty)
	if e == nil || kept(e.Kind) {
		return
	}
	if e.Kind == EntityGiantRoot || e.Kind == EntityGiantTree {
		w.felledGiantAt(tx, ty)
		return
	}
	w.removeEntity(e)
}

// propAt sets lair prop v on tile (tx, ty) whatever lies there (but what is
// kept), and reports whether it could.
func (w *World) propAt(v uint8, tx, ty int) bool {
	if !w.inBounds(tx, ty) || !w.tiles[ty*w.width+tx].Walkable() {
		return false
	}
	if e := w.entityAt(tx, ty); e != nil {
		if kept(e.Kind) || e.Kind == EntityLairProp {
			return false
		}
		w.clearFor(tx, ty)
	}
	w.addEntity(EntityLairProp, tx, ty).Var = v
	return true
}

// inPool reports whether (x, y) lies in the black pool of site s.
func (s *lairSite) inPool(x, y float64) bool {
	if s.layout != LayoutTemple {
		return false
	}
	dx, dy := (x-s.poolX)/s.poolRX, (y-s.poolY)/s.poolRY
	return dx*dx+dy*dy <= 1
}

// titanPoolAt reports whether tile (tx, ty) is the water of the temple's
// pool: ground to Titanoboa, which lies and swims in it.
func (w *World) titanPoolAt(tx, ty int) bool {
	s := &w.bosses.lairs[0].site
	return s.layout == LayoutTemple && s.inPool(float64(tx)+0.5, float64(ty)+0.5)
}

// buildTemple lays out the temple round lair l, a courtyard of radius r,
// and its camp and the trail between them.
func (w *World) buildTemple(l *bossLair, k float64) {
	s := &l.site
	rx, ry := TempleRX*k, TempleRY*k
	*s = lairSite{layout: LayoutTemple, r: max(rx, ry), rx: rx, ry: ry, poolX: l.x, poolY: l.y, poolRX: templePoolRX * k, poolRY: templePoolRY * k}
	seed := uint32(w.seed) ^ 0x7e3b1e
	h := func(a, b int) float64 { return unit(hash2(int32(a), int32(b), seed)) }
	cx, cy := int(math.Floor(l.x)), int(math.Floor(l.y))
	R := int(math.Ceil(s.r)) + 2

	// The camp first: the gate faces it.
	s.campX, s.campY, s.camp = w.campSite(l.x, l.y, campNear, campFar)
	gate := math.Atan2(w.spawnY-l.y, w.spawnX-l.x)
	if s.camp {
		gate = math.Atan2(s.campY-l.y, s.campX-l.x)
	}
	// at is the point at q (1: the edge) out from the middle along angle a.
	at := func(a, q float64) (int, int) {
		d := s.edge(a) * q
		return int(math.Floor(l.x + math.Cos(a)*d)), int(math.Floor(l.y + math.Sin(a)*d))
	}

	// The courtyard: paved (the client draws the flags on the path), the
	// pool in its middle.
	for ty := cy - R; ty <= cy+R; ty++ {
		for tx := cx - R; tx <= cx+R; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			fx, fy := float64(tx)+0.5, float64(ty)+0.5
			if s.q(fx-l.x, fy-l.y) > 1+0.5/ry {
				continue
			}
			w.clearFor(tx, ty)
			i := ty*w.width + tx
			if w.entityAt(tx, ty) != nil {
				continue // something kept: leave its tile be
			}
			if s.inPool(fx, fy) {
				w.tiles[i] = TileWater
				if w.relief != nil {
					w.relief[i] = w.relief[i].withVariant(VarLake)
				}
				continue
			}
			w.tiles[i] = TilePath
			if w.relief != nil {
				w.relief[i] = w.relief[i].withVariant(VarPlain)
			}
		}
	}
	for _, c := range w.fauna.creatures { // nothing lives in the courtyard but the snake
		if c != l.c && s.q(c.X-l.x, c.Y-l.y) <= 1.1 {
			c.gone = true
		}
	}
	w.fauna.shallow = nil

	free := func(tx, ty int) bool {
		if !w.inBounds(tx, ty) || w.tiles[ty*w.width+tx] != TilePath || w.entityAt(tx, ty) != nil {
			return false
		}
		return !s.inPool(float64(tx)+0.5, float64(ty)+0.5)
	}
	nearGate := func(a float64) bool { return math.Abs(angDiff(a, gate)) < 0.38 }

	// The dead tree, across the courtyard from the gate: its foot three
	// tiles wide and two deep, roots gripping the colonnade.
	if tx, ty := at(gate+math.Pi, templeTreeQ); free(tx, ty) {
		w.addEntity(EntityLairProp, tx, ty).Var = LairDeadTree
		for _, o := range [][2]int{{-1, 0}, {1, 0}, {-1, -1}, {0, -1}, {1, -1}} {
			if free(tx+o[0], ty+o[1]) {
				w.addEntity(EntityLairProp, tx+o[0], ty+o[1]).Var = LairTreeRoot
			}
		}
		s.markX, s.markY = float64(tx)+0.5, float64(ty)+0.5
	}

	// The colonnade: pillars round the courtyard near its edge, broken off,
	// some gone, none in the gate.
	for i := range templeColumns {
		a := gate + (float64(i)+0.5)*2*math.Pi/templeColumns
		if nearGate(a) || h(i, 3) < 0.22 {
			continue
		}
		if px, py := at(a, templeRingQ); free(px, py) {
			w.addEntity(EntityLairProp, px, py).Var = LairPillar
		}
	}

	// Columns fallen across the courtyard, lying three tiles long: cover.
	placed := 0
	for i := 0; i < 4*templeFallen && placed < templeFallen; i++ {
		a := gate + 0.55 + (float64(placed)+h(i, 7)*0.6)*(2*math.Pi-1.1)/templeFallen
		pq := 0.36 + h(i, 8)*0.34
		mx, my := at(a, pq)
		if s.inPool(float64(mx)+0.5, float64(my)+0.5) || math.Hypot((float64(mx)+0.5-l.x)/(s.poolRX+2.5), (float64(my)+0.5-l.y)/(s.poolRY+2)) < 1 {
			continue
		}
		ok := true
		for dy := -1; dy <= 1 && ok; dy++ { // a tile of paving all round it
			for dx := -2; dx <= 2; dx++ {
				if !free(mx+dx, my+dy) {
					ok = false
					break
				}
			}
		}
		if !ok {
			continue
		}
		w.addEntity(EntityLairProp, mx-1, my).Var = LairColumnL
		w.addEntity(EntityLairProp, mx, my).Var = LairColumnM
		w.addEntity(EntityLairProp, mx+1, my).Var = LairColumnR
		placed++
	}

	// Two stone serpents' heads at the pool's edge, left and right.
	for _, side := range []float64{-1, 1} {
		sx, sy := int(math.Floor(l.x+side*(s.poolRX+0.7))), int(math.Floor(l.y-0.3))
		if free(sx, sy) {
			w.addEntity(EntityLairProp, sx, sy).Var = LairSerpent
		}
	}
	// What is left of its meals, and a skin, about the courtyard's edge.
	for i := range 9 {
		a := gate + 0.5 + h(i, 11)*(2*math.Pi-1)
		if bx, by := at(a, 0.62+h(i, 12)*0.2); free(bx, by) {
			v := LairBones
			if i == 2 || i == 5 {
				v = LairSkin
			}
			w.addEntity(EntityLairProp, bx, by).Var = v
		}
	}

	w.templeTrail(l, gate)
	if s.camp {
		w.buildCamp(s.campX, s.campY, l.x, l.y)
	}
	w.wx.pondDone = false // the pool is new water
}

// templeTrail beats a trail from the courtyard's gate to the camp (or out
// towards the spawn), and lays the snake's signs along it and round about.
func (w *World) templeTrail(l *bossLair, gate float64) {
	s := &l.site
	seed := uint32(w.seed) ^ 0x7e3b2a
	h := func(a, b int) float64 { return unit(hash2(int32(a), int32(b), seed)) }
	ex, ey := l.x+math.Cos(gate)*(campNear+5), l.y+math.Sin(gate)*(campNear+5)
	if s.camp {
		ex, ey = s.campX, s.campY
	}
	ge := s.edge(gate) - 1
	sx, sy := l.x+math.Cos(gate)*ge, l.y+math.Sin(gate)*ge
	n := int(math.Hypot(ex-sx, ey-sy) * 2)
	nx, ny := -(ey - sy), ex-sx // across the way, for its wandering
	if d := math.Hypot(nx, ny); d > 0 {
		nx, ny = nx/d, ny/d
	}
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(max(1, n))
		wob := math.Sin(t*math.Pi*2.3+h(1, 1)*6) * 2.2 * math.Sin(t*math.Pi)
		x, y := sx+(ex-sx)*t+nx*wob, sy+(ey-sy)*t+ny*wob
		for _, o := range [][2]float64{{0, 0}, {0.5, 0.3}, {-0.4, 0.4}} {
			tx, ty := int(math.Floor(x+o[0])), int(math.Floor(y+o[1]))
			if !w.inBounds(tx, ty) {
				continue
			}
			i := ty*w.width + tx
			if t := w.tiles[i]; t != TileForest && t != TileGrass {
				continue
			}
			if e := w.entityAt(tx, ty); e != nil && (kept(e.Kind) || e.Kind == EntityLairProp) {
				continue
			}
			w.clearFor(tx, ty)
			w.tiles[i] = TilePath
		}
		// Along it: the signs of the snake, thicker towards the temple.
		if i%3 == 0 && h(i, 2) < 0.2+0.4*(1-t) {
			side := 1.6
			if h(i, 3) < 0.5 {
				side = -1.6
			}
			bx, by := int(math.Floor(x+nx*side)), int(math.Floor(y+ny*side))
			v := LairBones
			switch k := h(i, 4); {
			case k < 0.2:
				v = LairSkin
			case k < 0.45:
				v = LairTrail
			}
			if e := w.entityAt(bx, by); e == nil || wild(e.Kind) {
				w.lairPut(v, bx, by)
			}
		}
	}
	// Trees it broke, on either side of the way in.
	TR := int(s.r) + 12
	for ty := int(l.y) - TR; ty <= int(l.y)+TR; ty++ {
		for tx := int(l.x) - TR; tx <= int(l.x)+TR; tx++ {
			e := w.entityAt(tx, ty)
			if e == nil || e.Kind != EntityTree {
				continue
			}
			dx, dy := float64(tx)+0.5-l.x, float64(ty)+0.5-l.y
			if e := s.edge(math.Atan2(dy, dx)); math.Hypot(dx, dy) < e+1 || math.Hypot(dx, dy) > e+11 {
				continue
			}
			if h(tx, ty) < 0.16 {
				w.lairPut(LairStump, tx, ty)
			}
		}
	}
}

// campSite finds a camp between near and far tiles from (x, y): flat
// ground on the same land, by a stream if one runs there, on the side the
// spawn lies. ok is false where there is none.
func (w *World) campSite(x, y, near, far float64) (cx, cy float64, ok bool) {
	W, H := w.width, w.SurfaceHeight()
	R0 := int(far) + campRoom + 4
	room := w.flatRoom(campRoom+2, int(x)-R0, int(y)-R0, int(x)+R0+1, int(y)+R0+1)
	home := math.Atan2(w.spawnY-y, w.spawnX-x)
	best := math.Inf(-1)
	R := int(far) + 1
	for _, need := range []int{campRoom + 1, 3} {
		for ty := int(y) - R; ty <= int(y)+R; ty++ {
			for tx := int(x) - R; tx <= int(x)+R; tx++ {
				if tx < 2 || ty < 2 || tx >= W-2 || ty >= H-2 {
					continue
				}
				fx, fy := float64(tx)+0.5, float64(ty)+0.5
				d := math.Hypot(fx-x, fy-y)
				if d < near || d > far || room.at(tx, ty) < need || w.keptNear(tx, ty, campRoom+1) || w.nearVillage(fx, fy, 10) {
					continue
				}
				score := 3 * math.Cos(angDiff(math.Atan2(fy-y, fx-x), home))
				if w.streamNear(tx, ty, campRoom+2) {
					score += 6 // by a stream
				}
				score += unit(hash2(int32(tx), int32(ty), uint32(w.seed)^0xca4f))
				if score > best {
					best, cx, cy, ok = score, fx, fy, true
				}
			}
		}
		if ok {
			return cx, cy, true
		}
	}
	return 0, 0, false
}

// streamNear reports whether fresh running water or a lake lies within r of
// tile (tx, ty).
func (w *World) streamNear(tx, ty, r int) bool {
	for y := ty - r; y <= ty+r; y++ {
		for x := tx - r; x <= tx+r; x++ {
			if !w.inBounds(x, y) {
				continue
			}
			switch w.tiles[y*w.width+x] {
			case TileRiver, TileFord:
				return true
			case TileWater:
				if w.relief != nil && w.relief[y*w.width+x].Variant() == VarLake {
					return true
				}
			}
		}
	}
	return false
}

// buildCamp clears the camp at (x, y) (the lair at (lx, ly)): open ground,
// an old ring of stones where a fire was and a log to sit on, and room for
// a fire and a leaf tent between them.
func (w *World) buildCamp(x, y, lx, ly float64) {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - campRoom; ty <= cy+campRoom; ty++ {
		for tx := cx - campRoom; tx <= cx+campRoom; tx++ {
			d := math.Hypot(float64(tx-cx), float64(ty-cy)*1.2)
			if d > float64(campRoom)+0.3 || !w.inBounds(tx, ty) {
				continue
			}
			w.clearFor(tx, ty)
			// The ground of the camp itself trodden bare, where there is grass
			// or woods to tread (not the cay's sand).
			if i := ty*w.width + tx; d <= 2.6+unit(hash2(int32(tx), int32(ty), uint32(w.seed)^0xca9))*0.8 {
				if t := w.tiles[i]; t == TileGrass || t == TileForest {
					w.tiles[i] = TilePath
				}
			}
		}
	}
	// The ring on the lair's side, the log behind it; the middle stays open.
	side := 1
	if lx < x {
		side = -1
	}
	w.propAt(LairCampRing, cx+2*side, cy+1)
	w.propAt(LairCampLog, cx+2*side, cy-1)
}

// lairReserved reports whether tile (tx, ty) is a lair's own ground since
// StyleLairs, where the world's scatter puts nothing more: the temple's
// courtyard, the atoll, and the camps.
func (w *World) lairReserved(tx, ty int) bool {
	fx, fy := float64(tx)+0.5, float64(ty)+0.5
	for i := range w.bosses.lairs {
		l := &w.bosses.lairs[i]
		s := &l.site
		if !l.ok || s.layout == LayoutHollow {
			continue
		}
		if math.Abs(fx-l.x) <= s.r+2 && math.Abs(fy-l.y) <= s.r+2 && s.q(fx-l.x, fy-l.y) <= 1+1.5/min(s.rx, s.ry) {
			return true
		}
		if s.camp && math.Hypot(fx-s.campX, fy-s.campY) <= campRoom+1 {
			return true
		}
	}
	return false
}
