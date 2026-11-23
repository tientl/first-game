package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Player sprites. A player is drawn as four layers: hair that hangs behind
// the body, legs, the shirt (greyscale, tinted with the player's colour) and
// the top (head, hair, hands, belt). Every look (body, hair style, hair
// colour, skin) has its own set, drawn the first time someone with it is seen.

// hairColors are the hair colours on the character screen, by sim.Look.HairColor.
// The first is the original survivor's.
var hairColors = [sim.NumHairColors]struct {
	name string
	ramp ramp
}{
	{"Nâu đen", mkRamp(0x24150f, 0x3b2418, 0x573620, 0x74492b)},
	{"Đen", mkRamp(0x0e0c12, 0x1a1720, 0x29252f, 0x3c3744)},
	{"Nâu hạt dẻ", mkRamp(0x3d1c10, 0x5e2d17, 0x82421f, 0xa65b2b)},
	{"Vàng", mkRamp(0x6e4e22, 0xa07a34, 0xd0aa4e, 0xf0d68a)},
	{"Đỏ", mkRamp(0x561810, 0x8a2a14, 0xbc461e, 0xe2723a)},
	{"Bạch kim", mkRamp(0x5a5d6c, 0x878b9a, 0xb7bac6, 0xe8eaf0)},
	{"Xanh dương", mkRamp(0x18234e, 0x26387a, 0x3856a8, 0x5c80d2)},
	{"Hồng", mkRamp(0x66223f, 0x98345f, 0xc95688, 0xf08ab4)},
}

// skinTones are the skin colours on the character screen, by sim.Look.Skin,
// light to dark. The first is the original survivor's. blush is the cheeks.
var skinTones = [sim.NumSkinTones]struct {
	name  string
	ramp  ramp
	blush color.RGBA
}{
	{"Sáng", skinRamp, hex(0xe8907c)},
	{"Trung bình", mkRamp(0x8a4e36, 0xb47450, 0xd89c72, 0xeebb90), hex(0xd8806a)},
	{"Bánh mật", mkRamp(0x74402a, 0x9c6240, 0xc28660, 0xdca67c), hex(0xc86e58)},
	{"Ngăm", mkRamp(0x5e3322, 0x845034, 0xa66e4a, 0xc28c64), hex(0xb05e4a)},
	{"Nâu", mkRamp(0x4f2b1c, 0x703f28, 0x93593a, 0xb27652), hex(0xa05442)},
	{"Nâu sẫm", mkRamp(0x3c2016, 0x5a3220, 0x78482e, 0x96603e), hex(0x8a4a3a)},
}

// hairNames are the hair styles' names, by sim.HairStyle.
var hairNames = [sim.NumHairStyles]string{"Ngắn", "Dựng", "Bob", "Dài", "Đuôi ngựa", "Búi cao"}

// SetLook sets the look of the character made for a new game; it is sent
// to the server on joining. A player the save already knows keeps the
// saved look.
func (g *Game) SetLook(l sim.Look) { g.look = l }

// myLook is how this player looks, as the server last showed it.
func (g *Game) myLook() sim.Look {
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			return p.Look
		}
	}
	return g.look
}

func bodyName(female bool) string {
	if female {
		return "Nữ"
	}
	return "Nam"
}

var hairTie = hex(0xc83c3c)

// headroom is how many rows above the old 26-pixel-tall body the player
// sprites have, for spiky hair and buns.
const headroom = 4

// bodyArt is the sprite set of one look.
type bodyArt struct {
	back, legs, shirt, top [4][walkFrames]sprite // indexed by sim.Dir
	head                   *ebiten.Image         // facing the viewer: the map marker
}

// body returns the sprites of look l, drawing them on first use. It must be
// called inside the game loop, where images can be created.
func (a *art) body(l sim.Look) *bodyArt {
	l = l.Sanitized()
	if b, ok := a.bodies[l]; ok {
		return b
	}
	b := &bodyArt{head: genHeadMark(l)}
	for d := sim.DirDown; d <= sim.DirRight; d++ {
		for f := range walkFrames {
			b.back[d][f], b.legs[d][f], b.shirt[d][f], b.top[d][f] = genPlayer(l, d, f)
		}
	}
	if a.bodies == nil {
		a.bodies = make(map[sim.Look]*bodyArt)
	}
	a.bodies[l] = b
	return b
}

// drawBody draws a standing or walking player at (x, y), the feet, scaled
// by s. The body bobs up by bob art pixels on step frames.
func drawBody(dst *ebiten.Image, b *bodyArt, dir sim.Dir, frame int, x, y, s, bob float64, tint color.Color) {
	drawAt(dst, b.back[dir][frame], x, y-bob*s, s, s, 0, nil)
	drawAt(dst, b.legs[dir][frame], x, y, s, s, 0, nil)
	drawAt(dst, b.shirt[dir][frame], x, y-bob*s, s, s, 0, tint)
	drawAt(dst, b.top[dir][frame], x, y-bob*s, s, s, 0, nil)
}

// padTop returns c with n transparent rows added above it.
func padTop(c *canvas, n int) *canvas {
	p := newCanvas(c.w, c.h+n)
	copy(p.px[n*c.w:], c.px)
	return p
}

// genPlayer draws one direction and walk frame of look l as four layers.
// Walk frames are 0 stand, 1 left step, 2 stand, 3 right step.
func genPlayer(l sim.Look, d sim.Dir, frame int) (back, legs, shirt, top sprite) {
	l = l.Sanitized()
	sk := skinTones[l.Skin].ramp
	if d == sim.DirLeft {
		b, lg, s, t := genPlayer(l, sim.DirRight, frame)
		return mirrorSprite(b), mirrorSprite(lg), mirrorSprite(s), mirrorSprite(t)
	}
	const w, h = 18, 26
	const ax, ay = 9, 24
	L, S, T := newCanvas(w, h), newCanvas(w, h), newCanvas(w, h)
	step := 0 // -1 or +1 on the step frames
	switch frame {
	case 1:
		step = -1
	case 3:
		step = 1
	}

	// A man wears a shirt and trousers; a woman a dress (shirt and skirt in
	// the player's colour) over bare legs.
	legRamp := pantsRamp
	if l.Female {
		legRamp = sk
	}
	leg := func(x0, top, bottom int, lit float64) {
		for y := top; y <= bottom; y++ {
			for x := x0; x < x0+3; x++ {
				t := lit - float64(x-x0)*0.25
				if y >= bottom-1 {
					L.set(x, y, shoeRamp.pick(t, x, y))
				} else {
					L.set(x, y, legRamp.pick(t, x, y))
				}
			}
		}
	}
	// skirt hangs from the dress's waist, x0..x1, and flares one pixel each
	// side at the hem.
	skirt := func(x0, x1 int) {
		for y := 19; y <= 20; y++ {
			grow := y - 19
			for x := x0 - grow; x <= x1+grow; x++ {
				t := 0.9 - float64(x-x0+grow)/float64(x1-x0+2*grow)*0.6 - float64(grow)*0.15
				S.set(x, y, shirtRamp.pick(t, x, y))
			}
		}
	}
	// waist reports whether a torso pixel is cut away for a woman's waist.
	waist := func(x, y, left, right int) bool {
		return l.Female && (y == 16 || y == 17) && (x == left || x == right)
	}

	switch d {
	case sim.DirDown, sim.DirUp:
		lb, rb := 23, 23
		if step < 0 {
			lb = 22
		} else if step > 0 {
			rb = 22
		}
		leg(6, 18, lb, 0.8)
		leg(9, 18, rb, 0.55)
		if l.Female {
			skirt(5, 12)
		}
		// Torso with rounded shoulders.
		for y := 12; y <= 18; y++ {
			for x := 5; x <= 12; x++ {
				if y == 12 && (x == 5 || x == 12) || waist(x, y, 5, 12) {
					continue
				}
				S.set(x, y, shirtRamp.pick(0.95-float64(x-5)/7*0.6-float64(y-12)*0.03, x, y))
			}
		}
		// Arms swing opposite to the legs.
		la, ra := -step, step
		for y := 13; y <= 17; y++ {
			S.set(4, y+la, shirtRamp.pick(0.55, 4, y))
			S.set(13, y+ra, shirtRamp.pick(0.3, 13, y))
		}
		T.set(4, 18+la, sk[2])
		T.set(13, 18+ra, sk[1])
		if !l.Female { // a woman's shirt runs straight into the skirt
			T.rect(5, 19, 8, 1, func(x, y int) color.RGBA { return beltRamp[(x/2)%2] })
			if d == sim.DirDown {
				T.set(8, 19, hex(0xc9a44a))
			}
		}
	case sim.DirRight:
		back, frontX := 7, 8
		if step != 0 {
			back, frontX = 7-2*step, 8+2*step
		}
		leg(back, 18, 23, 0.35)
		leg(frontX, 18, 23, 0.8)
		if l.Female {
			skirt(6, 11)
		}
		for y := 12; y <= 18; y++ {
			for x := 6; x <= 11; x++ {
				if y == 12 && (x == 6 || x == 11) || waist(x, y, 6, 11) {
					continue
				}
				S.set(x, y, shirtRamp.pick(0.9-float64(x-6)/5*0.5, x, y))
			}
		}
		// One arm on the near side, swinging.
		for y := 13; y <= 17; y++ {
			off := 0
			if y >= 16 {
				off = -step
			}
			S.set(8+off, y, shirtRamp.pick(0.35, 8, y))
			S.set(9+off, y, shirtRamp.pick(0.2, 9, y))
		}
		T.set(8-step, 18, sk[2])
		T.set(9-step, 18, sk[1])
		if !l.Female {
			T.rect(6, 19, 6, 1, func(x, y int) color.RGBA { return beltRamp[(x/2)%2] })
		}
	}

	B := newCanvas(w, h+headroom)
	L, S, T = padTop(L, headroom), padTop(S, headroom), padTop(T, headroom)
	drawHead(T, B, l, d, 9, 7+headroom)
	for _, c := range []*canvas{B, L, S, T} {
		c.outline(0.7)
	}
	return mk(B, ax, ay+headroom), mk(L, ax, ay+headroom), mk(S, ax, ay+headroom), mk(T, ax, ay+headroom)
}

// genHeadMark draws l's head facing the viewer, for markers on the maps.
func genHeadMark(l sim.Look) *ebiten.Image {
	const w, h = 18, 22
	front, back := newCanvas(w, h), newCanvas(w, h)
	drawHead(front, back, l, sim.DirDown, 9, 11)
	back.over(front, 0, 0)
	back.outline(0.8)
	return back.image()
}

// drawHead draws a big round head (chibi proportions) centred on (cx, cy)
// in c, with l's face and hair. Hair hanging behind the body goes in behind.
// d is never DirLeft: that is the mirror image of DirRight.
func drawHead(c, behind *canvas, l sim.Look, d sim.Dir, cx, cy int) {
	const rx, ry = 6.0, 5.8
	l = l.Sanitized()
	hr, sk := hairColors[l.HairColor].ramp, skinTones[l.Skin].ramp
	fx, fy := float64(cx), float64(cy)
	c.ellipse(fx, fy, rx, ry, func(x, y int, nx, ny float64) color.RGBA {
		l2 := lum(nx, ny)
		if hairCap(l.Hair, d, x-cx+9, nx, ny) {
			return hr.pick(l2+0.1, x, y)
		}
		return sk.pick(l2+0.25, x, y)
	})
	// isHair reports whether the head pixel at (x, y) is hair, so the face
	// is not drawn over it.
	isHair := func(x, y int) bool {
		nx, ny := (float64(x)+0.5-fx)/rx, (float64(y)+0.5-fy)/ry
		return hairCap(l.Hair, d, x-cx+9, nx, ny)
	}
	face := func(x, y int, col color.RGBA) {
		if !isHair(x, y) {
			c.set(x, y, col)
		}
	}
	eye, blush := hex(0x1b1424), skinTones[l.Skin].blush
	switch d {
	case sim.DirDown:
		for _, ex := range []int{cx - 3, cx + 2} {
			face(ex, cy+1, eye)
			face(ex, cy+2, eye)
		}
		if l.Female { // lashes
			face(cx-4, cy, eye)
			face(cx+3, cy, eye)
		}
		face(cx-4, cy+3, blush)
		face(cx+3, cy+3, blush)
	case sim.DirRight:
		face(cx+3, cy+1, eye)
		face(cx+3, cy+2, eye)
		if l.Female {
			face(cx+4, cy, eye)
		}
		face(cx+5, cy+3, blush)
		face(cx-1, cy+1, sk[1]) // ear
	case sim.DirUp:
		c.set(cx, cy-4, hr[3])
		c.set(cx-1, cy-3, hr[3])
	}
	hairExtras(c, behind, l.Hair, d, fx, fy, hr)
}

// hairCap reports whether the head pixel with normalised offset (nx, ny) is
// hair. x is the pixel's column with the head centred on column 9, for
// patterns such as a fringe.
func hairCap(s sim.HairStyle, d sim.Dir, x int, nx, ny float64) bool {
	if d == sim.DirUp {
		return true
	}
	side := math.Abs(nx)
	switch s {
	case sim.HairSpiky:
		if d == sim.DirDown {
			return ny < -0.3 || (ny < 0.0 && x%3 == 0) || side > 0.85
		}
		return ny < -0.35 || nx < -0.2 || (ny < -0.05 && nx < 0.4 && x%2 == 0)
	case sim.HairBob:
		if d == sim.DirDown {
			return ny < 0.0 || side > 0.68
		}
		return ny < -0.1 || nx < -0.05 || (ny < 0.2 && nx < 0.25)
	case sim.HairLong:
		if d == sim.DirDown { // parted in the middle
			return ny < -0.3 || side > 0.65 || (ny < 0.05 && side > 0.25)
		}
		return ny < -0.3 || nx < 0.05 || (ny < 0.0 && nx < 0.4)
	case sim.HairPonytail, sim.HairBun: // pulled back off the face
		if d == sim.DirDown {
			return ny < -0.35 || side > 0.85 || (ny < -0.1 && side > 0.5)
		}
		return ny < -0.35 || nx < -0.3 || (ny < -0.1 && nx < 0.2)
	}
	// HairShort, the original cut.
	switch d {
	case sim.DirDown:
		return ny < -0.25 || (ny < 0.05 && (x/2)%2 == 0) || side > 0.82
	default:
		return ny < -0.3 || nx < -0.15 || (ny < 0.0 && nx < 0.3)
	}
}

// hairExtras draws the hair outside the head's outline: spikes, a bun, locks
// and tails. What hangs behind the body goes in behind.
func hairExtras(c, behind *canvas, s sim.HairStyle, d sim.Dir, cx, cy float64, hr ramp) {
	shade := func(t float64) color.RGBA { return hr.at(int(math.Round(clamp01(t) * float64(len(hr)-1)))) }
	// strands fills a hanging panel of hair: lighter at the top and on the
	// lit side, with darker strand lines, rounded off at the bottom.
	strands := func(dst *canvas, x0, y0, x1, y1 int) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if y == y1 && (x == x0 || x == x1) && x1 > x0 {
					continue
				}
				t := 0.7 - float64(y-y0)/float64(max(1, y1-y0))*0.3 - (float64(x)+0.5-cx)/16
				if x%3 == 1 && y > y0+1 {
					t -= 0.22
				}
				dst.set(x, y, shade(t))
			}
		}
	}
	// tail fills a lock of hair along the points pts, its radius going from
	// r0 at the first point to r1 at the last.
	tail := func(dst *canvas, pts [][2]float64, r0, r1 float64) {
		type sample struct{ x, y, r, t float64 }
		var ss []sample
		const per = 12
		for i := 0; i+1 < len(pts); i++ {
			for k := range per {
				f := float64(k) / per
				t := (float64(i) + f) / float64(len(pts)-1)
				ss = append(ss, sample{
					x: lerpf(pts[i][0], pts[i+1][0], f), y: lerpf(pts[i][1], pts[i+1][1], f),
					r: lerpf(r0, r1, t), t: t,
				})
			}
		}
		last := pts[len(pts)-1]
		ss = append(ss, sample{last[0], last[1], r1, 1})
		for y := int(cy) - 12; y <= int(cy)+14; y++ {
			for x := int(cx) - 10; x <= int(cx)+10; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				best, bs := math.Inf(1), sample{}
				for _, s := range ss {
					if d := math.Hypot(px-s.x, py-s.y) / s.r; d < best {
						best, bs = d, s
					}
				}
				if best > 1 {
					continue
				}
				dst.set(x, y, shade(0.78-0.3*bs.t-0.45*(px-bs.x)/bs.r))
			}
		}
	}
	// blob is a round lump of hair, lit from the top-left.
	blob := func(dst *canvas, x, y, r float64) {
		dst.ellipse(x, y, r, r, func(px, py int, nx, ny float64) color.RGBA {
			return hr.pick(lum(nx, ny)+0.1, px, py)
		})
	}
	ix, iy := int(cx), int(cy)
	switch s {
	case sim.HairSpiky:
		tips := [][2]float64{{-5.5, -6.5}, {-2.8, -8.5}, {0, -9.2}, {2.8, -8.5}, {5.5, -6.5}}
		lean := 0.0
		if d == sim.DirRight {
			lean = 1.6 // swept back
			tips = [][2]float64{{-7, -4}, {-5, -7.5}, {-2, -9}, {1.5, -8.5}, {4, -7}}
		}
		for _, tp := range tips {
			x, y := cx+tp[0], cy+tp[1]
			c.polygon([]float64{x + lean - 1.9, cy - 3.6, x, y, x + lean + 1.9, cy - 3.6}, func(px, py int) color.RGBA {
				return hr.pick(0.85-(float64(px)+0.5-cx)/12-(float64(py)-y)*0.04, px, py)
			})
		}
	case sim.HairBob:
		switch d {
		case sim.DirDown:
			strands(c, ix-7, iy-2, ix-5, iy+5)
			strands(c, ix+4, iy-2, ix+6, iy+5)
		case sim.DirUp:
			strands(c, ix-6, iy+2, ix+5, iy+5)
		case sim.DirRight:
			strands(c, ix-7, iy, ix-3, iy+4)
		}
	case sim.HairLong:
		switch d {
		case sim.DirDown:
			strands(c, ix-7, iy-1, ix-5, iy+8)
			strands(c, ix+4, iy-1, ix+6, iy+8)
			strands(behind, ix-6, iy+3, ix+5, iy+8)
		case sim.DirUp:
			strands(c, ix-6, iy+2, ix+5, iy+8)
		case sim.DirRight:
			strands(c, ix-7, iy, ix-3, iy+8)
		}
	case sim.HairPonytail:
		switch d {
		case sim.DirDown: // the tail swings out from behind the head
			tail(behind, [][2]float64{{cx + 3.5, cy - 3.5}, {cx + 6.2, cy - 1}, {cx + 7, cy + 2.5}, {cx + 6.2, cy + 6}}, 2.0, 0.8)
		case sim.DirUp:
			tail(c, [][2]float64{{cx, cy + 1}, {cx + 0.2, cy + 4}, {cx + 0.8, cy + 7}, {cx + 1.4, cy + 9.5}}, 1.9, 0.7)
			c.set(ix-1, iy+2, hairTie)
			c.set(ix, iy+2, hairTie)
		case sim.DirRight:
			tail(c, [][2]float64{{cx - 4.5, cy - 3}, {cx - 6.8, cy - 0.5}, {cx - 7.4, cy + 3}, {cx - 6.8, cy + 6.5}}, 1.9, 0.7)
			c.set(ix-6, iy-2, hairTie)
			c.set(ix-5, iy-3, hairTie)
		}
	case sim.HairBun:
		if d == sim.DirRight {
			blob(c, cx-1.5, cy-6, 2.6)
		} else {
			blob(c, cx, cy-6.4, 2.6)
		}
	}
}
