package client

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// dinoCanvases lists every pose of the new animals, with its anchor.
func dinoCanvases() (out []struct {
	name   string
	c      *canvas
	ax, ay float64
}) {
	add := func(name string, n int, f func(int) (*canvas, float64, float64)) {
		for p := range n {
			c, ax, ay := f(p)
			out = append(out, struct {
				name   string
				c      *canvas
				ax, ay float64
			}{name, c, ax, ay})
		}
	}
	add("trike", trikePoses, trikeCanvas)
	add("stego", stegoPoses, stegoCanvas)
	add("brachio", brachPoses, brachioCanvas)
	add("raptor", raptorPoses, raptorCanvas)
	add("rex", rexPoses, rexCanvas)
	add("ptero", pteroPoses, pteroCanvas)
	add("plesio", plesioPoses, plesioCanvas)
	add("mammoth", mammothPoses, mammothCanvas)
	add("sabre", sabrePoses, sabreCanvas)
	add("toro", trikePoses, func(p int) (*canvas, float64, float64) { return trikeCanvasOf(p, toroLook) })
	add("achero", raptorPoses, func(p int) (*canvas, float64, float64) { return raptorCanvasOf(p, acheroLook) })
	add("ornitho", ornithoPoses, ornithoCanvas)
	add("thesc", thescPoses, thescCanvas)
	add("anzu", anzuPoses, anzuCanvas)
	add("quetzal", quetzalPoses, quetzalCanvas)
	add("quetzalFly", pteroPerch, func(p int) (*canvas, float64, float64) { return pteroCanvasOf(p, quetzalFlyLook) })
	add("brachy", crocPoses, func(p int) (*canvas, float64, float64) { return crocCanvasOf(p, brachyLook) })
	add("champso", crocPoses, func(p int) (*canvas, float64, float64) { return crocCanvasOf(p, champsoLook) })
	return out
}

func TestDinosaurSpritesAreDrawn(t *testing.T) {
	for _, d := range dinoCanvases() {
		n := 0
		for _, p := range d.c.px {
			if p.A >= 128 {
				n++
			}
		}
		if n < 30 { // a lurker is only its eyes
			t.Errorf("%s: only %d pixels drawn", d.name, n)
		}
		if d.ax < 0 || d.ay < 0 || d.ax > float64(d.c.w) || d.ay > float64(d.c.h)+3 {
			t.Errorf("%s: anchor (%.0f, %.0f) off its %dx%d canvas", d.name, d.ax, d.ay, d.c.w, d.c.h)
		}
	}
	for _, c := range []*canvas{iconHide(), iconBone(), iconRexTooth()} {
		if c.w != 16 || c.h != 16 {
			t.Errorf("icon is %dx%d, want 16x16", c.w, c.h)
		}
	}
}

// TestDumpDinosaurSheet writes every pose, enlarged, to $DINO_SHEET.
func TestDumpDinosaurSheet(t *testing.T) {
	path := os.Getenv("DINO_SHEET")
	if path == "" {
		t.Skip("set DINO_SHEET to a .png path")
	}
	const k = 4
	all := dinoCanvases()
	for _, c := range []*canvas{iconHide(), iconBone(), iconRexTooth(), iconSaddle(), iconFur(), iconIvory(), iconFurCoat(), iconIvorySpear()} {
		all = append(all, struct {
			name   string
			c      *canvas
			ax, ay float64
		}{"icon", c, 0, 0})
	}
	W, H, x, y, row := 1400, 0, 8, 8, 0
	type at struct{ x, y int }
	var places []at
	for _, d := range all {
		if x+d.c.w*k > W {
			x, y, row = 8, y+row+8, 0
		}
		places = append(places, at{x, y})
		x += d.c.w*k + 8
		row = max(row, d.c.h*k)
	}
	H = y + row + 8
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := range img.Pix {
		img.Pix[i] = 0x9a
		if i%4 == 1 {
			img.Pix[i] = 0xb0 // a grassy grey backdrop
		}
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	for i, d := range all {
		for py := 0; py < d.c.h*k; py++ {
			for px := 0; px < d.c.w*k; px++ {
				p := d.c.px[(py/k)*d.c.w+px/k]
				if p.A < 128 {
					continue
				}
				img.Set(places[i].x+px, places[i].y+py, color.RGBA{p.R, p.G, p.B, 255})
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
