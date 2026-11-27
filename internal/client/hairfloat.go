package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// Hair that lies loose: spread over the pillow round a sleeper's head, and
// floating on the water behind a swimmer's, trailing and waving with the
// stroke. Each is drawn as locks from where the hair leaves the head, as
// many and as long as the style has (a long mane, a shaggy one ragged and
// thick, a bob with its thin braid, a ponytail in one bundle, short hair and
// spikes a few short locks, a bun none, it is tied up), in the hair's own
// colour, each lock lit along its length and parted from the next by a
// fine dark line, like the hair on the head (tallhead.go). Drawn the first
// time it is shown and kept, a few phases of the wave.

// hairLie is where loose hair lies.
type hairLie uint8

const (
	hairPillow  hairLie = iota // round the crown, the head on its back on the pillow
	hairPillowS                // the head on its side: behind it
	hairFloatS                 // floating behind a swimmer seen from the side (facing right)
	hairFloatF                 // behind one swimming towards the viewer: up the screen
	hairFloatB                 // behind one swimming away: down the screen, towards the viewer
)

type hairKey struct {
	style  sim.HairStyle
	color  uint8
	female bool
	lie    hairLie
	phase  uint8 // of the wave, eighths
}

var hairCache = map[hairKey]sprite{}

// hairLocks is how loose hair of style s lies: how many locks, how long
// (image pixels), how thick at the root and the tip, how unruly; and
// whether one of them is a braid.
func hairLocks(s sim.HairStyle, female bool) (n int, length, w0, w1, wild float64, braid bool) {
	switch s {
	case sim.HairLong:
		return 9, 38, 2.3, 1.1, 0.25, false
	case sim.HairShaggy:
		return 11, 30, 2.8, 1.4, 0.6, false
	case sim.HairBob:
		return 7, 18, 2.1, 1.2, 0.2, true
	case sim.HairPonytail:
		return 4, 28, 2.2, 1.0, 0.15, false
	case sim.HairShort:
		return 5, 8, 1.8, 0.9, 0.3, false
	case sim.HairSpiky:
		return 5, 7, 1.7, 0.4, 0.5, false
	}
	return 0, 0, 0, 0, 0, false // a bun stays tied
}

// hairSprite draws loose hair k, anchored where it leaves the head.
func hairSprite(k hairKey) sprite {
	if s, ok := hairCache[k]; ok {
		return s
	}
	n, length, w0, w1, wild, braid := hairLocks(k.style, k.female)
	const cw, ch = 34, 26 // art pixels
	ox, oy := float64(cw)/2*HDDensity, float64(ch)/2*HDDensity
	l := newFigLayer(cw*HDDensity, ch*HDDensity)
	var f figHD
	f.ox, f.oy = ox, oy
	hr := ext6(hairColors[k.color%sim.NumHairColors].ramp)
	seed := uint32(k.style)*31 + uint32(k.color)
	ph := float64(k.phase) / 8 * 2 * math.Pi
	for i := 0; i < n; i++ {
		u := (float64(i)+0.5)/float64(n)*2 - 1 // across the spread, -1 to 1
		jit := (hashf(i, 1, seed) - 0.5) * wild
		ln := length * (0.75 + 0.35*hashf(i, 2, seed))
		if k.style == sim.HairPonytail {
			u *= 0.25
		}
		var pts []float64
		const steps = 7
		for s := 0; s <= steps; s++ {
			t := float64(s) / steps
			r := ln * t
			var x, y float64
			switch k.lie {
			case hairPillow, hairPillowS: // spread out over the pillow from under the head, lying flat
				ang := math.Pi + u*0.6 + jit*0.5
				if k.lie == hairPillowS {
					ang = math.Pi*1.1 + u*0.5 + jit*0.5
				}
				r *= 0.6
				curl := 0.5 * t * t * (u + jit) // the tips curl away
				x, y = math.Cos(ang+curl)*r, math.Sin(ang+curl)*r*0.55+u*3
			case hairFloatS: // trailing behind on the surface, seen at a slant
				wave := math.Sin(ph-t*4.2+float64(i)*0.7) * 2.2 * t
				x = -r
				y = (u*0.55*t*ln*0.35 + wave) * 0.5
			case hairFloatF, hairFloatB: // up the screen, or down it
				wave := math.Sin(ph-t*4.2+float64(i)*0.7) * 2.2 * t
				x = u*0.75*t*ln + wave
				y = -r * 0.24 // lying flat on the water, seen at a slant
				if k.lie == hairFloatB {
					y = r * 0.4
				}
			}
			pts = append(pts, x, y)
		}
		lw0, lw1 := w0, w1
		if k.lie == hairPillow || k.lie == hairPillowS { // lying in a mass
			lw0, lw1 = w0*1.35, w1*1.5
		}
		lock := sPath(lw0, lw1, pts...)
		bias := 0.1*u + 0.12*(hashf(i, 3, seed)-0.5)
		isBraid := braid && i == 0
		f.fill(l, lock, func(x, y int, d float64) color.RGBA {
			v := lock.lum(x, y, 1.6) + bias
			if isBraid && (x+y)%3 == 0 {
				v -= 0.25 // plaited
			}
			return toneOf(hr, v*0.85)
		})
		if isBraid { // its tie at the end
			ex, ey := pts[len(pts)-2], pts[len(pts)-1]
			f.fill(l, sEll(ex, ey, 1.3, 1.3), plain(hairTie))
		}
	}
	innerLines(0.45, l)
	l.c.outline(0.7)
	s := cropSprite(l.c, ox/HDDensity, oy/HDDensity)
	hairCache[k] = s
	return s
}

// drawPillowHair spreads look l's loose hair over the pillow round the head
// of a sleeper in pose (see sleepPose) whose head is at (x, y): under the
// head. Turning over it is gathered up.
func (g *Game) drawPillowHair(dst drawTarget, l sim.Look, pose uint8, turning bool, x, y float64) {
	l = l.Sanitized()
	if turning {
		return
	}
	k := hairKey{style: l.Hair, color: l.HairColor, female: l.Female, lie: hairPillow}
	cx := x - 2.5 // the crown, left of the middle of the head
	if pose != 0 && pose != 4 {
		k.lie = hairPillowS
		cx, y = x-1.5, y-1
	}
	putSprite(dst, hairSprite(k), math.Round(cx), math.Round(y))
}

// drawFloatHair draws look l's hair floating behind the head of a swimmer
// facing dir, where it leaves the head at (x, y) on the surface, waving at
// phase (0 to 1) of the stroke.
func (g *Game) drawFloatHair(dst drawTarget, l sim.Look, dir sim.Dir, x, y, phase float64) {
	l = l.Sanitized()
	k := hairKey{style: l.Hair, color: l.HairColor, female: l.Female, phase: uint8(int(phase*8) % 8)}
	sx := 1.0
	switch dir {
	case sim.DirLeft:
		k.lie, sx = hairFloatS, -1
	case sim.DirRight:
		k.lie = hairFloatS
	case sim.DirDown:
		k.lie = hairFloatF
	default:
		k.lie = hairFloatB
	}
	s := hairSprite(k)
	if s.img == nil {
		return
	}
	if sx < 0 {
		s = mirrorSprite(s)
	}
	putSprite(dst, s, math.Round(x), math.Round(y))
}
