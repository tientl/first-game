package client

import (
	"image"
	"image/color"
	"path/filepath"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The leaf tent's sheets, for a look at it outside the game:
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -tentsheet dir
//
// writes tent_sheet.png (every thatch three times as big and as at 1080p,
// with the bag icon) and tent_sleep.gif (someone going to bed in a tent,
// asleep and turning over, and getting up again, through the game's own
// drawing). It must run inside the game loop (Draw), where images can be
// drawn.

// tentGrounds is the ground each thatch is shown on.
var tentGrounds = [numTentLeaves]struct {
	tile sim.Tile
	col  color.RGBA
	name string
}{
	{sim.TileGrass, color.RGBA{92, 132, 74, 255}, "fern on grass"},
	{sim.TileForest, color.RGBA{64, 100, 58, 255}, "broad leaves in woods and marsh"},
	{sim.TileSand, color.RGBA{214, 190, 140, 255}, "palm on sand and desert"},
	{sim.TileSnow, color.RGBA{226, 234, 242, 255}, "fir on snow"},
}

// TentSheets writes the leaf tent's sheet and its GIF into dir.
func (g *Game) TentSheets(dir string) error {
	if g.art == nil {
		g.art = getArt()
	}
	if err := sheetPNG(filepath.Join(dir, "tent_sheet.png"), tentSheet()); err != nil {
		return err
	}
	return g.tentSleepGIF(filepath.Join(dir, "tent_sleep.gif"))
}

// tentSheet lays out every thatch three times as big, then as at 1080p,
// and the bag icon: at 96 (a 1080p slot), three times that, and the old
// icon.
func tentSheet() *image.RGBA {
	const z, pad = 3, 24
	cw, ch := tentW*HDDensity, tentHt*HDDensity
	w := int(numTentLeaves)*(cw*z+pad) + pad
	h := pad + 16 + ch*z + pad + 16 + max(ch, 96*z) + pad
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fill(img, img.Bounds(), color.RGBA{32, 34, 44, 255})
	put := func(c *canvas, x0, y0, s int, bg color.RGBA) {
		fill(img, image.Rect(x0, y0, x0+c.w*s, y0+c.h*s), bg)
		for y := 0; y < c.h*s; y++ {
			for x := 0; x < c.w*s; x++ {
				img.SetRGBA(x0+x, y0+y, over(img.RGBAAt(x0+x, y0+y), c.px[(y/s)*c.w+x/s]))
			}
		}
	}
	whole := func(v tentLeaf) *canvas {
		back, front := genTent(v)
		c := hdCanvas(tentW, tentHt)
		copy(c.px, back.px)
		for i, p := range front.px {
			if p.A > 0 {
				c.px[i] = overStraight(c.px[i], p)
			}
		}
		return c
	}
	y2 := pad + 16 + ch*z + pad
	x := pad
	for v := tentLeaf(0); v < numTentLeaves; v++ {
		c := whole(v)
		label(img, pad+int(v)*(cw*z+pad), pad, tentGrounds[v].name+" (times three)")
		put(c, pad+int(v)*(cw*z+pad), pad+16, z, tentGrounds[v].col)
		label(img, x, y2, "as at full hd")
		put(c, x, y2+16, 1, tentGrounds[v].col)
		x += cw + pad
	}
	// The bag icon.
	slot := color.RGBA{44, 46, 60, 255}
	icon := bakeIcon(icLeafTent, 96)
	label(img, x, y2, "bag icon")
	put(icon, x, y2+16, 1, slot)
	x += 96 + pad
	label(img, x, y2, "bag icon (times three)")
	put(icon, x, y2+16, z, slot)
	x += 96*z + pad
	old := tentOldCanvas()
	label(img, x, y2, "old icon")
	put(old, x, y2+16, 6, slot)
	return img
}

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// overStraight lays straight-alpha c over straight-alpha bg.
func overStraight(bg, c color.RGBA) color.RGBA {
	if c.A == 255 || bg.A == 0 {
		return c
	}
	a := float64(c.A) / 255
	ba := float64(bg.A) / 255 * (1 - a)
	oa := a + ba
	m := func(x, y uint8) uint8 { return uint8((float64(y)*a + float64(x)*ba) / oa) }
	return color.RGBA{m(bg.R, c.R), m(bg.G, c.G), m(bg.B, c.B), uint8(oa * 255)}
}

// over lays straight-alpha c over opaque bg.
func over(bg, c color.RGBA) color.RGBA {
	a := float64(c.A) / 255
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-a) + float64(y)*a + 0.5) }
	return color.RGBA{m(bg.R, c.R), m(bg.G, c.G), m(bg.B, c.B), 255}
}

// tentReel is someone at a tent of thatch v: standing by it, going to bed
// in it, asleep and turning over each way (the clock jumped on to just
// before each turn), and getting up.
func (g *Game) tentReel(v tentLeaf) reel {
	const lead = 24
	turnAt := func(k int) int { return k*sleepTurnEvery - 211*int(sheetID) + bedLie }
	tent := protocol.EntityView{ID: 1<<30 + 91, Kind: sim.EntityLeafTent, TX: 40, TY: 40, Ready: true}
	at := func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
		g.sheetTent(tent, tentGrounds[v].tile)
		st := g.moveState(p.ID)
		st.blowAt = never
		p.Asleep = true
		age := t - lead
		switch {
		case t < lead: // standing at its door
			p.Asleep = false
			st.asleepAt, st.wokeAt, st.movedAt = never, never, g.frame-1000
			return
		case age < 100:
		case age < 170:
			age = turnAt(3) + age - 100 - 12 // onto the side, facing in
		case age < 240:
			age = turnAt(4) + age - 170 - 12 // back
		case age < 310:
			age = turnAt(5) + age - 240 - 12 // onto the other side
		}
		st.asleepAt, st.wokeAt = g.frame-age, never
		if t-lead >= 310 {
			p.Asleep = false
			st.wokeAt, st.asleepAt = g.frame-(t-lead-310), never
			st.movedAt = g.frame - 1000
		}
	}
	draw := func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		a := tentArtOf(v)
		putSprite(dst, a.back, d.sx, d.sy)
		if !p.Asleep {
			putSprite(dst, a.front, d.sx, d.sy)
		}
		g.drawPlayer(dst, p, d) // asleep, the shell goes over them: see drawSleeper
	}
	return reel{name: "tent", n: lead + 400, dirs: []sim.Dir{sim.DirDown}, at: at, draw: draw, bg: tentGrounds[v].col}
}

// sheetTent stands tent e in the world for the sheets, on ground tile.
func (g *Game) sheetTent(e protocol.EntityView, tile sim.Tile) {
	if g.entAt == nil {
		g.entAt = map[[2]int]int{}
	}
	key := [2]int{int(e.TX), int(e.TY)}
	if i, ok := g.entAt[key]; !ok || i >= len(g.curr.Entities) || g.curr.Entities[i].ID != e.ID {
		g.curr.Entities = append(g.curr.Entities, e)
		g.entAt[key] = len(g.curr.Entities) - 1
	}
	if i := int(e.TY)*g.width + int(e.TX); i >= 0 && i < len(g.tiles) {
		g.tiles[i] = tile
	}
}

// tentSleepGIF writes the tent reels side by side, twice as big, at 20
// frames a second.
func (g *Game) tentSleepGIF(path string) error {
	keepFrame := g.frame
	defer func() { g.frame = keepFrame }()
	buf := newFineBuf(cellW, cellH)
	vs := []tentLeaf{tentFern, tentBroad}
	const z = 2
	cw, ch := cellW*HDDensity, cellH*HDDensity
	var frames []*image.RGBA
	for vi, v := range vs {
		r := g.tentReel(v)
		g.resetMotions()
		for t := 0; t < r.n; t++ {
			img := g.render(r, sim.DirDown, t, buf)
			if t%3 != 0 {
				continue
			}
			k := t / 3
			if vi == 0 {
				frames = append(frames, image.NewRGBA(image.Rect(0, 0, len(vs)*cw*z, ch*z)))
			}
			f := frames[k]
			for y := 0; y < ch*z; y++ {
				for x := 0; x < cw*z; x++ {
					f.SetRGBA(vi*cw*z+x, y, img.RGBAAt(x/z, y/z))
				}
			}
		}
	}
	return sheetGIF(path, frames, 5)
}
