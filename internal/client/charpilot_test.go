package client

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"survivalgame/internal/sim"
)

// The style pilot's sheets (charpilot.go): the current figure and the three
// new looks side by side, enlarged, to choose between:
//
//	CHAR_PILOT=dir go test ./internal/client -run CharPilotSheets
//
// writes compare_front.png (standing, facing the viewer, four times the
// size), compare_turnaround.png (front, side, back), compare_motion.png
// (idle, walk, run and an axe's swing, from the side; the axe a stand-in),
// faces.png and hands.png (sixteen times), and for each style
// hair_<style>.png (every hair style, hair colour and skin tone),
// close_<style>.png (the upper body, ten times) and detail_<style>.png
// (the whole figure, eight times).

var pilotStyles = []struct {
	s    charStyle
	name string
}{
	{charCurrent, "current"},
	{charRefined, "a refined"},
	{charHeroic, "b heroic"},
	{charPainterly, "c painterly"},
}

var pilotLooks = []struct {
	l    sim.Look
	name string
}{
	{sim.Look{Hair: sim.HairShort, HairColor: 0, Skin: 0}, "man short hair"},
	{sim.Look{Hair: sim.HairLong, HairColor: 1, Skin: 3}, "man long hair"},
	{sim.Look{Female: true, Hair: sim.HairLong, HairColor: 2, Skin: 1}, "woman"},
	{sim.Look{Female: true, Hair: sim.HairPonytail, HairColor: 3, Skin: 4}, "woman braids"},
}

func setPilot(t *testing.T, s charStyle) {
	t.Helper()
	pilotStyle = s
	pilotSkinHold = map[*figGen]ramp{}
	pilotHoles = nil
	pilotOwned = map[*canvas]*figLayer{}
	pilotOwner = map[*canvas]*figGen{}
	pilotLegs = map[*canvas]*figLayer{}
}

// pilotShot puts one frame of the jointed figure together on img, its feet
// at (fx, fy), each image pixel k across, as limbShot does; axe draws an
// axe in the hand.
func pilotShot(img *image.RGBA, l sim.Look, up upKey, lo loKey, fx, fy, k int, axe bool) {
	mir := up.dir == sim.DirLeft
	if mir {
		up.dir = sim.DirRight
	}
	lo.dir = up.dir
	if lo.anim == loSit {
		up.frame |= upSit
	}
	uc := bakeUpper(l, 0, up, headCache{})
	lc, lp := bakeLower(l, 0, lo)
	skew := 0.0
	if up.dir == sim.DirRight {
		skew = math.Tan(lp.lean)
	}
	const hip = limbGround - hipHeight*HDDensity
	set := func(x, y float64, c color.RGBA) {
		if mir {
			x = float64(limbAX*HDDensity)*2 - 1 - x
		}
		px, py := fx+(int(math.Round(x))-limbAX*HDDensity)*k, fy+(int(math.Round(y))-limbGround)*k
		for dy := 0; dy < k; dy++ {
			for dx := 0; dx < k; dx++ {
				if image.Pt(px+dx, py+dy).In(img.Rect) {
					img.SetRGBA(px+dx, py+dy, c)
				}
			}
		}
	}
	put := func(c *canvas, tinted, upper bool) {
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				q := c.px[y*c.w+x]
				if q.A < 128 {
					continue
				}
				if tinted {
					q = limbTint(q)
				}
				xx, yy := float64(x), float64(y)
				if upper {
					xx += skew * (hip - float64(y))
					yy += lp.drop
				}
				set(xx, yy, q)
			}
		}
	}
	h := uc.arm[0]
	tool := func() {
		if !axe {
			return
		}
		hx := h.hx + limbPad*HDDensity + skew*(hip-h.hy)
		hy := h.hy + lp.drop
		// the haft across the fist, square to the forearm, out of the thumb's side
		a := math.Pi/2 - h.swing
		wood, wood2 := color.RGBA{132, 92, 54, 255}, color.RGBA{92, 60, 36, 255}
		stone, stone2 := color.RGBA{150, 152, 160, 255}, color.RGBA{96, 96, 108, 255}
		for r := -5.0; r < 22; r += 0.4 {
			for w := -1.0; w <= 1; w += 0.5 {
				c := wood
				if w > 0.4 {
					c = wood2
				}
				set(hx+math.Sin(a)*r+math.Cos(a)*w, hy-math.Cos(a)*r+math.Sin(a)*w, c)
			}
		}
		for r := 15.0; r < 22; r += 0.4 {
			for w := -7.0; w <= 1; w += 0.5 {
				c := stone
				if w < -5 || r > 20.5 {
					c = stone2
				}
				set(hx+math.Sin(a)*r+math.Cos(a)*w, hy-math.Cos(a)*r+math.Sin(a)*w, c)
			}
		}
	}
	over := h.layer == armOver
	if !over && up.dir == sim.DirUp {
		tool()
	}
	put(uc.layer[upFar], true, true)
	put(uc.layer[upBack], false, true)
	put(lc, false, false)
	put(uc.layer[upBody], false, true)
	put(uc.layer[upShirt], true, true)
	put(uc.layer[upArms], true, true)
	put(uc.layer[upTop], false, true)
	if !over && up.dir != sim.DirUp {
		tool()
	}
	put(uc.layer[upArmsOver], true, true)
	if over {
		tool()
	}
	put(uc.layer[upTopOver], false, true)
}

// pilotFont adds digits and a few marks to font3x5.
var pilotFont = map[rune][5]uint8{
	'0': {7, 5, 5, 5, 7}, '1': {2, 6, 2, 2, 7}, '2': {6, 1, 2, 4, 7}, '3': {6, 1, 2, 1, 6}, '4': {5, 5, 7, 1, 1},
	'5': {7, 4, 6, 1, 6}, '6': {3, 4, 7, 5, 7}, '7': {7, 1, 2, 2, 2}, '8': {7, 5, 7, 5, 7}, '9': {7, 5, 7, 1, 6},
	'-': {0, 0, 7, 0, 0}, '/': {1, 1, 2, 4, 4}, '+': {0, 2, 7, 2, 0}, '.': {0, 0, 0, 0, 2}, ':': {0, 2, 0, 2, 0},
}

// pilotLabel writes s at (x, y), each of the font's pixels sc across.
func pilotLabel(img *image.RGBA, x, y, sc int, s string, col color.RGBA) {
	for i, r := range s {
		g, ok := font3x5[r]
		if !ok {
			g, ok = pilotFont[r]
		}
		if !ok {
			continue
		}
		for row := 0; row < 5; row++ {
			for c := 0; c < 3; c++ {
				if g[row]>>(2-c)&1 == 1 {
					for dy := 0; dy < sc; dy++ {
						for dx := 0; dx < sc; dx++ {
							img.SetRGBA(x+i*4*sc+c*sc+dx, y+row*sc+dy, col)
						}
					}
				}
			}
		}
	}
}

func pilotBackdrop(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// the game's grass, a little mottled
			v := 0.5 + 0.5*vnoise(float64(x)/23, float64(y)/23, 5)
			c := mix(hex(0x4f7d3a), hex(0x6c9a48), v)
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func pFillRect(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// TestCharPilotBakes draws every style in every facing, standing, striding,
// running, chopping, punching and lying dead, bare and clothed: each must
// leave nothing of its bookkeeping behind (a skin held back from a hand,
// a fist's dots to put back, layers waiting for their outline), stand on
// the ground line, and keep its pixels opaque where the old fist's dots
// were put back.
func TestCharPilotBakes(t *testing.T) {
	defer setPilot(t, charCurrent)
	lo2, up2 := moChop.keys(sim.DirRight, 2)
	type pose struct {
		up upKey
		lo loKey
	}
	for _, st := range pilotStyles[1:] {
		setPilot(t, st.s)
		for _, l := range []sim.Look{pilotLooks[0].l, pilotLooks[1].l, pilotLooks[2].l, {Hair: sim.HairShaggy, Skin: 5}, {Female: true, Hair: sim.HairBun, HairColor: 7}} {
			for _, w := range []sim.Wear{0, sim.WearAll} {
				for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
					up2.dir, lo2.dir = d, d
					for _, p := range []pose{
						{upKey{dir: d}, loKey{dir: d}},
						{upKey{d, upWalk, 3}, loKey{d, loWalk, 6}},
						{upKey{d, upRun, 5}, loKey{d, loRun, 10}},
						{upKey{d, upPunch, atkFrames - 1}, loKey{dir: d}},
						{up2, lo2},
						{upKey{d, upFallen, 0}, loKey{dir: d}},
					} {
						uc := bakeUpper(l, w, p.up, headCache{})
						lc, _ := bakeLower(l, w, p.lo)
						if len(pilotSkinHold) != 0 || len(pilotHoles) != 0 || len(pilotOwned) != 0 || len(pilotOwner) != 0 || len(pilotLegs) != 0 {
							t.Fatalf("%s %+v wear %d facing %d %+v: left behind %d skins, %d dots, %d layers, %d legs", st.name, l, w, d, p.up,
								len(pilotSkinHold), len(pilotHoles), len(pilotOwned), len(pilotLegs))
						}
						if uc.layer[upTop] == nil || lc == nil {
							t.Fatalf("%s: no drawing", st.name)
						}
						if p.lo.anim == loStand && p.up.anim == upStand {
							bottom := -1
							for i, q := range lc.px {
								if q.A >= 128 {
									bottom = i / lc.w
								}
							}
							// the soles on limbGround, their outline a row under
							if bottom < limbGround-1 || bottom > limbGround+1 {
								t.Errorf("%s %+v facing %d: the feet end on row %d, the ground is %d", st.name, l, d, bottom, limbGround)
							}
						}
					}
				}
			}
		}
	}
}

func TestCharPilotSheets(t *testing.T) {
	dir := os.Getenv("CHAR_PILOT")
	if dir == "" {
		t.Skip("set CHAR_PILOT to a folder")
	}
	defer setPilot(t, charCurrent)
	white := color.RGBA{255, 255, 255, 255}
	dark := color.RGBA{24, 22, 30, 255}

	// compare_front: every style a column, every look a row, standing
	// facing the viewer, four times the size.
	{
		const k, cw, ch, top = 4, 230, 450, 40
		img := pilotBackdrop(len(pilotStyles)*cw, top+len(pilotLooks)*ch)
		pFillRect(img, image.Rect(0, 0, img.Rect.Dx(), top), dark)
		for si, st := range pilotStyles {
			setPilot(t, st.s)
			pilotLabel(img, si*cw+10, 10, 4, st.name, white)
			for li, lk := range pilotLooks {
				pilotShot(img, lk.l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}, si*cw+cw/2, top+li*ch+ch-24, k, false)
				if si == 0 {
					pilotLabel(img, 6, top+li*ch+6, 2, lk.name, white)
				}
			}
		}
		limbPNG(t, filepath.Join(dir, "compare_front.png"), img)
	}

	// compare_turnaround: every style a row; for three looks, front, side
	// and back; three times the size.
	{
		const k, cw, ch, left = 3, 150, 340, 150
		looks := pilotLooks[:3]
		img := pilotBackdrop(left+len(looks)*3*cw, len(pilotStyles)*ch+30)
		pFillRect(img, image.Rect(0, 0, img.Rect.Dx(), 30), dark)
		for li, lk := range looks {
			pilotLabel(img, left+li*3*cw+10, 8, 3, lk.name, white)
		}
		for si, st := range pilotStyles {
			setPilot(t, st.s)
			pilotLabel(img, 6, 30+si*ch+10, 3, st.name, white)
			for li, lk := range looks {
				for di, d := range []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp} {
					pilotShot(img, lk.l, upKey{dir: d}, loKey{dir: d}, left+(li*3+di)*cw+cw/2, 30+si*ch+ch-20, k, false)
				}
			}
		}
		limbPNG(t, filepath.Join(dir, "compare_turnaround.png"), img)
	}

	// compare_motion: every style a row; a man and a woman from the side,
	// standing, walking, running and swinging an axe.
	{
		const k, cw, ch, left = 3, 170, 340, 150
		type fr struct {
			up  upKey
			lo  loKey
			axe bool
		}
		lo2, up2 := moChop.keys(sim.DirRight, 1)
		frames := []fr{
			{upKey{dir: sim.DirRight}, loKey{dir: sim.DirRight}, false},
			{upKey{sim.DirRight, upWalk, 1}, loKey{sim.DirRight, loWalk, 3}, false},
			{upKey{sim.DirRight, upRun, 1}, loKey{sim.DirRight, loRun, 3}, false},
			{up2, lo2, true},
		}
		looks := []sim.Look{pilotLooks[0].l, pilotLooks[2].l}
		img := pilotBackdrop(left+len(looks)*len(frames)*cw, len(pilotStyles)*ch+30)
		pFillRect(img, image.Rect(0, 0, img.Rect.Dx(), 30), dark)
		for li := range looks {
			for fi, n := range []string{"idle", "walk", "run", "axe"} {
				pilotLabel(img, left+(li*len(frames)+fi)*cw+40, 8, 3, n, white)
			}
		}
		for si, st := range pilotStyles {
			setPilot(t, st.s)
			pilotLabel(img, 6, 30+si*ch+10, 3, st.name, white)
			for li, l := range looks {
				for fi, f := range frames {
					pilotShot(img, l, f.up, f.lo, left+(li*len(frames)+fi)*cw+cw/2, 30+si*ch+ch-20, k, f.axe)
				}
			}
		}
		limbPNG(t, filepath.Join(dir, "compare_motion.png"), img)
	}

	// For each new style: every hair style on a man and a woman, front and
	// side, and every hair colour and skin tone.
	for _, st := range pilotStyles[1:] {
		setPilot(t, st.s)
		const k, cw, ch = 3, 120, 330
		img := pilotBackdrop(int(sim.NumHairStyles)*2*cw, 3*ch)
		for fi := 0; fi < 2; fi++ {
			for h := 0; h < int(sim.NumHairStyles); h++ {
				l := sim.Look{Female: fi == 1, Hair: sim.HairStyle(h), HairColor: uint8((h*3 + fi) % 8), Skin: uint8((h + fi*2) % 6)}
				for di, d := range []sim.Dir{sim.DirDown, sim.DirRight} {
					pilotShot(img, l, upKey{dir: d}, loKey{dir: d}, (h*2+di)*cw+cw/2, fi*ch+ch-16, k, false)
				}
			}
		}
		for c := 0; c < 8; c++ {
			l := sim.Look{Female: c%2 == 1, Hair: sim.HairShort, HairColor: uint8(c), Skin: uint8(c % 6)}
			if c%2 == 1 {
				l.Hair = sim.HairBob
			}
			pilotShot(img, l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}, c*cw+cw/2, 2*ch+ch-16, k, false)
		}
		for s := 0; s < 6; s++ {
			l := sim.Look{Hair: sim.HairBun, HairColor: 0, Skin: uint8(s)}
			pilotShot(img, l, upKey{dir: sim.DirRight}, loKey{dir: sim.DirRight}, (8+s)*cw+cw/2, 2*ch+ch-16, k, false)
		}
		name := []string{"", "a", "b", "c"}[st.s]
		limbPNG(t, filepath.Join(dir, "hair_"+name+".png"), img)
	}

	// The upper bodies close, ten times the size: a man and a woman, from
	// the front and the side, and the man walking.
	for _, st := range pilotStyles {
		setPilot(t, st.s)
		const k, cw, ch = 10, 420, 620
		img := pilotBackdrop(5*cw, ch)
		shots := []struct {
			l  sim.Look
			up upKey
			lo loKey
		}{
			{pilotLooks[0].l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}},
			{pilotLooks[0].l, upKey{dir: sim.DirRight}, loKey{dir: sim.DirRight}},
			{pilotLooks[0].l, upKey{sim.DirRight, upWalk, 1}, loKey{sim.DirRight, loWalk, 3}},
			{pilotLooks[2].l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}},
			{pilotLooks[2].l, upKey{dir: sim.DirRight}, loKey{dir: sim.DirRight}},
		}
		for i, s := range shots {
			pilotShot(img, s.l, s.up, s.lo, i*cw+cw/2, ch+(limbGround-86)*k, k, false)
		}
		name := []string{"cur", "a", "b", "c"}[st.s]
		limbPNG(t, filepath.Join(dir, "close_"+name+".png"), img)
	}

	// The faces, sixteen times the size: every style a row; a man with short
	// hair, one with long hair and a woman, from the front and the side.
	{
		const k, cw, ch = 16, 360, 440
		img := pilotBackdrop(6*cw, len(pilotStyles)*ch)
		looks := []sim.Look{pilotLooks[0].l, pilotLooks[1].l, pilotLooks[2].l}
		for si, st := range pilotStyles {
			setPilot(t, st.s)
			for li, l := range looks {
				for di, d := range []sim.Dir{sim.DirDown, sim.DirRight} {
					sub := image.NewRGBA(image.Rect(0, 0, cw, ch))
					pilotShot(sub, l, upKey{dir: d}, loKey{dir: d}, cw/2, ch+(limbGround-50)*k, k, false)
					ox, oy := (li*2+di)*cw, si*ch
					for y := 0; y < ch; y++ {
						for x := 0; x < cw; x++ {
							if c := sub.RGBAAt(x, y); c.A > 0 {
								img.SetRGBA(ox+x, oy+y, c)
							}
						}
					}
				}
			}
		}
		limbPNG(t, filepath.Join(dir, "faces.png"), img)
	}

	// The hands, sixteen times the size, in the poses that show them: every
	// style a row.
	{
		const k, cw, ch = 16, 240, 240
		type pose struct {
			l   sim.Look
			up  upKey
			lo  loKey
			arm int
		}
		lo2, up2 := moChop.keys(sim.DirRight, 1)
		poses := []pose{
			{pilotLooks[0].l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}, 0},
			{pilotLooks[0].l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}, 1},
			{pilotLooks[0].l, upKey{dir: sim.DirRight}, loKey{dir: sim.DirRight}, 0},
			{pilotLooks[0].l, upKey{sim.DirRight, upWalk, 1}, loKey{sim.DirRight, loWalk, 3}, 0},
			{pilotLooks[0].l, upKey{sim.DirRight, upRun, 1}, loKey{sim.DirRight, loRun, 3}, 0},
			{pilotLooks[0].l, up2, lo2, 0},
			{pilotLooks[0].l, upKey{dir: sim.DirUp}, loKey{dir: sim.DirUp}, 0},
			{pilotLooks[2].l, upKey{dir: sim.DirDown}, loKey{dir: sim.DirDown}, 0},
		}
		img := pilotBackdrop(len(poses)*cw, len(pilotStyles)*ch)
		for si, st := range pilotStyles {
			setPilot(t, st.s)
			for pi, p := range poses {
				uc := bakeUpper(p.l, 0, p.up, headCache{})
				_, lp := bakeLower(p.l, 0, p.lo)
				h := uc.arm[p.arm]
				setPilot(t, st.s)
				sub := image.NewRGBA(image.Rect(0, 0, cw, ch))
				fx := cw/2 - int(math.Round(h.hx+limbPad*HDDensity-limbAX*HDDensity))*k
				fy := ch/2 - int(math.Round(h.hy+lp.drop-limbGround))*k
				pilotShot(sub, p.l, p.up, p.lo, fx, fy, k, false)
				for y := 0; y < ch; y++ {
					for x := 0; x < cw; x++ {
						if c := sub.RGBAAt(x, y); c.A > 0 {
							img.SetRGBA(pi*cw+x, si*ch+y, c)
						}
					}
				}
			}
		}
		limbPNG(t, filepath.Join(dir, "hands.png"), img)
	}

	// The heads and hands close: eight times the size.
	for _, st := range pilotStyles {
		setPilot(t, st.s)
		const k, cw, ch = 8, 340, 860
		img := pilotBackdrop(6*cw, ch)
		for li, l := range []sim.Look{pilotLooks[0].l, pilotLooks[2].l} {
			for di, d := range []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp} {
				pilotShot(img, l, upKey{dir: d}, loKey{dir: d}, (li*3+di)*cw+cw/2, ch-10, k, false)
			}
		}
		name := []string{"cur", "a", "b", "c"}[st.s]
		limbPNG(t, filepath.Join(dir, "detail_"+name+".png"), img)
	}
}
