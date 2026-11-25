package client

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"survivalgame/internal/sim"
)

// TestDumpCookSheet writes, to the file $COOK_SHEET, a sheet of the cooking
// gear (each piece at HD, twice enlarged) and of the bag icons of the
// cooking items (six times enlarged).
//
//	COOK_SHEET=/tmp/cook_sheet.png go test ./internal/client -run DumpCookSheet
func TestDumpCookSheet(t *testing.T) {
	path := os.Getenv("COOK_SHEET")
	if path == "" {
		t.Skip("set COOK_SHEET to a file")
	}
	layer := func(cs ...*canvas) *canvas {
		out := newCanvas(cs[0].w, cs[0].h)
		for _, c := range cs {
			for i, p := range c.px {
				if p.A > 0 {
					out.px[i] = p
				}
			}
		}
		return out
	}
	both := func(back, front *canvas) *canvas { return layer(back, front) }
	one := func(c *canvas, _, _ float64) *canvas { return c }
	rows := [][]*canvas{
		{both(genPotCanvas(-1)), both(genPotCanvas(0)), both(genPotCanvas(1)), both(genGrillCanvas(false)), both(genGrillCanvas(true)), both(genPanCanvas(false)), both(genPanCanvas(true))},
		{one(genMortarCanvas(false)), one(genMortarCanvas(true)), one(genOvenCanvas(false, false)), one(genOvenCanvas(true, false)), one(genOvenCanvas(true, true)), one(genClayCanvas(false)), one(genClayCanvas(true))},
		{one(genRackCanvas(0)), one(genRackCanvas(1)), one(genRackCanvas(2)), one(genRackCanvas(3)), one(genRackCanvas(4))},
	}
	const zoom, pad = 2, 16
	W, H := 0, pad
	var rowH []int
	for _, r := range rows {
		w, h := pad, 0
		for _, c := range r {
			w += c.w*zoom + pad
			h = max(h, c.h*zoom)
		}
		W, H = max(W, w), H+h+pad
		rowH = append(rowH, h)
	}
	const iz, cols = 6, 13
	iconRows := (int(sim.ItemFruitPie-sim.ItemClay) + cols) / cols
	W = max(W, pad+cols*(16*iz+pad))
	H += iconRows * (16*iz + pad)
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{96, 136, 78, 255}}, image.Point{}, draw.Src)
	blit := func(c *canvas, x0, y0, z int) {
		for y := 0; y < c.h*z; y++ {
			for x := 0; x < c.w*z; x++ {
				p := c.px[(y/z)*c.w+x/z]
				if p.A == 0 {
					continue
				}
				bg := img.RGBAAt(x0+x, y0+y)
				a := int(p.A)
				img.SetRGBA(x0+x, y0+y, color.RGBA{
					uint8((int(p.R)*a + int(bg.R)*(255-a)) / 255),
					uint8((int(p.G)*a + int(bg.G)*(255-a)) / 255),
					uint8((int(p.B)*a + int(bg.B)*(255-a)) / 255), 255})
			}
		}
	}
	y := pad
	for i, r := range rows {
		x := pad
		for _, c := range r {
			blit(c, x, y+rowH[i]-c.h*zoom, zoom)
			x += c.w*zoom + pad
		}
		y += rowH[i] + pad
	}
	draw.Draw(img, image.Rect(0, y-pad/2, W, H), &image.Uniform{color.RGBA{24, 26, 36, 255}}, image.Point{}, draw.Src)
	n := 0
	for k := sim.ItemClay; k <= sim.ItemFruitPie; k++ {
		c := cookIconCanvas(k)
		if c == nil {
			t.Errorf("no icon for %s", k.Def().Name)
			continue
		}
		blit(c, pad+n%cols*(16*iz+pad), y+n/cols*(16*iz+pad), iz)
		n++
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

// Every cooking item has an icon and a note in the bag.
func TestCookItemsHaveIconsAndNotes(t *testing.T) {
	for k := sim.ItemClay; k <= sim.ItemFruitPie; k++ {
		if cookIconCanvas(k) == nil {
			t.Errorf("%s has no icon", k.Def().Name)
		}
		if itemInfo[k] == "" {
			t.Errorf("%s has no note", k.Def().Name)
		}
	}
}
