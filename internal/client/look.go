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

// torsoSpan is the columns the torso covers on row y of the 18x26 frame
// (before the headroom) facing d, the shape the tunic and the armour follow;
// ok is false off the torso. A man is broad across the chest and shoulders
// and narrows to the waist; a woman is slimmer, with a narrow waist. Row 19
// is a man's tunic hem. d is never DirLeft: that is DirRight mirrored.
func torsoSpan(female bool, d sim.Dir, y int) (x0, x1 int, ok bool) {
	side := d == sim.DirRight || d == sim.DirLeft
	switch {
	// Everyone is round and soft: the torso swells from the neck to a
	// round belly and tucks in again at the hem.
	case !female && !side:
		switch {
		case y == 12, y == 19:
			return 6, 11, true
		case y == 13, y == 18:
			return 5, 12, true
		case y >= 14 && y <= 17:
			return 4, 13, true
		}
	case !female:
		switch {
		case y == 12, y == 19:
			return 7, 10, true
		case y == 13, y == 18:
			return 6, 11, true
		case y >= 14 && y <= 17:
			return 5, 12, true
		}
	case !side:
		switch {
		case y == 12:
			return 6, 11, true
		case y == 13, y == 17:
			return 5, 12, true
		case y >= 14 && y <= 16, y == 18:
			return 4, 13, true
		}
	default:
		switch {
		case y == 12:
			return 7, 10, true
		case y == 13, y == 17, y == 18:
			return 6, 11, true
		case y >= 14 && y <= 16:
			return 5, 12, true
		}
	}
	return 0, 0, false
}

// genPlayer draws one direction and walk frame of look l as four layers.
// Walk frames are 0 stand, 1 left step, 2 stand, 3 right step.
//
// The figure is a strong survivor: a man has broad, rounded shoulders that
// stand out past his head, thick bare arms and a chest that narrows to the
// waist; a woman is lean and toned. Coordinates are in the 18x26 frame
// before the headroom, feet at (9, 24); armorart.go, held.go and ride.go
// follow the same frame.
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
	male := !l.Female

	// Everyone dresses in hide: a man in a one-shouldered tunic, a woman in a
	// hide dress, both in the player's colour, ragged at the hem, over bare
	// legs with hide wraps round the shins and hide boots.
	leg := func(x0, top, bottom int, lit float64) {
		for y := top; y <= bottom; y++ {
			for x := x0; x < x0+3; x++ {
				t := lit - float64(x-x0)*0.25
				switch {
				case y >= bottom-1:
					L.set(x, y, darkHideRamp.pick(t, x, y)) // boots
				case y >= bottom-3:
					L.set(x, y, hideRamp.pick(t+0.1*float64(y%2), x, y)) // shin wraps
				default:
					L.set(x, y, sk.pick(t, x, y))
				}
			}
		}
	}
	// fur gives the tunic the nap of a hide.
	fur := func(x, y int, t float64) color.RGBA {
		return shirtRamp.pick(t+(hashf(x, y/2, 71)-0.5)*0.16, x, y)
	}
	// skirt hangs from the dress's waist, x0..x1, and flares one pixel each
	// side at the hem.
	skirt := func(x0, x1 int) {
		for y := 19; y <= 20; y++ {
			grow := y - 19
			for x := x0 - grow; x <= x1+grow; x++ {
				if y == 20 && (x+1)%3 == 0 {
					continue // a ragged hem
				}
				t := 0.9 - float64(x-x0+grow)/float64(x1-x0+2*grow)*0.6 - float64(grow)*0.15
				S.set(x, y, fur(x, y, t))
			}
		}
	}
	// limb paints bare skin from shade indexes into the skin ramp, rows top
	// to bottom, starting at column x0. -1 leaves a pixel empty.
	limb := func(c *canvas, x0, y0 int, rows ...[]int) {
		for dy, row := range rows {
			for dx, i := range row {
				if i >= 0 {
					c.set(x0+dx, y0+dy, sk.at(i))
				}
			}
		}
	}
	// torso fills the tunic or dress down to row last, leaving bare the
	// pixels bare() picks: those go on the legs layer, under the shirt.
	torso := func(last int, bare func(x, y int) bool, shade func(x, y int) float64) {
		for y := 12; y <= last; y++ {
			x0, x1, ok := torsoSpan(l.Female, d, y)
			for x := x0; ok && x <= x1; x++ {
				switch {
				case y == 19 && x%2 == 0:
					continue // a ragged hem
				case bare(x, y):
					// Bare skin, darker along the tunic's edge: the
					// underside of the chest muscle.
					t := 0.85
					if !bare(x, y+1) {
						t -= 0.35
					}
					L.set(x, y, sk.pick(t, x, y))
				default:
					t := shade(x, y)
					if y == 17 && male {
						t -= 0.3 // a sash of the same hide
					}
					S.set(x, y, fur(x, y, t))
				}
			}
		}
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
		// The tunic goes over one shoulder only. Facing the viewer the bare
		// one is on the left of the screen; from behind, on the right. A
		// man's is cut diagonally from it, so the muscle of the chest (or
		// back) shows, and comes down past the hips to a ragged hem.
		edge, dir := 4, 1 // the bare side's outer column, and inwards
		if l.Female {
			edge = 5
		}
		if d == sim.DirUp {
			edge, dir = 17-edge, -1
		}
		bare := func(x, y int) bool {
			in := (x - edge) * dir // columns in from the bare side's edge
			if male {
				return in >= 0 && in+(y-12) <= 3
			}
			return in >= 0 && (y == 12 && in <= 1 || y == 13 && in == 0)
		}
		last := 18
		if male {
			last = 19
		}
		torso(last, bare, func(x, y int) float64 {
			t := 0.95 - float64(x-4)/9*0.6 - float64(y-12)*0.03
			if male && y == 15 && x >= 8 {
				t -= 0.12 // the chest's lower edge under the tunic
			}
			return t
		})
		// Bare arms swing opposite to the legs: the shoulder stays, the
		// forearm and fist move. A man's are two pixels thick: a round
		// shoulder standing out past his head, a bulge of biceps with a
		// darker line above it, and the fists in by his hips. Shade indexes
		// into the skin ramp, lit from the top left.
		la, ra := -step, step
		if male {
			limb(T, 2, 12, []int{-1, 3}, []int{3, 3}, []int{2, 1}, []int{3, 2})
			limb(T, 14, 12, []int{2, -1}, []int{2, 1}, []int{1, 0}, []int{2, 1})
			for y := 16 + min(la, 0); y <= 17+la; y++ {
				limb(T, 2, y, []int{2, 1})
			}
			for y := 16 + min(ra, 0); y <= 17+ra; y++ {
				limb(T, 14, y, []int{1, 0})
			}
			limb(T, 3, 18+la, []int{3, 2}) // fists
			limb(T, 13, 18+ra, []int{2, 1})
		} else {
			// A woman's arms are lean: a toned upper arm, a slim forearm.
			limb(T, 3, 12, []int{-1, 3}, []int{3, 2}, []int{3, 2})
			limb(T, 13, 12, []int{2, -1}, []int{2, 1}, []int{2, 1})
			for y := 15 + min(la, 0); y <= 17+la; y++ {
				limb(T, 4, y, []int{2})
			}
			for y := 15 + min(ra, 0); y <= 17+ra; y++ {
				limb(T, 13, y, []int{1})
			}
			T.set(4, 18+la, sk[2])
			T.set(13, 18+ra, sk[1])
		}
	case sim.DirRight:
		back, frontX := 7, 8
		if step != 0 {
			back, frontX = 7-2*step, 8+2*step
		}
		leg(back, 18, 23, 0.35)
		leg(frontX, 18, 23, 0.8)
		last := 18
		if l.Female {
			skirt(6, 11)
		} else {
			last = 19
		}
		torso(last, func(int, int) bool { return false }, func(x, y int) float64 {
			return 0.95 - float64(x-5)/7*0.55
		})
		// The near arm is the bare one, swinging from a fixed shoulder.
		for y := 12; y <= 18; y++ {
			for x := 5; x <= 12; x++ {
				if i := sideArm(l.Female, step, x, y); i >= 0 {
					T.set(x, y, sk.at(i))
				}
			}
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

// headRX and headRY are the head's radii in art pixels: a big head, but not
// a chibi's.
const headRX, headRY = 6.0, 5.9

// drawHead draws a round head centred on (cx, cy) in c, with l's face and
// hair. Hair hanging behind the body goes in behind.
// d is never DirLeft: that is the mirror image of DirRight.
func drawHead(c, behind *canvas, l sim.Look, d sim.Dir, cx, cy int) {
	const rx, ry = headRX, headRY
	l = l.Sanitized()
	hr, sk := hairColors[l.HairColor].ramp, skinTones[l.Skin].ramp
	fx, fy := float64(cx), float64(cy)
	// A round head, round as a ball, man or woman.
	inHead := func(nx, ny float64) bool { return nx*nx+ny*ny <= 1 }
	for y := cy - 7; y <= cy+7; y++ { // a box round the head (headRX, headRY < 7)
		for x := cx - 7; x <= cx+7; x++ {
			nx, ny := (float64(x)+0.5-fx)/rx, (float64(y)+0.5-fy)/ry
			if !inHead(nx, ny) {
				continue
			}
			l2 := lum(nx, ny)
			if hairCap(l.Hair, d, x-cx+9, nx, ny) {
				c.set(x, y, hr.pick(l2+0.1, x, y))
			} else {
				c.set(x, y, sk.pick(l2+0.25, x, y))
			}
		}
	}
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
		} else { // heavy brows
			for _, bx := range []int{cx - 4, cx - 3, cx + 2, cx + 3} {
				face(bx, cy, hr[1])
			}
		}
		face(cx-4, cy+3, blush)
		face(cx+3, cy+3, blush)
	case sim.DirRight:
		face(cx+3, cy+1, eye)
		face(cx+3, cy+2, eye)
		if l.Female {
			face(cx+4, cy, eye)
		} else {
			face(cx+3, cy, hr[1])
			face(cx+4, cy, hr[1])
		}
		face(cx+2, cy+3, blush) // the cheek, below and behind the eye
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
	// The shapes below were laid out for a head six pixels in radius; k
	// scales them to the head's size. at, col and row place an offset from
	// the head's centre.
	k := headRX / 6
	at := func(dx, dy float64) [2]float64 { return [2]float64{cx + dx*k, cy + dy*k} }
	col := func(o float64) int { return int(cx) + int(math.Round(o*k)) }
	row := func(o float64) int { return int(cy) + int(math.Round(o*k)) }
	switch s {
	case sim.HairSpiky:
		tips := [][2]float64{{-5.5, -6.5}, {-2.8, -8.5}, {0, -9.2}, {2.8, -8.5}, {5.5, -6.5}}
		lean := 0.0
		if d == sim.DirRight {
			lean = 1.6 // swept back
			tips = [][2]float64{{-7, -4}, {-5, -7.5}, {-2, -9}, {1.5, -8.5}, {4, -7}}
		}
		base, half := cy-3.6*k, 1.9*k
		for _, tp := range tips {
			p := at(tp[0], tp[1])
			x, y, sx := p[0], p[1], p[0]+lean*k
			c.polygon([]float64{sx - half, base, x, y, sx + half, base}, func(px, py int) color.RGBA {
				return hr.pick(0.85-(float64(px)+0.5-cx)/12-(float64(py)-y)*0.04, px, py)
			})
		}
	case sim.HairBob:
		switch d {
		case sim.DirDown:
			strands(c, col(-7), row(-2), col(-5), row(5))
			strands(c, col(4), row(-2), col(6), row(5))
		case sim.DirUp:
			strands(c, col(-6), row(2), col(5), row(5))
		case sim.DirRight:
			strands(c, col(-7), row(0), col(-3), row(4))
		}
	case sim.HairLong:
		switch d {
		case sim.DirDown:
			strands(c, col(-7), row(-1), col(-5), row(8))
			strands(c, col(4), row(-1), col(6), row(8))
			strands(behind, col(-6), row(3), col(5), row(8))
		case sim.DirUp:
			strands(c, col(-6), row(2), col(5), row(8))
		case sim.DirRight:
			strands(c, col(-7), row(0), col(-3), row(8))
		}
	case sim.HairPonytail:
		switch d {
		case sim.DirDown: // the tail swings out from behind the head
			tail(behind, [][2]float64{at(3.5, -3.5), at(6.2, -1), at(7, 2.5), at(6.2, 6)}, 2.0*k, 0.8*k)
		case sim.DirUp:
			tail(c, [][2]float64{at(0, 1), at(0.2, 4), at(0.8, 7), at(1.4, 9.5)}, 1.9*k, 0.7*k)
			c.set(col(-1), row(2), hairTie)
			c.set(col(0), row(2), hairTie)
		case sim.DirRight:
			tail(c, [][2]float64{at(-4.5, -3), at(-6.8, -0.5), at(-7.4, 3), at(-6.8, 6.5)}, 1.9*k, 0.7*k)
			c.set(col(-6), row(-2), hairTie)
			c.set(col(-5), row(-3), hairTie)
		}
	case sim.HairBun:
		p := at(0, -6.4)
		if d == sim.DirRight {
			p = at(-1.5, -6)
		}
		blob(c, p[0], p[1], 2.6*k)
	}
}

// sideArm is the shade index into the skin ramp of the near arm's pixel
// (x, y) for a figure facing right on walk step step, or -1 off the arm.
// The shoulder and upper arm stay; the forearm and fist swing. A man's arm
// has a round shoulder and a bulging upper arm. armorart.go leaves the arm
// out of the armour with it.
func sideArm(female bool, step, x, y int) int {
	if y >= 16 && y <= 18 { // the forearm and the fist
		switch x {
		case 8 - step:
			return 2
		case 9 - step:
			return 1
		}
		return -1
	}
	upper, x0 := [4][]int{{-1, 3, 2, -1}, {3, 3, 2, 1}, {-1, 2, 1, 1}, {-1, 2, 3, 1}}, 7
	if female {
		upper, x0 = [4][]int{{3, 2}, {3, 2}, {2, 1}, {3, 1}}, 8
	}
	if y < 12 || y > 15 || x < x0 || x >= x0+len(upper[y-12]) {
		return -1
	}
	return upper[y-12][x-x0]
}
