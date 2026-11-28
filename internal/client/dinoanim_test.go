package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// animFrames lists the frames of a species' walk, run and blow (the wind-up,
// the lunge and the blow held a while), as rig poses.
func animFrames(species string, hit int) (walk, run, strike []rigPose) {
	for f := range gaitFrames {
		walk = append(walk, stridePose(rigPose{}, f, 1))
		run = append(run, stridePose(rigPose{tilt: 0.06}, f, 1.3))
	}
	fam := poseFamily(species)
	if w, ok := dinoWinds[fam]; ok {
		for age := 0; age < w.frames+lungeFrames; age++ {
			if age < w.frames {
				strike = append(strike, lerpPose(rigPose{}, w.pose, min(1, float64(age+1)/3)))
			} else {
				strike = append(strike, lerpPose(w.pose, dinoPoses[fam][hit], float64(age-w.frames+1)/(lungeFrames+1)))
			}
		}
		for range 8 {
			strike = append(strike, dinoPoses[fam][hit])
		}
	}
	return walk, run, strike
}

func TestStridesAndBlowsAreDrawn(t *testing.T) {
	for species, hit := range map[string]int{"rex": rexBite, "dakota": raptorPounce, "trike": trikeGore, "stego": stegoSwipe, "quetzal": quetzalStab} {
		walk, run, strike := animFrames(species, hit)
		if len(strike) == 0 {
			t.Errorf("%s has no wind-up", species)
		}
		seen := map[string]bool{}
		for _, p := range append(append(walk, run...), strike...) {
			c, ax, ay, ok := rigCanvas(dinoRigs[species], p)
			if !ok {
				t.Fatalf("%s: drawing missing", species)
			}
			if ax < 0 || ay < 0 || ax > float64(c.w) || ay > float64(c.h)+3 {
				t.Errorf("%s %+v: anchor (%.0f, %.0f) off its %dx%d canvas", species, p, ax, ay, c.w, c.h)
			}
			seen[fmt.Sprint(c.w, c.h, ax, ay, len(c.px), c.px[len(c.px)/2:len(c.px)/2+c.w])] = true
		}
		if len(seen) < gaitFrames {
			t.Errorf("%s: only %d different frames in its walk, run and blow", species, len(seen))
		}
	}
}

// TestDumpDinoAnim writes the frames of each drawn species' walk, run and
// blow to the folder $DINO_ANIM, as <species>_<walk|run|strike>_<n>.png on
// one frame size per species, and logs what drawing them all costs.
func TestDumpDinoAnim(t *testing.T) {
	dir := os.Getenv("DINO_ANIM")
	if dir == "" {
		t.Skip("set DINO_ANIM to a folder")
	}
	hits := map[string]int{"rex": rexBite, "dakota": raptorPounce, "achero": raptorPounce, "trike": trikeGore, "toro": trikeGore,
		"stego": stegoSwipe, "brachio": 0, "ornitho": 0, "thesc": 0, "anzu": anzuKick, "quetzal": quetzalStab, "troodon": 0, "plesio": plesioStrike}
	for species, hit := range hits {
		walk, run, strike := animFrames(species, hit)
		var took time.Duration
		n, bytes := 0, 0
		for name, poses := range map[string][]rigPose{"walk": walk, "run": run, "strike": strike} {
			for i, p := range poses {
				start := time.Now()
				c, ax, ay, _ := rigCanvas(dinoRigs[species], p)
				took += time.Since(start)
				n++
				bytes += 4 * c.w * c.h
				_, bw, bh, _ := rigAsDrawn(dinoRigs[species])
				k := max(1, c.dens)
				w, h := (bw+60)*k, (bh+40)*k
				img := image.NewNRGBA(image.Rect(0, 0, w, h))
				for j := range img.Pix {
					img.Pix[j] = []byte{88, 128, 70, 255}[j%4]
				}
				ox, oy := w/2-int(ax), h-8*k-int(ay)
				for y := 0; y < c.h; y++ {
					for x := 0; x < c.w; x++ {
						if q := c.px[y*c.w+x]; q.A >= 128 && image.Pt(x+ox, y+oy).In(img.Rect) {
							img.SetNRGBA(x+ox, y+oy, color.NRGBA{q.R, q.G, q.B, 255})
						}
					}
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s_%s_%02d.png", species, name, i)))
				if err != nil {
					t.Fatal(err)
				}
				png.Encode(f, img)
				f.Close()
			}
		}
		t.Logf("%-8s %2d frames, %6.1f ms each to draw, %5.1f MB of pixels for both sides", species, n,
			float64(took.Microseconds())/1000/float64(n), float64(2*bytes)/(1<<20))
	}
}

// TestStanceFeetStayPlanted walks and runs every jointed rig through a
// stride: a foot on the ground stays on it, moving back under the body at a
// steady speed, and every knee bends forward, every elbow back.
func TestStanceFeetStayPlanted(t *testing.T) {
	for species, r := range dinoRigs {
		if !r.jointed() {
			continue
		}
		rigCanvas(r, rigPose{step: 1}) // cuts its legs
		r, _, _, _ = rigAsDrawn(r)     // grown with its painting
		k := 1
		if dinoHD {
			k = HDDensity
		}
		kf := float64(k)
		cuts := legCuts[legKey{r.file, k}]
		if cuts == nil {
			t.Fatalf("%s: no legs cut", species)
		}
		for _, amp := range []float64{1, 1.3} {
			const steps = 64
			_, duty, _ := gaitTiming(r, amp)
			want := 2 * amp * r.stride * gaitReach * kf / steps / duty
			last := make([]float64, len(cuts))
			was := make([]bool, len(cuts))
			for s := 1; s <= 2*steps; s++ {
				p := rigPose{step: amp, cycle: float64((s-1)%steps+1) / steps}
				feet, drop := legPoses(r, p, cuts, kf)
				for i := range cuts {
					c := &cuts[i]
					J, _ := c.solve(feet[i], drop)
					n := len(J) - 1
					cross := (J[1].x-J[0].x)*(J[2].y-J[0].y) - (J[1].y-J[0].y)*(J[2].x-J[0].x)
					if math.Abs(cross) > 0.01 && (cross > 0) == c.fore {
						t.Errorf("%s leg %d at %.2f: the knee bends the wrong way", species, i, p.cycle)
					}
					if !feet[i].stance {
						was[i] = false
						continue
					}
					if off := math.Abs(J[n].y - c.P[n].y); off > kf {
						t.Errorf("%s x%.1f leg %d at %.2f: its foot is %.1f px off the ground", species, amp, i, p.cycle, off/kf)
					}
					if got := last[i] - J[n].x; was[i] && math.Abs(got-want) > 0.4*kf {
						t.Errorf("%s x%.1f leg %d at %.2f: its foot slid %.2f px, not %.2f", species, amp, i, p.cycle, got/kf, want/kf)
					}
					last[i], was[i] = J[n].x, true
				}
			}
		}
	}
}
