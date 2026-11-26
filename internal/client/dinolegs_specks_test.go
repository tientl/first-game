package client

import "testing"

// TestStridesLeaveNoSpecks: no frame of a walking beast's walk, run or
// sprint leaves bits of a turned leg about it: no loose specks, and few
// hairlines (a pixel with one solid neighbour of four, or none) below the
// hips, where a shred of thigh, a feather tip or a joint's filling turned
// the wrong way would hang. The claws' and feathers' own tips are a few
// each frame; the fuzzy one's tufts, more.
func TestStridesLeaveNoSpecks(t *testing.T) {
	limit := map[string]float64{"rex": 14, "trike": 14, "toro": 14, "stego": 14, "brachio": 14, "ankylo": 14, "thesc": 40}
	for _, sp := range []string{"dakota", "achero", "troodon", "ornitho", "anzu", "rex", "thesc", "trike", "toro", "stego", "brachio", "ankylo"} {
		r := dinoRigs[sp]
		hair, frames := 0, 0
		for _, amp := range []float64{1, 1.25, 1.4} {
			for f := range gaitFrames {
				c, _, _, ok := rigCanvas(r, stridePose(rigPose{}, f, amp))
				if !ok {
					t.Fatalf("%s: no drawing", sp)
				}
				if n := looseBits(c, 3*max(1, c.dens)); n > 0 {
					t.Errorf("%s, stride %.2f frame %d: %d loose bits of pixels", sp, amp, f, n)
				}
				hair += hairlines(c, (r.legs[0].top-1)*max(1, c.dens))
				frames++
			}
		}
		most := limit[sp]
		if most == 0 {
			most = 10
		}
		if per := float64(hair) / float64(frames); per > most {
			t.Errorf("%s: %.1f hairline pixels below the hips a frame (at most %g)", sp, per, most)
		}
	}
}

// hairlines counts the solid pixels of c from row y0 down with at most one
// solid neighbour of the four.
func hairlines(c *canvas, y0 int) int {
	n := 0
	solid := func(x, y int) bool { return x >= 0 && y >= 0 && x < c.w && y < c.h && c.px[y*c.w+x].A >= 128 }
	for y := max(0, y0); y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if !solid(x, y) {
				continue
			}
			k := 0
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if solid(x+d[0], y+d[1]) {
					k++
				}
			}
			if k <= 1 {
				n++
			}
		}
	}
	return n
}

// looseBits counts the pieces of c's solid pixels (8-connected) of at most
// n pixels.
func looseBits(c *canvas, n int) int {
	seen := make([]bool, len(c.px))
	count := 0
	var stack []int
	for i := range c.px {
		if seen[i] || c.px[i].A < 128 {
			continue
		}
		size := 0
		stack = append(stack[:0], i)
		seen[i] = true
		for len(stack) > 0 {
			q := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++
			x, y := q%c.w, q/c.w
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					X, Y := x+dx, y+dy
					if X < 0 || Y < 0 || X >= c.w || Y >= c.h {
						continue
					}
					if j := Y*c.w + X; !seen[j] && c.px[j].A >= 128 {
						seen[j] = true
						stack = append(stack, j)
					}
				}
			}
		}
		if size <= n {
			count++
		}
	}
	return count
}
