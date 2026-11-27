package client

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The window and how the game is drawn on it, as the settings ask
// (settings.go): the window mode, the resolution, the world's render
// quality, the frame cap and vsync.
//
// Window modes. Ebitengine's fullscreen keeps the desktop's resolution
// (it never changes the monitor's mode), on Windows and Linux through a
// window made the monitor's own (glfw's SetMonitor). Borderless is a plain
// window without a frame, as big as the monitor and at its corner
// (SetWindowDecorated, SetWindowSize, SetWindowPosition, all Ebitengine's
// own and working on Windows and Linux): it looks the same, but other
// windows can come over it and it stays put when the game loses focus,
// which suits a second monitor. On macOS both are the native fullscreen
// space or close to it.
//
// The screen is worked out in physical pixels: on a monitor scaled for
// high DPI (Windows at 150%, say), Ebitengine's window sizes are in
// device-independent pixels, and the game draws at the device's own pixels
// (Layout multiplies by the scale factor), so 1080p is 1080 real pixels,
// not 720 blown up.
//
// Resolution. In a window it is the window's size. In fullscreen it is the
// size the game is drawn at, which Ebitengine then scales to the monitor:
// a smaller one costs less to draw. The game's art is drawn for 1080p
// (HD_ART.md), so it is sharpest at 1920×1080 and at whole multiples of it
// (3840×2160, each art pixel two screen pixels); the list marks those.
//
// Render quality draws the world at a share of the screen's pixels and
// blows it up (renderScale); the interface is drawn at the screen's own
// resolution all the same, so text stays sharp on a weak machine.

// displayLive is set once the game manages a real window (ApplyDisplay):
// tests and the screenshot tool leave the window alone and draw at the
// size they ask for.
var displayLive bool

// ApplyDisplay puts the window settings in force on the game's window and
// keeps them in force as they change. The game calls it once, before
// ebiten.RunGame.
func ApplyDisplay() {
	displayLive = true
	applyWindow()
}

// resolution is a screen size in physical pixels; 0 by 0 is the monitor's.
type resolution struct{ w, h int }

func (r resolution) native() bool { return r.w <= 0 || r.h <= 0 }

// standardSizes are the resolutions offered, those that fit, 16:9.
var standardSizes = []resolution{
	{1024, 576}, {1152, 648}, {1280, 720}, {1366, 768}, {1600, 900}, {1920, 1080},
	{2560, 1440}, {3200, 1800}, {3840, 2160}, {5120, 2880}, {7680, 4320},
}

// sharpest reports whether the art lands on whole screen pixels at r:
// 1080p and its whole multiples.
func sharpest(r resolution) bool { return r.h > 0 && r.h%1080 == 0 && r.w*9 == r.h*16 }

// monitorInfo is the monitor the window is on: its size in physical
// pixels and its scale factor. Without one (tests), 1920×1080 at 1.
func monitorInfo() (w, h int, scale float64) {
	if m := ebiten.Monitor(); m != nil {
		mw, mh := m.Size()
		s := m.DeviceScaleFactor()
		if s <= 0 {
			s = 1
		}
		if mw > 0 && mh > 0 {
			return int(math.Round(float64(mw) * s)), int(math.Round(float64(mh) * s)), s
		}
	}
	return 1920, 1080, 1
}

// resolutionChoices are the resolutions offered in mode on a monitor mw by
// mh physical pixels at scale factor dsf. In fullscreen the first is the
// monitor's own (0 by 0) and the rest are smaller; in a window they are the
// sizes that fit on the monitor with the window's frame and the taskbar.
func resolutionChoices(mode WindowMode, mw, mh int, dsf float64) []resolution {
	var out []resolution
	if mode == WindowWindowed {
		for _, r := range standardSizes {
			if float64(r.w)/dsf <= float64(mw)/dsf-32 && float64(r.h)/dsf <= float64(mh)/dsf-100 {
				out = append(out, r)
			}
		}
		if len(out) == 0 { // a very small monitor: two thirds of it
			out = append(out, resolution{mw * 2 / 3, mh * 2 / 3})
		}
		return out
	}
	out = append(out, resolution{})
	for _, r := range standardSizes {
		if r.h >= 720 && r.w <= mw && r.h <= mh && (r.w != mw || r.h != mh) {
			out = append(out, r)
		}
	}
	return out
}

// currentResolution is the resolution chosen, as one of choices (the
// nearest below it, if the monitor has changed).
func currentResolution(choices []resolution, s Settings) resolution {
	want := resolution{s.Width, s.Height}
	if want.native() {
		if choices[0].native() {
			return choices[0]
		}
		return choices[len(choices)-1] // a window: the biggest that fits
	}
	best := choices[0]
	for _, r := range choices {
		if r == want {
			return r
		}
		if !r.native() && r.h <= want.h && r.w <= want.w && (best.native() || r.h > best.h) {
			best = r
		}
	}
	return best
}

// resolutionLabel names r on the settings screen, on a monitor mw by mh.
func resolutionLabel(r resolution, mw, mh int) string {
	if r.native() {
		return fmt.Sprintf("%d×%d (màn hình)", mw, mh)
	}
	return fmt.Sprintf("%d×%d", r.w, r.h)
}

// applyWindow puts the window mode, the resolution and vsync in force.
func applyWindow() {
	if !displayLive {
		return
	}
	s := settings
	ebiten.SetVsyncEnabled(s.VSync)
	mw, mh, dsf := monitorInfo()
	dip := func(px int) int { return max(1, int(math.Round(float64(px)/dsf))) }
	switch s.Window {
	case WindowFullscreen:
		ebiten.SetWindowDecorated(true) // for the window it goes back to
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		ebiten.SetFullscreen(true)
	case WindowBorderless:
		ebiten.SetFullscreen(false)
		ebiten.SetWindowDecorated(false)
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
		ebiten.SetWindowSize(dip(mw), dip(mh))
		ebiten.SetWindowPosition(0, 0)
	default:
		r := currentResolution(resolutionChoices(WindowWindowed, mw, mh, dsf), s)
		ebiten.SetFullscreen(false)
		ebiten.SetWindowDecorated(true)
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		w, h := dip(r.w), dip(r.h)
		ebiten.SetWindowSize(w, h)
		// Centred, its title bar on the screen.
		ebiten.SetWindowPosition(max(0, (dip(mw)-w)/2), max(32, (dip(mh)-h)/2-16))
	}
}

// ToggleFullscreen is F11 (and Alt+Enter): from a window to fullscreen,
// from fullscreen or borderless back to a window; kept in the settings.
func ToggleFullscreen() {
	changeSettings(func(s *Settings) {
		if s.Window == WindowWindowed {
			s.Window = WindowFullscreen
		} else {
			s.Window = WindowWindowed
		}
	})
}

// Layout is the screen layout every scene uses: the window's size in
// physical pixels, or in fullscreen the resolution chosen, if smaller.
func Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := outsideWidth, outsideHeight
	if displayLive {
		if m := ebiten.Monitor(); m != nil {
			if s := m.DeviceScaleFactor(); s > 0 {
				w, h = int(math.Round(float64(w)*s)), int(math.Round(float64(h)*s))
			}
		}
		if s := settings; s.Window != WindowWindowed && s.Width > 0 && s.Height > 0 && s.Width < w && s.Height <= h {
			w, h = s.Width, s.Height
		}
	}
	return max(w, 320), max(h, 240)
}

// --- the frame cap ----------------------------------------------------------------

var paceNext time.Time // when the next frame may be drawn

// PaceFrame holds the frame back as long as the frame cap (Settings.FPS)
// asks; the game calls it as it starts drawing. The game's clock is
// Ebitengine's 60 updates a second whatever the cap: at 30 frames a second
// each frame follows two updates.
func PaceFrame() {
	fps := settings.FPS
	if fps <= 0 {
		paceNext = time.Time{}
		return
	}
	period := time.Second / time.Duration(fps)
	now := time.Now()
	if paceNext.IsZero() || now.Sub(paceNext) > period {
		paceNext = now // fallen behind (or just begun): from now
	}
	// A frame within a millisecond of its time goes now, so a cap at the
	// screen's own refresh with vsync on never waits for nothing.
	if d := paceNext.Sub(now); d > time.Millisecond {
		time.Sleep(d)
	}
	paceNext = paceNext.Add(period)
}

// --- render quality ---------------------------------------------------------------

// renderScaleFor is the world's scale, in canvas pixels to an art pixel, at
// quality q (percent) for a world drawn at r on the screen: r itself at
// full quality, a whole number below it otherwise, so the old pixel art
// keeps its grain. Medium is about three quarters of r, low about half and
// below medium where there is room: at 1080p (r 3) 2 and 1, at 1440p 3 and
// 2, at 4K 5 and 3. At 720p (r 2) both are 1.
func renderScaleFor(r, q int) int {
	if r <= 1 || q >= 100 {
		return max(1, r)
	}
	med := max(1, min(r-1, int(math.Round(float64(r)*0.75))))
	if q >= 75 {
		return med
	}
	return max(1, min(max(1, med-1), int(math.Round(float64(r)*0.5))))
}

// renderScale is renderScaleFor at the quality in force.
func renderScale(r int) int { return renderScaleFor(r, settings.Quality) }

// renderSize is the size of the canvas the world is drawn on for a screen
// sw by sh drawn at r, at render scale rr.
func renderSize(sw, sh, r, rr int) (int, int) {
	if rr >= r {
		return sw, sh
	}
	return (sw*rr + r - 1) / r, (sh*rr + r - 1) / r
}

// presentWorld draws the world's canvas on the screen, blown up from render
// scale rr to the world's scale r.
func presentWorld(screen, canvas *ebiten.Image, r, rr int) {
	if rr >= r {
		screen.DrawImage(canvas, nil)
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	k := float64(r) / float64(rr)
	op.GeoM.Scale(k, k)
	screen.DrawImage(canvas, op)
}

// upscaleWorld blows the canvas up from rr to r into *buf, sw by sh, and
// returns it.
func upscaleWorld(buf **ebiten.Image, canvas *ebiten.Image, sw, sh, r, rr int) *ebiten.Image {
	up := sizedImage(buf, sw, sh)
	up.Clear()
	presentWorld(up, canvas, r, rr)
	return up
}

// sizedImage is *img, made anew if it is not w by h.
func sizedImage(img **ebiten.Image, w, h int) *ebiten.Image {
	if *img == nil || (*img).Bounds().Dx() != w || (*img).Bounds().Dy() != h {
		if *img != nil {
			(*img).Deallocate()
		}
		*img = ebiten.NewImage(w, h)
	}
	return *img
}
