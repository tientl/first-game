package client

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"survivalgame/internal/sim"
)

// TestFallenEveryLook draws the body and the bones of every look: both must
// have something of the player's colour on them, and the bag.
func TestFallenEveryLook(t *testing.T) {
	bag := genLootBag()
	if bag.w+fallenOX+19 > fallenW || bag.h+fallenOY+8 > fallenH {
		t.Fatalf("the bag does not fit the %dx%d frame", fallenW, fallenH)
	}
	tint := playerColors[1]
	for _, female := range []bool{false, true} {
		for h := range sim.NumHairStyles {
			for hc := range sim.NumHairColors {
				for s := range sim.NumSkinTones {
					l := sim.Look{Female: female, Hair: sim.HairStyle(h), HairColor: uint8(hc), Skin: uint8(s)}
					for _, bones := range []bool{false, true} {
						c := genFallen(l, tint, bones)
						blue, edge := 0, 0
						for y := 0; y < c.h; y++ {
							for x := 0; x < c.w; x++ {
								p := c.at(x, y)
								if p.A == 0 {
									continue
								}
								if int(p.B) > int(p.R)+40 && p.B > 120 { // playerColors[1] is blue; so is hair colour 6
									blue++
								}
								if x == 0 || y == 0 || x == c.w-1 || y == c.h-1 {
									edge++
								}
							}
						}
						if blue == 0 && hc != 6 {
							t.Errorf("%+v bones=%v: nothing in the player's colour", l, bones)
						}
						if edge > 0 {
							t.Errorf("%+v bones=%v: %d pixels on the frame's edge, cut off", l, bones, edge)
						}
					}
				}
			}
		}
	}
}

// TestFallenSheet writes a sheet of bodies and bones to the file named by
// BODY_SHEET, enlarged, to look at.
func TestFallenSheet(t *testing.T) {
	path := os.Getenv("BODY_SHEET")
	if path == "" {
		t.Skip("BODY_SHEET not set")
	}
	looks := []struct {
		l   sim.Look
		col int
	}{
		{sim.Look{}, 0},
		{sim.Look{Female: true, Hair: sim.HairLong, HairColor: 3, Skin: 0}, 1},
		{sim.Look{Hair: sim.HairSpiky, HairColor: 4, Skin: 2}, 3},
		{sim.Look{Female: true, Hair: sim.HairPonytail, HairColor: 1, Skin: 4}, 2},
		{sim.Look{Hair: sim.HairBun, HairColor: 5, Skin: 5}, 4},
		{sim.Look{Female: true, Hair: sim.HairBob, HairColor: 7, Skin: 1}, 5},
		{sim.Look{Hair: sim.HairLong, HairColor: 2, Skin: 3}, 1},
		{sim.Look{Female: true, Hair: sim.HairBun, HairColor: 6, Skin: 2}, 0},
		{sim.Look{Hair: sim.HairPonytail, HairColor: 3, Skin: 1}, 5},
		{sim.Look{Female: true, Hair: sim.HairSpiky, HairColor: 0, Skin: 5}, 3},
	}
	const k, pad, cols = 6, 4, 2
	cw, ch := (fallenW+pad)*k, (fallenH+pad)*k
	grounds := []color.RGBA{hex(0x5f9a4a), hex(0xd9c48a), hex(0xe9eef2), hex(0x6b6f76)}
	img := image.NewNRGBA(image.Rect(0, 0, cw*cols*2, ch*len(looks)/cols))
	for i, lk := range looks {
		for b := range 2 {
			c := genFallen(lk.l, playerColors[lk.col], b == 1)
			ox, oy := (i%cols*2+b)*cw, i/cols*ch
			bg := grounds[(i/cols)%len(grounds)]
			for y := range ch {
				for x := range cw {
					col := bg
					if p := c.at(x/k-pad/2, y/k-pad/2); p.A >= 128 {
						col = p
					}
					img.SetNRGBA(ox+x, oy+y, color.NRGBA{col.R, col.G, col.B, 255})
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
