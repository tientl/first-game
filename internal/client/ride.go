package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Tamed beasts and their riders: a rider drawn astride its mount on a
// saddle cloth, a little heart over every pet, hearts floating up while a
// beast is fed, and the saddle's icon.

// seat is where a rider sits on a mount of kind k: how far behind the
// sprite's anchor, and how high its back is, in art pixels (facing right).
func seat(k sim.CreatureKind) (back, height float64) {
	s := beastScale(k) // measured on the designs; the beasts are drawn bigger
	switch k {
	case sim.CreatureTrike:
		return 2 * s, 18 * s
	case sim.CreatureRaptor:
		return 1 * s, 14 * s
	case sim.CreatureMammoth:
		return 3 * s, 32 * s
	case sim.CreatureToro:
		return 2 * s, 18 * s
	case sim.CreatureOrnitho:
		return 0, 19 * s
	}
	return 0, 16 * s
}

// riderLegs is how far a rider's feet hang below the mount's back.
const riderLegs = 5

var saddleRamp = mkRamp(0x4a2a18, 0x6e4224, 0x925c34, 0xb57a48)

// drawRider draws p on the back of its mount c, drawn at d.
func (g *Game) drawRider(dst *ebiten.Image, p protocol.PlayerView, c protocol.CreatureView, d drawable) {
	back, height := seat(c.Kind)
	side := 1.0
	dir := sim.DirRight
	if c.FaceLeft {
		side, dir = -1, sim.DirLeft
	}
	_, bob := g.creatureSprite(c, d.moving)
	x, y := d.sx-back*side, d.sy-bob-height
	// The saddle cloth over its back.
	for row := 0.0; row < 4; row++ {
		w := 7 - row*0.5
		col := saddleRamp.at(3 - int(row))
		vector.FillRect(dst, float32(math.Round(x-w)), float32(math.Round(y+row-1)), float32(math.Round(2*w)), 1, col, false)
	}
	vector.FillRect(dst, float32(math.Round(x-7)), float32(math.Round(y+3)), 14, 1, saddleRamp[0], false)
	tint := playerColors[int(p.ID)%len(playerColors)]
	ride := bob
	if d.moving {
		ride += float64((g.frame / 6) % 2) // jogged along
	}
	fy := y + riderLegs - ride
	drawBody(dst, g.art.body(p.Look), dir, 0, x, fy, 1, 0, tint)
	g.drawHeld(dst, p.Held, dir, x, fy, 0, 0)
}

// drawPetMark draws a little heart over a pet: red over mine, pale over
// someone else's.
func (g *Game) drawPetMark(dst *ebiten.Image, c protocol.CreatureView, x, top float64) {
	col := color.RGBA{0xe0, 0x4a, 0x5a, 0xff}
	if c.Owner != g.myID {
		col = color.RGBA{0xd8, 0xd0, 0xd8, 0xff}
	}
	top += markInset(c.Kind) * beastScale(c.Kind) // down to the top of its head
	heart(dst, x, top-5, col)
	if c.Stay {
		vector.FillRect(dst, float32(math.Round(x+4)), float32(math.Round(top-6)), 1, 3, colWhite, false) // a pause mark
		vector.FillRect(dst, float32(math.Round(x+6)), float32(math.Round(top-6)), 1, 3, colWhite, false)
	}
}

// heart draws a 5x4 pixel heart centred on x with its top at y.
func heart(dst *ebiten.Image, x, y float64, col color.RGBA) {
	rows := [4]string{".#.#.", "#####", ".###.", "..#.."}
	x0, y0 := math.Round(x-2.5), math.Round(y)
	for r, row := range rows {
		for i, ch := range row {
			if ch == '#' {
				vector.FillRect(dst, float32(x0+float64(i)), float32(y0+float64(r)), 1, 1, col, false)
			}
		}
	}
}

// heartFX is a heart floating up from a beast being fed.
type heartFX struct {
	x, y float64 // in tiles
	born int     // frame
	big  bool    // tamed: a bigger burst
}

// tameEffects starts hearts and sounds for beasts fed or tamed between two
// snapshots.
func (g *Game) tameEffects(old, cur protocol.Snapshot) {
	was := make(map[sim.CreatureID]protocol.CreatureView, len(old.Creatures))
	for _, c := range old.Creatures {
		was[c.ID] = c
	}
	for _, c := range cur.Creatures {
		o, ok := was[c.ID]
		if !ok {
			continue
		}
		x, y := float64(c.X), float64(c.Y)
		switch {
		case c.Tamed && !o.Tamed:
			for i := range 6 {
				g.hearts = append(g.hearts, heartFX{x + float64(i-3)*0.25, y, g.frame + i*6, true})
			}
			snd.playAt("craft", x, y, 0.8, 0)
			snd.playAt("bellow", x, y, 0.4, 0.1)
		case c.Tame > o.Tame && !c.Tamed:
			g.hearts = append(g.hearts, heartFX{x, y, g.frame, false})
			snd.playAt("eat", x, y, 0.5, 0.1)
		}
	}
}

// drawHearts floats the feeding hearts up and fades them.
func (g *Game) drawHearts(dst *ebiten.Image, cam camera) {
	const life = 70
	live := g.hearts[:0]
	for _, h := range g.hearts {
		age := g.frame - h.born
		if age >= life {
			continue
		}
		live = append(live, h)
		if age < 0 {
			continue // not yet
		}
		sx, sy := cam.toScreen(h.x, h.y)
		f := float64(age) / life
		a := uint8(255 * (1 - f))
		rise := 22 + f*18
		if h.big {
			rise += 8
		}
		heart(dst, sx+math.Sin(f*6+h.x)*2, sy-rise, color.RGBA{0xe0, 0x4a, 0x5a, a})
	}
	g.hearts = live
}

// iconSaddle is a leather saddle with a girth strap.
func iconSaddle() *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{2, 6, 5, 7.5, 11, 7.5, 14, 5.5, 13.5, 9.5, 11, 11.5, 5, 11.5, 2.5, 9.5},
		func(x, y int) color.RGBA { return saddleRamp.pick(0.85-float64(y)*0.05, x, y) })
	c.polygon([]float64{3.5, 4, 6, 6, 5, 7.5, 2, 6}, func(x, y int) color.RGBA { return saddleRamp.pick(0.7, x, y) }) // the raised back
	c.line(8, 11, 8, 15, func(int, int, float64) color.RGBA { return saddleRamp[0] })                                 // the girth
	c.set(8, 15, hornRamp[1])
	c.outline(0.8)
	return c
}

// mountHint is what the R key would do for me right now, if anything, for
// the help panel: the verb, and whether it can be done.
func (g *Game) mountHint() (string, bool, bool) {
	me, ok := g.me()
	if !ok || me.Dead {
		return "", false, false
	}
	if me.Mount != 0 {
		return "xuống thú", true, true
	}
	for _, c := range g.curr.Creatures {
		if c.Owner != g.myID || c.Ridden || !c.Kind.Rideable() {
			continue
		}
		if math.Hypot(float64(c.X-me.X), float64(c.Y-me.Y)) <= sim.MountReach {
			if g.curr.Self.Inv.Count(sim.ItemSaddle) == 0 {
				return "cưỡi " + c.Kind.Name() + " (cần Yên da)", false, true
			}
			return "cưỡi " + c.Kind.Name(), true, true
		}
	}
	return "", false, false
}

// playerName is the name of player id as last seen, or a stand-in.
func (g *Game) playerName(id sim.PlayerID) string {
	for _, p := range g.curr.Players {
		if p.ID == id {
			return p.Name
		}
	}
	return "người khác"
}

// markInset is how far below the top of its sprite a pet's head or back
// begins, in design pixels, so the heart sits just over it.
func markInset(k sim.CreatureKind) float64 {
	switch k {
	case sim.CreatureToro:
		return 2
	case sim.CreatureOrnitho, sim.CreatureAnzu:
		return 1
	case sim.CreatureThesc:
		return 5
	case sim.CreatureAchero:
		return 3
	case sim.CreatureTrike:
		return 2
	case sim.CreatureStego:
		return 3
	case sim.CreatureBrachio:
		return 3
	case sim.CreatureRaptor:
		return 3
	case sim.CreatureMammoth:
		return 9
	case sim.CreatureSabre:
		return 4
	}
	return 0
}
