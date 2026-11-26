package client

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The trunk sheet: a dev picture of the foot of every kind of tree close up,
// the player beside each, three times the size of a 1080p screen, to judge
// how trunks meet the ground:
//
//	go run ./cmd/shot -w 1920 -h 1080 -trunksheet -o trunks.png

// trunkEntries is one tree of each kind of trunk.
func (g *Game) trunkEntries() []scaleEntry {
	a := g.art
	nShape := len(leafRamps)
	big := a.treesOf(1)
	anc := a.ancientOf(1)
	fruit := a.fruitOf()
	woods := woodsArtOf()
	return []scaleEntry{
		{name: "round", s: big.trees[int(shapeRound)*nShape]},
		{name: "oak", s: big.trees[int(shapeOak)*nShape+2]},
		{name: "weeping, vines", s: big.vined[int(shapeWeeping)*nShape+1]},
		{name: "spread", s: big.trees[int(shapeSpread)*nShape]},
		{name: "ginkgo", s: anc.ginkgos[0]},
		{name: "fig", s: fruit.figs[0]},
		{name: "pine", s: big.pines[1]},
		{name: "snow pine", s: a.snowPinesOf(1)[2]},
		{name: "araucaria", s: anc.araucarias[0]},
		{name: "redwood", s: anc.redwoods[0]},
		{name: "tree fern", s: anc.ferns[0]},
		{name: "cycad", s: anc.cycads[0]},
		{name: "palm", s: fruit.palms[0]},
		{name: "cypress", s: swampTree(1, 9)},
		{name: "snag", s: swampTree(1, 0)},
		{name: "giant", s: woods.giants[0]},
	}
}

// TrunkSheet draws the trunk sheet. It must run inside the game loop (it
// reads an image back).
func (g *Game) TrunkSheet() image.Image {
	if g.art == nil {
		g.art = getArt()
	}
	const (
		cols         = 4
		tileW, tileH = 84.0, 96.0 // art pixels
		ground       = 84.0       // the ground line in a tile
	)
	es := g.trunkEntries()
	rows := (len(es) + cols - 1) / cols
	const K = HDDensity
	img := image.NewRGBA(image.Rect(0, 0, int(tileW*cols)*K, int(tileH)*rows*K))
	me := protocol.PlayerView{ID: 1, Dir: sim.DirDown}
	for i, e := range es {
		// Each tile on its own, so a crown does not spill into the next.
		buf := newFineBuf(int(tileW), int(tileH))
		buf.img.Fill(color.RGBA{0x5c, 0x86, 0x4c, 0xff})
		fillRect(buf, 0, ground, tileW, 0.34, color.RGBA{0x2c, 0x48, 0x2a, 0xff}, false)
		drawAt(buf, e.s, tileW*0.58, ground, 1, 1, 0, nil)
		g.drawFigureJ(buf, me, sim.DirDown, loKey{dir: sim.DirDown}, upKey{dir: sim.DirDown}, 12, ground, 0)
		fillRect(buf, tileW-0.34, 0, 0.34, tileH, color.RGBA{0x24, 0x30, 0x22, 0xff}, false)
		fillRect(buf, 0, tileH-0.34, tileW, 0.34, color.RGBA{0x24, 0x30, 0x22, 0xff}, false)
		drawText(buf.img, e.name, 3*K, (ground+3)*K, colWhite, text.AlignStart, 1)
		tile := image.NewRGBA(buf.img.Bounds())
		buf.img.ReadPixels(tile.Pix)
		buf.img.Deallocate()
		x0, y0 := (i%cols)*int(tileW)*K, (i/cols)*int(tileH)*K
		for y := range tile.Rect.Dy() {
			copy(img.Pix[(y0+y)*img.Stride+x0*4:], tile.Pix[y*tile.Stride:(y+1)*tile.Stride])
		}
	}
	b := img.Bounds()
	// Three times as big, pixel for pixel.
	const z = 3
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*z, b.Dy()*z))
	for y := range b.Dy() * z {
		for x := range b.Dx() * z {
			si := (y/z)*img.Stride + (x/z)*4
			di := y*out.Stride + x*4
			copy(out.Pix[di:di+4], img.Pix[si:si+4])
		}
	}
	return out
}
