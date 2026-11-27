package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The minimap, under the clock in the top right corner, shows the ground
// around the player with north up. It is cut from the same baked map and
// fog as the full map (M), so it only shows what the player has seen.

const (
	miniW, miniH = 188, 141 // the minimap's inside, in screen pixels: 4:3 like a tile
	miniTiles    = 36       // tiles across the minimap
	miniPad      = 6
	miniTop      = 70 // the map's frame, just under the clock in the same block; see hud.go
)

// miniBottom is the screen y just below the minimap, for what goes under it.
func miniBottom() float64 { return miniTop + miniH + 2*miniPad }

// miniScale is the minimap's size of one tile, in screen pixels.
func miniScale() (tw, th float64) {
	tw = miniPx // whole pixels, so the minimap's pieces are drawn one to one: see terrainmap.go
	return tw, tw * TileH / TileW
}

// miniOrigin is where map tile (0, 0) lands on the minimap's canvas when it
// is centred on (cx, cy), in tiles. It is whole pixels, so nothing shimmers.
func miniOrigin(cx, cy float64) (ox, oy float64) {
	tw, th := miniScale()
	return math.Round(miniW/2 - cx*tw), math.Round(miniH/2 - cy*th)
}

// drawMinimap draws the minimap in the clock's block, whose left is x0.
func (g *Game) drawMinimap(screen *uiDst, x0 float64) {
	m := &g.wmap
	if m.mini == nil {
		m.mini = ebiten.NewImage(miniW, miniH)
	}
	c := m.mini

	// Centre on the player; before the first snapshot has them, the map's middle.
	alpha := g.alpha()
	cx, cy := float64(g.width)/2, float64(g.height)/2
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			cx, cy, _ = g.lerpPlayer(p, alpha)
		}
	}
	tw, th := miniScale()
	ox, oy := miniOrigin(cx, cy)
	at := func(x, y float64) (float32, float32) { return float32(ox + x*tw), float32(oy + y*th) }

	// Off the map is unknown sea: fog, like the unseen parts of the map.
	c.Fill(colFog)
	mw, mh := float64(g.width)*tw, float64(g.height)*th
	fog := m.fogImage()
	if img := m.img; img != nil && fog != nil {
		if !m.minis.draw(c, ox, oy, g.frame) {
			// Until its pieces are baked, the whole map scaled down.
			b := img.Bounds()
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(mw/float64(b.Dx()), mh/float64(b.Dy()))
			op.GeoM.Translate(ox, oy)
			op.Filter = ebiten.FilterLinear
			c.DrawImage(img, op)
			m.minis.draw(c, ox, oy, g.frame)
		}
	} else if fog != nil {
		fillRect(c, float32(ox), float32(oy), float32(mw), float32(mh), colWaterBG, false)
	}
	if fog != nil {
		b := fog.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(mw/float64(b.Dx()), mh/float64(b.Dy()))
		op.GeoM.Translate(ox, oy)
		op.Filter = ebiten.FilterLinear
		c.DrawImage(fog, op)
	}

	for _, f := range g.fires {
		x, y := at(float64(f.tx)+0.5, float64(f.ty)+0.5)
		drawFireMark(c, x, y)
	}
	for _, p := range g.curr.Players {
		if p.ID == g.myID || p.Dead {
			continue
		}
		wx, wy, _ := g.lerpPlayer(p, alpha)
		x, y := at(wx, wy)
		g.drawHeadMark(c, x, y, 1, playerColors[int(p.ID)%len(playerColors)], p.Look)
	}
	pulse := float32(g.frame%60) / 60
	strokeCircle(c, miniW/2, miniH/2, 9+pulse*8, 1.5, color.RGBA{255, 255, 255, uint8(180 * (1 - pulse))}, true)
	g.drawHeadMark(c, miniW/2, miniH/2, 1, playerColors[int(g.myID)%len(playerColors)], g.myLook())

	// The frame, then the canvas in it, with the key for the full map.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x0+miniPad, miniTop+miniPad)
	screen.DrawImage(c, op)
	strokeRect(screen, float32(x0+miniPad), miniTop+miniPad, miniW, miniH, 1, color.RGBA{0, 0, 0, 120}, false)
	kx, ky := float32(x0+miniPad+miniW-19), float32(miniTop+miniPad+miniH-19)
	fillRoundRect(screen, kx, ky, 15, 15, 4, color.RGBA{14, 16, 26, 230})
	g.text(screen, "M", float64(kx)+7.5, float64(ky), colKey, text.AlignCenter, 1)
}
