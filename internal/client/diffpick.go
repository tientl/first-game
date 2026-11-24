package client

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/sim"
)

// This file is the difficulty picker of the main menu: the three levels as
// cards side by side, each under a little painted scene, and "Quay lại"
// below them. The panel is laid out at one size and drawn into a buffer,
// which is shrunk to fit a window too small for it.

// diffBack is what the picker's update returns for "Quay lại"; the levels
// are 0 to diffBack-1, in the order of difficulties.
const diffBack = int(sim.NumDifficulties)

// The panel's layout, in its own pixels.
const (
	diffPad    = 20
	diffCardW  = 204
	diffCardH  = 290
	diffGap    = 12
	diffCardY  = 60
	diffSceneW = 192 // the scene on a card: a 96 by 56 painting, doubled
	diffSceneH = 112
	diffBackW  = 240
	diffBackH  = 44
	diffBackY  = diffCardY + diffCardH + 14
	diffW      = 2*diffPad + 3*diffCardW + 2*diffGap
	diffH      = diffBackY + diffBackH + 38
)

// diffCard is what one level's card says. The figures are those of
// sim/difficulty.go (diffTunes), written as the game writes numbers.
type diffCard struct {
	accent color.RGBA
	stats  [4]string // beasts' blows, hunger, cold, night-hunter packs
}

var diffStatNames = [4]string{"Thú đánh", "Đói", "Rét đêm đông", "Bầy săn đêm"}

var diffCards = [diffBack]diffCard{
	sim.DiffBasic:   {hex(0x8fd66a), [4]string{"x1", "x1", "x1", "2"}},
	sim.DiffSkilled: {hex(0xf4b548), [4]string{"x1,5", "x1,3", "x1,5", "3"}},
	sim.DiffLord:    {hex(0xf2644e), [4]string{"x2", "x1,6", "x2", "4"}},
}

// diffDeath is what dying costs at difficulty d, as the card says it.
func diffDeath(d sim.Difficulty) string {
	if d.KeepsItems() {
		return "Chết không mất đồ"
	}
	return "Chết mất đồ, lấy lại ở xác"
}

// diffKeys is one frame of the keys the picker listens to.
type diffKeys struct{ left, right, up, down, tab, ok bool }

type diffPicker struct {
	sel int // the card under the selection, or diffBack
	col int // the card last selected, to come back to from "Quay lại"

	frame  int
	buf    *ebiten.Image
	scenes [diffBack]diffScene

	// Where draw put the panel on the screen, for the mouse.
	ox, oy, scale  float64
	lastMX, lastMY int
}

// diffCardRect is where card i lies in the panel.
func diffCardRect(i int) image.Rectangle {
	x := diffPad + i*(diffCardW+diffGap)
	return image.Rect(x, diffCardY, x+diffCardW, diffCardY+diffCardH)
}

func diffBackRect() image.Rectangle {
	x := (diffW - diffBackW) / 2
	return image.Rect(x, diffBackY, x+diffBackW, diffBackY+diffBackH)
}

// hit is the card or button at panel point (x, y), or -1.
func diffHit(x, y float64) int {
	pt := image.Pt(int(math.Floor(x)), int(math.Floor(y)))
	for i := range diffBack {
		if pt.In(diffCardRect(i)) {
			return i
		}
	}
	if pt.In(diffBackRect()) {
		return diffBack
	}
	return -1
}

// press moves the selection as the keys ask and reports whether the
// selection was picked. Left and right run along the row of cards, wrapping;
// up and down go between the row and "Quay lại".
func (p *diffPicker) press(k diffKeys) bool {
	if p.sel < diffBack {
		p.col = p.sel
	}
	switch {
	case k.left && p.sel < diffBack:
		p.sel = (p.sel + diffBack - 1) % diffBack
	case k.right && p.sel < diffBack:
		p.sel = (p.sel + 1) % diffBack
	case k.up || k.down:
		if p.sel < diffBack {
			p.sel = diffBack
		} else {
			p.sel = p.col
		}
	case k.tab:
		p.sel = (p.sel + 1) % (diffBack + 1)
	}
	if p.sel < diffBack {
		p.col = p.sel
	}
	return k.ok
}

// update handles input and returns the level picked (0 to diffBack-1),
// diffBack for "Quay lại", or -1.
func (p *diffPicker) update() int {
	p.frame++
	prev := p.sel
	i := p.handle()
	if p.sel != prev {
		snd.ui("ui_move")
	}
	if i >= 0 {
		snd.ui("ui_select")
	}
	return i
}

func (p *diffPicker) handle() int {
	pressed := func(keys ...ebiten.Key) bool {
		for _, k := range keys {
			if inpututil.IsKeyJustPressed(k) {
				return true
			}
		}
		return false
	}
	if p.press(diffKeys{
		left:  pressed(ebiten.KeyArrowLeft, ebiten.KeyA),
		right: pressed(ebiten.KeyArrowRight, ebiten.KeyD),
		up:    pressed(ebiten.KeyArrowUp, ebiten.KeyW),
		down:  pressed(ebiten.KeyArrowDown, ebiten.KeyS),
		tab:   pressed(ebiten.KeyTab),
		ok:    pressed(ebiten.KeyEnter, ebiten.KeySpace, ebiten.KeyNumpadEnter),
	}) {
		return p.sel
	}
	if p.scale == 0 {
		return -1 // not drawn yet
	}
	mx, my := ebiten.CursorPosition()
	moved := mx != p.lastMX || my != p.lastMY
	p.lastMX, p.lastMY = mx, my
	i := diffHit((float64(mx)-p.ox)/p.scale, (float64(my)-p.oy)/p.scale)
	if i < 0 {
		return -1
	}
	if moved {
		p.sel = i // hovering selects, but a still mouse does not fight the keys
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.sel = i
		return i
	}
	return -1
}

// diffFit is how much the panel is shrunk to fit a screen of sw by sh, and
// where its top-left corner goes.
func diffFit(sw, sh int) (scale, ox, oy float64) {
	scale = min(1, float64(sw-12)/diffW, float64(sh-12)/diffH)
	ox = math.Round((float64(sw) - diffW*scale) / 2)
	oy = math.Round((float64(sh) - diffH*scale) / 2)
	return scale, ox, oy
}

func (p *diffPicker) draw(screen *ebiten.Image) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	vector.FillRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, 150}, false)
	if p.buf == nil {
		p.buf = ebiten.NewImage(diffW, diffH)
	}
	b := p.buf
	b.Clear()
	fillRoundRect(b, 0, 0, diffW, diffH, 12, color.RGBA{0, 0, 0, 120})
	fillRoundRect(b, 2, 2, diffW-4, diffH-4, 10, color.RGBA{18, 20, 32, 245})
	drawText(b, "Chọn độ khó", diffW/2, 16, colWhite, text.AlignCenter, 2)
	for i := range diffBack {
		p.drawCard(b, i)
	}

	r := diffBackRect()
	bg, edge := color.RGBA{14, 16, 26, 200}, color.RGBA{0, 0, 0, 90}
	if p.sel == diffBack {
		bg, edge = color.RGBA{58, 46, 22, 235}, colKey
	}
	fillRoundRect(b, float32(r.Min.X)-2, float32(r.Min.Y)-2, diffBackW+4, diffBackH+4, 10, edge)
	fillRoundRect(b, float32(r.Min.X), float32(r.Min.Y), diffBackW, diffBackH, 8, bg)
	if p.sel == diffBack {
		drawText(b, ">", float64(r.Min.X)+16, float64(r.Min.Y)+10, colKey, text.AlignStart, 2)
	}
	drawText(b, "Quay lại", diffW/2, float64(r.Min.Y)+10, colWhite, text.AlignCenter, 2)
	drawText(b, "A/D hoặc chuột để chọn · Enter để xác nhận · Esc để quay lại", diffW/2, diffH-26, colMuted, text.AlignCenter, 1)

	p.scale, p.ox, p.oy = diffFit(sw, sh)
	op := &ebiten.DrawImageOptions{}
	if p.scale != 1 {
		op.Filter = ebiten.FilterLinear
		op.GeoM.Scale(p.scale, p.scale)
	}
	op.GeoM.Translate(p.ox, p.oy)
	screen.DrawImage(b, op)
}

func (p *diffPicker) drawCard(b *ebiten.Image, i int) {
	card, d := diffCards[i], difficulties[i]
	r := diffCardRect(i)
	x, y := float32(r.Min.X), float32(r.Min.Y)
	sel := p.sel == i
	bg, edge := color.RGBA{14, 16, 26, 220}, color.RGBA{0, 0, 0, 110}
	if sel {
		bg, edge = color.RGBA{58, 46, 22, 240}, colKey
	}
	fillRoundRect(b, x-2, y-2, diffCardW+4, diffCardH+4, 10, edge)
	fillRoundRect(b, x, y, diffCardW, diffCardH, 8, bg)

	// The scene, in a thin frame of the level's colour.
	const inset = (diffCardW - diffSceneW) / 2
	sx, sy := r.Min.X+inset, r.Min.Y+inset
	frame := mix(card.accent, colBlack, 0.55)
	if sel {
		frame = card.accent
	}
	vector.FillRect(b, float32(sx)-1, float32(sy)-1, diffSceneW+2, diffSceneH+2, frame, false)
	view := b.SubImage(image.Rect(sx, sy, sx+diffSceneW, sy+diffSceneH)).(*ebiten.Image)
	p.drawScene(view, i, sx, sy, sel)
	if !sel {
		vector.FillRect(b, float32(sx), float32(sy), diffSceneW, diffSceneH, color.RGBA{8, 10, 20, 70}, false)
	}

	// The name, and the level as pips: one, two, three.
	cx := float64(r.Min.X) + diffCardW/2
	ty := float64(sy) + diffSceneH + 8
	drawText(b, d.Name(), cx, ty, card.accent, text.AlignCenter, 2)
	for j := range diffBack {
		c := color.RGBA{70, 74, 90, 255}
		if j <= i {
			c = card.accent
		}
		px := float32(cx) + float32(j-1)*14 - 4
		vector.FillRect(b, px+1, float32(ty)+37, 8, 4, color.RGBA{0, 0, 0, 160}, false)
		vector.FillRect(b, px, float32(ty)+36, 8, 4, c, false)
	}

	// What the level changes.
	const pad = 14
	ly := ty + 50
	for j, name := range diffStatNames {
		if j%2 == 0 {
			vector.FillRect(b, x+6, float32(ly)-1, diffCardW-12, 18, color.NRGBA{255, 255, 255, 14}, false)
		}
		drawText(b, name, float64(r.Min.X)+pad, ly, colMuted, text.AlignStart, 1)
		drawText(b, card.stats[j], float64(r.Max.X)-pad, ly, card.accent, text.AlignEnd, 1)
		ly += 18
	}
	ly += 6
	death := colWhite
	if !d.KeepsItems() {
		death = color.RGBA{255, 190, 170, 255}
	}
	for _, line := range wrapWords(diffDeath(d), diffCardW-2*pad) {
		drawText(b, line, cx, ly, death, text.AlignCenter, 1)
		ly += 16
	}
}

// wrapWords breaks s at spaces into lines no wider than w.
func wrapWords(s string, w float64) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		if line != "" && textWidth(line+" "+word, 1) > w {
			lines = append(lines, line)
			line = word
			continue
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// --- the scenes --------------------------------------------------------------

// diffScene is the painted part of a card's scene: what lies behind the
// beasts and what stands in front of them. Both are 96 by 56, drawn doubled.
type diffScene struct{ back, front *ebiten.Image }

const (
	diffArtW   = diffSceneW / 2
	diffArtH   = diffSceneH / 2
	diffGround = 36 // where the land meets the sky, in art pixels
)

var (
	diffDaySky   = mkRamp(0x4a9bd6, 0x66b0e2, 0x8cc8ee, 0xb6def5, 0xdff1f6)
	diffDuskSky  = mkRamp(0x2c2347, 0x5a2f55, 0x9a4553, 0xd96f44, 0xf5a742, 0xffd873)
	diffDuskLand = mkRamp(0x1c1a1e, 0x2e2a22, 0x4a3f26, 0x6b592c, 0x8f7a35)
	diffDoomSky  = mkRamp(0x1a0a12, 0x3a0d16, 0x6e121a, 0xa81e1a, 0xe2481a, 0xff8a26)
	diffDoomLand = mkRamp(0x0e080c, 0x1a0e12, 0x2a1418, 0x40191a)
)

// diffSky fills rows 0 to h of c from the top of rp to its end.
func diffSky(c *canvas, rp ramp, h int, curve float64) {
	c.rect(0, 0, c.w, h, func(x, y int) color.RGBA {
		return rp.pick(math.Pow(float64(y)/float64(h-1), curve), x, y)
	})
}

// diffRidge fills c below the line top(x) with what f gives for the depth
// below it.
func diffRidge(c *canvas, top func(x float64) float64, f func(x, y int, depth float64) color.RGBA) {
	for x := 0; x < c.w; x++ {
		t := top(float64(x))
		for y := max(0, int(math.Ceil(t))); y < c.h; y++ {
			c.put(x, y, f(x, y, float64(y)-t))
		}
	}
}

// diffBlades plants grass blades along the bottom of c, each leaning a
// little, from the dark of rp at the root to its light at the tip.
func diffBlades(c *canvas, rp ramp, n int, minH, maxH float64, seed uint32) {
	for i := range n {
		x := (float64(i) + hashf(i, 0, seed)) * float64(c.w) / float64(n)
		h := lerpf(minH, maxH, hashf(i, 1, seed))
		lean := (hashf(i, 2, seed) - 0.5) * 5
		tone := hashf(i, 3, seed) * 0.35
		c.line(x, float64(c.h), x+lean, float64(c.h)-h, func(_, _ int, t float64) color.RGBA {
			return rp.at(int((t*0.75 + tone) * float64(len(rp))))
		})
	}
}

// genDiffScene paints the back and front of level i's scene.
func genDiffScene(i int) diffScene {
	back, front := newCanvas(diffArtW, diffArtH), newCanvas(diffArtW, diffArtH)
	switch sim.Difficulty(i) {
	case sim.DiffBasic: // a clear morning over a meadow
		diffSky(back, diffDaySky, diffGround+2, 0.8)
		back.ellipse(17, 11, 9, 9, func(x, y int, nx, ny float64) color.RGBA {
			if d := math.Hypot(nx, ny); d > 0.62 && d-0.62 > dither(x, y)*0.38 {
				return color.RGBA{}
			}
			return hex(0xfff6c8)
		})
		back.ellipse(17, 11, 4.6, 4.6, func(int, int, float64, float64) color.RGBA { return hex(0xfffbe6) })
		for _, cl := range [][4]float64{{52, 9, 9, 2.6}, {59, 7, 6, 2.6}, {66, 10, 7, 2}, {84, 18, 8, 2}, {90, 16, 5, 2.2}, {36, 21, 6, 1.5}} {
			back.ellipse(cl[0], cl[1], cl[2], cl[3], func(x, y int, nx, ny float64) color.RGBA {
				if ny > 0.35 {
					return hex(0xd6ebf5)
				}
				return colWhite
			})
		}
		diffRidge(back, func(x float64) float64 { return 28 + 3.5*math.Sin(x*0.07+0.6) + 1.5*math.Sin(x*0.19) }, func(x, y int, d float64) color.RGBA {
			return mix(hex(0x7fb6a6), hex(0x6aa58f), clamp01(d/6))
		})
		diffRidge(back, func(x float64) float64 { return 33 + 2.5*math.Sin(x*0.09+2.4) }, func(x, y int, d float64) color.RGBA {
			return mix(hex(0x5fa56a), hex(0x4f9458), clamp01(d/5))
		})
		diffRidge(back, func(x float64) float64 { return diffGround + 0.8*math.Sin(x*0.21) }, func(x, y int, d float64) color.RGBA {
			t := 0.85 - d/26 + (vnoise(float64(x)*0.35, float64(y)*0.6, 31)-0.5)*0.5
			return grassRamp.pick(t, x, y)
		})
		for j := range 22 { // flowers and tufts
			x, y := int(hashf(j, 0, 33)*diffArtW), diffGround+3+int(hashf(j, 1, 33)*16)
			switch j % 3 {
			case 0:
				back.put(x, y, hex(0xfff3a0))
			case 1:
				back.put(x, y, hex(0xffffff))
			default:
				back.put(x, y, grassRamp[4])
				back.put(x, y+1, grassRamp[1])
			}
		}
		diffBlades(front, grassRamp, 30, 2, 5, 34)
	case sim.DiffSkilled: // dusk: something moves in the long grass
		diffSky(back, diffDuskSky, diffGround+2, 1.25)
		back.ellipse(45, 28, 12, 12, func(x, y int, nx, ny float64) color.RGBA {
			if d := math.Hypot(nx, ny); d > 0.7 && d-0.7 > dither(x, y)*0.3 {
				return color.RGBA{}
			}
			return hex(0xffe9a0)
		})
		back.ellipse(45, 28, 7.5, 7.5, func(int, int, float64, float64) color.RGBA { return hex(0xfff6d6) })
		for _, cl := range [][4]float64{{20, 12, 12, 1.6}, {30, 15, 9, 1.2}, {80, 9, 10, 1.4}, {70, 19, 14, 1}} {
			back.ellipse(cl[0], cl[1], cl[2], cl[3], func(int, int, float64, float64) color.RGBA { return hex(0x7a3552) })
		}
		// A far line of trees, black against the light.
		diffRidge(back, func(x float64) float64 {
			return 30 - 5*vnoise(x*0.16, 0, 41) - 2.5*math.Abs(math.Sin(x*0.9)) + 4*clamp01((x-10)/20)*clamp01((66-x)/20)
		}, func(int, int, float64) color.RGBA { return hex(0x3a2338) })
		diffRidge(back, func(x float64) float64 { return diffGround - 1 + math.Sin(x*0.3) }, func(x, y int, d float64) color.RGBA {
			t := 0.75 - d/22 + (vnoise(float64(x)*0.5, float64(y)*0.5, 42)-0.5)*0.4
			return diffDuskLand.pick(t, x, y)
		})
		tall := newCanvas(diffArtW, diffArtH)
		diffBlades(tall, diffDuskLand[1:], 46, 5, 13, 43)
		back.over(tall, 0, -7)
		diffBlades(front, diffDuskLand, 54, 7, 17, 44)
	default: // the tyrant's land, under a burning sky
		diffSky(back, diffDoomSky, diffGround+3, 1.5)
		for _, cl := range [][4]float64{{18, 6, 16, 2.2}, {40, 11, 13, 1.6}, {78, 5, 14, 2}, {88, 14, 10, 1.4}, {10, 17, 9, 1.2}} {
			back.ellipse(cl[0], cl[1], cl[2], cl[3], func(x, y int, nx, ny float64) color.RGBA {
				if ny > 0.2 {
					return hex(0x4a0f18)
				}
				return hex(0x24090f)
			})
		}
		// A volcano at the left, its mouth alight and lava down its flank.
		cone := func(x float64) float64 {
			return max(13, 13+math.Abs(x-22)*0.95-1.5*vnoise(x*0.4, 0, 51))
		}
		back.ellipse(22, 11, 7, 4, func(x, y int, nx, ny float64) color.RGBA {
			if math.Hypot(nx, ny) > dither(x, y)+0.35 {
				return color.RGBA{}
			}
			return lavaRamp[3]
		})
		diffRidge(back, cone, func(x, y int, d float64) color.RGBA {
			if d < 1.2 && math.Abs(float64(x)-22) < 4 {
				return lavaRamp[4]
			}
			return mix(hex(0x2a1016), hex(0x16090e), clamp01(float64(x-12)/18))
		})
		for y := 14; y < diffGround; y++ {
			x := 23 + int(math.Round(2.2*math.Sin(float64(y)*0.45)+float64(y-14)*0.25))
			back.put(x, y, lavaRamp.pick(1-float64(y-14)/26, x, y))
			if y%3 != 0 {
				back.put(x+1, y, lavaRamp[2])
			}
		}
		diffRidge(back, func(x float64) float64 {
			return 31 - 4*vnoise(x*0.2, 3, 52) + 5*clamp01((50-x)/40)
		}, func(int, int, float64) color.RGBA { return hex(0x210c12) })
		diffRidge(back, func(x float64) float64 { return diffGround + 0.7*math.Sin(x*0.4) }, func(x, y int, d float64) color.RGBA {
			if d < 1 {
				return hex(0x7a1c18) // the rim catches the glow
			}
			t := 0.8 - d/20 + (vnoise(float64(x)*0.4, float64(y)*0.7, 53)-0.5)*0.5
			return diffDoomLand.pick(t, x, y)
		})
		for j := range 7 { // cracks of fire in the ground
			x0, y0 := hashf(j, 0, 54)*diffArtW, diffGround+4+hashf(j, 1, 54)*14
			back.line(x0, y0, x0+4+hashf(j, 2, 54)*7, y0+(hashf(j, 3, 54)-0.5)*3, func(x, y int, t float64) color.RGBA {
				return lavaRamp.at(1 + int(3*math.Sin(t*math.Pi)))
			})
		}
		diffSkull(front, 12, 51)
		diffRidge(front, func(x float64) float64 {
			return 53 - 3*vnoise(x*0.3, 0, 55)*clamp01((x-30)/20)
		}, func(x, y int, d float64) color.RGBA {
			if d < 1 {
				return hex(0x3a1418)
			}
			return hex(0x12080c)
		})
	}
	return diffScene{back.image(), front.image()}
}

// diffSkull draws a small skull lying on the ground, its jaw at (x, y).
func diffSkull(c *canvas, x, y float64) {
	s := newCanvas(14, 12)
	s.ellipse(6.5, 4.5, 5.5, 4.5, func(px, py int, nx, ny float64) color.RGBA {
		if nx+ny > 0.75 {
			return colBoneSh
		}
		return colBone
	})
	s.rect(4, 7, 6, 3, func(px, py int) color.RGBA {
		if py == 9 && px%2 == 1 {
			return color.RGBA{}
		}
		return colBoneSh
	})
	for _, e := range [][2]int{{3, 4}, {4, 4}, {3, 5}, {4, 5}, {8, 4}, {9, 4}, {8, 5}, {9, 5}, {6, 7}} {
		s.put(e[0], e[1], colInk)
	}
	s.outline(0.85)
	c.over(s, int(x)-7, int(y)-10)
}

// drawScene draws level i's scene with its top-left at (x, y) of the
// buffer view is cut from. A selected card's scene moves.
func (p *diffPicker) drawScene(view *ebiten.Image, i, x, y int, sel bool) {
	if p.scenes[i].back == nil {
		p.scenes[i] = genDiffScene(i)
	}
	sc := p.scenes[i]
	layer := func(img *ebiten.Image) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(x), float64(y))
		view.DrawImage(img, op)
	}
	// put stands sprite s with its anchor at art pixel (ax, ay), tinted.
	put := func(s sprite, ax, ay float64, r, g, b float32) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-math.Round(s.ax), -math.Round(s.ay))
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(x)+ax*2, float64(y)+ay*2)
		op.ColorScale.Scale(r, g, b, 1)
		view.DrawImage(s.img, op)
	}
	t := 0
	if sel {
		t = p.frame
	}
	a := getArt()
	layer(sc.back)
	switch sim.Difficulty(i) {
	case sim.DiffBasic:
		put(a.trees[0][0], 12, 44, 1, 1, 1)
		put(a.bushFull, 34, 45, 1, 1, 1)
		put(a.campBack, 62, 48, 1, 1, 1)
		put(a.flames[(t/5)%flameFrames], 62, 48, 1, 1, 1)
		put(a.campFront, 62, 48, 1, 1, 1)
		put(a.beasts.rabbit[1][rabbitSit], 82, 50, 1, 1, 1)
		put(a.beasts.rabbit[0][rabbitSleep], 44, 52, 1, 1, 1)
	case sim.DiffSkilled:
		put(a.beasts.raptor[0][raptorStand], 5, 46, 0.3, 0.2, 0.27)
		pose := raptorStand
		if sel && (t/14)%2 == 1 {
			pose = raptorRunA
		}
		put(a.beasts.raptor[0][pose], 66, 53, 1, 0.86, 0.74)
	default:
		bob := 0.0
		if sel {
			bob = math.Round(math.Sin(float64(t) * 0.9)) // shaking with the roar
		}
		put(a.beasts.rex[0][rexRoar], 34+bob, 82, 1, 0.86, 0.8)
		for j := range 14 { // embers on the wind
			life := math.Mod(float64(t)*0.012+hashf(j, 0, 61), 1)
			ex := math.Mod(hashf(j, 1, 61)*diffArtW+life*18, diffArtW)
			ey := diffArtH - life*(30+hashf(j, 2, 61)*26)
			c := color.NRGBA(lavaRamp[2+j%3])
			c.A = uint8(255 * (1 - life*life))
			vector.FillRect(view, float32(x)+float32(math.Floor(ex))*2, float32(y)+float32(math.Floor(ey))*2, 2, 2, c, false)
		}
	}
	layer(sc.front)
}
