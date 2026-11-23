package client

import (
	"image/color"
	"math"
)

// The snake is drawn as a tube along a serpenoid curve, the shape real snakes
// take as they slither: the bend angle along the body follows a cosine, so
// the curves travel down the body from frame to frame. The body is seen from
// above at the game's 3/4 angle (lateral curves squashed into the ground),
// shaded round, with a ridge of light along the spine, dark saddle blotches
// down the back, a pale belly on the near side, and a wedge-shaped head with
// a dark stripe behind the eye. Resting snakes lie coiled with the head up.

var (
	snakeRamp  = mkRamp(0x4a2f10, 0x7a551f, 0xa97f30, 0xcfa646, 0xe8ca6c) // sandy gold
	blotchRamp = mkRamp(0x24100a, 0x431d0e, 0x6b3013, 0x8a4418)           // the diamonds on the back
	bellyRamp  = mkRamp(0xa8946a, 0xd6c693, 0xf3e8bd)
	snakeEye   = hex(0xffd23a)
	colTongue  = hex(0xd8344a)
	colFang    = hex(0xf2eee2)
)

// Sprite indices into creatureArt.snake.
const (
	snakeSlither = 4 // slither frames, 0 to snakeSlither-1
	snakeStrike  = snakeSlither
	snakeCoil    = snakeSlither + 1 // resting
	snakeCoilLik = snakeSlither + 2 // resting, flicking its tongue
	snakeSprites = snakeSlither + 3
)

const snakeDiamonds = 7 // down the back

const snakeSquash = 0.6 // how much the ground plane is foreshortened

// spinePt is one sample of a body's centre line, in canvas pixels.
type spinePt struct {
	x, y   float64
	nx, ny float64 // unit normal, pointing to the body's right (the near side, for a snake heading right)
	r      float64 // half the body's width here
	s      float64 // 0 at the tail tip, 1 at the neck
}

// withNormals fills in each point's normal from its neighbours.
func withNormals(pts []spinePt) []spinePt {
	for i := range pts {
		a, b := pts[max(0, i-1)], pts[min(len(pts)-1, i+1)]
		tx, ty := b.x-a.x, b.y-a.y
		if l := math.Hypot(tx, ty); l > 1e-9 {
			pts[i].nx, pts[i].ny = -ty/l, tx/l
		}
	}
	return pts
}

// snakeWidth is the body's half-width at s: a thin tail tip, a thick middle,
// a slimmer neck.
func snakeWidth(s float64) float64 {
	return 0.5 + 1.7*smooth(clamp01(s/0.35)) - 0.7*smooth(clamp01((s-0.8)/0.2))
}

// drawTube fills the body: each pixel takes the sample nearest the head that
// covers it, so a later stretch of body lies over an earlier one, as in a
// coil. Where a higher coil overlaps a lower one, the lower is shadowed.
func drawTube(c *canvas, pts []spinePt) { drawTubeWith(c, pts, snakeScale) }

// drawTubeWith is drawTube with its own skin: shade colours one pixel given
// the sample it belongs to and how far across the body it is (-1..1).
func drawTubeWith(c *canvas, pts []spinePt, shade func(x, y int, p *spinePt, t float64) color.RGBA) {
	if k := c.k(); k != 1 || c.tx != 0 || c.ty != 0 { // a scaled canvas: the spine is in design units
		scaled := make([]spinePt, len(pts))
		for i, p := range pts {
			p.x, p.y = c.toPx(p.x, p.y)
			p.r *= k
			scaled[i] = p
		}
		pts = scaled
	}
	best := make([]float64, c.w*c.h)
	for i := range best {
		best[i] = -1
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			var hit *spinePt
			for i := range pts {
				p := &pts[i]
				dx, dy := px-p.x, py-p.y
				if dx*dx+dy*dy <= p.r*p.r && (hit == nil || p.s > hit.s) {
					hit = p
				}
			}
			if hit == nil {
				continue
			}
			best[y*c.w+x] = hit.s
			t := ((px-hit.x)*hit.nx + (py-hit.y)*hit.ny) / hit.r // -1 far side .. 1 near side
			c.put(x, y, shade(x, y, hit, t))
		}
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			s := best[y*c.w+x]
			if s < 0 {
				continue
			}
			for _, d := range [2][2]int{{0, -1}, {-1, 0}} {
				nx, ny := x+d[0], y+d[1]
				if c.in(nx, ny) && best[ny*c.w+nx] > s+0.2 {
					c.put(x, y, mix(c.at(x, y), colInk, 0.45)) // tucked under the coil above
					break
				}
			}
		}
	}
}

// snakeScale colours one pixel of the body: sandy scales, a chain of dark
// chestnut diamonds down the back, each rimmed with cream, and a pale belly
// showing along the near side.
func snakeScale(x, y int, p *spinePt, t float64) color.RGBA {
	l := lum(p.nx*t*0.9, p.ny*t*0.9)
	back := t + 0.22 // across the body, 0 on the spine
	if math.Abs(back) < 0.25 {
		l += 0.14 // the light along the spine
	}
	at := math.Abs(math.Mod(p.s*snakeDiamonds+0.5, 1)-0.5) * 2 // 0 mid-diamond .. 1 between two
	d := at + math.Abs(back)*0.85
	patterned := p.s > 0.07 && p.s < 0.93
	switch {
	case t > 0.6:
		return bellyRamp.pick(l+0.2, x, y)
	case patterned && d < 0.62:
		return flat(blotchRamp, l+0.05)
	case patterned && d < 0.85:
		return flat(snakeRamp, l+0.38) // the pale rim of each diamond
	}
	return flat(snakeRamp, l-0.04)
}

// snakeHeads are the head, shut and striking, side-on and facing right: a
// pale crown, a dark stripe running back from a yellow eye, a cream lip.
var snakeHeads = [2][]string{
	{
		".AAAA...",
		"AddyAAb.",
		"bbbbbbbb",
		".cccccc.",
	},
	{
		"..AAAAA..",
		".AddyAAbb",
		"bbbbmwmw.",
		"bbmmmm...",
		"ccmmwcw..",
		".cccccc..",
	},
}

// stamp draws the rows of a little picture with its top-left at (x0, y0),
// each letter the colour pal gives it; any other letter is left clear.
func stamp(c *canvas, x0, y0 int, rows []string, pal map[byte]color.RGBA) {
	for y, row := range rows {
		for x := range len(row) {
			if col, ok := pal[row[x]]; ok {
				c.set(x0+x, y0+y, col)
			}
		}
	}
}

// flat takes the colour of rp for brightness l without dithering, for skin
// too finely patterned to dither.
func flat(rp ramp, l float64) color.RGBA {
	return rp.at(int(math.Round(clamp01(l) * float64(len(rp)-1))))
}

// drawSnakeHead draws the head with the back of its skull at (hx, hy),
// looking right. open shows the mouth and fangs; tongue flicks the forked
// tongue.
func drawSnakeHead(c *canvas, hx, hy float64, open, tongue bool) {
	rows := snakeHeads[0]
	x0, y0 := int(math.Round(hx)), int(math.Round(hy))-2
	if open {
		rows = snakeHeads[1]
	}
	stamp(c, x0, y0, rows, map[byte]color.RGBA{
		'A': snakeRamp[4], 'b': snakeRamp[2], 'd': blotchRamp[0], 'y': snakeEye,
		'c': bellyRamp[2], 'm': hex(0x7a1420), 'w': colFang,
	})
	if tongue {
		tx, ty := x0+len(rows[0]), y0+2
		if open {
			tx, ty = tx-3, ty+1
		}
		c.set(tx, ty, colTongue)
		c.set(tx+1, ty, colTongue)
		c.set(tx+2, ty-1, colTongue)
		c.set(tx+2, ty+1, colTongue)
	}
}

// serpenoid lays a body of length L along a serpenoid curve ending with its
// neck at (x1, y1), heading right. The head end barely sways, as a real
// snake keeps its head steady while the body waves behind it.
func serpenoid(L, bend, waves, phase, x1, y1 float64) []spinePt {
	const n = 140
	us, vs := make([]float64, n+1), make([]float64, n+1)
	for i := 1; i <= n; i++ {
		s := float64(i) / n
		steady := 1 - 0.75*smooth(clamp01((s-0.72)/0.28))
		th := bend * steady * math.Cos(2*math.Pi*waves*s-phase)
		us[i] = us[i-1] + math.Cos(th)*L/n
		vs[i] = vs[i-1] + math.Sin(th)*L/n
	}
	pts := make([]spinePt, n+1)
	for i := range pts {
		s := float64(i) / n
		pts[i] = spinePt{
			x: x1 + us[i] - us[n],
			y: y1 + (vs[i]-vs[n])*snakeSquash,
			r: snakeWidth(s), s: s,
		}
	}
	return withNormals(pts)
}

// genSnake draws one snake sprite, facing right. Frames below snakeSlither
// slither; then the strike and the resting coil, with and without the
// tongue out.
func genSnake(frame int) sprite {
	c, ax, ay := snakeCanvas(frame)
	return mk(c, ax, ay)
}

// snakeCanvas draws sprite frame on a canvas and returns its anchor.
func snakeCanvas(frame int) (c *canvas, ax, ay float64) {
	switch frame {
	case snakeCoil, snakeCoilLik:
		return snakeCoilCanvas(frame == snakeCoilLik)
	}
	c = newCanvas(47, 24)
	const neckX, neckY = 34.0, 13.0
	strike := frame == snakeStrike
	ph := float64(frame) * 2 * math.Pi / snakeSlither
	if strike {
		ph = 0
	}
	pts := serpenoid(32, 1.3, 1.15, ph, neckX, neckY)
	hx, hy := neckX-0.5, neckY
	if strike { // the fore-body rears up and the head lunges
		for i := range pts {
			pts[i].y -= 6 * smooth(clamp01((pts[i].s-0.55)/0.45))
			pts[i].x += 2 * smooth(clamp01((pts[i].s-0.7)/0.3))
		}
		pts = withNormals(pts)
		hx, hy = neckX+1.5, neckY-6
	}
	drawTube(c, pts)
	drawSnakeHead(c, hx, hy, strike, strike || frame == 1)
	c.outline(0.8)
	return c, 20, 14
}

// snakeCoilCanvas draws a snake at rest: coiled in a low pile, the head raised
// in the middle and looking right.
func snakeCoilCanvas(tongue bool) (*canvas, float64, float64) {
	c := newCanvas(30, 24)
	const cx, cy = 14.0, 16.0
	const n = 200
	const turns, inner, outer = 1.3, 2.6, 9.2
	const coilEnd = 0.86 // the rest is the neck
	pts := make([]spinePt, 0, n+1)
	spiral := func(s float64) (float64, float64) {
		ang := 0.3 + 2*math.Pi*turns*s
		rad := outer - (outer-inner)*s/coilEnd
		return cx + rad*math.Cos(ang), cy + rad*math.Sin(ang)*0.72 - 1.6*s // a flat pile, so the rings show
	}
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		var x, y float64
		if s < coilEnd { // spiralling in, each turn a little higher
			x, y = spiral(s)
		} else { // the neck rises out of the middle
			u := (s - coilEnd) / (1 - coilEnd)
			x0, y0 := spiral(coilEnd)
			x, y = lerpf(x0, cx+2, u), lerpf(y0, cy-7.5, smooth(u))
		}
		pts = append(pts, spinePt{x: x, y: y, r: snakeWidth(s) * 0.88, s: s})
	}
	drawTube(c, withNormals(pts))
	drawSnakeHead(c, cx+1.4, cy-8.2, false, tongue)
	c.outline(0.8)
	return c, cx, cy + 3
}
