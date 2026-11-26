package client

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

var sheetSand = color.RGBA{0xe0, 0xc5, 0x8c, 255}

// mangroveSheet is every look of the mangrove: on the sand and in the
// water, full grown and felled.
func mangroveSheet() []sheetRow {
	var rows []sheetRow
	for _, wet := range []bool{false, true} {
		bg := sheetSand
		if wet {
			bg = sheetWater
		}
		for _, full := range []bool{true, false} {
			r := sheetRow{bg: bg}
			for v := range mangroveVariants {
				c, ax, ay := genMangrove(v, full, wet)
				r.cells = append(r.cells, sheetCell{c, ax, ay})
			}
			rows = append(rows, r)
		}
	}
	return rows
}

// MANGROVE_DUMP=dir go test ./internal/client -run DumpMangroves -v writes
// the sheet of the mangroves there, as a 1080p screen shows them and three
// times that.
func TestDumpMangroves(t *testing.T) {
	dir := os.Getenv("MANGROVE_DUMP")
	if dir == "" {
		t.Skip("set MANGROVE_DUMP to a directory to write the sheets")
	}
	for k, name := range map[int]string{1: "mangroves_1x.png", 3: "mangroves_3x.png"} {
		if err := writeSheet(filepath.Join(dir, name), mangroveSheet(), k); err != nil {
			t.Fatal(err)
		}
	}
	// The roots close up, three times as big: on the sand, then in the water.
	for _, wet := range []bool{false, true} {
		bg, name := sheetSand, "mangrove_roots_sand.png"
		if wet {
			bg, name = sheetWater, "mangrove_roots_water.png"
		}
		var rows []sheetRow
		for v := range mangroveVariants {
			if v%2 == 0 {
				rows = append(rows, sheetRow{bg: bg})
			}
			c, ax, ay := genMangrove(v, true, wet)
			y0 := int((mgBase - 40) * fk)
			sub := newCanvas(c.w, c.h-y0)
			copy(sub.px, c.px[y0*c.w:])
			rows[len(rows)-1].cells = append(rows[len(rows)-1].cells, sheetCell{sub, ax, ay - float64(y0)})
		}
		if err := writeSheet(filepath.Join(dir, name), rows, 3); err != nil {
			t.Fatal(err)
		}
	}
}

// Every look of the mangrove is drawn, each different from the others,
// standing on its anchor with its roots on the ground round it, and
// nothing cut off by its canvas.
func TestMangroveLooks(t *testing.T) {
	var seen []*canvas
	for v := range mangroveVariants {
		for _, wet := range []bool{false, true} {
			for _, full := range []bool{true, false} {
				c, ax, ay := genMangrove(v, full, wet)
				if c.dens != HDDensity {
					t.Errorf("variant %d is not drawn at HD", v)
				}
				if ax != mgBX*fk || ay != mgBase*fk {
					t.Errorf("variant %d: anchor (%.0f, %.0f), not its ground point", v, ax, ay)
				}
				for x := 0; x < c.w; x++ {
					if c.px[x].A > 0 || c.px[(c.h-1)*c.w+x].A > 0 {
						t.Errorf("variant %d (wet %v, full %v) touches the top or bottom of its canvas", v, wet, full)
						break
					}
				}
				for y := 0; y < c.h; y++ {
					if c.px[y*c.w].A > 0 || c.px[y*c.w+c.w-1].A > 0 {
						t.Errorf("variant %d (wet %v, full %v) touches the side of its canvas", v, wet, full)
						break
					}
				}
				for _, o := range seen {
					same := true
					for i := range c.px {
						if c.px[i] != o.px[i] {
							same = false
							break
						}
					}
					if same {
						t.Errorf("variant %d (wet %v, full %v) looks like another", v, wet, full)
					}
				}
				seen = append(seen, c)
			}
		}
		for f := range 2 {
			roots := 0
			for _, ft := range mangroveLayoutOf(v).feet[f] {
				if !ft.pneu {
					roots++
				}
				if ft.y < -14 || ft.y > 8 || ft.x < -40 || ft.x > 40 {
					t.Errorf("variant %d: a root comes down at (%.1f, %.1f), far from the tree", v, ft.x, ft.y)
				}
			}
			if roots < 6+2*f { // a felled one has lost those that left the trunk higher up
				t.Errorf("variant %d (full %v) stands on only %d roots", v, f == 1, roots)
			}
		}
	}
}
