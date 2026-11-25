package client

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"survivalgame/internal/sim"
)

// canvasRGBA is c as an image, k times bigger.
func canvasRGBA(c *canvas, k int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.w*k, c.h*k))
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			p := c.px[y*c.w+x]
			for j := 0; j < k; j++ {
				for i := 0; i < k; i++ {
					img.SetRGBA(x*k+i, y*k+j, color.RGBA{uint8(int(p.R) * int(p.A) / 255), uint8(int(p.G) * int(p.A) / 255), uint8(int(p.B) * int(p.A) / 255), p.A})
				}
			}
		}
	}
	return img
}

// sheetRow is a row of drawings on one ground colour.
type sheetRow struct {
	bg    color.RGBA
	cells []sheetCell
}

type sheetCell struct {
	c      *canvas
	ax, ay float64 // the anchor, in image pixels
}

// writeSheet lays rows out, each drawing standing on a common line, k times
// its size, and writes the PNG.
func writeSheet(path string, rows []sheetRow, k int) error {
	const pad = 10
	W, H := 0, 0
	var hs []int
	for _, r := range rows {
		w, up, down := pad, 0, 0
		for _, c := range r.cells {
			w += c.c.w*k + pad
			up, down = max(up, int(c.ay)*k), max(down, (c.c.h-int(c.ay))*k)
		}
		W = max(W, w)
		hs = append(hs, up, down)
		H += up + down + 2*pad
	}
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	y := 0
	for i, r := range rows {
		up, down := hs[2*i], hs[2*i+1]
		draw.Draw(img, image.Rect(0, y, W, y+up+down+2*pad), &image.Uniform{r.bg}, image.Point{}, draw.Src)
		x := pad
		for _, c := range r.cells {
			src := canvasRGBA(c.c, k)
			draw.Draw(img, src.Bounds().Add(image.Pt(x, y+pad+up-int(c.ay)*k)), src, image.Point{}, draw.Over)
			x += c.c.w*k + pad
		}
		y += up + down + 2*pad
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

var (
	sheetGrass = color.RGBA{0x5c, 0x8a, 0x3c, 255}
	sheetWater = color.RGBA{0x2c, 0x6a, 0x86, 255}
	sheetSoil  = color.RGBA{0x5a, 0x40, 0x2a, 255}
)

// floraSheet is every new plant, full above picked, and the crops by stage.
func floraSheet() []sheetRow {
	var rows []sheetRow
	for _, full := range []bool{true, false} {
		land, wet := sheetRow{bg: sheetGrass}, sheetRow{bg: sheetWater}
		for f := sim.FloraKind(0); f < sim.NumFlora; f++ {
			if floraGens[f] == nil {
				continue
			}
			c, ax, ay := floraGens[f](full)
			if f.InWater() {
				wet.cells = append(wet.cells, sheetCell{c, ax, ay})
			} else {
				land.cells = append(land.cells, sheetCell{c, ax, ay})
			}
		}
		rows = append(rows, land, wet)
	}
	beds := sheetRow{bg: sheetSoil}
	for c := sim.CropKind(1); c < sim.NumCrops; c++ {
		for look := 0; look <= cropRipe; look++ {
			cv, ax, ay := cropGens[c](look)
			beds.cells = append(beds.cells, sheetCell{cv, ax, ay})
		}
	}
	return append(rows, beds)
}

// FLORA_DUMP=dir go test ./internal/client -run DumpFlora -v writes the
// sheet of the new plants there, at the size they have on a 1080p screen
// and twice that.
func TestDumpFlora(t *testing.T) {
	dir := os.Getenv("FLORA_DUMP")
	if dir == "" {
		t.Skip("set FLORA_DUMP to a directory to write the sheets")
	}
	for k, name := range map[int]string{1: "flora_sheet_1x.png", 2: "flora_sheet.png"} {
		if err := writeSheet(filepath.Join(dir, name), floraSheet(), k); err != nil {
			t.Fatal(err)
		}
	}
}

// Every plant draws, full and picked, differently, with its foot on its
// anchor and nothing cut off by the edge of its canvas.
func TestEveryPlantIsDrawn(t *testing.T) {
	for f := sim.FloraKind(0); f < sim.NumFlora; f++ {
		if floraGens[f] == nil {
			t.Errorf("%s has no drawing", f.Name())
			continue
		}
		full, ax, ay := floraGens[f](true)
		picked, _, _ := floraGens[f](false)
		if full.dens != HDDensity || picked.dens != HDDensity {
			t.Errorf("%s is not drawn at HD", f.Name())
		}
		if ax < 0 || ay < 0 || int(ax) >= full.w || int(ay) >= full.h {
			t.Errorf("%s: anchor (%.0f, %.0f) outside its %dx%d drawing", f.Name(), ax, ay, full.w, full.h)
		}
		same, n := true, 0
		for i, p := range full.px {
			if p.A > 0 {
				n++
			}
			same = same && p == picked.px[i]
		}
		if same {
			t.Errorf("%s looks the same picked", f.Name())
		}
		if n < 300 {
			t.Errorf("%s: only %d pixels drawn", f.Name(), n)
		}
		for _, c := range []*canvas{full, picked} {
			for x := 0; x < c.w; x++ {
				if c.px[x].A > 0 {
					t.Errorf("%s touches the top of its canvas at x=%d", f.Name(), x)
					break
				}
			}
			for y := 0; y < c.h; y++ {
				if c.px[y*c.w].A > 0 || c.px[y*c.w+c.w-1].A > 0 {
					t.Errorf("%s touches the side of its canvas at y=%d", f.Name(), y)
					break
				}
			}
		}
	}
}
