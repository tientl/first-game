package client

import (
	"image"
	"image/color"
	"image/draw"
	"path/filepath"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The combat sheets, for review: the shield on the figure in every facing
// (shield_sheet.png) and every effect of fx.go frame by frame
// (fx_sheet.png), drawn through the game's own code:
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -combatsheet dir
//
// It must run inside the game loop (Draw).

// CombatSheets writes the shield sheet and the effects sheet into dir.
func (g *Game) CombatSheets(dir string) error {
	keepFrame := g.frame
	defer func() { g.frame = keepFrame }()
	if err := g.shieldSheet(filepath.Join(dir, "shield_sheet.png")); err != nil {
		return err
	}
	if err := g.FireSheet(dir); err != nil {
		return err
	}
	return g.fxSheet(filepath.Join(dir, "fx_sheet.png"))
}

// shieldSheet draws the caveman with a sword in every facing: the shield on
// the back standing and walking, raised standing and walking, each twice as
// big as at 1080p.
func (g *Game) shieldSheet(path string) error {
	const z = 2
	cw, ch := cellW*HDDensity*z, cellH*HDDensity*z
	type col struct {
		name  string
		guard bool
		walk  bool
	}
	cols := []col{{"on the back", false, false}, {"walking", false, true}, {"raised", true, false}, {"raised walking", true, true}}
	sheet := image.NewRGBA(image.Rect(0, 0, len(cols)*cw, len(allDirs)*ch+20))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{36, 40, 36, 255}}, image.Point{}, draw.Src)
	buf := newFineBuf(cellW, cellH)
	for j, c := range cols {
		label(sheet, j*cw+6, 4, c.name)
		for i, d := range allDirs {
			g.resetMotions()
			var img *image.RGBA
			for t := 0; t < 9; t++ { // a few frames in, for the clocks that count them
				img = g.render(reel{bg: reelGrass, at: func(g *Game, t int, p *protocol.PlayerView, dd *drawable) {
					p.Held, p.Shield, p.Guard = sim.ItemCopperSword, true, c.guard
					if c.walk {
						p.X += float32(t) * 0.07 * float32(dirX(d))
						p.Y += float32(t) * 0.07 * float32(dirY(d))
						dd.moving = true
					}
				}}, d, t, buf)
			}
			blowUp(sheet, img, j*cw, 20+i*ch, z)
		}
	}
	return sheetPNG(path, sheet)
}

func dirX(d sim.Dir) int { x, _ := d.Delta(); return x }
func dirY(d sim.Dir) int { _, y := d.Delta(); return y }

// blowUp copies src onto dst at (x, y), z times bigger.
func blowUp(dst, src *image.RGBA, x, y, z int) {
	b := src.Bounds()
	for sy := 0; sy < b.Dy(); sy++ {
		for sx := 0; sx < b.Dx(); sx++ {
			c := src.RGBAAt(sx, sy)
			for dy := 0; dy < z; dy++ {
				for dx := 0; dx < z; dx++ {
					dst.SetRGBA(x+sx*z+dx, y+sy*z+dy, c)
				}
			}
		}
	}
}

// fxSheetRows is what the effects sheet shows, a row each.
var fxSheetRows = []struct {
	name string
	f    fxInst
	bg   color.RGBA
}{
	{"hit spark", fxInst{kind: fxSpark, z: 14, k: 1}, reelGrass},
	{"crit hit", fxInst{kind: fxCrit, z: 14, k: 1}, reelGrass},
	{"hit flash lines", fxInst{kind: fxLines, z: 16, k: 1}, reelGrass},
	{"claw swipe", fxInst{kind: fxClaw, z: 16, ang: 0.9, k: 1}, reelGrass},
	{"thrust", fxInst{kind: fxThrust, z: 14, k: 1}, reelGrass},
	{"spear sweep", fxInst{kind: fxSweep, z: 0, k: 1}, reelGrass},
	{"heavy cleave", fxInst{kind: fxCleave, z: 0, k: 1}, reelGrass},
	{"splash", fxInst{kind: fxSplash, z: 0, k: 1.4}, reelWater},
	{"dust (charge)", fxInst{kind: fxDust, z: 0, ang: 3.1, k: 1.2, pal: dustEarth}, reelGrass},
	{"stomp", fxInst{kind: fxStomp, z: 0, k: 1.2, pal: dustEarth}, reelGrass},
	{"leaves", fxInst{kind: fxLeaves, z: 20, k: 1}, reelGrass},
	{"impact bits", fxInst{kind: fxBits, z: 12, k: 1, pal: dustGrass}, reelGrass},
	{"shield block", fxInst{kind: fxBlock, z: 14, ang: 3.14, k: 1}, reelGrass},
	{"parry", fxInst{kind: fxParry, z: 14, ang: 3.14, k: 1}, reelGrass},
	{"footstep (sand)", fxInst{kind: fxStep, z: 0, ang: 3.1, k: 1, pal: dustSand}, color.RGBA{214, 190, 136, 255}},
}

// fxSheet draws each effect every few frames across a row, at 1080p.
func (g *Game) fxSheet(path string) error {
	const fw, fh = 64, 48 // a frame's cell, art pixels
	const cols = 12
	cw, ch := fw*HDDensity, fh*HDDensity
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cw+4, len(fxSheetRows)*(ch+4)))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{30, 32, 30, 255}}, image.Point{}, draw.Src)
	buf := newFineBuf(fw, fh)
	for i, row := range fxSheetRows {
		n := fxDefs[row.f.kind].frames
		step := max(1, (n+cols-1)/cols)
		for c := 0; c < cols; c++ {
			age := c * step
			if age >= n {
				break
			}
			buf.img.Fill(row.bg)
			// A ground line and a mark where the effect is.
			fillRect(buf, 0, fh-12, fw, 1, color.RGBA{0, 0, 0, 40}, false)
			var b fxBatch
			f := row.f
			f.seed = uint32(97 + i*31)
			drawFXAt(&b, f, fw/2, fh-12-f.z, fh-12, float64(age)/60)
			b.flush(buf)
			img := image.NewRGBA(buf.img.Bounds())
			buf.img.ReadPixels(img.Pix)
			draw.Draw(sheet, image.Rect(c*cw+2, i*(ch+4)+2, (c+1)*cw+2, i*(ch+4)+2+ch), img, image.Point{}, draw.Src)
		}
		label(sheet, 6, i*(ch+4)+4, row.name)
	}
	return sheetPNG(path, sheet)
}

// fireSheetItems is what the fire sheet shows, a row each.
var fireSheetItems = []struct {
	name string
	e    protocol.EntityView
	k    fireKind
	dy   float64 // the flames' root from the entity's point
}{
	{"campfire", protocol.EntityView{ID: 11, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel}, fireCamp, 1},
	{"campfire low", protocol.EntityView{ID: 12, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel / 5}, fireCampLow, 1},
	{"torch", protocol.EntityView{ID: 13, Kind: sim.EntityTorchStand, Ready: true, Fuel: 100}, fireTorch, 0},
	{"bonfire", protocol.EntityView{ID: 14, Kind: sim.EntityBonfire, Ready: true}, fireBonfire, 1},
	{"furnace", protocol.EntityView{ID: 15, Kind: sim.EntityFurnace, Ready: true, Fuel: 100}, fireMouth, -1},
	{"oven", protocol.EntityView{ID: 16, Kind: sim.EntityOven, Ready: true, Fuel: 100}, fireMouth, 0},
	{"pot on the fire", protocol.EntityView{ID: 17, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel, Var: uint8(sim.GearPot)}, fireCook, 1},
}

// FireSheet writes fire_compare.png into dir: each fire, old (left, three
// frames) and new (right, three frames and with its embers and smoke), at
// twice 1080p.
func (g *Game) FireSheet(dir string) error {
	keepFrame, keepHD := g.frame, fireHD
	defer func() { g.frame, fireHD = keepFrame, keepHD }()
	const fw, fh = 56, 56 // a cell, art pixels
	const z = 2
	cw, ch := fw*HDDensity*z, fh*HDDensity*z
	const per = 3
	bg := color.RGBA{58, 74, 52, 255}
	sheet := image.NewRGBA(image.Rect(0, 0, (2*per)*cw+24, len(fireSheetItems)*(ch+6)+24))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{28, 30, 28, 255}}, image.Point{}, draw.Src)
	label(sheet, 6, 6, "old")
	label(sheet, per*cw+24+6, 6, "new")
	buf := newFineBuf(fw, fh)
	for i, it := range fireSheetItems {
		e := it.e
		e.TX, e.TY = 40, 40
		g.curr.Entities = []protocol.EntityView{e}
		for side, hd := range []bool{false, true} {
			fireHD = hd
			for f := 0; f < per; f++ {
				g.frame = 1000 + f*fireStep*4
				buf.img.Fill(bg)
				d := drawable{sx: fw / 2, sy: fh - 12, idx: 0}
				align(buf, d.sx, d.sy)
				if hd {
					g.drawFireSmokeOf(buf, []fireSource{{d.sx, d.sy + it.dy, it.k, uint32(e.ID)}})
				}
				g.drawEntity(buf, e, d)
				if hd {
					src := fireSource{d.sx, d.sy + it.dy, it.k, uint32(e.ID)}
					if it.k == fireTorch {
						src.y = d.sy - g.art.tribal.torchStick.ay + 4
					}
					g.drawFireFXOf(buf, []fireSource{src}, 0.3)
				}
				img := image.NewRGBA(buf.img.Bounds())
				buf.img.ReadPixels(img.Pix)
				x := side*(per*cw+24) + f*cw
				blowUp(sheet, img, x, 24+i*(ch+6), z)
			}
		}
		label(sheet, 6, 24+i*(ch+6)+4, it.name)
	}
	g.curr.Entities = nil
	return sheetPNG(filepath.Join(dir, "fire_compare.png"), sheet)
}
