package client

import (
	"math"
)

// The village's carved post at HD density, painted with the critters' lit
// painter (critpaint.go), seen from the front: a trunk of grey-brown wood,
// its grain running up it, carved into three figures stacked one on another
// and crowned with a thunderbird, each carving standing out of the post so
// it lights and casts its shadow, painted in red ochre, white clay,
// charcoal, a copper green and yellow. From the top: the thunderbird, its
// wings spread wider than the post and barred, its hooked beak down; an
// owl's face with ringed eyes; a man holding his belly, round-eyed and
// showing his teeth; a bear with round ears and its tongue out; a band of
// chevrons at the foot, among stones. About 2.3 players tall.

var (
	totemWood   = newSurf(mkRamp(0x2a1a12, 0x46301e, 0x664628, 0x886038, 0xa87c4c), sFur, 0.05, 950).with(10, math.Pi/2)
	totemRed    = newSurf(mkRamp(0x4a0c08, 0x861c10, 0xbc301c, 0xdc5832, 0xf08a5a), sMatte, 0.1, 951)
	totemWhite  = newSurf(mkRamp(0x7e786e, 0xb0a898, 0xd8d0c0, 0xf4f0e4), sMatte, 0.1, 952)
	totemBlack  = newSurf(mkRamp(0x0c0808, 0x1a1414, 0x2c2424, 0x42383a), sMatte, 0.15, 953)
	totemGreen  = newSurf(mkRamp(0x0c3434, 0x18524e, 0x2a7870, 0x48a090, 0x7ac6b0), sMatte, 0.1, 954)
	totemYellow = newSurf(mkRamp(0x5e400e, 0x926616, 0xc28e26, 0xe4b846, 0xf4d880), sMatte, 0.1, 955)
	totemStone  = newSurf(stoneRamp, sShell, 0.3, 956)
)

// totemCanvas draws the post; it stands at (13, 72) art pixels.
func totemCanvas() (*canvas, float64, float64) {
	q := newCritAt(26, 75, 0, 1)
	const cx, gy = 13.0, 72.0
	relief := func(s shape, round, bias float64) { q.body(s, totemWood, look{round: round, bias: bias}) }
	paint := func(s shape, sf *surf) { q.coat(s, sf) }
	pair := func(dx float64, f func(x, side float64)) { f(cx-dx, -1); f(cx+dx, 1) }

	// The stones round its foot, behind and either side.
	for i, s := range [][3]float64{{-5.4, 70.6, 1.9}, {5.2, 70.8, 2.1}, {-2.6, 71.4, 1.5}} {
		q.body(aEll(cx+s[0], s[1], s[2], s[2]*0.75), totemStone, look{round: s[2] * 0.8, bias: -0.04 * float64(i)})
	}
	// The trunk, flaring at its foot.
	post := sBlend(2, aCap(cx, 12, cx, gy-1.5, 4.7, 5), aEll(cx, gy-1.6, 5.8, 1.6))
	relief(post, 4.2, 0)
	for i, s := range [][3]float64{{3.4, 71.6, 1.6}} {
		q.body(aEll(cx+s[0], s[1], s[2], s[2]*0.75), totemStone, look{round: s[2] * 0.8, bias: 0.04 * float64(i)})
	}

	// The foot: a band of chevrons, red on white.
	paint(post.and(aCap(cx-6, 63.2, cx+6, 63.2, 1.6, 1.6)), totemWhite)
	for k := -2.0; k <= 2; k++ {
		x := cx + k*2.2
		paint(aPoly(x-1, 62, x, 63.6, x+1, 62, x+1, 63, x, 64.6, x-1, 63), totemRed)
	}
	relief(post.and(aCap(cx-6, 61.2, cx+6, 61.2, 0.4, 0.4)), 0.4, 0.08) // a ridge carved above it
	relief(post.and(aCap(cx-6, 65.2, cx+6, 65.2, 0.4, 0.4)), 0.4, 0.04)

	// The bear: round ears, a broad snout, the mouth open on its teeth and
	// its tongue out.
	pair(3.8, func(x, side float64) {
		relief(aEll(x, 47.6, 1.7, 1.6), 1.2, 0.04)
		paint(aEll(x, 47.8, 0.8, 0.8), totemRed)
	})
	bear := aEll(cx, 53.2, 5.2, 5.4)
	relief(bear, 2.6, 0.02)
	paint(bear.and(aCap(cx-5, 50, cx+5, 50, 1.1, 1.1)), totemBlack) // a dark band across the eyes
	pair(2.1, func(x, side float64) {
		relief(aEll(x, 50, 1.2, 0.9), 0.8, 0.06)
		paint(aEll(x, 50, 0.9, 0.65), totemWhite)
		paint(aEll(x+side*0.1, 50.1, 0.42, 0.42), totemBlack)
	})
	snout := sBlend(0.8, aEll(cx, 53.6, 2.4, 1.9), aEll(cx, 52, 1.1, 1.4))
	relief(snout, 1.4, 0.08)
	paint(aEll(cx, 52.9, 1, 0.6), totemBlack) // the nose
	mouth := aEll(cx, 57.2, 3.2, 1.7)
	relief(mouth, 0.8, -0.3)
	paint(mouth, totemBlack)
	paint(aEll(cx, 58.2, 1.6, 1), totemRed) // the tongue
	for k := -1.5; k <= 1.5; k++ {
		paint(aPoly(cx+k*1.3-0.5, 55.7, cx+k*1.3+0.5, 55.7, cx+k*1.3, 56.9), totemWhite)
	}

	// The man: round eyes under a heavy brow, a flat nose, teeth; his hands
	// on his belly, his arms carved round it.
	man := aEll(cx, 36.4, 4.9, 5.6)
	relief(man, 2.4, 0.04)
	paint(man.and(aCap(cx-5, 33.2, cx+5, 33.2, 1.4, 1.4)), totemGreen)
	relief(aCap(cx-3.6, 32.6, cx+3.6, 32.6, 0.7, 0.7), 0.5, 0.1) // the brow
	pair(2, func(x, side float64) {
		relief(aEll(x, 34.4, 1.4, 1.3), 0.9, 0.06)
		paint(aEll(x, 34.4, 1.1, 1), totemWhite)
		paint(aEll(x, 34.5, 0.55, 0.55), totemBlack)
	})
	relief(aPoly(cx-0.7, 34, cx+0.7, 34, cx+1.4, 37.4, cx-1.4, 37.4), 0.6, 0.08)
	mm := aCap(cx-2.4, 39.4, cx+2.4, 39.4, 0.85, 0.85)
	relief(mm, 0.5, -0.24)
	paint(mm, totemBlack)
	for k := -1.5; k <= 1.5; k++ {
		paint(aEll(cx+k*1.15, 39.4, 0.4, 0.5), totemWhite)
	}
	paint(aEll(cx-3.4, 37.6, 0.9, 0.6), totemRed) // cheeks
	paint(aEll(cx+3.4, 37.6, 0.9, 0.6), totemRed)
	belly := aEll(cx, 44.6, 4.3, 2.4)
	relief(belly, 1.6, 0)
	paint(belly.and(aCap(cx-5, 45.4, cx+5, 45.4, 0.5, 0.5)), totemYellow)
	pair(4.4, func(x, side float64) {
		arm := aPath(x, 40.4, 1.1, x+side*0.2, 43.4, 1, cx+side*1.2, 44.6, 0.9)
		relief(arm, 0.8, 0.06)
		paint(aEll(cx+side*1.4, 44.6, 1.1, 0.9), totemRed) // the hands
		for k := 0.0; k < 3; k++ {
			paint(aCap(cx+side*(0.7+0.5*k), 44, cx+side*(0.7+0.5*k), 45.2, 0.14, 0.14), totemBlack)
		}
	})

	// The owl: ringed eyes, a hooked beak, feathered brows.
	owl := aEll(cx, 22.4, 5, 5.2)
	relief(owl, 2.4, 0.04)
	pair(2.2, func(x, side float64) {
		ring := aEll(x, 21.6, 2, 2)
		relief(ring, 1.2, 0.06)
		paint(ring, totemYellow)
		paint(aEll(x, 21.6, 1.35, 1.35), totemWhite)
		paint(aEll(x+side*0.15, 21.7, 0.7, 0.7), totemBlack)
		relief(aPath(x-side*1.8, 19, 0.5, x, 18.6, 0.6, x+side*2.2, 19.6, 0.4), 0.4, 0.1) // a brow of feathers
		paint(aPath(x-side*1.8, 19, 0.4, x, 18.6, 0.5, x+side*2.2, 19.6, 0.3), totemRed)
	})
	beak := aPoly(cx-1.1, 22.2, cx+1.1, 22.2, cx+0.5, 25.6, cx, 26.4, cx-0.5, 25.6)
	relief(beak, 0.6, 0.1)
	paint(beak, totemYellow)
	for k := -1.0; k <= 1; k++ { // feathers on its breast
		paint(aPoly(cx+k*1.8-0.8, 25.6, cx+k*1.8, 27.2, cx+k*1.8+0.8, 25.6), totemGreen)
	}

	// The thunderbird on top: its wings spread and barred, its head
	// between them, the beak hooked down.
	for _, side := range [2]float64{-1, 1} {
		w := aPoly(cx+side*2.6, 9.8, cx+side*12.6, 6.6, cx+side*12.8, 8.2, cx+side*11.4, 10, cx+side*12, 11, cx+side*10, 12.2, cx+side*10.4, 13.2, cx+side*7.6, 13.8, cx+side*3.4, 14.4)
		relief(w, 1, 0.04)
		for k := 0.0; k < 4; k++ { // the bars of its feathers
			x := cx + side*(4.6+2*k)
			paint(w.and(aCap(x, 9, x+side*0.6, 14.5, 0.45, 0.45)), []*surf{totemBlack, totemRed, totemBlack, totemWhite}[int(k)])
		}
		paint(w.and(aCap(cx+side*3, 9.8, cx+side*12.6, 6.8, 0.6, 0.4)), totemWhite) // the leading edge
	}
	head := sBlend(1, aEll(cx, 7.6, 2.8, 2.9), aEll(cx, 12, 3.2, 2.6))
	relief(head, 2, 0.06)
	paint(head.and(aEll(cx, 12.6, 3.2, 1.6)), totemRed)
	pair(1.3, func(x, side float64) {
		paint(aEll(x, 7, 0.9, 0.8), totemWhite)
		paint(aEll(x+side*0.1, 7.1, 0.42, 0.42), totemBlack)
	})
	bk := aPath(cx, 8.4, 0.95, cx, 10.4, 0.7, cx-0.3, 11.6, 0.3)
	relief(bk, 0.6, 0.12)
	paint(bk, totemYellow)
	paint(aEll(cx-0.3, 11.4, 0.4, 0.4), totemBlack) // the tip of the hook
	return q.done(0.8, cx, gy)
}

// genTotemHD is the village post as a sprite.
func genTotemHD() sprite {
	c, ax, ay := totemCanvas()
	return mkUp(c, ax*HDDensity, ay*HDDensity)
}
