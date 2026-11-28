package client

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"survivalgame/internal/sim"
)

// bodyKinds is every kind of beast that leaves a body (sim/drops.go).
func bodyKinds() []sim.CreatureKind {
	var out []sim.CreatureKind
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		switch k {
		case sim.CreatureWorm, sim.CreatureFish, sim.CreatureToad, sim.CreatureGlowToad:
			continue
		}
		out = append(out, k)
	}
	return out
}

// opaqueBox is the box round what is drawn on c.
func opaqueBox(c *canvas) (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = c.w, c.h, -1, -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A >= 128 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	return
}

// Every kind that leaves a body lies down: drawn from its own drawing where
// it has one, lower than it stood, and every step of its fall draws.
func TestCorpseEveryKind(t *testing.T) {
	for _, k := range bodyKinds() {
		sp := corpseSpecOf(k)
		if sp.style == corpseSprite {
			t.Logf("%v: its own picture laid low", k)
			continue
		}
		for step := 0; step <= corpseSteps; step++ {
			c, ax, ay, ok := corpseCanvas(k, float64(step)/corpseSteps)
			if !ok {
				t.Fatalf("%v step %d: no body drawn", k, step)
			}
			x0, y0, x1, y1 := opaqueBox(c)
			if x1 < x0 {
				t.Fatalf("%v step %d: the body is empty", k, step)
			}
			if ax < float64(x0)-2 || ax > float64(x1)+2 || ay < float64(y0)-2 {
				t.Errorf("%v step %d: anchor %.0f,%.0f off the body %d,%d-%d,%d", k, step, ax, ay, x0, y0, x1, y1)
			}
		}
		stand, _, _, _ := corpseCanvas(k, 0)
		lie, _, _, _ := corpseCanvas(k, 1)
		_, sy0, _, sy1 := opaqueBox(stand)
		_, ly0, _, ly1 := opaqueBox(lie)
		if sp.style == corpseSide && ly1-ly0 >= sy1-sy0 {
			t.Errorf("%v: lying %d high, no lower than standing (%d)", k, ly1-ly0, sy1-sy0)
		}
	}
}

// CORPSE_DUMP=dir go test ./internal/client -run DumpCorpses writes a sheet
// of every body: standing, its fall, and lying, three times 1080p.
func TestDumpCorpses(t *testing.T) {
	dir := os.Getenv("CORPSE_DUMP")
	if dir == "" {
		t.Skip("set CORPSE_DUMP to a folder")
	}
	bg := color.RGBA{79, 106, 64, 255}
	type cell struct {
		c      *canvas
		ax, ay float64
	}
	var rows [][]cell
	cw, ch := 0, 0
	l, u := 0.0, 0.0
	only := os.Getenv("CORPSE_ONLY")
	for _, k := range bodyKinds() {
		if corpseSpecOf(k).style == corpseSprite {
			continue
		}
		if only != "" && !strings.Contains(","+only+",", ","+fbKindName(k)+",") {
			continue
		}
		var row []cell
		for step := 0; step <= corpseSteps; step++ {
			if only != "" && step != 0 && step != corpseSteps {
				continue
			}
			c, ax, ay, _ := corpseCanvas(k, float64(step)/corpseSteps)
			row = append(row, cell{c, ax, ay})
			l, u = max(l, ax), max(u, ay)
		}
		rows = append(rows, row)
	}
	for _, row := range rows {
		for _, c := range row {
			cw = max(cw, int(l+float64(c.c.w)-c.ax)+8)
			ch = max(ch, int(u+float64(c.c.h)-c.ay)+8)
		}
	}
	z, cols := 1, corpseSteps+1
	if only != "" {
		z, cols = 3, 2
	}
	sheet := image.NewRGBA(image.Rect(0, 0, cw*cols, ch*len(rows)))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	for j, row := range rows {
		for i, c := range row {
			ox, oy := i*cw+4+int(l-c.ax), j*ch+4+int(u-c.ay)
			for y := 0; y < c.c.h; y++ {
				for x := 0; x < c.c.w; x++ {
					if p := c.c.px[y*c.c.w+x]; p.A >= 128 {
						sheet.SetRGBA(ox+x, oy+y, color.RGBA{p.R, p.G, p.B, 255})
					}
				}
			}
			sheet.SetRGBA(i*cw+4+int(l), j*ch+4+int(u), color.RGBA{255, 255, 0, 255}) // the anchor
		}
	}
	f, err := os.Create(filepath.Join(dir, "corpse_dump.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, enlarge(sheet, z)); err != nil {
		t.Fatal(err)
	}
}

// CORPSE_DUMP=dir CORPSE_LYING=1 go test ./internal/client -run DumpLying
// writes every body lying, beside its beast standing, four to a row, at
// 1080p (lying_dump.png).
func TestDumpLying(t *testing.T) {
	dir := os.Getenv("CORPSE_DUMP")
	if dir == "" || os.Getenv("CORPSE_LYING") == "" {
		t.Skip("set CORPSE_DUMP and CORPSE_LYING")
	}
	type cell struct {
		c              *canvas
		ax, ay         float64
		x0, y0, x1, y1 int
	}
	var cells []cell
	only := os.Getenv("CORPSE_ONLY")
	for _, k := range bodyKinds() {
		if corpseSpecOf(k).style == corpseSprite {
			continue
		}
		if only != "" && !strings.Contains(","+only+",", ","+fbKindName(k)+",") {
			continue
		}
		for _, st := range []float64{0, 1} {
			c, ax, ay, _ := corpseCanvas(k, st)
			x0, y0, x1, y1 := opaqueBox(c)
			cells = append(cells, cell{c, ax, ay, x0, y0, x1, y1})
		}
	}
	const cols = 8
	cw, ch := 0, 0
	for _, c := range cells {
		cw, ch = max(cw, c.x1-c.x0+12), max(ch, c.y1-c.y0+12)
	}
	rows := (len(cells) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cw*cols, ch*rows))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{79, 106, 64, 255}}, image.Point{}, draw.Src)
	for i, c := range cells {
		ox, oy := (i%cols)*cw+6-c.x0, (i/cols)*ch+ch-6-c.y1
		for y := c.y0; y <= c.y1; y++ {
			for x := c.x0; x <= c.x1; x++ {
				if p := c.c.px[y*c.c.w+x]; p.A >= 128 {
					sheet.SetRGBA(ox+x, oy+y, color.RGBA{p.R, p.G, p.B, 255})
				}
			}
		}
		for d := -3; d <= 3; d++ { // the anchor: where it touches the ground
			sheet.SetRGBA(ox+int(c.ax)+d, oy+int(c.ay), color.RGBA{255, 255, 0, 255})
		}
	}
	f, err := os.Create(filepath.Join(dir, "lying_dump.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		t.Fatal(err)
	}
}
