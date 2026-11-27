package client

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"survivalgame/internal/sim"
)

var coverKindNames = [numCoverKinds]string{"green grass", "tall grass", "tall grass, cut", "leafy clump", "fern", "flowers", "broad leaves", "berry bush", "berry bush, picked"}

// coverSheet lays out every drawing of the kinds given for season s, each
// image pixel z pixels across, on the ground they grow on.
func coverSheet(heading string, kinds []coverKind, seasons []sim.Season, z int) *image.RGBA {
	const pad, label, head = 10, 16, 22
	type cell struct {
		c      *canvas
		ax, ay float64
		name   string
		s      sim.Season
	}
	var rows [][]cell
	var titles []string
	for _, s := range seasons {
		for _, k := range kinds {
			var r []cell
			for v := range coverVariants[k] {
				c, ax, ay := genCoverCanvas(k, v, s)
				r = append(r, cell{c, ax, ay, "", s})
			}
			rows = append(rows, r)
			titles = append(titles, coverKindNames[k]+" - "+[]string{"spring", "summer", "autumn", "winter"}[s])
		}
	}
	W, H := 0, pad+head
	for _, r := range rows {
		w, h := pad, 0
		for _, c := range r {
			w += c.c.w*z + 2*pad + pad
			h = max(h, c.c.h*z+2*pad)
		}
		W, H = max(W, w), H+head+h+pad
	}
	W = max(W, 8*len(heading)+2*pad)
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{rkBgSheet}, image.Point{}, draw.Src)
	rkText(img, pad, pad+12, heading, rkTitleCol)
	y := pad + head
	for i, r := range rows {
		rkText(img, pad, y+14, titles[i], rkTitleCol)
		y += head
		h := 0
		for _, c := range r {
			h = max(h, c.c.h*z+2*pad)
		}
		x := pad
		for _, c := range r {
			bw := c.c.w*z + 2*pad
			// The ground, at the canvas's own pixel size.
			for py := 0; py < h; py++ {
				for px := 0; px < bw; px++ {
					hx, hy := (x+px)/z, (y+py)/z
					u, v := float64(hx)/hdTW, float64(hy)/hdTH
					var g color.RGBA
					if c.s == sim.Winter {
						g = snowGround(hx, hy, u, v)
					} else {
						g = grassGround(hx, hy, u, v)
					}
					img.SetRGBA(x+px, y+py, g)
				}
			}
			oy := y + h - pad - c.c.h*z
			for py := 0; py < c.c.h*z; py++ {
				for px := 0; px < c.c.w*z; px++ {
					p := c.c.px[(py/z)*c.c.w+px/z]
					if p.A == 0 {
						continue
					}
					bg := img.RGBAAt(x+pad+px, oy+py)
					img.SetRGBA(x+pad+px, oy+py, withAlpha(mix(bg, withAlpha(p, 255), float64(p.A)/255), 255))
				}
			}
			x += bw + pad
		}
		y += h + pad
	}
	return img
}

// TestDumpCover writes, to the folder $COVER_DUMP, every drawing of every
// small plant three times as big (cover_variants.png) and the four seasons
// of each kind (cover_seasons.png):
//
//	COVER_DUMP=/tmp/cover go test ./internal/client -run DumpCover
func TestDumpCover(t *testing.T) {
	dir := os.Getenv("COVER_DUMP")
	if dir == "" {
		t.Skip("set COVER_DUMP to a folder")
	}
	var all []coverKind
	for k := range numCoverKinds {
		all = append(all, k)
	}
	rkSave(t, filepath.Join(dir, "cover_variants.png"), coverSheet("every small plant, each drawing, summer, x3", all, []sim.Season{sim.Summer}, 3))
	rkSave(t, filepath.Join(dir, "cover_seasons.png"), coverSheet("the seasons, x2", []coverKind{cvTall, cvGrass, cvFern, cvFlowers, cvClump, cvBush}, []sim.Season{sim.Spring, sim.Summer, sim.Autumn, sim.Winter}, 2))
}

func TestCoverDrawingsStandOnTheirFoot(t *testing.T) {
	for s := range sim.NumSeasons {
		for k := range numCoverKinds {
			for v := range coverVariants[k] {
				c, ax, ay := genCoverCanvas(k, v, s)
				if ax < 0 || ay < 0 || ax > float64(c.w) || ay > float64(c.h) {
					t.Fatalf("kind %d variant %d season %d: anchor (%.0f, %.0f) off its %dx%d canvas", k, v, s, ax, ay, c.w, c.h)
				}
				n, top := 0, c.h
				for y := range c.h {
					for x := range c.w {
						if c.px[y*c.w+x].A >= 200 {
							n++
							top = min(top, y)
						}
					}
				}
				if n < 150 {
					t.Errorf("kind %d variant %d season %d: only %d solid pixels", k, v, s, n)
				}
				if top == 0 {
					t.Errorf("kind %d variant %d season %d: touches the top of its canvas", k, v, s)
				}
			}
		}
	}
}
