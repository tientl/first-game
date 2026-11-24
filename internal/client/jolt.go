package client

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// A player who is struck starts: the whole figure jumps up and back from
// the way it faces, flashes white and then red, throws its arms out for a
// moment, and comes down again. It is only drawn: the server does not move
// them. Each jolt starts when a snapshot shows PlayerView.HurtSeq changed.

const (
	joltFrames = 14 // how long a jolt lasts, in frames at 60 a second
	joltFling  = 6  // the arms are out until here
)

// How far the figure is thrown back and up on each frame of a jolt, in art
// pixels: out at once, down more slowly.
var (
	joltBack = [joltFrames]float64{2, 3, 4, 4, 4, 3, 3, 2, 2, 1, 1, 1, 0, 0}
	joltHop  = [joltFrames]float64{2, 4, 5, 5, 4, 4, 3, 2, 1, 0, 0, 0, 0, 0}
)

// noteHurts records when each player in s was last struck. It is called
// before the snapshot replaces the previous one.
func (g *Game) noteHurts(s protocol.Snapshot) {
	if g.hurtAt == nil {
		g.hurtAt = make(map[sim.PlayerID]int)
	}
	for _, p := range s.Players {
		if q, ok := g.prevPlayers[p.ID]; ok && q.HurtSeq != p.HurtSeq && !p.Dead {
			g.hurtAt[p.ID] = g.frame
		}
	}
}

// jolting reports how many frames into a jolt player id is.
func (g *Game) jolting(id sim.PlayerID) (age int, ok bool) {
	at, ok := g.hurtAt[id]
	age = g.since(at)
	return age, ok && age >= 0 && age < joltFrames
}

// joltShift is where a figure facing dir is drawn age frames into a jolt,
// from where it stands.
func joltShift(age int, dir sim.Dir) (dx, dy float64) {
	fx, _ := dir.Delta()
	return -float64(fx) * joltBack[age], -joltHop[age]
}

// flingPose is the arms of p on the first frames of a jolt: both thrown
// out, but for a hand that holds a tool.
func flingPose(p protocol.PlayerView) figPose {
	fp := figPose{off: poseFling, spark: -1}
	if fistHolds(p.Held) {
		fp.hand = poseFling
	}
	return fp
}

// The scratch image a jolted figure is drawn in, and where its feet are.
const (
	joltBufSize = 128
	joltBufX    = 64
	joltBufY    = 92
)

// drawJolted draws a figure standing at (x, y) age frames into a jolt: draw
// is called to draw it at the place it is given in a scratch image, which is
// then put on dst thrown back, with the flash over it.
func (g *Game) drawJolted(dst *ebiten.Image, age int, dir sim.Dir, x, y float64, draw func(buf *ebiten.Image, x, y float64)) {
	if g.joltBuf == nil {
		g.joltBuf = ebiten.NewImage(joltBufSize, joltBufSize)
	}
	g.joltBuf.Clear()
	fx, fy := x-math.Floor(x), y-math.Floor(y)
	draw(g.joltBuf, joltBufX+fx, joltBufY+fy)
	dx, dy := joltShift(age, dir)
	ox, oy := math.Floor(x)-joltBufX+dx, math.Floor(y)-joltBufY+dy
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	dst.DrawImage(g.joltBuf, op)

	// The flash: the figure's shape in one colour over it.
	r, gr, b, a := 1.0, 1.0, 1.0, 0.0
	switch {
	case age < 2:
		a = 0.85
	case age < 4:
		r, gr, b, a = 1, 0.2, 0.15, 0.6
	case age < 7:
		r, gr, b, a = 1, 0.2, 0.15, 0.3
	}
	if a == 0 {
		return
	}
	var cm colorm.ColorM
	cm.Scale(0, 0, 0, a)
	cm.Translate(r, gr, b, 0)
	cop := &colorm.DrawImageOptions{}
	cop.GeoM.Translate(ox, oy)
	colorm.DrawImage(dst, g.joltBuf, cm, cop)
}
