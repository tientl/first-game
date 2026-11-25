package client

import (
	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/dino3d"
	"survivalgame/internal/r3d"
)

// A trial: the tyrant as a 3D model, built and animated in code, rendered
// by a small software rasteriser from the game's camera and baked down to
// pixel-art frames at start-up (see package dino3d). It is off unless
// SetRex3D turns it on before the game is made; the drawn tyrant
// (dinos/rex.png) stays the default.
//
// The model can face eight ways; the game only knows left and right so far,
// so only those two are baked here.

var rex3D bool

// SetRex3D makes games made from now on show the 3D tyrant.
func SetRex3D(on bool) { rex3D = on }

// rex3DArt is the baked tyrant, [0] facing right and [1] left: more frames
// than the drawn one has poses.
type rex3DArt struct {
	on   bool
	walk [2][]sprite
	idle [2][]sprite
}

// genRex3D bakes the tyrant, and puts its frames in place of the drawn
// poses too, so that whatever shows a tyrant shows this one.
func genRex3D(a *creatureArt) {
	baked := dino3d.BakeRex(dino3d.DirRight, dino3d.DirLeft)
	conv := func(s r3d.Sprite) sprite { return sprite{img: ebiten.NewImageFromImage(s.Img), ax: s.AX, ay: s.AY} }
	for side, dir := range [2]int{dino3d.DirRight, dino3d.DirLeft} {
		f := &baked.Frames[dir]
		for _, s := range f[dino3d.RexWalk] {
			a.rex3.walk[side] = append(a.rex3.walk[side], conv(s))
		}
		for _, s := range f[dino3d.RexIdle] {
			a.rex3.idle[side] = append(a.rex3.idle[side], conv(s))
		}
		n := len(a.rex3.walk[side])
		a.rex[side][rexWalkA] = a.rex3.idle[side][0]
		a.rex[side][rexWalkB] = a.rex3.walk[side][n/4]
		a.rex[side][rexRoar] = conv(f[dino3d.RexRoar][0])
		a.rex[side][rexBite] = conv(f[dino3d.RexBite][0])
		a.rex[side][rexEatA] = conv(f[dino3d.RexEat][0])
		a.rex[side][rexEatB] = conv(f[dino3d.RexEat][1])
	}
	a.rex3.on = true
}

// rex3DMove is the 3D tyrant's frame when it is walking (period is how many
// ticks two steps take) or standing; the other poses come from a.rex.
func (a *creatureArt) rex3DMove(side, t int, moving bool, period int) sprite {
	if !moving {
		return a.rex3.idle[side][(t/36)%len(a.rex3.idle[side])]
	}
	w := a.rex3.walk[side]
	return w[(t*len(w)/period)%len(w)]
}
