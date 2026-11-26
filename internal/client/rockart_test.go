package client

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// The colours of the grounds the rocks lie on, for the sheets.
var (
	rkBgGrass  = hex(0x5a8f3e)
	rkBgRocky  = hex(0x7d7566)
	rkBgSand   = hex(0xd9a35c)
	rkBgAsh    = hex(0x2e2b2f)
	rkBgSnow   = hex(0xdfe8ee)
	rkBgCave   = hex(0x2c2829)
	rkBgSheet  = hex(0x24262e)
	rkTextCol  = hex(0xe5efef)
	rkTitleCol = hex(0xe1a845)
)

// rkCell is one drawing on a sheet: a canvas, the ground under it and its
// name.
type rkCell struct {
	c    *canvas
	bg   color.RGBA
	name string
}

// rkText writes s on img with its baseline at (x, y).
func rkText(img *image.RGBA, x, y int, s string, col color.RGBA) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(col), Face: bitmapfont.Face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

// rkSheet lays rows of cells out, each image pixel of a canvas z pixels
// across, each row under its title.
func rkSheet(heading string, titles []string, rows [][]rkCell, z int) *image.RGBA {
	const pad, label, head = 10, 16, 18
	W, H := 0, pad+head+pad
	for _, r := range rows {
		w, h := pad, 0
		for _, c := range r {
			w += c.c.w*z + 2*pad + pad
			h = max(h, c.c.h*z+2*pad)
		}
		W, H = max(W, w), H+head+h+label+pad
	}
	W = max(W, 8*len(heading)+2*pad)
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{rkBgSheet}, image.Point{}, draw.Src)
	rkText(img, pad, pad+12, heading, rkTitleCol)
	y := pad + head + pad
	for i, r := range rows {
		rkText(img, pad, y+12, titles[i], rkTitleCol)
		y += head
		h := 0
		for _, c := range r {
			h = max(h, c.c.h*z+2*pad)
		}
		x := pad
		for _, c := range r {
			bw := c.c.w*z + 2*pad
			draw.Draw(img, image.Rect(x, y, x+bw, y+h), &image.Uniform{c.bg}, image.Point{}, draw.Src)
			oy := y + h - pad - c.c.h*z
			for py := 0; py < c.c.h*z; py++ {
				for px := 0; px < c.c.w*z; px++ {
					p := c.c.px[(py/z)*c.c.w+px/z]
					if p.A == 0 {
						continue
					}
					img.SetRGBA(x+pad+px, oy+py, withAlpha(mix(c.bg, withAlpha(p, 255), float64(p.A)/255), 255))
				}
			}
			rkText(img, x, y+h+12, c.name, rkTextCol)
			x += bw + pad
		}
		y += h + label + pad
	}
	return img
}

// rkSave writes img to path as a PNG.
func rkSave(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// rkGroundBg is the colour of ground g (rockGroundOf) on the sheets.
var rkGroundBg = [numRockGrounds]color.RGBA{rkBgGrass, rkBgSand, rkBgAsh, rkBgSnow, rkBgCave}

// rkOreBg is the colour of the ground each ore lies on.
var rkOreBg = [numRockOres]color.RGBA{rkBgRocky, rkBgCave, rkBgSand, rkBgAsh, rkBgCave}

var rkOreNames = [numRockOres]string{"flint", "jade", "agate", "obsidian", "coal"}

// rkAfterRows are the rows of the sheet of every rock as drawn now.
func rkAfterRows(k float64) ([]string, [][]rkCell) {
	var titles []string
	var rows [][]rkCell
	for g, name := range [numRockGrounds]string{"granite (grass, rocky hills, woods)", "sandstone (desert)", "basalt (ash)", "snow", "cave"} {
		var row []rkCell
		for _, big := range []bool{false, true} {
			for v := 0; v < 3; v++ {
				c, _, _ := boulderScene(g, v, big).draw(k, 5)
				size := "small"
				if big {
					size = "big"
				}
				row = append(row, rkCell{c, rkGroundBg[g], fmt.Sprintf("%s v%d", size, v)})
			}
		}
		c, _, _ := pebblesScene(g).draw(k, 5)
		row = append(row, rkCell{c, rkGroundBg[g], "pebbles"})
		titles, rows = append(titles, "Boulders: "+name), append(rows, row)
	}
	var row []rkCell
	for o := 0; o < numRockOres; o++ {
		for v := 0; v < 3; v++ {
			c, _, _ := oreScene(o, v).draw(k, 5)
			row = append(row, rkCell{c, rkOreBg[o], fmt.Sprintf("%s v%d", rkOreNames[o], v)})
		}
		if o == 1 {
			titles, rows = append(titles, "Ore veins"), append(rows, row)
			row = nil
		}
	}
	titles, rows = append(titles, "Ore veins and coal"), append(rows, row)
	fc, _, _ := furnaceScene(false).draw(k, 5)
	fl, _, _ := furnaceScene(true).draw(k, 5)
	titles, rows = append(titles, "Furnace"), append(rows, []rkCell{{fc, rkBgGrass, "furnace"}, {fl, rkBgGrass, "furnace lit"}})
	return titles, rows
}

// TestDumpRocks writes, to the folder $ROCK_DUMP, the sheet of every rock
// (rocks_after.png, as at 1080p; rocks_zoom.png twice as big), the steps
// one ore vein is drawn in (rocks_steps.png), and, given the sheet of the
// rocks as they were in $ROCK_BEFORE, both together (rocks_sheet.png):
//
//	ROCK_DUMP=/tmp/rocks go test ./internal/client -run DumpRocks
func TestDumpRocks(t *testing.T) {
	dir := os.Getenv("ROCK_DUMP")
	if dir == "" {
		t.Skip("set ROCK_DUMP to a folder")
	}
	titles, rows := rkAfterRows(HDDensity)
	after := rkSheet("AFTER (1080p: 3 screen px per art px)", titles, rows, 1)
	rkSave(t, filepath.Join(dir, "rocks_after.png"), after)
	rkSave(t, filepath.Join(dir, "rocks_zoom.png"), rkSheet("AFTER, twice as big", titles, rows, 2))
	if b := os.Getenv("ROCK_BEFORE"); b != "" {
		f, err := os.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		before, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		bb := before.Bounds()
		W, H := max(bb.Dx(), after.Bounds().Dx()), bb.Dy()+after.Bounds().Dy()
		img := image.NewRGBA(image.Rect(0, 0, W, H))
		draw.Draw(img, img.Bounds(), &image.Uniform{rkBgSheet}, image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(0, 0, bb.Dx(), bb.Dy()), before, bb.Min, draw.Src)
		draw.Draw(img, image.Rect(0, bb.Dy(), W, H), after, image.Point{}, draw.Src)
		rkSave(t, filepath.Join(dir, "rocks_sheet.png"), img)
	}
	rkSave(t, filepath.Join(dir, "rocks_steps.png"), rkSteps(rkAgate, 0))
	// A few, four times as big, to look at their pixels.
	var focus []rkCell
	for g := range numRockGrounds {
		c, _, _ := boulderScene(g, 0, true).draw(HDDensity, 5)
		focus = append(focus, rkCell{c, rkGroundBg[g], ""})
	}
	var ores []rkCell
	for o := range numRockOres {
		c, _, _ := oreScene(o, 0).draw(HDDensity, 5)
		ores = append(ores, rkCell{c, rkOreBg[o], rkOreNames[o]})
	}
	fc, _, _ := furnaceScene(false).draw(HDDensity, 5)
	fl, _, _ := furnaceScene(true).draw(HDDensity, 5)
	furn := []rkCell{{fc, rkBgGrass, "furnace"}, {fl, rkBgGrass, "lit"}}
	rkSave(t, filepath.Join(dir, "rocks_focus.png"), rkSheet("four times as big", []string{"boulders", "ores", "furnace"}, [][]rkCell{focus, ores, furn}, 4))
}

// rkSteps is ore o drawn step by step, as the tutorial draws it, four
// times as big as at 1080p.
func rkSteps(o, v int) *image.RGBA {
	s := oreScene(o, v)
	names := []string{"1 palette", "2 silhouette", "3 light", "4 dark", "5 details", "6 the ore", "in game (1:1)"}
	const z, pad, label, cols = 4, 14, 20, 4
	var cs []*canvas
	for st := 1; st <= 5; st++ {
		c, _, _ := s.draw(HDDensity, st)
		cs = append(cs, c)
	}
	cw, ch := cs[0].w*z, cs[0].h*z
	W := pad + cols*(cw+pad)
	H := pad + 2*(label+ch+pad) + label
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{rkBgSheet}, image.Point{}, draw.Src)
	bg := rkOreBg[o]
	cell := func(i int) (int, int) { return pad + i%cols*(cw+pad), pad + i/cols*(label+ch+pad) + label }
	box := func(i int) (int, int) {
		x, y := cell(i)
		draw.Draw(img, image.Rect(x, y, x+cw, y+ch), &image.Uniform{bg}, image.Point{}, draw.Src)
		rkText(img, x, y-6, names[i], rkTitleCol)
		return x, y
	}
	// The palette: the ramps this ore is drawn from, out of COMFY52.
	x0, y0 := box(0)
	host := []*rkMat{rkGraniteMat, rkCaveMat, rkSandMat, rkBasaltMat, rkGraniteMat}[o]
	ore := []*rkMat{rkFlintMat, rkJadeMat, rkAgateMat, rkGlassMat, rkCoalMat}[o]
	sw := cw / 7
	for r, rp := range []ramp{append(ramp{host.ink}, host.tone...), append(ramp{ore.ink}, ore.tone...), {host.glint, ore.glint}} {
		for i, col := range rp {
			x, y := x0+8+i*sw, y0+8+r*(sw+8)
			draw.Draw(img, image.Rect(x, y, x+sw-4, y+sw), &image.Uniform{col}, image.Point{}, draw.Src)
		}
	}
	blit := func(c *canvas, x0, y0, z int) {
		for py := 0; py < c.h*z; py++ {
			for px := 0; px < c.w*z; px++ {
				if p := c.px[(py/z)*c.w+px/z]; p.A > 0 {
					img.SetRGBA(x0+px, y0+py, withAlpha(p, 255))
				}
			}
		}
	}
	for i, c := range cs {
		x, y := box(i + 1)
		blit(c, x, y, z)
	}
	x, y := box(6)
	blit(cs[4], x+(cw-cs[4].w)/2, y+ch-cs[4].h-pad, 1)
	rkText(img, pad, H-6, "Mạch mã não (agate), variant 0: the steps of the tutorial, 4x the size it has at 1080p.", rkTextCol)
	return img
}

// rkOklab is colour c in OKLab, where distance is close to how different
// two colours look.
func rkOklab(c color.RGBA) [3]float64 {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	r, g, b := lin(c.R), lin(c.G), lin(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return [3]float64{
		0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

// rkSnap52 gives every pixel of img the nearest COMFY52 colour.
func rkSnap52(img image.Image) *image.RGBA {
	var pal [len(comfy52)][3]float64
	for i, v := range comfy52 {
		pal[i] = rkOklab(hex(v))
	}
	seen := map[color.RGBA]color.RGBA{}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
			s, ok := seen[c]
			if !ok {
				p, best := rkOklab(c), math.Inf(1)
				for i, q := range pal {
					if d := (p[0]-q[0])*(p[0]-q[0]) + (p[1]-q[1])*(p[1]-q[1]) + (p[2]-q[2])*(p[2]-q[2]); d < best {
						best, s = d, hex(comfy52[i])
					}
				}
				seen[c] = s
			}
			out.SetRGBA(x, y, s)
		}
	}
	return out
}

// TestPalettePreview writes to $PALETTE_PREVIEW the middle of each of the
// screenshots in $PALETTE_SCENES (comma-separated), 960 by 540 pixels as
// shot, beside the same with every pixel snapped to the nearest COMFY52
// colour: a look at the game in that palette.
func TestPalettePreview(t *testing.T) {
	out, scenes := os.Getenv("PALETTE_PREVIEW"), os.Getenv("PALETTE_SCENES")
	if out == "" || scenes == "" {
		t.Skip("set PALETTE_PREVIEW to a file and PALETTE_SCENES to screenshots")
	}
	const cw, ch, pad, head = 960, 540, 8, 22
	paths := strings.Split(scenes, ",")
	img := image.NewRGBA(image.Rect(0, 0, 2*cw+3*pad, head+len(paths)*(ch+pad)+pad))
	draw.Draw(img, img.Bounds(), &image.Uniform{rkBgSheet}, image.Point{}, draw.Src)
	rkText(img, pad, 15, "Left: the game as it is (1080p, 1:1 crops).   Right: every pixel snapped to the nearest COMFY52 colour (OKLab).", rkTitleCol)
	for i, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		src, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		b := src.Bounds()
		x0, y0 := b.Min.X+(b.Dx()-cw)/2, b.Min.Y+(b.Dy()-ch)/2
		crop := image.NewRGBA(image.Rect(0, 0, cw, ch))
		draw.Draw(crop, crop.Bounds(), src, image.Point{x0, y0}, draw.Src)
		y := head + i*(ch+pad)
		draw.Draw(img, image.Rect(pad, y, pad+cw, y+ch), crop, image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(2*pad+cw, y, 2*pad+2*cw, y+ch), rkSnap52(crop), image.Point{}, draw.Src)
	}
	rkSave(t, out, img)
}
