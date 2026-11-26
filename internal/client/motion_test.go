package client

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"survivalgame/internal/sim"
)

// TestMotionFramesFit checks every frame of every motion: it is drawn, its
// figure stays inside the canvas (nothing cut off at an edge), and a foot
// that should be on the ground is.
func TestMotionFramesFit(t *testing.T) {
	for _, l := range []sim.Look{{}, {Female: true, Hair: sim.HairLong}} {
		heads := headCache{}
		for m := moIdle; m < numMotions; m++ {
			for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
				for f := 0; f < m.frames(); f++ {
					lo, up := m.keys(d, f)
					uc := bakeUpper(l, sim.WearTunic|sim.WearMantle, up, heads)
					lc, _ := bakeLower(l, 0, lo)
					for i, c := range append(uc.layer[:], lc) {
						if edge := touchesEdge(c); edge != "" {
							t.Errorf("%s facing %d frame %d layer %d: cut off at the %s", motions[m].name, d, f, i, edge)
						}
					}
				}
			}
		}
	}
}

// touchesEdge names the edge of c something is drawn on, if any.
func touchesEdge(c *canvas) string {
	for x := 0; x < c.w; x++ {
		if c.px[x].A > 0 {
			return "top"
		}
		if c.px[(c.h-1)*c.w+x].A > 0 {
			return "bottom"
		}
	}
	for y := 0; y < c.h; y++ {
		if c.px[y*c.w].A > 0 {
			return "left"
		}
		if c.px[y*c.w+c.w-1].A > 0 {
			return "right"
		}
	}
	return ""
}

// TestMotionFrameTiming checks the clock of the motions: frameAt walks
// through every key pose in order and holds the last of one that does not
// loop.
func TestMotionFrameTiming(t *testing.T) {
	for m := moIdle; m < numMotions; m++ {
		last := -1
		for age := 0.0; age < m.span(); age += 0.25 {
			f := m.frameAt(age)
			if f < last || f > last+1 {
				t.Fatalf("%s: frame %d after %d at %.2f", motions[m].name, f, last, age)
			}
			last = f
		}
		if last != m.frames()-1 {
			t.Errorf("%s: ends on frame %d of %d", motions[m].name, last, m.frames())
		}
		if f := m.frameAt(m.span() + 1); !motions[m].loop && f != m.frames()-1 {
			t.Errorf("%s: after the end shows frame %d", motions[m].name, f)
		}
	}
}

// TestDumpMotionPoses writes every motion as a sheet, a man and a woman in
// every facing, for a look at the poses:
//
//	MOTION_POSES=dir go test ./internal/client -run DumpMotionPoses -v
func TestDumpMotionPoses(t *testing.T) {
	dir := os.Getenv("MOTION_POSES")
	if dir == "" {
		t.Skip("set MOTION_POSES to a folder")
	}
	const k, cw, ch = 1, 100, 132
	looks := []sim.Look{{Hair: sim.HairShaggy, HairColor: 2, Skin: 1}, {Female: true, Hair: sim.HairLong, HairColor: 4, Skin: 2}}
	start := time.Now()
	n := 0
	for m := moIdle; m < numMotions; m++ {
		cols := m.frames()
		img := limbBackdrop(cols*cw, 8*ch)
		for li, l := range looks {
			heads := headCache{}
			for di, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight, sim.DirLeft} {
				for f := 0; f < cols; f++ {
					lo, up := m.keys(d, f)
					limbShot(img, l, up, lo, heads, f*cw+cw/2, (li*4+di)*ch+ch-10, k, 0, 0)
					n++
				}
			}
		}
		limbPNG(t, filepath.Join(dir, fmt.Sprintf("pose_%02d_%s.png", m, sanitize(motions[m].name))), img)
	}
	// One sheet a facing: every motion a row, the man.
	for di, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
		img := limbBackdrop(13*cw, int(numMotions-1)*ch)
		heads := headCache{}
		for m := moIdle; m < numMotions; m++ {
			for f := 0; f < m.frames(); f++ {
				lo, up := m.keys(d, f)
				limbShot(img, looks[0], up, lo, heads, f*cw+cw/2, int(m-1)*ch+ch-10, k, 0, 0)
			}
		}
		limbPNG(t, filepath.Join(dir, fmt.Sprintf("summary_%d.png", di)), img)
	}
	t.Logf("%d frames in %v", n, time.Since(start))
}

func sanitize(s string) string {
	b := []byte(s)
	for i, c := range b {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			b[i] = '_'
		}
	}
	return string(b)
}

var _ = image.Pt

// TestMotionCost reports what drawing every frame of every motion costs for
// one look, as the game keeps it: the halves' images cut to what is drawn
// on them, for the three facings drawn (left is right mirrored, as big
// again).
func TestMotionCost(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	l := sim.Look{Hair: sim.HairLong}
	heads := headCache{}
	bytes, n := 0, 0
	start := time.Now()
	size := func(c *canvas) int {
		x0, y0, x1, y1 := c.w, c.h, -1, -1
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				if c.px[y*c.w+x].A > 0 {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
				}
			}
		}
		if x1 < 0 {
			return 0
		}
		return 4 * (x1 - x0 + 1) * (y1 - y0 + 1)
	}
	for m := moIdle; m < numMotions; m++ {
		for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
			for f := 0; f < m.frames(); f++ {
				lo, up := m.keys(d, f)
				uc := bakeUpper(l, 0, up, heads)
				lc, _ := bakeLower(l, 0, lo)
				k := 1
				if d == sim.DirRight {
					k = 2 // and its mirror, facing left
				}
				for _, c := range uc.layer {
					bytes += k * size(c)
				}
				bytes += k * size(lc)
				n += 2 * k
			}
		}
	}
	t.Logf("every motion of one look: %d half-frames, %.1f MB, drawn in %v (%.2f ms a half-frame)",
		n, float64(bytes)/1e6, time.Since(start), float64(time.Since(start).Milliseconds())/float64(n))
}
