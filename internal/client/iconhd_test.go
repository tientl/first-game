package client

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"survivalgame/internal/sim"
)

// iconTestSizes are the sizes HD icons are baked at: 24 to 64 for small
// slots and ingredients, 96 for a 1080p bag, 104 to 192 for bigger screens.
var iconTestSizes = []int{24, 32, 48, 64, 72, 96, 128, 192}

// TestHDIcons bakes every HD icon at every size: it must cover a fair
// share of its box and stay inside it, so its outline is never cut off.
func TestHDIcons(t *testing.T) {
	n := 0
	for k, f := range hdIcons {
		if f == nil {
			continue
		}
		n++
		kind := sim.ItemKind(k)
		if !kind.Valid() {
			t.Errorf("item %d has an HD icon but is no item", k)
			continue
		}
		for _, px := range iconTestSizes {
			c := bakeIcon(f, px)
			if c.w != px || c.h != px {
				t.Fatalf("%s at %d: baked %dx%d", kind.Def().Name, px, c.w, c.h)
			}
			solid, edge := 0, 0
			for y := 0; y < px; y++ {
				for x := 0; x < px; x++ {
					if c.px[y*px+x].A < 128 {
						continue
					}
					solid++
					if x == 0 || y == 0 || x == px-1 || y == px-1 {
						edge++
					}
				}
			}
			if share := float64(solid) / float64(px*px); share < 0.08 || share > 0.85 {
				t.Errorf("%s at %d: covers %.0f%% of its box", kind.Def().Name, px, 100*share)
			}
			if edge > 0 {
				t.Errorf("%s at %d: %d solid pixels on the box's edge (cut off)", kind.Def().Name, px, edge)
			}
		}
	}
	if n < len(iconPilot) {
		t.Errorf("%d HD icons registered, want at least the %d of the pilot", n, len(iconPilot))
	}
}

// TestIconOldSize: the design box is the old icons' size, so an HD icon
// takes an old one's place exactly.
func TestIconOldSize(t *testing.T) {
	if iconOldPx != int(iconBox) {
		t.Fatalf("the old icons are %d across, the design box %v", iconOldPx, iconBox)
	}
}

// BenchmarkBakeIcon times baking the pilot's icons at a 1080p bag's size.
func BenchmarkBakeIcon(b *testing.B) {
	for b.Loop() {
		for _, k := range iconPilot {
			bakeIcon(hdIcons[k], 96)
		}
	}
}

// TestDumpIcons writes a sheet of the HD icons when ICON_DUMP is set to a
// directory: each at 96 pixels (a 1080p bag slot) on the slot's colour, then
// three times bigger, and how long baking each took. It needs no game loop,
// so it is the quick way to see an icon while drawing it:
//
//	ICON_DUMP=dir go test ./internal/client -run DumpIcons -v
//
// ICON_ONLY=<item number> keeps one item.
func TestDumpIcons(t *testing.T) {
	dir := os.Getenv("ICON_DUMP")
	if dir == "" {
		t.Skip("set ICON_DUMP to a directory")
	}
	var items []sim.ItemKind
	for _, k := range iconSheetItems() {
		if only := os.Getenv("ICON_ONLY"); only == "" || only == itoa(int(k)) {
			items = append(items, k)
		}
	}
	const n, z, gap = 96, 3, 12
	cols := min(len(items), 4)
	rows := (len(items) + cols - 1) / cols
	cell := n + gap + n*z + gap
	img := image.NewRGBA(image.Rect(0, 0, cols*cell+gap, rows*(n*z+gap)+gap))
	bg := color.RGBA{24, 26, 36, 255}
	slot := color.RGBA{34, 37, 50, 255}
	fill := func(x0, y0, w, h int, c color.RGBA) {
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	fill(0, 0, img.Bounds().Dx(), img.Bounds().Dy(), bg)
	for i, k := range items {
		t0 := time.Now()
		c := bakeIcon(hdIcons[k], n)
		t.Logf("%-3d %-20s baked at %d in %v", k, k.Def().Name, n, time.Since(t0).Round(10*time.Microsecond))
		x0, y0 := gap+(i%cols)*cell, gap+(i/cols)*(n*z+gap)
		fill(x0, y0, n, n, slot)
		fill(x0+n+gap, y0, n*z, n*z, slot)
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				p := c.px[y*n+x]
				if p.A == 0 {
					continue
				}
				put := func(px, py int) {
					b := img.RGBAAt(px, py)
					img.SetRGBA(px, py, mix(b, withAlpha(p, 255), float64(p.A)/255))
				}
				put(x0+x, y0+y)
				for dy := 0; dy < z; dy++ {
					for dx := 0; dx < z; dx++ {
						put(x0+n+gap+x*z+dx, y0+y*z+dy)
					}
				}
			}
		}
	}
	path := filepath.Join(dir, "icons_hd.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", path)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for ; v > 0; v /= 10 {
		b = append([]byte{byte('0' + v%10)}, b...)
	}
	return string(b)
}
