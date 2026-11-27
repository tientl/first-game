package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The held painters (heldhd.go): each item in its own frame, art pixels,
// the grip at (0, 0), the working end up, set to its real size beside the
// 31-pixel figure by its scale (TestHeldSizes): an axe, a pick or a hoe 13
// pixels from butt to head, a knife 7.5, a spear 37, a torch 11, a rod 31.
// Tools stand upright with the head's edge forward (+u); the stone of a
// tool's head is its tier's (heldMats).

func init() {
	for t := sim.TierStone; t < sim.NumTiers; t++ {
		m := heldMats[t]
		regHeld(sim.ToolItem(sim.ToolAxe, t), heldAxe(m, t))
		regHeld(sim.ToolItem(sim.ToolPickaxe, t), heldPick(m, t))
		regHeld(sim.ToolItem(sim.ToolHoe, t), heldHoe(m, t))
		if t > sim.TierStone {
			regHeld(sim.SwordItem(t), heldKnife(m, t))
		}
	}
	regHeld(sim.ItemSpear, heldSpearDef)
	regHeld(sim.ItemIvorySpear, heldIvorySpearDef)
	regHeld(sim.ItemBattleAxe, heldBattleAxeDef)
	regHeld(sim.ItemTorch, heldTorchDef)
	regHeld(sim.ItemRod, heldRodDef)
}

// --- materials --------------------------------------------------------------------

// heldMat is the stone of a tool's head.
type heldMat struct {
	r     ramp
	knap  float64    // the scars' size, art pixels; 0 for stone ground smooth
	gloss float64    // 0 matt to 1 glassy: how sharply it catches the light
	bands bool       // agate's bands
	glint color.RGBA // the light it catches
}

var heldMats = [sim.NumTiers]heldMat{
	{r: icStoneR, knap: 1.9},                                // stone: knapped in broad scars, matt
	{r: icWFlintR, knap: 1.25, gloss: 0.55, glint: icGlint}, // flint: finer scars, a waxy gloss
	{r: mkRamp(0x0e3326, 0x18523c, 0x2a8260, 0x49ab84, 0x8fd6b2), gloss: 0.8, // jade: ground and polished
		glint: color.RGBA{0xe0, 0xff, 0xf0, 0xff}},
	{r: mkRamp(0x4a1812, 0x8a3222, 0xbb5733, 0xdf8a56, 0xf3c39a), gloss: 0.8, bands: true,
		glint: color.RGBA{0xff, 0xf4, 0xe0, 0xff}},
	{r: mkRamp(0x08070c, 0x17121f, 0x2c2338, 0x4e4266, 0x8a7cb0), knap: 1.6, gloss: 1, // glass: big shell-like scars
		glint: color.RGBA{0xf0, 0xec, 0xff, 0xff}},
}

// heldSun is the sun's direction (lightDir).
var heldSun = lightDir

// mirror is how much of the sun a surface with screen normal (nx, ny)
// throws towards the viewer: 1 straight at them.
func mirror(nx, ny float64) float64 {
	nz := math.Sqrt(max(0, 1-nx*nx-ny*ny))
	dot := nx*heldSun[0] + ny*heldSun[1] + nz*heldSun[2]
	return 2*dot*nz - heldSun[2]
}

// face is the colour of m on a face whose normal leans (nu, nv) in the
// item's frame, at item point (u, v), lift lighter; (cu, cv) is the middle
// of the head, for agate's bands. Near its edge the face turns out to it,
// a bevel, so the edges facing the light are lit and glassy stone glints
// there.
func (m heldMat) face(h *hpv, f *ifrag, u, v, nu, nv, lift, cu, cv float64, seed uint32) color.RGBA {
	nx, ny := h.dir(nu, nv)
	const bevel = 0.55
	edge := 1 - clamp01(-f.d/bevel)
	if edge > 0 {
		t := math.Sqrt(edge) * 0.8
		nx, ny = lerpf(nx, f.gx, t), lerpf(ny, f.gy, t)
	}
	l := litOf(nx, ny) + lift
	ridge := false
	if m.knap > 0 && edge < 0.9 {
		l, ridge = h.knapAt(u, v, l, m.knap, seed)
	} else if m.knap == 0 {
		l += (fbm2(u*0.8, v*0.8, seed, 2) - 0.5) * 0.14 // a cloud in the stone
	}
	if m.bands { // agate: bands round the middle, wavering
		d := math.Hypot(u-cu, (v-cv)*1.25) + 0.9*fbm2(u*0.45, v*0.45, seed+3, 2)
		switch b := math.Mod(d*1.5, 1); {
		case b < 0.2:
			l += 0.2
		case b < 0.32:
			l -= 0.12
		}
	}
	if m.gloss > 0 {
		// A glint where an edge or a ridge turns the sun to the eye; a flat
		// face that does only brightens, or a carried blade would flash white.
		r := mirror(nx, ny)
		switch {
		case r > 0.985-0.02*m.gloss && (edge > 0.25 || ridge):
			return m.glint
		case r > 0.9:
			l += 0.12 * m.gloss
		}
		if ridge && l > 0.6 {
			l += 0.15 * m.gloss
		}
	}
	return pick(m.r, l)
}

// knapAt is knap (iconhd_pilot.go) on a held item: scars that stay on the
// item as it turns, each a shallow dish whose normal is turned onto the
// canvas, with a lit ridge between two. ridge is true on it.
func (h *hpv) knapAt(u, v, base, scale float64, seed uint32) (float64, bool) {
	gx, gy := u/scale, v/scale
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
	// Inside a scar the surface leans in towards its middle, more at its rim.
	k := 0.55 * math.Min(1, best*1.4)
	X, Y := h.dir(-(gx-bx)*k, -(gy-by)*k)
	l := base + (litOf(X, Y)-0.62)*0.75
	ridge := second-best < 0.09
	switch {
	case ridge:
		l += 0.2
	case second-best < 0.17:
		l -= 0.08 // the crease beside it
	}
	return l, ridge
}

// shWood shades a round wooden rod lying along v, rad thick: lit across
// it, grain streaking along it, fixed to the wood.
func (h *hpv) shWood(r ramp, rad, bias float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		l := h.rod(u, rad) + bias
		n := fbm2(v*0.5, u*2.4/rad+7, seed, 2)
		if s := math.Abs(math.Sin((u/rad*1.8 + n*3.2) * math.Pi)); s > 0.9 {
			l -= 0.17
		}
		if k := vnoise(v*0.8+3, u*0.6, seed+5); k > 0.88 {
			l -= 0.12 // a knot, or a darker streak
		}
		return pick(r, l)
	}
}

// haft paints a wooden haft along the item's axis from v0 (its butt) to
// v1, r0 thick at the butt and r1 at the top.
func (h *hpv) haft(v0, v1, r0, r1 float64, r ramp, seed uint32) {
	h.fill(sCap(0, v0, 0, v1, r0, r1), h.shWood(r, (r0+r1)/2, 0, seed))
}

// icLashR is rawhide lashing as held: a step darker than an icon's, so it
// stands out on a pale haft.
var icLashR = mkRamp(0x2b1c22, 0x533b41, 0x825646, 0xb07850, 0xd8a878)

// lash binds the haft from v0 to v1 with rawhide, hw to each side: turns of
// cord across it, a little slanted, round with the haft.
func (h *hpv) lash(v0, v1, hw float64, r ramp, seed uint32) {
	h.fill(sBox(-hw, v0, hw, v1, 0.3).rough(0.05, 0.4, seed), func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		l := h.rod(u, hw*1.05) - 0.04
		if math.Mod(v*1.1+u*0.3+50, 1) < 0.3 {
			l -= 0.32 // between two turns
		}
		return pick(r, l)
	})
}

// strap is a cord from (u0, v0) to (u1, v1), round across: a turn of a
// lashing crossing a head.
func (h *hpv) strap(u0, v0, u1, v1, w float64, r ramp) {
	a := mkAxis(u0, v0, u1, v1)
	h.fill(sCap(u0, v0, u1, v1, w, w), func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		s, t := a.uv(u, v)
		cx, cy := a.across()
		l := h.lit(cx*t/w, cy*t/w)
		if math.Mod(s*1.6+50, 1) < 0.25 {
			l -= 0.25
		}
		return pick(r, l)
	})
}

// --- tools ----------------------------------------------------------------------------

// heldAxe is the axe of stone m: a knapped head lashed to a wooden haft,
// the edge forward.
func heldAxe(m heldMat, t sim.Tier) *heldDef {
	seed := uint32(5 + 10*t)
	return &heldDef{x0: -2.6, y0: -15.2, x1: 7, y1: 3.6, grip: 0.85, scale: 0.75, head: 9.5, reach: 15.2, paint: func(h *hpv) {
		h.haft(2.8, -13.1, 0.84, 0.74, iiWood, 11)
		head := sPoly(-1.8, -11.6, -0.8, -12.5, 1.8, -12.2, 3.6, -13.8, 5.0, -14.5, 6.0, -13.0,
			6.4, -10.3, 6.0, -7.6, 5.0, -6.0, 3.6, -6.7, 1.8, -8.4, -0.8, -8.2, -1.8, -9.0).rough(0.08, 0.5, seed)
		h.fill(head, func(f *ifrag) color.RGBA {
			u, v := h.loc(f.x, f.y)
			// Two faces meeting along a ridge from the butt to the edge, the
			// edge ground back from them.
			ridge := -10.3 + 0.05*u
			nu, nv, lift := 0.08, -0.55, 0.0
			if v > ridge {
				nv = 0.5
			}
			if math.Abs(v-ridge) < 0.22 {
				lift = 0.14
			}
			if u > 4.6 {
				nu += 0.4
				lift += 0.08
			}
			return m.face(h, f, u, v, nu, nv, lift, 2.6, -10.3, seed+1)
		})
		h.lash(-12.4, -8.0, 1.1, icLashR, seed+2)
		h.strap(-0.9, -12.1, 2.3, -8.5, 0.32, icLashR)
		h.strap(-0.9, -8.5, 2.3, -12.1, 0.32, icLashR)
	}}
}

// heldPick is the pickaxe of stone m: a curved head with two points across
// the top of the haft, the front one the longer.
func heldPick(m heldMat, t sim.Tier) *heldDef {
	seed := uint32(9 + 10*t)
	return &heldDef{x0: -7.4, y0: -14.2, x1: 7.6, y1: 3.6, grip: 0.85, scale: 0.8, head: 11.5, reach: 13.5, paint: func(h *hpv) {
		h.haft(2.8, -11.4, 0.84, 0.74, iiWood, 21)
		// The head's middle line: highest over the haft, the points drooping.
		mid := func(u float64) float64 { return -11.7 + 2.0*(u/6.6)*(u/6.6) }
		var front, back []float64
		for i := 0; i <= 8; i++ {
			s := float64(i) / 8
			front = append(front, s*7.0, mid(s*7.0))
			back = append(back, -s*6.2, mid(-s*6.2))
		}
		head := sPath(1.45, 0.14, front...).or(sPath(1.45, 0.18, back...))
		h.fill(head, func(f *ifrag) color.RGBA {
			u, v := h.loc(f.x, f.y)
			hw := lerpf(1.45, 0.16, math.Abs(u)/7)
			// Round across the head: the normal leans out from its middle line.
			tt := max(-1, min(1, (v-mid(u))/hw))
			slope := 2 * 2.0 * u / (6.6 * 6.6)
			lift := 0.0
			if math.Abs(tt) < 0.16 {
				lift = 0.1 // the crest
			}
			return m.face(h, f, u, v, -slope*tt*0.5, tt*0.7, lift, 0, -11.4, seed)
		})
		h.lash(-12.8, -9.8, 1.1, icLashR, seed+2)
		h.strap(-1.0, -12.6, 1.0, -10.0, 0.3, icLashR)
	}}
}

// heldHoe is the hoe of stone m: a flat blade bent like a 7, lashed to the
// top of the haft, its edge hanging forward and down, so it does not read
// as an axe.
func heldHoe(m heldMat, t sim.Tier) *heldDef {
	seed := uint32(33 + 10*t)
	return &heldDef{x0: -1.8, y0: -14.6, x1: 7.8, y1: 3.6, grip: 0.85, scale: 0.78, head: 10, reach: 13, paint: func(h *hpv) {
		h.haft(2.8, -12.7, 0.82, 0.72, iiWood, 31)
		blade := sPoly(-0.7, -13.4, 3.6, -13.8, 5.2, -13.2, 6.3, -10.6, 7.2, -7.4, 7.6, -5.9, 5.8, -5.5,
			4.0, -5.6, 3.9, -7.0, 3.4, -9.6, 2.4, -11.4, -0.7, -11.6).rough(0.07, 0.5, seed)
		h.fill(blade, func(f *ifrag) color.RGBA {
			u, v := h.loc(f.x, f.y)
			// The arm faces up; the hanging blade faces forward, ground to its
			// edge at the bottom.
			nu, nv, lift := 0.1, -0.45, 0.0
			if u > 3.2 && v > -11.6+0.4*(u-3.2) {
				nu, nv = 0.42, 0.05
				if v > -7.6 {
					nv = 0.4
					lift = 0.06
				}
			}
			return m.face(h, f, u, v, nu, nv, lift, 4.5, -9.5, seed+1)
		})
		h.lash(-13.9, -10.6, 1.08, icLashR, seed+2)
		h.strap(-0.9, -13.6, 1.6, -10.8, 0.3, icLashR)
	}}
}

// heldKnife is the knife (sword) of stone m: a leaf-shaped blade, a ridge
// down it, bound into a wooden grip with a knob at its end.
func heldKnife(m heldMat, t sim.Tier) *heldDef {
	seed := uint32(40 + t)
	// A hand's length of blade, the grip a fist wide: a quarter of the figure.
	return &heldDef{x0: -1.5, y0: -5.4, x1: 1.5, y1: 2.8, grip: 0.7, head: 2.9, reach: 5.0, paint: func(h *hpv) {
		h.haft(1.8, -0.9, 0.56, 0.5, iiWood, seed)
		h.fill(sEll(0, 1.95, 0.62, 0.56), func(f *ifrag) color.RGBA { // the knob
			u, v := h.loc(f.x, f.y)
			return pick(iiWood, h.lit(u/0.62*0.85, (v-1.95)/0.56*0.85))
		})
		blade := sPoly(-0.6, -0.8, 0.6, -0.8, 0.98, -1.8, 1.05, -2.9, 0.78, -3.9, 0.12, -5.0,
			-0.38, -4.3, -0.8, -3.2, -0.98, -1.9).rough(0.04, 0.3, seed+1)
		h.fill(blade, func(f *ifrag) color.RGBA {
			u, v := h.loc(f.x, f.y)
			// Two faces meeting along a ridge from the grip to the point.
			ridge := 0.04 + 0.01*(v+0.8)
			nu, lift := -0.5, 0.0
			if u > ridge {
				nu = 0.5
			}
			if math.Abs(u-ridge) < 0.12 {
				lift = 0.16
			}
			return m.face(h, f, u, v, nu, -0.1, lift, 0, -2.9, seed+2)
		})
		h.lash(-1.5, -0.5, 0.7, icHideIn, seed+3)
	}}
}

// heldSpearDef is the spear: a long shaft, a knapped flint point lashed on,
// a strip of hide hanging from the lashing.
// It is a head and a half taller than the figure, gripped a third of the way
// up from its butt.
var heldSpearDef = &heldDef{x0: -2.4, y0: -25.4, x1: 2.8, y1: 12.6, grip: 0.8, head: 20.5, reach: 25, carry: carryPole, paint: func(h *hpv) {
	h.haft(12.0, -17.4, 0.58, 0.5, iiWood, 51)
	point := sPoly(-0.8, -16.8, 0.8, -16.8, 1.7, -18.8, 2.05, -20.7, 1.4, -22.8, 0.15, -25.2,
		-1.0, -23.0, -1.85, -20.7, -1.6, -18.8).rough(0.06, 0.35, 52)
	m := heldMats[sim.TierCopper]
	h.fill(point, func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		ridge := 0.03 * (v + 16.8)
		nu, lift := -0.5, 0.0
		if u > ridge {
			nu = 0.5
		}
		if math.Abs(u-ridge) < 0.18 {
			lift = 0.16
		}
		return m.face(h, f, u, v, nu, -0.1, lift, 0, -20.8, 53)
	})
	h.lash(-18.1, -16.0, 0.9, icLashR, 54)
	h.fill(sPath(0.34, 0.22, 0.6, -16.4, 1.5, -15.0, 1.8, -13.4, 1.5, -12.1), shBulge(icLashR, 0.3, 0.05))
}}

// heldIvorySpearDef is the ivory spear: a slender point cut from a tusk,
// bound with hide.
var heldIvorySpearDef = &heldDef{x0: -2.4, y0: -26.4, x1: 2.6, y1: 12.6, grip: 0.8, head: 21.5, reach: 26, carry: carryPole, paint: func(h *hpv) {
	h.haft(12.0, -18.2, 0.6, 0.52, iiWood, 61)
	point := sPoly(-0.9, -17.6, 0.9, -17.6, 1.4, -20.2, 1.2, -23.2, 0.3, -26.0, -0.5, -24.0, -1.2, -21.0)
	h.fill(point, func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		// Round and polished: lit as a tube, a glint where it turns to the
		// light, and the faint rings of the tusk's growth.
		hw := lerpf(1.25, 0.2, clamp01((-v-17.6)/8.4))
		c := lerpf(0, 0.2, clamp01((-v-17.6)/8.4))
		tt := max(-1, min(1, (u-c)/hw))
		nx, ny := h.dir(tt*0.85, -0.12)
		l := litOf(nx, ny) - 0.18
		if math.Mod(-v*1.2+0.3*fbm2(u, v, 62, 2), 1) < 0.16 {
			l -= 0.12
		}
		if mirror(nx, ny) > 0.97 {
			return icGlint
		}
		return pick(icBoneR, l)
	})
	h.lash(-18.8, -16.6, 0.92, icHideIn, 63)
}}

// heldBattleAxeDef is the war axe: a double-bitted jade head, each bit a
// crescent, lashed with hide to a long haft gripped low.
var heldBattleAxeDef = &heldDef{x0: -7.2, y0: -18.6, x1: 7.2, y1: 3.8, grip: 0.9, scale: 0.72, head: 12, reach: 18, paint: func(h *hpv) {
	h.haft(3.0, -17.2, 0.9, 0.8, iiWood, 71)
	m := heldMats[sim.TierIron] // jade
	const cv = -12.0            // the head's middle
	// A bit: a convex edge, drawn back to the middle in two hollows.
	var pts []float64
	for i := -5; i <= 5; i++ {
		a := float64(i) / 5 * 1.08
		pts = append(pts, 0.9+5.7*math.Cos(a), cv+5.7*math.Sin(a))
	}
	pts = append(pts, 2.6, cv+3.1, 1.5, cv+1.8, 0.5, cv+1.5, 0.5, cv-1.5, 1.5, cv-1.8, 2.6, cv-3.1)
	bit := sPoly(pts...)
	head := bit.or(bit.flipped(0)).or(sBox(-1.6, cv-1.7, 1.6, cv+1.7, 0.4))
	h.fill(head, func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		nu, nv, lift := 0.0, -0.5, 0.0
		if v > cv {
			nv = 0.45
		}
		if math.Abs(v-cv) < 0.2 {
			lift = 0.14
		}
		if r := math.Hypot(math.Abs(u)-0.9, v-cv); r > 4.5 {
			nu = math.Copysign(0.42, u) // ground to the edge
		}
		return m.face(h, f, u, v, nu, nv, lift, 0, cv, 72)
	})
	// Hide wound over the middle of the head, criss-crossed.
	h.lash(cv-2.4, cv+2.4, 1.32, icLashR, 73)
	h.strap(-1.3, cv-2.2, 1.3, cv+2.2, 0.32, darkHideRamp)
	h.strap(1.3, cv-2.2, -1.3, cv+2.2, 0.32, darkHideRamp)
	// A wrap of hide where it is held.
	h.lash(-1.6, 1.8, 1.04, darkHideRamp, 74)
}}

// The torch's head, where its flame burns, in its frame.
const torchFlameV = -10.6

// heldTorchDef is a torch: a stick with a bundle of tarred cloth and straw
// bound round its top. Its flame is drawn apart, upright (torchFlame).
var heldTorchDef = &heldDef{x0: -2, y0: -12.6, x1: 2, y1: 3.4, grip: 0.75, scale: 0.76, carry: carryTorch, paint: func(h *hpv) {
	h.haft(2.7, -8.4, 0.64, 0.58, iiWood, 81)
	// Straw ends sticking out of the top of the bundle.
	for i := 0; i < 5; i++ {
		u := -0.9 + 0.45*float64(i)
		top := -11.6 - 0.5*hashf(i, 1, 82)
		h.fill(sCap(u*0.8, -10.4, u*1.25, top, 0.26, 0.16), func(f *ifrag) color.RGBA {
			_, v := h.loc(f.x, f.y)
			return pick(icStrawR, 0.6-0.35*clamp01((-v-10.4)/1.4))
		})
	}
	bundle := sCap(0, -7.4, 0, -10.2, 1.25, 1.5).rough(0.08, 0.5, 83)
	h.fill(bundle, func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		l := h.rod(u, 1.45) + 0.06
		if math.Mod(v*1.0+u*0.45+50, 1) < 0.32 {
			l -= 0.28 // the turns of the wrap
		}
		l -= 0.35 * clamp01((-v-9.2)/1.2) // charred at the top
		return pick(icClothR, l)
	})
	h.lash(-7.9, -6.8, 0.76, icLashR, 84)
}}

// The rod's tip, in its frame: where the line goes from.
const rodTipU, rodTipV = 1.0, -24.4

// heldRodDef is the fishing rod: a long pole of peeled sapling, thinning
// and bending a little to its tip, cord wound where it is held and at the
// tip.
var heldRodDef = &heldDef{x0: -1.4, y0: -25.2, x1: 2.2, y1: 4.6, grip: 0.7, scale: 1.04, carry: carryPole, paint: func(h *hpv) {
	bend := func(v float64) float64 { return rodTipU * (v / rodTipV) * (v / rodTipV) }
	var pts []float64
	for i := 0; i <= 10; i++ {
		v := lerpf(3.9, rodTipV, float64(i)/10)
		pts = append(pts, bend(v), v)
	}
	h.fill(sPath(0.6, 0.2, pts...), func(f *ifrag) color.RGBA {
		u, v := h.loc(f.x, f.y)
		rad := lerpf(0.6, 0.2, clamp01((3.9-v)/(3.9-rodTipV)))
		l := h.rod(u-bend(v), rad)
		if s := math.Abs(math.Sin(((u-bend(v))/rad*1.8 + fbm2(v*0.5, u, 91, 2)*3) * math.Pi)); s > 0.92 {
			l -= 0.16
		}
		if math.Mod(-v*0.21+0.15, 1) < 0.035 {
			l -= 0.25 // a node where a twig was cut
		}
		return pick(iiPale, l)
	})
	h.lash(-1.8, 1.4, 0.76, icLashR, 92)
	tx := bend(-23.2)
	h.fill(sBox(tx-0.4, -23.7, tx+0.4, -22.5, 0.15), shBulge(icRopeR, 0.2, 0.1))
}}
