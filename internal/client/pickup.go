package client

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Picking up is E's, never the left button's or Space's (see sim/pickup.go):
// the pile under the pointer, else the loose thing under it (a branch, a
// pebble…), else the nearest pile, else the nearest loose thing. It is
// marked with corners and named apart from what the left button works on;
// held down, E goes on picking up one thing after another.

// pickState is what E picks up now, and E held down.
type pickState struct {
	view targetView // what E picks up, for the corners, the label and the hint
	held bool       // E went down on something to pick up and is still down
	next time.Time  // when held E picks up the next thing

	cornerKey labelKey // what the corners are on, and since when: they close in on a new one
	cornerAt  int
}

// colPickCorner marks what E picks up: paler and greener than the left
// button's corners, so the two read apart in a fight among the spoils.
var colPickCorner = color.RGBA{204, 240, 182, 255}

// drawPickCues marks what E would pick up with soft corners, as drawAimCues
// marks the left button's target: full on the one under the pointer, faint
// on the nearest.
func (g *Game) drawPickCues(screen *ebiten.Image, cam camera, scale float64) {
	pk := g.pick.view
	if !g.ePicks(g.hudTarget, pk) {
		return // nothing, or E opens or rides what is pointed at first
	}
	x, y, w, h, ok := g.targetBox(pk, cam)
	if !ok {
		return
	}
	if key := targetKey(pk); key != g.pick.cornerKey || g.frame < g.pick.cornerAt {
		g.pick.cornerKey, g.pick.cornerAt = key, g.frame
	}
	strength := 1.0
	if !pk.aimed {
		strength = 0.45
	}
	drawCorners(screen, x*scale, y*scale, w*scale, h*scale, colPickCorner, g.frame-g.pick.cornerAt, g.frame, strength)
}

// pickRepeat is how often E held down picks up the next thing: the server's
// pick cooldown. The first repeat waits as long, so a tap takes one thing.
const pickRepeat = sim.PickCooldown * sim.TickDuration

// pickTarget is what E would pick up right now, by sim.PickUpFrom; ok is
// false when there is nothing (aboard or astride: nothing pointed at, as E
// then gets off).
func (g *Game) pickTarget() targetView {
	p, ok := g.me()
	if g.curr.Self.Dead || !ok || p.Asleep {
		return targetView{}
	}
	ax, ay, aim := g.aim()
	aim = aim && g.overlay == overlayNone && !g.build.on
	x, y := float64(p.X), float64(p.Y)
	drops := make([]sim.DropCandidate, 0, len(g.curr.Drops))
	for _, d := range g.curr.Drops {
		if math.Abs(float64(d.X)-x) <= sim.DropReach && math.Abs(float64(d.Y)-y) <= sim.DropReach {
			drops = append(drops, sim.DropCandidate{ID: d.ID, X: float64(d.X), Y: float64(d.Y)})
		}
	}
	k, ok := sim.PickUpFrom(x, y, aim, ax, ay, p.Mount != 0 || p.Boat != 0, drops, g.candidateAt)
	if !ok {
		return targetView{}
	}
	if k.Drop != 0 {
		for _, d := range g.curr.Drops {
			if d.ID == k.Drop {
				return targetView{ok: true, drop: true, d: d, aimed: k.Aimed}
			}
		}
		return targetView{}
	}
	tv, ok := g.viewOf(sim.Target{Entity: k.Entity})
	if !ok {
		return targetView{}
	}
	tv.aimed = k.Aimed
	return tv
}

// labelled is what the name under the pointer is of: a beast the left
// button would strike there, else what E would pick up there, else tg.
func (g *Game) labelled(tg targetView) targetView {
	if pk := g.pick.view; pk.ok && pk.aimed && !(tg.ok && tg.aimed && tg.creature) {
		return pk
	}
	return tg
}

// pickHint is the hint over the hotbar for E picking pk up, when that is
// what E would do and there is no beast to fight first: one under the
// pointer, or one near when nothing to pick up is pointed at.
func (g *Game) pickHint(tg, pk targetView) (string, bool) {
	fight := tg.ok && tg.creature && !tg.c.Tamed && (tg.aimed || !pk.aimed)
	if fight || !g.ePicks(tg, pk) {
		return "", false
	}
	return "nhặt " + pickName(pk), true
}

// ePicks reports whether E pressed now would pick pk up, by the order of
// the server's interactKey: what E picks up under the pointer first, then
// a chest, bed, body or station, or a beast of mine to get on, under the
// pointer, then the nearest pile or loose thing. tg is the left button's
// target, which is what is under the pointer when tg.aimed.
func (g *Game) ePicks(tg, pk targetView) bool {
	if !pk.ok {
		return false
	}
	if pk.aimed || !tg.ok || !tg.aimed {
		return true
	}
	if tg.creature {
		return !(tg.c.Owner == g.myID && tg.c.Kind.Rideable())
	}
	switch tg.e.Kind {
	case sim.EntityChest, sim.EntityGrave, sim.EntityTreasure, sim.EntityBed, sim.EntityLeafTent:
		return false
	}
	return sim.StationOf(tg.e.Kind, tg.e.Var) == sim.StationNone
}

// pickName is what E picks up in tv, in lower case, with how many in a pile.
func pickName(tv targetView) string {
	if tv.drop {
		s := strings.ToLower(tv.d.Item.Def().Name)
		if tv.d.Count > 1 {
			s += fmt.Sprintf(" ×%d", tv.d.Count)
		}
		return s
	}
	return strings.ToLower(sim.EntityLabel(tv.e.Kind, tv.e.Var))
}

// updatePickHold sends CmdPickUp every pickRepeat while E, pressed on
// something to pick up, is held down. pressed says E went down this frame
// (and sent CmdInteract).
func (g *Game) updatePickHold(pressed bool) {
	switch {
	case pressed:
		g.pick.held = g.ePicks(g.hudTarget, g.pickTarget())
		g.pick.next = time.Now().Add(pickRepeat)
		return
	case !g.pick.held || !ebiten.IsKeyPressed(ebiten.KeyE):
		g.pick.held = false
		return
	case time.Now().Before(g.pick.next):
		return
	}
	g.pick.next = time.Now().Add(pickRepeat)
	if g.pickTarget().ok {
		g.command(sim.Command{Kind: sim.CmdPickUp}) // see sim/pickup.go
	}
}

// pickedUp reports whether p, starting to act between old and cur, picked
// something up with E rather than swung at anything: a pile in their reach
// went or shrank (not one they walked over), or a loose thing in their reach
// was taken. Their swing is then a crouch to the ground, with no sound of a
// blow, whatever beast or tree stands by.
func pickedUp(old, cur protocol.Snapshot, p protocol.PlayerView) bool {
	x, y := float64(p.X), float64(p.Y)
	const slack = 0.3 // they may have moved a little
	in := func(px, py, r float64) bool {
		dx, dy := px-x, py-y
		return dx*dx+dy*dy <= (r+slack)*(r+slack)
	}
	left := make(map[sim.DropID]uint8, len(cur.Drops))
	for _, d := range cur.Drops {
		left[d.ID] = d.Count
	}
	for _, d := range old.Drops {
		dx, dy := float64(d.X), float64(d.Y)
		if n, still := left[d.ID]; (still && n >= d.Count) || !in(dx, dy, sim.DropReach) {
			continue
		}
		walked := math.Abs(dx-x) <= sim.DropTouch && math.Abs(dy-y) <= sim.DropTouch && d.Age >= sim.DropPickupDelay
		if !walked {
			return true
		}
	}
	ready := make(map[sim.EntityID]bool, len(cur.Entities))
	for _, e := range cur.Entities {
		ready[e.ID] = e.Ready
	}
	for _, e := range old.Entities {
		if !sim.LooseItem(e.Kind) || !e.Ready {
			continue
		}
		if r, seen := ready[e.ID]; seen && !r && in(float64(e.TX)+0.5, float64(e.TY)+0.5, sim.InteractReach) {
			return true
		}
	}
	return false
}
