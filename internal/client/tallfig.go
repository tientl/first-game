package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The clothes of the jointed figure (limbs.go), five and a half heads tall.
// Bare it is cavefig.go's: a fur loincloth and nothing else. The tunic and
// the mantle are things made and carried (sim.Wear). A man is broad in the
// shoulder: a belted tunic to the middle of the thigh in the player's
// colour, a fur mantle pinned with a brooch. A woman is slighter: an apron
// dress to below the knee over a lighter shift, pinned with two brooches
// and a string of beads, a fur collar. The arms and the legs are limbs.go's,
// the head tallhead.go's.
//
// Image pixels, the middle on column 27: the crown on row 27, the chin on
// 43, the shoulders on 49, the belt on 67, the hips on 74, the knees on 95
// and the soles on limbGround.

const (
	manHem   = 83.0 // the row a tunic's hem hangs to
	womanHem = 99.0 // and a dress's
	beltY    = 67.0
)

// skirtT is how far down the thigh the hem of what is worn hangs from: a
// tunic's or a dress's, or the fur's about the hips. See skirtOf.
func skirtT(female bool, wear sim.Wear) float64 {
	switch tunic := wear&sim.WearTunic != 0; {
	case tunic && female:
		return 1
	case tunic:
		return 0.5
	case female:
		return (skirtHemF - hipY) / thighLen
	}
	return (loinHem - hipY) / thighLen
}

// tallTorso is the tunic or the dress without the arms, facing d (never
// DirLeft), its skirt carried by the legs as sk has it: the shape the
// armour follows too. hemAt is the hem's row at column x.
func tallTorso(female bool, d sim.Dir, sk skirt) (torso shape, hemAt func(x float64) float64) {
	hem := manHem
	if female {
		hem = womanHem
	}
	if d == sim.DirRight {
		fx, bx, room := 35.0, 19.5, 6.0
		if female {
			fx, bx, room = 35.5, 18.5, 5.4
		}
		fy, by := hem, hem
		if sk.set {
			fx, bx = math.Max(fx, sk.fx+room), math.Min(bx, sk.bx-room)
			fy, by = hem+sk.fy, hem+sk.by
		}
		hemAt = func(x float64) float64 { return lerpf(by, fy, clamp01((x-bx)/(fx-bx))) }
		if female {
			return sPoly(23, 46.5, 31.5, 46.5, 34, 54.5, 31.8, 65, 33.5, 74, fx, fy, bx, by, 20.2, 74, 22, 65, 21.2, 54), hemAt
		}
		return sPoly(21, 46.5, 32.4, 46.5, 35.2, 54, 33.4, 67, 34, 74, fx, fy, bx, by, 20.2, 74, 21, 67, 19.4, 54), hemAt
	}
	l, r := hem+sk.dy[1], hem+sk.dy[0]
	mid := math.Max(l, r)
	hemAt = func(x float64) float64 {
		if x < 27 {
			return lerpf(l, mid, clamp01((x-16)/11))
		}
		return lerpf(mid, r, clamp01((x-27)/11))
	}
	if female {
		return sPoly(19.5, 46.5, 34.5, 46.5, 35, 54, 33, 65, 36, 75, 38.5, r, 27, mid, 15.5, l, 18, 75, 21, 65, 19, 54), hemAt
	}
	return sPoly(16, 46.5, 38, 46.5, 38, 55, 35.6, 67, 36.6, 74, 38, r, 27, mid, 16, l, 17.4, 74, 18.4, 67, 16, 55), hemAt
}

// tallBody draws the body: bare on U (cavefig.go), or under a tunic the
// clothes on S, in greys for the player's colour, and the neck and the belt
// on T; then the mantle on T, if one is worn. The hook is run before the
// mantle, for the arms.
func (g *figGen) tallBody(d sim.Dir, sk skirt) {
	f := &g.f
	female, up, side := g.look.Female, d == sim.DirUp, d == sim.DirRight
	if g.wear&sim.WearTunic == 0 {
		g.bareBody(d, sk)
		if g.hook != nil {
			g.hook()
		}
	} else {
		g.tunic(d, sk)
	}
	if g.wear&sim.WearMantle == 0 {
		return
	}
	switch {
	case female && side:
		m := sBox(21.5, 45.5, 33, 50.5, 2.4).jag(1.1, 3, 48.5, 23)
		f.fill(g.T, m, furShade(furHD, m, 3.5, 0.02, 33))
	case female:
		m := sBox(17, 45.5, 37, 50.5, 2.4).jag(1.1, 3, 48.5, 23)
		f.fill(g.T, m, furShade(furHD, m, 3.5, 0.02, 33))
	case side:
		m := sBox(20, 45.5, 34.6, 50.5, 3).or(sEll(27, 50.5, 7.4, 4.8)).jag(1.4, 3.4, 51.5, 21)
		f.fill(g.T, m, furShade(furHD, m, 4, 0, 31))
		g.brooch(33.2, 52, 2.2, true)
	case up:
		m := sBox(12.5, 45.5, 41.5, 57, 3.5).jag(1.7, 4, 54, 21)
		f.fill(g.T, m, furShade(furHD, m, 5, 0, 31))
	default:
		m := sBox(12.5, 45.5, 41.5, 50, 2.5).or(sEll(16.2, 50.5, 5.2, 4.6)).or(sEll(37.8, 50.5, 5.2, 4.6)).jag(1.4, 3.4, 51.5, 21)
		f.fill(g.T, m, furShade(furHD, m, 4, 0, 31))
		g.brooch(19.5, 52, 2.4, true)
	}
}

// tunic draws a man's tunic or a woman's dress on S, and the neck, the belt
// and the dress's brooches on T; the hook is run after the belt.
func (g *figGen) tunic(d sim.Dir, sk skirt) {
	f := &g.f
	female, up, side := g.look.Female, d == sim.DirUp, d == sim.DirRight
	torso, hemAt := tallTorso(female, d, sk)
	bias := 0.03
	if side {
		bias = -0.12 // a step down from the near arm
	}
	if !female {
		f.fill(g.S, torso, func(x, y int, _ float64) color.RGBA {
			i := int((torso.lum(x, y, 7) + bias) * 6)
			dx := float64(x) + 0.5 - 27
			h := int(math.Round(hemAt(float64(x) + 0.5)))
			switch {
			case y >= h-4: // a pale band along the hem, stitched
				i = max(i, 2) + 1
				if y == h-3 && x%3 == 0 {
					i -= 2
				}
				if y >= h-1 {
					i--
				}
			case y >= int(beltY)+3:
				for _, k := range [4]float64{-0.3, -0.1, 0.12, 0.33} {
					if math.Abs(dx-k*(float64(y)-50)) < 0.5 {
						i-- // folds of the skirt
					}
				}
				if y == int(beltY)+3 {
					i-- // under the belt
				}
			case !up && !side && x == 27 && y < 59:
				i -= 2 // the slit at the neck
			case !up && !side && (x == 25 || x == 29) && y < 59:
				i++ // and its trim
			case y >= 63 && (x+y)%7 == 0:
				i-- // gathered into the belt
			}
			return shirtHD.at(max(i, 0))
		})
	} else {
		if !side {
			bias = -0.1
		} else {
			bias = -0.2
		}
		f.fill(g.S, torso, func(x, y int, _ float64) color.RGBA {
			// A light shift, and over it the apron dress, darker.
			i := int((torso.lum(x, y, 8) + bias) * 6)
			dx := float64(x) + 0.5 - 27
			h := int(math.Round(hemAt(float64(x) + 0.5)))
			switch {
			case y >= h-3: // the shift's hem under the apron
				i = max(i+2, 4)
				if y == h-3 {
					i = 1
				}
			case y >= h-7 && y <= h-5: // a woven band
				i = 1
				if y == h-6 && x%4 == 1 {
					i = 5
				} else if y != h-6 && x%4 == 3 {
					i = 3
				}
			case y == 65 || y == 66: // tied at the waist
				i -= 1 + (66 - y)
			case side:
			case !up && y < 57 && math.Abs(dx) < (57-float64(y))*0.5: // the shift at the neck
				i = 5
				if y < 51 {
					i = 4
				}
			case !up && y < 64 && math.Abs(math.Abs(dx)-5) < 1.2: // the straps
				i--
			}
			if up && y < 64 { // the straps cross on the back
				if c := 5.5 - 11*(float64(y)-48)/16; math.Abs(dx-c) < 1.2 || math.Abs(dx+c) < 1.2 {
					i--
				}
			}
			if y > 67 && y < h-7 {
				ks := []float64{-0.3, -0.12, 0.1, 0.3}
				if side {
					ks = []float64{-0.24, 0.02, 0.26}
				}
				for _, k := range ks {
					if math.Abs(dx-k*(float64(y)-46)) < 0.5 {
						i-- // folds
					}
				}
			}
			return shirtHD.at(max(i, 0))
		})
	}

	switch {
	case female:
	case side:
		g.tallBelt(21, 33.6, false)
		for y := int(beltY); y <= int(beltY)+2; y++ { // the buckle, round the front
			f.dot(g.T, 31, y, goldHD[int(beltY)+3-y])
			f.dot(g.T, 32, y, goldHD[int(beltY)+4-y])
		}
	default:
		g.tallBelt(18.2, 35.8, !up)
	}
	if g.hook != nil {
		g.hook()
	}

	// The neck, in the chin's shade.
	neck := sBox(24.4, 40, 29.6, 48.5, 1)
	if side {
		neck = sPoly(24.6, 40, 30.4, 40, 30.6, 48.5, 24, 48.5)
	}
	if female {
		neck = neck.and(sBox(25, 0, 30, 60, 0))
	}
	f.fill(g.T, neck, func(x, y int, _ float64) color.RGBA {
		if y < 45 || x >= 29 {
			return g.sk[2]
		}
		return g.sk[3]
	})

	switch {
	case female && side:
		g.brooch(32, 53, 2, false)
	case female:
		if !up {
			// Two oval brooches with a string of beads between.
			for i := 0; i < 5; i++ {
				t := float64(i+1) / 6
				bx, by := lerpf(22.5, 31.5, t), 54+4*4*t*(1-t)
				col := beadColors[i%len(beadColors)]
				f.n++
				f.dot(g.T, int(bx), int(by), col)
				f.dot(g.T, int(bx), int(by)+1, mix(col, colInk, 0.35))
			}
			g.brooch(22, 53, 2, false)
			g.brooch(32, 53, 2, false)
		}
	}
}

// tallBelt draws a man's leather belt across columns x0 to x1; buckle adds
// the buckle in front and the end hanging from it.
func (g *figGen) tallBelt(x0, x1 float64, buckle bool) {
	f := &g.f
	const y0 = int(beltY)
	b := sBox(x0, beltY, x1, beltY+3, 0.4)
	f.fill(g.T, b, func(x, y int, _ float64) color.RGBA {
		i := 3 - int(float64(x)-x0)*3/int(x1-x0) // darker to the right
		switch y - y0 {
		case 0:
			i += 2
		case 2:
			i--
		default:
			if x%3 == 1 {
				i++ // stitches
			}
		}
		return leatherHD.at(i + 1)
	})
	if !buckle {
		return
	}
	end := sBox(29.5, beltY+2, 32, beltY+11, 1)
	f.fill(g.T, end, func(x, y int, _ float64) color.RGBA {
		if y >= y0+9 {
			return goldHD[2] // a metal tip
		}
		return leatherHD.at(3 - (x - 29))
	})
	ring := sBox(23.5, beltY-1, 30.5, beltY+4, 1.6)
	f.fill(g.T, ring, func(x, y int, d float64) color.RGBA {
		if d < -1.3 {
			return leatherHD[1] // the belt through it
		}
		return toneOf(goldHD, ring.lum(x, y, 1.5)+0.08)
	})
	f.dot(g.T, 27, y0+1, goldHD[4]) // the prong
	f.dot(g.T, 28, y0+1, goldHD[3])
	f.dot(g.T, 24, y0-1, goldHD[5])
}
