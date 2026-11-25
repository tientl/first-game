package client

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Every sound in the game is synthesised in code at start-up (sfx.go,
// music.go), like the art. They are played through one software mixer that
// feeds a single Ebitengine player, so panning, pitch variation and smooth
// volume changes are all under our control.

const sampleRate = 44100

type bus uint8

const (
	busSFX bus = iota
	busMusic
	numBuses
)

// voice is one sound being played.
type voice struct {
	data   []float32 // mono, or interleaved stereo when stereo is set
	stereo bool
	pos    float64 // in frames
	rate   float64 // playback speed; also shifts the pitch
	gain   float32 // gain at the end of the previous mix buffer
	target float32 // gain wanted now; reached smoothly over one buffer
	pan    float32 // -1 left … 1 right
	loop   bool
	bus    bus
	done   bool
}

func (v *voice) frames() int {
	if v.stereo {
		return len(v.data) / 2
	}
	return len(v.data)
}

// mixer is an endless float32 stereo stream. Its Read runs on the audio
// goroutine; everything else locks mu.
type mixer struct {
	mu        sync.Mutex
	voices    []*voice
	busGain   [numBuses]float32
	busTarget [numBuses]float32
	buf       []float32
}

const maxVoices = 48

func (m *mixer) add(v *voice) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.voices) >= maxVoices {
		// Drop the oldest one-shot to make room.
		for i, o := range m.voices {
			if !o.loop {
				m.voices = append(m.voices[:i], m.voices[i+1:]...)
				break
			}
		}
	}
	m.voices = append(m.voices, v)
}

func (m *mixer) Read(p []byte) (int, error) {
	frames := len(p) / 8
	if frames == 0 {
		return 0, nil
	}
	if cap(m.buf) < frames*2 {
		m.buf = make([]float32, frames*2)
	}
	out := m.buf[:frames*2]
	clear(out)
	inv := 1 / float32(frames)

	m.mu.Lock()
	live := m.voices[:0]
	for _, v := range m.voices {
		g0 := v.gain * m.busGain[v.bus]
		g1 := v.target * m.busTarget[v.bus]
		n := v.frames()
		if n < 2 {
			continue
		}
		if g0 == 0 && g1 == 0 {
			// Silent: keep time moving so loops stay in step.
			v.pos += v.rate * float64(frames)
		} else {
			var l, r float32 = 1, 1
			if !v.stereo {
				a := (float64(v.pan) + 1) * math.Pi / 4
				l, r = float32(math.Cos(a)*math.Sqrt2), float32(math.Sin(a)*math.Sqrt2)
			}
			for i := 0; i < frames; i++ {
				if v.pos >= float64(n-1) {
					if !v.loop {
						v.done = true
						break
					}
					v.pos -= float64(n - 1)
				}
				g := g0 + (g1-g0)*float32(i)*inv
				ip := int(v.pos)
				f := float32(v.pos - float64(ip))
				if v.stereo {
					a, b := v.data[2*ip], v.data[2*ip+2]
					c, d := v.data[2*ip+1], v.data[2*ip+3]
					out[2*i] += (a + (b-a)*f) * g
					out[2*i+1] += (c + (d-c)*f) * g
				} else {
					s := (v.data[ip] + (v.data[ip+1]-v.data[ip])*f) * g
					out[2*i] += s * l
					out[2*i+1] += s * r
				}
				v.pos += v.rate
			}
		}
		if v.loop && v.pos >= float64(n-1) {
			v.pos = math.Mod(v.pos, float64(n-1))
		}
		v.gain = v.target
		if !v.done && (v.loop || v.pos < float64(n-1)) {
			live = append(live, v)
		}
	}
	clear(m.voices[len(live):])
	m.voices = live
	m.busGain = m.busTarget
	m.mu.Unlock()

	for i, s := range out {
		// Soft clip, so a pile-up of loud sounds bends instead of cracking.
		x := float64(s) * 0.9
		if x > 0.6 || x < -0.6 {
			x = math.Tanh(x)
		}
		binary.LittleEndian.PutUint32(p[4*i:], math.Float32bits(float32(x)))
	}
	return frames * 8, nil
}

// --- settings ---------------------------------------------------------------

// audioSettings are the player's volume choices, kept in a small JSON file.
type audioSettings struct {
	Music   float64 `json:"music"`
	Effects float64 `json:"effects"`
	Zoom    int     `json:"zoom,omitempty"`   // the view's zoom; see render.go
	Ground  string  `json:"ground,omitempty"` // the look of the ground: "" (green) or "warm"; see groundhd.go
}

// --- sound system -----------------------------------------------------------

type soundSystem struct {
	ctx    *audio.Context
	player *audio.Player
	mix    *mixer
	rng    *rand.Rand

	sfx   map[string][][]float32
	loops map[string]*voice
	// loopWant is the gain each ambience loop is heading for; the actual
	// gain follows it slowly so nothing pops.
	loopWant map[string]float64
	loopCur  map[string]float64

	music      map[string]*voice
	musicCur   map[string]float64
	musicWant  string
	musicReady chan musicTrack

	settings     audioSettings
	settingsPath string

	lx, ly   float64 // listener position, in tiles
	lastPlay map[string]time.Time

	onPlay func(name string) // test hook
}

type musicTrack struct {
	name string
	data []float32
}

// snd is the game's sound system, or nil when audio is unavailable (tests,
// the screenshot tool). Every method is safe to call on nil.
var snd *soundSystem

// InitAudio starts sound and music. settingsPath is where the volume
// settings are read from and saved to. Without a call, the game is silent.
func InitAudio(settingsPath string) {
	s := &soundSystem{
		mix:          &mixer{},
		rng:          rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7)),
		loops:        make(map[string]*voice),
		loopWant:     make(map[string]float64),
		loopCur:      make(map[string]float64),
		music:        make(map[string]*voice),
		musicCur:     make(map[string]float64),
		musicReady:   make(chan musicTrack, 4),
		settings:     audioSettings{Music: 0.6, Effects: 0.8},
		settingsPath: settingsPath,
		lastPlay:     make(map[string]time.Time),
	}
	if b, err := os.ReadFile(settingsPath); err == nil {
		var st audioSettings
		if json.Unmarshal(b, &st) == nil {
			s.settings.Music = clamp01(st.Music)
			s.settings.Effects = clamp01(st.Effects)
			s.settings.Zoom = max(minZoom, min(maxZoom, st.Zoom))
			viewZoom = s.settings.Zoom
			if s.settings.Ground = st.Ground; st.Ground == "warm" {
				groundStyle = GroundWarm
			}
		}
	}

	t0 := time.Now()
	s.sfx = genSFX()
	for name, data := range genLoops() {
		v := &voice{data: data, rate: 1, loop: true, bus: busSFX}
		s.loops[name] = v
		s.mix.add(v)
	}
	log.Printf("audio: sound effects ready in %v", time.Since(t0).Round(time.Millisecond))
	for _, name := range []string{"menu", "day", "night", "cave", "boss_jungle", "boss_sea"} {
		go func() { s.musicReady <- musicTrack{name, genMusic(name)} }()
	}

	s.ctx = audio.NewContext(sampleRate)
	p, err := s.ctx.NewPlayerF32(s.mix)
	if err != nil {
		log.Printf("audio: %v", err)
		return
	}
	p.SetBufferSize(50 * time.Millisecond)
	p.Play()
	s.player = p
	s.applyVolumes()
	snd = s
}

func (s *soundSystem) applyVolumes() {
	// Squared, so the slider feels even to the ear.
	s.mix.mu.Lock()
	s.mix.busTarget[busSFX] = float32(s.settings.Effects * s.settings.Effects)
	s.mix.busTarget[busMusic] = float32(s.settings.Music * s.settings.Music)
	s.mix.mu.Unlock()
}

func (s *soundSystem) saveSettings() {
	b, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.settingsPath), 0o755)
	if err := os.WriteFile(s.settingsPath, b, 0o644); err != nil {
		log.Printf("audio: saving settings: %v", err)
	}
}

func (s *soundSystem) volume(music bool) float64 {
	if s == nil {
		return 0
	}
	if music {
		return s.settings.Music
	}
	return s.settings.Effects
}

func (s *soundSystem) setVolume(music bool, v float64) {
	if s == nil {
		return
	}
	v = math.Round(clamp01(v)*10) / 10
	if music {
		s.settings.Music = v
	} else {
		s.settings.Effects = v
	}
	s.applyVolumes()
	s.saveSettings()
}

// play starts a one-shot sound. vol is 0..1, pan -1..1, rate 1 = as made.
func (s *soundSystem) play(name string, vol, pan, rate float64) {
	if s == nil || vol <= 0.001 || !(rate > 0.01 && rate < 100) {
		return // a rate at or below zero would play backwards, off the start of the data
	}
	vs := s.sfx[name]
	if len(vs) == 0 {
		return
	}
	if s.onPlay != nil {
		s.onPlay(name)
	}
	data := vs[s.rng.IntN(len(vs))]
	s.mix.add(&voice{
		data: data, rate: rate, gain: float32(vol), target: float32(vol),
		pan: float32(max(-1, min(1, pan))), bus: busSFX,
	})
}

// playVar plays a sound with a little random pitch, so repeats do not sound
// mechanical.
func (s *soundSystem) playVar(name string, vol, pan, spread float64) {
	if s == nil {
		return
	}
	s.play(name, vol, pan, 1+(s.rng.Float64()*2-1)*spread)
}

// hearingRange is how far away, in tiles, a sound can still be heard.
const hearingRange = 14.0

// playAt plays a sound placed in the world, quieter and panned with
// distance from the listener.
func (s *soundSystem) playAt(name string, x, y, vol, spread float64) {
	if s == nil {
		return
	}
	dx, dy := x-s.lx, (y-s.ly)*1.3
	d := math.Hypot(dx, dy)
	if d >= hearingRange {
		return
	}
	a := 1 - d/hearingRange
	s.playVar(name, vol*a*a, dx/8, spread)
}

// ui plays an interface sound, at most once per few frames.
func (s *soundSystem) ui(name string) {
	if s == nil || !s.ready(name, 40*time.Millisecond) {
		return
	}
	s.play(name, 0.5, 0, 1)
}

// ready rate-limits a sound: it reports whether name may play again now.
func (s *soundSystem) ready(name string, gap time.Duration) bool {
	now := time.Now()
	if now.Sub(s.lastPlay[name]) < gap {
		return false
	}
	s.lastPlay[name] = now
	return true
}

func (s *soundSystem) setListener(x, y float64) {
	if s != nil {
		s.lx, s.ly = x, y
	}
}

// setLoop sets where an ambience loop's volume is heading.
func (s *soundSystem) setLoop(name string, gain float64) {
	if s != nil {
		s.loopWant[name] = gain
	}
}

func (s *soundSystem) silenceLoops() {
	if s != nil {
		for k := range s.loopWant {
			s.loopWant[k] = 0
		}
	}
}

// setMusic picks the track to fade to ("" for silence).
func (s *soundSystem) setMusic(name string) {
	if s != nil {
		s.musicWant = name
	}
}

// update runs once per frame on the game thread: it picks up finished music
// and moves every volume a step towards its target.
func (s *soundSystem) update() {
	if s == nil {
		return
	}
	for {
		select {
		case t := <-s.musicReady:
			v := &voice{data: t.data, stereo: true, rate: 1, loop: true, bus: busMusic}
			s.music[t.name] = v
			s.mix.add(v)
			continue
		default:
		}
		break
	}

	const loopStep, musicStep = 1.0 / 30, 1.0 / 240 // half a second; four seconds
	s.mix.mu.Lock()
	for name, v := range s.loops {
		cur := approach(s.loopCur[name], s.loopWant[name], loopStep)
		s.loopCur[name] = cur
		v.target = float32(cur)
	}
	for name, v := range s.music {
		want := 0.0
		if name == s.musicWant {
			want = 1
		}
		cur := approach(s.musicCur[name], want, musicStep)
		if want == 0 {
			cur = approach(s.musicCur[name], want, musicStep*1.5)
		}
		s.musicCur[name] = cur
		v.target = float32(cur * cur)
	}
	s.mix.mu.Unlock()
}

func approach(cur, want, step float64) float64 {
	switch {
	case cur < want:
		return min(want, cur+step)
	case cur > want:
		return max(want, cur-step)
	}
	return cur
}

// saveGround keeps the look of the ground with the settings.
func (s *soundSystem) saveGround(g GroundStyle) {
	name := ""
	if g == GroundWarm {
		name = "warm"
	}
	if s == nil || s.settings.Ground == name {
		return
	}
	s.settings.Ground = name
	s.saveSettings()
}

// saveZoom keeps the view's zoom with the settings.
func (s *soundSystem) saveZoom(z int) {
	if s == nil || s.settings.Zoom == z {
		return
	}
	s.settings.Zoom = z
	s.saveSettings()
}
