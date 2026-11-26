package client

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func critNames() []string {
	var names []string
	for n := range critKinds {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// TestCrittersAreDrawn draws every frame of every code-drawn animal: each
// must come out with its anchor on its canvas, and a walk must really move.
func TestCrittersAreDrawn(t *testing.T) {
	for _, name := range critNames() {
		k := critKinds[name]
		if k.hd == nil {
			t.Errorf("%s has no HD drawing", name)
			continue
		}
		for p := range k.poses {
			seen := map[string]bool{}
			for f := 0; f <= k.anim[p]; f++ {
				c, ax, ay := k.hd(p, f)
				if c.dens != HDDensity {
					t.Fatalf("%s pose %d: not drawn at HD density", name, p)
				}
				if ax < 0 || ay < 0 || ax*cK > float64(c.w) || ay*cK > float64(c.h)+9 {
					t.Errorf("%s pose %d frame %d: anchor (%.0f, %.0f) off its canvas", name, p, f, ax, ay)
				}
				n := 0
				for _, px := range c.px {
					if px.A >= 128 {
						n++
					}
				}
				if n < 40 {
					t.Errorf("%s pose %d frame %d: nearly empty", name, p, f)
				}
				seen[fmt.Sprint(c.px)] = true
			}
			if k.anim[p] >= cycFrames && len(seen) < cycFrames-1 {
				t.Errorf("%s pose %d: only %d different frames in its cycle", name, p, len(seen))
			}
			if oc, oax, oay := k.old(p); oc != nil {
				c, ax, ay := k.hd(p, 0)
				// The HD drawing stands where the old one stood: the same
				// feet, give or take a couple of art pixels.
				if d := (float64(c.h)/cK - ay) - (float64(oc.h) - oay); d > 6 || d < -6 {
					t.Errorf("%s pose %d: stands %.0f art pixels off the old drawing", name, p, d)
				}
				_ = oax
				_ = ax
			}
		}
	}
}

// TestDumpCritters writes, to the folder $CRITTER_DUMP, a sheet of each
// animal (the old drawing of every pose above the HD one, then the frames of
// each moving pose) and a GIF of each moving pose, and logs what drawing
// them costs. $CRITTER_ONLY picks animals by name (comma separated).
func TestDumpCritters(t *testing.T) {
	dir := os.Getenv("CRITTER_DUMP")
	if dir == "" {
		t.Skip("set CRITTER_DUMP to a folder")
	}
	only := os.Getenv("CRITTER_ONLY")
	bg := color.NRGBA{96, 136, 78, 255}
	var total, base time.Duration
	baseBytes := 0
	frames, bytes := 0, 0
	for _, name := range critNames() {
		if only != "" && !strings.Contains(","+only+",", ","+name+",") {
			continue
		}
		k := critKinds[name]
		type cell struct {
			c      *canvas
			ax, ay float64 // image pixels
		}
		at3 := func(c *canvas, ax, ay float64) cell {
			if c.dens == HDDensity {
				return cell{c, ax * cK, ay * cK}
			}
			big := newCanvas(c.w*cK, c.h*cK) // the old art, enlarged
			for y := 0; y < big.h; y++ {
				for x := 0; x < big.w; x++ {
					big.px[y*big.w+x] = c.px[(y/cK)*c.w+x/cK]
				}
			}
			return cell{big, ax * cK, ay * cK}
		}
		var rows [][]cell
		var olds, news []cell
		for p := range k.poses {
			olds = append(olds, at3(k.old(p)))
			if oc, oax, oay := k.old(p); os.Getenv("CRITTER_BOXES") != "" {
				x0, y0, x1, y1 := oc.w, oc.h, 0, 0
				for y := 0; y < oc.h; y++ {
					for x := 0; x < oc.w; x++ {
						if oc.px[y*oc.w+x].A >= 128 {
							x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
						}
					}
				}
				t.Logf("%s %d: %dx%d anchor %.1f,%.1f box %d,%d-%d,%d", name, p, oc.w, oc.h, oax, oay, x0, y0, x1, y1)
			}
			if k.hd == nil {
				continue
			}
			start := time.Now()
			c, ax, ay := k.hd(p, 0)
			total += time.Since(start)
			base += time.Since(start)
			baseBytes += 4 * c.w * c.h
			frames++
			bytes += 4 * c.w * c.h
			news = append(news, at3(c, ax, ay))
		}
		rows = append(rows, olds, news)
		var animPoses []int
		for p := range k.poses {
			if k.anim[p] == 0 {
				continue
			}
			animPoses = append(animPoses, p)
			var row []cell
			for f := 1; f <= k.anim[p]; f++ {
				start := time.Now()
				c, ax, ay := k.hd(p, f)
				total += time.Since(start)
				frames++
				bytes += 4 * c.w * c.h
				row = append(row, at3(c, ax, ay))
			}
			rows = append(rows, row)
		}
		// One cell size for the lot, every drawing set down on its anchor.
		var l, r, u, d float64
		n := 0
		for _, row := range rows {
			n = max(n, len(row))
			for _, c := range row {
				l, r = max(l, c.ax), max(r, float64(c.c.w)-c.ax)
				u, d = max(u, c.ay), max(d, float64(c.c.h)-c.ay)
			}
		}
		cw, ch := int(l+r)+6, int(u+d)+6
		put := func(img draw.Image, c cell, ox, oy int) {
			for y := 0; y < c.c.h; y++ {
				for x := 0; x < c.c.w; x++ {
					if p := c.c.px[y*c.c.w+x]; p.A >= 128 {
						img.Set(ox+3+int(l-c.ax)+x, oy+3+int(u-c.ay)+y, color.NRGBA{p.R, p.G, p.B, 255})
					}
				}
			}
		}
		sheet := image.NewNRGBA(image.Rect(0, 0, cw*n, ch*len(rows)))
		draw.Draw(sheet, sheet.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
		for j, row := range rows {
			for i, c := range row {
				put(sheet, c, i*cw, j*ch)
			}
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, sheet)
		f.Close()
		for j, p := range animPoses {
			g := &gif.GIF{}
			pal := append(color.Palette{}, palette.Plan9...)
			for _, c := range rows[2+j] {
				fr := image.NewPaletted(image.Rect(0, 0, cw*2, ch*2), pal)
				small := image.NewNRGBA(image.Rect(0, 0, cw, ch))
				draw.Draw(small, small.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
				put(small, c, 0, 0)
				for y := 0; y < ch*2; y++ {
					for x := 0; x < cw*2; x++ {
						fr.Set(x, y, small.At(x/2, y/2))
					}
				}
				g.Image = append(g.Image, fr)
				g.Delay = append(g.Delay, 9)
			}
			gf, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s_%d.gif", name, p)))
			if err != nil {
				t.Fatal(err)
			}
			gif.EncodeAll(gf, g)
			gf.Close()
		}
	}
	t.Logf("%d frames drawn in %v, %.1f MB of pixels; of that the poses drawn at start-up: %v, %.1f MB (twice that with their mirrors)",
		frames, total, float64(bytes)/1e6, base, float64(baseBytes)/1e6)
}
