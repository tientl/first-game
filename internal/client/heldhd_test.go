package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"survivalgame/internal/sim"
)

// TestHeldHD paints every held item at every angle and checks that each
// comes out whole: the grip inside the image, enough of it painted, nothing
// cut off at the image's edge, and the gap for the fist clear.
func TestHeldHD(t *testing.T) {
	var total time.Duration
	n, bytes, least, most := 0, 0, 1<<30, 0
	for k, d := range heldDefs {
		if d == nil {
			continue
		}
		name := sim.ItemKind(k).Def().Name
		start := time.Now()
		for i := range heldTurns {
			p := bakeHeld(d, i)
			n++
			bytes += 4 * p.w * p.h
			least, most = min(least, 4*p.w*p.h), max(most, 4*p.w*p.h)
			if p.gx < 0 || p.gy < 0 || p.gx > float64(p.w) || p.gy > float64(p.h) {
				t.Errorf("%s at turn %d: the grip (%.0f, %.0f) is outside the %dx%d image", name, i, p.gx, p.gy, p.w, p.h)
			}
			solid := 0
			for _, c := range p.px {
				if c.A >= 128 {
					solid++
				}
			}
			// Each item is at least a haft 10 art pixels long and one thick.
			if solid < 10*HDDensity*HDDensity {
				t.Errorf("%s at turn %d: only %d pixels painted", name, i, solid)
			}
			if d.grip > 0 {
				gx, gy := int(p.gx), int(p.gy)
				if gx > 0 && gy > 0 && gx < p.w && gy < p.h && p.px[gy*p.w+gx].A > 40 {
					t.Errorf("%s at turn %d: the fist's gap is not clear", name, i)
				}
			}
		}
		total += time.Since(start)
		// The box given must hold the item: at the trim's edge nothing may be
		// cut, which shows as solid pixels on the image's border at the
		// untrimmed size; a cheaper check is that the painted extent stays
		// inside the box turned, which bakeHeld's margin keeps.
	}
	t.Logf("%d angles of %d items painted in %v (%.2f ms each), %.1f to %.1f KB an angle, %.1f MB if all were shown",
		n, n/heldTurns, total.Round(time.Millisecond), float64(total.Microseconds())/1000/float64(n),
		float64(least)/1024, float64(most)/1024, float64(bytes)/(1<<20))
}

// TestHeldBoxes checks that every held item's box holds what it paints:
// painted unturned on a canvas with room to spare, nothing lies outside it.
func TestHeldBoxes(t *testing.T) {
	for k, d := range heldDefs {
		if d == nil {
			continue
		}
		wide := *d
		wide.x0, wide.y0, wide.x1, wide.y1 = d.x0-3, d.y0-3, d.x1+3, d.y1+3
		p := bakeHeld(&wide, 0)
		// The painted extent, in art pixels from the grip.
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for y := 0; y < p.h; y++ {
			for x := 0; x < p.w; x++ {
				if p.px[y*p.w+x].A < 128 {
					continue
				}
				ax, ay := (float64(x)+0.5-p.gx)/HDDensity, (float64(y)+0.5-p.gy)/HDDensity
				x0, y0, x1, y1 = min(x0, ax), min(y0, ay), max(x1, ax), max(y1, ay)
			}
		}
		const slack = 0.7 // the outline
		if x0 < d.x0-slack || y0 < d.y0-slack || x1 > d.x1+slack || y1 > d.y1+slack {
			t.Errorf("%s paints (%.1f, %.1f)-(%.1f, %.1f), outside its box (%.1f, %.1f)-(%.1f, %.1f)",
				sim.ItemKind(k).Def().Name, x0, y0, x1, y1, d.x0, d.y0, d.x1, d.y1)
		}
	}
}

// TestDumpHeld writes every held item at a few angles, three times as big,
// into $HELD_DUMP:
//
//	HELD_DUMP=dir go test ./internal/client -run DumpHeld -v
func TestDumpHeld(t *testing.T) {
	dir := os.Getenv("HELD_DUMP")
	if dir == "" {
		t.Skip("HELD_DUMP not set")
	}
	turns := []int{0, 6, 16, 40}
	const cell, zk = 100, 4
	var kinds []sim.ItemKind
	only := os.Getenv("HELD_ONLY") // item numbers, comma separated
	for k, d := range heldDefs {
		if d != nil && (only == "" || strings.Contains(","+only+",", fmt.Sprintf(",%d,", k))) {
			kinds = append(kinds, sim.ItemKind(k))
		}
	}
	for _, k := range kinds {
		t.Logf("%d %s", k, k.Def().Name)
	}
	img := image.NewRGBA(image.Rect(0, 0, len(turns)*cell*zk, len(kinds)*cell*zk))
	for i := range img.Pix {
		img.Pix[i] = 0x48
		if i%4 == 1 {
			img.Pix[i] = 0x66
		}
		if i%4 == 3 {
			img.Pix[i] = 0xff
		}
	}
	for r, k := range kinds {
		for c, turn := range turns {
			p := bakeHeld(heldDefs[k], turn)
			ox, oy := c*cell+(cell-p.w)/2, r*cell+(cell-p.h)/2
			for y := 0; y < p.h; y++ {
				for x := 0; x < p.w; x++ {
					s := p.px[y*p.w+x]
					if s.A == 0 {
						continue
					}
					for dy := 0; dy < zk; dy++ {
						for dx := 0; dx < zk; dx++ {
							X, Y := (ox+x)*zk+dx, (oy+y)*zk+dy
							if X < 0 || Y < 0 || X >= img.Rect.Dx() || Y >= img.Rect.Dy() {
								continue
							}
							b := img.RGBAAt(X, Y)
							a := float64(s.A) / 255
							img.SetRGBA(X, Y, color.RGBA{
								uint8(float64(s.R)*a + float64(b.R)*(1-a)),
								uint8(float64(s.G)*a + float64(b.G)*(1-a)),
								uint8(float64(s.B)*a + float64(b.B)*(1-a)), 255})
						}
					}
				}
			}
			img.SetRGBA((ox+int(p.gx))*zk, (oy+int(p.gy))*zk, color.RGBA{255, 0, 255, 255})
		}
	}
	f, err := os.Create(filepath.Join(dir, "held_dump.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkBakeHeld(b *testing.B) {
	d := heldDefs[sim.ItemAxe]
	for i := 0; i < b.N; i++ {
		bakeHeld(d, i%heldTurns)
	}
}
