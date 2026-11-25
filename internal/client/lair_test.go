package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// A boss waking starts its entrance once: the keys are held for the first
// second, the view goes to the boss and comes back, and waking again soon
// after does not play it a second time.
func TestBossIntroPlaysOncePerFight(t *testing.T) {
	g := &Game{}
	set := func(awake bool) {
		g.curr.Lairs = []protocol.LairView{{Kind: sim.CreatureTitanoboa, X: 40, Y: 40, Radius: sim.TitanArena, Awake: awake, Closed: awake}}
		g.curr.Boss = protocol.BossView{}
		if awake {
			g.curr.Boss.Kind = sim.CreatureTitanoboa
		}
		g.frame++
		g.lairUpdate()
	}
	set(false)
	if _, on := g.introT(); on || g.introLocked() {
		t.Fatalf("the entrance plays for a sleeping boss")
	}
	set(true)
	if _, on := g.introT(); !on || !g.introLocked() {
		t.Fatalf("the boss woke: no entrance, or the keys are not held")
	}
	if x, y := g.introCamera(30, 40); x != 30 || y != 40 {
		t.Errorf("the view jumps to (%.1f, %.1f) on the first frame", x, y)
	}
	g.frame += introFrames / 2
	if g.introLocked() {
		t.Errorf("the keys are still held half way through")
	}
	if x, _ := g.introCamera(30, 40); x < 35 || x > 40 {
		t.Errorf("half way through the view is at x %.1f, want most of the way from 30 to the boss at 40", x)
	}
	if g.lair.roots == 0 {
		set(true)
	}
	if g.lair.roots <= 0 {
		t.Errorf("the roots do not rise round the shut arena")
	}
	g.frame += introFrames
	set(true)
	if _, on := g.introT(); on {
		t.Fatalf("the entrance never ends")
	}
	set(false) // lost, and found again at once
	set(true)
	if _, on := g.introT(); on {
		t.Errorf("the entrance played twice within %d frames", introAgain)
	}
	set(false)
	g.frame += introAgain
	set(true)
	if _, on := g.introT(); !on {
		t.Errorf("a new fight long after did not get its entrance")
	}
}

func TestIntroPullGoesOutAndComesBack(t *testing.T) {
	if introPull(0) != 0 || introPull(1) != 0 {
		t.Errorf("the view is not home at the ends: %.2f, %.2f", introPull(0), introPull(1))
	}
	if p := introPull(0.5); p < 0.99 {
		t.Errorf("mid-scene the view is pulled %.2f of the way", p)
	}
}

// Every prop of a lair has its drawing, and only those that lie flat are
// drawn under what stands.
func TestLairPropsAreDrawn(t *testing.T) {
	g := &Game{}
	for v := uint8(0); v < sim.NumLairProps; v++ {
		s := g.lairSprite(protocol.EntityView{Kind: sim.EntityLairProp, Var: v, TX: 5, TY: 7})
		if s.img == nil || s.w() < 8 {
			t.Errorf("lair prop %d has no drawing", v)
		}
	}
	if !lairFlat(sim.LairNest) || !lairFlat(sim.LairTrail) || lairFlat(sim.LairRuin) || lairFlat(sim.LairCover) {
		t.Errorf("the wrong props lie flat")
	}
	a := lairArtOf()
	if c := a.cover; c.w() < 150 || c.h() < 120 {
		t.Errorf("the fronds over the snake are %v x %v art pixels: too small to hide it", c.w(), c.h())
	}
	for i, s := range a.vine {
		if s.img == nil {
			t.Errorf("no drawing for the arena's roots %d", i)
		}
	}
}
