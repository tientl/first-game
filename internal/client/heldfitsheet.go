package client

import (
	"image"
	"image/color"
	"image/draw"
	"path/filepath"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The held-fit sheets: every tool in the caveman's hand standing in each
// facing (held_size.png), and GIFs of a run seen from behind and from the
// side with things in hand (held_run_back.gif, held_run_side.gif), drawn by
// the game's own code as the motion sheets are (motionsheet.go):
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -heldfit dir
//
// It must run inside the game loop (Draw).

// heldFitItems is what the sheets show, in order.
var heldFitItems = []sim.ItemKind{
	sim.ItemAxe, sim.ItemPickaxe, sim.ItemHoe, sim.ItemCopperSword, sim.ItemBattleAxe,
	sim.ItemSpear, sim.ItemIvorySpear, sim.ItemRod, sim.ItemTorch,
}

// HeldFitSheets writes the held-fit sheets into dir.
func (g *Game) HeldFitSheets(dir string) error {
	keep := g.frame
	defer func() { g.frame = keep }()
	if err := g.heldSizeSheet(filepath.Join(dir, "held_size.png")); err != nil {
		return err
	}
	run := func(held sim.ItemKind, dirs []sim.Dir) reel {
		return reel{name: asciiName(held), n: 48, dirs: dirs, bg: reelGrass, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Held = held
			dx, dy := p.Dir.Delta()
			along := float32(1.6 * float64(t) / TileW)
			p.X += float32(dx) * along
			p.Y += float32(dy) * along
			d.moving = true
		}}
	}
	var back, side []reel
	for _, k := range []sim.ItemKind{sim.ItemAxe, sim.ItemSpear, sim.ItemTorch, sim.ItemCopperSword} {
		back = append(back, run(k, []sim.Dir{sim.DirUp, sim.DirDown}))
		side = append(side, run(k, []sim.Dir{sim.DirRight, sim.DirLeft}))
	}
	buf := newFineBuf(cellW, cellH)
	if err := g.writeReelGIF(filepath.Join(dir, "held_run_back.gif"), back, buf); err != nil {
		return err
	}
	return g.writeReelGIF(filepath.Join(dir, "held_run_side.gif"), side, buf)
}

// heldSizeSheet draws each of heldFitItems in the hand of the caveman
// standing still, in every facing, a row each.
func (g *Game) heldSizeSheet(path string) error {
	cw, ch := cellW*HDDensity, cellH*HDDensity
	dirs := []sim.Dir{sim.DirRight, sim.DirDown, sim.DirLeft, sim.DirUp}
	sheet := image.NewRGBA(image.Rect(0, 0, len(dirs)*cw, len(heldFitItems)*ch))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{36, 40, 36, 255}}, image.Point{}, draw.Src)
	buf := newFineBuf(cellW, cellH)
	for r, k := range heldFitItems {
		for c, d := range dirs {
			g.resetMotions()
			img := g.render(reel{bg: reelGrass, at: func(g *Game, t int, p *protocol.PlayerView, dr *drawable) {
				p.Held = k
			}}, d, 0, buf)
			draw.Draw(sheet, image.Rect(c*cw, r*ch, (c+1)*cw, (r+1)*ch), img, image.Point{}, draw.Src)
		}
		label(sheet, 4, r*ch+4, asciiName(k))
	}
	return sheetPNG(path, sheet)
}
