package client

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"survivalgame/internal/sim"
)

// lookCell draws one direction and walk frame of look l, as the game layers
// it, on img with its top-left at (ox, oy), each art pixel k wide.
func lookCell(img *image.RGBA, l sim.Look, d sim.Dir, frame int, tint color.RGBA, ox, oy, k int) {
	flip := d == sim.DirLeft
	if flip {
		d = sim.DirRight
	}
	B, L, S, T := playerLayers(l, d, frame)
	bob := 0
	if frame%2 == 1 {
		bob = 1
	}
	for i, c := range []*canvas{B, L, S, T} {
		dy := -bob
		if i == 1 {
			dy = 0
		}
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				p := c.at(x, y)
				if p.A < 128 {
					continue
				}
				if i == 2 {
					p.R = uint8(int(p.R) * int(tint.R) / 255)
					p.G = uint8(int(p.G) * int(tint.G) / 255)
					p.B = uint8(int(p.B) * int(tint.B) / 255)
				}
				px := x
				if flip {
					px = c.w - 1 - x
				}
				for a := range k {
					for b := range k {
						img.SetRGBA(ox+px*k+a, oy+(y+dy+1)*k+b, p)
					}
				}
			}
		}
	}
}

func lookGround(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, hex(0x6fa552))
		}
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// TestLookSheet writes every direction and walk frame of both sexes and all
// hair styles, then every hair colour on three skin tones, to the file named
// by LOOK_SHEET, enlarged, to look at.
func TestLookSheet(t *testing.T) {
	path := os.Getenv("LOOK_SHEET")
	if path == "" {
		t.Skip("LOOK_SHEET not set")
	}
	const k, cw, ch = 4, bodyW + 3, bodyH + 3
	dirs := []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp}
	rows := 2*int(sim.NumHairStyles) + 6
	img := image.NewRGBA(image.Rect(0, 0, 16*cw*k, rows*ch*k))
	lookGround(img)
	row := 0
	for _, female := range []bool{false, true} {
		for h := range sim.NumHairStyles {
			l := sim.Look{Female: female, Hair: sim.HairStyle(h), HairColor: uint8((h * 3) % sim.NumHairColors), Skin: uint8(h % sim.NumSkinTones)}
			tint := playerColors[row%len(playerColors)]
			col := 0
			for _, d := range dirs {
				for f := range walkFrames {
					lookCell(img, l, d, f, tint, col*cw*k+k, row*ch*k, k)
					col++
				}
			}
			lookCell(img, l, sim.DirLeft, 0, tint, col*cw*k+k, row*ch*k, k)
			row++
		}
	}
	for _, skin := range []uint8{0, 3, 5} {
		for _, female := range []bool{false, true} {
			for hc := range sim.NumHairColors {
				l := sim.Look{Female: female, Hair: sim.HairStyle((hc + int(skin)) % int(sim.NumHairStyles)), HairColor: uint8(hc), Skin: skin}
				if !female {
					l.Hair = sim.HairShort
				}
				tint := playerColors[hc%len(playerColors)]
				lookCell(img, l, sim.DirDown, 0, tint, 2*hc*cw*k+k, row*ch*k, k)
				lookCell(img, l, sim.DirRight, 0, tint, (2*hc+1)*cw*k+k, row*ch*k, k)
			}
			row++
		}
	}
	writePNG(t, path, img)
}

// TestLookCompare writes a few looks from the front, the side and the back
// to the file named by LOOK_CMP, enlarged and at 1x.
func TestLookCompare(t *testing.T) {
	path := os.Getenv("LOOK_CMP")
	if path == "" {
		t.Skip("LOOK_CMP not set")
	}
	looks := []sim.Look{
		{},
		{Hair: sim.HairShort, HairColor: 1, Skin: 5},
		{Female: true, Hair: sim.HairPonytail, HairColor: 3, Skin: 0},
		{Female: true, Hair: sim.HairBun, HairColor: 4, Skin: 1},
	}
	const k, cw, ch = 6, bodyW + 3, bodyH + 3
	img := image.NewRGBA(image.Rect(0, 0, 3*cw*k+cw+6, len(looks)*ch*k))
	lookGround(img)
	for i, l := range looks {
		tint := playerColors[i%len(playerColors)]
		for j, d := range []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp} {
			lookCell(img, l, d, 0, tint, j*cw*k+k, i*ch*k, k)
		}
		lookCell(img, l, sim.DirDown, 0, tint, 3*cw*k+2, i*ch*k+4, 1)
		lookCell(img, l, sim.DirRight, 0, tint, 3*cw*k+2, i*ch*k+4+ch+4, 1)
	}
	writePNG(t, path, img)
}
