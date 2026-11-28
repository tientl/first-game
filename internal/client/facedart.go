package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// What turns, drawn each way it can face (sim/facing.go). Facing south is
// the drawing everything had before; each other facing is drawn in the
// same HD style: a thing seen from behind (a door's bar, a chest's
// hinges, a tent's closed end) or from the side has a drawing of its own,
// and one that differs only left from right is that drawing mirrored. Each
// is drawn the first time it is shown. Walls and fences, which also join,
// are in wallrun.go; the tent in tentart.go, the furnace in rockart.go,
// the cooking gear in cookart.go.

// flipCanvas is c mirrored left to right.
func flipCanvas(c *canvas) *canvas {
	f := newCanvas(c.w, c.h)
	f.dens, f.s = c.dens, c.s
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			f.px[y*c.w+c.w-1-x] = c.px[y*c.w+x]
		}
	}
	if c.ol != nil {
		f.ol = make([]bool, len(c.ol))
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				f.ol[y*c.w+c.w-1-x] = c.ol[y*c.w+x]
			}
		}
	}
	return f
}

// mkFlipped is mkUp of c mirrored, its anchor (ax, ay), image pixels of an
// HD canvas, mirrored with it.
func mkFlipped(c *canvas, ax, ay float64) sprite {
	return mkUp(flipCanvas(c), float64(c.w)-ax, ay)
}

// facedKey names one drawing of a kind facing a way, in one state.
type facedKey struct {
	k     sim.EntityKind
	face  sim.Dir
	state uint8
}

var facedCache = map[facedKey]sprite{}

// faced is the drawing named by key, made by mk the first time.
func faced(key facedKey, mk func() sprite) sprite {
	if s, ok := facedCache[key]; ok {
		return s
	}
	s := mk()
	facedCache[key] = s
	return s
}

// facedSprite is e's sprite drawn the way it faces, for the kinds that
// turn and are drawn as one sprite; ok is false for the rest, and for
// those facing south, which keep their old drawing (walls and fences
// excepted: they join).
func (g *Game) facedSprite(e protocol.EntityView) (sprite, bool) {
	if isRun(e.Kind) {
		return g.runSprite(e), true // see wallrun.go
	}
	if e.Face == sim.DirDown || !sim.Turns(e.Kind, e.Var) {
		return sprite{}, false
	}
	face := sim.FacingOf(e.Kind, e.Var, e.Face)
	key := facedKey{k: e.Kind, face: face}
	switch e.Kind {
	case sim.EntityDoor:
		return faced(key, func() sprite { return doorFacedHD(face) }), true
	case sim.EntityBed:
		return faced(key, func() sprite { return bedFacedHD(face) }), true
	case sim.EntityChest:
		open := g.overlay == overlayChest && g.curr.Self.Chest == e.ID
		if open {
			key.state = 1
		}
		return faced(key, func() sprite { return chestFacedHD(face, open) }), true
	case sim.EntityFurnace:
		if e.Fuel > 0 {
			key.state = 1
		}
		return faced(key, func() sprite { return furnaceFacedHD(face, key.state == 1) }), true // see rockart.go
	case sim.EntityMortar, sim.EntityOven, sim.EntityDryRack:
		return g.cookFacedSprite(e, face), true // see cookart.go
	}
	return sprite{}, false
}

// facedShadow is the width of the shadow of e where its facing changes it.
func (g *Game) facedShadow(e protocol.EntityView) (float64, bool) {
	switch {
	case isRun(e.Kind):
		return g.runShadow(e), true
	case e.Kind == sim.EntityDoor && sim.RunsNS(e.Face):
		return 10, true
	case e.Kind == sim.EntityBed && sim.RunsNS(e.Face):
		return 18, true
	case e.Kind == sim.EntityChest && sim.RunsNS(e.Face):
		return 15, true
	case e.Kind == sim.EntityDryRack && sim.RunsNS(e.Face):
		return 14, true
	}
	return 0, false
}

// drawFacedEntity draws the things that turn and are more than one
// sprite, and reports whether it drew e: a head on a post looking left.
func (g *Game) drawFacedEntity(dst drawTarget, e protocol.EntityView, d drawable) bool {
	if e.Kind != sim.EntityTrophy || sim.FacingOf(e.Kind, e.Var, e.Face) != sim.DirLeft {
		return false
	}
	a := &g.art.tribal
	drawAt(dst, a.post, d.sx, d.sy, 1, 1, 0, nil)
	if h, ok := g.art.heads[sim.CreatureKind(e.Var)]; ok {
		k := min(1, 22/max(h.w(), h.h()))
		drawAt(dst, h, d.sx, d.sy-a.post.ay+5, -k, k, 0, nil) // the profile, mirrored
	}
	return true
}

// --- the door ---------------------------------------------------------------

// doorFacedHD is the door facing face: from behind, with its bar; or in a
// wall running north–south, seen end on, standing a little open towards
// the side it faces so its face shows.
func doorFacedHD(face sim.Dir) sprite {
	const k = HDDensity
	switch face {
	case sim.DirUp:
		return mkUp(doorCanvasHD(true), 12*k, (wallArtH+4)*k)
	case sim.DirLeft:
		c, ax, ay := doorSideHD()
		return mkFlipped(c, ax, ay)
	}
	c, ax, ay := doorSideHD()
	return mkUp(c, ax, ay)
}

// doorSideHD is the door facing east in a wall running north–south: its
// two posts one behind the other, the lintel going back between their
// tops, and the door hung on the far post, swung a little open towards
// the east, so its planks, its battens and its ring show.
func doorSideHD() (*canvas, float64, float64) {
	const k = HDDensity
	c := hdCanvas(runW, runH)
	const postW = 3.2 * k
	backY, frontY := -7.4, 7.4 // the posts' feet, art pixels from the middle
	top := float64(wallArtH) - 0.5
	post := func(y float64, lit float64, seed uint32) {
		x, f := runPx(0, y)
		logHD(c, int(x-postW/2), int(x+postW/2), int(f-top*k), int(f)+k-1, trunkRamp, lit, seed)
	}
	post(backY, 0.72, 950)
	// The door: planks hung on the far post, its free edge swung out to
	// the east, in front of the gap; light falls on its face from the left
	// and behind, so it is in half shade.
	const open = 0.42 // radians
	const panelL = 13.6 // its width, art pixels on the ground
	hx, hf := runPx(0.9, backY+1.2)
	ex := hx + panelL*math.Sin(open)*k
	ef := hf + panelL*math.Cos(open)*0.75*k
	ph := (float64(wallArtH) - 4) * k // its height
	minX, maxX := int(math.Min(hx, ex))-1, int(math.Max(hx, ex))+2
	for x := minX; x <= maxX; x++ {
		t := (float64(x) - hx) / (ex - hx) // 0 at the hinge, 1 at the free edge
		if t < 0 || t > 1 {
			continue
		}
		foot := hf + (ef-hf)*t
		plank := int(t * 4)
		pt := t*4 - float64(plank)
		for y := int(foot - ph); y <= int(foot); y++ {
			up := (foot - float64(y)) / ph // 0 at the foot, 1 at the top
			l := 0.42 + float64(plank%2)*0.05 + (hashf(plank, 0, 951)-0.5)*0.08 - t*0.06
			switch {
			case pt < 0.12:
				l -= 0.22
			case pt < 0.24:
				l += 0.08
			}
			// The battens across, at the heights they are at the front.
			for _, b := range []float64{8, 22, wallArtH - 2} {
				bh := 1 - b/(wallArtH+4) // its height up the door
				if math.Abs(up-bh) < 1.0/(wallArtH-4) {
					l = 0.5 - (up-bh)*8
				}
			}
			c.put(x, y, grainHD(plankRamp, x, y, l, true, 952+uint32(plank)))
		}
	}
	// The free edge: the planks' end grain, a thin pale strip.
	for y := int(ef - ph); y <= int(ef); y++ {
		c.put(int(ex)+1, y, plankRamp.at(4))
		c.put(int(ex)+2, y, plankRamp.at(1))
	}
	// The ring, near the free edge at the height of a hand.
	rx, ry := hx+(ex-hx)*0.78, hf+(ef-hf)*0.78-(wallArtH-15)*k
	c.ellipse(rx, ry, 2.2, 2.8, func(x, y int, nx, ny float64) color.RGBA { return bandRamp.pick(0.3+lum(nx, ny)*0.5, x, y) })
	for a := 0.0; a < 2*math.Pi; a += 0.12 {
		x, y := int(rx+math.Cos(a)*2.6), int(ry+2*k+math.Sin(a)*3.6)
		c.put(x, y, bandRamp.pick(0.75-math.Sin(a)*0.3, x, y))
	}
	// The lintel, from the far post's top to the near one's.
	bx, bf := runPx(0, backY)
	fx, ff := runPx(0, frontY)
	drawLimbHD(c, limb{bx, bf - top*k + 2, bx, (bf + ff) / 2 - top*k + 2, fx, ff - top*k + 2, 1.6 * k, 1.6 * k}, trunkRamp, 0.86, 953)
	post(frontY, 0.82, 954)
	// The lintel's cut end over the near post.
	c.ellipse(fx, ff-top*k+2, 1.6*k, 1.2*k, func(px, py int, nx, ny float64) color.RGBA { return woodEndHD(px, py, nx, ny, 3, 955) })
	c.outline(0.7)
	return c, runAX * k, runAY * k
}

// --- the bed ------------------------------------------------------------------

// bedHead is where the head of a bed facing face lies (sim/facing.go: the
// side climbed into faces the way it faces, the head to its left).
func bedHead(face sim.Dir) sim.Dir {
	switch face {
	case sim.DirLeft:
		return sim.DirUp
	case sim.DirUp:
		return sim.DirRight
	case sim.DirRight:
		return sim.DirDown
	}
	return sim.DirLeft
}

// bedFacedHD is the bed facing face: across with its head to the right
// (the old bed mirrored), or lengthways, its head up or down the screen.
func bedFacedHD(face sim.Dir) sprite {
	const k = HDDensity
	switch bedHead(face) {
	case sim.DirRight:
		return mkFlipped(bedCanvasHD(), 15*k, 16*k)
	case sim.DirUp, sim.DirDown:
		c, ax, ay := bedLongHD(bedHead(face) == sim.DirUp)
		return mkUp(c, ax, ay)
	}
	return mkUp(bedCanvasHD(), 15*k, 16*k)
}

// The bed lying lengthways: its size on the screen, art pixels, from the
// middle of its tile.
const (
	bedLongHW   = 7.5  // half its width
	bedLongBack = -11  // its head or foot rail, at the back
	bedLongFore = 9.0  // its rail at the front
	bedLongRail = 2.6  // a rail's thickness
	bedLongPill = -6.5 // the pillow's middle with the head up the screen (down: mirrored about the middle)
)

// bedLongHD is the bed lying north–south: the side rails going back, a
// rail across at each end, the front one showing its face, the mattress of
// dry grass and the rolled hide pillow at the head (up the screen, or down
// by the front rail). Anchored on the middle of its tile.
func bedLongHD(headUp bool) (*canvas, float64, float64) {
	const k = HDDensity
	c := hdCanvas(20, 30)
	const ax, ay = 10.0, 18.0 // the middle of the tile on the canvas, art pixels
	px := func(x, y float64) float64 { return (x + ax) * k }
	py := func(y float64) float64 { return (y + ay) * k }
	// The frame under it all: dark logs.
	for y := int(py(bedLongBack - 1.5)); y < int(py(bedLongFore+bedLongRail)); y++ {
		for x := int(px(-bedLongHW, 0)); x < int(px(bedLongHW, 0)); x++ {
			c.put(x, y, trunkRamp.pick(0.3, x, y))
		}
	}
	// The mattress: dry grass in short strokes lying every way.
	m0, m1 := py(bedLongBack+0.5), py(bedLongFore-0.4)
	for y := int(m0); y < int(m1); y++ {
		for x := int(px(-bedLongHW+bedLongRail, 0)); x < int(px(bedLongHW-bedLongRail, 0)); x++ {
			l := 0.52 + (vnoise(float64(x)*0.23, float64(y)*0.3, 920)-0.5)*0.5
			slant := 1
			if vnoise(float64(x)*0.09, float64(y)*0.11, 921) > 0.5 {
				slant = -1
			}
			switch s := hashf(x+slant*y, (x-slant*y)/9, 922); {
			case s < 0.2:
				l += 0.25
			case s > 0.84:
				l -= 0.25
			}
			if float64(x) < px(-bedLongHW+bedLongRail+1.2, 0) {
				l -= 0.18 // in the left rail's shade
			}
			c.put(x, y, strawRamp.pick(l, x, y))
		}
	}
	// The pillow: a rolled hide across the bed, with its seam and stitches.
	pyc := bedLongPill
	if !headUp {
		pyc = bedLongFore - 3.6
	}
	c.ellipse(px(0, 0), py(pyc), 4.4*k, 2.3*k, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if math.Abs(ny-0.25*math.Sin(nx*3)) < 0.09 {
			l -= 0.3
		}
		if d := nx*nx + ny*ny; d > 0.72 && d < 0.8 && (x+y)%3 != 0 {
			l -= 0.25
		}
		return mix(hex(0xb0a284), hex(0xf4eedb), l)
	})
	// The side rails, going back: lit along their tops, the left one more.
	for i, s := range []float64{-1, 1} {
		x0, x1 := px(s*bedLongHW-bedLongRail/2*(1+s), 0), px(s*bedLongHW+bedLongRail/2*(1-s), 0)
		for y := int(py(bedLongBack - 1)); y < int(py(bedLongFore)); y++ {
			for x := int(x0); x < int(x1); x++ {
				u := (float64(x)+0.5-(x0+x1)/2)/((x1-x0)/2)
				c.put(x, y, barkAt(trunkRamp, u, bedLongRail*k/2, float64(y)*1.1, x, y, 0.86-float64(i)*0.16, 918+uint32(i)))
			}
		}
	}
	// The back rail across, seen from above, and the front rail with its face.
	for y := int(py(bedLongBack - 1.6)); y < int(py(bedLongBack+0.6)); y++ {
		for x := int(px(-bedLongHW-0.6, 0)); x < int(px(bedLongHW+0.6, 0)); x++ {
			u := (float64(y)+0.5-py(bedLongBack-0.5))/(1.1*k)
			c.put(x, y, barkAt(trunkRamp, u*0.8, 3, float64(x)*1.1, x, y, 0.8, 917))
		}
	}
	for y := int(py(bedLongFore - 0.4)); y < int(py(bedLongFore+bedLongRail)); y++ {
		for x := int(px(-bedLongHW-0.6, 0)); x < int(px(bedLongHW+0.6, 0)); x++ {
			c.put(x, y, barkAt(trunkRamp, (float64(y)-py(bedLongFore-0.4))/(1.5*k)-1, 3, float64(x), x, y, 0.55, 916))
		}
	}
	// The cut ends of the side rails over the front one.
	for _, s := range []float64{-1, 1} {
		c.ellipse(px(s*(bedLongHW-bedLongRail/2), 0), py(bedLongFore+0.5), 1.4*k, 1.6*k, func(x, y int, nx, ny float64) color.RGBA {
			return woodEndHD(x, y, nx, ny, 3, 7)
		})
	}
	c.outline(0.8)
	return c, ax * k, ay * k
}

// --- the chest ------------------------------------------------------------------

// chestFacedHD is the chest facing face, shut or open: from behind (its
// hinges, the lid lifted towards the viewer when open), or end on (the
// curve of its lid, the hasp of its lock at the side it faces).
func chestFacedHD(face sim.Dir, open bool) sprite {
	const k = HDDensity
	switch face {
	case sim.DirUp:
		return mkUp(chestBackHD(open), 12*k, 20*k)
	case sim.DirLeft:
		return mkFlipped(chestEndHD(open), 12*k, 20*k)
	}
	return mkUp(chestEndHD(open), 12*k, 20*k)
}

// The chest's woods and irons, as chestui.go's.
var (
	chestWood = woodRamp[:3]
	chestLid  = trunkRamp
	chestBand = ironRamp
)

// chestBackHD is the chest seen from behind: the box, its lid with no lock
// and two iron hinges along its back edge; open, the lid stands up before
// the box, its outside to the viewer.
func chestBackHD(open bool) *canvas {
	const k = HDDensity
	if !open {
		c := chestHD(false, chestWood, chestLid, chestBand, chestBand, 27)
		// Take the lock off: the box's planks and band where it was.
		const top = 10 * k
		for y := top - 2; y < top+3*k; y++ {
			for x := 10 * k; x < 14*k+2; x++ {
				if !c.opaque(x, y) {
					continue
				}
				l := 0.72 - float64(x-2*k)/float64(20*k)*0.42
				col := grainHD(chestWood, x, y, l, false, 27)
				if y < top {
					col = grainHD(chestLid, x, y, 0.6-float64(x-2*k)/float64(20*k)*0.45, false, 30)
				}
				c.put(x, y, col)
			}
		}
		// Two hinges: straps over the lid's lip, a knuckle each.
		for _, hx := range []int{6 * k, 18 * k} {
			for y := top - 3*k; y < top+2*k; y++ {
				for x := hx - 4; x < hx+5; x++ {
					c.put(x, y, chestBand.pick(0.55-float64(x-hx)*0.05+float64(y-top)*0.01, x, y))
				}
			}
			for x := hx - 5; x < hx+6; x++ {
				c.put(x, top-1, chestBand.at(3))
				c.put(x, top, chestBand.at(1))
			}
			nailHD(c, hx-1, top-2*k)
			nailHD(c, hx-1, top+k)
		}
		c.outline(0.8)
		return c
	}
	c := hdCanvas(24, 22)
	box := chestHD(false, chestWood, chestLid, chestBand, chestBand, 27)
	// The box alone (the lid's rows left out), then the lid standing up
	// over its back edge, towards the viewer.
	const top = 10 * k
	for y := top; y < box.h; y++ {
		for x := 0; x < box.w; x++ {
			if p := box.px[y*box.w+x]; p.A > 0 {
				c.put(x, y, p)
			}
		}
	}
	for y := top - 9*k; y < top+2*k; y++ {
		f := float64(y-(top-9*k)) / float64(11*k)
		inset := int(2.2 * k * (1 - math.Sqrt(math.Max(0, 1-(1-f*3)*(1-f*3)))) * float64(btoi(f < 0.33)))
		for x := 2*k + inset; x < 22*k-inset; x++ {
			l := 0.85 - float64(x-2*k)/float64(20*k)*0.42 - f*0.2
			col := grainHD(chestLid, x, y, l, true, 31)
			if x >= 5*k && x < 7*k || x >= 17*k && x < 19*k {
				col = chestBand.pick(l+0.05, x, y)
			}
			c.put(x, y, col)
		}
	}
	c.outline(0.8)
	return c
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// chestEndHD is the chest facing east seen end on: the end of the box in
// planks with its bands round the corners, the round of the lid over it
// and its top going back, banded; the hasp of the lock stands out at the
// right. Open, the lid is thrown back to the left on its hinges and the
// dark inside shows.
func chestEndHD(open bool) *canvas {
	const k = HDDensity
	c := hdCanvas(24, 22)
	const x0, x1, top, base = 5 * k, 19 * k, 10 * k, 20*k + 2
	const back = 6 * k // how far its top reaches back up the screen
	// The box's top going back (seen over the end when open: the dark inside).
	for y := top - back; y < top; y++ {
		for x := x0; x < x1; x++ {
			l := 0.7 - float64(x-x0)/float64(x1-x0)*0.3
			col := grainHD(chestWood, x, y+40, l, true, 28)
			if open {
				col = mix(hex(0x1a0e08), hex(0x2c1a10), float64(y-(top-back))/float64(back))
				if x < x0+3 || x >= x1-3 || y < top-back+3 {
					col = grainHD(chestWood, x, y, 0.5, true, 28)
				}
			}
			c.put(x, y, col)
		}
	}
	// The end of the box: planks across, banded at both corners.
	for y := top; y <= base; y++ {
		for x := x0; x < x1; x++ {
			l := 0.62 - float64(x-x0)/float64(x1-x0)*0.3
			switch (y - top) % (3*k + 2) {
			case 3*k + 1:
				l = 0.1
			case 0:
				l += 0.1
			}
			col := grainHD(chestWood, x, y+(y-top)/(3*k+2)*50, l, false, 27)
			if x < x0+2*k || x >= x1-2*k {
				col = chestBand.pick(l+0.1, x, y)
				if (y+2)%7 == 0 && (x-x0)%(2*k) == 2 {
					col = chestBand.at(len(chestBand) - 1)
				}
			}
			c.put(x, y, col)
		}
	}
	if !open {
		// The lid: its rounded end over the box, and its top going back.
		for y := top - back - 5*k; y < top; y++ {
			for x := x0; x < x1; x++ {
				endY := top - 5*k*math.Sqrt(math.Max(0, 1-math.Pow((float64(x)+0.5-float64(x0+x1)/2)/(float64(x1-x0)/2), 2)))
				switch {
				case float64(y) >= endY: // the lid's end, a half round of planks
					l := 0.55 - float64(x-x0)/float64(x1-x0)*0.25 + (float64(y)-endY)/(5*k)*0.1
					if y == top-1 {
						l = 0.12
					}
					c.put(x, y, grainHD(chestLid, x, y, l, false, 30))
				case float64(y) >= endY-back: // its top going back, lit
					u := (float64(x) + 0.5 - float64(x0+x1)/2) / (float64(x1-x0) / 2)
					l := 0.9 - (u+1)/2*0.4
					col := grainHD(chestLid, x, y, l, true, 31)
					if by := int(float64(y) - (endY - back)); by >= back/4 && by < back/4+5 || by >= back*3/4 && by < back*3/4+5 {
						col = chestBand.pick(l, x, y) // the bands over the lid
					}
					c.put(x, y, col)
				}
			}
		}
		// The hasp: an iron tongue standing out from the front face.
		for y := top - 2; y < top+3*k; y++ {
			for x := x1; x < x1+3; x++ {
				c.put(x, y, ironRamp.pick(0.7-float64(x-x1)*0.15, x, y))
			}
		}
	} else {
		// The lid thrown back: its curve standing up behind the box, to the left.
		cx := float64(x0) + 1.5*k
		for y := top - back - 9*k; y < top-back+2; y++ {
			f := float64(y-(top-back-9*k)) / float64(9*k)
			w := 5 * k * math.Sqrt(math.Max(0, 1-math.Pow(f*2-1, 2)))
			for x := int(cx - w); x < int(cx+2); x++ {
				l := 0.5 - float64(x-int(cx))/(5*k)*0.2
				c.put(x, y, grainHD(chestLid, x, y, l, true, 32))
			}
		}
	}
	c.outline(0.8)
	return c
}

// --- the furnace ---------------------------------------------------------------

// furnaceYaw is how far a furnace facing face is turned from facing south.
func furnaceYaw(face sim.Dir) float64 {
	switch face {
	case sim.DirRight:
		return math.Pi / 2
	case sim.DirUp:
		return math.Pi
	case sim.DirLeft:
		return -math.Pi / 2
	}
	return 0
}

// furnaceFacedHD is the furnace turned to face face, cold or lit: the
// scene of stones (rockart.go) turned and seen again, so its light still
// falls from the upper left.
func furnaceFacedHD(face sim.Dir, lit bool) sprite {
	c, ax, ay := furnaceSceneYaw(lit, furnaceYaw(face)).draw(hdK(), 5)
	return mkUp(c, ax, ay)
}

// mouthAt is where the fire in the mouth of a furnace or an oven e shows,
// from its ground point, art pixels, and how wide it is (1 as it was): at
// the side it faces, narrower on a flank; none when the mouth is round the
// back.
func mouthAt(e protocol.EntityView) (dx, dy, w float64, ok bool) {
	face := sim.FacingOf(e.Kind, e.Var, e.Face)
	if e.Kind == sim.EntityOven {
		switch face {
		case sim.DirUp:
			return 0, 0, 0, false
		case sim.DirRight:
			return 11.2, -3, 0.5, true
		case sim.DirLeft:
			return -11.2, -3, 0.5, true
		}
		return -1.5, -0.5, 1, true
	}
	switch face {
	case sim.DirUp:
		return 0, 0, 0, false
	case sim.DirRight:
		return 8.5, -4, 0.55, true
	case sim.DirLeft:
		return -8.5, -4, 0.55, true
	}
	return 0, -1, 1, true
}

// drawMouthFire is the fire burning in the mouth of furnace or oven e at d.
func (g *Game) drawMouthFire(dst drawTarget, e protocol.EntityView, d drawable) {
	dx, dy, w, ok := mouthAt(e)
	if !ok {
		return
	}
	x, y := d.sx+dx, d.sy+dy
	if w == 1 && g.drawFireOf(dst, e, fireMouth, x, y) { // see fireart.go
		return
	}
	if fireHD {
		s := fireSprite(fireMouth, g.fireFrame(uint32(e.ID)))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.ax, -s.ay)
		op.GeoM.Scale(w, 1)
		op.GeoM.Translate(math.Round(x*HDDensity)/HDDensity, math.Round(y*HDDensity)/HDDensity)
		drawSprite(dst, s, op)
		return
	}
	f := g.art.flames[(g.frame/4+int(e.ID))%flameFrames]
	if e.Kind == sim.EntityOven {
		drawAt(dst, f, x, y, 0.6*w, 0.42, 0, nil)
		return
	}
	drawAt(dst, f, x, y, 0.45*w, 0.4, 0, nil)
}

// --- the cooking gear ----------------------------------------------------------

// rackLook is which drying rack drawing e shows: as cookSprite picks it.
func rackLook(e protocol.EntityView) int {
	loaded, _, done := cookFlags(e)
	switch {
	case loaded && e.Cook&sim.CookFlagSalt != 0:
		return 3
	case loaded && e.Cook&sim.CookFlagFish != 0:
		return 2
	case loaded:
		return 1
	case done:
		return 4
	}
	return 0
}

// cookFacedSprite is a mortar, an oven or a drying rack e facing face (not
// south): the mortar's pestle leaning a quarter turn further round each
// time (back right as drawn; out over the near rim to the right facing
// west, to the left facing north; back left facing east), the oven's mouth
// on its flank or round the back, the rack with its low pole before it or
// turned to run north–south.
func (g *Game) cookFacedSprite(e protocol.EntityView, face sim.Dir) sprite {
	loaded, _, done := cookFlags(e)
	switch e.Kind {
	case sim.EntityMortar:
		full := loaded || done
		key := facedKey{k: e.Kind, face: face, state: uint8(btoi(full))}
		return faced(key, func() sprite {
			c, ax, ay := genMortarLean(full, face == sim.DirLeft || face == sim.DirUp)
			if face == sim.DirUp || face == sim.DirRight {
				return mkFlipped(c, ax, ay)
			}
			return mkUp(c, ax, ay)
		})
	case sim.EntityOven:
		lit, loaf := e.Fuel > 0, done
		key := facedKey{k: e.Kind, face: face, state: uint8(btoi(lit) + 2*btoi(loaf))}
		return faced(key, func() sprite {
			mouth := ovenMouthSide
			if face == sim.DirUp {
				mouth = ovenMouthBack
			}
			c, ax, ay := genOvenFacing(lit, loaf, mouth)
			if face == sim.DirLeft {
				return mkFlipped(c, ax, ay)
			}
			return mkUp(c, ax, ay)
		})
	}
	look := rackLook(e)
	key := facedKey{k: e.Kind, face: face, state: uint8(look)}
	return faced(key, func() sprite {
		if face == sim.DirUp {
			c, ax, ay := genRackSide(look, true)
			return mkUp(c, ax, ay)
		}
		c, ax, ay := genRackLong(look)
		if face == sim.DirLeft {
			return mkFlipped(c, ax, ay)
		}
		return mkUp(c, ax, ay)
	})
}

// grillLong is the grill turned north–south: what stands behind the flames
// and what is in front, bare or laden.
func grillLong(laden bool) (back, front sprite) {
	key := facedKey{k: sim.EntityCampfire, face: sim.DirRight, state: uint8(btoi(laden))}
	front = faced(key, func() sprite {
		_, f := genGrillLong(laden)
		return mkUp(f, gearFX, gearFY)
	})
	key.state += 2
	back = faced(key, func() sprite {
		b, _ := genGrillLong(laden)
		return mkUp(b, gearFX, gearFY)
	})
	return back, front
}
