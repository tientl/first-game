package client

import (
	"image"
	"image/color"
	"os"
	"testing"

	"survivalgame/internal/sim"
)

// hdStack draws canvases over each other on img at (ox, oy), each pixel k
// wide; the layer numbered tintAt takes tint, and layers[i] is shifted up
// by dys[i] pixels.
func hdStack(img *image.RGBA, layers []*canvas, dys []int, tintAt int, tint color.RGBA, ox, oy, k int, flip bool) {
	for i, c := range layers {
		if c == nil {
			continue
		}
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				p := c.at(x, y)
				if p.A < 128 {
					continue
				}
				if i == tintAt {
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
						img.SetRGBA(ox+px*k+a, oy+(y-dys[i])*k+b, p)
					}
				}
			}
		}
	}
}

// hdCell draws one direction and walk frame of look l at HD density, as
// the game layers it, each image pixel k wide.
func hdCell(img *image.RGBA, l sim.Look, d sim.Dir, frame int, tint color.RGBA, ox, oy, k int) {
	flip := d == sim.DirLeft
	if flip {
		d = sim.DirRight
	}
	B, L, S, T := playerLayersHD(l, d, frame, false, armNone)
	bob := int(walkBob(frame) * HDDensity)
	hdStack(img, []*canvas{B, L, S, T}, []int{bob, 0, bob, bob}, 2, tint, ox, oy+3*k, k, flip)
}

// TestLookSheetHD writes every direction and walk frame of both sexes and
// all hair styles at HD density, then every hair colour on three skin
// tones, to the file named by LOOK_SHEET_HD.
func TestLookSheetHD(t *testing.T) {
	path := os.Getenv("LOOK_SHEET_HD")
	if path == "" {
		t.Skip("LOOK_SHEET_HD not set")
	}
	const k, cw, ch = 2, figW + 4, figH + 6
	dirs := []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp}
	rows := 2*int(sim.NumHairStyles) + 6
	img := image.NewRGBA(image.Rect(0, 0, (3*walkFrames+1)*cw*k, rows*ch*k))
	lookGround(img)
	row := 0
	for _, female := range []bool{false, true} {
		for h := range sim.NumHairStyles {
			l := sim.Look{Female: female, Hair: sim.HairStyle(h), HairColor: uint8((h * 3) % sim.NumHairColors), Skin: uint8(h % sim.NumSkinTones)}
			tint := playerColors[row%len(playerColors)]
			col := 0
			for _, d := range dirs {
				for f := range walkFrames {
					hdCell(img, l, d, f, tint, col*cw*k+k, row*ch*k, k)
					col++
				}
			}
			hdCell(img, l, sim.DirLeft, 0, tint, col*cw*k+k, row*ch*k, k)
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
				hdCell(img, l, sim.DirDown, 0, tint, 2*hc*cw*k+k, row*ch*k, k)
				hdCell(img, l, sim.DirRight, 0, tint, (2*hc+1)*cw*k+k, row*ch*k, k)
			}
			row++
		}
	}
	writePNG(t, path, img)
}

// TestLookCompareHD writes a few looks from the front, the side and the
// back, the old art beside the HD art at the same size, to the file named
// by LOOK_CMP_HD.
func TestLookCompareHD(t *testing.T) {
	path := os.Getenv("LOOK_CMP_HD")
	if path == "" {
		t.Skip("LOOK_CMP_HD not set")
	}
	looks := []sim.Look{
		{},
		{Hair: sim.HairSpiky, HairColor: 4, Skin: 3},
		{Female: true, Hair: sim.HairPonytail, HairColor: 3, Skin: 0},
		{Female: true, Hair: sim.HairLong, HairColor: 1, Skin: 1},
	}
	const k, cw, ch = 4, figW + 6, figH + 6 // each HD pixel k wide: an art pixel is 3k
	img := image.NewRGBA(image.Rect(0, 0, 6*cw*k, len(looks)*ch*k))
	lookGround(img)
	for i, l := range looks {
		tint := playerColors[i%len(playerColors)]
		for j, d := range []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp} {
			lookCell(img, l, d, 0, tint, (2*j)*cw*k+3*k, i*ch*k+2*k, 3*k)
			hdCell(img, l, d, 0, tint, (2*j+1)*cw*k+3*k, i*ch*k, k)
		}
	}
	writePNG(t, path, img)
}

// TestArmorSheetHD writes every armour tier on a man and a woman, from the
// front, the side and the back, and the arms of a punch, to the file named
// by ARMOR_SHEET_HD.
func TestArmorSheetHD(t *testing.T) {
	path := os.Getenv("ARMOR_SHEET_HD")
	if path == "" {
		t.Skip("ARMOR_SHEET_HD not set")
	}
	const k, cw, ch = 4, figW + 6, figH + 6
	dirs := []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp}
	img := image.NewRGBA(image.Rect(0, 0, 12*cw*k, 4*ch*k))
	lookGround(img)
	for i, female := range []bool{false, true} {
		l := sim.Look{Female: female, Hair: sim.HairStyle(i * 3), HairColor: uint8(i * 3)}
		for tier := sim.Tier(1); tier < sim.NumTiers; tier++ {
			for j, d := range dirs {
				B, L, S, T := playerLayersHD(l, d, 2*j, false, armNone)
				P, H := armorLayersHD(tier, female, d, 2*j, false)
				hdStack(img, []*canvas{B, L, S, P, T, H}, make([]int, 6), 2, playerColors[i+1], ((int(tier)-1)*3+j)*cw*k+3*k, i*ch*k+3*k, k, false)
			}
		}
	}
	// The arms of a blow by hand, over the body drawn without them.
	for i, female := range []bool{false, true} {
		l := sim.Look{Female: female, Skin: uint8(i * 2)}
		col := 0
		for _, d := range dirs {
			for _, pose := range []armPose{poseWind, poseHalf, poseFull, poseFling} {
				B, L, S, T := playerLayersHD(l, d, 0, false, armHand)
				hdStack(img, []*canvas{B, L, S, T}, make([]int, 4), 2, playerColors[i+1], col*cw*k+3*k, (2+i)*ch*k+3*k, k, false)
				sl, fi, sh := armCanvasesHD(l, d, armHand, pose)
				if !sh.none {
					hdStack(img, []*canvas{sl, fi}, make([]int, 2), 0, playerColors[i+1], col*cw*k+3*k-armPad*HDDensity*k, (2+i)*ch*k+3*k, k, false)
				}
				col++
			}
		}
	}
	writePNG(t, path, img)
}
