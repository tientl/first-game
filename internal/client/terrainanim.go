package client

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The water of the newest worlds moving: rivers flowing, waterfalls
// pouring, foam churning in the pools under them, mist and spray rising,
// glints on lakes, light rippling over the shallows of the sea and foam
// along their edge. The ground is baked once (groundhd.go); what moves is
// drawn over it every frame by a shader from a small water map baked with
// each chunk that has such water: one pixel to an art pixel, saying what
// water it is and which way it runs. The shader draws on the full-resolution
// canvas, so its lines are as fine as the baked art.
//
// A texel of the water map: alpha 255 where there is water that moves (0
// elsewhere); blue is the kind in its top two bits and more in the rest:
//
//   - 1, a river: red and green its flow (128 is still), blue's low bits
//     how much it foams (3 bits: rapids, the pool under a fall) and how deep
//     it is (3 bits);
//   - 2, a waterfall: red how far down it the texel is (0 at the lip), blue's
//     low bits which way it pours (0 south, 1 east, 2 north, 3 west);
//   - 3, still water: blue's bit 5 set for the shallows of the sea (red is how
//     much is shallow, 0.5 at their edge), clear for a lake (red its depth).

// waterTexel is the water map's texel for chunk pixel (x, y), the middle of
// an art pixel; ok=false where no water moves.
func (b *bake) waterTexel(x, y int, u, v float64) (tx [4]byte, ok bool) {
	t := b.t
	if b.wall != nil {
		if p, on := b.wall.at(x, y); on {
			if p.kind != wallFall {
				return tx, false
			}
			dir := byte(0)
			switch {
			case p.gx > 0.6:
				dir = 1
			case p.gx < -0.6:
				dir = 3
			case p.gy < -0.6:
				dir = 2
			}
			return [4]byte{byte(clamp01(p.f) * 255), 0, 2<<6 | dir, 255}, true
		}
	}
	s := float64(b.coast[(y+coastUp)*chunkPW+x])
	if s >= 0 {
		rf := b.river
		if rf == nil {
			return tx, false
		}
		f := rf.share(x, y)
		j := y*chunkPW + x
		up, down := int(rf.up[j]), int(rf.down[j])
		if f < riverEdge+0.03 || up <= 11 {
			return tx, false
		}
		fx, fy := t.flowAt(u, v)
		depth := min(clamp01((f-riverEdge)/0.42), float64(up-6)/20, float64(down)/16)
		cs := cornersAt(u, v)
		foam := cs.share(t.rapid)
		if b.wall != nil {
			foam = max(foam, b.wall.underFall(x, y))
		}
		depth *= 1 - 0.7*cs.share(func(x, y int) bool { return t.tile(x, y) == sim.TileFord })
		aux := byte(min(7, foam*7.99))<<3 | byte(min(7, clamp01(depth)*7.99))
		return [4]byte{byte(128 + 127*fx), byte(128 + 127*fy), 1<<6 | aux, 255}, true
	}
	ttx, tty := int(math.Floor(u)), int(math.Floor(v))
	if !t.has(ttx, tty, featShallow) {
		return tx, false
	}
	if t.pondAt(u, v) > 0.5 {
		if cornersAt(u, v).share(func(x, y int) bool { return t.rel(x, y).Variant() == sim.VarLake && t.tile(x, y) == sim.TileWater }) <= 0 {
			return tx, false
		}
		return [4]byte{byte(clamp01(-s/2.6) * 255), 0, 3 << 6, 255}, true
	}
	if sh := t.shallowShare(u, v); sh > 0.25 {
		return [4]byte{byte(clamp01(sh) * 255), 0, 3<<6 | 32, 255}, true
	}
	return tx, false
}

// shallowShare is how much of sea point (u, v) is shallows, warped as
// shallowSea draws them.
func (t *terrain) shallowShare(u, v float64) float64 {
	wu := u + (vnoise(u*0.35, v*0.35, 958)-0.5)*1.2 + (vnoise(u*1.6, v*1.6, 1330)-0.5)*0.25
	wv := v + (vnoise(u*0.35+5, v*0.35, 959)-0.5)*1.2 + (vnoise(u*1.6+4, v*1.6, 1331)-0.5)*0.25
	return cornersAt(wu, wv).share(func(x, y int) bool { return t.tile(x, y) == sim.TileShallow })
}

// waterShaderSrc draws the moving water from a chunk's water map.
const waterShaderSrc = `//kage:unit pixels

package main

var Time float
var Origin vec2

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453)
}

func noise(p vec2) float {
	i := floor(p)
	f := fract(p)
	u := f * f * (3.0 - 2.0*f)
	a := hash(i)
	b := hash(i + vec2(1, 0))
	c := hash(i + vec2(0, 1))
	d := hash(i + vec2(1, 1))
	return mix(mix(a, b, u.x), mix(c, d, u.x), u.y)
}

func texel(p vec2) vec4 {
	q := clamp(p, vec2(0.5), imageSrc0Size()-vec2(0.5))
	return imageSrc0At(imageSrc0Origin() + q)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// On the grid of the baked art: a third of an art pixel.
	lp := (floor((srcPos-imageSrc0Origin())*3.0) + 0.5) / 3.0
	w := Origin + lp
	b := lp - 0.5
	i := floor(b)
	f := b - i
	t00 := texel(i + vec2(0.5, 0.5))
	t10 := texel(i + vec2(1.5, 0.5))
	t01 := texel(i + vec2(0.5, 1.5))
	t11 := texel(i + vec2(1.5, 1.5))
	if t00.a+t10.a+t01.a+t11.a < 3.99 {
		return vec4(0) // only well inside the water
	}
	n := texel(floor(lp) + 0.5)
	code := floor(n.b*255.0 + 0.5)
	kind := floor(code / 64.0)
	aux := code - kind*64.0
	light := vec3(0.87, 0.96, 0.99)
	a := 0.0

	if kind == 1.0 {
		// A river: streaks of light running down it, ripples, foam where it
		// is fast or churned, glints.
		fl := mix(mix(t00.rg, t10.rg, f.x), mix(t01.rg, t11.rg, f.x), f.y)*2.0 - 1.0
		if length(fl) < 0.15 {
			fl = vec2(0, 1)
		}
		fl = normalize(fl)
		foam := floor(aux/8.0) / 7.0
		depth := (aux - floor(aux/8.0)*8.0) / 7.0
		along := dot(w, fl)
		across := dot(w, vec2(-fl.y, fl.x))
		speed := 10.0 + 16.0*foam
		s1 := noise(vec2(across*0.5, (along-Time*speed)*0.065))
		s2 := noise(vec2(across*1.4+17.0, (along-Time*speed*1.25)*0.2))
		if s1 > 0.7-0.04*depth {
			a = 0.2 + 0.2*depth
		}
		if s2 > 0.8 {
			a = max(a, 0.32)
		}
		if foam > 0.0 {
			fn := noise(vec2(across*0.8, (along-Time*speed*1.5)*0.3))
			if fn < foam*0.55 {
				a = max(a, 0.45+0.45*foam)
			}
			// Mist over a churning pool, drifting.
			m := noise(vec2(w.x*0.1, w.y*0.16+Time*0.8))
			if foam > 0.55 && m > 0.6 {
				a = max(a, 0.22)
			}
			// Spray thrown up from it.
			cell := floor(w / 3.0)
			h := hash(cell + 3.0)
			if foam > 0.6 && h > 0.7 {
				ph := fract(Time*0.9 + h*7.0)
				pp := cell*3.0 + vec2(1.5+sin(h*40.0), 3.0-ph*4.0)
				if length(w-pp) < 0.45 {
					a = 0.85 * (1.0 - ph)
				}
			}
		}
		h := hash(floor(w * 0.6))
		if h > 0.992-0.004*depth && fract(Time*0.8+h*13.0) < 0.15 {
			a = 0.9 // a glint
		}
	} else if kind == 2.0 {
		// A waterfall: streaks pouring down it, the lip shimmering, foam
		// churning at its foot.
		prog := n.r
		dir := aux - floor(aux/4.0)*4.0
		ax := w.x
		d := w.y
		if dir == 1.0 {
			ax, d = w.y, w.x
		} else if dir == 2.0 {
			d = -w.y
		} else if dir == 3.0 {
			ax, d = w.y, -w.x
		}
		st := noise(vec2(ax*0.85, d*0.09-Time*5.5))
		if st > 0.6 {
			a = 0.6
		} else if st < 0.28 {
			return vec4(0.08, 0.2, 0.28, 1.0) * 0.22
		}
		if prog < 0.12 && noise(vec2(ax*0.5-Time*1.5, Time*0.5)) > 0.55 {
			a = 0.45
		}
		if prog > 0.78 && noise(vec2(ax*0.45, d*0.4-Time*3.0)) > 0.4 {
			a = 0.8
		}
	} else if kind == 3.0 {
		sea := aux >= 32.0
		if sea {
			// Light rippling over the sand of the shallows.
			c1 := noise(w*0.085 + vec2(Time*0.18, Time*0.11))
			c2 := noise(w*0.11 + vec2(-Time*0.14, Time*0.19) + 5.0)
			ca := 1.0 - abs(c1-c2)*9.0
			if ca > 0.0 && n.r > 0.55 {
				a = 0.16 * ca
			}
			// Foam along the drop at the shallows' edge, coming and going.
			sh := mix(mix(t00.r, t10.r, f.x), mix(t01.r, t11.r, f.x), f.y)
			e := 1.0 - abs(sh-0.52)/0.045
			if e > 0.0 {
				p := 0.5 + 0.5*sin(Time*1.7+w.x*0.07+w.y*0.05)
				if hash(floor(w*1.5)+floor(Time*4.0)) < e*p*0.9 {
					a = max(a, 0.55)
				}
			}
		} else {
			// A lake: still, the sky's light lying on it in long faint
			// streaks that drift, and a glint now and then.
			if noise(vec2(w.x*0.025+Time*0.04, w.y*0.5)) > 0.82 && n.r > 0.2 {
				a = 0.12
			}
			h := hash(floor(w * 0.5))
			if h > 0.995 && fract(Time*0.5+h*11.0) < 0.12 {
				a = 0.8
			}
		}
	}
	if a <= 0.0 {
		return vec4(0)
	}
	a = floor(a*5.0+0.5) / 5.0
	return vec4(light*a, a)
}
`

var waterShader *ebiten.Shader

func getWaterShader() *ebiten.Shader {
	if waterShader == nil {
		s, err := ebiten.NewShader([]byte(waterShaderSrc))
		if err != nil {
			log.Printf("water shader: %v", err)
			return nil
		}
		waterShader = s
	}
	return waterShader
}

// drawWaterAnim draws the moving water of one chunk, whose top-left is at
// (x, y) on dst and at world pixel (wx, wy), from its water map m.
func drawWaterAnim(dst drawTarget, m *ebiten.Image, x, y, wx, wy float64, frame int) {
	sh := getWaterShader()
	if sh == nil {
		return
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(x, y)
	op.Images[0] = m
	op.Uniforms = map[string]any{
		"Time":   float32(float64(frame%(60*60*10)) / 60),
		"Origin": []float32{float32(wx), float32(wy)},
	}
	drawShaderFine(dst, chunkW, chunkH, sh, op)
}

// liveGround is the ground cache of the game running, for GroundCost.
var liveGround *groundCache

// GroundCost is what the ground cache holds now: how many chunks, the bytes
// of all its images, and of those the water maps the moving water is drawn
// from.
func GroundCost() (chunks int, bytes, water int64) {
	gc := liveGround
	if gc == nil {
		return 0, 0, 0
	}
	size := func(img *ebiten.Image) int64 {
		b := img.Bounds()
		return int64(4 * b.Dx() * b.Dy())
	}
	for _, img := range gc.chunks {
		bytes += size(img)
	}
	for _, m := range []map[chunkKey]*ebiten.Image{gc.shores, gc.roofs} {
		for _, img := range m {
			bytes += size(img)
		}
	}
	for _, img := range gc.waters {
		water += size(img)
	}
	return len(gc.chunks), bytes + water, water
}
