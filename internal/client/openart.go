package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Sprites and icons for the open world's resources: cacti in the desert,
// reeds and mushrooms in the swamp, coal seams in caves and on the ash.

type openArt struct {
	cactus, cactusCut sprite
	reed, reedCut     sprite
	mushroom          sprite
	coal              sprite
	clams             sprite
}

var (
	cactusRamp = mkRamp(0x1d3b23, 0x285530, 0x387040, 0x4f8c4c, 0x6ea85c)
	reedRamp   = mkRamp(0x3a4423, 0x4e5a2d, 0x66733a, 0x828f4a)
	cattail    = mkRamp(0x3a2214, 0x5a3420, 0x7a4a2c)
	capRed     = mkRamp(0x6e1414, 0xa8221e, 0xd8402e, 0xf07a5a)
	capBrown   = mkRamp(0x4a2e1c, 0x6e4628, 0x92643a, 0xb88a56)
	coalRamp   = mkRamp(0x0d0c0f, 0x18161a, 0x242126, 0x343036, 0x4c474e)
	mushStem   = hex(0xe8e0cc)
)

func genOpenArt() openArt {
	return openArt{
		cactus:    genCactus(true),
		cactusCut: genCactus(false),
		reed:      genReeds(true),
		reedCut:   genReeds(false),
		mushroom:  genMushrooms(),
		coal:      genCoalSeam(),
		clams:     genClamBed(),
	}
}

var shellRamp = mkRamp(0x8a6a70, 0xb98f8c, 0xdcb7a8, 0xf3dccb, 0xfff4ea)

// shell draws one ribbed clam shell, hinge at the top.
func shell(c *canvas, cx, cy, r float64, rp ramp) {
	c.ellipse(cx, cy, r, r*0.8, func(x, y int, nx, ny float64) color.RGBA {
		if ny < -0.6 && math.Abs(nx) < 0.35 {
			return rp[0] // the hinge
		}
		l := lum(nx, ny)
		if int(math.Floor((math.Atan2(ny+0.9, nx)+math.Pi)*3))%2 == 0 {
			l -= 0.18 // ribs fanning out from the hinge
		}
		return rp.pick(l, x, y)
	})
}

// genClamBed is a few clams half buried in wet sand.
func genClamBed() sprite {
	c := newCanvas(20, 12)
	shell(c, 6, 6, 3.4, shellRamp)
	shell(c, 13.5, 5, 3, mkRamp(0x5e5a6e, 0x7d7890, 0xa29db4, 0xc8c3d8, 0xe8e4f2))
	shell(c, 10, 8.5, 2.6, shellRamp)
	c.outline(0.7)
	return mk(c, 10, 9)
}

// genCactus draws a saguaro with two arms, ribbed and spiny; cut, it is a
// short stump.
func genCactus(full bool) sprite {
	c := newCanvas(22, 34)
	const bx, by = 11, 31
	col := func(x, y int, x0, x1 float64) color.RGBA {
		f := (float64(x) + 0.5 - x0) / (x1 - x0)
		l := 0.85 - f*0.65
		if (x-int(x0))%2 == 1 {
			l -= 0.12 // ribs
		}
		if hashf(x, y, 170) < 0.07 {
			l += 0.35 // a spine catching the light
		}
		return cactusRamp.pick(l, x, y)
	}
	limb := func(x0, x1 float64, top, bottom int) {
		for y := top; y <= bottom; y++ {
			for x := int(x0); x < int(math.Ceil(x1)); x++ {
				if y == top && (x == int(x0) || x == int(math.Ceil(x1))-1) {
					continue // rounded end
				}
				c.set(x, y, col(x, y, x0, x1))
			}
		}
	}
	if !full {
		limb(bx-3, bx+3, by-7, by)
		c.ellipse(bx, by-7, 3, 1.3, func(x, y int, nx, ny float64) color.RGBA { return hex(0xa8c48a) })
		c.outline(0.75)
		return mk(c, bx, by)
	}
	limb(bx-3, bx+3, 4, by)
	limb(3, 7, 10, 17) // left arm, rising
	limb(3, bx-2, 15, 18)
	limb(15, 19, 13, 21) // right arm
	limb(bx+2, 19, 19, 22)
	c.outline(0.75)
	return mk(c, bx, by)
}

// genReeds draws a clump of tall stems, some topped with brown cattails.
func genReeds(full bool) sprite {
	c := newCanvas(20, 28)
	const bx, by = 10, 26
	for i := 0; i < 8; i++ {
		x0 := float64(bx-5) + float64(i)*1.4
		h := 15 + hashf(i, 0, 180)*8
		lean := (x0 - bx) * 0.25
		if !full {
			h = 3
		}
		for s := 0; s <= 30; s++ {
			t := float64(s) / 30
			x, y := int(math.Floor(x0+lean*t)), int(math.Floor(by-h*t))
			c.set(x, y, reedRamp.pick(0.2+t*0.7-float64(i%2)*0.15, x, y))
		}
		if full && i%2 == 0 {
			tx, ty := x0+lean, by-h+3
			c.ellipse(tx+0.5, ty, 1.3, 2.8, func(x, y int, nx, ny float64) color.RGBA { return cattail.pick(lum(nx, ny), x, y) })
		}
	}
	c.outline(0.5)
	return mk(c, bx, by)
}

// genMushrooms draws a red, white-spotted mushroom beside a smaller brown one.
func genMushrooms() sprite {
	c := newCanvas(18, 14)
	shroom := func(cx, top, h, r float64, rp ramp, spots bool) {
		c.rect(int(cx-1), int(top+r*0.5), 2, int(h), solid(mushStem))
		c.ellipse(cx, top+r*0.6, r, r*0.65, func(x, y int, nx, ny float64) color.RGBA {
			if ny > 0.35 {
				return rp[0] // the gills' shade under the rim
			}
			if spots && hashf(x, y, 190) < 0.18 {
				return colWhite
			}
			return rp.pick(lum(nx, ny), x, y)
		})
	}
	shroom(11.5, 2, 9, 4.5, capRed, true)
	shroom(5.5, 6, 6, 3.2, capBrown, false)
	c.outline(0.75)
	return mk(c, 9, 12)
}

// genCoalSeam is a seam of coal in its rock (see genCoalCanvas).
func genCoalSeam() sprite {
	c, ax, ay := genCoalCanvas(0)
	return mk(c, ax, ay)
}

// genOpenIcons draws the bag icons of cactus flesh, mushrooms and coal.
func genOpenIcons(icons *[sim.NumItems]*ebiten.Image) {
	flesh := newCanvas(16, 16)
	flesh.ellipse(8, 8, 6.5, 6.5, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		switch {
		case d > 0.7:
			return cactusRamp.pick(lum(nx, ny)*0.8, x, y) // the skin
		case hashf(x, y, 195) < 0.1:
			return hex(0x5c7a3a) // seeds
		default:
			return mix(hex(0xb6d08a), hex(0xdbe8b4), lum(nx, ny))
		}
	})
	flesh.outline(0.8)
	icons[sim.ItemCactus] = flesh.image()

	shroom := newCanvas(16, 16)
	shroom.rect(6, 7, 4, 8, func(x, y int) color.RGBA { return mix(mushStem, hex(0xb8ab92), float64(x-6)/4) })
	shroom.ellipse(8, 6, 7, 5, func(x, y int, nx, ny float64) color.RGBA {
		if ny > 0.45 {
			return capRed[0]
		}
		if hashf(x, y, 196) < 0.16 {
			return colWhite
		}
		return capRed.pick(lum(nx, ny), x, y)
	})
	shroom.outline(0.8)
	icons[sim.ItemMushroom] = shroom.image()

	coal := newCanvas(16, 16)
	coal.polygon([]float64{3, 7, 7, 2.5, 13, 4, 14.5, 10, 10, 14, 4, 12.5}, func(x, y int) color.RGBA {
		l := 0.65 - float64(x+y)*0.025
		if (x*3+y*5)%7 == 0 {
			l += 0.25 // a facet catching light
		}
		return coalRamp.pick(l, x, y)
	})
	coal.set(6, 5, hex(0x9a94a6))
	coal.set(7, 4, hex(0x9a94a6))
	coal.outline(0.85)
	icons[sim.ItemCoal] = coal.image()

	clam := newCanvas(16, 16)
	shell(clam, 6, 9, 4.6, shellRamp)
	shell(clam, 10.5, 6.5, 4.2, mkRamp(0x5e5a6e, 0x7d7890, 0xa29db4, 0xc8c3d8, 0xe8e4f2))
	clam.outline(0.8)
	icons[sim.ItemClam] = clam.image()

	roast := newCanvas(16, 16)
	roast.ellipse(8, 10, 6.5, 4.2, func(x, y int, nx, ny float64) color.RGBA {
		if nx*nx+ny*ny < 0.45 {
			return roastRamp.pick(lum(nx, ny)*0.8+0.2, x, y) // the cooked meat
		}
		return shellRamp.pick(lum(nx, ny)*0.7, x, y) // the open shell's rim
	})
	roast.set(6, 9, charCol)
	roast.set(9, 11, charCol)
	roast.outline(0.85)
	icons[sim.ItemRoastClam] = roast.image()
}

// --- caves and lava in the light ------------------------------------------------

// tileMask writes one pixel per tile of the view (plus a margin) into *img:
// opaque where match is true. It reports false, leaving *img alone, when no
// tile in view matches, so the caller can skip drawing it.
func (g *Game) tileMask(img **ebiten.Image, cam camera, w, h int, match func(sim.Tile) bool) (tx0, ty0 int, ok bool) {
	tx0, ty0 = int(math.Floor(cam.ox/TileW))-2, int(math.Floor(cam.oy/TileH))-2
	nw, nh := w/TileW+5, h/TileH+5
	px := make([]byte, nw*nh*4)
	any := false
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			if match(g.tileAt(tx0+x, ty0+y)) {
				i := (y*nw + x) * 4
				px[i], px[i+1], px[i+2], px[i+3] = 255, 255, 255, 255
				any = true
			}
		}
	}
	if !any {
		return 0, 0, false
	}
	if *img == nil || (*img).Bounds().Dx() != nw || (*img).Bounds().Dy() != nh {
		if *img != nil {
			(*img).Deallocate()
		}
		*img = ebiten.NewImage(nw, nh)
	}
	(*img).WritePixels(px)
	return tx0, ty0, true
}

// maskOp places a tile mask over the view, one mask pixel per tile, with
// linear filtering so it fades smoothly across tile borders.
func maskOp(cam camera, tx0, ty0 int) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(TileW, TileH)
	op.GeoM.Translate(float64(tx0)*TileW-cam.ox, float64(ty0)*TileH-cam.oy)
	op.Filter = ebiten.FilterLinear
	return op
}

const caveDarkness = 0.94 // caves are darker than the deepest night

func isCave(t sim.Tile) bool { return t == sim.TileCave }
func isLava(t sim.Tile) bool { return t == sim.TileLava }

// lavaLight draws lava's glow: a hole in the darkness when target is the
// darkness layer (punch), or warm light added to the scene otherwise.
func (g *Game) lavaLight(dst drawTarget, cam camera, tx0, ty0 int, punch bool, k float32) {
	for _, o := range [][2]float64{{0, 0}, {-0.5, 0}, {0.5, 0}, {0, -0.5}, {0, 0.5}} {
		op := maskOp(cam, tx0, ty0)
		op.GeoM.Translate(o[0]*TileW, o[1]*TileH)
		if punch {
			op.Blend = ebiten.BlendDestinationOut
			op.ColorScale.ScaleAlpha(0.55)
		} else {
			op.ColorScale.Scale(k, k*0.42, k*0.1, k)
			op.Blend = ebiten.BlendLighter
		}
		dst.DrawImage(g.lavaMask, op)
	}
}
