package client

import (
	"image"
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
		u := bakeUpper(sim.Look{Female: female, Hair: sim.HairPonytail}, upKey{dir: sim.DirDown}, headCache{})
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
