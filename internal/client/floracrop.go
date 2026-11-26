package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Rice, corn and squash at HD: as they grow in a bed, stage by stage, and
// as they grow wild. A look is a number: 0 to sim.CropStages-1 the stages of
// growing, sim.CropStages ripe, then the wild plant (cropWild) and what is
// left of that when it is picked (cropWildPicked).

const (
	cropRipe       = sim.CropStages
	cropWild       = sim.CropStages + 1
	cropWildPicked = sim.CropStages + 2
	cropLooks      = sim.CropStages + 3
)

var (
	riceRamp     = mkRamp(0x1e4a1c, 0x327026, 0x4e9434, 0x74b846, 0xa2d864, 0xccf090)
	riceRipeRamp = mkRamp(0x5a5a1c, 0x86862a, 0xb0aa3c, 0xd4c858, 0xeee084)
	grainRamp    = mkRamp(0x8a641a, 0xb88c2a, 0xdcb444, 0xf4d66c, 0xfdf0a8)
	cornRamp     = mkRamp(0x1c4820, 0x2c6a2a, 0x449034, 0x66b446, 0x94d460, 0xc4ee8c)
	huskRamp     = mkRamp(0x5c7c30, 0x86a444, 0xb0c864, 0xd4e490, 0xeef4c0)
	kernelRamp   = mkRamp(0xa46814, 0xd49822, 0xf2c43a, 0xfce674, 0xfff8c0)
	squashLeaf   = mkRamp(0x183e1e, 0x265a28, 0x3a7a32, 0x549a3e, 0x78b850, 0xa6d670)
	squashRamp   = mkRamp(0x7c3410, 0xb0521a, 0xdc7a24, 0xf4a23c, 0xfcc866, 0xffe8a4)
	squashGreen  = mkRamp(0x1e4020, 0x34622a, 0x528838, 0x7cac4c, 0xb0cc74, 0xdce8a8)
)

// --- rice --------------------------------------------------------------------------------

// panicle hangs a head of grain from art point (x, y), drooping to the
// side dir, n grains long.
func (f *flo) panicle(x, y, dir float64, n int, rp ramp, seed uint32) {
	for k := 0; k < n; k++ {
		t := float64(k) / float64(n)
		px, py := x+dir*(0.3+3.6*t), y+0.4+5.2*t*t
		for _, side := range []float64{-1, 1} {
			gx, gy := px+side*0.55, py+side*0.25+hashf(k, int(side)+1, seed)*0.4
			f.c.ellipse(gx*fk, gy*fk, 1.5, 2.1, func(ix, iy int, nx, ny float64) color.RGBA {
				return rp.pick(0.62+lum(nx, ny)*0.4-0.14*float64(k%2), ix, iy)
			})
		}
		f.q.dot(px, py, 0, 0, rp.at(1))
	}
}

func genRiceHD(look int) (*canvas, float64, float64) {
	f := newFlo(40, 34)
	rng := rand.New(rand.NewPCG(88, uint64(look)+1))
	const bx, base = 20.0, 31.0
	stems, h, spread, heads, rp := 12, 17.0, 6.0, 0, riceRamp
	switch look {
	case 0:
		stems, h, spread = 7, 4.5, 5
	case 1:
		stems, h, spread = 11, 10, 5.5
	case 2:
		stems, h, heads = 13, 16, 3
	case cropRipe:
		stems, h, heads, rp = 13, 17, 8, riceRipeRamp
	case cropWild:
		stems, h, spread, heads = 15, 19, 7.5, 7
	case cropWildPicked:
		stems, h, spread = 14, 3.5, 7.5
	}
	type tip struct{ x, y, dir float64 }
	var tips []tip
	for row := 0; row < 2; row++ {
		for i := 0; i < stems; i++ {
			if (i%2 == 0) != (row == 0) {
				continue
			}
			x0 := bx + (float64(i)/float64(stems-1)-0.5)*2*spread*0.55 + (rng.Float64()-0.5)*0.8
			hh := h * (0.72 + 0.4*rng.Float64())
			lean := (x0-bx)*0.75 + (rng.Float64()-0.5)*3
			lo := 0.08 + 0.16*float64(row)
			r := rp
			if look == cropWild && i%3 == 0 {
				r = riceRipeRamp // some stems ripe already
			}
			bladeHD(f.c, x0*fk, base*fk, lean*fk, hh*fk, 2.6+rng.Float64(), r, lo, lo+0.62)
			if look >= 1 && look != cropWildPicked { // a long leaf bending away
				bladeHD(f.c, (x0+0.5)*fk, base*fk, lean*2.2*fk+math.Copysign(5, lean), hh*0.68*fk, 2.4, r, lo-0.04, lo+0.5)
			}
			tips = append(tips, tip{x0 + lean, base - hh, math.Copysign(1, lean+0.01)})
		}
	}
	for i := 0; i < heads && i < len(tips); i++ {
		t := tips[(i*5+2)%len(tips)]
		g := grainRamp
		if look == 2 {
			g = riceRamp[1:] // still green in the ear
		}
		f.panicle(t.x, t.y, t.dir, 6+i%3, g, 880+uint32(i))
	}
	if look >= cropWild {
		f.tuft(rng, bx, base+0.6, 16, 9, 3, bladeRamp)
	}
	return f.done(0.5, bx, base)
}

// --- corn --------------------------------------------------------------------------------

// cornStalk stands one corn plant on art point (x, base), h tall: a jointed
// stalk, long leaves arching off it left and right, a tassel when it has
// one and ears ears.
func (f *flo) cornStalk(rng *rand.Rand, x, base, h, lean float64, leaves int, tassel bool, ears int, cut bool) {
	if cut {
		st := aCap(x, base, x+0.3, base-3.2, 0.75, 0.65)
		f.q.fill(st, shd(riceRipeRamp, st, 1, 0))
		f.q.flat(aEll(x+0.3, base-3.3, 0.6, 0.3), plain(riceRipeRamp[4]))
		f.leafAt(x, base-1.5, 0.25, 6, 1, 0.5, leafStrap, riceRipeRamp, 0, -0.1) // a dry leaf left lying
		return
	}
	top := [2]float64{x + lean, base - h}
	tb := newCTube(bez(14, 0.8, 0.4, [2]float64{x, base}, [2]float64{x, base - h*0.4}, [2]float64{x + lean*0.5, base - h*0.75}, top))
	at := func(t float64) tpt { return tb.p[min(len(tb.p)-1, int(t*float64(len(tb.p)-1)+0.5))] }
	leaf := func(i int, lit float64) {
		t := 0.12 + 0.8*float64(i)/float64(max(1, leaves))
		p := at(t)
		a := 0.42 + 0.62*rng.Float64()
		if i%2 == 0 {
			a = math.Pi - a
		}
		l := math.Min(h*0.62, 10.5) * (0.7 + 0.4*math.Sin(math.Pi*t)) * (0.75 + 0.5*rng.Float64())
		f.leafAt(p.x, p.y, a, l, math.Min(1.4, 0.5+h*0.04), 0.45+0.5*rng.Float64(), leafStrap, cornRamp, 0, lit+(rng.Float64()-0.5)*0.1)
	}
	for i := 0; i < leaves; i += 2 { // the leaves behind the stalk
		leaf(i, -0.12)
	}
	if tassel {
		for k := -3; k <= 3; k++ { // a plume of thin spikes
			a := math.Pi/2 + float64(k)*0.3
			ex, ey := top[0]+math.Cos(a)*(3.6-0.25*math.Abs(float64(k))), top[1]-math.Sin(a)*4.6
			f.c.line(top[0]*fk, top[1]*fk, ex*fk, ey*fk, func(px, py int, t float64) color.RGBA {
				if (px+py)%3 == 0 {
					return grainRamp[4]
				}
				return grainRamp.at(1 + int(t*2.5))
			})
		}
	}
	f.stalk(cornRamp, 0.8, 0.4, 0, [2]float64{x, base}, [2]float64{x, base - h*0.4}, [2]float64{x + lean*0.5, base - h*0.75}, top,
		func(s, v, l float64, col color.RGBA) color.RGBA {
			if u := s * h / 4.2; u-math.Floor(u) < 0.1 {
				return toneOf(cornRamp, l*0.86-0.22)
			}
			return col
		})
	for e := 0; e < ears; e++ { // an ear in its husk, the kernels showing at the tip, the silk hanging out
		t := 0.4 + 0.16*float64(e)
		p := at(t)
		side := float64(e%2)*2 - 1
		ex, ey := p.x+side*2.4, p.y-3.4
		ear := aCap(p.x+side*0.6, p.y, ex, ey, 1.35, 0.95)
		f.q.fill(ear, func(px, py int, _ float64) color.RGBA {
			l := ear.lum(px, py, 1.5*fk)*0.85 + 0.12
			ay := float64(py) / fk
			if ay < ey+0.9 { // the kernels, in rows
				if (px+py)%3 == 0 {
					l -= 0.16
				}
				return toneOf(kernelRamp, l)
			}
			if (px*2+py)%5 == 0 {
				l -= 0.14 // the folds of the husk
			}
			return toneOf(huskRamp, l)
		})
		for k := 0; k < 4; k++ {
			f.c.line(ex*fk, (ey-0.6)*fk, (ex+side*(0.6+float64(k)*0.5))*fk, (ey-1.6+float64(k)*0.8)*fk, func(int, int, float64) color.RGBA { return hex(0xa8683a) })
		}
	}
	for i := 1; i < leaves; i += 2 { // and the leaves in front of it
		leaf(i, 0.04)
	}
}

func genCornHD(look int) (*canvas, float64, float64) {
	f := newFlo(44, 44)
	rng := rand.New(rand.NewPCG(89, uint64(look)+1))
	const bx, base = 22.0, 41.0
	switch look {
	case 0:
		f.cornStalk(rng, bx, base, 4.5, 0.3, 2, false, 0, false)
	case 1:
		f.cornStalk(rng, bx-3.5, base-0.4, 9, -0.5, 3, false, 0, false)
		f.cornStalk(rng, bx+3, base+0.3, 12, 0.6, 4, false, 0, false)
	case 2:
		f.cornStalk(rng, bx-3.5, base-0.4, 19, -1.4, 6, true, 0, false)
		f.cornStalk(rng, bx+3, base+0.3, 23, 1, 7, true, 0, false)
	case cropRipe:
		f.cornStalk(rng, bx-3.5, base-0.4, 25, -1.6, 7, true, 1, false)
		f.cornStalk(rng, bx+3, base+0.3, 29, 1.2, 8, true, 2, false)
	case cropWild:
		f.cornStalk(rng, bx-4.5, base+0.4, 19, -2.5, 5, true, 1, false)
		f.cornStalk(rng, bx+4.5, base-0.2, 22, 2.4, 6, true, 1, false)
		f.cornStalk(rng, bx, base+0.8, 26, 0.4, 7, true, 2, false)
		f.tuft(rng, bx, base+1, 16, 8, 3, bladeRamp)
	case cropWildPicked:
		f.cornStalk(rng, bx-4.5, base+0.4, 0, 0, 0, false, 0, true)
		f.cornStalk(rng, bx+4.5, base-0.2, 0, 0, 0, false, 0, true)
		f.cornStalk(rng, bx, base+0.8, 0, 0, 0, false, 0, true)
		f.tuft(rng, bx, base+1, 16, 8, 3, bladeRamp)
	}
	return f.done(0.6, bx, base)
}

// --- squash --------------------------------------------------------------------------------

// leafLobed is a squash leaf: broad, heart-shaped, its edge in shallow lobes.
var leafLobed = leafForm(func(t float64) float64 {
	return leafHeart(t) * (0.84 + 0.16*math.Abs(math.Sin(t*math.Pi*3.5)))
})

// squash is a ribbed fruit lying at art point (x, y), rx by ry.
func (f *flo) squash(x, y, rx, ry float64, rp ramp, striped bool, seed uint32) {
	s := aEll(x, y, rx, ry)
	f.q.fill(s, func(px, py int, _ float64) color.RGBA {
		nx := ((float64(px)+0.5)/fk - x) / rx
		l := s.lum(px, py, math.Min(rx, ry)*0.9*fk)*0.88 + 0.08
		// Ribs: grooves running from stalk to tip, bunched towards the sides.
		rib := math.Asin(clamp(nx, -1, 1)) / (math.Pi / 2) * 3.2
		fr := rib - math.Floor(rib)
		if fr < 0.16 {
			l -= 0.2
		} else if fr > 0.4 && fr < 0.7 {
			l += 0.06
		}
		col := toneOf(rp, l)
		if striped && fr > 0.3 && fr < 0.8 && hashf(px/2, py/5, seed) < 0.8 {
			col = toneOf(squashGreen, l*0.8+0.05) // dark green stripes on a pale one
		}
		return col
	})
	st := aCap(x-rx*0.1, y-ry*0.92, x-rx*0.3, y-ry-1.4, 0.5, 0.42)
	f.q.fill(st, shd(squashGreen, st, 0.8, -0.05))
	f.glint(x-rx*0.45, y-ry*0.5, rp.at(len(rp)-1))
}

func genSquashHD(look int) (*canvas, float64, float64) {
	f := newFlo(36, 26)
	rng := rand.New(rand.NewPCG(90, uint64(look)+1))
	const bx, base = 18.0, 22.0
	leaves, size, flowers, fruit := 8, 1.0, 2, 0
	striped := false
	switch look {
	case 0:
		leaves, size, flowers = 3, 0.5, 0
	case 1:
		leaves, size, flowers = 5, 0.75, 0
	case 2:
		leaves, size, flowers = 7, 0.95, 3
	case cropRipe:
		fruit = 2
	case cropWild:
		leaves, flowers, fruit, striped = 9, 2, 2, true
	case cropWildPicked:
		leaves, flowers = 7, 1
	}
	// The vine creeps out both ways along the ground, a leaf on a stalk at
	// every joint.
	type lf struct{ x, y, a, l float64 }
	var ls []lf
	for i := 0; i < leaves; i++ {
		t := (float64(i) + 0.5) / float64(leaves)
		a := math.Pi * (0.06 + 0.88*t)
		if i%2 == 1 {
			a = math.Pi - a
		}
		d := (3 + 6.5*math.Abs(math.Cos(a))) * size
		ls = append(ls, lf{bx + math.Cos(a)*d*1.25, base - 1 - math.Sin(a)*d*0.55, a + (rng.Float64()-0.5)*0.5, (4.4 + rng.Float64()*1.6) * (0.55 + 0.45*size)})
	}
	for _, l := range ls {
		f.stalk(squashLeaf, 0.34, 0.3, 0, [2]float64{bx, base}, [2]float64{lerpf(bx, l.x, 0.4), base - 0.5}, [2]float64{lerpf(bx, l.x, 0.8), l.y + 1.5}, [2]float64{l.x, l.y}, nil)
	}
	for i, l := range ls {
		f.leafAt(l.x, l.y, l.a, l.l, l.l*0.62, 0.05, leafLobed, squashLeaf, 5, (rng.Float64()-0.5)*0.12-0.04*float64(i%2))
	}
	for i := 0; i < flowers && i < len(ls); i++ { // orange trumpets
		l := ls[(i*3+1)%len(ls)]
		bloomHD(f.c, (l.x+math.Cos(l.a)*2)*fk, (l.y-2.2)*fk, 4.8, mkRamp(0xd47a14, 0xf2a424, 0xfcd04c, 0xffec94), hex(0xc8641c), rng.Float64())
	}
	rp := squashRamp
	if striped {
		rp = squashGreen[2:]
	}
	if fruit > 0 {
		f.squash(bx+5.5, base-0.6, 4.4, 3.3, rp, striped, 901)
	}
	if fruit > 1 {
		f.squash(bx-7, base+0.4, 3.1, 2.4, rp, striped, 902)
	}
	if look >= 1 { // tendrils, curling
		for _, s := range []float64{-1, 1} {
			x0, y0 := bx+s*9.5*size, base-2.5
			for k := 0; k < 10; k++ {
				a := float64(k) * 0.75
				r := 1.5 - float64(k)*0.12
				f.q.dot(x0+s*(float64(k)*0.25+math.Cos(a)*r), y0-math.Sin(a)*r*0.8, 0, 0, squashLeaf[4])
			}
		}
	}
	return f.done(0.62, bx, base)
}

// --- the sprites ------------------------------------------------------------------------------

var cropGens = [sim.NumCrops]func(look int) (*canvas, float64, float64){
	sim.CropRice:   genRiceHD,
	sim.CropCorn:   genCornHD,
	sim.CropSquash: genSquashHD,
}

var cropArt [sim.NumCrops][cropLooks]sprite

// cropLook is the sprite of crop c at look (a stage, ripe, wild or picked).
func cropLook(c sim.CropKind, look int) sprite {
	c %= sim.NumCrops
	if cropArt[c][look].img == nil {
		cropArt[c][look] = mkUp(cropGens[c](look))
	}
	return cropArt[c][look]
}

func init() {
	wild := func(c sim.CropKind) func(bool) (*canvas, float64, float64) {
		return func(full bool) (*canvas, float64, float64) {
			if full {
				return cropGens[c](cropWild)
			}
			return cropGens[c](cropWildPicked)
		}
	}
	floraGens[sim.FloraWildRice] = wild(sim.CropRice)
	floraGens[sim.FloraWildCorn] = wild(sim.CropCorn)
	floraGens[sim.FloraWildSquash] = wild(sim.CropSquash)
}

// newCropSprite is cropSprite for the crops that are not sweet potatoes.
func newCropSprite(e protocol.EntityView) (sprite, bool) {
	c := sim.CropKind(e.Var)
	if c == sim.CropPotato || c >= sim.NumCrops {
		return sprite{}, false
	}
	if e.Ready {
		return cropLook(c, cropRipe), true
	}
	return cropLook(c, min(int(e.Stage), sim.CropStages-1)), true
}
