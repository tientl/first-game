package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/sim"
)

// The ground is drawn foreshortened (tiles shorter than they are wide) while
// sprites stand upright on it: the classic 3/4 view.
const (
	TileW = 48
	TileH = 36

	tileVariants = 4
	cliffH       = 12 // earth wall drawn below land that borders the sea
	shadowSize   = 64
	lightSize    = 256
)

// sprite is an image plus its anchor: the pixel that sits on the ground point.
type sprite struct {
	img    *ebiten.Image
	ax, ay float64
}

// art holds every placeholder image, generated in code at start-up. Swapping
// in real art later only touches this file.
type art struct {
	tiles [sim.NumTiles][tileVariants]*ebiten.Image
	cliff *ebiten.Image

	tree, stump          sprite
	boulder              sprite
	bushFull, bushPicked sprite
	tuftFull, tuftPicked sprite
	campfire             sprite
	flame, flameCore     sprite
	tombstone            sprite

	legs  sprite
	body  sprite // white, tinted per player
	heads [4]sprite
	tool  sprite

	shadow *ebiten.Image // black disc, drawn squashed
	light  *ebiten.Image // radial white gradient, alpha-premultiplied
	marker *ebiten.Image
	icons  [sim.NumItems]*ebiten.Image
}

// Palette.
var (
	colGrass    = color.RGBA{98, 160, 72, 255}
	colForest   = color.RGBA{66, 114, 60, 255}
	colSand     = color.RGBA{222, 202, 150, 255}
	colRocky    = color.RGBA{140, 132, 118, 255}
	colWater    = color.RGBA{52, 110, 170, 255}
	colTrunk    = color.RGBA{110, 74, 44, 255}
	colLeafDark = color.RGBA{38, 96, 52, 255}
	colLeafMid  = color.RGBA{52, 124, 62, 255}
	colLeafLite = color.RGBA{86, 156, 80, 255}
	colStone    = color.RGBA{128, 128, 134, 255}
	colStoneLit = color.RGBA{166, 166, 172, 255}
	colStoneDim = color.RGBA{96, 96, 104, 255}
	colBerry    = color.RGBA{204, 40, 64, 255}
	colStraw    = color.RGBA{190, 196, 92, 255}
	colSkin     = color.RGBA{240, 200, 160, 255}
	colHair     = color.RGBA{70, 46, 30, 255}
	colLegs     = color.RGBA{60, 52, 70, 255}
	colFlame    = color.RGBA{255, 140, 30, 255}
	colFlameHot = color.RGBA{255, 226, 96, 255}
	colWhite    = color.RGBA{255, 255, 255, 255}
	colBlack    = color.RGBA{0, 0, 0, 255}
)

func newArt() *art {
	a := &art{}
	for t := sim.Tile(0); t < sim.NumTiles; t++ {
		for v := 0; v < tileVariants; v++ {
			a.tiles[t][v] = genTile(t, v)
		}
	}
	a.cliff = genCliff()

	a.tree = genTree()
	a.stump = genStump()
	a.boulder = genBoulder()
	a.bushFull, a.bushPicked = genBush(true), genBush(false)
	a.tuftFull, a.tuftPicked = genTuft(true), genTuft(false)
	a.campfire = genCampfire()
	a.flame = genFlame(colFlame, 1)
	a.flameCore = genFlame(colFlameHot, 0.55)
	a.tombstone = genTombstone()

	a.legs = genLegs()
	a.body = genBody()
	for d := sim.DirDown; d <= sim.DirRight; d++ {
		a.heads[d] = genHead(d)
	}
	a.tool = genTool()

	a.shadow = ebiten.NewImage(shadowSize, shadowSize)
	vector.FillCircle(a.shadow, shadowSize/2, shadowSize/2, shadowSize/2, colBlack, true)
	a.light = genLight()
	a.marker = genMarker()
	a.icons = genIcons()
	return a
}

// --- ground -----------------------------------------------------------------

type pixels struct {
	w, h int
	px   []byte
}

func newPixels(w, h int) *pixels { return &pixels{w: w, h: h, px: make([]byte, w*h*4)} }

func (p *pixels) set(x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= p.w || y >= p.h {
		return
	}
	i := (y*p.w + x) * 4
	p.px[i], p.px[i+1], p.px[i+2], p.px[i+3] = c.R, c.G, c.B, c.A
}

func (p *pixels) image() *ebiten.Image {
	img := ebiten.NewImage(p.w, p.h)
	img.WritePixels(p.px)
	return img
}

func shade(c color.RGBA, d int) color.RGBA {
	ch := func(v uint8) uint8 { return uint8(max(0, min(255, int(v)+d))) }
	return color.RGBA{ch(c.R), ch(c.G), ch(c.B), c.A}
}

func genTile(t sim.Tile, variant int) *ebiten.Image {
	rng := rand.New(rand.NewPCG(uint64(t)*7919+1, 17))
	if t != sim.TileWater { // water variants are animation frames of one pattern
		rng = rand.New(rand.NewPCG(uint64(t)*7919+uint64(variant)+1, 17))
	}
	p := newPixels(TileW, TileH)
	base := map[sim.Tile]color.RGBA{
		sim.TileWater: colWater, sim.TileSand: colSand, sim.TileGrass: colGrass,
		sim.TileForest: colForest, sim.TileRocky: colRocky,
	}[t]
	for y := 0; y < TileH; y++ {
		for x := 0; x < TileW; x++ {
			p.set(x, y, shade(base, rng.IntN(9)-4))
		}
	}

	switch t {
	case sim.TileGrass, sim.TileForest:
		blade := shade(base, -24)
		for range 28 {
			x, y := rng.IntN(TileW), rng.IntN(TileH)
			p.set(x, y, blade)
			p.set(x, y+1, blade)
		}
		if t == sim.TileForest {
			for range 10 {
				p.set(rng.IntN(TileW), rng.IntN(TileH), color.RGBA{112, 92, 54, 255})
			}
		}
	case sim.TileSand:
		for range 30 {
			p.set(rng.IntN(TileW), rng.IntN(TileH), shade(base, []int{-26, 18}[rng.IntN(2)]))
		}
	case sim.TileRocky:
		for range 9 {
			x, y := rng.IntN(TileW), rng.IntN(TileH)
			p.set(x, y, shade(base, -30))
			p.set(x+1, y, shade(base, -30))
			p.set(x, y-1, shade(base, 24))
		}
	case sim.TileWater:
		// The same streaks drift sideways from frame to frame.
		for range 6 {
			x0, y, n := rng.IntN(TileW), rng.IntN(TileH), 5+rng.IntN(8)
			for i := 0; i < n; i++ {
				p.set((x0+i+variant*2)%TileW, y, color.RGBA{96, 154, 206, 255})
			}
		}
	}
	return p.image()
}

func genCliff() *ebiten.Image {
	rng := rand.New(rand.NewPCG(5, 5))
	p := newPixels(TileW, cliffH)
	for y := 0; y < cliffH; y++ {
		base := color.RGBA{uint8(128 - y*4), uint8(92 - y*3), uint8(60 - y*2), 255}
		for x := 0; x < TileW; x++ {
			c := shade(base, rng.IntN(11)-5)
			if y == 0 {
				c = color.RGBA{158, 126, 84, 255}
			}
			p.set(x, y, c)
		}
	}
	return p.image()
}

// --- shapes -----------------------------------------------------------------

func fillEllipse(dst *ebiten.Image, cx, cy, rx, ry float32, clr color.Color) {
	var path vector.Path
	const n = 32
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / n
		x, y := cx+rx*float32(math.Cos(a)), cy+ry*float32(math.Sin(a))
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	path.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &path, nil, op)
}

func fillPolygon(dst *ebiten.Image, clr color.Color, pts ...float32) {
	var path vector.Path
	path.MoveTo(pts[0], pts[1])
	for i := 2; i+1 < len(pts); i += 2 {
		path.LineTo(pts[i], pts[i+1])
	}
	path.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &path, nil, op)
}

func fillRoundRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.Color) {
	vector.FillRect(dst, x+r, y, w-2*r, h, clr, true)
	vector.FillRect(dst, x, y+r, w, h-2*r, clr, true)
	vector.FillCircle(dst, x+r, y+r, r, clr, true)
	vector.FillCircle(dst, x+w-r, y+r, r, clr, true)
	vector.FillCircle(dst, x+r, y+h-r, r, clr, true)
	vector.FillCircle(dst, x+w-r, y+h-r, r, clr, true)
}

// --- sprites ----------------------------------------------------------------

func genTree() sprite {
	img := ebiten.NewImage(56, 92)
	vector.FillRect(img, 24, 52, 8, 36, colTrunk, true)
	vector.FillRect(img, 24, 52, 3, 36, shade(colTrunk, -20), true)
	fillEllipse(img, 28, 88, 7, 3, shade(colTrunk, -10))
	for _, c := range []struct {
		x, y, r float32
		col     color.RGBA
	}{
		{28, 50, 20, colLeafDark}, {13, 56, 12, colLeafDark}, {43, 56, 12, colLeafDark},
		{28, 32, 18, colLeafMid}, {17, 42, 13, colLeafMid}, {39, 42, 13, colLeafMid},
		{23, 24, 8, colLeafLite}, {33, 34, 6, colLeafLite},
	} {
		vector.FillCircle(img, c.x, c.y, c.r, c.col, true)
	}
	return sprite{img, 28, 88}
}

func genStump() sprite {
	img := ebiten.NewImage(24, 18)
	vector.FillRect(img, 4, 6, 16, 9, colTrunk, true)
	fillEllipse(img, 12, 15, 8, 3, colTrunk)
	fillEllipse(img, 12, 6, 8, 3.5, color.RGBA{196, 160, 112, 255})
	fillEllipse(img, 12, 6, 4, 1.8, color.RGBA{170, 132, 90, 255})
	return sprite{img, 12, 15}
}

func genBoulder() sprite {
	img := ebiten.NewImage(44, 34)
	vector.FillCircle(img, 12, 22, 9, colStoneDim, true)
	vector.FillCircle(img, 32, 22, 9, colStoneDim, true)
	vector.FillCircle(img, 22, 18, 13, colStone, true)
	fillEllipse(img, 22, 26, 18, 6, colStone)
	vector.FillCircle(img, 17, 13, 5, colStoneLit, true)
	return sprite{img, 22, 30}
}

func genBush(berries bool) sprite {
	img := ebiten.NewImage(40, 34)
	leaf, leafLite := colLeafMid, colLeafLite
	if !berries {
		leaf, leafLite = color.RGBA{92, 126, 60, 255}, color.RGBA{120, 150, 76, 255}
	}
	vector.FillCircle(img, 10, 23, 9, leaf, true)
	vector.FillCircle(img, 30, 23, 9, leaf, true)
	vector.FillCircle(img, 20, 18, 12, leaf, true)
	vector.FillCircle(img, 17, 13, 6, leafLite, true)
	if berries {
		for _, b := range [][2]float32{{12, 20}, {22, 14}, {28, 24}, {18, 26}, {26, 17}, {9, 27}} {
			vector.FillCircle(img, b[0], b[1], 2.6, colBerry, true)
		}
	}
	return sprite{img, 20, 31}
}

func genTuft(full bool) sprite {
	img := ebiten.NewImage(30, 28)
	length := float32(22)
	col := colStraw
	if !full {
		length, col = 5, shade(colStraw, -40)
	}
	for i := -3; i <= 3; i++ {
		a := float64(i) * 0.22
		x := 15 + float32(math.Sin(a))*length
		y := 26 - float32(math.Cos(a))*length
		vector.StrokeLine(img, 15+float32(i), 26, x, y, 2, shade(col, i*6), true)
	}
	return sprite{img, 15, 26}
}

func genCampfire() sprite {
	img := ebiten.NewImage(40, 24)
	for i := 0; i < 9; i++ {
		a := 2 * math.Pi * float64(i) / 9
		vector.FillCircle(img, 20+float32(math.Cos(a))*15, 13+float32(math.Sin(a))*7, 3.5, colStone, true)
	}
	vector.StrokeLine(img, 10, 16, 30, 9, 4, colTrunk, true)
	vector.StrokeLine(img, 10, 9, 30, 16, 4, shade(colTrunk, 16), true)
	return sprite{img, 20, 13}
}

func genFlame(col color.RGBA, scale float32) sprite {
	img := ebiten.NewImage(24, 34)
	s := scale
	vector.FillCircle(img, 12, 32-9*s, 9*s, col, true)
	fillPolygon(img, col, 12-9*s, 32-10*s, 12, 32-30*s, 12+9*s, 32-10*s)
	return sprite{img, 12, 32}
}

func genTombstone() sprite {
	img := ebiten.NewImage(24, 30)
	fillRoundRect(img, 4, 4, 16, 24, 7, colStone)
	vector.FillRect(img, 4, 14, 16, 14, colStone, true)
	vector.FillRect(img, 11, 9, 2, 11, colStoneDim, true)
	vector.FillRect(img, 8, 12, 8, 2, colStoneDim, true)
	return sprite{img, 12, 27}
}

func genLegs() sprite {
	img := ebiten.NewImage(14, 8)
	vector.FillRect(img, 2, 0, 4, 8, colLegs, true)
	vector.FillRect(img, 8, 0, 4, 8, colLegs, true)
	return sprite{img, 7, 8}
}

func genBody() sprite {
	img := ebiten.NewImage(20, 18)
	fillRoundRect(img, 1, 0, 18, 18, 5, colWhite)
	return sprite{img, 10, 18}
}

func genHead(d sim.Dir) sprite {
	img := ebiten.NewImage(22, 22)
	vector.FillCircle(img, 11, 10, 9.5, colHair, true)
	eye := color.RGBA{30, 24, 24, 255}
	switch d {
	case sim.DirDown:
		vector.FillCircle(img, 11, 12.5, 8, colSkin, true)
		vector.FillCircle(img, 8, 13, 1.4, eye, true)
		vector.FillCircle(img, 14, 13, 1.4, eye, true)
	case sim.DirLeft:
		vector.FillCircle(img, 9, 12.5, 7.5, colSkin, true)
		vector.FillCircle(img, 5.5, 13, 1.4, eye, true)
	case sim.DirRight:
		vector.FillCircle(img, 13, 12.5, 7.5, colSkin, true)
		vector.FillCircle(img, 16.5, 13, 1.4, eye, true)
	case sim.DirUp:
		// Back of the head: hair only.
	}
	return sprite{img, 11, 20}
}

func genTool() sprite {
	img := ebiten.NewImage(10, 18)
	vector.FillRect(img, 4, 3, 2, 15, colTrunk, true)
	vector.FillRect(img, 0, 1, 10, 4, colStoneLit, true)
	return sprite{img, 5, 17} // pivot at the grip
}

// genLight is a soft white disc for punching holes in the darkness and for
// additive fire glow. Pixels are alpha-premultiplied, as Ebitengine expects.
func genLight() *ebiten.Image {
	p := newPixels(lightSize, lightSize)
	c := float64(lightSize) / 2
	for y := 0; y < lightSize; y++ {
		for x := 0; x < lightSize; x++ {
			d := math.Hypot(float64(x)+0.5-c, float64(y)+0.5-c) / c
			a := 1.0
			if d > 0.35 {
				t := min(1, (d-0.35)/0.65)
				a = 1 - t*t*(3-2*t)
			}
			v := uint8(math.Round(a * 255))
			p.set(x, y, color.RGBA{v, v, v, v})
		}
	}
	return p.image()
}

func genMarker() *ebiten.Image {
	img := ebiten.NewImage(12, 8)
	fillPolygon(img, colWhite, 0, 0, 12, 0, 6, 8)
	return img
}

func genIcons() [sim.NumItems]*ebiten.Image {
	var icons [sim.NumItems]*ebiten.Image
	for k := sim.ItemKind(1); k < sim.NumItems; k++ {
		icons[k] = ebiten.NewImage(22, 22)
	}

	wood := icons[sim.ItemWood]
	vector.FillRect(wood, 2, 8, 16, 8, colTrunk, true)
	fillEllipse(wood, 18, 12, 3, 4, color.RGBA{196, 160, 112, 255})
	vector.FillRect(wood, 4, 10, 10, 1, shade(colTrunk, -24), true)

	stone := icons[sim.ItemStone]
	vector.FillCircle(stone, 9, 13, 6, colStoneDim, true)
	vector.FillCircle(stone, 14, 11, 5.5, colStone, true)
	vector.FillCircle(stone, 12, 9, 2, colStoneLit, true)

	grass := icons[sim.ItemGrass]
	for i := -2; i <= 2; i++ {
		vector.StrokeLine(grass, 11, 19, 11+float32(i)*3, 3, 2, shade(colStraw, i*8), true)
	}
	vector.FillRect(grass, 7, 13, 8, 2, colTrunk, true)

	berry := icons[sim.ItemBerry]
	for _, b := range [][2]float32{{8, 14}, {14, 14}, {11, 9}} {
		vector.FillCircle(berry, b[0], b[1], 4, colBerry, true)
	}
	vector.FillCircle(berry, 15, 5, 2.5, colLeafLite, true)
	return icons
}
