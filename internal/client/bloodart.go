package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
)

// Blood under a body, at HD: a pool seeping out from under the wound side,
// dark crimson going to maroon where it soaks into the ground, its edge
// ragged, a glossy light on it while it is wet, and a few drops thrown
// round. It spreads over bloodSpread frames from the moment the body falls
// (each pixel of the pool has the moment it is reached), then dulls and
// darkens as it dries, and fades a little as the body rots. Its size goes
// with the body's.

const (
	bloodStages = 10      // steps of the pool spreading
	bloodSpread = 60 * 3 / 2 // game frames it takes to spread
	bloodGloss  = 60 * 25    // game frames it stays wet and glossy
	bloodDry    = 60 * 90    // game frames it takes to dry dark
)

// The pool's colours, from where it has soaked in to where it lies thick.
var bloodHD = hexes(0x1c0306, 0x2c050a, 0x3e080e, 0x4e0b12, 0x5c1016, 0x6a151b)

type bloodKey struct {
	size, variant, stage int
}

var bloodSprites = map[bloodKey]sprite{}
var bloodGlossSprites = map[[2]int]sprite{}

// bloodSizes is the radius across of each size of pool, art pixels.
var bloodSizes = []float64{2.6, 3.6, 4.8, 6.2, 8, 10.4, 13}

// bloodSizeOf is the size of pool under a body length art pixels long.
func bloodSizeOf(length float64) int {
	r := length * 0.2
	best := 0
	for i, s := range bloodSizes {
		if math.Abs(s-r) < math.Abs(bloodSizes[best]-r) {
			best = i
		}
	}
	return best
}

// bloodField is the pool of size and variant: for each image pixel of a
// canvas w by h, how soon the blood reaches it (0 to 1; above 1 never),
// and its depth there (0 at its edge to 1 thick).
type bloodField struct {
	w, h   int
	ox, oy float64 // where it comes from, image pixels
	when   []float64
	depth  []float64
	drop   []bool // a drop thrown off, not the pool
}

var bloodFields = map[[2]int]*bloodField{}

func bloodFieldOf(size, variant int) *bloodField {
	key := [2]int{size, variant}
	if f, ok := bloodFields[key]; ok {
		return f
	}
	const K = HDDensity
	R := bloodSizes[size]
	seed := uint32(size*31 + variant*977 + 7)
	wArt, hArt := 2*R*1.9+6, R*1.5+6
	f := &bloodField{w: int(math.Ceil(wArt * K)), h: int(math.Ceil(hArt * K))}
	f.ox, f.oy = float64(f.w)/2, float64(f.h)*0.4
	f.when = make([]float64, f.w*f.h)
	f.depth = make([]float64, f.w*f.h)
	f.drop = make([]bool, f.w*f.h)
	// Lobes: the pool is the union of a few blobs, each reached in turn,
	// spreading sideways along the ground and out towards the viewer.
	type lobe struct{ x, y, rx, ry, t float64 }
	r := fxRng{seed}
	lobes := []lobe{{0, 0.15 * R, R * 0.62, R * 0.3, 0}}
	for i := 0; i < 4; i++ {
		side := float64(i%2*2 - 1)
		x := side * R * r.in(0.25, 0.75)
		y := R * r.in(0.05, 0.4)
		lobes = append(lobes, lobe{x, y, R * r.in(0.3, 0.5), R * r.in(0.16, 0.26), r.in(0.25, 0.7)})
	}
	for y := 0; y < f.h; y++ {
		for x := 0; x < f.w; x++ {
			i := y*f.w + x
			f.when[i] = 2
			ax := (float64(x)+0.5-f.ox)/K + 0
			ay := (float64(y)+0.5-f.oy)/K + 0
			// The edge ragged with noise, finer and coarser.
			n := (vnoise(ax*0.55, ay*0.9, seed) - 0.5) + 0.5*(vnoise(ax*1.6, ay*2.4, seed+1)-0.5)
			best, deep := 2.0, 0.0
			for _, l := range lobes {
				d := math.Hypot((ax-l.x)/l.rx, (ay-l.y)/l.ry) + n*0.38
				if d >= 1 {
					continue
				}
				when := l.t + (1-l.t)*d*0.9
				if when < best {
					best = when
				}
				deep = math.Max(deep, 1-d)
			}
			f.when[i], f.depth[i] = best, deep
		}
	}
	// Drops thrown off as it fell: small, round, reached at once.
	for i := 0; i < 3+variant%3; i++ {
		a := r.in(-math.Pi, math.Pi)
		dd := R * r.in(0.95, 1.5)
		cx, cy := math.Cos(a)*dd*1.2, 0.2*R+math.Sin(a)*dd*0.45
		rr := r.in(0.35, 0.75)
		if i == 0 {
			rr = r.in(0.6, 0.95)
		}
		for y := 0; y < f.h; y++ {
			for x := 0; x < f.w; x++ {
				ax := (float64(x)+0.5-f.ox)/K - cx
				ay := (float64(y)+0.5-f.oy)/K - cy
				if d := math.Hypot(ax/rr, ay/(rr*0.7)); d < 1 {
					j := y*f.w + x
					if f.when[j] > 1 {
						f.when[j], f.depth[j], f.drop[j] = 0.05, 1-d, true
					}
				}
			}
		}
	}
	bloodFields[key] = f
	return f
}

// bloodSprite is the pool of size and variant spread to stage.
func bloodSprite(size, variant, stage int) sprite {
	key := bloodKey{size, variant, stage}
	if s, ok := bloodSprites[key]; ok {
		return s
	}
	f := bloodFieldOf(size, variant)
	g := float64(stage) / bloodStages
	px := make([]color.RGBA, f.w*f.h)
	n := len(bloodHD)
	for y := 0; y < f.h; y++ {
		for x := 0; x < f.w; x++ {
			i := y*f.w + x
			if f.when[i] > g {
				continue
			}
			// How deep it lies here, as far as the blood has come: near the
			// front of the spreading blood it is thin still.
			edge := clamp01((g - f.when[i]) * 6)
			d := f.depth[i] * (0.4 + 0.6*edge)
			if f.drop[i] {
				d = 0.35 + 0.5*f.depth[i]
			}
			// Bands, dithered where they meet so no stripe shows.
			v := 0.6 + d*float64(n-1)*0.85
			if dither(x, y) < v-math.Floor(v) {
				v = math.Ceil(v)
			} else {
				v = math.Floor(v)
			}
			c := bloodHD[max(0, min(n-1, int(v)))]
			// Soaked in at its edge: the ground shows through.
			a := 0.62 + 0.36*smooth(clamp01(d*2.2))
			c.A = uint8(255 * a)
			px[i] = c
		}
	}
	// A dark rim where it stands thick at its edge, on its far (upper) side.
	for y := 1; y < f.h; y++ {
		for x := 0; x < f.w; x++ {
			i := y*f.w + x
			if px[i].A != 0 && px[i-f.w].A == 0 && !f.drop[i] {
				px[i] = withAlpha(bloodHD[0], px[i].A)
			}
		}
	}
	img := ebiten.NewImage(f.w, f.h)
	img.WritePixels(straightBytes(px))
	artBytes.Add(int64(4 * f.w * f.h))
	s := spriteHD(img, f.ox/HDDensity, f.oy/HDDensity)
	bloodSprites[key] = s
	return s
}

// bloodGlossSprite is the wet light on a pool: a soft, broken highlight on
// its thick middle and a glint on each drop, drawn added to it.
func bloodGlossSprite(size, variant int) sprite {
	key := [2]int{size, variant}
	if s, ok := bloodGlossSprites[key]; ok {
		return s
	}
	f := bloodFieldOf(size, variant)
	R := bloodSizes[size]
	px := make([]color.RGBA, f.w*f.h)
	hl := hex(0xb0424a)
	for y := 0; y < f.h; y++ {
		for x := 0; x < f.w; x++ {
			i := y*f.w + x
			if f.when[i] > 1 {
				continue
			}
			ax := (float64(x)+0.5-f.ox)/HDDensity + R*0.22
			ay := (float64(y)+0.5-f.oy)/HDDensity - R*0.02
			a := 0.0
			if f.drop[i] {
				if f.depth[i] > 0.55 && y > 0 && f.when[i-f.w] <= 1 && x%2 == 0 {
					a = 0.55
				}
			} else if d := math.Hypot(ax/(R*0.3), ay/(R*0.09)); d < 1 && f.depth[i] > 0.45 {
				a = 0.5 * (1 - d*d)
				if hashf(x/2, y, 61) < 0.25 {
					a *= 0.4 // broken up, not a painted stripe
				}
			}
			if a > 0.04 {
				c := hl
				c.A = uint8(255 * a)
				px[i] = c
			}
		}
	}
	img := ebiten.NewImage(f.w, f.h)
	img.WritePixels(straightBytes(px))
	artBytes.Add(int64(4 * f.w * f.h))
	s := spriteHD(img, f.ox/HDDensity, f.oy/HDDensity)
	bloodGlossSprites[key] = s
	return s
}

// drawBodyBlood draws the blood of body e, length art pixels long, seeping
// from (x, y), age game frames after it fell.
func (g *Game) drawBodyBlood(dst drawTarget, e protocol.EntityView, x, y, length float64, age int) {
	size := bloodSizeOf(length)
	variant := int(uint32(e.ID)*2654435761>>29) % 3
	stage := bloodStages
	if age < bloodSpread {
		t := float64(age) / bloodSpread
		stage = int(math.Ceil(bloodStages * (1 - (1-t)*(1-t)))) // fast at first, then slowly
	}
	if stage <= 0 {
		return
	}
	s := bloodSprite(size, variant, stage)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Translate(math.Round(x*HDDensity)/HDDensity, math.Round(y*HDDensity)/HDDensity)
	dry := clamp01(float64(age) / bloodDry)
	k := float32(1 - 0.28*dry)
	op.ColorScale.Scale(k, k*(1-0.1*float32(dry)), k*(1-0.1*float32(dry)), 1)
	drawSprite(dst, s, op)
	if wet := 1 - clamp01(float64(age-bloodSpread/2)/bloodGloss); wet > 0.02 && stage >= bloodStages/2 {
		gs := bloodGlossSprite(size, variant)
		op.ColorScale.Reset()
		w := float32(wet * clamp01(float64(stage)/bloodStages))
		op.ColorScale.Scale(w, w, w, w)
		op.Blend = ebiten.BlendLighter
		drawSprite(dst, gs, op)
	}
}
