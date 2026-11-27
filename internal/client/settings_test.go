package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// keepSettings has a test work on settings of its own, kept in a file in
// a temporary folder, and puts everything back after it.
func keepSettings(t *testing.T) string {
	t.Helper()
	oldCur, oldSaved, oldPath := settings, settingsSaved, settingsPath
	t.Cleanup(func() {
		settingsSaved, settingsPath = oldSaved, oldPath
		setSettings(oldCur)
	})
	settingsPath = ""
	setSettings(DefaultSettings())
	return filepath.Join(t.TempDir(), "settings.json")
}

// readSettingsFile is what the file at path holds.
func readSettingsFile(t *testing.T, path string) Settings {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("the settings file is not JSON: %v\n%s", err, b)
	}
	return s
}

// A first run has no file: the defaults are fullscreen at the monitor's
// resolution, full quality, the interface sized automatically, vsync on;
// nothing is written until something is changed.
func TestFirstRunTakesTheDefaults(t *testing.T) {
	path := keepSettings(t)
	if err := LoadSettings(path); err != nil {
		t.Fatal(err)
	}
	s := CurrentSettings()
	if s.Window != WindowFullscreen || s.Width != 0 || s.Height != 0 || s.Quality != 100 || s.UIScale != 0 || !s.VSync || s.FPS != 0 {
		t.Errorf("first run: %+v", s)
	}
	if s.Master != 1 || s.Music != 0.6 || s.Effects != 0.8 || s.Voices != 0.8 || s.MuteUnfocused {
		t.Errorf("first run's sound: %+v", s)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a first run wrote the settings file before anything changed (%v)", err)
	}
}

// A settings file from before the settings screen (volumes, zoom, ground)
// loads: what it holds is kept, what it lacks takes the defaults, and the
// creatures' voices, which played with the effects, start at their volume.
func TestAnOldSettingsFileLoads(t *testing.T) {
	path := keepSettings(t)
	old := `{
  "music": 0.3,
  "effects": 0.5,
  "zoom": 2,
  "ground": "warm"
}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadSettings(path); err != nil {
		t.Fatal(err)
	}
	s := CurrentSettings()
	if s.Music != 0.3 || s.Effects != 0.5 || s.Voices != 0.5 || s.Master != 1 {
		t.Errorf("volumes from an old file: %+v", s)
	}
	if s.Zoom != 2 || viewZoom != 2 || s.Ground != "warm" || groundStyle != GroundWarm {
		t.Errorf("zoom %d (view %d), ground %q (%v) from an old file", s.Zoom, viewZoom, s.Ground, groundStyle)
	}
	if s.Window != WindowFullscreen || s.Quality != 100 || !s.VSync || s.NoShake {
		t.Errorf("what an old file lacks is not the default: %+v", s)
	}
}

// What is changed is saved at once, and is what the next start loads.
func TestSettingsSurviveARestart(t *testing.T) {
	path := keepSettings(t)
	if err := LoadSettings(path); err != nil {
		t.Fatal(err)
	}
	changeSettings(func(s *Settings) {
		s.Master, s.Voices, s.MuteUnfocused = 0.7, 0.25, true
		s.Window, s.Width, s.Height = WindowWindowed, 1600, 900
		s.Quality, s.UIScale, s.FPS, s.VSync = 75, 150, 60, false
		s.Zoom, s.NoShake, s.PostFX, s.Ground = -1, true, true, "warm"
	})
	want := CurrentSettings()
	if got := readSettingsFile(t, path); got != want {
		t.Errorf("saved %+v\nwant  %+v", got, want)
	}
	if uiScaleSetting != 1.5 || !postFX || screenShake || viewZoom != -1 {
		t.Errorf("not in force: UI scale %g, light %v, shake %v, zoom %d", uiScaleSetting, postFX, screenShake, viewZoom)
	}

	// The next start.
	setSettings(DefaultSettings())
	settingsSaved = DefaultSettings()
	if err := LoadSettings(path); err != nil {
		t.Fatal(err)
	}
	if got := CurrentSettings(); got != want {
		t.Errorf("loaded again %+v\nwant         %+v", got, want)
	}
}

// A choice made on the command line is in force for the session, and is
// not saved when something else is changed.
func TestCommandLineSettingsAreNotSaved(t *testing.T) {
	path := keepSettings(t)
	if err := LoadSettings(path); err != nil {
		t.Fatal(err)
	}
	OverrideSettings(func(s *Settings) { s.Quality, s.Window = 50, WindowWindowed })
	if s := CurrentSettings(); s.Quality != 50 || s.Window != WindowWindowed {
		t.Fatalf("the command line's choice is not in force: %+v", s)
	}
	changeSettings(func(s *Settings) { s.Music = 0.2 })
	saved := readSettingsFile(t, path)
	if saved.Quality != 100 || saved.Window != WindowFullscreen || saved.Music != 0.2 {
		t.Errorf("saved %+v: the command line's choices went into the file", saved)
	}
	if s := CurrentSettings(); s.Quality != 50 || s.Music != 0.2 {
		t.Errorf("in force %+v", s)
	}
	// Changed in the menu, it is the player's choice and is saved.
	changeSettings(func(s *Settings) { s.Quality = 75 })
	if saved := readSettingsFile(t, path); saved.Quality != 75 {
		t.Errorf("a choice made in the menu was not saved: %+v", saved)
	}
}

// Whatever a file holds, what is put in force is something the game offers.
func TestStrangeSettingsAreBroughtInRange(t *testing.T) {
	s, err := parseSettings([]byte(`{"master": 3, "music": -1, "effects": 0.333, "window": "huge",
		"width": -5, "height": 900, "quality": 80, "uiScale": 260, "fps": 144, "zoom": 9, "ground": "purple", "grade": "nope"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultSettings()
	want.Master, want.Music, want.Effects, want.Voices = 1, 0, 0.33, 0.33
	want.Quality, want.UIScale, want.FPS, want.Zoom = 75, 250, 120, maxZoom
	if s != want {
		t.Errorf("got  %+v\nwant %+v", s, want)
	}
	if _, err := parseSettings([]byte(`{"music": `)); err == nil {
		t.Error("a broken file parsed")
	}
}

// A file that is not JSON leaves the defaults in force.
func TestABrokenSettingsFileGivesTheDefaults(t *testing.T) {
	path := keepSettings(t)
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadSettings(path); err == nil {
		t.Error("a broken file loaded without an error")
	}
	if CurrentSettings() != DefaultSettings() {
		t.Errorf("a broken file gave %+v", CurrentSettings())
	}
}

// The buses follow the volumes: each squared, under the whole, and all
// silent while muted.
func TestVolumesSetTheBuses(t *testing.T) {
	st := DefaultSettings()
	st.Master, st.Music, st.Effects, st.Voices = 0.5, 1, 0.5, 0
	g := busGains(st, false)
	if g[busMusic] != 0.25 || g[busSFX] != 0.0625 || g[busVoice] != 0 {
		t.Errorf("gains %v", g)
	}
	if g := busGains(st, true); g != ([numBuses]float32{}) {
		t.Errorf("muted, the gains are %v", g)
	}
	for _, c := range []struct {
		name  string
		voice bool
	}{{"rex_roar", true}, {"trike_call", true}, {"bellow", true}, {"tribe_warcry", true}, {"monkey_chatter", true},
		{"chop", false}, {"step_grass", false}, {"ui_select", false}, {"chomp", false}, {"splash", false}} {
		if creatureVoice(c.name) != c.voice {
			t.Errorf("%s is a creature's voice: %v, want %v", c.name, creatureVoice(c.name), c.voice)
		}
	}
}

// The render quality draws the world at a whole number of canvas pixels to
// an art pixel below the screen's: about three quarters and a half.
func TestRenderScale(t *testing.T) {
	for _, c := range []struct{ r, q, want int }{
		{3, 100, 3}, {3, 75, 2}, {3, 50, 1}, // 1080p
		{4, 100, 4}, {4, 75, 3}, {4, 50, 2}, // 1440p
		{6, 75, 5}, {6, 50, 3}, // 4K
		{2, 75, 1}, {2, 50, 1}, {1, 50, 1}, // 720p, and zoomed far out
	} {
		if got := renderScaleFor(c.r, c.q); got != c.want {
			t.Errorf("scale %d at %d%%: %d, want %d", c.r, c.q, got, c.want)
		}
	}
	if w, h := renderSize(1920, 1080, 3, 2); w != 1280 || h != 720 {
		t.Errorf("1080p at 2 of 3 is drawn at %dx%d", w, h)
	}
	if w, h := renderSize(1920, 1080, 3, 3); w != 1920 || h != 1080 {
		t.Errorf("1080p at full quality is drawn at %dx%d", w, h)
	}
}

// The resolutions offered fit the monitor: in fullscreen its own first,
// then the smaller; in a window, the sizes that fit with a frame and a
// taskbar. The sharpest are 1080p and its whole multiples.
func TestResolutionChoices(t *testing.T) {
	has := func(rs []resolution, r resolution) bool {
		for _, x := range rs {
			if x == r {
				return true
			}
		}
		return false
	}
	full := resolutionChoices(WindowFullscreen, 2560, 1440, 1)
	if !full[0].native() || !has(full, resolution{1920, 1080}) || !has(full, resolution{1280, 720}) || has(full, resolution{2560, 1440}) || has(full, resolution{3840, 2160}) {
		t.Errorf("fullscreen on 1440p: %v", full)
	}
	win := resolutionChoices(WindowWindowed, 1920, 1080, 1)
	if has(win, resolution{1920, 1080}) || !has(win, resolution{1600, 900}) || win[0].native() {
		t.Errorf("windowed on 1080p: %v", win)
	}
	// A 4K monitor scaled to 150%: the window's sizes are in real pixels.
	win4k := resolutionChoices(WindowWindowed, 3840, 2160, 1.5)
	if !has(win4k, resolution{3200, 1800}) || has(win4k, resolution{3840, 2160}) {
		t.Errorf("windowed on 4K at 150%%: %v", win4k)
	}
	for r, want := range map[resolution]bool{{1920, 1080}: true, {3840, 2160}: true, {2560, 1440}: false, {1280, 720}: false, {1600, 900}: false} {
		if sharpest(r) != want {
			t.Errorf("%v sharpest: %v", r, !want)
		}
	}
	// A size from another monitor comes down to the nearest that fits.
	s := DefaultSettings()
	s.Window, s.Width, s.Height = WindowWindowed, 2560, 1440
	if got := currentResolution(win, s); got != (resolution{1600, 900}) {
		t.Errorf("2560x1440 in a window on 1080p is %v", got)
	}
	s.Width, s.Height = 0, 0
	if got := currentResolution(win, s); got != win[len(win)-1] {
		t.Errorf("no size in a window on 1080p is %v", got)
	}
}

// The settings screen's rows change the settings: a step along a choice, a
// switch flipped, a slider moved, and "Mặc định" puts a tab back.
func TestTheSettingsScreenChangesTheSettings(t *testing.T) {
	keepSettings(t)
	s := newSettingsScreen(false)
	row := func(id setRowID) setRow {
		for _, r := range s.rows() {
			if r.id == id {
				return r
			}
		}
		t.Fatalf("no row %d on tab %d", id, s.tab)
		return setRow{}
	}

	s.tab = setTabAudio
	s.step(row(rowMaster), -1)
	s.step(row(rowVoices), -1)
	s.step(row(rowMuteUnfocused), 1)
	if st := CurrentSettings(); st.Master != 0.95 || st.Voices != 0.75 || !st.MuteUnfocused {
		t.Errorf("audio: %+v", st)
	}

	s.showTab("display")
	s.step(row(rowQuality), 1)
	s.step(row(rowWindow), 1)
	s.step(row(rowZoom), 1)
	flip(rowShake)
	s.step(row(rowGround), 1)
	st := CurrentSettings()
	if st.Quality != 75 || st.Window != WindowBorderless || st.Zoom != 1 || !st.NoShake || st.Ground != "warm" {
		t.Errorf("display: %+v", st)
	}
	if screenShake || groundStyle != GroundWarm || viewZoom != 1 {
		t.Errorf("not in force: shake %v, ground %v, zoom %d", screenShake, groundStyle, viewZoom)
	}
	s.step(row(rowUIScale), 1) // Tự động → 100%
	if CurrentSettings().UIScale != 100 || uiScaleSetting != 1 {
		t.Errorf("the interface's size: %d (%g)", CurrentSettings().UIScale, uiScaleSetting)
	}

	s.resetTab() // the display's settings back; the sound's stay
	st = CurrentSettings()
	if st.Quality != 100 || st.Window != WindowFullscreen || st.Zoom != 0 || st.NoShake || st.Ground != "" || st.UIScale != 0 {
		t.Errorf("after Mặc định on Hiển thị: %+v", st)
	}
	if st.Master != 0.95 || !st.MuteUnfocused {
		t.Errorf("Mặc định on Hiển thị changed the sound: %+v", st)
	}
	if s.setTab(setTabControls); len(s.rows()) != 0 || s.line != 0 || s.foot != 1 {
		t.Errorf("the controls tab: %d rows, line %d, foot %d", len(s.rows()), s.line, s.foot)
	}
}

// Every row of every tab, with the most rows the display tab can have,
// lies above the foot of the panel.
func TestTheSettingsRowsFitThePanel(t *testing.T) {
	keep := colourGrades
	defer func() { colourGrades = keep }()
	colourGrades = []struct{ name, label string }{{"comfy", "COMFY52"}}
	for tab := range numSetTabs {
		rows := setRows(tab)
		if n := len(rows); n > 0 {
			if bottom := rowY(rows, n-1) + rowPitch(rows) - setRowGap; bottom > setFootY-12 {
				t.Errorf("tab %d: the last row ends at %.0f, under the foot's line at %d", tab, bottom, setFootY-10)
			}
		}
	}
}
