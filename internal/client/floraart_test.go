package client

import (
	"fmt"
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
	rows = append(rows, beds)
	for _, kind := range []struct {
		glow, mask bool
		bg         color.RGBA
	}{{false, false, sheetGrass}, {true, false, sheetGrass}, {true, true, color.RGBA{10, 14, 30, 255}}} {
		r := sheetRow{bg: kind.bg}
		add := func(pose, f int) {
			c, ax, ay := toadHD(kind.glow, kind.mask, pose, f)
			r.cells = append(r.cells, sheetCell{c, ax * HDDensity, ay * HDDensity})
		}
		add(toadSit, 0)
		add(toadSit, 1)
		for f := 1; f <= toadHopFrames; f++ {
			add(toadHop, f)
		}
		add(toadPuff, 0)
		rows = append(rows, r)
	}
	ic := sheetRow{bg: color.RGBA{0x2a, 0x24, 0x20, 255}}
	cs := floraIconCanvases()
	for k := sim.ItemBamboo; k <= sim.ItemGlowGland; k++ {
		big := newCanvas(48, 48) // as the bag shows it: three times its size
		for i := range big.px {
			big.px[i] = cs[k].px[(i/48/3)*16+(i%48)/3]
		}
		ic.cells = append(ic.cells, sheetCell{big, 0, 47})
	}
	return append(rows, ic)
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
	// The hop, as the game plays it: both toads leaping across the grass,
	// one frame of the drawing to three of the screen, to put together as a
	// GIF (toad_hop_NN.png).
	const k, W, H = 4, 150, 44
	n := 0
	for step := 0; step < 3*12; step++ {
		f := step % 12 // 0, 1: sitting; 2 to 9: the hop; 10, 11: settled
		hops := float64(step / 12)
		pose, fr, along := toadSit, 0, 0.0
		switch {
		case f >= 2 && f <= 9:
			pose, fr, along = toadHop, f-1, float64(f-2)/7
		case f > 9:
			pose, fr, along = toadHop, toadHopFrames, 1
		}
		img := image.NewRGBA(image.Rect(0, 0, W*k, H*k))
		draw.Draw(img, img.Bounds(), &image.Uniform{sheetGrass}, image.Point{}, draw.Src)
		for i, glow := range []bool{false, true} {
			c, ax, ay := toadHD(glow, false, pose, fr)
			x := 14 + (hops+along)*20.4 + float64(i)*52 // a hop is sim.ToadHop of a tile: about 20 art pixels
			y := 34.0 - toadHopBob[fr]
			big := image.NewRGBA(image.Rect(0, 0, c.w*k/HDDensity, c.h*k/HDDensity))
			for py := 0; py < big.Bounds().Dy(); py++ {
				for px := 0; px < big.Bounds().Dx(); px++ {
					p := c.px[(py*HDDensity/k)*c.w+px*HDDensity/k]
					big.SetRGBA(px, py, color.RGBA{uint8(int(p.R) * int(p.A) / 255), uint8(int(p.G) * int(p.A) / 255), uint8(int(p.B) * int(p.A) / 255), p.A})
				}
			}
			// its shadow stays on the ground
			for sx := -5 * k; sx <= 5*k; sx++ {
				for sy := -k; sy <= k; sy++ {
					if float64(sx*sx)/float64(25*k*k)+float64(sy*sy)/float64(k*k) <= 1 {
						img.SetRGBA(int(x*k)+sx, 35*k+sy, color.RGBA{0x46, 0x70, 0x30, 255})
					}
				}
			}
			draw.Draw(img, big.Bounds().Add(image.Pt(int((x-ax)*k), int((y-ay)*k))), big, image.Point{}, draw.Over)
		}
		fp, err := os.Create(filepath.Join(dir, fmt.Sprintf("toad_hop_%02d.png", n)))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(fp, img); err != nil {
			t.Fatal(err)
		}
		fp.Close()
		n++
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

// A toad has every frame of its hop, each different from the last, and its
// glow mask only ever lights what the toad itself covers.
func TestToadHopFrames(t *testing.T) {
	for _, glow := range []bool{false, true} {
		var last *canvas
		for f := 1; f <= toadHopFrames; f++ {
			c, _, _ := toadHD(glow, false, toadHop, f)
			if last != nil {
				same := true
				for i := range c.px {
					same = same && c.px[i] == last.px[i]
				}
				if same {
					t.Errorf("glow %v: hop frame %d is frame %d again", glow, f, f-1)
				}
			}
			last = c
			for x := 0; x < c.w; x++ {
				if c.px[x].A > 0 || c.px[(c.h-1)*c.w+x].A > 0 {
					t.Errorf("glow %v: hop frame %d is cut off by its canvas", glow, f)
					break
				}
			}
			m, _, _ := toadHD(glow, true, toadHop, f)
			lit := 0
			for i, p := range m.px {
				if p.A == 0 {
					continue
				}
				lit++
				if c.px[i].A == 0 {
					t.Fatalf("glow %v: frame %d's mask shines outside the toad", glow, f)
				}
			}
			if glow && lit < 40 {
				t.Errorf("the glow toad's frame %d has %d shining pixels", f, lit)
			}
		}
	}
	if toadHopBob[0] != 0 || toadHopBob[toadHopFrames] != 0 {
		t.Error("a hop does not start and end on the ground")
	}
}
