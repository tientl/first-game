package client

import (
	"image/color"
	"math"
	"sort"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Walls and fences that join (sim/facing.go): a palisade, a stone wall or
// a fence of stakes reaches out to each neighbour that is a wall, a fence,
// or a door standing in a wall that runs its way, so pieces put side by
// side are one line, round corners and into T's and crosses. A tile with
// neighbours one way only is a straight run, the whole tile long, its open
// ends finished with a stout end post (a palisade's, a fence's) or a pillar
// (a stone wall's); a corner, a T or a cross has a post in the middle where
// the arms meet. A piece standing alone lies the way it faces: east–west as
// drawn before, or north–south.
//
// A run north–south is seen end on, as the game's view sees it: the stakes
// of a palisade one behind the other, each a little higher up the screen,
// their points a saw edge going back; a stone wall's top going back with
// its merlons, its face at the near end; a fence's crossed stakes in a
// row of X's along its rail. Each look of a tile is drawn the first time it
// is shown.

// The neighbours a run joins, as bits.
const (
	joinN uint8 = 1 << iota
	joinE
	joinS
	joinW
)

var joinDirs = [4]struct {
	bit    uint8
	dx, dy int
}{{joinN, 0, -1}, {joinE, 1, 0}, {joinS, 0, 1}, {joinW, -1, 0}}

// isRun reports whether entities of kind k join into lines.
func isRun(k sim.EntityKind) bool {
	return k == sim.EntityWall || k == sim.EntityStoneWall || k == sim.EntitySpikes
}

// runJoins reports whether a thing of kind k facing face, on the
// neighbouring tile that lies bit's way, joins a run: any wall or fence,
// and a door whose wall runs that way.
func runJoins(k sim.EntityKind, face sim.Dir, bit uint8) bool {
	switch {
	case isRun(k):
		return true
	case k == sim.EntityDoor:
		if sim.RunsNS(face) {
			return bit == joinN || bit == joinS
		}
		return bit == joinE || bit == joinW
	}
	return false
}

// runMask is which neighbours of tile (tx, ty) a run there joins, as this
// client sees them.
func (g *Game) runMask(tx, ty int) uint8 {
	var m uint8
	for _, d := range joinDirs {
		i, ok := g.entAt[[2]int{tx + d.dx, ty + d.dy}]
		if !ok || i >= len(g.curr.Entities) {
			continue
		}
		if e := g.curr.Entities[i]; runJoins(e.Kind, e.Face, d.bit) && !g.hiddenInCave(float64(e.TX)+0.5, float64(e.TY)+0.5) {
			m |= d.bit
		}
	}
	return m
}

// runShape is which parts a tile of a run draws: its halves each way from
// the middle, a post in the middle where arms meet, and end posts at the
// open ends of a straight run.
type runShape struct {
	n, e, s, w             bool
	mid                    bool
	capN, capE, capS, capW bool
}

// shapeOf is the shape of a run joining the neighbours in mask, facing face.
func shapeOf(mask uint8, face sim.Dir) runShape {
	ew := mask&(joinE|joinW) != 0
	ns := mask&(joinN|joinS) != 0
	var sh runShape
	switch {
	case ew && ns: // a corner, a T or a cross
		sh.n, sh.e, sh.s, sh.w = mask&joinN != 0, mask&joinE != 0, mask&joinS != 0, mask&joinW != 0
		sh.mid = true
	case ew || !ns && !sim.RunsNS(face): // east–west, the whole tile long
		sh.e, sh.w = true, true
		sh.capE, sh.capW = mask&joinE == 0, mask&joinW == 0
	default: // north–south
		sh.n, sh.s = true, true
		sh.capN, sh.capS = mask&joinN == 0, mask&joinS == 0
	}
	return sh
}

// runsNS reports whether a run's shape lies only north–south: its shadow
// is narrow.
func (sh runShape) runsNS() bool { return !sh.e && !sh.w }

func (sh runShape) key() uint16 {
	var k uint16
	for i, b := range []bool{sh.n, sh.e, sh.s, sh.w, sh.mid, sh.capN, sh.capE, sh.capS, sh.capW} {
		if b {
			k |= 1 << i
		}
	}
	return k
}

var runCache = map[[2]uint16]sprite{}

// runSprite is the look of run e on its tile, joined to its neighbours.
func (g *Game) runSprite(e protocol.EntityView) sprite {
	return runSpriteOf(e.Kind, shapeOf(g.runMask(int(e.TX), int(e.TY)), e.Face))
}

// runSpriteOf is the look of a run of kind k shaped sh, drawn the first time.
func runSpriteOf(k sim.EntityKind, sh runShape) sprite {
	key := [2]uint16{uint16(k), sh.key()}
	if s, ok := runCache[key]; ok {
		return s
	}
	var s sprite
	switch k {
	case sim.EntityStoneWall:
		s = genStoneRunHD(sh)
	case sim.EntitySpikes:
		s = genSpikeRunHD(sh)
	default:
		s = genPalisadeHD(sh)
	}
	runCache[key] = s
	return s
}

// runShadow is the width of a run's shadow: narrow for one going back.
func (g *Game) runShadow(e protocol.EntityView) float64 {
	if shapeOf(g.runMask(int(e.TX), int(e.TY)), e.Face).runsNS() {
		return 9
	}
	return 26
}

// The canvas of a run: wide enough for end posts, tall enough for a
// palisade going back a whole tile; anchored on the middle of its tile.
const (
	runW, runH   = 28, 62
	runAX, runAY = 14, 52
)

// The posts of a run, from the middle of the tile, art pixels on the
// screen: five across a tile east–west, five back along it north–south.
var (
	runXs = [5]float64{-9.6, -4.8, 0, 4.8, 9.6}
	runYs = [5]float64{-7.2, -3.6, 0, 3.6, 7.2}
)

// runPost is a post of a run: where it stands, and what it is.
type runPost struct {
	x, y  float64 // its foot, from the anchor, art pixels on the screen
	along bool    // part of the east–west row (else going back)
	cap   bool    // an end post or a corner post
	i     int     // its index, for its look
}

// runPosts lays out the posts of shape sh: back to front, left to right.
func runPosts(sh runShape) []runPost {
	var ps []runPost
	if sh.w {
		ps = append(ps, runPost{x: runXs[0], along: true, cap: sh.capW, i: 0}, runPost{x: runXs[1], along: true, i: 1})
	}
	if sh.e {
		ps = append(ps, runPost{x: runXs[3], along: true, i: 3}, runPost{x: runXs[4], along: true, cap: sh.capE, i: 4})
	}
	if sh.n {
		ps = append(ps, runPost{y: runYs[0], cap: sh.capN, i: 10}, runPost{y: runYs[1], i: 11})
	}
	if sh.s {
		ps = append(ps, runPost{y: runYs[3], i: 13}, runPost{y: runYs[4], cap: sh.capS, i: 14})
	}
	switch {
	case sh.mid:
		ps = append(ps, runPost{cap: true, along: true, i: 2})
	case sh.e && sh.w:
		ps = append(ps, runPost{along: true, i: 2})
	case sh.n && sh.s:
		ps = append(ps, runPost{i: 12})
	}
	sort.SliceStable(ps, func(a, b int) bool {
		if ps[a].y != ps[b].y {
			return ps[a].y < ps[b].y
		}
		return ps[a].x < ps[b].x
	})
	return ps
}

// runPx is the image pixel of art point (x, y) from a run's anchor.
func runPx(x, y float64) (float64, float64) {
	return (x + runAX) * HDDensity, (y + runAY) * HDDensity
}

// --- the palisade ----------------------------------------------------------

// stakeHD paints an upright log with a sharpened point: its foot's middle
// at image pixel (cx, foot), w pixels wide, h tall, lit from the left.
func stakeHD(c *canvas, cx, foot, w, h float64, seed uint32, cut bool) {
	const k = HDDensity
	point := 2.0*k + 2
	if cut {
		point = 0
	}
	x0, x1 := int(math.Round(cx-w/2)), int(math.Round(cx+w/2))
	top := int(math.Round(foot - h))
	logHD(c, x0, x1, top+int(point), int(foot), trunkRamp, 0.82, seed)
	half := float64(x1-x0) / 2
	mx := float64(x0) + half
	if cut {
		// A flat-cut top: the pale end grain, rings and all, as an ellipse.
		c.ellipse(mx, float64(top)+1.5, half, half*0.5, func(px, py int, nx, ny float64) color.RGBA {
			return woodEndHD(px, py, nx, ny, 3, seed)
		})
		return
	}
	// The point: cut faces of pale wood.
	for y := top; y < top+int(point); y++ {
		f := float64(y-top) / point // 0 at the tip
		hw := 0.8 + f*(half-0.8)
		for x := int(math.Floor(mx - hw)); x < int(math.Ceil(mx+hw)); x++ {
			u := (float64(x) + 0.5 - mx) / hw
			l := 0.75 - u*0.45 + (hashf(x, y, seed+912)-0.5)*0.1
			if math.Abs(u) < 0.18 {
				l -= 0.2 // the ridge between two faces
			}
			c.put(x, y, woodRamp.pick(l, x, y))
		}
	}
}

// collarHD winds a lashing of rope round whatever is opaque in the band of
// rows y to y+5 between columns x0 and x1.
func collarHD(c *canvas, x0, x1, y int) {
	for yy := y; yy < y+5; yy++ {
		for x := x0; x < x1; x++ {
			if !c.opaque(x, yy) {
				continue
			}
			t := (x + yy*2 + 60) % 4
			l := 2 - t/2
			if yy == y+4 {
				l = 0
			}
			c.put(x, yy, ropeRamp.at(l))
		}
	}
}

// genPalisadeHD is a palisade of sharpened stakes shaped sh: those along
// it backed by the dark between them and bound with three ropes, those
// going back each bound where the ropes pass, end and corner posts stouter,
// taller and lashed under a flat-cut top.
func genPalisadeHD(sh runShape) sprite {
	const k = HDDensity
	c := hdCanvas(runW, runH)
	ps := runPosts(sh)
	ropes := [3]float64{30, 18, 6} // the ropes' heights over the ground, art pixels
	height := func(p runPost) float64 {
		h := float64(wallArtH) - 2 + hashf(p.i, 3, 910)*3.5
		if p.cap {
			h = wallArtH + 2.5
		}
		return h
	}
	width := func(p runPost) float64 {
		if p.cap {
			return 5.6 * k
		}
		return 4.3 * k
	}
	// The row along it, at the middle: the dark behind, the stakes, the ropes.
	var row []runPost
	for _, p := range ps {
		if p.along {
			row = append(row, p)
		}
	}
	drawRow := func() {
		if len(row) == 0 {
			return
		}
		lx, _ := runPx(row[0].x, 0)
		rx, foot := runPx(row[len(row)-1].x, 0)
		for y := int(foot - (wallArtH-5)*k); y <= int(foot); y++ {
			for x := int(lx); x < int(rx); x++ {
				c.put(x, y, trunkRamp.at(0))
			}
		}
		for _, p := range row {
			x, f := runPx(p.x, p.y)
			stakeHD(c, x, f, width(p), height(p)*k, 911+uint32(p.i), p.cap)
			if hashf(p.i, 0, 913) > 0.45 {
				knotHD(c, x+(hashf(p.i, 1, 915)-0.5)*5, f-(10+hashf(p.i, 2, 914)*14)*k, 1.8, trunkRamp)
			}
		}
		x0, _ := runPx(row[0].x, 0)
		x1, _ := runPx(row[len(row)-1].x, 0)
		for _, r := range ropes {
			ropeHD(c, int(x0-3*k), int(x1+3*k), int(foot-r*k), 0)
		}
		for _, p := range row {
			if p.cap {
				x, f := runPx(p.x, p.y)
				collarHD(c, int(x-3*k), int(x+3*k), int(f-(wallArtH-3)*k))
			}
		}
	}
	rowDone := false
	for _, p := range ps {
		if p.along {
			continue
		}
		if p.y > 0 && !rowDone {
			drawRow()
			rowDone = true
		}
		x, f := runPx(p.x, p.y)
		stakeHD(c, x, f, width(p), height(p)*k, 921+uint32(p.i), p.cap)
		w := width(p)
		for _, r := range ropes { // bound where the ropes go round
			collarHD(c, int(x-w/2-1), int(x+w/2+1), int(f-r*k)-2)
		}
		if p.cap {
			collarHD(c, int(x-w/2-1), int(x+w/2+1), int(f-(wallArtH-3)*k))
		}
	}
	if !rowDone {
		drawRow()
	}
	c.outline(0.8)
	return mkUp(c, runAX*k, runAY*k)
}

// --- the stone wall ----------------------------------------------------------

// stoneFace paints a stretch of stone wall face from image column x0 to x1,
// its foot on row foot, height h image pixels over it, merlons standing
// proud every other block; the courses keep to the tile's own grid (gx, gy
// is the image pixel the grid starts from) so stretches side by side meet.
func stoneFace(c *canvas, x0, x1, foot, h int, gx, gy int, merlons bool, seed uint32) {
	const k = HDDensity
	const B = 4 * k // a stone with its mortar
	for y := foot - h - 3*k; y < foot+k; y++ {
		for x := x0; x < x1; x++ {
			lx := x - gx
			top := foot - h
			if merlons && ((lx+600*B)/B)%2 == 0 {
				top -= 3 * k
			}
			if y < top {
				continue
			}
			row := (y - gy + 100*B) / B
			sx := lx + B/2*(row%2) + 600*B
			bx, by4 := sx%B, (y-gy+100*B)%B
			stone := hashf(sx/B, row, seed)
			l := 0.74 - float64((lx+600*B)%(24*k))/(24*k)*0.32 + (stone-0.5)*0.26
			l += (hashf(x, y, seed+3) - 0.5) * 0.08
			l += (vnoise(float64(lx)*0.4, float64(y)*0.4, seed+4) - 0.5) * 0.14
			round := (bx == 0 || bx == B-2) && (by4 == 0 || by4 == B-2)
			switch {
			case bx == B-1 || by4 == B-1 || round:
				l = 0.12 + (hashf(x, y, seed+5)-0.5)*0.1
			case bx == 0 || by4 == 0:
				l += 0.16
			case bx == B-2 || by4 == B-2:
				l -= 0.16
			case hashf(x/2, y/2, seed+6) < 0.03:
				l -= 0.2
			}
			if y > foot-3*k {
				l -= 0.1
			}
			if y < top+2 {
				l += 0.12 // the lit lip of a block's top
			}
			col := stoneRamp.pick(l, x, y)
			m := vnoise(float64(lx)*0.2, float64(y)*0.25, seed+1)
			if up := float64(foot-y) / k; m > 0.42+up*0.05 || (bx >= B-2 || by4 >= B-2) && m > 0.62 {
				col = mossRamp.pick(l+0.35+(hashf(x, y, seed+2)-0.5)*0.4, x, y)
			}
			c.put(x, y, col)
		}
	}
}

// stoneTop paints the top of a stone wall going back, seen from above,
// between image columns x0 and x1 and rows y0 (the back) and y1: capstones
// in courses across it, lit, worn at their edges, with a merlon standing on
// every other one.
func stoneTop(c *canvas, x0, x1, y0, y1 int, seed uint32) {
	const k = HDDensity
	const D = 3.6 * k // a capstone's depth on the screen, as the slices of the run
	for y := y0; y < y1; y++ {
		course := int(math.Floor((float64(y-y0) + 600*D) / D))
		fy := math.Mod(float64(y-y0)+600*D, D)
		for x := x0; x < x1; x++ {
			l := 0.86 - float64(x-x0)/float64(x1-x0)*0.25 + (hashf(course, 0, seed)-0.5)*0.2
			l += (hashf(x, y, seed+1) - 0.5) * 0.08
			switch {
			case fy < 1:
				l = 0.2 // the joint between capstones
			case fy < 2.2:
				l += 0.1
			case fy > D-2:
				l -= 0.12
			}
			if x == x0 || x == x1-1 {
				l -= 0.2
			}
			c.put(x, y, stoneRamp.pick(l, x, y))
		}
	}
}

// genStoneRunHD is a stone wall shaped sh: the stretches along it are the
// old wall's face of coursed blocks with its battlements; a stretch going
// back is its top with merlons, and the face of its near end; end and
// corner pieces are square pillars standing a little higher.
func genStoneRunHD(sh runShape) sprite {
	const k = HDDensity
	c := hdCanvas(runW, runH)
	const H = (wallArtH - 5) * k // the face's height under the battlements, as the old wall's
	_, foot0 := runPx(0, 0)
	gx, _ := runPx(-12, 0)
	gy := int(foot0) - (wallArtH+4)*k + 4*k // the old wall's courses
	const half = 5 * k                      // half the wall's thickness going back, on the screen
	mx, _ := runPx(0, 0)
	// Whether it goes on north and south into the next tile.
	joinedN := sh.n && (sh.mid || !sh.capN)
	joinedS := sh.s && (sh.mid || !sh.capS)
	// Going back: from the back end to the front, its top and merlons.
	if sh.n || sh.s {
		back, front := 0.0, 0.0
		if sh.n {
			back = -9
		}
		if sh.s {
			front = 9
		}
		_, yb := runPx(0, back)
		_, yf := runPx(0, front)
		stoneTop(c, int(mx)-half, int(mx)+half, int(yb)-H, int(yf)-H, 931)
		// Merlons along it: a block on every other capstone.
		for y := back + 1.8; y < front; y += 7.2 {
			_, fy := runPx(0, y)
			stoneFace(c, int(mx)-half, int(mx)+half, int(fy)-H, 3*k, int(mx)-half, int(fy)-H-3*k, false, 932)
			stoneTop(c, int(mx)-half, int(mx)+half, int(fy)-H-3*k-5, int(fy)-H-3*k, 933)
		}
		// The face of its near end, unless the run goes on south over it.
		if !joinedS {
			stoneFace(c, int(mx)-half, int(mx)+half, int(yf), H, int(mx)-half, gy, true, 934)
		}
	}
	// Along it.
	if sh.w || sh.e {
		x0, x1 := -12.0, 12.0
		if !sh.w {
			x0 = 0
		}
		if !sh.e {
			x1 = 0
		}
		if sh.mid { // the arms meet the pillar
			if !sh.w {
				x0 = -5
			}
			if !sh.e {
				x1 = 5
			}
		}
		px0, _ := runPx(x0, 0)
		px1, _ := runPx(x1, 0)
		stoneFace(c, int(px0), int(px1), int(foot0), H, int(gx), gy, true, 920)
	}
	// Pillars: at the open ends and where arms meet.
	pillar := func(x, y float64) {
		px, py := runPx(x, y)
		stoneTop(c, int(px)-half-2, int(px)+half+2, int(py)-H-5*k-9*k/2, int(py)-H-5*k, 935)
		stoneFace(c, int(px)-half-2, int(px)+half+2, int(py), H+5*k, int(px)-half-2, gy-2*k, false, 936)
	}
	if sh.capN {
		pillar(0, -6.5)
	}
	if sh.capW {
		pillar(-8.5, 0)
	}
	if sh.capE {
		pillar(8.5, 0)
	}
	if sh.mid {
		pillar(0, 0)
	}
	if sh.capS {
		pillar(0, 6.5)
	}
	c.outline(0.8)
	if joinedN { // no ink across the top where it runs on from the tile behind
		_, yb := runPx(0, -9)
		for y := int(yb) - H - 2; y < int(yb)-H; y++ {
			for x := int(mx) - half - 1; x <= int(mx)+half; x++ {
				if c.in(x, y) && c.ol != nil && c.ol[y*c.w+x] {
					c.px[y*c.w+x], c.ol[y*c.w+x] = color.RGBA{}, false
				}
			}
		}
	}
	return mkUp(c, runAX*k, runAY*k)
}

// --- the fence of stakes --------------------------------------------------------

// spikeHD paints a sharpened stake from its foot at image pixel (x, foot)
// up h pixels, leaning lean pixels sideways at the top, three pixels thick,
// lit on its left.
func spikeHD(c *canvas, x, foot, h, lean float64, seed uint32) {
	n := int(h)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n) // 0 at the foot
		px := x + lean*t
		py := foot - h*t
		w := 2.0
		if t > 0.86 { // the point
			w = 2 * (1 - (t-0.86)/0.14)
		}
		for d := -w; d <= w; d++ {
			l := 0.5 - d/w*0.25
			rp := plankRamp
			if t > 0.86 {
				rp, l = woodRamp, 0.7-d/max(w, 1)*0.3
			}
			c.put(int(math.Round(px+d)), int(py), rp.pick(l+(hashf(int(px+d), int(py), seed)-0.5)*0.12, int(px+d), int(py)))
		}
	}
}

// genSpikeRunHD is a fence of sharpened stakes shaped sh: along it, a rail
// on short posts with the stakes crossed over it, leaning one way and the
// other; going back, the rail runs up the screen and the stakes cross over
// it in a row of X's; end and corner posts stand up stout.
func genSpikeRunHD(sh runShape) sprite {
	const k = HDDensity
	c := hdCanvas(runW, runH)
	const railH = 6.5   // the rail's height, art pixels
	const spikeH = 17.5 // the stakes' height
	ps := runPosts(sh)
	rail := func(x0, y0, x1, y1 float64) {
		ax, ay := runPx(x0, y0-railH)
		bx, by := runPx(x1, y1-railH)
		drawLimbHD(c, limb{ax, ay, (ax + bx) / 2, (ay + by) / 2, bx, by, 3.2, 3.2}, trunkRamp, 0.72, 940)
	}
	// Going back first (the back half), then the rail along, then the front.
	drawBack := func(front bool) {
		for _, p := range ps {
			if p.along || (p.y > 0) != front || p.cap {
				continue
			}
			x, f := runPx(p.x, p.y)
			lean := 2.6 * k
			spikeHD(c, x-lean/2, f, spikeH*k, lean, 941+uint32(p.i))
			spikeHD(c, x+lean/2, f+1, spikeH*k, -lean, 942+uint32(p.i))
		}
	}
	if sh.n || sh.s {
		back := 0.0
		if sh.n {
			back = -9
		}
		drawBack(false)
		rail(0, back, 0, 0)
	}
	if sh.w || sh.e {
		x0, x1 := -12.0, 12.0
		if !sh.w {
			x0 = 0
		}
		if !sh.e {
			x1 = 0
		}
		for _, p := range ps {
			if p.along && !p.cap {
				x, f := runPx(p.x, p.y)
				lean := float64(p.i%2*2-1) * 2.5 * k
				spikeHD(c, x, f, spikeH*k, lean, 943+uint32(p.i))
			}
		}
		rail(x0, 0, x1, 0)
	}
	if sh.n || sh.s {
		front := 0.0
		if sh.s {
			front = 9
		}
		rail(0, 0, 0, front)
		drawBack(true)
	}
	for _, p := range ps {
		if p.cap {
			x, f := runPx(p.x, p.y)
			stakeHD(c, x, f, 4.4*k, (spikeH+3)*k, 944+uint32(p.i), false)
			collarHD(c, int(x-3*k), int(x+3*k), int(f-railH*k)-2)
		}
	}
	c.outline(0.75)
	return mkUp(c, runAX*k, runAY*k)
}
