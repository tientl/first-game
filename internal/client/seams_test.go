package client

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The ground is drawn in chunks laid edge to edge (ground.go). Drawn at a
// scale that blew each chunk's image up by a fraction (1080p zoomed in once,
// 1440p: four canvas pixels to an art pixel, the HD art at 4/3), the smooth
// samples along every chunk's edge took in the transparent border round its
// image, and a thin pale line showed along each edge: a grid over the
// beach. These tests draw ground of one even colour across a corner where
// four chunks meet, through the world pass as the game draws it, and read
// the screen back: every pixel must be that colour.
//
// Reading pixels back needs Ebitengine's loop, which runs once in a
// process, so the checks run in a process of their own, as the interface's
// do (uiscreens_test.go); without a display they are skipped.

const seamChildEnv = "CHUNK_SEAMS_CHILD"

// The world is drawn at a scale the HD art lands whole on, or smaller than
// it was painted: never blown up by a fraction.
func TestCrispScale(t *testing.T) {
	for rr, want := range []int{1: 1, 2: 2, 3: 3, 4: 3, 5: 3, 6: 6, 7: 6, 8: 6, 9: 9, 10: 9} {
		if rr == 0 {
			continue
		}
		if got := crispScale(rr); got != want {
			t.Errorf("crispScale(%d) = %d, want %d", rr, got, want)
		}
	}
	old := settings.Quality
	defer func() { settings.Quality = old }()
	settings.Quality = 100
	if renderScale(3) != 3 {
		t.Error("1080p at full quality must be drawn straight on the screen, as before")
	}
	if renderScale(4) != 3 {
		t.Error("1080p zoomed in once (and 1440p) must be drawn at 3 and blown up")
	}
}

func TestNoChunkSeams(t *testing.T) {
	if os.Getenv(seamChildEnv) == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestNoChunkSeams$", "-test.v", "-test.count=1")
		cmd.Env = append(os.Environ(), seamChildEnv+"=1")
		out, err := cmd.CombinedOutput()
		if strings.Contains(string(out), "--- SKIP") {
			t.Skip(string(out))
		}
		if err != nil {
			t.Fatalf("the seams, in their own process: %v\n%s", err, out)
		}
		return
	}
	inGameLoop(t, func() {
		seamsAtEveryScale(t)
		seamsDrawnDirect(t)
	})
}

// seamGrounds are the chunks' looks: sand, solid; and the sea, which the
// bake leaves see-through over the deep blue floor.
var seamGrounds = []struct {
	name string
	c    color.RGBA // premultiplied
}{
	{"sand", color.RGBA{0xe0, 0xc5, 0x8c, 0xff}},
	{"sea", color.RGBA{0x4c, 0x7e, 0x7c, 0x99}},
}

// seamsAtEveryScale draws the world as Draw does at every world scale and
// render quality: on a canvas at the render scale, blown up to the screen.
func seamsAtEveryScale(t *testing.T) {
	old := settings.Quality
	defer func() { settings.Quality = old }()
	for r := 1; r <= 9; r++ {
		for _, q := range []int{100, 75, 50} {
			settings.Quality = q
			rr := renderScale(r)
			if rr > HDDensity && rr%HDDensity != 0 {
				t.Errorf("scale %d at %d%%: the world is drawn at %d, which blows the HD art up by a fraction", r, q, rr)
			}
			for sub := 0; sub < rr; sub++ {
				checkSeams(t, fmt.Sprintf("scale %d at %d%% (drawn at %d), camera %d/%d of an art pixel on", r, q, rr, sub, rr), r, rr, sub)
			}
		}
	}
}

// seamsDrawnDirect draws the world straight at scales the game no longer
// draws it at, the HD art blown up by a fraction: the chunks must still
// meet without a line (drawChunkImage).
func seamsDrawnDirect(t *testing.T) {
	for _, r := range []int{4, 5, 7, 8} {
		checkSeams(t, fmt.Sprintf("drawn straight at %d", r), r, r, 1)
	}
}

// checkSeams draws ground of each look across a corner of four chunks,
// with the world at scale r on a 480 by 270 screen, drawn at render scale
// rr with the camera sub/rr of an art pixel past a whole one, and reports
// any pixel not of the ground's colour.
func checkSeams(t *testing.T, where string, r, rr, sub int) {
	t.Helper()
	const sw, sh = 480, 270
	cw, ch := (sw+r-1)/r, (sh+r-1)/r
	// The corner of chunk (1, 1) in the middle of the view.
	cam := camera{ox: float64(chunkW-cw/2) + float64(sub)/float64(rr), oy: float64(chunkH-ch/2) + float64(sub)/float64(rr), r: float64(rr)}
	screen := ebiten.NewImage(sw, sh)
	defer screen.Deallocate()
	canvas := screen
	if rr < r {
		w, h := renderSize(sw, sh, r, rr)
		canvas = ebiten.NewImage(w, h)
		defer canvas.Deallocate()
	}
	chunk := ebiten.NewImage(chunkPW, chunkPH)
	defer chunk.Deallocate()
	buf := make([]byte, 4*sw*sh)
	for _, gr := range seamGrounds {
		screen.Fill(color.RGBA{14, 18, 28, 255})
		canvas.Fill(color.RGBA{14, 18, 28, 255})
		chunk.Fill(gr.c)
		wd := &worldDst{}
		wd.begin(canvas, rr, cw, ch, cam)
		fillRect(wd, -1, -1, float32(cw)+2, float32(ch)+2, colWaterBG, false) // the sea's floor, as drawWater has it
		for cy := -1; cy <= 3; cy++ {
			for cx := -1; cx <= 3; cx++ {
				drawChunkImage(wd, chunk, float64(cx*chunkW)-cam.ox, float64(cy*chunkH)-cam.oy, ebiten.ColorScale{})
			}
		}
		wd.end()
		if canvas != screen {
			presentWorld(screen, canvas, r, rr)
		}
		screen.ReadPixels(buf)
		mid := 4 * (sh/4*sw + sw/4)
		want := [3]byte{buf[mid], buf[mid+1], buf[mid+2]}
		bad, first := 0, ""
		cols := map[int]bool{}
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				i := 4 * (y*sw + x)
				for c := 0; c < 3; c++ {
					if d := int(buf[i+c]) - int(want[c]); d > 2 || d < -2 {
						if bad == 0 {
							first = fmt.Sprintf("(%d, %d) is %v, not %v", x, y, buf[i:i+3], want)
						}
						bad++
						cols[x] = true
						break
					}
				}
			}
		}
		if bad > 0 {
			t.Errorf("%s, %s: %d pixels off the ground's colour in %d columns; %s", where, gr.name, bad, len(cols), first)
		}
	}
}
