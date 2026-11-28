package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
)

// The player's settings: sound, the window and how the game is drawn, kept
// in settings.json next to the save (in the user's config directory) and
// changed on the settings screen (settingsui.go). Each change takes effect
// at once and is saved at once.
//
// Two copies are kept: what the file holds (settingsSaved) and what is in
// force (settings), which is the file with the command line's choices on
// top (OverrideSettings). A change on the settings screen goes into both,
// so a choice made with a flag lasts the session and is never saved unless
// the player changes it in the menu.
//
// A file from an older game (music, effects, zoom, ground) loads: what it
// lacks keeps its default, and the creatures' voices, which used to play
// with the effects, start at the effects' volume.

// WindowMode is how the game takes the screen.
type WindowMode string

const (
	WindowFullscreen WindowMode = "fullscreen" // Ebitengine's fullscreen (it keeps the desktop's resolution)
	WindowBorderless WindowMode = "borderless" // a window without a frame, as big as the monitor
	WindowWindowed   WindowMode = "windowed"   // a window with a frame
)

// Settings are the player's choices.
type Settings struct {
	// Volumes, 0 to 1: the whole, and each kind of sound under it.
	Master  float64 `json:"master"`
	Music   float64 `json:"music"`
	Effects float64 `json:"effects"`
	Voices  float64 `json:"voices"` // the creatures' voices
	// MuteUnfocused silences the game while its window is not in front.
	MuteUnfocused bool `json:"muteUnfocused"`

	// Window is how the game takes the screen; see display.go.
	Window WindowMode `json:"window"`
	// Width and Height are the resolution: the window's size in a window,
	// the size the game is drawn at in fullscreen. 0 is the monitor's own.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// Quality is the share of the screen's pixels the world is drawn with,
	// in percent (100, 75, 50), before it is blown up to the screen; the
	// interface is always drawn at the screen's own resolution.
	Quality int `json:"quality"`
	// UIScale is the interface's size in percent of one screen pixel to a
	// UI pixel (100, 125, 150...), 0 for automatic: see uiscale.go.
	UIScale int `json:"uiScale"`
	// FPS caps the frames drawn a second, 0 for no cap; VSync waits for the
	// screen's refresh.
	FPS   int  `json:"fps"`
	VSync bool `json:"vsync"`

	Zoom    int    `json:"zoom,omitempty"`    // the view's zoom, minZoom to maxZoom; see render.go
	NoShake bool   `json:"noshake,omitempty"` // screen shake turned off; see feel.go
	PostFX  bool   `json:"postfx"`            // the lighting shaders: glow and grading; see postfx.go
	Grade   string `json:"grade,omitempty"`   // a colour grade of the world: "" (the game's own) or "comfy"
	// Brightness lifts the dark of the night and of the caves for a dim
	// screen: 0 as the game is, 1 brighter, 2 brightest. Even at 2 a cave
	// away from any light is too dark to find the way without a torch; see
	// brightnessLevels.
	Brightness int `json:"brightness,omitempty"`
}

// brightnessLevels is what each choice of Settings.Brightness does: how
// much it raises the dark of the natural caves (cavedark.go), and how much
// of the night's dark it keeps (lighting.go).
var brightnessLevels = [...]struct{ cave, night float32 }{
	{1, 1},
	{1.35, 0.88},
	{1.7, 0.76},
}

// brightnessSetting is the brightness setting, within its choices.
func brightnessSetting() int { return max(0, min(len(brightnessLevels)-1, settings.Brightness)) }

// The choices the settings screen offers.
var (
	qualityChoices = []int{100, 75, 50}
	uiScaleChoices = []int{0, 100, 125, 150, 200, 250, 300, 400}
	fpsChoices     = []int{30, 60, 120, 0}
)

// DefaultSettings are the settings of a first run: fullscreen at the
// monitor's resolution, drawn at full quality, the interface sized for the
// screen, in step with its refresh.
func DefaultSettings() Settings {
	return Settings{
		Master: 1, Music: 0.6, Effects: 0.8, Voices: 0.8,
		Window: WindowFullscreen, Quality: 100, FPS: 0, VSync: true,
	}
}

var (
	settings      = DefaultSettings() // in force
	settingsSaved = DefaultSettings() // as in the file
	settingsPath  string              // where they are kept; "" keeps them nowhere (tests, the screenshot tool)
)

// CurrentSettings are the settings in force.
func CurrentSettings() Settings { return settings }

// LoadSettings reads the settings kept at path (none there: the defaults)
// and puts them in force; later changes are saved there. A file that will
// not read leaves the defaults in force, and the error says why. It does
// not touch the window: ApplyDisplay does.
func LoadSettings(path string) error {
	settingsPath = path
	s := DefaultSettings()
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if s, err = parseSettings(b); err != nil {
			err = fmt.Errorf("%s: %w (using the defaults)", path, err)
		}
	case os.IsNotExist(err):
		err = nil
	}
	settingsSaved = s
	setSettings(s)
	return err
}

// parseSettings reads a settings file, old or new, over the defaults.
func parseSettings(b []byte) (Settings, error) {
	s := DefaultSettings()
	if err := json.Unmarshal(b, &s); err != nil {
		return DefaultSettings(), err
	}
	var keys map[string]json.RawMessage
	if json.Unmarshal(b, &keys) == nil {
		if _, ok := keys["voices"]; !ok {
			if _, ok := keys["effects"]; ok {
				s.Voices = s.Effects // an old file: the voices were heard with the effects
			}
		}
	}
	return s.valid(), nil
}

// valid is s with every value brought within what the game offers.
func (s Settings) valid() Settings {
	d := DefaultSettings()
	for _, v := range []*float64{&s.Master, &s.Music, &s.Effects, &s.Voices} {
		if math.IsNaN(*v) {
			*v = 0
		}
		*v = math.Round(clamp01(*v)*100) / 100
	}
	switch s.Window {
	case WindowFullscreen, WindowBorderless, WindowWindowed:
	default:
		s.Window = d.Window
	}
	if s.Width <= 0 || s.Height <= 0 || s.Width > 16384 || s.Height > 16384 {
		s.Width, s.Height = 0, 0
	}
	if !slices.Contains(qualityChoices, s.Quality) {
		s.Quality = nearest(qualityChoices, s.Quality, d.Quality)
	}
	if !slices.Contains(uiScaleChoices, s.UIScale) {
		s.UIScale = nearest(uiScaleChoices[1:], s.UIScale, 0)
	}
	if s.FPS != 0 && !slices.Contains(fpsChoices, s.FPS) {
		s.FPS = nearest(fpsChoices[:len(fpsChoices)-1], s.FPS, 0)
	}
	s.Zoom = max(minZoom, min(maxZoom, s.Zoom))
	s.Brightness = max(0, min(len(brightnessLevels)-1, s.Brightness))
	if !gradeKnown(s.Grade) {
		s.Grade = ""
	}
	return s
}

// nearest is the choice closest to v, or def when v is 0 or less.
func nearest(choices []int, v, def int) int {
	if v <= 0 {
		return def
	}
	best := choices[0]
	for _, c := range choices {
		if c > 0 && abs(float64(c-v)) < abs(float64(best-v)) {
			best = c
		}
	}
	return best
}

// saveSettings writes the saved settings to their file, through a
// temporary file so a crash never leaves half of one.
func saveSettings() {
	if settingsPath == "" {
		return
	}
	b, err := json.MarshalIndent(settingsSaved, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		log.Printf("settings: %v", err)
		return
	}
	tmp := settingsPath + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		log.Printf("settings: saving: %v", err)
		return
	}
	if err := os.Rename(tmp, settingsPath); err != nil {
		log.Printf("settings: saving: %v", err)
		_ = os.Remove(tmp)
	}
}

// changeSettings makes a change the player chose: in force at once, and
// saved.
func changeSettings(f func(s *Settings)) {
	saved := settingsSaved
	f(&saved)
	saved = saved.valid()
	cur := settings
	f(&cur)
	cur = cur.valid()
	if saved != settingsSaved {
		settingsSaved = saved
		saveSettings()
	}
	setSettings(cur)
}

// OverrideSettings puts a choice from the command line in force for this
// session, without saving it.
func OverrideSettings(f func(s *Settings)) {
	cur := settings
	f(&cur)
	setSettings(cur.valid())
}

// setSettings puts s in force: everything that follows a setting is told.
func setSettings(s Settings) {
	old := settings
	settings = s
	viewZoom = s.Zoom
	postFX = s.PostFX
	uiScaleSetting = float64(s.UIScale) / 100
	applyScreenShake(!s.NoShake)
	applyGrade(s.Grade)
	snd.applyVolumes()
	if old.Window != s.Window || old.Width != s.Width || old.Height != s.Height || old.VSync != s.VSync {
		applyWindow() // see display.go
	}
}

// applyScreenShake turns screen shake on or off: see feel.go.
func applyScreenShake(on bool) { screenShake = on }

// --- a hook for a setting whose code is on another branch ---------------------------

// colourGrades are the colour grades of the world the settings screen
// offers besides the game's own, by the name kept in the settings. None is
// on this branch, so the choice is not shown.
//
// HOOK(colour grade): claude/palette-test has the COMFY52 grade
// (palettegrade.go: ParsePaletteGrade, SetPaletteGrade). Once it is merged,
// list {"comfy", "COMFY52"} here and set the grade in applyGrade.
var colourGrades = []struct{ name, label string }{}

func gradeKnown(name string) bool {
	if name == "" {
		return true
	}
	for _, g := range colourGrades {
		if g.name == name {
			return true
		}
	}
	return false
}

// applyGrade puts colour grade name on the world ("" is the game's own).
func applyGrade(name string) {}
