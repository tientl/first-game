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

// ankyloFrame is one drawn frame of the armoured one, and what it is.
type ankyloFrame struct {
	name   string
	c      *canvas
	ax, ay float64
}

// ankyloFrames draws every frame the game shows of the Ankylosaurus, facing
// right: standing, its guard swaying, hit, eating, the walk (forwards and
// backing up), and its blow frame by frame (the wild one's and a pet's).
func ankyloFrames(t *testing.T) (still, walk, blow, quick []ankyloFrame) {
	t.Helper()
	r := dinoRigs["ankylo"]
	draw := func(name string, p rigPose) ankyloFrame {
		c, ax, ay, ok := rigCanvas(r, p)
		if !ok {
			t.Fatalf("%s: the drawing is missing", name)
		}
		return ankyloFrame{name, c, ax, ay}
	}
	poses := dinoPoses["ankylo"]
	still = append(still, draw("stand", rigPose{}))
	for f := 0; f < 8*9; f += 18 {
		still = append(still, draw("guard", ankyloGuardPose(f)))
	}
	still = append(still, draw("hurt", poses[ankyloHurt]), draw("eatA", poses[ankyloEatA]), draw("eatB", poses[ankyloEatB]),
		draw("wind", ankyloWind), draw("swing", poses[ankyloSwing]))
	for f := range gaitFrames {
		walk = append(walk, draw("walk", stridePose(rigPose{}, f, 1)))
	}
	for f := range gaitFrames {
		walk = append(walk, draw("back", stridePose(poses[ankyloGuard], gaitFrames-1-f, 1)))
	}
	frame := func(age int, q bool) ankyloFrame {
		p, smear := ankyloBlowFrame(age, q)
		if smear < 0 {
			return draw("blow", p)
		}
		c, ax, ay, ok := ankyloSmearCanvas(smear)
		if !ok {
			t.Fatal("no swing drawn")
		}
		return ankyloFrame{"swing", c, ax, ay}
	}
	for age := 0; age < 64; age++ {
		blow = append(blow, frame(age, false))
	}
	for age := 0; age < 34; age++ {
		quick = append(quick, frame(age, true))
	}
	return still, walk, blow, quick
}

// lowest is the lowest row of the drawing left of column x, from its anchor.
func lowest(f ankyloFrame, x float64) float64 {
	for y := f.c.h - 1; y >= 0; y-- {
		for px := 0; px < f.c.w && float64(px) < f.ax+x; px++ {
			if f.c.px[y*f.c.w+px].A >= 128 {
				return float64(y) - f.ay
			}
		}
	}
	return math.Inf(-1)
}

func TestTheAnkylosaurIsDrawnInEveryPose(t *testing.T) {
	still, walk, blow, quick := ankyloFrames(t)
	for _, f := range append(append(append(still, walk...), blow...), quick...) {
		if f.ax < 0 || f.ay < 0 || f.ax > float64(f.c.w) || f.ay > float64(f.c.h)+3 {
			t.Errorf("%s: anchor (%.0f, %.0f) off its %dx%d canvas", f.name, f.ax, f.ay, f.c.w, f.c.h)
		}
		n := 0
		for _, q := range f.c.px {
			if q.A >= 128 {
				n++
			}
		}
		if n < 9000 {
			t.Errorf("%s: only %d pixels drawn", f.name, n)
		}
	}
	// Swung round towards the viewer, the club comes down the picture, in
	// front of the hind legs; drawn back, it goes up and behind.
	k := float64(HDDensity)
	stand, swing, wind := still[0], still[len(still)-1], still[len(still)-2]
	if lowest(swing, -40*k) < lowest(stand, -40*k)+6*k {
		t.Errorf("the swung club is at %.0f, the standing one at %.0f: it should come down towards the viewer",
			lowest(swing, -40*k)/k, lowest(stand, -40*k)/k)
	}
	if lowest(wind, -40*k) > lowest(stand, -40*k) {
		t.Error("drawn back, the club should go up and away")
	}
	// The blow swings round as the sim lands it.
	land := sim.WindupTicks(sim.CreatureAnkylo) * 60 / sim.TicksPerSecond
	if _, s := ankyloBlowFrame(land, false); s < 0 {
		t.Errorf("frame %d, when the sim lands the blow, is not the swing", land)
	}
}

// TestDumpAnkylo writes, to the folder $ANKYLO_DUMP: dino_ankylo_compare.png
// (the guide and the painting at the same size, then the head, the armour
// and the club close up), dino_ankylo_poses.png (every pose at its size at
// 1080p) and dino_ankylo_attack.gif (the blow, twice that size).
func TestDumpAnkylo(t *testing.T) {
	dir := os.Getenv("ANKYLO_DUMP")
	if dir == "" {
		t.Skip("set ANKYLO_DUMP to a folder")
	}
	k := HDDensity
	low, hd := dinoImage("ankylo", false), dinoImage("ankylo", true)
	if low == nil || hd == nil {
		t.Fatal("drawings missing")
	}
	backdrop := func(img *image.NRGBA, c color.NRGBA) {
		for j := 0; j < len(img.Pix); j += 4 {
			img.Pix[j], img.Pix[j+1], img.Pix[j+2], img.Pix[j+3] = c.R, c.G, c.B, 255
		}
	}
	put := func(img *image.NRGBA, c *canvas, x0, y0, sx, sy, sw, sh, z int) {
		for y := 0; y < sh*z; y++ {
			for x := 0; x < sw*z; x++ {
				gx, gy := sx+x/z, sy+y/z
				if gx < 0 || gy < 0 || gx >= c.w || gy >= c.h {
					continue
				}
				if q := c.px[gy*c.w+gx]; q.A >= 128 && image.Pt(x0+x, y0+y).In(img.Rect) {
					img.SetNRGBA(x0+x, y0+y, color.NRGBA{q.R, q.G, q.B, 255})
				}
			}
		}
	}
	// The comparison: guide (three times as big, pixel for pixel) and painting
	// side by side at twice their size, then close-ups.
	const z = 2
	pad := 24
	W := 2*(hd.w*z) + 3*pad
	closeZ := 9
	head := [4]int{80 * k, 20 * k, 25 * k, 26 * k}
	armour := [4]int{38 * k, 2 * k, 30 * k, 26 * k}
	club := [4]int{0, 18 * k, 30 * k, 24 * k}
	rowH := max(head[3], armour[3], club[3]) * closeZ / 3
	H := hd.h*z + rowH + 3*pad
	cmp := image.NewNRGBA(image.Rect(0, 0, max(W, 3*30*k*closeZ/3+4*pad), H))
	backdrop(cmp, color.NRGBA{58, 64, 60, 255})
	put(cmp, low, pad, pad, 0, 0, low.w, low.h, k*z)
	put(cmp, hd, 2*pad+hd.w*z, pad, 0, 0, hd.w, hd.h, z)
	x := pad
	for _, r := range [][4]int{head, armour, club} {
		zz := closeZ / 3
		put(cmp, hd, x, 2*pad+hd.h*z, r[0], r[1], r[2], r[3], zz)
		x += r[2]*zz + pad
	}
	reviewPNG(t, filepath.Join(dir, "dino_ankylo_compare.png"), cmp)

	still, walk, blow, quick := ankyloFrames(t)
	_ = quick
	// The poses: every frame at its 1080p size, on grass, anchors in line.
	cw, ch := (low.w+44)*k, (low.h+34)*k
	death := deathOf(still[0])
	rows := [][]ankyloFrame{
		still[:5],
		append(append([]ankyloFrame{}, still[5:]...), death),
		walk[:8], walk[8:],
		pickFrames(blow, 0, 6, 20, 31, 32, 33, 34),
		pickFrames(blow, 35, 40, 50, 53, 56, 59),
	}
	perRow := 0
	for _, r := range rows {
		perRow = max(perRow, len(r))
	}
	sheet := image.NewNRGBA(image.Rect(0, 0, cw*perRow, ch*len(rows)))
	reviewGrass(sheet)
	for ri, row := range rows {
		for i, f := range row {
			blendPut(sheet, f.c, i*cw+cw/2-int(f.ax), ri*ch+ch-8*k-int(f.ay), 1)
		}
	}
	reviewPNG(t, filepath.Join(dir, "dino_ankylo_poses.png"), sheet)

	// The blow, twice the size: its guard a moment, the blow, its guard again.
	var frames []*image.NRGBA
	gw, gh := (low.w+40)*k, (low.h+30)*k
	add := func(f ankyloFrame) {
		img := image.NewNRGBA(image.Rect(0, 0, gw, gh))
		reviewGrass(img)
		blendPut(img, f.c, gw/2+12*k-int(f.ax), gh-12*k-int(f.ay), 1)
		frames = append(frames, img)
	}
	for i := range 6 {
		add(still[1+i%4])
	}
	for age := 0; age < len(blow); age += 2 {
		add(blow[age])
	}
	for i := range 8 {
		add(still[1+i%4])
	}
	reviewGIF(t, filepath.Join(dir, "dino_ankylo_attack.gif"), frames, 3)
}

func pickFrames(fs []ankyloFrame, at ...int) []ankyloFrame {
	var out []ankyloFrame
	for _, i := range at {
		out = append(out, fs[i])
	}
	return out
}

// deathOf is a frame laid down as a body is drawn (drawBody): on its back,
// squashed, the colour gone out of it.
func deathOf(f ankyloFrame) ankyloFrame {
	const squash = 0.85
	h := int(math.Ceil(float64(f.c.h) * squash))
	out := newCanvas(f.c.w, h)
	for y := 0; y < h; y++ {
		sy := f.c.h - 1 - int(float64(y)/squash)
		for x := 0; x < f.c.w; x++ {
			if q := f.c.px[sy*f.c.w+x]; q.A >= 128 {
				out.px[y*out.w+x] = color.RGBA{uint8(float64(q.R) * 0.78), uint8(float64(q.G) * 0.72), uint8(float64(q.B) * 0.72), 255}
			}
		}
	}
	return ankyloFrame{"death", out, f.ax, float64(h) - 3*HDDensity}
}

// blendPut draws canvas c on img with its top left at (x, y), blending the
// faint pixels (a blurred club) over what is there, as the game draws them.
func blendPut(img *image.NRGBA, c *canvas, x, y, k int) {
	for py := 0; py < c.h*k; py++ {
		for px := 0; px < c.w*k; px++ {
			q := c.px[(py/k)*c.w+px/k]
			if q.A == 0 || !image.Pt(x+px, y+py).In(img.Rect) {
				continue
			}
			o := img.NRGBAAt(x+px, y+py)
			a := float64(q.A) / 255
			m := func(s, d uint8) uint8 { return uint8(float64(s)*a + float64(d)*(1-a) + 0.5) }
			img.SetNRGBA(x+px, y+py, color.NRGBA{m(q.R, o.R), m(q.G, o.G), m(q.B, o.B), 255})
		}
	}
}
