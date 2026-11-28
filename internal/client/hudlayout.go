package client

import "survivalgame/internal/sim"

// The HUD's layout, in UI pixels: small pieces in the corners and the world
// left free between them. Every piece's place is worked out here, so the
// drawing (hud.go, uimap.go, bossui.go, hudtoasts.go) and the tests agree.
//
//	top left    the vitals: thin bars on a little hide; the status chips
//	            under them; for a moment after a status begins, its long
//	            line, under the chips
//	top right   the round minimap in the ring of the day; under it the day,
//	            the season and the weather, and the level underground
//	top middle  a boss's bar; the notices under it, three at most
//	bottom      the hotbar, and one line over it: the held item's name for
//	            a moment after it changes, else the one hint that matters

const (
	hudM = 12 // the margin from the screen's edges

	vitalX, vitalY = hudM, hudM // the vitals panel's corner
	vitalPad       = 6          // round the bars
	vitalIcon      = 15         // a bar's picture, across
	vitalBarW      = 113
	vitalBarH      = 11
	vitalRow       = 15 // from one bar to the next
	vitalPanelW    = vitalPad + vitalIcon + 4 + vitalBarW + vitalPad
	maxVitals      = 4 // health, hunger, a swimmer's strength, the boat

	chipD   = 20 // a status chip, across
	chipGap = 4

	statusToastW = 210 // the widest the long line of a status gets

	miniD    = 112 // the round minimap, across
	miniRing = 8   // the ring of the day round it
	cornerW  = miniD + 2*miniRing
	infoH    = 17 // the line of the day, season and weather under it

	noticeH   = 20 // a notice of one line
	noticeGap = 4
	maxNotes  = 3 // notices shown at once

	lineH = 18 // the one line over the hotbar
)

// vitalsBottom is the y just under the vitals panel with n bars.
func vitalsBottom(n int) float64 {
	return vitalY + 2*vitalPad + float64(n*vitalRow-(vitalRow-vitalBarH))
}

// chipsPerRow is how many status chips go in a row: as wide as the vitals.
func chipsPerRow() int { return (vitalPanelW + chipGap) / (chipD + chipGap) }

// chipsTop is the top of the status chips under n vitals.
func chipsTop(n int) float64 { return vitalsBottom(n) + 5 }

// chipsBottom is the y just under nc status chips under nv vitals (under
// the vitals, if there are none).
func chipsBottom(nv, nc int) float64 {
	if nc <= 0 {
		return vitalsBottom(nv)
	}
	rows := (nc + chipsPerRow() - 1) / chipsPerRow()
	return chipsTop(nv) + float64(rows*(chipD+chipGap)-chipGap)
}

// miniCentre is the middle of the round minimap on a canvas uw wide.
func miniCentre(uw int) (x, y float64) {
	return float64(uw) - hudM - cornerW/2, hudM + cornerW/2
}

// cornerLeft is the left edge of the top right corner.
func cornerLeft(uw int) float64 { return float64(uw) - hudM - cornerW }

// infoTop is the top of the line under the minimap; with no minimap (a
// window is open) it goes up into the corner.
func infoTop(withMap bool) float64 {
	if withMap {
		return hudM + cornerW + 4
	}
	return hudM
}

// cornerBottom is the y just below the top right corner: the minimap, the
// line under it, and the level underground.
func cornerBottom(withMap, cave bool) float64 {
	y := infoTop(withMap) + infoH
	if cave {
		y += 3 + caveLineH
	}
	return y
}

const caveLineH = 15

// The boss's bar, top middle: a slim slab of stone between the vitals and
// the minimap, never over either.
const (
	bossTop   = 6
	bossSlabH = 40
	bossTusk  = 30 // room the tusks curl out into, beyond the bar's ends
)

// bossBarW is the width of the boss's bar on a canvas uw wide.
func bossBarW(uw int) float64 {
	side := float64(hudM + max(vitalPanelW, cornerW) + 8 + bossTusk)
	return max(120, min(400, float64(uw)-2*side))
}

// bossBottom is the y just under the boss's bar, with its line about being
// held if holds.
func bossBottom(holds bool) float64 {
	y := float64(bossTop + bossSlabH)
	if holds {
		y += 4 + noticeH
	}
	return y
}

// noticesTop is the top of the notices: under the boss's bar if one is up.
func noticesTop(boss, holds bool) float64 {
	if boss {
		return bossBottom(holds) + 6
	}
	return hudM
}

// noticeMaxW is the widest a notice gets on a canvas uw wide (a longer one
// takes two lines): clear of the long line of a status on the left (and so
// of the minimap on the right).
func noticeMaxW(uw int) float64 {
	side := float64(hudM+statusToastW) + 10
	side = max(side, float64(hudM+cornerW)+10)
	return max(120, min(420, float64(uw)-2*side))
}

// lineTop is the top of the line over the hotbar on a canvas sh tall.
func lineTop(sh int) float64 { return float64(hotbarY(sh)) - 10 - 6 - lineH }

// hudRect is a piece of the HUD's layout at its biggest, for the tests.
type hudRect struct {
	name       string
	x, y, w, h float64
}

func (r hudRect) overlaps(o hudRect) bool {
	return r.x < o.x+o.w && o.x < r.x+r.w && r.y < o.y+o.h && o.y < r.y+r.h
}

// hudLayout is every piece of the HUD on a canvas uw by uh at its biggest:
// all the vitals, three rows of chips (every status at once), a long line
// of three lines, a boss holding the player, three notices of two lines,
// and a long hint.
func hudLayout(uw, uh int) []hudRect {
	vb := vitalsBottom(maxVitals)
	cb := chipsBottom(maxVitals, 3*chipsPerRow())
	sLine := lineUI * statusToastScale
	nLine := lineUI * noticeScale
	bw := bossBarW(uw)
	nw := noticeMaxW(uw)
	hotW := float64(sim.HotbarSlots*hotPx+(sim.HotbarSlots-1)*hotGap) + 20
	mx, my := miniCentre(uw)
	return []hudRect{
		{"vitals", vitalX, vitalY, vitalPanelW, vb - vitalY},
		{"chips", vitalX, chipsTop(maxVitals), vitalPanelW, cb - chipsTop(maxVitals)},
		{"status line", vitalX, cb + 4, statusToastW, 3*sLine + 6},
		{"tutorial", vitalX, trackTop(maxVitals, 3*chipsPerRow(), 3*sLine+6), trackW, trackMaxH},
		{"minimap", mx - cornerW/2, my - cornerW/2, cornerW, cornerW},
		{"day line", cornerLeft(uw), infoTop(true), cornerW, cornerBottom(true, true) - infoTop(true)},
		{"boss", (float64(uw)-bw)/2 - bossTusk, bossTop, bw + 2*bossTusk, bossBottom(true) - bossTop},
		{"notices", (float64(uw) - nw) / 2, noticesTop(true, true), nw, maxNotes*(2*nLine+6) + (maxNotes-1)*noticeGap},
		{"hint", float64(uw)/2 - 200, lineTop(uh), 400, lineH},
		{"hotbar", float64(hotbarX0(uw)) - 10, float64(hotbarY(uh)) - 10, hotW, hotPx + 20},
	}
}
