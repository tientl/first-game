package client

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/sim"
)

// The moment a boss fight starts. When the boss this player has come upon
// wakes, the game goes on but for two and a half seconds it is shown as a
// scene: black bars slide in above and below, the view is drawn across to
// the boss as it rears and roars, its name comes up in great letters, and
// the view comes back. For the first second the player's keys are not
// read, so nobody runs blind out of the picture; the boss itself only
// roars that long (sim.BossRoarTicks), so nothing is lost by it. The battle
// music starts on the same beat (sound_events.go: a boss bar up).
//
// A boss that is lost and found again within introAgain does not get the
// scene twice.

const (
	introFrames = 156     // the whole scene: 2.6 s
	introLock   = 60      // the keys are not read this long
	introAgain  = 50 * 60 // frames before the same boss is shown in again
)

// bossIntro is the scene under way.
type bossIntro struct {
	kind   sim.CreatureKind
	at     int                      // the frame it began; the scene is on while frame-at < introFrames
	on     bool                     //
	bx, by float64                  // where the boss was
	last   map[sim.CreatureKind]int // the frame each boss was last shown in
}

// introUpdate starts the scene when a boss near the player wakes.
func (g *Game) introUpdate() {
	f := &g.lair
	in := &f.intro
	if in.on && g.frame-in.at >= introFrames {
		in.on = false
	}
	if f.awake == nil {
		f.awake, in.last = map[sim.CreatureKind]bool{}, map[sim.CreatureKind]int{}
	}
	seen := map[sim.CreatureKind]bool{}
	for _, l := range g.curr.Lairs {
		seen[l.Kind] = true
		was := f.awake[l.Kind]
		f.awake[l.Kind] = l.Awake
		if !l.Awake || was || g.curr.Boss.Kind != l.Kind {
			if l.Awake && g.curr.Boss.Kind != l.Kind {
				f.awake[l.Kind] = false // not this player's fight yet
			}
			continue
		}
		if l.Kind == sim.CreatureTitanoboa {
			g.leafBurst() // it comes out of its fronds
		}
		if t, ok := in.last[l.Kind]; ok && g.frame-t < introAgain {
			continue
		}
		in.kind, in.at, in.on = l.Kind, g.frame, true
		in.last[l.Kind] = g.frame
		in.bx, in.by = float64(l.X), float64(l.Y)
		if c, ok := g.bossOf(l.Kind); ok {
			in.bx, in.by = float64(c.X), float64(c.Y)
		}
		if snd != nil {
			snd.play("boom", 0.9, 0, 0.7)
			snd.play("thunder", 0.35, 0, 0.6)
		}
	}
	for k := range f.awake {
		if !seen[k] {
			delete(f.awake, k)
		}
	}
}

// introT is how far the scene is through, 0 to 1, and whether it is on.
func (g *Game) introT() (float64, bool) {
	in := &g.lair.intro
	if !in.on {
		return 0, false
	}
	return clamp01(float64(g.frame-in.at) / introFrames), true
}

// introLocked reports whether the player's keys are not read just now.
func (g *Game) introLocked() bool {
	in := &g.lair.intro
	return in.on && g.frame-in.at < introLock
}

// introPull is how far the view is drawn to the boss, 0 to 1, at t.
func introPull(t float64) float64 {
	return smooth(clamp01(t/0.2)) * smooth(clamp01((1-t)/0.27))
}

// introCamera moves the view's middle (cx, cy) towards the boss while the
// scene plays.
func (g *Game) introCamera(cx, cy float64) (float64, float64) {
	t, ok := g.introT()
	if !ok {
		return cx, cy
	}
	in := &g.lair.intro
	bx, by := in.bx, in.by
	if c, ok := g.bossOf(in.kind); ok {
		bx, by = float64(c.X), float64(c.Y)-2.6 // between its coils and its reared head
	}
	k := 0.8 * introPull(t)
	return cx + (bx-cx)*k, cy + (by-cy)*k
}

// drawBossIntro draws the scene's frame: the bars, the dark at the edges
// and the name.
func (g *Game) drawBossIntro(screen *ebiten.Image, sw, sh int) {
	t, ok := g.introT()
	if !ok {
		return
	}
	in := &g.lair.intro
	bars := smooth(clamp01(t/0.1)) * smooth(clamp01((1-t)/0.12))
	bh := float32(float64(sh) * 0.115 * bars)
	vector.FillRect(screen, 0, 0, float32(sw), bh, color.RGBA{0, 0, 0, 255}, false)
	vector.FillRect(screen, 0, float32(sh)-bh, float32(sw), bh, color.RGBA{0, 0, 0, 255}, false)

	vop := &ebiten.DrawImageOptions{} // the edges close in
	vb := g.art.vignette.Bounds()
	vop.GeoM.Scale(float64(sw)/float64(vb.Dx()), float64(sh)/float64(vb.Dy()))
	vop.ColorScale.ScaleAlpha(float32(0.75 * bars))
	vop.Filter = ebiten.FilterLinear
	screen.DrawImage(g.art.vignette, vop)

	// The name: it comes up from below as the boss roars, holds, and fades.
	show := smooth(clamp01((t-0.16)/0.1)) * smooth(clamp01((0.9-t)/0.1))
	if show <= 0 {
		return
	}
	u := float64(sh) / 1080
	name, epithet := strings.ToUpper(in.kind.Name()), sim.BossEpithet(in.kind)
	a := uint8(255 * show)
	y := float64(sh)*0.685 + (1-smooth(clamp01((t-0.16)/0.14)))*40*u
	ns := 7 * u
	spaced := strings.Join(strings.Split(name, ""), " ")
	nw := textWidth(spaced, ns)
	cx := float64(sw) / 2
	vector.FillRect(screen, 0, float32(y-18*u), float32(sw), float32(190*u), color.RGBA{0, 0, 0, uint8(120 * show)}, false)
	g.text(screen, spaced, cx, y, color.RGBA{uint8(int(246) * int(a) / 255), uint8(int(226) * int(a) / 255), uint8(int(170) * int(a) / 255), a}, text.AlignCenter, ns)
	ly := float32(y + 108*u)
	grow := float32(smooth(clamp01((t - 0.2) / 0.16)))
	vector.FillRect(screen, float32(cx)-float32(nw)/2*grow, ly, float32(nw)*grow, float32(math.Max(2, 3*u)), color.RGBA{uint8(200 * show), uint8(60 * show), uint8(40 * show), a}, false)
	g.text(screen, "— "+epithet+" —", cx, y+122*u, color.RGBA{a, a, a, a}, text.AlignCenter, 3*u)
}
