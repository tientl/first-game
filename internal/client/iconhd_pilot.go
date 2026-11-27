package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The first HD item icons: the twenty seen most. Each painter draws in the
// 16 x 16 design box (iconpaint.go), lit from the top left; tools lie from
// the bottom left to the top right, their working end up; food is turned a
// little. Colours are the old icons' ramps (itemicons.go, cookicons.go) and,
// for stone, the COMFY52 tones of the boulders (rockart.go).

// iconPilot is the order the icon sheet shows them in (shot -iconsheet).
var iconPilot = []sim.ItemKind{
	sim.ItemAxe, sim.ItemPickaxe, sim.ItemSpear, sim.ItemCopperSword,
	sim.ItemWood, sim.ItemStick, sim.ItemStone, sim.ItemCopperOre,
	sim.ItemGrass, sim.ItemRope, sim.ItemBerry, sim.ItemMeat,
	sim.ItemCookedMeat, sim.ItemFish, sim.ItemBanana, sim.ItemTorch,
	sim.ItemClay, sim.ItemHide, sim.ItemBone, sim.ItemStew,
}

// The axe, the pickaxe, the spear, the flint knife and the torch are their
// held painters (iconhd_tools.go), so the bag and the hand match.
func init() {
	regIcon(sim.ItemWood, icWood)
	regIcon(sim.ItemStick, icStick)
	regIcon(sim.ItemStone, icStone)
	regIcon(sim.ItemCopperOre, icFlint)
	regIcon(sim.ItemGrass, icGrass)
	regIcon(sim.ItemRope, icRope)
	regIcon(sim.ItemBerry, icBerry)
	regIcon(sim.ItemMeat, icMeat)
	regIcon(sim.ItemCookedMeat, icRoast)
	regIcon(sim.ItemFish, icFish)
	regIcon(sim.ItemBanana, icBanana)
	regIcon(sim.ItemClay, icClay)
	regIcon(sim.ItemHide, icHide)
	regIcon(sim.ItemBone, icBone)
	regIcon(sim.ItemStew, icStew)
}

// --- shapes and frames ----------------------------------------------------------

// iEll is an ellipse with semi-axes ra along angle ang and rb across it.
func iEll(cx, cy, ra, rb, ang float64) shape {
	sn, cs := math.Sin(ang), math.Cos(ang)
	m := math.Min(ra, rb)
	r := math.Max(ra, rb)
	return shape{cx - r, cy - r, cx + r, cy + r, func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		u, v := (dx*cs+dy*sn)/ra, (-dx*sn+dy*cs)/rb
		return (math.Sqrt(u*u+v*v) - 1) * m
	}}
}

// iBlend joins shapes into one lump, the seams filled in over k units.
func iBlend(k float64, ss ...shape) shape {
	out := ss[0]
	for _, b := range ss[1:] {
		a, b := out, b
		out = shape{math.Min(a.x0, b.x0) - k/4, math.Min(a.y0, b.y0) - k/4, math.Max(a.x1, b.x1) + k/4, math.Max(a.y1, b.y1) + k/4,
			func(x, y float64) float64 {
				da, db := a.d(x, y), b.d(x, y)
				h := math.Max(k-math.Abs(da-db), 0) / k
				return math.Min(da, db) - h*h*k/4
			}}
	}
	return out
}

// rough frays a's edge by up to amp units, in lumps size units across.
func (a shape) rough(amp, size float64, seed uint32) shape {
	return shape{a.x0 - amp, a.y0 - amp, a.x1 + amp, a.y1 + amp, func(x, y float64) float64 {
		return a.d(x, y) + amp*(fbm2(x/size, y/size, seed, 2)-0.5)*2
	}}
}

// frame2 places points by how far along (s) and across (t) a direction
// they are from an origin: a tool's head is drawn in its haft's frame.
type frame2 struct{ ox, oy, ex, ey, px, py float64 }

// frameOf is the frame at (ox, oy) along (dx, dy), t growing to its left
// (seen going along it).
func frameOf(ox, oy, dx, dy float64) frame2 {
	l := math.Hypot(dx, dy)
	dx, dy = dx/l, dy/l
	return frame2{ox, oy, dx, dy, dy, -dx}
}

func (f frame2) at(s, t float64) (float64, float64) {
	return f.ox + f.ex*s + f.px*t, f.oy + f.ey*s + f.py*t
}

// st is where (x, y) is in the frame.
func (f frame2) st(x, y float64) (s, t float64) {
	dx, dy := x-f.ox, y-f.oy
	return dx*f.ex + dy*f.ey, dx*f.px + dy*f.py
}

// poly is the polygon through the frame's points (s, t pairs).
func (f frame2) poly(st ...float64) shape {
	pts := make([]float64, len(st))
	for i := 0; i+1 < len(st); i += 2 {
		pts[i], pts[i+1] = f.at(st[i], st[i+1])
	}
	return sPoly(pts...)
}

// path is points (s, t pairs) of the frame as x, y pairs.
func (f frame2) path(st ...float64) []float64 {
	pts := make([]float64, len(st))
	for i := 0; i+1 < len(st); i += 2 {
		pts[i], pts[i+1] = f.at(st[i], st[i+1])
	}
	return pts
}

// sTaper is a brush along pts, w0 thick at the start, wm in the middle
// and w1 at the end: a blade of grass, a banana.
func sTaper(w0, wm, w1 float64, pts ...float64) shape {
	n := len(pts) / 2
	h := n / 2
	a := make([]float64, 0, 2*(h+1))
	for i := h; i >= 0; i-- {
		a = append(a, pts[2*i], pts[2*i+1])
	}
	return sPath(wm, w0, a...).or(sPath(wm, w1, pts[2*h:]...))
}

// sRing is a ring of rope thick w along the ellipse rx by ry about
// (cx, cy).
func sRing(cx, cy, rx, ry, w float64) shape {
	return shape{cx - rx - w, cy - ry - w, cx + rx + w, cy + ry + w, func(x, y float64) float64 {
		// The distance to the ellipse, to first order: its equation over
		// the length of its gradient.
		dx, dy := x-cx, y-cy
		f := dx*dx/(rx*rx) + dy*dy/(ry*ry) - 1
		g := 2 * math.Hypot(dx/(rx*rx), dy/(ry*ry))
		if g < 1e-9 {
			return math.Min(rx, ry) - w
		}
		return math.Abs(f/g) - w
	}}
}

// --- materials --------------------------------------------------------------------

var (
	icStoneR  = rkGraniteMat.tone                                        // COMFY52, as the boulders in the world
	icFlintR  = mkRamp(0x1a1f2e, 0x2b222a, 0x3f333b, 0x533b41, 0x7d595d) // rkFlintMat's, a step deeper
	icWFlintR = mkRamp(0x2b222a, 0x533b41, 0x6d4e4b, 0x867066, 0xb49d7e) // worked flint: a blade, a point
	icCortex  = mkRamp(0x6d4e4b, 0x867066, 0xb49d7e, 0xc4c6b8, 0xe5efef) // flint's chalky rind
	icCordR   = mkRamp(0x533b41, 0x825646, 0xb77862, 0xb49d7e, 0xeecb90) // rawhide lashing
	icBarkR   = mkRamp(0x2b222a, 0x533b41, 0x6d4e4b, 0x825646, 0xb77862) // a branch's bark
	icStalk   = mkRamp(0x3f4328, 0x5f7132, 0x94ad39, 0xc2d64f)           // green stalks
	icFatR    = mkRamp(0xd8a49a, 0xf0c8bc, 0xfbe4d8, 0xfff6ee)           // raw fat
	icBananaR = mkRamp(0x8d4830, 0xc57835, 0xe1a845, 0xf2d060, 0xfff0a0) // ripe yellow
	icPeel    = mkRamp(0x3c252b, 0x684039, 0x825646)                     // its brown ends
	icCarrot  = mkRamp(0x8d3649, 0xc33c40, 0xe47259, 0xeca070)           // a root in the stew
	icHideIn  = mkRamp(0x684039, 0x825646, 0xb77862, 0xd8a074, 0xeecb90) // the flesh side of a hide
	icPelt    = mkRamp(0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b, 0x867066) // its fur
	icRoastR  = mkRamp(0x3c1a14, 0x6a2c1a, 0x9a4a24, 0xc87436, 0xeeac5c) // a roast's crust
	icRawR    = mkRamp(0x5c2b34, 0x8d3649, 0xc33c40, 0xe47259, 0xedaaa3) // raw red meat, COMFY52
	icFishR   = mkRamp(0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5) // a fish's back, COMFY52
	icBellyR  = mkRamp(0x8d99a4, 0xbac9cd, 0xe5efef, 0xffffff)           // its belly
	icBoneR   = mkRamp(0x7d595d, 0xa38c78, 0xd0c4a8, 0xece4cc, 0xfffaea) // bone
	icClayR   = mkRamp(0x3c2a26, 0x64483a, 0x8e6a52, 0xb48c6c, 0xd6b090) // wet clay, ciClay a step down
	icBowlR   = ciPot                                                    // fired clay
	icBrothR  = ciBroth                                                  // the stew
	icGreenR  = mkRamp(0x2a5c2a, 0x4a8a34, 0x7cb648, 0xb8dc70)           // greens in it
	icStrawR  = mkRamp(0x5a4a1c, 0x8c7a2c, 0xbca844, 0xdccc6c, 0xf2e8a0) // iiStraw
	icTwineR  = mkRamp(0x3c252b, 0x684039, 0x8d4830, 0xc57835, 0xe1a845) // a band of twisted straw
	icRopeR   = mkRamp(0x533b41, 0x7d595d, 0xa38c78, 0xc8b48c, 0xe8d8a8) // flax rope
	icClothR  = mkRamp(0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b, 0x867066) // a torch's tarred wrap
	icGlint   = color.RGBA{0xff, 0xfd, 0xf0, 0xff}
)

// shKnap shades knapped stone: shallow scallops, each a dish whose far rim
// catches the light, over the tone the face has.
func shKnap(r ramp, base, scale float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA { return pick(r, knap(f.x, f.y, base, scale, seed)) }
}

func knap(x, y, base, scale float64, seed uint32) float64 {
	gx, gy := x/scale, y/scale
	ix, iy := math.Floor(gx), math.Floor(gy)
	best, second := 9.0, 9.0
	var bx, by float64
	for dy := -1.0; dy <= 1; dy++ {
		for dx := -1.0; dx <= 1; dx++ {
			cx := ix + dx + 0.15 + 0.7*hashf(int(ix+dx), int(iy+dy), seed)
			cy := iy + dy + 0.15 + 0.7*hashf(int(ix+dx), int(iy+dy), seed+7)
			d := math.Hypot(gx-cx, gy-cy)
			if d < best {
				second, best, bx, by = best, d, cx, cy
			} else if d < second {
				second = d
			}
		}
	}
	// Inside a dish the surface leans in towards its centre.
	nx, ny := (gx-bx)*0.6, (gy-by)*0.6
	l := base + (litOf(-nx, -ny)-0.62)*0.9
	if second-best < 0.1 {
		l += 0.16 // the ridge between two scars
	}
	return l
}

// shSphere shades a round thing about (cx, cy), rx by ry, with a light
// bounced into its far side and, if glint, a sharp highlight.
func shSphere(r ramp, cx, cy, rx, ry, bias float64, glint bool) ishader {
	return func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := litOf(nx*0.92, ny*0.92) + bias
		if nx*0.45+ny*0.89 > 0.7 && nx*nx+ny*ny > 0.62 {
			l += 0.2 // bounced light
		}
		if glint {
			if gx, gy := nx+0.38, ny+0.44; gx*gx*1.5+gy*gy < 0.045 {
				return icGlint
			}
		}
		return pick(r, l)
	}
}

// shCord shades a lashing: round across its band, twisted strands along it.
func shCord(r ramp, a iaxis, rad, twist float64) ishader {
	return func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, rad)
		if math.Mod(u*twist+v*1.4+100, 1) < 0.3 {
			l -= 0.24
		}
		return pick(r, l)
	}
}

// binding wraps a haft running along a with rawhide at (x, y): a band len
// units along it, hw to each side, turns of cord across it.
func binding(p *icv, a iaxis, x, y, length, hw float64) {
	fr := frameOf(x, y, a.ex, a.ey)
	band := fr.poly(-length/2, -hw, length/2, -hw, length/2+0.15, 0, length/2, hw, -length/2, hw, -length/2-0.15, 0)
	p.fill(band, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := a.rod(-t, hw*1.05)
		// The turns: across the haft, a little slanted.
		if math.Mod(s*2.3+t*0.55+50, 1) < 0.28 {
			l -= 0.3
		}
		return pick(icCordR, l)
	})
}

// --- materials of the land ---------------------------------------------------------

// icWood is two cut logs, one on the other, their ends towards the viewer.
func icWood(p *icv) {
	log1 := func(ax, ay, bx, by, r float64, seed uint32) {
		a := mkAxis(ax, ay, bx, by)
		p.fill(sCap(ax, ay, bx, by, r, r), shRod(iiWood, a, r, -0.02, 1, seed))
		ang := math.Atan2(a.ey, a.ex)
		p.fill(iEll(ax, ay, r*0.52, r*0.98, ang), shRing(iiPale, ax, ay, r*0.52, r, 3, seed))
	}
	log1(3.2, 11.9, 12.8, 9.3, 2.8, 3)
	log1(4.2, 6.5, 12.9, 3.6, 2.6, 9)
}

// icStick is a forked branch with two leaves at its tip.
func icStick(p *icv) {
	a := mkAxis(2.2, 14.6, 13.4, 3.0)
	p.fill(sPath(0.88, 0.62, 2.2, 14.6, 7.4, 9.6, 13.4, 3.0), shRod(icBarkR, a, 0.8, 0.12, 1, 51))
	// The fork, out of the upper side.
	b := mkAxis(7.6, 9.4, 10.8, 9.2)
	p.under(sPath(0.55, 0.36, 7.4, 9.6, 9.4, 9.0, 11.2, 9.4), shRod(icBarkR, b, 0.5, 0.12, 1, 52))
	// The cut end, pale wood, at the bottom.
	ang := math.Atan2(a.ey, a.ex)
	p.fill(iEll(2.2, 14.6, 0.48, 0.9, ang), shRing(iiPale, 2.2, 14.6, 0.48, 0.9, 2, 3))
	leaf := func(x, y, l, w, ang float64) {
		cx, cy := x+math.Cos(ang)*l/2, y+math.Sin(ang)*l/2
		p.fill(iEll(cx, cy, l/2, w, ang), func(f *ifrag) color.RGBA {
			rib := mkAxis(x, y, x+math.Cos(ang)*l, y+math.Sin(ang)*l)
			_, v := rib.uv(f.x, f.y)
			lv := bulge(f, w*0.9)
			if math.Abs(v) < 0.12 {
				lv -= 0.22
			}
			if v > 0 {
				lv -= 0.1
			}
			return pick(iiLeaf, lv)
		})
	}
	leaf(13.2, 3.2, 2.8, 0.95, -2.5)
	leaf(13.0, 3.4, 2.6, 0.9, 0.35)
	leaf(11.0, 9.3, 2.3, 0.8, -0.7)
}

// icStone is a lump of grey stone, its faces cut flat like the boulders'.
func icStone(p *icv) {
	// The faces share their corners: top, left, front, right.
	top := sPoly(2.6, 6.8, 5.2, 3.6, 9.8, 2.7, 13.4, 4.6, 11.8, 7.2, 6.2, 8.0)
	left := sPoly(2.6, 6.8, 6.2, 8.0, 6.6, 13.4, 3.6, 12.6, 1.6, 9.8)
	front := sPoly(6.2, 8.0, 11.8, 7.2, 12.6, 12.8, 6.6, 13.4)
	right := sPoly(11.8, 7.2, 13.4, 4.6, 14.8, 8.6, 14.2, 11.6, 12.6, 12.8)
	p.fill(top, shFacet(icStoneR, 0.86, 0.14, 1))
	p.add(left, shFacet(icStoneR, 0.56, 0.14, 2))
	p.add(front, shFacet(icStoneR, 0.43, 0.14, 3))
	p.add(right, shFacet(icStoneR, 0.22, 0.12, 4))
	// A crack down the front, lit along its lower lip, and a chip off the
	// top's edge.
	p.on(sPath(0.17, 0.1, 9.2, 7.6, 8.7, 9.6, 9.4, 11.0, 9.0, 12.9), solidC(icStoneR[0]))
	p.on(sPath(0.1, 0.06, 9.45, 7.9, 8.95, 9.7, 9.65, 11.0), solidC(icStoneR[3]))
	p.on(sEll(4.6, 9.6, 0.5, 0.35), solidC(icStoneR[2]))
}

// icFlint is a nodule of flint broken open: a chalky rind round a dark
// glassy core, the ripples of the blow faint on it.
func icFlint(p *icv) {
	body := iBlend(1.6, sEll(7.4, 8.6, 5.8, 4.8), sEll(11.2, 9.8, 3.6, 3.2), sEll(4.8, 10.6, 3.6, 3.0)).rough(0.35, 1.1, 71)
	p.fill(body, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.6) - 0.04 + (fbm2(f.x*2.2, f.y*2.2, 72, 2)-0.5)*0.28
		return pick(icCortex, l)
	})
	// The broken face: a glassy dark bulge in the top of it, a pale band
	// of broken rind round it.
	face := sPoly(3.0, 8.6, 5.2, 4.8, 9.6, 4.0, 12.8, 6.0, 13.0, 9.2, 9.6, 11.4, 5.2, 11.6)
	p.add(face.grow(0.42).rough(0.22, 0.5, 74), func(f *ifrag) color.RGBA { return pick(icCortex, 0.72+0.12*toward(f.gx, f.gy)) })
	ix, iy := 3.2, 8.4 // where the blow landed
	p.fill(face, func(f *ifrag) color.RGBA {
		// Lighter towards the top left, as glass that bulges.
		l := 0.42 - (f.x-8)*0.035 - (f.y-7.6)*0.05
		// Ripples spread from the blow, farther apart as they go.
		d := math.Hypot(f.x-ix, f.y-iy)
		if ph := math.Mod(math.Sqrt(d)*2.6+0.12*vnoise(f.x*1.5, f.y*1.5, 73), 1); ph < 0.1 {
			l += 0.14
		}
		if -f.d < 0.4 && toward(f.gx, f.gy) > 0 {
			l += 0.3
		}
		return pick(icFlintR, l)
	})
	// Its glassy gleam.
	p.on(sTaper(0.1, 0.3, 0.08, 6.4, 6.0, 8.2, 5.2, 10.4, 5.6), solidC(icGlint))
	p.on(sEll(5.6, 7.6, 0.38, 0.24), solidC(icGlint))
}

// icGrass is a sheaf of dry grass tied in the middle.
func icGrass(p *icv) {
	type blade struct{ x0, x1, bend float64 }
	bl := []blade{{4.6, 1.8, -0.6}, {5.4, 3.6, -0.3}, {6.2, 5.6, 0.2}, {7.0, 7.4, -0.2}, {7.8, 9.4, 0.3},
		{8.6, 11.2, 0.2}, {9.4, 13.0, 0.5}, {10.2, 14.4, 0.6}}
	for i, b := range bl {
		seed := uint32(80 + i)
		// From the bottom through the tie at y 9.6 to its head at the top.
		bx := 8.0 + (b.x0-8.0)*0.42
		x2 := b.x1 + b.bend
		pts := []float64{bx + (b.x0-8)*0.35, 15.0, bx, 9.6, (bx + x2) / 2, 5.6, x2, 1.4 + 0.5*hashf(i, 0, 81)}
		a := mkAxis(pts[2], pts[3], pts[6], pts[7])
		p.fill(sTaper(0.6, 0.62, 0.12, pts...), func(f *ifrag) color.RGBA {
			_, v := a.uv(f.x, f.y)
			l := 0.66 - v*0.35 + float64(i%3)*0.06 + (fbm2(f.x*0.6, f.y*3, seed, 2)-0.5)*0.3
			return pick(icStrawR, l)
		})
	}
	// Seed heads on a few blades.
	for i, b := range []blade{bl[1], bl[3], bl[5], bl[7]} {
		x := b.x1 + b.bend
		y := 1.8 + 0.5*hashf(i, 0, 81)
		p.fill(iEll(x, y+0.6, 0.6, 1.25, -0.2+float64(i)*0.15), shBulge(icStrawR, 0.6, 0.05))
	}
	// The tie: a band of twisted straw round the waist.
	tie := sBox(4.9, 8.8, 11.0, 10.6, 0.7)
	a := mkAxis(4.9, 9.7, 11.0, 9.7)
	p.fill(tie, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, 0.95)
		if math.Mod(u*1.6+v*0.9+50, 1) < 0.3 {
			l -= 0.28
		}
		return pick(icTwineR, l)
	})
}

// icRope is a coil of rope with its end hanging out.
func icRope(p *icv) {
	shRope := func(cx, cy, rx, ry float64) ishader {
		return func(f *ifrag) color.RGBA {
			ang := math.Atan2((f.y-cy)/ry, (f.x-cx)/rx)
			l := bulge(f, 0.85)
			if math.Mod(ang*7+f.d*2.2+50, 1) < 0.32 {
				l -= 0.26
			}
			return pick(icRopeR, l)
		}
	}
	// The coils, back to front: each a ring seen from above.
	for i, c := range [][4]float64{{8.4, 6.4, 5.6, 3.0}, {7.8, 8.4, 5.9, 3.2}, {8.2, 10.4, 5.7, 3.1}} {
		_ = i
		p.fill(sRing(c[0], c[1], c[2], c[3], 0.95), shRope(c[0], c[1], c[2], c[3]))
	}
	// The end, out over the front and down.
	end := []float64{12.4, 12.0, 13.6, 13.6, 14.2, 15.0}
	a := mkAxis(end[0], end[1], end[4], end[5])
	p.fill(sPath(0.9, 0.8, end...), func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, 0.9)
		if math.Mod(u*2.4+v*1.5+50, 1) < 0.32 {
			l -= 0.26
		}
		return pick(icRopeR, l)
	})
	// Its frayed tip.
	p.fill(sEll(14.3, 15.2, 0.75, 0.45), shBulge(icStrawR, 0.4, 0.1))
}

// --- food --------------------------------------------------------------------------

// icBerry is three red berries on their stalks, with a leaf.
func icBerry(p *icv) {
	sk := func(x0, y0, x1, y1 float64) {
		a := mkAxis(x0, y0, x1, y1)
		p.fill(sCap(x0, y0, x1, y1, 0.34, 0.28), shRod(icStalk, a, 0.32, 0, 0, 0))
	}
	sk(8.8, 3.6, 5.2, 8.0)
	sk(8.8, 3.6, 11.2, 8.4)
	sk(8.8, 3.6, 8.2, 10.4)
	// The leaf, a lens along its rib, out to the right.
	ang := -0.5
	lx, ly := 8.8, 3.4
	leaf := iEll(lx+math.Cos(ang)*2.6, ly+math.Sin(ang)*2.6, 2.75, 1.3, ang)
	rib := mkAxis(lx, ly, lx+math.Cos(ang)*5, ly+math.Sin(ang)*5)
	p.fill(leaf, func(f *ifrag) color.RGBA {
		_, v := rib.uv(f.x, f.y)
		l := bulge(f, 1.0)
		if math.Abs(v) < 0.14 {
			l -= 0.25
		}
		if v > 0 {
			l -= 0.12
		}
		return pick(iiLeaf, l)
	})
	berry := func(cx, cy, r float64) {
		p.fill(sEll(cx, cy, r, r), shSphere(iiBerry, cx, cy, r, r, -0.06, true))
	}
	berry(4.8, 9.8, 2.85)
	berry(11.3, 10.1, 2.8)
	berry(8.0, 12.4, 2.9)
}

// icMeat is a raw steak on the bone: red meat marbled with fat on its cut
// face, a rind of fat round its side, the bone's end standing out.
func icMeat(p *icv) {
	// The bone first, under the meat.
	bx, by := 12.9, 12.5
	p.fill(sCap(9.6, 9.4, bx, by, 0.95, 0.85), shRod(icBoneR, mkAxis(9.6, 9.4, bx, by), 0.9, 0.05, 0, 0))
	p.fill(sEll(bx+0.5, by-0.4, 1.05, 1.05).or(sEll(bx-0.3, by+0.6, 1.05, 1.05)), shBulge(icBoneR, 0.8, 0.05))
	face := iBlend(2.2, sEll(7.2, 6.4, 5.9, 3.9), sEll(10.2, 8.4, 3.0, 2.3), sEll(4.0, 8.0, 2.6, 2.1))
	// Its side, the face moved down: fat at the top, meat below.
	p.fill(face.moved(0, 2.0).or(face.moved(0, 1.0)), func(f *ifrag) color.RGBA {
		below := face.d(f.x, f.y)
		l := 0.5 - (f.x-8)*0.03
		if below < 0.75 {
			return pick(icFatR, l+0.2)
		}
		if math.Mod(f.x*0.9+fbm2(f.x, f.y, 96, 2)*1.4, 1) < 0.14 {
			l -= 0.18 // the grain of the muscle, across the side
		}
		return pick(icRawR, l-0.25)
	})
	p.fill(face, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.1) + 0.02
		// The fat rind round the back of the face.
		if -f.d < 0.62 && toward(f.gx, f.gy) > -0.1 {
			return pick(icFatR, l)
		}
		// Marbling: thin veins of fat winding through the red, branching,
		// fading out here and there.
		n := fbm2(f.x*0.42, f.y*0.55, 91, 3)
		vein := math.Abs(n - 0.5)
		if w := 0.035 * (0.4 + fbm2(f.x*0.8, f.y*0.8, 92, 2)); vein < w {
			return mix(pick(icRawR, l), icFatR[1+int(l*1.9)], 0.8)
		}
		m := fbm2(f.x*0.7+9, f.y*0.9, 93, 2)
		if math.Abs(m-0.5) < 0.018 {
			return mix(pick(icRawR, l), icFatR[1], 0.55)
		}
		return pick(icRawR, l)
	})
	// Wet glints on the lit side.
	p.on(sEll(4.6, 5.4, 0.5, 0.26).or(sEll(6.0, 4.3, 0.32, 0.18)).or(sEll(3.4, 7.0, 0.24, 0.18)), solidC(icGlint))
}

// icRoast is roast meat on the bone: a browned haunch with the grill's
// marks across it and the bone's knob out of its end.
func icRoast(p *icv) {
	bx, by := 12.8, 12.6
	p.fill(sCap(9.0, 9.0, bx, by, 0.9, 0.82), shRod(icBoneR, mkAxis(9.0, 9.0, bx, by), 0.86, 0.05, 0, 0))
	p.fill(sEll(bx+0.5, by-0.4, 1.05, 1.05).or(sEll(bx-0.3, by+0.6, 1.05, 1.05)), shBulge(icBoneR, 0.8, 0.05))
	fr := frameOf(6.4, 6.6, 0.7, 0.7)
	haunch := iBlend(2.4, iEll(6.4, 6.6, 5.4, 4.2, math.Pi/4), iEll(9.6, 9.6, 2.2, 1.7, math.Pi/4))
	p.fill(haunch, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.4) - 0.04 + (fbm2(f.x*1.4, f.y*1.4, 95, 2)-0.5)*0.22
		s, _ := fr.st(f.x, f.y)
		// The grill's marks: bands across the haunch.
		if m := math.Mod(s+50.4, 2.1); m < 0.42 && -f.d > 0.3 {
			l -= 0.36
		}
		if -f.d < 0.55 && toward(f.gx, f.gy) < 0 {
			l -= 0.12 // the crisp dark edge
		}
		return pick(icRoastR, l)
	})
	// The juices' gleam on top.
	p.on(sTaper(0.1, 0.26, 0.08, 3.2, 6.2, 4.4, 4.0, 6.6, 2.8), solidC(mix(icGlint, icRoastR[4], 0.3)))
	p.on(sEll(4.4, 7.6, 0.32, 0.22), solidC(icGlint))
}

// icFish is a fish on its side, nose to the left, its tail a little up.
func icFish(p *icv) {
	fr := frameOf(1.0, 10.4, 0.96, -0.28) // nose to tail; t is up its side
	body := iBlend(1.2,
		sTaper(0.95, 2.9, 0.7, fr.path(0, 0, 3.4, 0.1, 7.2, 0.05, 10.9, 0)...),
		fr.poly(4.4, 2.2, 6.6, 3.7, 8.8, 2.0))
	tail := fr.poly(10.2, 0, 13.4, 3.0, 14.4, 2.7, 13.1, 0, 14.4, -2.7, 13.4, -3.0)
	fin := fr.poly(4.6, -1.8, 5.6, -3.7, 7.2, -1.9)
	p.fill(fin, shBulge(icFishR, 0.5, -0.05))
	p.fill(tail, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := 0.55 + t*0.06
		if math.Mod(math.Atan2(t, s-10.2)*9+50, 1) < 0.3 {
			l -= 0.2 // its rays
		}
		return pick(icFishR, l)
	})
	p.fill(body, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := bulge(f, 1.6)
		// Silver below, dark on the back.
		if t < -0.85 {
			return pick(icBellyR, l+0.08)
		}
		// Scales: rows of arcs along the back.
		u, v := s*1.6, t*1.6
		row := math.Floor(v)
		cu := math.Floor(u+row*0.5) + 0.5 - row*0.5
		if d := math.Hypot(u-cu-0.5, (v-row-0.5)*1.4); d > 0.42 && d < 0.6 && s > 3.0 && s < 10.4 {
			l -= 0.14
		}
		c := pick(icFishR, l)
		// A rosy stripe along its side, as a trout's.
		if t > -0.85 && t < -0.25 && s > 2.8 && s < 11 {
			c = mix(c, hex(0xd16974), 0.45)
		}
		return c
	})
	// The gill's curve and the eye.
	gx, gy := fr.at(2.9, 0)
	p.on(sRing(gx, gy, 1.0, 1.7, 0.13).and(sEll(gx-1.0, gy+0.3, 1.5, 2.6)), solidC(icFishR[0]))
	ex, ey := fr.at(1.45, 0.5)
	p.on(sEll(ex, ey, 0.56, 0.56), solidC(hex(0xe1a845)))
	p.on(sEll(ex+0.06, ey+0.04, 0.36, 0.36), solidC(hex(0x1a1f2e)))
	p.on(sEll(ex-0.16, ey-0.16, 0.13, 0.13), solidC(icGlint))
}

// icBanana is a hand of three bananas hanging from their stalk.
func icBanana(p *icv) {
	sx, sy := 12.2, 3.8 // where they join the stalk
	one := func(cx, cy, tx, ty, w float64) {
		const n = 10
		pts := make([]float64, 0, 2*(n+1))
		for i := 0; i <= n; i++ {
			t := float64(i) / n
			a, b, c := (1-t)*(1-t), 2*t*(1-t), t*t
			pts = append(pts, a*sx+b*cx+c*tx, a*sy+b*cy+c*ty)
		}
		s := sTaper(0.45, w, 0.32, pts...)
		p.fill(s, func(f *ifrag) color.RGBA {
			l := bulge(f, w*0.85) + 0.06
			// A ridge along its upper side: bananas are not round.
			if r := -f.d / w; r > 0.42 && r < 0.56 && f.gy < 0 {
				l += 0.16
			}
			return pick(icBananaR, l)
		})
		// The brown tip.
		p.on(sEll(tx, ty, 0.62, 0.62), shBulge(icPeel, 0.4, 0.15))
	}
	one(7.0, 9.6, 1.8, 6.2, 1.55)
	one(8.2, 12.2, 2.6, 11.2, 1.6)
	one(10.0, 13.4, 5.6, 14.6, 1.6)
	// The stalk they hang from: green and fibrous, ridged along its length,
	// the crown where the hand grows from it knobbed, its end cut pale.
	a := mkAxis(sx, sy+0.4, 14.0, 0.9)
	p.fill(sCap(sx-0.4, sy+0.8, 14.0, 0.9, 1.25, 0.85), func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, 1.0) + 0.1
		if math.Abs(math.Sin(v*4.4+fbm2(u*0.4, v, 89, 2)*1.5)) > 0.86 {
			l -= 0.18 // its ridges
		}
		if u < 1.1 {
			return pick(icPeel, l-0.1) // the crown, brown and knobbed
		}
		return pick(icStalk, l-0.12)
	})
	p.fill(iEll(14.0, 0.9, 0.48, 0.82, math.Atan2(a.ey, a.ex)), func(f *ifrag) color.RGBA {
		d := math.Hypot(f.x-14.0, f.y-0.9)
		if d < 0.3 {
			return pick(icStalk, 0.55)
		}
		return pick(icStalk, 0.9) // the cut, pale and wet
	})
}

// icClay is wet clay as it is dug by the water: smooth lumps slumped in a
// puddle of their own slip, a thumb pressed into the big one, everything
// slick and shining.
func icClay(p *icv) {
	cr := clayRamp
	// The puddle of slip they sit in: flat, dark, a sky's glint on it.
	puddle := iBlend(1.2, sEll(8.0, 13.2, 7.0, 2.2), sEll(3.6, 12.6, 2.6, 1.6)).rough(0.2, 0.9, 100)
	p.fill(puddle, func(f *ifrag) color.RGBA {
		l := 0.34 + 0.1*toward(f.gx, f.gy)*clamp01(1+f.d/0.5)
		if math.Abs(f.y-12.4-(f.x-8)*0.05) < 0.16 && f.x > 9.6 && f.x < 13.4 {
			return mix(pick(cr, 0.5), hex(0xd8e8f0), 0.7)
		}
		return pick(cr, l)
	})
	lump := func(s shape, cx, cy, rx, ry, bias float64, seed uint32) {
		p.fill(s, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
			l := litOf(nx*0.9, ny*0.9) + bias + (fbm2(f.x*0.8, f.y*0.8, seed, 2)-0.5)*0.1
			if nx*0.45+ny*0.89 > 0.62 {
				l += 0.12 // the slip's light thrown up under it
			}
			return pick(cr, l)
		})
	}
	big := iBlend(1.6, sEll(7.2, 8.0, 5.4, 4.6), sEll(7.6, 11.0, 6.0, 2.4)).rough(0.15, 1.6, 101)
	lump(big, 6.8, 8.6, 5.8, 5.0, 0.0, 102)
	// A thumb pressed into it: a dish, dark on its lit side, lit on the far.
	cx, cy, r := 8.2, 7.0, 1.6
	p.on(sEll(cx, cy, r, r*0.75), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/r, (f.y-cy)/(r*0.75)
		return pick(cr, litOf(-nx*0.85, -ny*0.85)-0.08)
	})
	small := sEll(12.8, 10.8, 2.2, 1.9)
	lump(small, 12.8, 10.8, 2.2, 1.9, 0.02, 103)
	lump(sEll(2.8, 11.4, 1.3, 1.1), 2.8, 11.4, 1.3, 1.1, 0.04, 104)
	// The wet shine: sharp, cool highlights.
	wet := hex(0xfff6f0)
	p.on(sTaper(0.06, 0.28, 0.05, 2.6, 8.6, 3.4, 5.6, 6.0, 3.8), solidC(wet))
	p.on(sEll(9.4, 6.2, 0.36, 0.2), solidC(wet))
	p.on(sTaper(0.05, 0.18, 0.04, 11.2, 10.2, 11.8, 9.4, 12.8, 9.1), solidC(wet))
	p.on(sEll(2.4, 10.9, 0.3, 0.18), solidC(wet))
}

// icHide is a hide scraped and laid out flat: the pale flesh side up,
// creased where it was stretched, the fur showing round its edge, one
// corner folded back to show the fur side, the legs and tail out at its
// corners.
func icHide(p *icv) {
	pelt := sPoly(
		4.2, 2.6, 6.2, 3.6, 8.0, 2.8, 9.8, 3.6, 11.8, 2.4, 12.4, 4.2, 11.6, 5.6, 12.6, 8.0,
		11.8, 10.6, 13.2, 12.4, 12.4, 14.0, 10.4, 12.8, 8.2, 13.8, 6.0, 12.8, 3.6, 14.0, 2.8, 12.4,
		4.2, 10.6, 3.4, 8.0, 4.4, 5.6, 3.4, 4.2)
	fur := iBlend(0.8, pelt).rough(0.35, 0.4, 111)
	shFurEdge := func(f *ifrag) color.RGBA {
		l := bulge(f, 0.9) + (fbm2(f.x*3.5, f.y*0.9, 112, 2)-0.5)*0.45
		return pick(icPelt, l)
	}
	p.fill(fur, shFurEdge)
	inner := iBlend(0.8, pelt).grow(-0.8)
	p.fill(inner, func(f *ifrag) color.RGBA {
		// Flat, mottled, a little lit towards the top left; the edge rolls.
		l := 0.6 - (f.x+f.y-16)*0.018 + (fbm2(f.x*0.9, f.y*0.9, 113, 3)-0.5)*0.32
		if -f.d < 0.55 {
			l += 0.18 * toward(f.gx, f.gy)
		}
		// Creases where it was stretched: from the middle out to the legs.
		for _, c := range [][4]float64{{8, 8, 3.4, 3.6}, {8, 8, 3.4, 12.6}, {8, 8, 12.4, 12.6}} {
			a := mkAxis(c[0], c[1], c[2], c[3])
			u, v := a.uv(f.x, f.y)
			if u > 1.2 && u < a.l-1.0 && math.Abs(v-0.3*math.Sin(u*1.3)) < 0.14 {
				l -= 0.2
			} else if u > 1.2 && u < a.l-1.0 && math.Abs(v-0.3*math.Sin(u*1.3)-0.26) < 0.1 {
				l += 0.12
			}
		}
		return pick(icHideIn, l)
	})
	// The spine's line down the middle.
	p.on(sPath(0.2, 0.14, 8.0, 3.8, 8.2, 8.0, 8.1, 12.6), func(f *ifrag) color.RGBA {
		return mix(p.px[f.iy*p.n+f.ix], icHideIn[0], 0.35)
	})
	// The top right corner folded back over: its fur side up.
	flap := sPoly(9.4, 3.4, 12.2, 2.8, 11.6, 6.2)
	p.fill(flap.grow(0.25).rough(0.25, 0.35, 114), shFurEdge)
	p.fill(flap, func(f *ifrag) color.RGBA {
		l := 0.55 + (fbm2(f.x*2.6, f.y*0.8, 115, 2)-0.5)*0.5
		if math.Abs(math.Sin((f.x*2.4+f.y*0.8)*math.Pi)) > 0.86 {
			l -= 0.15
		}
		return pick(icPelt, l)
	})
}

// icBone is a bone with a knobbed end each side.
func icBone(p *icv) {
	ax, ay, bx, by := 4.4, 11.6, 11.6, 4.4
	a := mkAxis(ax, ay, bx, by)
	cx, cy := a.across()
	knob := func(x, y float64) shape {
		return iBlend(0.9, sEll(x+cx*1.15, y+cy*1.15, 1.75, 1.75), sEll(x-cx*1.15, y-cy*1.15, 1.75, 1.75))
	}
	bone := iBlend(1.2, sCap(ax, ay, bx, by, 1.05, 1.05), knob(ax-a.ex*0.6, ay-a.ey*0.6), knob(bx+a.ex*0.6, by+a.ey*0.6))
	p.fill(bone, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.3) + 0.02
		return pick(icBoneR, l)
	})
	// A hairline crack and a worn pit.
	p.on(sPath(0.1, 0.07, 7.4, 8.0, 7.9, 7.8, 8.2, 7.1), solidC(icBoneR[1]))
}

// icStew is a clay bowl of stew, steaming: meat, roots and greens in a
// rich broth.
func icStew(p *icv) {
	cx, rimY := 8.0, 8.2
	body := sEll(cx, rimY, 6.9, 5.8).and(sBelow(rimY))
	p.fill(body, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/6.9, (f.y-rimY)/5.8
		l := litOf(nx*0.95, ny*0.6) + 0.04
		if math.Abs(f.y-(rimY+1.9)) < 0.28 {
			l -= 0.3 // a scored band
		}
		return pick(icBowlR, l)
	})
	p.under(sEll(cx, 13.4, 3.2, 1.2), shBulge(icBowlR, 0.6, -0.3)) // its foot, under the body
	p.fill(sEll(cx, rimY, 6.9, 2.5), func(f *ifrag) color.RGBA {
		return pick(icBowlR, 0.78-(f.x-cx)/6.9*0.25)
	})
	broth := sEll(cx, rimY+0.15, 5.9, 1.85)
	p.fill(broth, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/5.9, (f.y-rimY)/1.85
		l := 0.55 - nx*0.15 + ny*0.12
		if ny < -0.55 {
			l -= 0.3 // the rim's shadow on it
		}
		return pick(icBrothR, l)
	})
	// What is in it.
	chunk := func(x, y, rx, ry float64, r ramp) {
		p.fill(sBox(x-rx, y-ry, x+rx, y+ry, 0.25).and(broth.grow(0.4)), func(f *ifrag) color.RGBA {
			l := 0.55 - (f.x-x)/rx*0.2 - (f.y-y)/ry*0.25
			return pick(r, l)
		})
	}
	chunk(6.0, 7.9, 1.05, 0.7, ciDark)
	chunk(9.6, 8.6, 1.0, 0.65, ciDark)
	p.fill(sEll(10.9, 7.6, 0.85, 0.5), shBulge(icCarrot, 0.5, 0.05))
	p.fill(sEll(4.6, 8.8, 0.8, 0.48), shBulge(icCarrot, 0.5, 0.05))
	p.fill(iEll(7.9, 9.0, 1.2, 0.42, 0.3), shBulge(icGreenR, 0.4, 0.05))
	p.fill(iEll(8.2, 7.2, 0.95, 0.35, -0.4), shBulge(icGreenR, 0.4, 0.05))
	// Steam rising.
	for i, x := range []float64{5.8, 8.6, 11.0} {
		x, sd := x, uint32(120+i)
		p.soft(sBox(x-1.6, 0.0, x+1.6, 6.6, 0), func(f *ifrag) color.RGBA {
			t := (6.6 - f.y) / 6.6
			wob := math.Sin(f.y*1.3+float64(sd)) * 0.7 * t
			d := math.Abs(f.x-x-wob) / (0.45 + 0.4*t)
			a := clamp01(1-d) * clamp01(t*3) * clamp01((1-t)*2.2) * 0.55
			return color.RGBA{0xf4, 0xf0, 0xe8, uint8(255 * a)}
		})
	}
}
