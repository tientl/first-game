package client

import (
	"image"
	"image/color"
	"math"
)

// The application icon: a long-necked dinosaur standing on a small island
// beside a palm, its head against the sun, inside a rounded tile with a dark
// rim. The window, the taskbar and the .exe all use it (cmd/icon exports the
// files).
//
// A big icon scaled down turns to mush, so each small size is its own drawing:
// 16 is a pixel map typed by hand, 24 and 32 are the scene laid out again on
// their own pixel grids with the small parts left out, and 48 and up share a
// 64-unit design that gains detail as it grows.

// GameIconSizes are the sizes GameIcons returns, in order.
var GameIconSizes = []int{16, 24, 32, 48, 64, 128, 256}

// GameIcons is the application icon at every size in GameIconSizes, for
// ebiten.SetWindowIcon and for the .ico file.
func GameIcons() []image.Image {
	imgs := make([]image.Image, 0, len(GameIconSizes))
	for _, n := range GameIconSizes {
		imgs = append(imgs, gameIconImage(gameIconCanvas(n)))
	}
	return imgs
}

func gameIconCanvas(n int) *canvas {
	switch {
	case n <= 16:
		return gameIconFromMap(gameIcon16)
	case n <= 24:
		return gameIconScene(gameIconSpec24, 1)
	case n <= 32:
		return gameIconScene(gameIconSpec32, 1)
	case n <= 48:
		return gameIconScene(gameIconSpec64, 0.75)
	case n <= 64:
		return gameIconScene(gameIconSpec64, 1)
	case n <= 128:
		return gameIconScene(gameIconSpec64, 2)
	}
	// The biggest is the 128 drawing with every pixel doubled, so that it
	// stays pixel art instead of turning smooth.
	return gameIconDoubled(gameIconScene(gameIconSpec64, 2))
}

func gameIconImage(c *canvas) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, c.w, c.h))
	for i, p := range c.px {
		if p.A < 128 {
			continue // keep the edge hard, as pixel art is
		}
		img.Pix[4*i], img.Pix[4*i+1], img.Pix[4*i+2], img.Pix[4*i+3] = p.R, p.G, p.B, 255
	}
	return img
}

func gameIconDoubled(c *canvas) *canvas {
	d := newCanvas(c.w*2, c.h*2)
	for y := 0; y < d.h; y++ {
		for x := 0; x < d.w; x++ {
			d.px[y*d.w+x] = c.px[(y/2)*c.w+x/2]
		}
	}
	return d
}

// --- palette ----------------------------------------------------------------

var (
	gameIconDino  = mkRamp(0x1c4a2a, 0x2c6d33, 0x45913b, 0x6bb347, 0x9bd25c, 0xc8ea86)
	gameIconSky   = mkRamp(0x2f86d8, 0x4aa0e4, 0x6dbaee, 0x97d3f4, 0xc4e9f8)
	gameIconSea   = mkRamp(0x1b55a6, 0x2268b8, 0x2b7dca, 0x3f96d8)
	gameIconGlint = hex(0x8fd4f4)
	gameIconFoam  = hex(0xe9f8ff)
	gameIconSun   = hex(0xffd44a)
	gameIconSunIn = hex(0xffec94)
	gameIconCloud = hex(0xf6fcff)
	gameIconWhite = hex(0xffffff)
)

// --- 16 x 16 ----------------------------------------------------------------

// gameIcon16 is the taskbar icon, typed pixel by pixel: at this size there is
// room for a head, a neck, a palm and a strip of sand, and nothing else.
var gameIcon16 = []string{
	"...##########...",
	".##sssssssyyy##.",
	".#ssssssddddyy#.",
	"#ssssssdGGGGdyy#",
	"#ssssssdG#GGGdy#",
	"#PPsPPsdgggggdy#",
	"#ppPppsdgGdddyy#",
	"#pSTtpSdgGdyyyS#",
	"#SSTtSSdgGdyySS#",
	"#SSSTtSdgGgdSSS#",
	"#SNNTtNdgGggdNS#",
	"#bnnnnnnnnnnnnb#",
	"#bbwwwwwwwwwwbb#",
	".#bBBbbbbbbBBb#.",
	".##bbbbBBbbbb##.",
	"...##########...",
}

var gameIconMapColours = map[byte]color.RGBA{
	'#': colInk,
	's': gameIconSky[1],
	'S': gameIconSky[3],
	'y': gameIconSun,
	'd': gameIconDino[0],
	'g': gameIconDino[2],
	'G': gameIconDino[4],
	'p': palmRamp[1],
	'P': palmRamp[3],
	't': palmTrunk[1],
	'T': palmTrunk[3],
	'N': sandRamp[4],
	'n': sandRamp[1],
	'b': gameIconSea[1],
	'B': gameIconSea[3],
	'w': gameIconFoam,
}

func gameIconFromMap(rows []string) *canvas {
	c := newCanvas(len(rows[0]), len(rows))
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			if col, ok := gameIconMapColours[row[x]]; ok {
				c.put(x, y, col)
			}
		}
	}
	return c
}

// --- the scene --------------------------------------------------------------

// gameIconSpec lays the scene out in design units; a unit is a pixel times
// the scale the scene is drawn at.
type gameIconSpec struct {
	n        int     // the tile is n by n units
	radius   float64 // of the tile's corners
	borderPx [4]int  // the rim, in pixels, at detail 0 to 3
	horizon  float64
	sun      [3]float64 // x, y, radius
	sand     [4]float64 // x, y, rx, ry
	grass    [4]float64
	body     [4]float64
	legs     [5]float64 // x of the hind and the front leg, top, bottom, radius; radius 0: none
	tail     [5]float64 // from x, y to x, y, radius at the root; radius 0: none
	neck     [8]float64 // a cubic curve from the shoulder to the head
	neckR    [2]float64 // its radius at the shoulder and at the head
	head     [4]float64
	snout    [4]float64
	eye      [3]int     // x, y, size
	mouth    [4]float64 // a line; all zero: none
	palmBase [2]float64
	palmTop  [2]float64
	trunkR   [2]float64
	frond    [2]float64 // length and radius
	fronds   int
	detail   int // of the scene at scale 1: 0 bare, 1 some, 2 full
}

var gameIconSpec64 = gameIconSpec{
	n: 64, radius: 13, borderPx: [4]int{1, 2, 2, 4}, horizon: 43,
	sun:   [3]float64{45, 19, 13.5},
	sand:  [4]float64{32, 52.5, 26, 7.5},
	grass: [4]float64{31, 49, 21, 4.6},
	body:  [4]float64{31, 39, 12.5, 8.5},
	legs:  [5]float64{25, 37, 42, 49.5, 3.2},
	tail:  [5]float64{22, 39, 8, 47.5, 4.5},
	neck:  [8]float64{38, 37, 45, 33, 40, 23, 45.5, 14},
	neckR: [2]float64{5.6, 3.6},
	head:  [4]float64{47.5, 12.5, 6.5, 4.6},
	snout: [4]float64{52, 13.5, 5, 3.6},
	eye:   [3]int{47, 10, 2}, mouth: [4]float64{51, 15.5, 56, 14.5},
	palmBase: [2]float64{16.5, 50}, palmTop: [2]float64{14, 22.5},
	trunkR: [2]float64{2.5, 1.7}, frond: [2]float64{12.5, 2.3}, fronds: 7,
	detail: 2,
}

// gameIconSpec32 is the title-bar size: the same scene set on a 32-pixel
// grid, with a bigger head and a thicker neck than a scaled-down 64 would
// have, and without the tail, the mouth, the cloud and the spots.
var gameIconSpec32 = gameIconSpec{
	n: 32, radius: 6.5, borderPx: [4]int{1, 1, 1, 1}, horizon: 21,
	sun:      [3]float64{22.5, 9.5, 7.5},
	sand:     [4]float64{16, 26, 13.5, 4},
	grass:    [4]float64{15.5, 24.2, 10, 2.2},
	body:     [4]float64{15.5, 19.5, 6.5, 4.4},
	legs:     [5]float64{12.5, 18.5, 21, 24.5, 1.7},
	neck:     [8]float64{19, 18.5, 23, 16, 20, 11, 22.5, 7.5},
	neckR:    [2]float64{3, 2.2},
	head:     [4]float64{23.5, 6.5, 3.6, 2.7},
	snout:    [4]float64{26, 7, 2.7, 2.1},
	eye:      [3]int{23, 5, 1},
	palmBase: [2]float64{8, 25}, palmTop: [2]float64{7, 11.5},
	trunkR: [2]float64{1.3, 1}, frond: [2]float64{6.5, 1.3}, fronds: 5,
}

// gameIconSpec24 is cut down further: no legs and no grass.
var gameIconSpec24 = gameIconSpec{
	n: 24, radius: 5, borderPx: [4]int{1, 1, 1, 1}, horizon: 16,
	sun:      [3]float64{17, 7, 5.5},
	sand:     [4]float64{12, 19.5, 10, 3},
	body:     [4]float64{11.5, 15, 4.6, 3.4},
	neck:     [8]float64{14, 14.5, 17, 12.5, 15, 8.5, 17, 6},
	neckR:    [2]float64{2.3, 1.8},
	head:     [4]float64{17.5, 5, 3, 2.2},
	snout:    [4]float64{19.5, 5.5, 2.2, 1.7},
	eye:      [3]int{17, 4, 1},
	palmBase: [2]float64{6, 19}, palmTop: [2]float64{5.5, 8.5},
	trunkR: [2]float64{1, 0.8}, frond: [2]float64{4.8, 1.1}, fronds: 5,
}

// gameIconScene draws the scene of sp at scale s.
func gameIconScene(sp gameIconSpec, s float64) *canvas {
	detail := sp.detail
	if s >= 2 {
		detail++
	} else if s < 1 {
		detail--
	}
	detail = max(0, min(3, detail))
	layer := func() *canvas { return newScaledCanvas(sp.n, sp.n, s) }
	outline := func(c *canvas, strength float64) {
		c.outline(strength)
		if s >= 2 {
			c.outline(1) // a two-pixel rim, to match the bigger pixels
		}
	}

	// Sky, sun and sea.
	bg := layer()
	horizonPx := sp.horizon * s
	for y := 0; y < bg.h; y++ {
		for x := 0; x < bg.w; x++ {
			fy := float64(y) + 0.5
			if fy < horizonPx {
				// Flat above, paling in bands towards the horizon.
				t := max(0, fy/horizonPx-0.45) / 0.55
				if detail >= 3 {
					bg.put(x, y, gameIconSky[1:].pick(t, x/2, y/2))
				} else {
					bg.put(x, y, gameIconSky.at(1+int(t*float64(len(gameIconSky)-1))))
				}
				continue
			}
			t := (fy - horizonPx) / (float64(bg.h) - horizonPx)
			bg.put(x, y, gameIconSea.at(len(gameIconSea)-1-int(t*float64(len(gameIconSea)))))
		}
	}
	bg.ellipse(sp.sun[0], sp.sun[1], sp.sun[2], sp.sun[2], func(x, y int, nx, ny float64) color.RGBA {
		if float64(y)+0.5 >= horizonPx {
			return color.RGBA{}
		}
		if nx*nx+ny*ny < 0.6 {
			return gameIconSunIn
		}
		return gameIconSun
	})
	if detail >= 2 {
		for _, e := range [][4]float64{{31, 8.5, 5.5, 2}, {34, 7, 3.2, 2.2}, {28.5, 7.5, 2.6, 1.6}} {
			bg.ellipse(e[0], e[1], e[2], e[3], func(int, int, float64, float64) color.RGBA { return gameIconCloud })
		}
		bg.rect(26, 9, 11, 1, solid(gameIconSky[4]))
	}
	if detail >= 1 {
		// Glints on the water: short dashes, more of them further out.
		step := max(1, int(math.Round(s)))
		for y := int(horizonPx) + step; y < bg.h; y += 2 * step {
			for x := 0; x < bg.w; x += step {
				if hashf(x/(4*step), y, 77) < 0.3 {
					for i := 0; i < step; i++ {
						bg.put(x, y+i, gameIconGlint)
					}
				}
			}
		}
	}
	// Surf around the island.
	bg.ellipse(sp.sand[0], sp.sand[1]+0.7, sp.sand[2]+1.6, sp.sand[3]+1, func(x, y int, _, _ float64) color.RGBA {
		if float64(y)+0.5 < horizonPx {
			return color.RGBA{}
		}
		return gameIconFoam
	})

	// The island.
	isle := layer()
	isle.ellipse(sp.sand[0], sp.sand[1], sp.sand[2], sp.sand[3], func(x, y int, nx, ny float64) color.RGBA {
		switch {
		case ny > 0.55:
			return sandRamp[0]
		case ny > 0.05:
			return sandRamp[2]
		}
		return sandRamp[4]
	})
	if sp.grass[2] > 0 {
		isle.ellipse(sp.grass[0], sp.grass[1], sp.grass[2], sp.grass[3], func(x, y int, nx, ny float64) color.RGBA {
			switch {
			case ny > 0.45:
				return grassRamp[0]
			case ny > -0.35 || nx > 0.6:
				return grassRamp[2]
			}
			return grassRamp[4]
		})
	}
	outline(isle, 0.72)
	bg.over(isle, 0, 0)

	// The dinosaur: legs and tail first, then the body, the neck and the head.
	dino := layer()
	skin := func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		return gameIconDino.at(int((0.08 + 0.92*l) * float64(len(gameIconDino))))
	}
	if r := sp.legs[4]; r > 0 {
		for _, lx := range sp.legs[:2] {
			gameIconTube(dino, gameIconLine(lx, sp.legs[2], lx, sp.legs[3]), r, r,
				func(x, y int, nx, ny, _ float64) color.RGBA { return skin(x, y, nx*0.9+0.25, 0.35) })
		}
	}
	if r := sp.tail[4]; r > 0 {
		gameIconTube(dino, gameIconLine(sp.tail[0], sp.tail[1], sp.tail[2], sp.tail[3]), r, 0.8,
			func(x, y int, nx, ny, _ float64) color.RGBA { return skin(x, y, nx, ny) })
	}
	dino.ellipse(sp.body[0], sp.body[1], sp.body[2], sp.body[3], skin)
	nk := sp.neck
	gameIconTube(dino, func(t float64) (float64, float64) {
		u := 1 - t
		a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
		return a*nk[0] + b*nk[2] + c*nk[4] + d*nk[6], a*nk[1] + b*nk[3] + c*nk[5] + d*nk[7]
	}, sp.neckR[0], sp.neckR[1], func(x, y int, nx, ny, _ float64) color.RGBA { return skin(x, y, nx, ny*0.5) })
	// The head is lit more evenly than a ball would be, to keep the face bright.
	face := func(x, y int, nx, ny float64) color.RGBA { return skin(x, y, nx*0.5-0.1, ny*0.75) }
	dino.ellipse(sp.snout[0], sp.snout[1], sp.snout[2], sp.snout[3], face)
	dino.ellipse(sp.head[0], sp.head[1], sp.head[2], sp.head[3], face)
	if detail >= 2 {
		// Darker blotches along the back, as the game's long-neck has.
		k := 3.4 * s
		for i, p := range dino.px {
			x, y := i%dino.w, i/dino.w
			if p.A == 0 || vnoise(float64(x)/k, float64(y)/k, 9) < 0.66 {
				continue
			}
			for j := 2; j < len(gameIconDino); j++ {
				if p == gameIconDino[j] {
					dino.px[i] = gameIconDino[j-1]
				}
			}
		}
	}
	if sp.mouth != ([4]float64{}) {
		dino.line(sp.mouth[0], sp.mouth[1], sp.mouth[2], sp.mouth[3],
			func(int, int, float64) color.RGBA { return gameIconDino[0] })
		if s >= 2 {
			dino.line(sp.mouth[0], sp.mouth[1]+0.5, sp.mouth[2], sp.mouth[3]+0.5,
				func(int, int, float64) color.RGBA { return gameIconDino[0] })
		}
	}
	// The eye is a white column and a dark one, looking the way the head
	// faces. It is sized in whole pixels so that no scale loses the pupil.
	efx, efy := dino.toPx(float64(sp.eye[0]), float64(sp.eye[1]))
	ex, ey := int(math.Floor(efx)), int(math.Floor(efy))
	ew := max(1, int(math.Round(float64(sp.eye[2])*s/2)))
	eh := max(1, int(math.Round(float64(sp.eye[2])*s)))
	for y := ey; y < ey+eh; y++ {
		for x := ex; x < ex+ew; x++ {
			dino.put(x, y, gameIconWhite)
			dino.put(x+ew, y, colInk)
		}
	}
	if detail >= 2 {
		dino.set(sp.eye[0]+7, sp.eye[1]+1, gameIconDino[0]) // a nostril
	}
	outline(dino, 0.85)
	bg.over(dino, 0, 0)

	// The palm, in front.
	palm := layer()
	bx, by, tx, ty := sp.palmBase[0], sp.palmBase[1], sp.palmTop[0], sp.palmTop[1]
	gameIconTube(palm, func(t float64) (float64, float64) {
		bow := math.Sin(t*math.Pi) * (by - ty) * 0.07
		return lerpf(bx, tx, t) + bow, lerpf(by, ty, t)
	}, sp.trunkR[0], sp.trunkR[1], func(x, y int, nx, ny, t float64) color.RGBA {
		i := 2
		if nx < -0.25 {
			i = 3
		} else if nx > 0.4 {
			i = 1
		}
		if detail >= 2 {
			if f := t * 9; f-math.Floor(f) < 0.22 {
				i-- // rings up the trunk
			}
		}
		return palmTrunk.at(i)
	})
	fl, fr := sp.frond[0], sp.frond[1]
	for i := 0; i < sp.fronds; i++ {
		a := (-205 + 230*float64(i)/float64(sp.fronds-1)) * math.Pi / 180
		dx, dy := math.Cos(a), math.Sin(a)*0.75
		l := fl * (0.82 + 0.18*math.Abs(dx)) // the upright ones are shorter
		ex, ey := tx+l*dx, ty+l*dy+l*0.42
		mx, my := tx+l*0.6*dx, ty+l*0.6*dy-l*0.3
		gameIconTube(palm, func(t float64) (float64, float64) {
			u := 1 - t
			return u*u*tx + 2*u*t*mx + t*t*ex, u*u*ty + 2*u*t*my + t*t*ey
		}, fr, fr*0.35, func(x, y int, nx, ny, t float64) color.RGBA {
			switch {
			case ny < -0.3:
				return palmRamp[4]
			case ny < 0.35:
				return palmRamp[3]
			}
			return palmRamp[1]
		})
	}
	if detail >= 2 {
		for _, o := range [][2]float64{{-1.6, 2.2}, {1.5, 2.6}} {
			palm.ellipse(tx+o[0], ty+o[1], 1.6, 1.6, func(x, y int, nx, ny float64) color.RGBA {
				if nx+ny < -0.3 {
					return palmTrunk[2]
				}
				return palmTrunk[0]
			})
		}
	}
	outline(palm, 0.85)
	bg.over(palm, 0, 0)

	// The tile: round the corners off and put the rim on.
	n := float64(bg.w)
	r, border := sp.radius*s, float64(sp.borderPx[detail])
	inside := func(x, y int, inset float64) bool {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		lo, hi, rr := inset, n-inset, max(r-inset, 0)
		if fx < lo || fx > hi || fy < lo || fy > hi {
			return false
		}
		dx := max(lo+rr-fx, fx-(hi-rr), 0)
		dy := max(lo+rr-fy, fy-(hi-rr), 0)
		return dx*dx+dy*dy <= rr*rr
	}
	for y := 0; y < bg.h; y++ {
		for x := 0; x < bg.w; x++ {
			switch {
			case !inside(x, y, 0):
				bg.put(x, y, color.RGBA{})
			case !inside(x, y, border):
				bg.put(x, y, colInk)
			}
		}
	}
	return bg
}

func gameIconLine(x0, y0, x1, y1 float64) func(float64) (float64, float64) {
	return func(t float64) (float64, float64) { return lerpf(x0, x1, t), lerpf(y0, y1, t) }
}

// gameIconTube fills a round stroke along path (design units, t from 0 to 1)
// whose radius runs from r0 to r1. f colours each pixel from its offset
// (nx, ny) from the centre line, in [-1, 1], and its place t along the path.
func gameIconTube(c *canvas, path func(t float64) (x, y float64), r0, r1 float64,
	f func(x, y int, nx, ny, t float64) color.RGBA) {
	const steps = 40
	k := c.k()
	var xs, ys, rs [steps + 1]float64
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i := range xs {
		t := float64(i) / steps
		px, py := path(t)
		xs[i], ys[i] = c.toPx(px, py)
		rs[i] = lerpf(r0, r1, t) * k
		x0, y0 = min(x0, xs[i]-rs[i]), min(y0, ys[i]-rs[i])
		x1, y1 = max(x1, xs[i]+rs[i]), max(y1, ys[i]+rs[i])
	}
	for y := int(math.Floor(y0)); y <= int(math.Ceil(y1)); y++ {
		for x := int(math.Floor(x0)); x <= int(math.Ceil(x1)); x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			best, bi := math.Inf(1), 0
			for i := range xs {
				if d := math.Hypot(fx-xs[i], fy-ys[i]) / rs[i]; d < best {
					best, bi = d, i
				}
			}
			if best > 1 {
				continue
			}
			nx, ny := (fx-xs[bi])/rs[bi], (fy-ys[bi])/rs[bi]
			if col := f(x, y, nx, ny, float64(bi)/steps); col.A > 0 {
				c.put(x, y, col)
			}
		}
	}
}
