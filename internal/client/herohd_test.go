package client

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"survivalgame/internal/sim"
)

func hdBlit(img *image.RGBA, c *canvas, ox, oy, k int, flip bool) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			p := c.at(x, y)
			if p.A == 0 {
				continue
			}
			px := x
			if flip {
				px = c.w - 1 - x
			}
			for a := range k {
				for b := range k {
					X, Y := ox+px*k+a, oy+y*k+b
					if p.A == 255 {
						img.SetRGBA(X, Y, p)
					} else {
						img.SetRGBA(X, Y, mix(img.RGBAAt(X, Y), withAlpha(p, 255), float64(p.A)/255))
					}
				}
			}
		}
	}
}

// TestHeroHDSheet writes the trial hero's frames to the directory named by
// HEROHD_OUT: a sheet of everything beside the old figure, and each frame
// on its own for making animations.
func TestHeroHDSheet(t *testing.T) {
	dir := os.Getenv("HEROHD_OUT")
	if dir == "" {
		t.Skip("HEROHD_OUT not set")
	}
	const k = 4
	const cw, ch = 60, 64 // the part of a frame shown on the sheet
	const cx0, cy0 = hdOX - cw/2, hdOY + 4 - ch
	tint := playerColors[1]
	bg := hex(0x262438)
	dirs := []sim.Dir{sim.DirDown, sim.DirRight, sim.DirUp}
	names := []string{"idle", "run", "attack"}
	rows := 0
	for an := range numHDAnims {
		_ = an
		rows += len(dirs)
	}
	img := image.NewRGBA(image.Rect(0, 0, (8*cw+64)*k, rows*ch*k))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := bg
			if ((y / k) % ch) >= ch-6 {
				c = hex(0x2f2c46)
			}
			img.SetRGBA(x, y, c)
		}
	}
	row := 0
	for an := range numHDAnims {
		for _, d := range dirs {
			for i := range hdFrames[an] {
				fr := genHeroHD(tint, an, d, i, an == hdAttack)
				sub := newCanvas(cw, ch)
				nsub := newCanvas(cw, ch)
				for y := 0; y < ch; y++ {
					for x := 0; x < cw; x++ {
						sub.px[y*cw+x] = fr.col.at(x+cx0, y+cy0)
						nsub.px[y*cw+x] = fr.nrm.at(x+cx0, y+cy0)
					}
				}
				hdBlit(img, sub, i*cw*k, row*ch*k, k, false)
				one := image.NewRGBA(image.Rect(0, 0, hdW*k, hdH*k))
				for p := 0; p < len(one.Pix); p += 4 {
					one.Pix[p], one.Pix[p+1], one.Pix[p+2], one.Pix[p+3] = bg.R, bg.G, bg.B, 255
				}
				hdBlit(one, fr.col, 0, 0, k, false)
				writePNG(t, filepath.Join(dir, fmt.Sprintf("%s_%d_%d.png", names[an], d, i)), one)
				if an == hdIdle && i == 0 {
					nimg := image.NewRGBA(image.Rect(0, 0, cw*k, ch*k))
					hdBlit(nimg, nsub, 0, 0, k, false)
					writePNG(t, filepath.Join(dir, fmt.Sprintf("normal_%d.png", d)), nimg)
				}
			}
			row++
		}
	}
	// the old figure, at the same zoom
	for r, d := range dirs {
		for fr := range 4 {
			lookCell(img, sim.Look{}, d, fr, color.RGBA{tint.R, tint.G, tint.B, 255}, (6*cw+10+fr*26)*k, (r*ch+ch-6-bodyFeet-2)*k, k)
		}
	}
	writePNG(t, filepath.Join(dir, "sheet.png"), img)
}
