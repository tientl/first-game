package client

import "testing"

// Every pose keeps its eye, so the eyeshine has somewhere to be.
func TestTroodonPosesHaveAnEye(t *testing.T) {
	for p := range troodonPoses {
		c, ax, ay := troodonCanvas(p)
		ex, ey, ok := eyeOf(c)
		if !ok {
			t.Errorf("pose %d: no eye found", p)
			continue
		}
		if ex <= ax || ey >= ay {
			t.Errorf("pose %d: eye at (%.1f, %.1f), not up and ahead of the feet at (%.1f, %.1f)", p, ex, ey, ax, ay)
		}
	}
}
