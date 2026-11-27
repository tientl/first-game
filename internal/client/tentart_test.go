package client

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"survivalgame/internal/sim"
)

// Every thatch of the leaf tent is drawn within its canvas, covers what a
// tent two tiles wide should, and leaves its open front to the inside.
func TestLeafTentDrawings(t *testing.T) {
	for v := tentLeaf(0); v < numTentLeaves; v++ {
		back, front := genTent(v)
		n, edge, ex, ey := 0, false, 0, 0
		for y := 0; y < front.h; y++ {
			for x := 0; x < front.w; x++ {
				if front.px[y*front.w+x].A < 128 {
					continue
				}
				n++
				if x == 0 || y == 0 || x == front.w-1 || y == front.h-1 {
					edge, ex, ey = true, x, y
				}
			}
		}
		if edge {
			t.Errorf("thatch %d touches the edge of its canvas at %d,%d", v, ex, ey)
		}
		if share := float64(n) / float64(front.w*front.h); share < 0.2 || share > 0.75 {
			t.Errorf("thatch %d: the shell covers %.0f%% of its canvas", v, 100*share)
		}
		// The middle of the front, where a sleeper lies, is the inside's.
		cx, cy := int(tentAX*tk), int((tentAY-8)*tk)
		if front.px[cy*front.w+cx].A >= 128 || back.px[cy*back.w+cx].A < 128 {
			t.Errorf("thatch %d: the open front is not open", v)
		}
	}
}

// TENT_DUMP=dir go test ./internal/client -run DumpTent writes every thatch
// three times as big, whole and in its two parts.
func TestDumpTent(t *testing.T) {
	dir := os.Getenv("TENT_DUMP")
	if dir == "" {
		t.Skip("set TENT_DUMP to a folder")
	}
	grounds := [numTentLeaves]color.RGBA{{92, 132, 74, 255}, {64, 100, 58, 255}, {214, 190, 140, 255}, {226, 234, 242, 255}}
	const z = 3
	cw, ch := tentW*HDDensity, tentHt*HDDensity
	img := image.NewRGBA(image.Rect(0, 0, int(numTentLeaves)*cw*z, 2*ch*z))
	for v := tentLeaf(0); v < numTentLeaves; v++ {
		back, front := genTent(v)
		for row, layers := range [][]*canvas{{back, front}, {back}} {
			for y := 0; y < ch; y++ {
				for x := 0; x < cw; x++ {
					col := grounds[v]
					for _, c := range layers {
						col = over(col, c.px[y*cw+x])
					}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							img.Set(int(v)*cw*z+x*z+dx, row*ch*z+y*z+dy, col)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "tent_dump.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// TestDumpTent also writes tent_dump1.png: every thatch as at 1080p, empty
// and with someone asleep in it, then twice as big.
func TestDumpTentAsleep(t *testing.T) {
	dir := os.Getenv("TENT_DUMP")
	if dir == "" {
		t.Skip("set TENT_DUMP to a folder")
	}
	grounds := [numTentLeaves]color.RGBA{{92, 132, 74, 255}, {64, 100, 58, 255}, {214, 190, 140, 255}, {226, 234, 242, 255}}
	look := sim.Look{Hair: sim.HairShaggy, HairColor: 2, Skin: 1}
	k := laidKey{look: look, tint: playerColors[int(sheetID)%len(playerColors)], m: moSleep, lay: true, cover: 2}
	sl, sax, say := laidCanvas(k)
	cw, ch := tentW*HDDensity, tentHt*HDDensity
	base := image.NewRGBA(image.Rect(0, 0, int(numTentLeaves)*2*cw, ch))
	for v := tentLeaf(0); v < numTentLeaves; v++ {
		back, front := genTent(v)
		for j := 0; j < 2; j++ {
			ox := (int(v)*2 + j) * cw
			for y := 0; y < ch; y++ {
				for x := 0; x < cw; x++ {
					base.Set(ox+x, y, over(grounds[v], back.px[y*cw+x]))
				}
			}
			if j == 1 { // the sleeper, put where drawSleeper puts them
				px := float64(ox) + (tentAX+tentSleepDX-sax)*tk
				py := (tentAY + tentSleepDY - say) * tk
				for y := 0; y < sl.h; y++ {
					for x := 0; x < sl.w; x++ {
						if c := sl.px[y*sl.w+x]; c.A > 0 {
							X, Y := int(px)+x, int(py)+y
							base.Set(X, Y, over(base.RGBAAt(X, Y), c))
						}
					}
				}
			}
			for y := 0; y < ch; y++ {
				for x := 0; x < cw; x++ {
					base.Set(ox+x, y, over(base.RGBAAt(ox+x, y), front.px[y*cw+x]))
				}
			}
		}
	}
	const z = 2
	img := image.NewRGBA(image.Rect(0, 0, base.Bounds().Dx()*z, ch*(1+z)))
	for y := 0; y < ch; y++ {
		for x := 0; x < base.Bounds().Dx(); x++ {
			img.Set(x, y, base.At(x, y))
		}
	}
	for y := 0; y < ch*z; y++ {
		for x := 0; x < base.Bounds().Dx()*z; x++ {
			img.Set(x, ch+y, base.At(x/z, y/z))
		}
	}
	f, err := os.Create(filepath.Join(dir, "tent_dump1.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
