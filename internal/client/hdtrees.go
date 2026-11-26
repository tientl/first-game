package client

import (
	"math"
	"math/rand/v2"
)

// genPineHD is genPineOf at HD: the same tiers on the same trunk, each tier
// a skirt of drooping boughs with their own lit and shaded sides, tips of
// uneven length and a grain of needles; snow lies bough by bough.
func genPineHD(variant int, s float64, seed uint64, snowy bool) sprite {
	const k = HDDensity
	rng := rand.New(rand.NewPCG(seed, 121))
	S := s * k
	c := markHD(newCanvas(int(math.Ceil(34*S)), int(math.Ceil(62*S))))
	bx, by := float64(c.w/2), float64(c.h)-3*k // ground point
	tw := math.Round(trunkHalf(s)*0.8) * k     // half the trunk's width
	drawLimbHD(c, limb{bx, by, bx, by - 6*S, bx, by - 12*S, tw, tw * 0.85}, trunkRamp, 0.75, 7)
	for _, side := range []float64{-1, 1} { // the foot spreads a little
		drawLimbHD(c, limb{bx, by - 3*S, bx + side*tw*0.6, by - S, bx + side*tw*1.5, by, tw * 0.5, 0.5 * k}, trunkRamp, 0.75, 7)
	}
	tiers := 4 + variant%3
	lean := (rng.Float64() - 0.5) * 2.5 * S
	tex := 120 + uint32(variant)
	for t := range tiers {
		fk, ft := float64(t), float64(tiers)
		yb := by - (8+fk*36/ft)*S
		yt := yb - (17-fk*0.5)*S
		hw := (14 - fk*10/ft) * S * (0.85 + rng.Float64()*0.3)
		ox := bx + lean*fk/ft + (rng.Float64()-0.5)*1.5*k
		nb := max(5, int(hw*2/7)) // boughs across the tier
		for y := int(yt); y <= int(yb)+3*k; y++ {
			for x := int(ox - hw - 2); x <= int(ox+hw+2); x++ {
				px := float64(x) + 0.5 - ox
				f := (float64(y) + 0.5 - yt) / (yb - yt)
				if f < 0 {
					continue
				}
				// Where across the skirt, -1 to 1, and in which bough.
				a := px / (hw*math.Min(f, 1) + 0.3)
				if a < -1 || a > 1 {
					continue
				}
				fb := (a + 1) / 2 * float64(nb)
				b := min(nb-1, int(fb))
				u := fb - float64(b) // 0 to 1 across the bough
				// A bough ends in a rounded tip, some longer than others.
				end := 1 + (hashf(b, t, tex+3)-0.35)*0.16 - (u-0.5)*(u-0.5)*0.5
				if f > end {
					continue
				}
				l := 0.62 - px/hw*0.38 + (1-f)*0.12 + (vnoise(float64(x)*0.2, float64(y)*0.27, tex)-0.5)*0.26
				l += (0.45 - u) * 0.22 // each bough lit from the left
				if u > 0.86 {
					l -= 0.16 // the seam to the next
				}
				if f > end-0.2 {
					l -= 0.2 // the underside
				}
				l += (hashf(x+int(u*3), y/3, tex+1) - 0.5) * 0.22 // needles: short strokes down the bough
				if hashf(x, y, 121) < 0.04 {
					l += 0.15
				}
				col := pineRamp.pick(l, x, y)
				if snowy {
					edge := 0.34 + hashf(b, t, tex+5)*0.2 - (u-0.5)*(u-0.5)*0.4
					if f < edge || f < 0.75 && u < 0.6 && vnoise(float64(x)*0.25, float64(y)*0.3, tex+7) > 0.66 {
						col = snowRamp.pick(l+0.38, x, y) // snow on the bough
					}
				}
				c.put(x, y, col)
			}
		}
	}
	c.outline(0.8)
	return trimmed(c, bx, by)
}
