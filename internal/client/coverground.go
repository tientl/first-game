package client

import (
	"image/color"
	"math"
)

// The ground cover baked into the grass and the floor of the woods (see
// grassGround and forestGround): the sward thick in places and thin in
// others, patches of clover, cushions of moss, worn spots of bare earth with
// grass growing ragged over their rims, and in the woods drifts of fallen
// leaves between the moss. Each is a pixel of the bake, from noise of the
// world's position, so the ground costs no more memory than before.

var (
	soilHD   = mkRamp(0x4a4030, 0x5a4e3a, 0x6b5e46, 0x7c6e53, 0x8d7f62, 0xa09272).sub(2)
	cloverHD = mkRamp(0x2a5a2c, 0x346a32, 0x40793a, 0x4f8b42, 0x63a050, 0x7cb560).sub(2)
	mossHD   = mkRamp(0x34521f, 0x405f24, 0x4e6f2a, 0x5d8031, 0x6e9139, 0x84a346).sub(2)
	duffHD   = mkRamp(0x221b11, 0x2d2416, 0x3a2e1b, 0x473921, 0x554428).sub(2)
)

// coverSward is how thick the grass is at (u, v), in tiles: from -1, thin
// and short, to 1, deep and lush, over patches a few tiles across.
func coverSward(u, v float64) float64 {
	return (fbm2(u*0.33+13, v*0.42+5, 39, 2)-0.5)*2.2 + (vnoise(u*1.1+3, v*1.3, 40)-0.5)*0.5
}

// cloverAt is a patch of clover: trefoil leaves crowded together, each of
// three round leaflets lit from the upper left with a dark crease between.
func cloverAt(hx, hy int, dens float64) (color.RGBA, bool) {
	const cw, ch = 7, 6
	for dy := 0; dy <= 1; dy++ { // a leaf may reach into the next cell down
		for dx := 0; dx <= 1; dx++ {
			cx, cy := fdiv(hx, cw)-dx, fdiv(hy, ch)-dy
			h := hash2(int32(cx), int32(cy), 851)
			if float64(h&0xffff)/65536 >= dens {
				continue
			}
			ox := float64(cx*cw) + 2 + float64(h>>16&7)*0.6
			oy := float64(cy*ch) + 2 + float64(h>>19&7)*0.5
			turn := float64(h>>22&63) / 63 * 2 * math.Pi
			px, py := float64(hx)+0.5-ox, float64(hy)+0.5-oy
			for q := range 3 {
				a := turn + float64(q)*2*math.Pi/3
				lx, ly := px-math.Cos(a)*1.5, (py-math.Sin(a)*1.1)*1.25
				d := lx*lx + ly*ly
				if d > 2.4 {
					continue
				}
				l := 0.55 - 0.18*lx - 0.22*ly + float64(h>>28&3)*0.06
				if d > 1.5 {
					l -= 0.25 // the leaflet's rim
				}
				if math.Abs(px*math.Sin(a)-py*math.Cos(a)) < 0.35 && d < 1.2 {
					l -= 0.2 // the crease down its middle
				}
				return cloverHD.g(l, hx, hy), true
			}
		}
	}
	return color.RGBA{}, false
}

// mossAt is a cushion of moss: velvet, a fine grain of tiny lit tips over
// dark, no blades.
func mossAt(hx, hy int, u, v float64) color.RGBA {
	l := 0.45 + (vnoise(u*9, v*11, 852)-0.5)*0.5
	if hashf(hx, hy, 853) < 0.18 {
		l += 0.22 // a tip in the light
	} else if hashf(hx+1, hy+1, 853) < 0.18 {
		l -= 0.15 // and its shadow
	}
	return mossHD.g(l, hx, hy)
}

// soilAt is bare earth: fine grains, a pebble here and there with its
// shadow.
func soilAt(hx, hy int, u, v float64) color.RGBA {
	l := 0.5 + (vnoise(u*5, v*6, 854)-0.5)*0.5 + (hashf(hx, hy, 855)-0.5)*0.18
	switch part, pl := pebble(hx, hy, 11, 9, 856, 0.16, 1.2, 2.4); part {
	case pebStone:
		return sandStoneHD.g(pl, hx, hy)
	case pebShadow:
		l -= 0.3
	}
	return soilHD.g(l, hx, hy)
}

// coverPatch is the cover baked over the grass at pixel (hx, hy), (u, v)
// in tiles, where it is not plain turf: clover or moss (w 1), or worn earth
// showing through the grass by w (0 to 1).
func coverPatch(hx, hy int, u, v float64) (col color.RGBA, w float64) {
	fine := vnoise(float64(hx)/5, float64(hy)/4, 857)
	// Worn spots: the grass thinning, a muted earth showing more and more
	// between the blades towards the middle; the blades stand over it.
	wear := vnoise(u*1.05+3, v*1.25+9, 858) + (fine-0.5)*0.06
	if s := (wear - 0.79) / 0.12; s > 0 {
		w = smooth(clamp01(s)) * (0.55 + 0.45*hashf(hx, fdiv(hy, 2), 860))
		if blade(hx+1, hy+2, 4, 8, 859, 0.75*(1-s)) != bladeNone {
			w *= 0.25
		}
		return soilAt(hx, hy, u, v), w * 0.9
	}
	// Clover, in patches.
	if c := vnoise(u*0.9+21, v*1.1+4, 861) + (fine-0.5)*0.08; c > 0.68 {
		if col, ok := cloverAt(hx, hy, clamp01((c-0.68)/0.06)*0.8); ok {
			return col, 1
		}
	}
	// Moss, where it is damp and low.
	if m := vnoise(u*0.9+51, v*1.1+17, 862) + (fine-0.5)*0.12; m > 0.76 {
		return mix(mossAt(hx, hy, u, v), grassHD.at(5), 0.25), 0.9 * smooth(clamp01((m-0.76)/0.07))
	}
	return color.RGBA{}, 0
}

// forestPatch is the cover baked over the floor of the woods at (hx, hy):
// cushions of moss, and dark bare duff where the litter has rotted.
func forestPatch(hx, hy int, u, v float64) (color.RGBA, bool) {
	fine := vnoise(float64(hx)/5, float64(hy)/4, 863)
	if m := vnoise(u*0.55+7, v*0.7+29, 864) + (fine-0.5)*0.14; m > 0.64 {
		c := mossAt(hx, hy, u, v)
		if m < 0.67 && hashf(hx, hy, 865) < 0.5 {
			c = mix(c, forestHD.at(4), 0.4)
		}
		return c, true
	}
	if d := vnoise(u*0.8+61, v*0.95+3, 866) + (fine-0.5)*0.1; d > 0.82 {
		l := 0.55 + (vnoise(u*6, v*7, 867)-0.5)*0.6 + (hashf(hx, hy, 868)-0.5)*0.2
		return mix(duffHD.g(l, hx, hy), forestHD.at(3), 0.25), true
	}
	return color.RGBA{}, false
}

// litterThick is how much of the woods' floor fallen leaves cover at (u, v):
// drifts of them, and thin stretches.
func litterThick(u, v float64) float64 {
	return 0.35 + 1.5*vnoise(u*0.45+71, v*0.55+11, 869)
}
