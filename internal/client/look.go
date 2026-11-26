package client

import (
	"image/color"

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
	{"Nâu đen", mkRamp(0x23130d, 0x3d2416, 0x5c3920, 0x8a5c30)},
	{"Đen", mkRamp(0x0d0b14, 0x1a1724, 0x2b2738, 0x4b4a66)},
	{"Nâu hạt dẻ", mkRamp(0x3a190e, 0x5e2c16, 0x8a4520, 0xb96c32)},
	{"Vàng", mkRamp(0x6a4720, 0xa37730, 0xd8ad4c, 0xf8e294)},
	{"Đỏ", mkRamp(0x521510, 0x8a2814, 0xc2481e, 0xf0823e)},
	{"Bạch kim", mkRamp(0x565a6c, 0x858a9c, 0xbcc0cc, 0xf2f4f8)},
	{"Xanh dương", mkRamp(0x151f4a, 0x24367a, 0x3858ae, 0x6a94e4)},
	{"Hồng", mkRamp(0x62203d, 0x98325e, 0xcf578a, 0xf89cc2)},
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
var hairNames = [sim.NumHairStyles]string{"Ngắn", "Mào dựng", "Bob tết", "Dài", "Bím dài", "Búi cao", "Bờm xù"}

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
	asleep                 sprite                // top facing the viewer, eyes shut
	head                   *ebiten.Image         // facing the viewer: the map marker

	look  sim.Look
	wear  sim.Wear              // the clothes on the jointed figure; see cavefig.go
	bare  [numArmSides]*bareArt // without one arm, for a punch; see punch.go
	fists *fistArt              // the arms thrown out in a punch
	limbs limbArt               // the jointed figure's frames; see limbdraw.go
}

// bodyKey names the sprites of a look in its clothes.
type bodyKey struct {
	look sim.Look
	wear sim.Wear
}

// body returns the sprites of look l in nothing but its furs.
func (a *art) body(l sim.Look) *bodyArt { return a.bodyIn(l, 0) }

// bodyIn returns the sprites of look l wearing w, drawing them on first
// use. It must be called inside the game loop, where images can be created.
func (a *art) bodyIn(l sim.Look, w sim.Wear) *bodyArt {
	l = l.Sanitized()
	if !(playerHD && playerJointed) {
		w = 0 // only the jointed figure changes its clothes
	}
	key := bodyKey{l, w}
	if b, ok := a.bodies[key]; ok {
		return b
	}
	b := &bodyArt{head: genHeadMark(l), look: l, wear: w}
	if a.bodies == nil {
		a.bodies = make(map[bodyKey]*bodyArt)
	}
	if playerHD && playerJointed { // its frames are drawn as they are shown: see limbdraw.go
		a.bodies[key] = b
		return b
	}
	if playerHD { // see lookhd.go
		_, _, _, shut := playerLayersHD(l, sim.DirDown, 0, true, armNone)
		b.asleep = figSprite(shut)
	} else {
		_, _, _, shut := playerLayersWith(l, sim.DirDown, 0, true)
		b.asleep = mk(shut, 9, bodyFeet)
	}
	for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
		for f := range walkFrames {
			if playerHD {
				b.back[d][f], b.legs[d][f], b.shirt[d][f], b.top[d][f] = genPlayerHD(l, d, f)
			} else {
				b.back[d][f], b.legs[d][f], b.shirt[d][f], b.top[d][f] = genPlayer(l, d, legacyFrame(f))
			}
		}
	}
	for f := range walkFrames { // the mirror image
		b.back[sim.DirLeft][f], b.legs[sim.DirLeft][f] = mirrorSprite(b.back[sim.DirRight][f]), mirrorSprite(b.legs[sim.DirRight][f])
		b.shirt[sim.DirLeft][f], b.top[sim.DirLeft][f] = mirrorSprite(b.shirt[sim.DirRight][f]), mirrorSprite(b.top[sim.DirRight][f])
	}
	a.bodies[key] = b
	return b
}

// drawBody draws a standing or walking player at (x, y), the feet, scaled
// by s. The body bobs up by bob art pixels on step frames.
func drawBody(dst drawTarget, b *bodyArt, dir sim.Dir, frame int, x, y, s, bob float64, tint color.Color) {
	if playerHD && playerJointed {
		drawBodyJ(dst, b, dir, frame, x, y, s, tint)
		return
	}
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

// The figure is a sturdy little Norse settler, two heads tall: a big round
// head, a fur mantle on the shoulders, a belted tunic in the player's colour
// over trousers, leg wraps and fur boots for a man, who wears a braided
// beard; an apron dress pinned with two brooches over a lighter shift for a
// woman. Everything below is laid out in the sprite's own pixels: 18 wide,
// 26 tall plus the headroom, feet standing on row bodyFeet, the head
// centred on (9, 11).
const (
	bodyW    = 18
	bodyH    = 26 + headroom
	bodyFeet = 24 + headroom // the anchor's row: the soles are just above it
)

// headSpan is the columns the bare head covers on sprite row y; ok is false
// above and below it.
func headSpan(y int) (x0, x1 int, ok bool) {
	switch {
	case y == 5, y == 15:
		return 5, 12, true
	case y == 6, y == 14:
		return 4, 13, true
	case y >= 7 && y <= 13:
		return 3, 14, true
	}
	return 0, 0, false
}

// torsoRow is the columns the tunic or dress covers on sprite row y facing
// d (never DirLeft), without the arms; ok is false off it.
func torsoRow(female bool, d sim.Dir, y int) (x0, x1 int, ok bool) {
	side := d == sim.DirRight || d == sim.DirLeft
	switch {
	case !female && !side:
		if y >= 16 && y <= 21 {
			return 5, 12, true
		}
	case !female:
		if y >= 16 && y <= 21 {
			return 6, 11, true
		}
	case !side:
		switch {
		case y >= 16 && y <= 20:
			return 6, 11, true
		case y >= 21 && y <= 23:
			return 5, 12, true
		case y >= 24 && y <= 25:
			return 4, 13, true
		}
	default:
		switch {
		case y >= 16 && y <= 20:
			return 7, 11, true
		case y == 21:
			return 6, 11, true
		case y >= 22 && y <= 23:
			return 6, 12, true
		case y >= 24 && y <= 25:
			return 5, 12, true
		}
	}
	return 0, 0, false
}

// torsoSpan is torsoRow for row y of the 18x26 frame before the headroom,
// the shape the armour follows (armorart.go).
func torsoSpan(female bool, d sim.Dir, y int) (x0, x1 int, ok bool) {
	return torsoRow(female, d, y+headroom)
}

// sideArmX is the left column of the two-pixel-wide near arm on sprite row y
// of a figure facing right on walk step step; ok is false off the arm. The
// shoulder stays, the forearm and the hand swing.
func sideArmX(step, y int) (x int, ok bool) {
	switch {
	case y >= 16 && y <= 18:
		return 8, true
	case y >= 19 && y <= 20:
		return 8 - step, true
	case y >= 21 && y <= 22:
		return 8 - 2*step, true
	}
	return 0, false
}

// sideArm reports whether (x, y), in the 18x26 frame before the headroom, is
// on the near arm or hand of a figure facing right on walk step step.
// armorart.go leaves the arm out of the armour with it.
func sideArm(step, x, y int) bool {
	x0, ok := sideArmX(step, y+headroom)
	return ok && (x == x0 || x == x0+1)
}

var (
	trouserRamp = mkRamp(0x2c2019, 0x443227, 0x5c4636, 0x765c47)
	wrapRamp    = mkRamp(0x7d6a4c, 0xa8936c, 0xcdb98f, 0xe6d8b4)
	mantleRamp  = mkRamp(0x2f2a2c, 0x4d4548, 0x736a69, 0x9c938d, 0xc6beb4)
	leatherRamp = mkRamp(0x33200f, 0x543619, 0x774f26, 0x9a6a36)
	broochRamp  = mkRamp(0x8a5a14, 0xd09a26, 0xf6cf52, 0xfff2b0)
	eyeDark     = hex(0x1d1626)
	eyeShine    = hex(0xffffff)
	// irisColors are the eyes that go with each skin tone, light enough to
	// show beside the pupil.
	irisColors = [sim.NumSkinTones]color.RGBA{hex(0x4f8fd6), hex(0x4fa07a), hex(0x9a6a34), hex(0xb07c3c), hex(0xc08a44), hex(0xc8924a)}
	mouthDark  = hex(0x6e2a2a)
	beadColors = []color.RGBA{hex(0xe8a83a), hex(0xd24a3a), hex(0x4fb4c8), hex(0xe8a83a)}
)

// artPalette maps the letters of the pixel maps below to colours for look l:
//
//	H h d D  hair, light to dark      x  stubble (shaved hair)
//	S s e E  skin, light to dark      r  a hair tie
//	F f u U  fur, light to dark       y o O  gold, light to dark
//	m        the mouth
func artPalette(l sim.Look) func(ch byte) (color.RGBA, bool) {
	hr, sk := hairColors[l.HairColor].ramp, skinTones[l.Skin].ramp
	stubble := mix(sk[1], hr[1], 0.5)
	return func(ch byte) (color.RGBA, bool) {
		switch ch {
		case 'H':
			return hr[3], true
		case 'h':
			return hr[2], true
		case 'd':
			return hr[1], true
		case 'D':
			return hr[0], true
		case 'x':
			return stubble, true
		case 'S':
			return sk[3], true
		case 's':
			return sk[2], true
		case 'e':
			return sk[1], true
		case 'E':
			return sk[0], true
		case 'F':
			return mantleRamp[4], true
		case 'f':
			return mantleRamp[3], true
		case 'u':
			return mantleRamp[2], true
		case 'U':
			return mantleRamp[1], true
		case 'y':
			return broochRamp[3], true
		case 'o':
			return broochRamp[2], true
		case 'O':
			return broochRamp[1], true
		case 'r':
			return hairTie, true
		case 'm':
			return mouthDark, true
		}
		return color.RGBA{}, false
	}
}

// beardPalette is artPalette for a man's beard: a step lighter than dark
// hair, a step darker than fair hair, so the beard and the hair above it
// read as two things and the jaw shows between them.
func beardPalette(l sim.Look) func(ch byte) (color.RGBA, bool) {
	hr := hairColors[l.HairColor].ramp
	pal := artPalette(l)
	var br ramp // dark to light, like hr
	if luma(hr[2]) < 50 {
		br = ramp{hr[1], mix(hr[2], hr[3], 0.35), hr[3], mix(hr[3], colWhite, 0.25)}
	} else {
		br = ramp{mix(hr[0], colInk, 0.3), hr[0], hr[1], hr[2]}
	}
	return func(ch byte) (color.RGBA, bool) {
		switch ch {
		case 'H':
			return br[3], true
		case 'h':
			return br[2], true
		case 'd':
			return br[1], true
		case 'D':
			return br[0], true
		}
		return pal(ch)
	}
}

// luma is how bright c looks, 0 to 255.
func luma(c color.RGBA) float64 {
	return 0.3*float64(c.R) + 0.59*float64(c.G) + 0.11*float64(c.B)
}

// pixMap is a block of pixel art: one string per row, 18 letters wide, the
// first on sprite row y0. '.' is empty.
type pixMap struct {
	y0   int
	rows []string
}

func (m pixMap) stamp(c *canvas, pal func(byte) (color.RGBA, bool)) {
	for dy, row := range m.rows {
		for x := 0; x < len(row); x++ {
			if col, ok := pal(row[x]); ok {
				c.put(x, m.y0+dy, col)
			}
		}
	}
}

// has reports whether the map covers (x, y).
func (m pixMap) has(x, y int) bool {
	dy := y - m.y0
	return dy >= 0 && dy < len(m.rows) && x >= 0 && x < len(m.rows[dy]) && m.rows[dy][x] != '.'
}

// hairArt is one hair style seen from the front, the back and the right
// side. behind is what hangs behind the body when facing the viewer.
type hairArt struct {
	front, back, side, behind pixMap
}

var hairArts = [sim.NumHairStyles]hairArt{
	sim.HairShort: { // swept over the top, the sides cropped
		front: pixMap{3, []string{
			"......hhhhhd......",
			"....hHHHHhhhhd....",
			"...hHHHHhhhhhhd...",
			"...hHhhhhhdhhdd...",
			"...dhhhd.hhhd.D...",
			"...d..........D...",
			"...d..........D...",
		}},
		back: pixMap{3, []string{
			"......hhhhhd......",
			"....hHHHHhhhhd....",
			"...hHHHHhhhhhhd...",
			"...hHHhhhhhhhhd...",
			"...hHhhhhhhhhdd...",
			"...hhhhhhhhhhdd...",
			"...hhhdhhdhhddd...",
			"...dhhdhhdhhdDd...",
			"...ddhdhhdhhdDD...",
			"...ddddddddddDD...",
			"....dDDDDDDDDD....",
		}},
		side: pixMap{3, []string{
			".....hhhhhhhd.....",
			"...hhHHHHHhhhhd...",
			"...hHHHHhhhhhhhd..",
			"...hHHhhhhhhhhhd..",
			"...hhhhhhdhhhdhd..",
			"...dhhhhdhhhd.d...",
			"...dhhhdhh........",
			"...ddhhd..........",
			"...dddd...........",
			"....ddd...........",
			"....dD............",
		}},
	},
	sim.HairSpiky: { // a crest down the middle, the sides shaved
		front: pixMap{0, []string{
			"........Hh........",
			".......hHhd.......",
			".......hHhd.......",
			"......hHHhhd......",
			"....xxhHHhhdxx....",
			"...xxxhHhhhdxxx...",
			"...xxxxhHhdxxxx...",
			"...xx...hd...xx...",
			"...x..........x...",
			"...x..........x...",
		}},
		back: pixMap{0, []string{
			"........Hh........",
			".......hHhd.......",
			".......hHhd.......",
			"......hHHhhd......",
			"....xxhHHhhdxx....",
			"...xxxhHhhhdxxx...",
			"...xxxxhHhdxxxx...",
			"...xxxxhHhdxxxx...",
			"...xxxxhHhdxxxx...",
			"...xxxxhhhdxxxx...",
			"...xxxxdhhdxxxx...",
			"...xxxxxhdxxxxx...",
			"...xxxxxxxxxxxx...",
			"....xxxxxxxxxx....",
		}},
		side: pixMap{0, []string{
			".......H..H..H....",
			"......hHh.Hh.Hh...",
			".....hHHhhHhhHhd..",
			"....hhHHhHHhHHhd..",
			"...hhhhhhhhhhhdd..",
			"...xxddhhhhhhdd...",
			"...xxxxxxxxxxx....",
			"...xxxxxxxxxx.....",
			"...xxxxxxxx.......",
			"...xxxxxxx........",
			"...xxxxxx.........",
			"...xxxx...........",
			"....xxx...........",
			"....xx............",
		}},
	},
	sim.HairBob: { // chin length with a straight fringe, a thin braid at one side
		front: pixMap{3, []string{
			"......hhhhhd......",
			"....hhHHHHhhhd....",
			"..hhHHHHhhhhhhhd..",
			"..hHHhhhhhhhhhhd..",
			"..hHhdhhhdhhhdhd..",
			"..hd..d...d...hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........dd..",
			"..d............D..",
		}},
		back: pixMap{3, []string{
			"......hhhhhd......",
			"....hhHHHHhhhd....",
			"..hhHHHHhhhhhhhd..",
			"..hHHHhhhhhhhhhd..",
			"..hHHhhhhhhhhhhd..",
			"..hHhhhhhhhhhhdd..",
			"..hhhhhdhhhdhhdd..",
			"..hhhhhdhhhdhhdd..",
			"..hhhhdhhhdhhhdd..",
			"..hhhhdhhhdhhddd..",
			"..hdhhdhhhdhhdDd..",
			"...ddddddddddDD...",
		}},
		side: pixMap{3, []string{
			".....hhhhhhhd.....",
			"...hhHHHHHhhhhd...",
			"..hhHHHHhhhhhhhd..",
			"..hHHHhhhhhhhhhd..",
			"..hHhhhhhhdhhdhd..",
			"..hhhhhhhdhd..d...",
			"..hhdhhhhhd.......",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hddhhhdd........",
			"...dddddd.........",
		}},
	},
	sim.HairLong: { // loose, parted in the middle, down past the shoulders
		front: pixMap{3, []string{
			"......hhhhhd......",
			"....hhHHhdHhhd....",
			"..hhHHHhddhHhhhd..",
			"..hHHhhd..dhhhhd..",
			"..hHhhd....dhhhd..",
			"..hhhd......dhhd..",
			"..hhd........dhd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........hd..",
			"..hd..........dd..",
			"..d............D..",
		}},
		behind: pixMap{14, []string{
			"...hhdhhhhhhdhd...",
			"...hhdhhhhhhdhd...",
			"...hhdhhhhhhdhd...",
			"...hhdhhhhhhdhd...",
			"...hhdhhhhhhdhd...",
			"...hddhhhhhddhd...",
			"...dhdhhdhhdhdd...",
			"....d.dd.dd.d.....",
		}},
		back: pixMap{3, []string{
			"......hhhhhd......",
			"....hhHHHHhhhd....",
			"..hhHHHHhhhhhhhd..",
			"..hHHHhhhhhhhhhd..",
			"..hHHhhhhhhhhhhd..",
			"..hHhhhhhhhhhhdd..",
			"..hhhhhdhhhdhhdd..",
			"..hhhhhdhhhdhhdd..",
			"..hhhhdhhhhdhhdd..",
			"..hhhhdhhhdhhhdd..",
			"..hhhhdhhhdhhhdd..",
			"..hhhdhhhhdhhhdd..",
			"..hhhdhhhdhhhddd..",
			"...hhdhhhdhhhdd...",
			"...hhdhhhdhhhdd...",
			"...hdhhhdhhhdhd...",
			"...hdhhhdhhhddD...",
			"...dhdhhdhhdhdD...",
			"....d.dd.dd.d.....",
		}},
		side: pixMap{3, []string{
			".....hhhhhhhd.....",
			"...hhHHHHHhhhhd...",
			"..hhHHHHhhhhhhhd..",
			"..hHHHhhhhhhhhhd..",
			"..hHhhhhhhhhhdd...",
			"..hhhhhhhhhd......",
			"..hhdhhhhhd.......",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hhdhhhhd........",
			"..hhdhhhd.........",
			"..hhdhhd..........",
			"..hdhhdd..........",
			"..dhdhd...........",
			"...ddD............",
		}},
	},
	sim.HairPonytail: { // pulled back into a long braid; a woman wears two
		front: pixMap{4, []string{
			".....hhHHhhhd.....",
			"...hhHHHHhdhhhd...",
			"...hHHhdhhhdhhd...",
			"...hhd......dhd...",
			"...d..........D...",
			"...d..........D...",
		}},
		back: pixMap{4, []string{
			".....hhHHhhhd.....",
			"...hhHHHHhhhhhd...",
			"...hHHHhhhhhhhd...",
			"...hHHhhhdhhhhd...",
			"...hHhhhhdhhhdd...",
			"...hhhdhhdhhdhd...",
			"...hhhdhhdhhdhd...",
			"...hhhhdhdhdhdd...",
			"...dhhhdhdhdhdD...",
			"....dhhhddhhdD....",
			".....ddddddDD.....",
		}},
		side: pixMap{4, []string{
			"....hhhhhhhhd.....",
			"...hhHHHHhhhhd....",
			"...hHHHhhhhdhhd...",
			"...hHhhhdhhhdd....",
			"...hhhdhhhd.......",
			"...hhdhhhd........",
			"...hhdhd..........",
			"...hdd............",
			"...dd.............",
		}},
	},
	sim.HairBun: { // tied up in a knot on the crown, the brow bare
		front: pixMap{0, []string{
			".......hhhd.......",
			"......hHHhhd......",
			"......dhhhdD......",
			"........rr........",
			"......hhhhhd......",
			"....hhHHdhhdhd....",
			"...hHHhdhhhhdhd...",
			"...hd........dd...",
			"...d..........D...",
		}},
		back: pixMap{0, []string{
			".......hhhd.......",
			"......hHHhhd......",
			"......dhhhdD......",
			"........rr........",
			"......hhhhhd......",
			"....hhHHdhhdhd....",
			"...hHHhhdhhdhhd...",
			"...hHHhhdhhdhhd...",
			"...hHhhdhhhhdhd...",
			"...hhhhdhhhhdhd...",
			"...hhhdhhhhhdhd...",
			"...hhhdhhhhhhdd...",
			"...dhhhhhhhhhdD...",
			"....dhhhhhhhdD....",
			".....ddddddDD.....",
		}},
		side: pixMap{0, []string{
			".....hhhd.........",
			"....hHHhhd........",
			"....dhhhdD........",
			"......rr..........",
			".....hhhhhhhd.....",
			"...hhHHHdhhhhd....",
			"...hHHhdhhhdhhd...",
			"...hHhdhhhhdd.....",
			"...hhdhhhd........",
			"...hhdhhd.........",
			"...hhdd...........",
			"...hd.............",
			"...dd.............",
		}},
	},
}

// A man's beard: a moustache parted under the nose, a full beard and a short
// braid with a bead, seen from the front and from the side. It is drawn a
// step off the hair's own colour (beardPalette), so the jaw shows.
var (
	beardFront = pixMap{12, []string{
		"...d..........d...",
		"...d.hhh..hhh.d...",
		"....dhhhmmhhhd....",
		".....hHHhhhdd.....",
		"......hHhhdd......",
		".......hhdd.......",
		"........o.........",
	}}
	beardSide = pixMap{13, []string{
		"........d...hhh...",
		"........dhhhhhh...",
		".........dhHHhhd..",
		"..........dhhhd...",
		"...........hdd....",
		"............o.....",
	}}
)

// The fur mantle on the shoulders, pinned with a round brooch.
var (
	mantleFront = pixMap{16, []string{
		"...fFFFFfffffuu...",
		"...ufyo....ufuU...",
		"...U.Ou......uU...",
	}}
	mantleBack = pixMap{16, []string{
		"...fFFFFFffffuu...",
		"...ffFffufffuuU...",
		"...uU.uu.uu.uUU...",
	}}
	mantleSide = pixMap{16, []string{
		".....fFFFFffu.....",
		".....ufFffuyo.....",
		".....U.uu.u.O.....",
	}}
	collarFront = pixMap{16, []string{"....fFFFffffuu...."}}
	collarSide  = pixMap{16, []string{"......fFFffu......"}}
)

// braid draws a plait two pixels wide down column x from row y0 to y1, tied
// near the end.
func braid(c *canvas, hr ramp, x, y0, y1 int) {
	for y := y0; y <= y1; y++ {
		a, b := hr[2], hr[1]
		switch {
		case y == y1:
			c.put(x+(y1-y0)%2, y, hr[1]) // the tuft
			continue
		case y == y1-1:
			a, b = hairTie, hairTie
		case (y-y0)%2 == 1:
			a, b = hr[0], hr[3]
		}
		c.put(x, y, a)
		c.put(x+1, y, b)
	}
}

// genPlayer draws one direction and walk frame of look l as four layers:
// the density-1 figure, which lookhd.go's HD figure took the place of (see
// playerHD); the bodies on the ground and the map markers are still made
// of it. Its walk frames are 0 stand, 1 left step, 2 stand, 3 right step.
//
// armorart.go, held.go and ride.go follow the same frame: 18x26 before the
// headroom, feet at (9, 24), the head centred on (9, 7).
func genPlayer(l sim.Look, d sim.Dir, frame int) (back, legs, shirt, top sprite) {
	if d == sim.DirLeft {
		b, lg, s, t := genPlayer(l, sim.DirRight, frame)
		return mirrorSprite(b), mirrorSprite(lg), mirrorSprite(s), mirrorSprite(t)
	}
	B, L, S, T := playerLayers(l, d, frame)
	return mk(B, 9, bodyFeet), mk(L, 9, bodyFeet), mk(S, 9, bodyFeet), mk(T, 9, bodyFeet)
}

// playerLayers draws the four layers of genPlayer; d is not DirLeft. What
// goes where: B is hair behind the body; L the legs, which stay on the
// ground while the rest bobs; S the clothes, in greys the player's colour
// tints; T the head, hands, fur, belt and brooches.
func playerLayers(l sim.Look, d sim.Dir, frame int) (B, L, S, T *canvas) {
	return playerLayersWith(l, d, frame, false)
}

// playerLayersWith is playerLayers, with the eyes shut if asleep.
func playerLayersWith(l sim.Look, d sim.Dir, frame int, asleep bool) (B, L, S, T *canvas) {
	return playerLayersPose(l, d, frame, asleep, armNone)
}

// playerLayersPose is playerLayersWith, drawn without the arm hide names:
// punch.go draws that arm itself, thrown out in a punch.
func playerLayersPose(l sim.Look, d sim.Dir, frame int, asleep bool, hide armSide) (B, L, S, T *canvas) {
	l = l.Sanitized()
	sk := skinTones[l.Skin].ramp
	B, L, S, T = newCanvas(bodyW, bodyH), newCanvas(bodyW, bodyH), newCanvas(bodyW, bodyH), newCanvas(bodyW, bodyH)
	step := walkStep(frame)
	pal := artPalette(l)
	female, up := l.Female, d == sim.DirUp
	cloth := func(x, y, tone int) { S.put(x, y, shirtRamp.at(tone)) }

	// leg draws a trouser leg three wide from column x0 down to row bottom:
	// cloth, cross-bound wraps, then a boot with a fur cuff, its toe
	// pointing toe columns outwards (0 for none). dim darkens the far leg.
	leg := func(x0, bottom, dim, toe int) {
		for y := 19; y <= bottom; y++ {
			for i := range 3 {
				x := x0 + i
				tone := 2 - dim
				if i == 2 {
					tone = 0
				} else if i == 0 && dim == 0 {
					tone = 3
				}
				switch {
				case y == bottom:
					L.put(x, y, darkHideRamp.at(tone))
				case y == bottom-1:
					L.put(x, y, mantleRamp.at(tone+1))
				case y >= bottom-3:
					if y == bottom-2 && i != 2 {
						tone-- // the binding
					}
					L.put(x, y, wrapRamp.at(tone))
				default:
					L.put(x, y, trouserRamp.at(min(tone, 2)))
				}
			}
		}
		if toe != 0 {
			tx := x0 - 1
			if toe > 0 {
				tx = x0 + 3
			}
			L.put(tx, bottom, darkHideRamp.at(1-dim))
		}
	}
	// hand is a fist two pixels square with its top-left at (x, y).
	hand := func(c *canvas, x, y, lit int) {
		c.put(x, y, sk.at(lit+1))
		c.put(x+1, y, sk.at(lit))
		c.put(x, y+1, sk.at(lit))
		c.put(x+1, y+1, sk.at(lit-1))
	}

	switch d {
	case sim.DirDown, sim.DirUp:
		lb, rb := bodyFeet-1, bodyFeet-1
		if step < 0 {
			lb--
		} else if step > 0 {
			rb--
		}
		leg(6, lb, 0, -1)
		leg(9, rb, 1, 1)

		// The tunic or the dress.
		for y := 16; y <= 25; y++ {
			x0, x1, ok := torsoRow(female, d, y)
			for x := x0; ok && x <= x1; x++ {
				if !female {
					tone := 2
					switch {
					case x <= x0+1:
						tone = 3
					case x >= x1-1:
						tone = 1
					}
					switch {
					case y == 21: // a pale band along the hem
						tone = 3
						if x >= x1-1 {
							tone = 2
						}
					case y == 20 || y == 18:
						tone-- // under the belt, under the mantle
					}
					cloth(x, y, tone)
					continue
				}
				// A light shift, and over it the apron dress, darker.
				tone := 1
				switch {
				case x == x0+1 || x == x0+2:
					tone = 2
				case x == x1:
					tone = 0
				}
				switch {
				case y == 25:
					tone = 3 // the shift's hem under the apron
					if x >= x1-1 {
						tone = 2
					}
				case y == 24:
					tone = 0 // a woven band
				case y == 16:
					tone = 3
				case y == 17 && (x == 8 || x == 9) && !up:
					tone = 3 // the shift at the neck
				case y >= 21 && y <= 23 && x == x1-2:
					tone = 0 // a fold
				}
				if y == 25 && (step < 0 && x < 9 || step > 0 && x >= 9) {
					continue // the hem lifts over the stepping foot
				}
				cloth(x, y, tone)
			}
		}

		// Arms swing opposite to the legs.
		lx, rx := 3, 13
		if female {
			lx, rx = 4, 12
		}
		la, ra := -step, step
		for y := 16; y <= 20+la && hide != armOff; y++ {
			a, b := 3, 0 // lit, then the line between the arm and the body
			if y == 20+la && !female {
				a, b = 2, 0 // the cuff
			}
			cloth(lx, y, a)
			cloth(lx+1, y, b)
		}
		for y := 16; y <= 20+ra && hide != armHand; y++ {
			a, b := 0, 2
			if y == 20+ra && !female {
				a, b = 0, 1
			}
			cloth(rx, y, a)
			cloth(rx+1, y, b)
		}
		if hide != armOff {
			hand(T, lx, 21+la, 2)
		}
		if hide != armHand {
			hand(T, rx, 21+ra, 1)
		}

		if !female {
			// A leather belt with a buckle; its end hangs down.
			for x := 5; x <= 12; x++ {
				tone := 2
				if x >= 11 {
					tone = 1
				} else if x <= 6 {
					tone = 3
				}
				T.put(x, 19, leatherRamp.at(tone))
			}
			if up {
				mantleBack.stamp(T, pal)
			} else {
				T.put(8, 19, broochRamp[3])
				T.put(9, 19, broochRamp[1])
				T.put(10, 20, leatherRamp[1])
				T.put(10, 21, leatherRamp[0])
				mantleFront.stamp(T, pal)
			}
		} else {
			collarFront.stamp(T, pal)
			if up {
				// The apron's straps cross on the back.
				for y := 17; y <= 18; y++ {
					cloth(7, y, 0)
					cloth(10, y, 0)
				}
			} else {
				// Two oval brooches with a string of beads between.
				for _, x := range []int{6, 11} {
					T.put(x, 17, broochRamp[3])
					T.put(x, 18, broochRamp[1])
				}
				T.put(7, 18, beadColors[0])
				T.put(8, 19, beadColors[1])
				T.put(9, 19, beadColors[2])
				T.put(10, 18, beadColors[3])
			}
		}

	case sim.DirRight:
		near, far := 7+2*step, 8-2*step
		leg(far, bodyFeet-1, 1, 1)
		leg(near, bodyFeet-1, 0, 1)
		for y := 16; y <= 25; y++ {
			x0, x1, ok := torsoRow(female, d, y)
			for x := x0; ok && x <= x1; x++ {
				// The body is a step down from the near arm, darkest at
				// the back.
				tone := 1
				if x == x1 {
					tone = 2
				}
				switch {
				case !female && y == 21:
					tone = 3
					if x == x0 {
						tone = 2
					}
				case female && y == 25:
					tone = 3
				case female && y == 24:
					tone = 0
				case female && y >= 21 && y <= 23 && x == x1-2:
					tone = 0
				case female && y >= 21 && x > x0 && x < x1-2:
					tone = 2
				}
				cloth(x, y, max(tone, 0))
			}
		}
		// The near arm, set off from the body by a darker edge each side.
		for y := 16; y <= 20 && hide != armHand; y++ {
			x, _ := sideArmX(step, y)
			a, b := 3, 3
			if y >= 19 {
				a, b = 3, 2
			}
			if y == 20 && !female {
				a, b = 2, 2 // the cuff
			}
			cloth(x-1, y, 0)
			cloth(x, y, a)
			cloth(x+1, y, b)
			if _, x1, _ := torsoRow(female, d, y); x+2 <= x1 {
				cloth(x+2, y, 0)
			}
		}
		if hx, _ := sideArmX(step, 21); hide != armHand {
			hand(T, hx, 21, 2)
		}
		// The far hand shows past the body as it swings the other way.
		if step > 0 {
			T.put(12, 19, sk[1])
			T.put(12, 20, sk[0])
		} else if step < 0 {
			T.put(5, 19, sk[1])
			T.put(5, 20, sk[0])
		}
		if !female {
			for x := 6; x <= 11; x++ {
				if ax, _ := sideArmX(step, 19); hide != armHand && (x == ax || x == ax+1) {
					continue
				}
				T.put(x, 19, leatherRamp.at(2-(x-6)/3))
			}
			T.put(11, 19, broochRamp[2])
			mantleSide.stamp(T, pal)
		} else {
			collarSide.stamp(T, pal)
			T.put(11, 17, broochRamp[3])
			T.put(11, 18, broochRamp[1])
		}
	}

	drawHead(T, B, l, d, 0, asleep)
	L.outline(0.7)
	rimLayers(B, S, T)
	return B, L, S, T
}

// rimLayers gives the layers that bob together one shared dark outline,
// round their joint silhouette: each rim pixel goes in the layer it borders,
// so the rim of the clothes takes the player's colour too.
func rimLayers(layers ...*canvas) {
	type rim struct {
		c    *canvas
		x, y int
		col  color.RGBA
	}
	top := func(x, y int) (*canvas, color.RGBA) {
		for i := len(layers) - 1; i >= 0; i-- {
			if layers[i].opaque(x, y) {
				return layers[i], layers[i].at(x, y)
			}
		}
		return nil, color.RGBA{}
	}
	var rims []rim
	c0 := layers[0]
	for y := 0; y < c0.h; y++ {
		for x := 0; x < c0.w; x++ {
			if c, _ := top(x, y); c != nil {
				continue
			}
			// The light is from the top left: the rim is thinner in
			// colour above and to the left, heavy below and to the right.
			for i, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
				if c, col := top(x+d[0], y+d[1]); c != nil {
					rims = append(rims, rim{c, x, y, mix(col, colInk, rimInk[i])})
					break
				}
			}
		}
	}
	for _, r := range rims {
		r.c.put(r.x, r.y, r.col)
	}
}

// rimInk is how dark rimLayers makes the rim above, left of, right of and
// below the figure.
var rimInk = [4]float64{0.64, 0.7, 0.82, 0.86}

// headMarkDens is how many pixels of a head marker's image go to one of the
// density-1 head it took the place of.
const headMarkDens = 2

// genHeadMark draws l's head facing the viewer, for markers on the maps:
// the jointed figure's (tallhead.go), its hair and its beard.
func genHeadMark(l sim.Look) *ebiten.Image { return headMarkCanvas(l).image() }

func headMarkCanvas(l sim.Look) *canvas {
	const w, h, x0, y0 = 32, 36, limbAX*HDDensity - 16, 19
	hd := headCache(nil).get(l.Sanitized(), headKey{dir: sim.DirDown})
	innerLines(0.4, hd.B, hd.T)
	c := newCanvas(w, h)
	for _, src := range []*canvas{hd.B.c, hd.T.c} {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if p := src.at(x+x0, y+y0); p.A >= 128 {
					c.put(x, y, p)
				}
			}
		}
	}
	c.outline(0.75)
	return c
}

// drawHead draws l's head, face, hair and beard in c, oy rows below its
// place in the player sprite, where it is centred on (9, 11). Hair hanging
// behind the body goes in behind. d is never DirLeft: that is the mirror
// image of DirRight. shut closes the eyes.
func drawHead(c, behind *canvas, l sim.Look, d sim.Dir, oy int, shut bool) {
	l = l.Sanitized()
	hr, sk := hairColors[l.HairColor].ramp, skinTones[l.Skin].ramp
	pal := artPalette(l)
	ha := hairArts[l.Hair]
	hair := ha.front
	switch d {
	case sim.DirUp:
		hair = ha.back
	case sim.DirRight:
		hair = ha.side
	}
	shift := func(m pixMap) pixMap { m.y0 += oy; return m }
	put := func(x, y int, col color.RGBA) { c.put(x, y+oy, col) }

	// The head: bright, so the face stands out of the hair; a step darker
	// down the far cheek, under the chin and in the shadow of the hair. In
	// profile the whole face is in the light.
	side := d == sim.DirRight
	for y := 5; y <= 15; y++ {
		x0, x1, _ := headSpan(y)
		for x := x0; x <= x1; x++ {
			tone := 3
			switch {
			case y == 15 || x == x1 && !side:
				tone = 1
			case x == x1 || x >= x1-2 && !side || y == 14 && x >= x1-4:
				tone = 2
			}
			if hair.has(x, y-1) && !hair.has(x, y) && y <= 10 {
				tone-- // the fringe's shadow
			}
			put(x, y, sk.at(tone))
		}
	}
	blush := skinTones[l.Skin].blush
	iris := irisColors[l.Skin]
	lips := mix(blush, mouthDark, 0.5)
	brow := hr[1]
	if luma(brow) < 45 {
		brow = mix(hr[2], sk[1], 0.3) // black brows would run into the eyes
	}
	// eye is two wide and three tall: the lid, a glint beside the pupil,
	// the iris beside it.
	eye := func(ex int) {
		if shut {
			put(ex, 12, eyeDark)
			put(ex+1, 12, eyeDark)
			return
		}
		put(ex, 10, eyeDark)
		put(ex+1, 10, eyeDark)
		put(ex, 11, eyeShine)
		put(ex+1, 11, eyeDark)
		put(ex, 12, iris)
		put(ex+1, 12, eyeDark)
	}
	switch d {
	case sim.DirDown:
		eye(5)
		eye(11)
		if l.Female {
			put(4, 9, eyeDark) // lashes
			put(13, 9, eyeDark)
			put(4, 13, blush)
			put(5, 13, blush)
			put(12, 13, blush)
			put(13, 13, blush)
			put(8, 14, lips)
			put(9, 14, lips)
		} else {
			for _, bx := range []int{4, 5, 6, 11, 12, 13} { // heavy brows
				put(bx, 9, brow)
			}
			put(4, 13, blush)
			put(13, 13, blush)
		}
		put(9, 12, sk[1]) // the nose's shadow
	case sim.DirRight:
		put(14, 12, sk[3]) // the nose
		put(14, 13, sk[1])
		eye(11)
		put(7, 11, sk[1]) // the ear
		put(7, 12, sk[0])
		put(8, 12, sk[1])
		if l.Female {
			put(13, 9, eyeDark)
			put(10, 13, blush)
			put(11, 13, blush)
			put(13, 14, lips)
			put(14, 14, lips)
		} else {
			put(11, 9, brow)
			put(12, 9, brow)
			put(13, 9, brow)
			put(10, 13, blush)
		}
	}

	shift(hair).stamp(c, pal)
	if d == sim.DirDown && len(ha.behind.rows) > 0 {
		shift(ha.behind).stamp(behind, pal)
	}
	if !l.Female {
		switch d {
		case sim.DirDown:
			shift(beardFront).stamp(c, beardPalette(l))
		case sim.DirRight:
			shift(beardSide).stamp(c, beardPalette(l))
		}
	}
	switch l.Hair {
	case sim.HairBob: // a thin braid by the cheek
		switch d {
		case sim.DirDown:
			braid(c, hr, 2, 15+oy, 19+oy)
		case sim.DirRight:
			braid(c, hr, 6, 14+oy, 18+oy)
		}
	case sim.HairPonytail:
		switch {
		case d == sim.DirUp && l.Female:
			braid(c, hr, 4, 14+oy, 21+oy)
			braid(c, hr, 12, 14+oy, 21+oy)
		case d == sim.DirUp:
			braid(c, hr, 8, 14+oy, 22+oy)
		case d == sim.DirRight:
			braid(c, hr, 3, 13+oy, 21+oy)
		case l.Female:
			braid(c, hr, 4, 13+oy, 20+oy)
			braid(c, hr, 12, 13+oy, 20+oy)
		default: // over one shoulder
			braid(c, hr, 13, 13+oy, 20+oy)
		}
	}
}
