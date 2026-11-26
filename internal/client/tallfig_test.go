package client

import (
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"survivalgame/internal/sim"
)

// tallOver lays canvas c over img, its anchor at (fx, fy), each image pixel
// k across.
func tallOver(img *image.RGBA, c *canvas, fx, fy, k int) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			q := c.px[y*c.w+x]
			if q.A < 128 {
				continue
			}
			for a := 0; a < k*k; a++ {
				img.SetRGBA(fx+(x-limbAX*HDDensity)*k+a%k, fy+(y-limbGround)*k+a/k, q)
			}
		}
	}
}

// TestDumpTallSheet writes every hair style of both figures in three
// facings, and under them every armour:
//
//	TALL_SHEET=file.png go test ./internal/client -run DumpTallSheet
func TestDumpTallSheet(t *testing.T) {
	out := os.Getenv("TALL_SHEET")
	if out == "" {
		t.Skip("set TALL_SHEET to a file")
	}
	const k, cw, ch = 3, 210, 400
	dirs := []sim.Dir{sim.DirRight, sim.DirDown, sim.DirUp}
	img := limbBackdrop(18*cw, 4*ch)
	for fi := 0; fi < 2; fi++ {
		for h := 0; h < 6; h++ {
			l := sim.Look{Female: fi == 1, Hair: sim.HairStyle(h), HairColor: uint8((h*3 + fi) % 8), Skin: uint8((h + fi) % 6)}
			for di, d := range dirs {
				limbShot(img, l, upKey{dir: d}, loKey{dir: d}, headCache{}, (h*3+di)*cw+cw/2, fi*ch+ch-40, k, 0, 0)
			}
		}
		for ti := sim.TierCopper; ti <= sim.TierDiamond; ti++ {
			l := sim.Look{Female: fi == 1, Hair: sim.HairShort, HairColor: uint8(ti)}
			for di, d := range dirs {
				fx, fy := ((int(ti)-1)*3+di)*cw+cw/2, (2+fi)*ch+ch-40
				limbShot(img, l, upKey{dir: d}, loKey{dir: d}, headCache{}, fx, fy, k, 0, 0)
				B, H := armorLayersTall(ti, l.Female, d)
				tallOver(img, B, fx, fy, k)
				tallOver(img, H, fx, fy, k)
			}
		}
	}
	limbPNG(t, out, img)

	// The two looks of TestDumpLimbAnim standing, bigger: to set beside
	// other drawings of them.
	if stand := os.Getenv("TALL_STAND"); stand != "" {
		const cell = 480
		img := limbBackdrop(6*cell, cell)
		for li, l := range []sim.Look{{Hair: sim.HairShort, HairColor: 1}, {Female: true, Hair: sim.HairPonytail, HairColor: 3, Skin: 1}} {
			for di, d := range dirs {
				limbShot(img, l, upKey{dir: d}, loKey{dir: d}, headCache{}, (li*3+di)*cell+cell/2, cell-30, k, 0, 0)
			}
		}
		limbPNG(t, stand, img)
	}
}

// TestTallFigureProportions checks that the figure is not a chibi: about
// five and a half heads tall, the hands hanging to the middle of the thigh.
func TestTallFigureProportions(t *testing.T) {
	for _, female := range []bool{false, true} {
		u := bakeUpper(sim.Look{Female: female, Hair: sim.HairPonytail}, limbWear, upKey{dir: sim.DirDown}, headCache{})
		top := -1
		c := u.layer[upTop]
		for i, p := range c.px {
			if p.A >= 128 {
				top = i / c.w
				break
			}
		}
		const head = 17.0 // the skull, crown to chin
		if heads := float64(limbGround-top) / head; heads < 5.3 || heads > 6.2 {
			t.Errorf("female %v: the figure is %.1f heads tall", female, heads)
		}
		hand := u.arm[0].hy
		if hand < hipY+3 || hand > hipY+thighLen*0.6 {
			t.Errorf("female %v: the hand hangs to row %.1f, the hips are on %.1f and the knees on %.1f", female, hand, hipY, hipY+thighLen)
		}
	}
}

// TestDumpCave writes the figure as everyone starts, in furs and barefoot:
// cave_base.png (a man and a woman of three skin colours from the side, the
// front and the back), cave_wear.png (each of the clothes put on, and the
// armours over bare skin), cave_poses.png (blows, a start, swimming, riding)
// and cave_walk.gif and cave_run.gif (from the side):
//
//	CAVE_DUMP=dir go test ./internal/client -run DumpCave
func TestDumpCave(t *testing.T) {
	dir := os.Getenv("CAVE_DUMP")
	if dir == "" {
		t.Skip("set CAVE_DUMP to a folder")
	}
	defer func(w sim.Wear) { limbWear = w }(limbWear)
	const k, cw, ch = 4, 200, 440
	dirs := []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp}
	looks := func(female bool) []sim.Look {
		return []sim.Look{
			{Female: female, Hair: sim.HairShaggy, HairColor: 0, Skin: 1},
			{Female: female, Hair: sim.HairShaggy, HairColor: 2, Skin: 0},
			{Female: female, Hair: sim.HairShaggy, HairColor: 1, Skin: 4},
		}
	}
	img := limbBackdrop(9*cw, 2*ch)
	for fi := 0; fi < 2; fi++ {
		for li, l := range looks(fi == 1) {
			for di, d := range dirs {
				limbShot(img, l, upKey{dir: d}, loKey{dir: d}, headCache{}, (li*3+di)*cw+cw/2, fi*ch+ch-36, k, 0, 0)
			}
		}
	}
	limbPNG(t, dir+"/cave_base.png", img)

	// Each of the clothes, then all of them; every hair on bare shoulders.
	wears := []sim.Wear{0, sim.WearTunic, sim.WearMantle, sim.WearLegWraps, sim.WearBoots, sim.WearAll}
	img = limbBackdrop(len(wears)*3*150, 4*330)
	for fi := 0; fi < 2; fi++ {
		l := sim.Look{Female: fi == 1, Hair: sim.HairShaggy, HairColor: uint8(fi * 3), Skin: uint8(fi)}
		for wi, w := range wears {
			limbWear = w
			for di, d := range dirs {
				limbShot(img, l, upKey{dir: d}, loKey{dir: d}, headCache{}, (wi*3+di)*150+75, fi*330+300, 3, 0, 0)
			}
		}
		limbWear = 0
		for ti := sim.TierCopper; ti <= sim.TierDiamond; ti++ {
			for di, d := range dirs {
				fx, fy := ((int(ti)-1)*3+di)*150+75, (2+fi)*330+300
				limbShot(img, l, upKey{dir: d}, loKey{dir: d}, headCache{}, fx, fy, 3, 0, 0)
				B, H := armorLayersTall(ti, l.Female, d)
				tallOver(img, B, fx, fy, 3)
				tallOver(img, H, fx, fy, 3)
			}
		}
		for h := 0; h < 2; h++ {
			hl := sim.Look{Female: fi == 1, Hair: sim.HairStyle([2]int{0, 3}[h] + fi), HairColor: 5, Skin: 3}
			for di, d := range dirs {
				limbShot(img, hl, upKey{dir: d}, loKey{dir: d}, headCache{}, ((4+h)*3+di)*150+75, (2+fi)*330+300, 3, 0, 0)
			}
		}
	}
	limbPNG(t, dir+"/cave_wear.png", img)

	// What the body does besides walking.
	man, woman := looks(false)[0], looks(true)[1]
	type pose struct {
		up upKey
		lo loKey
	}
	var poses []pose
	for _, d := range []sim.Dir{sim.DirRight, sim.DirDown, sim.DirUp} {
		poses = append(poses,
			pose{upKey{d, upPunch, 0}, loKey{dir: d}}, pose{upKey{d, upPunch, atkFrames - 1}, loKey{dir: d}},
			pose{upKey{d, upChop, 0}, loKey{dir: d}}, pose{upKey{d, upChop, atkFrames - 1}, loKey{dir: d}},
			pose{upKey{d, upThrust, 3}, loKey{dir: d}}, pose{upKey{dir: d, anim: upFling}, loKey{dir: d}},
			pose{upKey{d, upSwim, 1}, loKey{dir: d}}, pose{upKey{d, upStand, upSit}, loKey{dir: d, anim: loSit}},
			pose{upKey{d, upRun, 2}, loKey{d, loRun, 4}}, pose{upKey{dir: d, anim: upAsleep}, loKey{dir: d}})
	}
	img = limbBackdrop(10*170, 6*330)
	for i, p := range poses {
		for li, l := range []sim.Look{man, woman} {
			limbShot(img, l, p.up, p.lo, headCache{}, i%10*170+85, (i/10*2+li)*330+300, 3, 0, 0)
		}
	}
	limbPNG(t, dir+"/cave_poses.png", img)

	// The strides from the side, the front and the back, over a line where
	// the figure stands: a planted foot is on it.
	for name, an := range map[string][2]uint8{"walk": {uint8(loWalk), uint8(upWalk)}, "run": {uint8(loRun), uint8(upRun)}} {
		const gk, cell, rowH = 3, 210, 350
		heads := make([]headCache, 6)
		var frames []*image.RGBA
		for f := 0; f < loFrames; f++ {
			fr := limbBackdrop(3*cell, 2*rowH)
			for li, l := range []sim.Look{man, woman} {
				for di, d := range []sim.Dir{sim.DirRight, sim.DirDown, sim.DirUp} {
					fx, fy := di*cell+cell/2, li*rowH+rowH-24
					groundLine(fr, fx, fy, 90)
					limbShot(fr, l, upKey{d, upAnim(an[1]), uint8(f * upFrames / loFrames)}, loKey{d, loAnim(an[0]), uint8(f)},
						heads[li*3+di], fx, fy, gk, 0, 0)
				}
			}
			frames = append(frames, fr)
		}
		gd, speed := stepGait, 0.75
		if name == "run" {
			gd, speed = runGait, 1.6
		}
		limbGIF(t, dir+"/cave_"+name+".gif", frames, max(2, int(math.Round(gd.cycle/speed/loFrames*100/60))))
		strip := limbBackdrop(8*240, 4*330)
		for f := 0; f < 8; f++ {
			for li, l := range []sim.Look{man, woman} {
				limbShot(strip, l, upKey{sim.DirRight, upAnim(an[1]), uint8(f)}, loKey{sim.DirRight, loAnim(an[0]), uint8(2 * f)},
					heads[li*3], f*240+120, li*330+300, 3, 0, 0)
			}
			groundLine(strip, f*240+120, 2*330+300, 90)
			limbShot(strip, man, upKey{sim.DirUp, upAnim(an[1]), uint8(f)}, loKey{sim.DirUp, loAnim(an[0]), uint8(2 * f)},
				heads[2], f*240+120, 2*330+300, 3, 0, 0)
			groundLine(strip, f*240+120, 3*330+300, 90)
			limbShot(strip, man, upKey{sim.DirDown, upAnim(an[1]), uint8(f)}, loKey{sim.DirDown, loAnim(an[0]), uint8(2 * f)},
				heads[1], f*240+120, 3*330+300, 3, 0, 0)
		}
		limbPNG(t, dir+"/cave_"+name+"_strip.png", strip)
	}
}

// groundLine marks the row a figure stands on, w pixels either side of
// (x, y): darker than the grass, under where the soles go.
func groundLine(img *image.RGBA, x, y, w int) {
	for i := x - w; i <= x+w; i++ {
		if image.Pt(i, y).In(img.Rect) {
			img.SetRGBA(i, y, color.RGBA{70, 104, 58, 255})
		}
	}
}
