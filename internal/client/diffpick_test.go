package client

import (
	"strings"
	"testing"

	"survivalgame/internal/sim"
)

// TestDiffPickerKeys checks how the keys move about the difficulty picker:
// left and right along the row of cards, up and down between the row and
// "Quay lại", and back to the card it left.
func TestDiffPickerKeys(t *testing.T) {
	var p diffPicker
	steps := []struct {
		name string
		k    diffKeys
		want int
	}{
		{"right", diffKeys{right: true}, 1},
		{"right again", diffKeys{right: true}, 2},
		{"right wraps", diffKeys{right: true}, 0},
		{"left wraps", diffKeys{left: true}, 2},
		{"left", diffKeys{left: true}, 1},
		{"down to back", diffKeys{down: true}, diffBack},
		{"left on back stays", diffKeys{left: true}, diffBack},
		{"right on back stays", diffKeys{right: true}, diffBack},
		{"up returns to the card", diffKeys{up: true}, 1},
		{"up to back", diffKeys{up: true}, diffBack},
		{"down returns to the card", diffKeys{down: true}, 1},
		{"tab", diffKeys{tab: true}, 2},
		{"tab to back", diffKeys{tab: true}, diffBack},
		{"tab wraps", diffKeys{tab: true}, 0},
		{"nothing", diffKeys{}, 0},
	}
	for _, s := range steps {
		if p.press(s.k) {
			t.Errorf("%s: picked without Enter", s.name)
		}
		if p.sel != s.want {
			t.Fatalf("%s: selection %d, want %d", s.name, p.sel, s.want)
		}
	}
	p.sel = 2
	if !p.press(diffKeys{ok: true}) || p.sel != 2 {
		t.Errorf("Enter on the third card: selection %d", p.sel)
	}
}

// TestDiffPickerLayout checks that the cards and the button lie inside the
// panel without overlapping, that a point in each finds it, and that the
// panel fits windows down to the smallest.
func TestDiffPickerLayout(t *testing.T) {
	if len(difficulties) != diffBack {
		t.Fatalf("%d cards for %d difficulties", diffBack, len(difficulties))
	}
	for i := 0; i <= diffBack; i++ {
		r := diffBackRect()
		if i < diffBack {
			r = diffCardRect(i)
		}
		if r.Min.X < diffPad || r.Max.X > diffW-diffPad || r.Min.Y < 0 || r.Max.Y > diffH {
			t.Errorf("box %d at %v is outside the %dx%d panel", i, r, diffW, diffH)
		}
		c := r.Min.Add(r.Max).Div(2)
		if got := diffHit(float64(c.X), float64(c.Y)); got != i {
			t.Errorf("the middle of box %d hits %d", i, got)
		}
		if i > 0 && i < diffBack && r.Overlaps(diffCardRect(i-1)) {
			t.Errorf("cards %d and %d overlap", i-1, i)
		}
	}
	if got := diffHit(float64(diffCardRect(0).Max.X)+diffGap/2, diffCardY+10); got != -1 {
		t.Errorf("the gap between cards hits %d", got)
	}
	for _, sz := range [][2]int{{1920, 1080}, {1280, 720}, {800, 600}, {640, 360}, {320, 240}} {
		s, ox, oy := diffFit(sz[0], sz[1])
		if s <= 0 || s > 1 || ox < 0 || oy < 0 || ox+diffW*s > float64(sz[0]) || oy+diffH*s > float64(sz[1]) {
			t.Errorf("at %dx%d the panel is at (%v, %v) scaled %v", sz[0], sz[1], ox, oy, s)
		}
		if sz[0] >= 800 && sz[1] >= 600 && s != 1 {
			t.Errorf("at %dx%d the panel is scaled %v, want 1", sz[0], sz[1], s)
		}
	}
}

// TestDiffCardText checks the cards against the rules they describe.
func TestDiffCardText(t *testing.T) {
	for i, d := range difficulties {
		if got := strings.Contains(diffDeath(d), "không mất"); got != d.KeepsItems() {
			t.Errorf("%s: the card says %q but KeepsItems is %v", d.Name(), diffDeath(d), d.KeepsItems())
		}
		for j, s := range diffCards[i].stats {
			if s == "" {
				t.Errorf("%s: no figure for %s", d.Name(), diffStatNames[j])
			}
		}
	}
	if diffCards[sim.DiffBasic].stats != [4]string{"x1", "x1", "x1", "2"} || sim.TroodonPacksNear != 2 {
		t.Errorf("the basic card is %v with %d packs", diffCards[sim.DiffBasic].stats, sim.TroodonPacksNear)
	}
}
