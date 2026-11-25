package client

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"survivalgame/internal/sim"
)

// TestPlayerStanceFeetStayPlanted checks that a foot on the ground goes back
// under the body exactly as fast as the ground does, in every facing: the
// stride is matched to the ground covered, so nothing slides.
func TestPlayerStanceFeetStayPlanted(t *testing.T) {
	for name, gd := range map[string]gaitDef{"walk": stepGait, "run": runGait} {
		anim := loWalk
		if name == "run" {
			anim = loRun
		}
		step := gd.cycle * HDDensity / loFrames // image pixels of ground a frame
		for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
			planted := 0
			for f := 0; f < loFrames; f++ {
				p0 := lowerPoseOf(loKey{d, anim, uint8(f)})
				p1 := lowerPoseOf(loKey{d, anim, uint8((f + 1) % loFrames)})
				for i := 0; i < 2; i++ {
					ph := (float64(f)+0.5)/loFrames + 0.5*float64(i)
					if !gd.stanceAt(ph) || !gd.stanceAt(ph+1.0/loFrames) {
						continue
					}
					planted++
					a, b := legGeo(d, i, p0), legGeo(d, i, p1)
					// The ankle's rise as the foot rolls is not a slide: take it out.
					ay, by := a.ay+p0.leg[i].u, b.ay+p1.leg[i].u
					var dx, dy float64
					switch d {
					case sim.DirRight:
						dx, dy = -step, 0
					case sim.DirDown:
						dx, dy = 0, -step*groundK
					case sim.DirUp:
						dx, dy = 0, step*groundK
					}
					if math.Abs(b.ax-a.ax-dx) > 0.01 || math.Abs(by-ay-dy) > 0.01 {
						t.Errorf("%s facing %d, frame %d, leg %d: the foot moved (%.2f, %.2f), the ground (%.2f, %.2f)",
							name, d, f, i, b.ax-a.ax, by-ay, dx, dy)
					}
					// And the leg reaches it: the knee is a real joint.
					if th := math.Hypot(a.kx-a.hx, a.ky-a.hy); d == sim.DirRight && math.Abs(th-thighLen) > 0.01 {
						t.Errorf("%s frame %d leg %d: thigh %.2f long", name, f, i, th)
					}
					if sh := math.Hypot(a.ax-a.kx, a.ay-a.ky); d == sim.DirRight && math.Abs(sh-shinLen) > 0.01 {
						t.Errorf("%s frame %d leg %d: shin %.2f long", name, f, i, sh)
					}
				}
			}
			if planted == 0 {
				t.Errorf("%s facing %d: no foot is ever down for two frames", name, d)
			}
		}
		// Drawn, a planted boot's sole is on the ground row.
		for f := 0; f < loFrames; f++ {
			k := loKey{sim.DirRight, anim, uint8(f)}
			c, p := bakeLower(sim.Look{}, k)
			flat := false
			for i := 0; i < 2; i++ {
				if l := p.leg[i]; l.u == 0 && l.pitch == 0 {
					flat = true
				}
			}
			if !flat {
				continue
			}
			low := 0
			for y := 0; y < c.h; y++ {
				for x := 0; x < c.w; x++ {
					if c.px[y*c.w+x].A >= 128 {
						low = y
					}
				}
			}
			if low < figFeet-1 || low > figFeet {
				t.Errorf("%s frame %d: the lowest pixel is on row %d, the ground is %d", name, f, low, figFeet)
			}
		}
	}
}

// TestLimbFramesDiffer checks that the knees and elbows really bend through
// a stride and a blow.
func TestLimbFramesDiffer(t *testing.T) {
	bend := func(g legGeom) float64 {
		return math.Abs((g.kx-g.hx)*(g.ay-g.hy) - (g.ky-g.hy)*(g.ax-g.hx))
	}
	lo, hi := math.Inf(1), 0.0
	for f := 0; f < loFrames; f++ {
		b := bend(legGeo(sim.DirRight, 0, lowerPoseOf(loKey{sim.DirRight, loRun, uint8(f)})))
		lo, hi = math.Min(lo, b), math.Max(hi, b)
	}
	if hi < lo+40 {
		t.Errorf("the knee hardly bends through a run: %.1f to %.1f", lo, hi)
	}
	a := upperPoseOf(upKey{sim.DirRight, upRun, 0}).arm
	if (a[0].sh-0.15)*(a[1].sh-0.15) >= 0 {
		t.Errorf("the arms swing together: %.2f and %.2f", a[0].sh, a[1].sh)
	}
	for _, an := range []upAnim{upPunch, upChop, upThrust} {
		w := armGeo(false, sim.DirRight, 0, upperPoseOf(upKey{sim.DirRight, an, 0}).arm[0])
		s := armGeo(false, sim.DirRight, 0, upperPoseOf(upKey{sim.DirRight, an, atkFrames - 1}).arm[0])
		if s.hx-w.hx < 8 {
			t.Errorf("blow %d: the hand goes from %.1f to %.1f", an, w.hx, s.hx)
		}
	}
}

func limbTint(q color.RGBA) color.RGBA {
	tint := playerColors[1]
	return color.RGBA{uint8(int(q.R) * int(tint.R) / 255), uint8(int(q.G) * int(tint.G) / 255), uint8(int(q.B) * int(tint.B) / 255), 255}
}

// limbShot puts one frame of the jointed figure together on img, its feet
// at (fx, fy), each image pixel k across: what drawFig does, without a
// screen. A length draws a tool as a line that long from the hand, at
// angle tool.
func limbShot(img *image.RGBA, l sim.Look, up upKey, lo loKey, heads headCache, fx, fy, k int, tool, length float64) {
	mir := up.dir == sim.DirLeft
	if mir {
		up.dir = sim.DirRight
	}
	lo.dir = up.dir
	uc := bakeUpper(l, up, heads)
	lc, lp := bakeLower(l, lo)
	skew := 0.0
	if up.dir == sim.DirRight {
		skew = math.Tan(lp.lean)
	}
	const hip = figFeet - hipHeight*HDDensity
	set := func(x, y float64, c color.RGBA) {
		if mir {
			x = float64(limbAX*HDDensity)*2 - 1 - x
		}
		px, py := fx+(int(math.Round(x))-limbAX*HDDensity)*k, fy+(int(math.Round(y))-figFeet)*k
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
					xx += skew * float64(hip-y)
					yy += lp.drop
				}
				set(xx, yy, q)
			}
		}
	}
	h := uc.arm[0]
	stick := func() {
		if length == 0 {
			return
		}
		hx := h.hx + limbPad*HDDensity + skew*(hip-h.hy)
		hy := h.hy + lp.drop
		for r := -4.0; r < length; r += 0.5 {
			c := color.RGBA{120, 84, 50, 255}
			if r > length-9 {
				c = color.RGBA{214, 220, 228, 255}
			}
			for w := -1.0; w <= 1; w++ {
				set(hx+math.Sin(tool)*r+math.Cos(tool)*w, hy-math.Cos(tool)*r+math.Sin(tool)*w, c)
			}
		}
	}
	over := h.layer == armOver
	if !over && up.dir == sim.DirUp {
		stick()
	}
	put(uc.layer[upFar], true, true)
	put(uc.layer[upBack], false, true)
	put(lc, false, false)
	put(uc.layer[upShirt], true, true)
	put(uc.layer[upArms], true, true)
	put(uc.layer[upTop], false, true)
	if !over && up.dir != sim.DirUp {
		stick()
	}
	put(uc.layer[upArmsOver], true, true)
	if over {
		stick()
	}
	put(uc.layer[upTopOver], false, true)
}

func limbBackdrop(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 96, 136, 78, 255
	}
	return img
}

// limbGIF writes frames as a GIF, delay hundredths of a second apiece.
func limbGIF(t *testing.T, path string, frames []*image.RGBA, delay int) {
	count := map[color.RGBA]int{}
	for _, f := range frames {
		for i := 0; i < len(f.Pix); i += 4 {
			count[color.RGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}]++
		}
	}
	cols := make([]color.RGBA, 0, len(count))
	for c := range count {
		cols = append(cols, c)
	}
	sort.Slice(cols, func(i, j int) bool {
		if count[cols[i]] != count[cols[j]] {
			return count[cols[i]] > count[cols[j]]
		}
		a, b := cols[i], cols[j]
		return int(a.R)<<16|int(a.G)<<8|int(a.B) < int(b.R)<<16|int(b.G)<<8|int(b.B)
	})
	var pal color.Palette
	for _, c := range cols[:min(len(cols), 256)] {
		pal = append(pal, c)
	}
	idx := map[color.RGBA]uint8{}
	g := &gif.GIF{}
	for _, f := range frames {
		p := image.NewPaletted(f.Rect, pal)
		for i := 0; i < len(f.Pix); i += 4 {
			c := color.RGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}
			v, ok := idx[c]
			if !ok {
				v = uint8(pal.Index(c))
				idx[c] = v
			}
			p.Pix[i/4] = v
		}
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, delay)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := gif.EncodeAll(out, g); err != nil {
		t.Fatal(err)
	}
}

func limbPNG(t *testing.T, path string, img image.Image) {
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	png.Encode(out, img)
}

// TestDumpLimbAnim writes the jointed figure's strides and blows as sheets
// and GIFs, a man and a woman in every facing, and what they cost:
//
//	LIMB_ANIM=dir go test ./internal/client -run DumpLimbAnim -v
func TestDumpLimbAnim(t *testing.T) {
	dir := os.Getenv("LIMB_ANIM")
	if dir == "" {
		t.Skip("set LIMB_ANIM to a folder")
	}
	os.MkdirAll(dir, 0o755)
	looks := []sim.Look{
		{Hair: sim.HairShort, HairColor: 1},
		{Female: true, Hair: sim.HairPonytail, HairColor: 3, Skin: 1},
	}
	dirs := []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp, sim.DirLeft}
	const k, cell, foot = 2, 104 * 2, 104*2 - 34
	heads := []headCache{{}, {}}
	// row draws both looks in every facing along row y of img.
	row := func(img *image.RGBA, y int, up func(d sim.Dir) upKey, lo func(d sim.Dir) loKey, tool func(d sim.Dir) float64, length float64) {
		for li, l := range looks {
			for di, d := range dirs {
				a := 0.0
				if tool != nil {
					a = tool(d)
				}
				limbShot(img, l, up(d), lo(d), heads[li], (li*4+di)*cell+cell/2, y*cell+foot, k, a, length)
			}
		}
	}
	stand := func(d sim.Dir) loKey { return loKey{dir: d} }
	for name, an := range map[string][2]uint8{"walk": {uint8(loWalk), uint8(upWalk)}, "run": {uint8(loRun), uint8(upRun)}} {
		var frames []*image.RGBA
		for f := 0; f < loFrames; f++ {
			img := limbBackdrop(8*cell, cell)
			row(img, 0, func(d sim.Dir) upKey { return upKey{d, upAnim(an[1]), uint8(f * upFrames / loFrames)} },
				func(d sim.Dir) loKey { return loKey{d, loAnim(an[0]), uint8(f)} }, nil, 0)
			frames = append(frames, img)
		}
		// As long as a stride takes in the game: a frame is a sixtieth.
		gd, speed := stepGait, 0.75
		if name == "run" {
			gd, speed = runGait, 1.6
		}
		limbGIF(t, filepath.Join(dir, "limbs_"+name+".gif"), frames, max(2, int(math.Round(gd.cycle/speed/loFrames*100/60))))
		strip := limbBackdrop(loFrames*cell, 2*cell)
		for f := 0; f < loFrames; f++ {
			for li, l := range looks {
				limbShot(strip, l, upKey{sim.DirRight, upAnim(an[1]), uint8(f * upFrames / loFrames)}, loKey{sim.DirRight, loAnim(an[0]), uint8(f)},
					heads[li], f*cell+cell/2, li*cell+foot, k, 0, 0)
			}
		}
		limbPNG(t, filepath.Join(dir, "limbs_"+name+"_side.png"), strip)
	}

	// The blows, on the clock the game runs them by: a punch, a sword, an
	// axe and a spear.
	side := func(d sim.Dir, a float64) float64 {
		if d == sim.DirLeft {
			return -a
		}
		return a
	}
	var blows []*image.RGBA
	for f := 0; f < 26; f++ {
		age := float64(f)
		img := limbBackdrop(8*cell, 4*cell)
		punch := upKey{anim: upPunch, frame: 3}
		switch {
		case f < punchWind:
			punch.frame = 0
		case f >= punchHit && f < punchHold:
			punch.frame = atkFrames - 1
		case f >= punchEnd:
			punch = upKey{}
		}
		row(img, 0, func(d sim.Dir) upKey { return upKey{d, punch.anim, punch.frame} }, stand, nil, 0)
		sa := slashAngle(min(1, age/slashFrames))
		row(img, 1, func(d sim.Dir) upKey { return upKey{d, upChop, atkFrame(sa, slashFrom, slashTo)} }, stand,
			func(d sim.Dir) float64 { return side(d, sa) }, 40)
		ca := chopAngle(age)
		row(img, 2, func(d sim.Dir) upKey { return upKey{d, upChop, atkFrame(ca, chopRaised, chopImpact)} }, stand,
			func(d sim.Dir) float64 { return side(d, ca) }, 30)
		out := reach(age)
		th := uint8(math.Round(lerpf(2, atkFrames-1, out)))
		if f < 2 {
			th = 0
		}
		row(img, 3, func(d sim.Dir) upKey { return upKey{d, upThrust, th} }, stand, func(d sim.Dir) float64 {
			if d == sim.DirDown || d == sim.DirUp {
				return 0.15
			}
			return side(d, 1.25+0.3*out)
		}, 46)
		blows = append(blows, img)
	}
	limbGIF(t, filepath.Join(dir, "limbs_attacks.gif"), blows, 7)
	strip := limbBackdrop(atkFrames*cell, 6*cell)
	for i, an := range []upAnim{upPunch, upChop, upThrust} {
		for j, d := range []sim.Dir{sim.DirRight, sim.DirDown} {
			for f := 0; f < atkFrames; f++ {
				limbShot(strip, looks[j], upKey{d, an, uint8(f)}, loKey{dir: d}, heads[j], f*cell+cell/2, (2*i+j)*cell+foot, k, 0, 0)
			}
		}
	}
	limbPNG(t, filepath.Join(dir, "limbs_attack_poses.png"), strip)

	// The stiff figure's eight frames over the jointed one's: sideways (the
	// stiff walk, the jointed walk, the jointed run) and from the front.
	cmp := limbBackdrop(8*cell, 10*cell)
	stiff := func(l sim.Look, d sim.Dir, f, fx, fy int) {
		B, L, S, T := playerLayersHD(l, d, f, false, armNone)
		bob := int(math.Round(walkBob(f) * HDDensity))
		for i, c := range []*canvas{B, L, S, T} {
			for py := 0; py < c.h; py++ {
				for px := 0; px < c.w; px++ {
					q := c.px[py*c.w+px]
					if q.A < 128 {
						continue
					}
					if i == 2 {
						q = limbTint(q)
					}
					oy := -bob
					if i == 1 {
						oy = 0
					}
					for a := 0; a < k*k; a++ {
						if at := image.Pt(fx+(px-27)*k+a%k, fy+(py+oy-figFeet)*k+a/k); at.In(cmp.Rect) {
							cmp.SetRGBA(at.X, at.Y, q)
						}
					}
				}
			}
		}
	}
	for li, l := range looks {
		for f := 0; f < walkFrames; f++ {
			fx, y := f*cell+cell/2, li*5
			stiff(l, sim.DirRight, f, fx, y*cell+foot)
			limbShot(cmp, l, upKey{sim.DirRight, upWalk, uint8(f)}, loKey{sim.DirRight, loWalk, uint8(2 * f)}, heads[li], fx, (y+1)*cell+foot, k, 0, 0)
			limbShot(cmp, l, upKey{sim.DirRight, upRun, uint8(f)}, loKey{sim.DirRight, loRun, uint8(2 * f)}, heads[li], fx, (y+2)*cell+foot, k, 0, 0)
			stiff(l, sim.DirDown, f, fx, (y+3)*cell+foot)
			limbShot(cmp, l, upKey{sim.DirDown, upRun, uint8(f)}, loKey{sim.DirDown, loRun, uint8(2 * f)}, heads[li], fx, (y+4)*cell+foot, k, 0, 0)
		}
	}
	limbPNG(t, filepath.Join(dir, "limbs_compare.png"), cmp)

	// What it costs: every frame of one look.
	area := func(c *canvas) int {
		x0, y0, x1, y1 := c.w, c.h, -1, -1
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				if c.px[y*c.w+x].A > 0 {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
				}
			}
		}
		if x1 < 0 {
			return 0
		}
		return 4 * (x1 - x0 + 3) * (y1 - y0 + 3)
	}
	start := time.Now()
	n, bytes := 0, 0
	hc := headCache{}
	for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
		for _, an := range []struct {
			a upAnim
			n int
		}{{upStand, 1}, {upWalk, upFrames}, {upRun, upFrames}, {upPunch, atkFrames}, {upPunchOff, atkFrames}, {upChop, atkFrames}, {upThrust, atkFrames}, {upFling, 1}, {upSwim, 4}} {
			for f := 0; f < an.n; f++ {
				u := bakeUpper(looks[0], upKey{d, an.a, uint8(f)}, hc)
				for _, c := range u.layer {
					bytes += area(c)
				}
				n++
			}
		}
		for _, an := range []struct {
			a loAnim
			n int
		}{{loStand, 1}, {loWalk, loFrames}, {loRun, loFrames}, {loSit, 1}} {
			for f := 0; f < an.n; f++ {
				c, _ := bakeLower(looks[0], loKey{d, an.a, uint8(f)})
				bytes += area(c)
				n++
			}
		}
	}
	took := time.Since(start)
	start = time.Now()
	bakeUpper(looks[1], upKey{dir: sim.DirRight}, headCache{})
	bakeLower(looks[1], loKey{dir: sim.DirRight})
	first := time.Since(start)
	start = time.Now()
	for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
		for f := 0; f < walkFrames; f++ {
			playerLayersHD(looks[0], d, f, false, armNone)
		}
	}
	old := time.Since(start)
	fmt.Printf("jointed figure: %d half-frames of one look in %v (%.2f ms each), %.2f MB with the mirrored ones; the first frame of a look %v; the stiff figure's 24 frames, all drawn when a look is first seen: %v\n",
		n, took, float64(took.Microseconds())/1000/float64(n), float64(bytes)*4/3/1e6, first, old)
}
