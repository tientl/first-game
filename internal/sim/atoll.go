package sim

import "math"

// The Megalodon's atoll, in worlds made since StyleLairs. Out in the open
// sea kept for it (deepBasin), far from any coast, a broken ring of reef
// stands round a deep lagoon about thirty-two tiles across. The reef is
// walked on (a flat of coral rock awash: TileSand with VarReef) and broken
// by three channels a boat or a swimmer can pass; small sandbars rise along
// it, one with a whale's bones and a palm. A wreck lies stranded across the
// reef on its far side: its hull is a deck to stand on, its mast the mark
// seen from far off, seabirds wheeling over it. Outside the ring the reef
// shelves off in shallows to the deep. Thirty-odd tiles towards home a
// little cay of sand with a palm is the camp.
//
// The shark lives in the lagoon and cannot leave it (it keeps to deep
// water: see creatureFits). It keeps after whoever comes to its atoll,
// standing on the reef or not; it cannot reach dry land, but it lunges at
// whoever stands at the water's edge, rams the reef to break the footholds
// under them (breakReef: water for a while, then the reef settles back),
// overturns boats that come into the lagoon and leaps clear of the water to
// come down in a heaving splash. See sharkfight.go.

// Atoll tuning, in tiles.
const (
	AtollLagoon   = 19.0 // the deep lagoon's radius: 38 tiles across, more than a screen
	AtollReef     = 3.6  // the reef ring's width
	atollShelf    = 3.4  // the shallows outside it
	atollChannels = 3    // gaps in the ring, of...
	atollGap      = 2.6  // ...this width, about
	atollBars     = 4    // sandbars along the ring
	atollCay      = 39.0 // the camp's cay lies this far from the middle
	atollCayR     = 3.6  // and is this big
	wreckLen      = 8    // the wreck's hull: tiles long...
	wreckBeam     = 3    // ...and wide
)

// atollRing is where the reef ring's middle runs.
func atollRing() float64 { return AtollLagoon + AtollReef/2 }

// buildAtoll raises the atoll round lair l, out of the open sea.
func (w *World) buildAtoll(l *bossLair) {
	s := &l.site
	r := AtollLagoon + AtollReef + 1.5
	*s = lairSite{layout: LayoutAtoll, lagoon: AtollLagoon, r: r, rx: r, ry: r, reefAt: map[int32]bool{}}
	seed := uint32(w.seed) ^ 0xa7011
	h := func(a, b int) float64 { return unit(hash2(int32(a), int32(b), seed)) }
	home := math.Atan2(w.spawnY-l.y, w.spawnX-l.x)
	inner, outer := AtollLagoon, AtollLagoon+AtollReef
	shelf := outer + atollShelf

	// The channels, spread round the ring, one of them towards home.
	var gaps [atollChannels]float64
	for i := range gaps {
		gaps[i] = home + float64(i)*2*math.Pi/atollChannels + (h(i, 1)-0.5)*0.6
	}
	inGap := func(a, r float64) bool {
		for _, g := range gaps {
			if math.Abs(angDiff(a, g))*r < atollGap/2 {
				return true
			}
		}
		return false
	}
	// The sandbars, between the channels.
	var bars [atollBars]float64
	for i := range bars {
		bars[i] = home + math.Pi/atollChannels + float64(i)*2*math.Pi/atollBars + (h(i, 2)-0.5)*0.4
	}
	inBar := func(a, r float64) bool {
		for i, b := range bars {
			if math.Abs(angDiff(a, b))*r < 2.6+h(i, 3)*2 {
				return true
			}
		}
		return false
	}
	set := func(i int, t Tile, v GroundVariant) {
		w.tiles[i] = t
		if w.relief != nil {
			w.relief[i] = w.relief[i].withLevel(0).withVariant(v)
		}
	}
	R := int(shelf) + 2
	cx, cy := int(math.Floor(l.x)), int(math.Floor(l.y))
	for ty := cy - R; ty <= cy+R; ty++ {
		for tx := cx - R; tx <= cx+R; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			fx, fy := float64(tx)+0.5, float64(ty)+0.5
			a := math.Atan2(fy-l.y, fx-l.x)
			// The ring's edges wander a little, so it reads as grown, not drawn.
			wob := 0.55*math.Sin(a*5+h(0, 4)*6) + 0.35*math.Sin(a*11+h(0, 5)*6)
			d := math.Hypot(fx-l.x, fy-l.y) - wob
			i := ty*w.width + tx
			w.clearFor(tx, ty)
			switch {
			case d < inner:
				set(i, TileWater, VarPlain) // the deep lagoon
			case d < outer && inGap(a, d):
				set(i, TileShallow, VarReef) // a channel
			case d < outer && inBar(a, d) && d > inner+0.6 && d < outer-0.4:
				set(i, TileSand, VarDunes) // a sandbar
			case d < outer:
				set(i, TileSand, VarReef) // the reef flat: a foothold the shark can break
				s.reef = append(s.reef, int32(i))
			case d < shelf+h(tx, ty)*0.8:
				set(i, TileShallow, VarReef)
			}
		}
	}

	// The wreck, across the reef on its far side from home.
	wa := home + math.Pi
	mx, my := l.x+math.Cos(wa)*atollRing(), l.y+math.Sin(wa)*atollRing()
	hx0, hy0 := int(math.Floor(mx))-wreckLen/2, int(math.Floor(my))-wreckBeam/2
	for ty := hy0; ty < hy0+wreckBeam; ty++ {
		for tx := hx0; tx < hx0+wreckLen; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			i := ty*w.width + tx
			set(i, TileSand, VarReef) // the deck stands on the reef: walked on, never broken
			w.clearFor(tx, ty)
		}
	}
	wx, wy := hx0+wreckLen/2, hy0+wreckBeam/2
	for ty := hy0; ty < hy0+wreckBeam; ty++ {
		for tx := hx0; tx < hx0+wreckLen; tx++ {
			switch {
			case tx == wx && ty == wy:
				w.addEntity(EntityLairProp, tx, ty).Var = LairWreck
			case tx == wx+2 && ty == wy:
				w.addEntity(EntityLairProp, tx, ty).Var = LairMast
			default:
				w.addEntity(EntityLairProp, tx, ty).Var = LairWreckPart
			}
		}
	}
	s.markX, s.markY = float64(wx+2)+0.5, float64(wy)+0.5
	for _, i := range s.reef {
		if x, y := int(i)%w.width, int(i)/w.width; x >= hx0 && x < hx0+wreckLen && y >= hy0 && y < hy0+wreckBeam {
			continue
		}
		s.reefAt[i] = true
	}
	s.reef = s.reef[:0]
	for ty := cy - R; ty <= cy+R; ty++ { // in reading order, so the list is the same every time
		for tx := cx - R; tx <= cx+R; tx++ {
			if w.inBounds(tx, ty) && s.reefAt[int32(ty*w.width+tx)] {
				s.reef = append(s.reef, int32(ty*w.width+tx))
			}
		}
	}

	// On the biggest sandbar, the whale's bones and a palm.
	for i, b := range bars {
		bx, by := int(math.Floor(l.x+math.Cos(b)*atollRing())), int(math.Floor(l.y+math.Sin(b)*atollRing()))
		if w.TileAt(bx, by) != TileSand || w.entityAt(bx, by) != nil {
			continue
		}
		switch i {
		case 0:
			w.addEntity(EntityLairProp, bx, by).Var = LairWhale
			l.bar, l.barX, l.barY = true, bx, by
		case 2:
			w.addEntity(EntityLairProp, bx, by).Var = LairPalm
		}
	}

	// The camp's cay, towards home: the first angle out from home where the
	// sea round it is open.
	for k := range 7 {
		a := home + float64((k+1)/2)*0.45*float64(1-2*(k%2))
		x, y := l.x+math.Cos(a)*atollCay, l.y+math.Sin(a)*atollCay
		if !w.openSea(x, y, atollCayR+4) {
			continue
		}
		w.raiseCay(x, y)
		s.camp, s.campX, s.campY = true, math.Floor(x)+0.5, math.Floor(y)+0.5
		w.buildCamp(s.campX, s.campY, l.x, l.y)
		w.propAt(LairPalm, int(math.Floor(x))-1, int(math.Floor(y))-2)
		break
	}

	for _, c := range w.fauna.creatures { // nothing lives in the lagoon but the shark
		if c != l.c && math.Hypot(c.X-l.x, c.Y-l.y) <= shelf+2 {
			c.gone = true
		}
	}
	w.fauna.shallow = nil
	w.wx.pondDone = false
}

// openSea reports whether everything within r of (x, y) is deep water.
func (w *World) openSea(x, y, r float64) bool {
	R := int(math.Ceil(r))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - R; ty <= cy+R; ty++ {
		for tx := cx - R; tx <= cx+R; tx++ {
			if math.Hypot(float64(tx-cx), float64(ty-cy)) <= r && (!w.inBounds(tx, ty) || ty >= w.SurfaceHeight() || w.tiles[ty*w.width+tx] != TileWater) {
				return false
			}
		}
	}
	return true
}

// raiseCay raises the little cay of sand at (x, y), with its shallows.
func (w *World) raiseCay(x, y float64) {
	R := int(math.Ceil(atollCayR + 3))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	seed := uint32(w.seed) ^ 0xca7
	for ty := cy - R; ty <= cy+R; ty++ {
		for tx := cx - R; tx <= cx+R; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			d := math.Hypot(float64(tx-cx), float64(ty-cy)*1.15) + (unit(hash2(int32(tx), int32(ty), seed))-0.5)*0.6
			i := ty*w.width + tx
			switch {
			case d <= atollCayR:
				w.tiles[i] = TileSand
				if w.relief != nil {
					w.relief[i] = w.relief[i].withLevel(0).withVariant(VarPlain)
				}
			case d <= atollCayR+2:
				w.tiles[i] = TileShallow
				if w.relief != nil {
					w.relief[i] = w.relief[i].withLevel(0).withVariant(VarShoal)
				}
			}
		}
	}
}

// reefTile reports whether tile i is one of lair l's footholds of reef.
func (l *bossLair) reefTile(i int) bool { return l.site.reefAt[int32(i)] }

// breakReef has the shark of lair l break the footholds of reef within r of
// (x, y): they are water till they settle back, ReefDownTicks later. It
// reports how many broke.
func (w *World) breakReef(l *bossLair, x, y, r float64) int {
	n := 0
	R := int(math.Ceil(r)) + 1
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - R; ty <= cy+R; ty++ {
		for tx := cx - R; tx <= cx+R; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			i := ty*w.width + tx
			fx, fy := float64(tx)+0.5, float64(ty)+0.5
			if !l.reefTile(i) || w.tiles[i] != TileSand || math.Hypot(fx-x, fy-y) > r {
				continue
			}
			if e := w.entityAt(tx, ty); e != nil {
				continue // something stands on it: it holds
			}
			w.tiles[i] = TileShallow
			w.addLairHazard(LairHazard{Kind: HazardReef, X: fx, Y: fy, R: 0.5, Ends: w.tick + ReefDownTicks})
			n++
		}
	}
	if n > 0 {
		w.fauna.shallow = nil
	}
	return n
}

// settleReef is broken reef h settling back: walkable again.
func (w *World) settleReef(h LairHazard) {
	tx, ty := int(math.Floor(h.X)), int(math.Floor(h.Y))
	if !w.inBounds(tx, ty) {
		return
	}
	if i := ty*w.width + tx; w.tiles[i] == TileShallow {
		w.tiles[i] = TileSand
		w.fauna.shallow = nil
	}
}

// reefWhole sets the broken footholds of every atoll back in t, a copy of
// the tiles: as they will be once they settle.
func (w *World) reefWhole(t []Tile) {
	for _, h := range w.hz.list {
		if h.Kind != HazardReef {
			continue
		}
		if tx, ty := int(math.Floor(h.X)), int(math.Floor(h.Y)); w.inBounds(tx, ty) {
			t[ty*w.width+tx] = TileSand
		}
	}
}

// ReefBroken reports whether a foothold of reef is broken on tile (tx,
// ty) now: water until it settles.
func (w *World) ReefBroken(tx, ty int) bool {
	for _, h := range w.hz.list {
		if h.Kind == HazardReef && int(math.Floor(h.X)) == tx && int(math.Floor(h.Y)) == ty {
			return true
		}
	}
	return false
}
