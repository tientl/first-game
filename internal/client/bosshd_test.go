package client

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// bhAnims are the animations drawn at HDDensity, by name.
func bhAnims() map[string]*bhAnim {
	return map[string]*bhAnim{
		"titan_slither": titanSlither, "titan_strike": titanStrike, "titan_idle": titanIdle, "titan_roar": titanRoar, "titan_grab": titanGrab,
		"mega_swim": megaSwim, "mega_surface": megaSurface, "mega_lurk": megaLurk, "mega_strike": megaStrike, "mega_roar": megaRoar,
		"mega_breach": megaBreach, "mega_slam": megaSlam,
		"mosa_rise": mosaRise, "mosa_bite": mosaBite, "mosa_back": mosaBack, "mosa_shadow": mosaShadow,
		"kraken_arm": krakenArm, "kraken_head": krakenHead, "kraken_shadow": krakenShadow,
	}
}

// TestHDAnimFrames draws every frame once: none may be empty, and a run's
// anchor must stay on its canvas.
func TestHDAnimFrames(t *testing.T) {
	if testing.Short() {
		t.Skip("draws every frame")
	}
	for name, a := range bhAnims() {
		for i := range a.fr {
			f := a.canvas(i)
			if f.c == nil || f.c.w < 3*HDDensity || f.c.h < 3*HDDensity {
				t.Errorf("%s frame %d is empty", name, i)
			}
		}
	}
}

// TestDumpHDAnim writes the frames of the HD bosses and sea monsters, each
// run on one canvas with the anchors lined up, and says what they cost:
//
//	BOSS_ANIM=dir go test ./internal/client -run DumpHDAnim -v
func TestDumpHDAnim(t *testing.T) {
	dir := os.Getenv("BOSS_ANIM")
	if dir == "" {
		t.Skip("set BOSS_ANIM to a directory")
	}
	only := os.Getenv("BOSS_ONLY")
	var total time.Duration
	bytes := 0
	for name, a := range bhAnims() {
		if only != "" && !(len(name) >= len(only) && name[:len(only)] == only) {
			continue
		}
		start := time.Now()
		l, u, r, d := 0.0, 0.0, 0.0, 0.0
		for i := range a.fr {
			f := a.canvas(i)
			l, u = max(l, f.ax*HDDensity), max(u, f.ay*HDDensity)
			r, d = max(r, float64(f.c.w)-f.ax*HDDensity), max(d, float64(f.c.h)-f.ay*HDDensity)
			bytes += 4 * f.c.w * f.c.h
		}
		el := time.Since(start)
		total += el
		t.Logf("%-14s %2d frames  %4.0f x %-4.0f px  %6.0f ms", name, len(a.fr), l+r, u+d, el.Seconds()*1000)
		bg := color.RGBA{86, 132, 62, 255}
		if name[0] == 'm' || name[0] == 'k' {
			bg = color.RGBA{30, 74, 120, 255}
		}
		for i := range a.fr {
			f := &a.fr[i]
			img := image.NewRGBA(image.Rect(0, 0, int(l+r), int(u+d)))
			draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
			src := &image.NRGBA{Pix: f.c.bytesNRGBA(), Stride: 4 * f.c.w, Rect: image.Rect(0, 0, f.c.w, f.c.h)}
			draw.Draw(img, src.Bounds().Add(image.Pt(int(l-f.ax*HDDensity), int(u-f.ay*HDDensity))), src, image.Point{}, draw.Over)
			w, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s_%02d.png", name, i)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(w, img); err != nil {
				t.Fatal(err)
			}
			w.Close()
		}
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("all frames: %.1f s on one thread, %.0f MB of pixels (as much again for those seen facing left)", total.Seconds(), float64(bytes)/1e6)
}

// bytesNRGBA is the canvas as straight-alpha bytes.
func (c *canvas) bytesNRGBA() []byte {
	b := make([]byte, 4*len(c.px))
	for i, p := range c.px {
		b[4*i], b[4*i+1], b[4*i+2], b[4*i+3] = p.R, p.G, p.B, p.A
	}
	return b
}
