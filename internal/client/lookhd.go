package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The player at Full HD: the figure of look.go redrawn HDDensity times
// finer. It is the same settler in the same frame (18 by 26 art pixels and
// the headroom, the feet at (9, bodyFeet), the head on rows 5 to 15), so
// everything that goes with it (armour, punches, what is held, mounts, the
// bed) lines up as before; but each art pixel is now three by three, which
// is room for a face, locks of hair, tufts of fur, stitches and buckles.
//
// It is drawn in image pixels: figW by figH of them, the soles on row figFeet.
// The layers are look.go's: B hair behind the body, L the legs, S the
// clothes in greys for the player's colour, T the rest.

const (
	figW    = bodyW * HDDensity
	figH    = bodyH * HDDensity
	figFeet = bodyFeet * HDDensity
)

// playerHD picks the HD figure; off, players are drawn with look.go's
// density-1 art.
var playerHD = true

// walkSwing is how far the arms and legs are swung on each walk frame, -1
// to 1: frame 0 and 4 stand, 2 is the left step, 6 the right.
var walkSwings = [walkFrames]float64{0, -0.7, -1, -0.7, 0, 0.7, 1, 0.7}

func walkSwing(f int) float64 { return walkSwings[((f%walkFrames)+walkFrames)%walkFrames] }

// walkFrameAt is the walk frame on game frame n: a stride takes 28 frames,
// as it always did.
func walkFrameAt(n int) int { return (n * 2 / 7) % walkFrames }

// walkBob is how far the body rises on walk frame f, in art pixels.
func walkBob(f int) float64 { return math.Round(math.Abs(walkSwing(f))*HDDensity) / HDDensity }

// legacyFrame is the frame of look.go's four-frame walk that goes with
// walk frame f.
func legacyFrame(f int) int {
	switch sw := walkSwing(f); {
	case sw < 0:
		return 1
	case sw > 0:
		return 3
	}
	return 0
}

var (
	shirtHD   = mkRamp(0x4c4c4c, 0x6e6e6e, 0x929292, 0xb6b6b6, 0xd8d8d8, 0xf8f8f8)
	trouserHD = ext6(trouserRamp)
	wrapHD    = ext6(wrapRamp)
	furHD     = mkRamp(0x221d21, 0x35302f, 0x514947, 0x766d6b, 0x9c938d, 0xc6beb4, 0xe6e0d6)
	leatherHD = ext6(leatherRamp)
	goldHD    = mkRamp(0x5a370c, 0x8a5a14, 0xd09a26, 0xf6cf52, 0xfff2b0, 0xfffdf0)
	bootHD    = mkRamp(0x1c120e, 0x2a1c14, 0x45301f, 0x624530, 0x7f5c40, 0xa07a56)
	garnet    = mkRamp(0x6a1420, 0xb02a34, 0xf06a6a)
	eyeWhite  = hex(0xf7f5f0)
	eyeShade  = hex(0xc4c8d8)
)

// figGen draws one frame of one look.
type figGen struct {
	f          figHD
	B, L, S, T *figLayer
	look       sim.Look
	sk, hr, br ramp // skin, hair and beard, six tones each
	seed       uint32

	// The jointed figure (limbs.go): the layers of the arms, what to draw
	// between the belt and the mantle, and how far what hangs trails.
	FA, A, AO, TO *figLayer
	hook          func()
	lagX, lagY    float64
}

// lagged is s with its lower end trailing by the figure's lag: hair that
// hangs, the tip of a braid.
func (g *figGen) lagged(s shape) shape {
	if g.lagX == 0 && g.lagY == 0 {
		return s
	}
	lx, ly := g.lagX, g.lagY
	t := func(y float64) float64 { v := clamp01((y - 40) / 22); return v * v }
	return shape{s.x0 - math.Abs(lx), s.y0 - math.Abs(ly), s.x1 + math.Abs(lx), s.y1 + math.Abs(ly), func(x, y float64) float64 {
		k := t(y)
		return s.d(x-lx*k, y-ly*k)
	}}
}

func skin6(sk ramp) ramp {
	return ramp{mix(sk[0], colInk, 0.3), sk[0], sk[1], sk[2], sk[3], mix(sk[3], colWhite, 0.32)}
}

// beardRamp is the beard's colours, dark to light: a step off the hair's,
// as beardPalette has it.
func beardRamp(l sim.Look) ramp {
	hr := hairColors[l.HairColor].ramp
	if luma(hr[2]) < 50 {
		return ramp{hr[1], mix(hr[2], hr[3], 0.35), hr[3], mix(hr[3], colWhite, 0.25)}
	}
	return ramp{mix(hr[0], colInk, 0.3), hr[0], hr[1], hr[2]}
}

func newFigGen(l sim.Look, w, h int) *figGen {
	l = l.Sanitized()
	g := &figGen{look: l,
		B: newFigLayer(w, h), L: newFigLayer(w, h), S: newFigLayer(w, h), T: newFigLayer(w, h),
		sk: skin6(skinTones[l.Skin].ramp), hr: ext6(hairColors[l.HairColor].ramp), br: ext6(beardRamp(l)),
		seed: uint32(l.Hair)*31 + uint32(l.HairColor)*7 + 5,
	}
	return g
}

// finish gives the layers their lines: a fine dark one where a part lies
// over another, and the outline round the whole figure.
func (g *figGen) finish() {
	innerLines(0.4, g.B, g.S, g.T)
	innerLines(0.4, g.L)
	g.L.c.outline(0.72)
	rimLayers(g.B.c, g.S.c, g.T.c)
}

// playerLayersHD draws one direction and walk frame of look l at HD
// density, as playerLayersPose does at density 1; d is not DirLeft.
func playerLayersHD(l sim.Look, d sim.Dir, frame int, asleep bool, hide armSide) (B, L, S, T *canvas) {
	g := newFigGen(l, figW, figH)
	g.body(d, frame, hide)
	g.head(d, asleep)
	g.finish()
	return g.B.c, g.L.c, g.S.c, g.T.c
}

// genPlayerHD is genPlayer at HD density.
func genPlayerHD(l sim.Look, d sim.Dir, frame int) (back, legs, shirt, top sprite) {
	if d == sim.DirLeft {
		b, lg, s, t := genPlayerHD(l, sim.DirRight, frame)
		return mirrorSprite(b), mirrorSprite(lg), mirrorSprite(s), mirrorSprite(t)
	}
	B, L, S, T := playerLayersHD(l, d, frame, false, armNone)
	return figSprite(B), figSprite(L), figSprite(S), figSprite(T)
}

// figSprite makes a sprite of an HD figure layer, anchored between the feet.
func figSprite(c *canvas) sprite { return spriteHD(c.image(), 9, bodyFeet) }

// --- the body ------------------------------------------------------------------

// hdTorso is the tunic or the dress without the arms, facing d (never
// DirLeft): the shape the armour follows too. hemL and hemR lift the dress's
// hem over a stepping foot.
func hdTorso(female bool, d sim.Dir, hemL, hemR float64) shape {
	side := d == sim.DirRight
	switch {
	case !female && !side:
		return sPoly(15, 47.5, 39, 47.5, 38.6, 57, 40.2, 66, 13.8, 66, 15.4, 57)
	case !female:
		return sPoly(18, 47.5, 36, 47.5, 36.2, 57, 37.2, 66, 16.8, 66, 17.8, 57)
	case !side:
		return sPoly(18, 47.5, 36, 47.5, 36.4, 60, 39.6, 69, 42.6, 78-hemR, 27, 78-math.Min(hemL, hemR), 11.4, 78-hemL, 14.4, 69, 17.6, 60)
	}
	return sPoly(21, 47.5, 36, 47.5, 36, 61, 38.4, 67, 39.4, 78, 14.6, 78, 17.2, 67, 20.6, 60)
}

// hdNearArm is the near arm and its hand, seen from the side on a walk
// swing of sw: where the armour leaves a gap.
func hdNearArm(sw float64) (sleeve, hand shape) {
	ex, wx := 27-1.5*sw, 27-4.6*sw
	sleeve = sPath(3, 3, 27, 50.5, ex, 56.5, wx, 60.5)
	hx := math.Round(24 - 6*sw)
	hand = sBox(hx, 63, hx+6, 69, 2.2)
	return sleeve.or(sCap(wx, 60.5, hx+3, 63.5, 3, 3)), hand
}

func (g *figGen) body(d sim.Dir, frame int, hide armSide) {
	f := &g.f
	sw := walkSwing(frame)
	female, up := g.look.Female, d == sim.DirUp
	sk := g.sk

	// leg is a trouser leg nine pixels wide from column x0, raised lift
	// pixels: cloth, cross-bound wraps, then a boot with a fur cuff, its toe
	// pointing toe pixels sideways. dim darkens the far leg.
	leg := func(x0, lift, dim, toe float64) {
		y := -math.Round(lift)
		tr := sBox(x0+0.5, 55, x0+8.5, 72+y, 2)
		f.fill(g.L, tr, bevel(trouserHD, tr, 4, -dim))
		wr := sBox(x0+0.2, 67.6+y, x0+8.8, 79+y, 1.5)
		f.fill(g.L, wr, func(px, py int, _ float64) color.RGBA {
			i := int((wr.lum(px, py, 4) - dim + 0.04) * 6)
			a, b := px-int(x0)+60, py-int(y)
			if (a+b)%6 == 0 || (a-b)%6 == 0 {
				i -= 2 // the bindings, crossed
			}
			return wrapHD.at(max(i, 0))
		})
		boot := sBox(x0-0.3, 78.4+y, x0+9.3, 84+y, 2.4).or(sEll(x0+4.5+toe, 81.7+y, 5.6, 2.5)).and(sAbove(84 + y))
		f.fill(g.L, boot, func(px, py int, _ float64) color.RGBA {
			if py >= 83+int(y) {
				return bootHD[0] // the sole
			}
			return toneOf(bootHD, boot.lum(px, py, 3.5)-dim-0.06)
		})
		cuff := sBox(x0-1.2, 74.4+y, x0+10.2, 79.6+y, 2.6).jag(1.3, 3.6, 77+y, 5)
		f.fill(g.L, cuff, furShade(furHD, cuff, 3, -dim-0.02, 11))
	}
	// hand is a fist six pixels square with its top-left at (x, y).
	hand := func(l *figLayer, s shape, bias float64) {
		f.fill(l, s, bevel(sk, s, 3, bias))
		x, y := int(s.x0), int(s.y0)
		f.dot(l, x+2, y+3, sk.at(2)) // between the thumb and the fingers
		f.dot(l, x+3, y+3, sk.at(2))
		f.dot(l, x+3, y+4, sk.at(2))
	}
	// sleeveShade is a sleeve's cloth, with a turned cuff on a man's.
	sleeveShade := func(s shape, bias float64, cuffY int) shader {
		return func(x, y int, _ float64) color.RGBA {
			i := int((s.lum(x, y, 3) + bias) * 6)
			if !female && y >= cuffY {
				i--
				if y == cuffY {
					i += 2
				}
			}
			return shirtHD.at(max(i, 0))
		}
	}

	torso := hdTorso(female, d, 0, 0)
	switch d {
	case sim.DirDown, sim.DirUp:
		liftL, liftR := 3*math.Max(0, -sw), 3*math.Max(0, sw)
		leg(18, liftL, 0, -2.6)
		leg(27, liftR, 0.1, 2.6)
		if female {
			torso = hdTorso(female, d, math.Round(liftL*0.7), math.Round(liftR*0.7))
		}

		if !female {
			f.fill(g.S, torso, func(x, y int, _ float64) color.RGBA {
				i := int((torso.lum(x, y, 9) + 0.03) * 6)
				switch {
				case y >= 63: // a pale band along the hem, stitched
					i = max(i, 2) + 1
					if y == 64 && x%3 == 0 {
						i -= 2
					}
					if y == 65 {
						i--
					}
				case y >= 60:
					if x == 19 || x == 26 || x == 32 || x == 36 {
						i-- // folds of the skirt
					}
					if y == 60 {
						i-- // under the belt
					}
				case !up && x == 27 && y < 56:
					i -= 2 // the slit at the neck
				case !up && (x == 25 || x == 29) && y < 56:
					i++ // and its trim
				}
				return shirtHD.at(max(i, 0))
			})
		} else {
			hemAt := func(x int) int {
				if x < 27 {
					return 78 - int(math.Round(liftL*0.7))
				}
				return 78 - int(math.Round(liftR*0.7))
			}
			f.fill(g.S, torso, func(x, y int, _ float64) color.RGBA {
				// A light shift, and over it the apron dress, darker.
				i := int((torso.lum(x, y, 10) - 0.1) * 6)
				hem := hemAt(x)
				dx := float64(x) + 0.5 - 27
				switch {
				case y >= hem-3: // the shift's hem under the apron
					i = max(i+2, 4)
					if y == hem-3 {
						i = 1
					}
				case y >= hem-7 && y <= hem-5: // a woven band
					i = 1
					if y == hem-6 && x%4 == 1 {
						i = 5
					} else if y != hem-6 && x%4 == 3 {
						i = 3
					}
				case y == 60 || y == 61: // tied at the waist
					i -= 1 + (61 - y)
				case y < 60 && (math.Abs(math.Abs(dx)-6.5) < 1.6): // the straps
					if up {
						break
					}
					i--
				case !up && y < 57 && math.Abs(dx) < (57-float64(y))*0.55: // the shift at the neck
					i = 5
					if y < 51 {
						i = 4
					}
				}
				if up && y < 60 { // the straps cross on the back
					if c := 7 - 14*(float64(y)-48)/12; math.Abs(dx-c) < 1.4 || math.Abs(dx+c) < 1.4 {
						i--
					}
				}
				if y > 62 && y < hem-7 {
					for _, k := range [4]float64{-0.36, -0.13, 0.15, 0.38} {
						if math.Abs(dx-k*(float64(y)-44)) < 0.5 {
							i-- // folds
						}
					}
				}
				return shirtHD.at(max(i, 0))
			})
		}

		// Arms swing opposite to the legs.
		lx, rx := 9.0, 39.0
		if female {
			lx, rx = 12, 36
		}
		la, ra := math.Round(-3*sw), math.Round(3*sw)
		if hide != armOff && hide != armBoth {
			s := sBox(lx, 47.5, lx+6, 63+la, 2.6)
			f.fill(g.S, s, sleeveShade(s, 0.12, int(60+la)))
			hand(g.T, sBox(lx, 63+la, lx+6, 69+la, 2.2), 0.1)
		}
		if hide != armHand && hide != armBoth {
			s := sBox(rx, 47.5, rx+6, 63+ra, 2.6)
			f.fill(g.S, s, sleeveShade(s, -0.16, int(60+ra)))
			hand(g.T, sBox(rx, 63+ra, rx+6, 69+ra, 2.2), -0.12)
		}

		if !female {
			g.belt(15, 39, !up)
		}
		if g.hook != nil {
			g.hook()
		}
		if !female {
			if up {
				m := sBox(8, 46.5, 46, 56.5, 3.5).jag(1.8, 4.2, 53, 21)
				f.fill(g.T, m, furShade(furHD, m, 6, 0, 31))
			} else {
				m := sBox(8, 46.5, 46, 51.5, 2.5).or(sEll(13.5, 52, 6.4, 5.2)).or(sEll(40.5, 52, 6.4, 5.2)).jag(1.6, 3.8, 53, 21)
				f.fill(g.T, m, furShade(furHD, m, 5, 0, 31))
				g.brooch(16.5, 53, 3.1, true)
			}
		} else {
			m := sBox(11, 46.5, 43, 52, 2.6).jag(1.3, 3.6, 49.5, 23)
			f.fill(g.T, m, furShade(furHD, m, 4, 0.02, 33))
			if !up {
				// Two oval brooches with a string of beads between.
				for i := 0; i < 7; i++ {
					t := float64(i+1) / 8
					bx, by := lerpf(21, 33, t), 55.5+5.5*4*t*(1-t)
					col := beadColors[i%len(beadColors)]
					f.n++
					f.dot(g.T, int(bx), int(by), col)
					f.dot(g.T, int(bx), int(by)+1, mix(col, colInk, 0.35))
				}
				g.brooch(20.5, 54.5, 2.6, false)
				g.brooch(33.5, 54.5, 2.6, false)
			}
		}

	case sim.DirRight:
		near, far := math.Round(21+6*sw), math.Round(24-6*sw)
		var liftN, liftF float64
		switch frame {
		case 3, 5:
			liftN = 2
		case 7, 1:
			liftF = 2
		}
		leg(far, liftF, 0.16, 2.8)
		leg(near, liftN, 0, 2.8)

		// The far hand shows past the body as it swings the other way.
		if sw > 0.3 {
			h := sBox(35, 57+3*(1-sw), 39.5, 63+3*(1-sw), 2)
			f.fill(g.B, h, bevel(sk, h, 3, -0.25))
		} else if sw < -0.3 {
			hx := 14.5
			if female {
				hx = 17.5
			}
			h := sBox(hx, 57+3*(1+sw), hx+4.5, 63+3*(1+sw), 2)
			f.fill(g.B, h, bevel(sk, h, 3, -0.25))
		}

		f.fill(g.S, torso, func(x, y int, _ float64) color.RGBA {
			// The body is a step down from the near arm.
			i := int((torso.lum(x, y, 8) - 0.14) * 6)
			dx := float64(x) + 0.5 - 27
			switch {
			case !female && y >= 63:
				i = max(i, 2) + 1
				if y == 64 && x%3 == 0 {
					i -= 2
				}
				if y == 65 {
					i--
				}
			case !female && y >= 60:
				if x == 21 || x == 31 {
					i--
				}
				if y == 60 {
					i--
				}
			case female && y >= 75:
				i = max(i+2, 4)
				if y == 75 {
					i = 1
				}
			case female && y >= 71 && y <= 73:
				i = 1
				if y == 72 && x%4 == 1 {
					i = 5
				} else if y != 72 && x%4 == 3 {
					i = 3
				}
			case female && (y == 60 || y == 61):
				i -= 1 + (61 - y)
			case female && y > 62:
				for _, k := range [3]float64{-0.3, 0.02, 0.3} {
					if math.Abs(dx-1-k*(float64(y)-44)) < 0.5 {
						i--
					}
				}
			}
			return shirtHD.at(max(i, 0))
		})
		if !female {
			g.belt(18, 36, false)
			for y := 57; y <= 59; y++ { // the buckle, round the front
				f.dot(g.T, 33, y, goldHD[60-y])
				f.dot(g.T, 34, y, goldHD[61-y])
			}
		}
		if hide != armHand && hide != armBoth {
			sl, hd := hdNearArm(sw)
			g.sideSleeve(sl)
			hand(g.T, hd, 0.1)
		}
		if g.hook != nil {
			g.hook()
		}
		if !female {
			m := sBox(14, 46.5, 39, 52, 3).or(sEll(26, 52, 8.5, 5.4)).jag(1.6, 3.8, 53, 21)
			f.fill(g.T, m, furShade(furHD, m, 5, 0, 31))
			g.brooch(33.5, 53, 2.8, true)
		} else {
			m := sBox(17, 46.5, 38, 52, 2.6).jag(1.3, 3.6, 49.5, 23)
			f.fill(g.T, m, furShade(furHD, m, 4, 0.02, 33))
			g.brooch(34.5, 54.5, 2.4, false)
		}
	}
}

// sideSleeve draws the near arm's sleeve of a figure seen from the side: on
// S over the body, to take the player's colour, with the belt cut away
// under it.
func (g *figGen) sideSleeve(sl shape) {
	f := &g.f
	female := g.look.Female
	for y := 0; y < g.T.c.h; y++ {
		for x := 0; x < g.T.c.w; x++ {
			if sl.d(float64(x)+0.5, float64(y)+0.5) <= 0 {
				g.T.c.px[y*g.T.c.w+x] = color.RGBA{}
				g.T.part[y*g.T.c.w+x] = 0
			}
		}
	}
	f.fill(g.S, sl, func(x, y int, _ float64) color.RGBA {
		i := int((sl.lum(x, y, 3) + 0.14) * 6)
		if !female && y >= 60 {
			i--
			if y == 60 {
				i += 2
			}
		}
		return shirtHD.at(max(i, 0))
	})
}

// belt draws a man's leather belt across columns x0 to x1; buckle adds the
// buckle in front and the end hanging from it.
func (g *figGen) belt(x0, x1 float64, buckle bool) {
	f := &g.f
	b := sBox(x0, 57, x1, 60, 0.4)
	f.fill(g.T, b, func(x, y int, _ float64) color.RGBA {
		i := 3 - int(float64(x)-x0)*3/int(x1-x0) // darker to the right
		switch y {
		case 57:
			i += 2
		case 59:
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
	end := sBox(30, 59, 33, 66, 1)
	f.fill(g.T, end, func(x, y int, _ float64) color.RGBA {
		if y >= 65 {
			return goldHD[2] // a metal tip
		}
		return leatherHD.at(3 - (x - 30))
	})
	ring := sBox(23.5, 56, 30.5, 61, 1.6)
	f.fill(g.T, ring, func(x, y int, d float64) color.RGBA {
		if d < -1.3 {
			return leatherHD[1] // the belt through it
		}
		return toneOf(goldHD, ring.lum(x, y, 1.5)+0.08)
	})
	f.dot(g.T, 27, 58, goldHD[4]) // the prong
	f.dot(g.T, 28, 58, goldHD[3])
	f.dot(g.T, 24, 56, goldHD[5])
}

// brooch is a gold brooch of radius r at (cx, cy): round with a garnet
// for a man's mantle, a plain oval for a woman's apron.
func (g *figGen) brooch(cx, cy, r float64, round bool) {
	f := &g.f
	s := sEll(cx, cy, r, r)
	if !round {
		s = sEll(cx, cy, r*0.8, r*1.25)
	}
	f.fill(g.T, s, func(x, y int, d float64) color.RGBA {
		if round && d < -r+1.2 {
			return garnet[1]
		}
		return toneOf(goldHD, s.lum(x, y, r*0.9)+0.1)
	})
	if round {
		f.dot(g.T, int(cx-0.5), int(cy-0.5), garnet[2])
	} else {
		f.dot(g.T, int(cx-0.5), int(cy-1.5), goldHD[5])
	}
}
