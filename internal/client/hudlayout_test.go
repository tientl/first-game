package client

import (
	"fmt"
	"testing"
)

// The HUD's pieces, each at its biggest, keep clear of one another and on
// the screen at every common screen size and every UI size the settings
// allow there; the corners keep their margin.
func TestHUDLayoutKeepsClear(t *testing.T) {
	screens := [][2]int{
		{1280, 720}, {1366, 768}, {1600, 900}, {1920, 1080}, {2560, 1440}, {3840, 2160},
		{1280, 800}, {1920, 1200}, {1024, 768}, {1280, 1024}, {2560, 1080}, {3440, 1440},
	}
	for _, s := range screens {
		w, h := s[0], s[1]
		scales := []float64{autoUIScale(h), maxUIScale(h)}
		for k := 1.0; k <= maxUIScale(h); k += 0.25 {
			scales = append(scales, k)
		}
		for _, k := range scales {
			uw, uh := uiSize(w, h, k)
			where := fmt.Sprintf("%dx%d at %.0f%% (%dx%d UI pixels)", w, h, k*100, uw, uh)
			rs := hudLayout(uw, uh)
			for i, a := range rs {
				if a.x < hudM/2 || a.y < 0 || a.x+a.w > float64(uw)-hudM/2 || a.y+a.h > float64(uh) {
					t.Errorf("%s: the %s (%.0f, %.0f) %.0fx%.0f is off the screen", where, a.name, a.x, a.y, a.w, a.h)
				}
				for _, b := range rs[i+1:] {
					if a.overlaps(b) {
						t.Errorf("%s: the %s (%.0f, %.0f) %.0fx%.0f runs into the %s (%.0f, %.0f) %.0fx%.0f",
							where, a.name, a.x, a.y, a.w, a.h, b.name, b.x, b.y, b.w, b.h)
					}
				}
			}
			if mx, _ := miniCentre(uw); float64(uw)-(mx+cornerW/2) != hudM {
				t.Errorf("%s: the minimap is %.0f from the right edge, want %d", where, float64(uw)-(mx+cornerW/2), hudM)
			}
		}
	}
	// And the pieces are as small as they were meant to be: about 60% of the
	// old 230x70 vitals and 200x211 corner.
	if vitalPanelW > 0.66*230 || vitalsBottom(2)-vitalY > 0.66*70 {
		t.Errorf("the vitals are %dx%.0f", vitalPanelW, vitalsBottom(2)-vitalY)
	}
	if cornerW > 0.66*200 {
		t.Errorf("the minimap's corner is %d across", cornerW)
	}
}

// A status that comes and goes at once (a cold at the edge of a fire's
// warmth) says its long line the first time only; one gone a long while
// says it again.
func TestStatusLongLineOnce(t *testing.T) {
	g := &Game{}
	cold := []status{{kind: statusCold, label: "Lạnh", long: "Lạnh cóng!"}}
	shouting := func() bool { return g.frame-g.statusAt[statusCold] < statusShout }
	g.frame = 100
	g.noteStatuses(cold)
	if !shouting() {
		t.Fatal("a status begun is not said")
	}
	g.frame += statusShout + 10
	g.noteStatuses(cold)
	if shouting() {
		t.Fatal("still said after its time")
	}
	g.frame += 30
	g.noteStatuses(nil) // gone a moment
	g.frame += 30
	g.noteStatuses(cold)
	if shouting() {
		t.Error("a status back after a moment is said again")
	}
	g.noteStatuses(nil)
	g.frame += statusQuiet + 1
	g.noteStatuses(cold)
	if !shouting() {
		t.Error("a status back after a long while is not said")
	}
}

// The notices: three at most, the newest on top, the same one again kept
// rather than stacked.
func TestNoticesStack(t *testing.T) {
	g := &Game{}
	g.ShowNotices("một", "hai", "ba", "bốn")
	g.takeNotices(g.hudFX.notes.list[0].born)
	l := g.hudFX.notes.list
	if len(l) != maxNotes || l[0].text != "bốn" || l[2].text != "hai" {
		t.Fatalf("notices %+v", l)
	}
	n := len(l)
	g.showNotice("bốn")
	g.takeNotices(l[0].born)
	if len(g.hudFX.notes.list) != n {
		t.Errorf("the same notice again made %d", len(g.hudFX.notes.list))
	}
}
