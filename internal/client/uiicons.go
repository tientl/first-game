package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The interface's own little pictures (the hearts of the vitals, the status
// chips' icons, the buffs of dishes), painted like the HD item icons
// (iconpaint.go): parts in a 16 x 16 design box, shaded in clean bands of a
// ramp, lit from the top left, outlined and shadowed by the bake. Each is
// baked the first time it is shown at a size, in screen pixels.

// uiIcon is one of the interface's pictures.
type uiIcon uint8

const (
	uiHeart uiIcon = iota
	uiCold
	uiPoison
	uiWeb
	uiHungry
	uiStarving
	uiBees
	uiRegen
	uiWarm
	uiSwift
	uiMight
	uiSated
	uiStun   // what the beasts' moves leave: see skillhud.go
	uiBleed
	uiSlow
	uiPinned
	numUIIcons
)

var uiIconPainters = [numUIIcons]iconPainter{
	uiHeart:    icHeart,
	uiCold:     icSnowflake,
	uiPoison:   icPoisonDrop,
	uiWeb:      icWeb,
	uiHungry:   icRoast,
	uiStarving: icGnawedBone,
	uiBees:     icBee,
	uiRegen:    icHeartPlus,
	uiWarm:     icFlame,
	uiSwift:    icFeather,
	uiMight:    icClub,
	uiSated:    icBowl,
	uiStun:     icDazed,
	uiBleed:    icBloodDrop,
	uiSlow:     icSnail,
	uiPinned:   icClaws,
}

type uiIconKey struct {
	i  uiIcon
	px int
}

var uiIconBaked = map[uiIconKey]*ebiten.Image{}

// drawUIIcon draws picture i size UI pixels across, centred on (cx, cy).
func drawUIIcon(d *uiDst, i uiIcon, cx, cy, size float64, alpha float32) {
	px := int(math.Round(size * d.k))
	if px < 4 || i >= numUIIcons {
		return
	}
	key := uiIconKey{i, px}
	img := uiIconBaked[key]
	if img == nil {
		c := bakeIcon(uiIconPainters[i], px)
		img = ebiten.NewImage(px, px)
		img.WritePixels(c.bytes())
		artBytes.Add(int64(4 * px * px))
		uiIconBaked[key] = img
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(d.X(cx)-float64(px)/2), math.Round(d.Y(cy)-float64(px)/2))
	op.ColorScale.ScaleAlpha(alpha)
	d.img.DrawImage(img, op)
}

// statusIcon is the picture of status k.
func statusIcon(k statusKind) uiIcon {
	if i, ok := skillStatusIcon(k); ok {
		return i // see skillhud.go
	}
	switch k {
	case statusCold:
		return uiCold
	case statusPoison:
		return uiPoison
	case statusWeb:
		return uiWeb
	case statusStarving:
		return uiStarving
	case statusHungry:
		return uiHungry
	case statusBees:
		return uiBees
	}
	switch sim.BuffKind(k - statusBuff) {
	case sim.BuffRegen:
		return uiRegen
	case sim.BuffWarm:
		return uiWarm
	case sim.BuffSwift:
		return uiSwift
	case sim.BuffMight:
		return uiMight
	}
	return uiSated
}

var (
	icHeartR    = mkRamp(0x4a0c1c, 0x8c1a2c, 0xc8323a, 0xec5c58, 0xffb0a0)
	icIceR      = mkRamp(0x1e3c6a, 0x3a6aa8, 0x6aa2dc, 0xa8d4f4, 0xeefaff)
	icToxR      = mkRamp(0x12320e, 0x2a6a1a, 0x4ea42a, 0x86d044, 0xd0f490)
	icSilkR     = mkRamp(0x5a5a66, 0x8a8a98, 0xb8b8c6, 0xdcdce6, 0xffffff)
	icBeeR      = mkRamp(0x5a3a06, 0xa86c0a, 0xe0a418, 0xf8d044, 0xfff0a0)
	icWingR     = mkRamp(0x6a7a8a, 0x9aaabb, 0xc8d6e2, 0xe6eef6, 0xffffff)
	icFlameR    = mkRamp(0x7a1a06, 0xc84410, 0xf07c1c, 0xfcc040, 0xfff2b0)
	icFeathR    = mkRamp(0x1e4a5a, 0x2e7a8a, 0x4ab0b0, 0x8ad8c8, 0xd8fff0)
	icClubR     = mkRamp(0x2b1a12, 0x5a3420, 0x8a5430, 0xb87c44, 0xe0aa6a)
	icWoodBowlR = mkRamp(0x3a2214, 0x6a4020, 0x9a6232, 0xc48a4c, 0xecc488)
	icStewR     = mkRamp(0x4a1e0c, 0x86401a, 0xb8682a, 0xdc9a46, 0xf8d08a)
)

// icHeart is a plump red heart with a sharp glint.
func icHeart(p *icv) {
	heart := iBlend(1.6, sEll(5.3, 6.3, 3.6, 3.5), sEll(10.7, 6.3, 3.6, 3.5), sPoly(2.0, 7.6, 14.0, 7.6, 8.0, 14.6))
	p.fill(heart, func(f *ifrag) color.RGBA {
		// Shaded as one round body, like a berry, so the tones run smoothly
		// across it rather than ringing its outline.
		l := litOf((f.x-8)/7.4*0.94, (f.y-7.8)/7.4*0.94) + 0.06
		if gx, gy := f.x-4.6, f.y-4.8; gx*gx*0.9+gy*gy < 1.1 {
			return icGlint
		}
		return pick(icHeartR, l)
	})
	p.on(sEll(11.6, 5.2, 0.55, 0.4), solidC(mix(icGlint, icHeartR[3], 0.4)))
}

// icHeartPlus is the heart with a white cross: health coming back.
func icHeartPlus(p *icv) {
	icHeart(p)
	p.fill(sBox(10.2, 7.6, 15.2, 9.6, 0.5).or(sBox(11.7, 6.1, 13.7, 11.1, 0.5)), shFlat(icSilkR, 4, 0.3))
}

// icSnowflake is a six-armed flake of ice.
func icSnowflake(p *icv) {
	var flake shape
	for i := range 3 {
		a := float64(i)*math.Pi/3 + math.Pi/2
		dx, dy := math.Cos(a)*6.6, math.Sin(a)*6.6
		arm := sCap(8-dx, 8-dy, 8+dx, 8+dy, 0.75, 0.75)
		for _, sgn := range []float64{-1, 1} {
			ex, ey := 8+sgn*dx*0.62, 8+sgn*dy*0.62
			for _, side := range []float64{-0.6, 0.6} {
				b := a + side
				if sgn < 0 {
					b = a + math.Pi + side
				}
				arm = arm.or(sCap(ex, ey, ex+math.Cos(b)*2.4, ey+math.Sin(b)*2.4, 0.55, 0.4))
			}
		}
		if i == 0 {
			flake = arm
		} else {
			flake = flake.or(arm)
		}
	}
	flake = flake.or(sEll(8, 8, 1.6, 1.6))
	p.fill(flake, func(f *ifrag) color.RGBA {
		l := bulge(f, 0.5) + (0.5-f.y/16)*0.3
		return pick(icIceR, l+0.1)
	})
}

// icPoisonDrop is a green drop with a skull's two dark eyes.
func icPoisonDrop(p *icv) {
	drop := iBlend(2.0, sEll(8, 10.4, 4.6, 4.4), sPoly(8, 1.2, 11.6, 8.6, 4.4, 8.6))
	p.fill(drop, shSphere(icToxR, 8, 10, 4.6, 5.4, 0.05, true))
	p.on(sEll(6.3, 10.8, 1.0, 1.15).or(sEll(9.7, 10.8, 1.0, 1.15)), solidC(color.RGBA{0x10, 0x24, 0x0a, 0xff}))
	p.on(sCap(7.2, 13.2, 8.8, 13.2, 0.35, 0.35), solidC(color.RGBA{0x10, 0x24, 0x0a, 0xff}))
}

// icWeb is a spider's web: spokes and turns of silk.
func icWeb(p *icv) {
	var web shape
	for i := range 6 {
		a := float64(i)*math.Pi/3 + 0.3
		s := sCap(8, 8, 8+math.Cos(a)*7.2, 8+math.Sin(a)*7.2, 0.32, 0.28)
		if i == 0 {
			web = s
		} else {
			web = web.or(s)
		}
	}
	web = web.or(sRing(8, 8, 2.4, 2.4, 0.28)).or(sRing(8, 8, 4.6, 4.6, 0.28)).or(sRing(8, 8, 6.6, 6.6, 0.26))
	p.fill(web, func(f *ifrag) color.RGBA { return pick(icSilkR, 0.75+(0.5-f.y/16)*0.3) })
}

// icGnawedBone is a bone picked clean: nothing left to eat.
func icGnawedBone(p *icv) {
	a := mkAxis(3.6, 12.4, 12.4, 3.6)
	p.fill(sCap(3.6, 12.4, 12.4, 3.6, 1.15, 1.15), shRod(icBoneR, a, 1.1, 0.05, 0.3, 7))
	knob := func(x, y float64) {
		p.fill(iBlend(0.8, sEll(x-0.9, y+0.6, 1.5, 1.5), sEll(x+0.6, y-0.9, 1.5, 1.5)), shBulge(icBoneR, 0.9, 0.05))
	}
	knob(3.0, 13.0)
	knob(13.0, 3.0)
}

// icBee is a striped bee with pale wings.
func icBee(p *icv) {
	p.fill(iEll(5.6, 5.0, 3.0, 1.9, -0.9), shBulge(icWingR, 1.0, 0.1))
	p.fill(iEll(10.0, 4.6, 3.0, 1.9, -2.3), shBulge(icWingR, 1.0, 0.1))
	body := iEll(8, 10, 4.8, 3.6, 0.15)
	p.fill(body, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.0)
		if m := math.Mod(f.x*0.98+f.y*0.15+20, 3.0); m < 1.1 && f.x > 6 {
			return pick(mkRamp(0x120c06, 0x241a0c, 0x3a2a14, 0x52401e, 0x6a5428), l)
		}
		return pick(icBeeR, l)
	})
	p.fill(sEll(3.6, 9.4, 1.9, 1.9), shSphere(mkRamp(0x120c06, 0x241a0c, 0x3a2a14, 0x52401e, 0x6a5428), 3.6, 9.4, 1.9, 1.9, 0, true))
	p.fill(sPoly(12.4, 9.0, 15.0, 10.6, 12.4, 11.6), shFlat(mkRamp(0x120c06, 0x241a0c, 0x3a2a14, 0x52401e, 0x6a5428), 2, 0.2))
}

// icFlame is a flame with a hot core: warmth.
func icFlame(p *icv) {
	outer := iBlend(2.2, sEll(8, 11, 4.8, 4.0), sPoly(8, 0.8, 12.2, 9.0, 3.8, 9.0), sPoly(4.0, 3.6, 7.0, 8.4, 4.0, 9.4))
	p.fill(outer, func(f *ifrag) color.RGBA {
		l := 0.3 + (f.y/16)*0.1 + (1-math.Min(1, -f.d/3))*-0.2 + bulge(f, 2.6)*0.45
		return pick(icFlameR, l)
	})
	p.bareLast()
	inner := iBlend(1.4, sEll(8, 12, 2.6, 2.4), sPoly(8, 5.6, 10.2, 11, 5.8, 11))
	p.fill(inner, func(f *ifrag) color.RGBA { return pick(icFlameR, 0.8+bulge(f, 1.5)*0.2) })
	p.bareLast()
	p.soft(sEll(8, 9, 7.6, 7.6), func(f *ifrag) color.RGBA {
		return color.RGBA{0xff, 0xb0, 0x40, uint8(60 * math.Max(0, math.Min(1, -f.d/4)))}
	})
}

// icFeather is a light feather: speed.
func icFeather(p *icv) {
	fr := frameOf(3.0, 13.4, 0.62, -0.78)
	vane := fr.poly(1.6, 0, 4.0, 3.4, 8.6, 3.0, 12.4, 0.6, 12.8, 0, 12.4, -0.6, 8.6, -3.0, 4.0, -3.4)
	p.fill(vane, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := bulge(f, 1.4) + t*0.05
		if math.Mod(s*1.6-math.Abs(t)*0.9+40, 1) < 0.22 {
			l -= 0.2 // the barbs
		}
		return pick(icFeathR, l)
	})
	x0, y0 := fr.at(-0.6, 0)
	x1, y1 := fr.at(12.6, 0)
	p.fill(sCap(x0, y0, x1, y1, 0.42, 0.18), shRod(icBoneR, mkAxis(x0, y0, x1, y1), 0.4, 0.1, 0, 0))
}

// icClub is a knotted club: might.
func icClub(p *icv) {
	a := mkAxis(3.0, 13.6, 11.6, 4.0)
	p.fill(iBlend(1.8, sCap(3.0, 13.6, 9.6, 6.2, 0.95, 2.2), sEll(11.0, 4.8, 3.2, 3.0)), shRod(icClubR, a, 2.4, 0, 0.6, 31))
	p.on(sEll(10.0, 6.4, 0.6, 0.5).or(sEll(12.2, 3.6, 0.55, 0.45)).or(sEll(8.0, 8.4, 0.45, 0.4)), solidC(icClubR[0]))
	binding(p, a, 4.6, 11.8, 1.8, 1.1)
}

// icBowl is a wooden bowl of stew: a full belly.
func icBowl(p *icv) {
	bowl := sEll(8, 9.4, 7.0, 5.0).sub(sOver(func(float64) float64 { return 8.6 }))
	p.fill(sEll(8, 8.8, 6.6, 2.2), func(f *ifrag) color.RGBA {
		l := 0.55 + (fbm2(f.x*1.6, f.y*1.6, 77, 2)-0.5)*0.5
		return pick(icStewR, l)
	})
	p.fill(bowl, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.4) - (f.x-8)*0.03
		if math.Mod(f.x*0.7+f.y*0.2+30, 1.6) < 0.25 {
			l -= 0.12 // the grain of the wood
		}
		return pick(icWoodBowlR, l)
	})
	p.on(sEll(6.0, 8.2, 0.6, 0.35).or(sEll(9.6, 8.6, 0.5, 0.3)), solidC(icGlint))
}
