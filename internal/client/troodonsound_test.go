package client

import "testing"

func TestTroodonSoundsAreUsable(t *testing.T) {
	bank := genSFX()
	for _, name := range []string{"troodon_call", "troodon_shriek", "troodon_click"} {
		vs := bank[name]
		if len(vs) == 0 {
			t.Errorf("%s: missing", name)
			continue
		}
		for _, x := range vs {
			checkFinite(t, name, x)
			if peak, rms := stats(x); peak > 1.0001 || peak < 0.3 || rms < 0.003 {
				t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
			}
		}
	}
}
