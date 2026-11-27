package client

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// What a player has in hand through a deed (motionpick.go): the food they
// eat, bitten smaller bite by bite with crumbs falling; the bowl they sip
// from; the flask tipped up; the bowl or flask dipped in the water, with
// rings spreading where it goes in. Drawn at the hands of the frame, so it
// goes where they go.

// thrustFrame is the frame of moThrust age frames into a spear's blow, on
// the server's clock (reach in swing.go): drawn back for two frames, out on
// the third, when the blow lands, held, and back.
func thrustFrame(age float64) int {
	switch {
	case age < 1:
		return 1
	case age < 2:
		return 2
	case age < 4:
		return 3
	case age < 6:
		return 4
	case age < 8:
		return 5
	case age < 10:
		return 6
	case age < 13:
		return 7
	}
	return 0
}

// startOf is how many frames (at 60 a second) into m frame f begins.
func (m motionID) startOf(f int) float64 {
	t := 0
	for i := 0; i < f && i < len(motions[m].ms); i++ {
		t += motions[m].ms[i]
	}
	return float64(t) * 60 / 1000
}

// deedItems sets what f holds through frame fr of motion m, a deed of p.
func (g *Game) deedItems(dst drawTarget, f *figDraw, p protocol.PlayerView, m motionID, fr int) {
	st := g.moveState(p.ID)
	k := st.deedItem
	dir := f.up.dir
	side := facingSide(dir)
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	age := float64(g.frame - st.deedAt)
	switch m {
	case moEatBite, moEatHand, moEatBowl, moDrink, moScoop, moThrow:
		f.tool = false // what the deed holds instead
	}
	switch m {
	case moEatBite, moEatHand:
		bites := 0
		for _, b := range [2]int{1, 4} {
			if fr >= b {
				bites++
			}
		}
		scale := 0.36
		if m == moEatHand {
			scale = 0.27
		}
		f.item = func(dst drawTarget, hx, hy, _ float64) {
			if fr == 7 && m == moEatHand {
				return // all gone
			}
			eaten := float64(bites) / 3
			if m == moEatHand {
				eaten = 0 // a handful at a time: the hand is refilled
			}
			cx, cy := hx+side*1.5, hy-1
			if !sideways {
				cx, cy = hx, hy-1.5
			}
			g.drawFood(dst, k, dir, cx, cy, scale, eaten)
			for _, b := range [2]int{1, 4} { // crumbs from each bite
				if t := age - m.startOf(b); t >= 0 && t < 22 {
					mx, my := hx-side*2, hy-2
					if !sideways {
						mx, my = hx, hy-3
					}
					crumbs(dst, k, mx, my, side, t, uint32(p.ID)*7+uint32(b))
				}
			}
		}
		if m == moEatHand {
			f.pair = func(dst drawTarget, _, _, ox, oy float64) { // the rest, in the other hand
				if fr < 6 && dir != sim.DirUp {
					g.drawFood(dst, k, dir, ox+side*0.5, oy-1, 0.3, 0)
				}
			}
		}
	case moEatBowl:
		tip := [6]float64{0, 0, 0.6, 1, 0.8, 0}[fr]
		f.item = nil
		f.pair = func(dst drawTarget, hx, hy, ox, oy float64) {
			x, y := (hx+ox)/2, (hy+oy)/2-1
			if sideways {
				x, y = hx+side*1.5, hy-1
			}
			if dir == sim.DirUp {
				return // before the face, out of sight
			}
			g.drawIconAny(dst, k, x, y, 0.42, side, -side*0.9*tip)
		}
	case moDrink:
		tip := [7]float64{0, 0.2, 1, 1, 0.9, 0, 0}[fr]
		f.item = func(dst drawTarget, hx, hy, _ float64) {
			if dir == sim.DirUp && tip > 0.5 {
				return
			}
			g.drawIconAny(dst, k, hx+side*1.5, hy-1.5, 0.4, side, -side*(0.3+1.5*tip))
		}
	case moScoop:
		vessel := k
		if fr < 3 { // still empty
			vessel = sim.ItemClayBowl
			if k == sim.ItemDirtyWater {
				vessel = sim.ItemFlask
			}
		}
		f.item = func(dst drawTarget, hx, hy, _ float64) {
			g.drawIconAny(dst, vessel, hx+side*1.5, hy, 0.4, side, 0)
			if t := age - m.startOf(2); t >= 0 && t < 70 { // where it went in
				ripples(dst, hx+side*2, hy+2, t)
			}
			if fr == 4 || fr == 5 { // dripping as it comes out
				for i := 0; i < 2; i++ {
					ph := math.Mod(age/9+float64(i)*0.5, 1)
					fillRect(dst, float32(math.Round(hx+side*(1+float64(i)))), float32(math.Round(hy+2+ph*6)), 1, 1,
						premul(color.RGBA{170, 210, 235, 255}, 0.8*(1-ph)), false)
				}
			}
		}
	case moThrow:
		f.item = func(dst drawTarget, hx, hy, _ float64) {
			if fr < 4 { // let go on frame 4: the pile flies from there (drops.go)
				g.drawIconAny(dst, k, hx+side, hy, 0.38, side, 0)
			}
		}
	}
}

// heldIcon is item k's HD icon for drawing size art pixels across in the
// world, and true; or false if it has none (iconhd.go).
func heldIcon(k sim.ItemKind, size float64) (*ebiten.Image, bool) {
	img := hdIcon(k, max(8, int(math.Round(size*HDDensity))))
	return img, img != nil
}

// drawIcon draws item k's HD icon at (x, y), scale of its bag size (16 art
// pixels), mirrored for side -1 and turned by turn. It reports false, and
// draws nothing, for an item that has no HD icon.
func (g *Game) drawIcon(dst drawTarget, k sim.ItemKind, x, y, scale, side, turn float64) bool {
	if !k.Valid() {
		return false
	}
	img, ok := heldIcon(k, 16*scale)
	if !ok {
		return false
	}
	n := float64(img.Bounds().Dx())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-n/2, -n/2)
	op.GeoM.Scale(side/HDDensity, 1.0/HDDensity)
	op.GeoM.Rotate(turn)
	op.GeoM.Translate(x, y)
	if turn == 0 {
		op.Filter = ebiten.FilterNearest
	} else {
		op.Filter = ebiten.FilterLinear
	}
	drawImageD(dst, img, scaleBack(op), HDDensity)
	return true
}

// scaleBack undoes the 1/HDDensity drawImageD puts in front of op: drawIcon
// writes its GeoM in art pixels already.
func scaleBack(op *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {
	var m ebiten.GeoM
	m.Scale(HDDensity, HDDensity)
	m.Concat(op.GeoM)
	op.GeoM = m
	return op
}

// drawIconAny is drawIcon, falling back on the old 16 x 16 icon redrawn
// three times as fine (iconFine), or, with the HD held items off, the old
// icon itself drawn smooth at that scale.
func (g *Game) drawIconAny(dst drawTarget, k sim.ItemKind, x, y, scale, side, turn float64) {
	if g.drawIcon(dst, k, x, y, scale, side, turn) || !k.Valid() {
		return
	}
	if fine := g.iconFine(k); fine != nil && heldHD {
		w := float64(fine.Bounds().Dx()) / HDDensity // in art pixels: 16
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-w/2, -w/2)
		op.GeoM.Scale(scale*side, scale)
		op.GeoM.Rotate(turn)
		op.GeoM.Translate(x, y)
		op.Filter = ebiten.FilterLinear
		drawImageD(dst, fine, op, HDDensity)
		return
	}
	icon := g.art.icons[k]
	if icon == nil {
		return
	}
	b := icon.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(scale*side, scale)
	op.GeoM.Rotate(turn)
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(icon, op)
}

// drawFood draws food k held at (x, y), eaten of it (0 to 1) bitten away
// from the end nearest the mouth: sideways the end towards the face, from
// the front its top.
func (g *Game) drawFood(dst drawTarget, k sim.ItemKind, dir sim.Dir, x, y, scale, eaten float64) {
	img, hd := heldIcon(k, 16*scale)
	dens := HDDensity
	sc := 1.0 // what is left to scale it by
	if !hd {
		img, dens, sc = g.art.icons[k], 1, scale
		if fine := g.iconFine(k); fine != nil && heldHD {
			img, dens, hd = fine, HDDensity, true
		}
		if img == nil {
			return
		}
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	side := facingSide(dir)
	r := image.Rect(0, 0, w, h)
	if eaten > 0 {
		cut := int(math.Round(eaten * 0.55 * float64(w)))
		if dir == sim.DirLeft || dir == sim.DirRight {
			r.Min.X = cut
		} else {
			r.Min.Y = cut
		}
	}
	sub := img.SubImage(r.Add(b.Min)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	if hd { // drawImageD takes the image to art pixels first
		op.GeoM.Translate((float64(r.Min.X)-float64(w)/2)/HDDensity, (float64(r.Min.Y)-float64(h)/2)/HDDensity)
		op.GeoM.Scale(sc*side, sc)
		op.GeoM.Translate(x, y)
		op.Filter = ebiten.FilterNearest
		if sc != 1 {
			op.Filter = ebiten.FilterLinear
		}
		drawImageD(dst, sub, op, dens)
		return
	}
	op.GeoM.Translate(float64(r.Min.X)-float64(w)/2, float64(r.Min.Y)-float64(h)/2)
	op.GeoM.Scale(scale*side, scale)
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(sub, op)
}

// crumbs draws the crumbs a bite of k scatters from the mouth at (x, y), t
// frames after it.
func crumbs(dst drawTarget, k sim.ItemKind, x, y, side, t float64, seed uint32) {
	col := crumbColor(k)
	for i := 0; i < 4; i++ {
		vx := side * (0.15 + 0.35*hashf(i, 1, seed))
		vy := -0.25 - 0.3*hashf(i, 2, seed)
		px := x + vx*t + (hashf(i, 3, seed)-0.5)*2
		py := y + vy*t + 0.018*t*t
		a := 1 - t/22
		fillRect(dst, float32(math.Round(px)), float32(math.Round(py)), 1, 1, premul(col, a), false)
	}
}

// crumbColor is what the bits of k that fall look like.
func crumbColor(k sim.ItemKind) color.RGBA {
	switch k {
	case sim.ItemBerry, sim.ItemNightshade:
		return color.RGBA{120, 40, 90, 255}
	case sim.ItemBread, sim.ItemFruitPie, sim.ItemGinkgoNut, sim.ItemCorn, sim.ItemRice:
		return color.RGBA{214, 170, 104, 255}
	case sim.ItemFish, sim.ItemDriedFish, sim.ItemHerbFish:
		return color.RGBA{224, 204, 170, 255}
	}
	return color.RGBA{150, 70, 50, 255} // meat
}

// ripples draws rings spreading on the water from (x, y), t frames after
// something went in.
func ripples(dst drawTarget, x, y, t float64) {
	for k := 0; k < 2; k++ {
		u := (t - float64(k)*14) / 56
		if u < 0 || u > 1 {
			continue
		}
		r := 1.5 + u*7
		a := 0.75 * (1 - u)
		for s := 0; s < 18; s++ {
			ang := 2 * math.Pi * float64(s) / 18
			fillRect(dst, float32(math.Round(x+r*math.Cos(ang))), float32(math.Round(y+r*0.42*math.Sin(ang))), 1, 1,
				premul(color.RGBA{225, 240, 250, 255}, a), false)
		}
	}
}
