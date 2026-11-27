package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
)

// The chips for what the beasts' moves leave on the player (sim/status.go),
// beside the poison's and the web's (hud.go): small round chips under the
// vitals, their long lines said once under them as each begins; and their
// pictures, painted like the interface's others (uiicons.go).

var (
	colStun   = color.RGBA{255, 222, 110, 255}
	colBleed  = color.RGBA{236, 70, 64, 255}
	colSlow   = color.RGBA{160, 210, 100, 255}
	colPinned = color.RGBA{236, 150, 70, 255}

	icStarR  = mkRamp(0x6a4a10, 0xb08a20, 0xe8c640, 0xfbe678, 0xfffbd8)
	icShellR = mkRamp(0x3a2a1c, 0x6a4a2c, 0x9c7040, 0xc89a5c, 0xeccf90)
	icSlimeR = mkRamp(0x2c4a1c, 0x4a7a2c, 0x78a844, 0xa8d070, 0xe0f4b0)
	icClawR  = mkRamp(0x2a2420, 0x4a4038, 0x7a6c5c, 0xb4a48c, 0xece0c8)
)

// skillStatuses are the chips for self's statuses from the beasts' moves.
func skillStatuses(self protocol.SelfState) []status {
	var out []status
	if self.Pinned {
		out = append(out, status{statusPinned, "Bị đè", "Bị đè xuống! Bấm đánh liên tục để vùng ra", colPinned})
	}
	if self.Stunned {
		out = append(out, status{statusStun, "Choáng", "Choáng váng — không đi, không đánh được", colStun})
	}
	if self.Bleeding {
		out = append(out, status{statusBleed, "Chảy máu", "Chảy máu! Máu tụt dần một lúc", colBleed})
	}
	if self.Slowed {
		out = append(out, status{statusSlow, "Chậm", "Trúng nọc Troodon — đi chậm lại", colSlow})
	}
	return out
}

// skillStatusIcon is the picture of one of these statuses; ok is false for
// any other.
func skillStatusIcon(k statusKind) (uiIcon, bool) {
	switch k {
	case statusStun:
		return uiStun, true
	case statusBleed:
		return uiBleed, true
	case statusSlow:
		return uiSlow, true
	case statusPinned, statusHeld: // claws, or a squid's arms: held down either way
		return uiPinned, true
	}
	return 0, false
}

// icDazed is three stars wheeling round: stunned.
func icDazed(p *icv) {
	star := func(cx, cy, r float64) shape {
		var pts []float64
		for i := range 8 {
			a := float64(i)*math.Pi/4 - math.Pi/2
			rr := r
			if i%2 == 1 {
				rr = r * 0.42
			}
			pts = append(pts, cx+math.Cos(a)*rr, cy+math.Sin(a)*rr)
		}
		return sPoly(pts...)
	}
	p.fill(sRing(8, 9, 6, 3, 0.45), func(f *ifrag) color.RGBA { return pick(icStarR, 0.35) })
	for _, s := range [][3]float64{{3, 8.5, 3.2}, {12.6, 7.4, 3.6}, {8, 12.4, 2.8}} {
		st := star(s[0], s[1], s[2])
		p.fill(st, func(f *ifrag) color.RGBA { return pick(icStarR, bulge(f, 0.6)+0.15) })
	}
}

// icBloodDrop is a drop of blood.
func icBloodDrop(p *icv) {
	drop := iBlend(2.0, sEll(8, 10.4, 4.4, 4.3), sPoly(8, 1.4, 11.4, 8.6, 4.6, 8.6))
	p.fill(drop, shSphere(icHeartR, 8, 10, 4.4, 5.2, 0.02, true))
}

// icSnail is a snail: slowed.
func icSnail(p *icv) {
	body := iBlend(1.2, sEll(7.5, 12.6, 6.6, 1.8), sEll(13.2, 10.8, 1.5, 2.4))
	p.fill(body, func(f *ifrag) color.RGBA { return pick(icSlimeR, bulge(f, 0.5)+0.1) })
	p.fill(sCap(13.4, 9, 14.6, 5.8, 0.4, 0.3), func(f *ifrag) color.RGBA { return pick(icSlimeR, 0.6) })
	shell := sEll(7, 8.2, 4.9, 4.6)
	p.fill(shell, shSphere(icShellR, 7, 8.2, 4.9, 4.6, 0.05, true))
	p.on(sRing(7.2, 8.4, 2.6, 2.4, 0.45).or(sRing(7.6, 8.7, 1.0, 0.9, 0.4)), solidC(color.RGBA{0x3a, 0x26, 0x18, 0xff}))
}

// icClaws is three hooked claws pressing down: pinned.
func icClaws(p *icv) {
	for i := range 3 {
		x := 3.6 + float64(i)*4.4
		claw := iBlend(0.8, sCap(x-0.6, 2.2, x+0.8, 8.4, 1.3, 1.0), sCap(x+0.8, 8.4, x-0.2, 12.6, 1.0, 0.35))
		p.fill(claw, func(f *ifrag) color.RGBA { return pick(icClawR, bulge(f, 0.5)+0.2) })
	}
	p.fill(sCap(1.6, 14.2, 14.4, 14.2, 0.9, 0.9), func(f *ifrag) color.RGBA { return pick(icShellR, 0.4) })
}
