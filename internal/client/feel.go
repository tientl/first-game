package client

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// How a fight feels, as the client shows it; none of it touches the server.
//
//   - Hit-stop: a blow that lands freezes the picture for 2 to 5 frames,
//     longer for a harder blow, a critical one or a heavy weapon, and when
//     the player is struck. The world drawn at the moment of the hit is kept
//     and shown again while the stop lasts (hitStop); the client goes on
//     sending input and taking snapshots all the while, and catches up the
//     moment it ends, so the simulation never waits for it.
//   - Flash: a struck beast's sprite flashes white, then red (hitFlash); a
//     struck player the same, then a red tint (jolt.go).
//   - Screen shake, by trauma: every heavy blow, critical, big creature's
//     blow on the player, stomp and roar adds trauma, which drains away; the
//     view is thrown by trauma squared along smooth noise. Off with the
//     "screen shake" setting (SetScreenShake), kept in the settings file.
//   - Effects (fx.go) at the blow: sparks, crits, claw marks, the thrust's
//     flare, the cleave's embers, a splash in the water, dust and leaves.

// --- screen shake --------------------------------------------------------------

// screenShake is the "screen shake" setting: on unless turned off.
var screenShake = true

// SetScreenShake turns screen shake on or off, and keeps the choice.
func SetScreenShake(on bool) {
	screenShake = on
	snd.saveShake(on)
}

// ScreenShakeOn reports the screen shake setting.
func ScreenShakeOn() bool { return screenShake }

// Shake tuning.
const (
	shakeMax   = 0.3  // tiles the view is thrown at full trauma
	shakeDecay = 1.5  // trauma drained a second
	shakeReach = 14.0 // tiles: a roar further off than this shakes nothing
)

var shakeState struct {
	trauma   float64
	at       time.Time // when trauma was last drained
	hold     float64   // a rumble kept up to this: a roar
	holdFrom time.Time
	holdTill time.Time
}

// addTrauma shakes the view by t more, 0 to 1.
func addTrauma(t float64) {
	drainTrauma(time.Now())
	shakeState.trauma = min(1, shakeState.trauma+t)
}

// holdTrauma keeps the shake at level at least, for d, rising quickly and
// dying away over its last second.
func holdTrauma(level float64, d time.Duration) {
	now := time.Now()
	if now.Before(shakeState.holdTill) && level <= shakeState.hold {
		return
	}
	shakeState.hold, shakeState.holdFrom, shakeState.holdTill = level, now, now.Add(d)
}

func drainTrauma(now time.Time) {
	if !shakeState.at.IsZero() {
		shakeState.trauma = max(0, shakeState.trauma-shakeDecay*now.Sub(shakeState.at).Seconds())
	}
	shakeState.at = now
}

// roarShake shakes the view for a roar d tiles off: it rumbles as long as
// the roar does.
func roarShake(d float64) {
	if d >= shakeReach {
		return
	}
	holdTrauma(0.8*(1-d/shakeReach), 1800*time.Millisecond)
}

// traumaNow is how shaken the view is now, 0 to 1.
func traumaNow() float64 {
	now := time.Now()
	drainTrauma(now)
	t := shakeState.trauma
	if left := shakeState.holdTill.Sub(now).Seconds(); left > 0 {
		in := min(1, now.Sub(shakeState.holdFrom).Seconds()/0.25)
		t = max(t, shakeState.hold*min(1, left/1.2)*in)
	}
	return t
}

// shake is how far the view is thrown now, in tiles: nothing with the
// setting off.
func shake(frame int) (dx, dy float64) {
	if !screenShake {
		return 0, 0
	}
	t := traumaNow()
	if t <= 0.001 {
		return 0, 0
	}
	a := shakeMax * t * t
	s := float64(time.Now().UnixNano()%1e12) / 1e9
	nx := math.Sin(s*31.7) + 0.6*math.Sin(s*53.3+1.7) + 0.3*math.Sin(s*89.1+0.3)
	ny := math.Sin(s*29.1+2.1) + 0.6*math.Sin(s*61.9+0.4) + 0.3*math.Sin(s*97.3+1.1)
	return a * nx / 1.9, a * 0.75 * ny / 1.9
}

// --- hit-stop --------------------------------------------------------------------

// hitStop is the picture held still after a blow.
type hitStop struct {
	left    int           // frames still to hold it
	capture bool          // the next frame drawn is the one held
	img     *ebiten.Image // the world as it was drawn then
	total   int           // stops so far, for tests
}

// Hit-stop tuning, in frames at 60 a second.
const (
	stopMin = 2
	stopMax = 5
)

// stopFor holds the picture for frames frames from the next one drawn.
func (g *Game) stopFor(frames int) {
	frames = max(stopMin, min(stopMax, frames))
	if frames <= g.stop.left {
		return
	}
	if g.stop.left == 0 {
		g.stop.capture = true
	}
	g.stop.left = frames
	g.stop.total++
}

// tickStop counts the hold down, once a game frame. It never holds up
// Update: input and snapshots go on as ever.
func (g *Game) tickStop() {
	if g.stop.left > 0 && !g.stop.capture {
		g.stop.left--
	}
}

// frozen reports whether this frame shows the held picture.
func (g *Game) frozen() bool { return g.stop.left > 0 && !g.stop.capture && g.stop.img != nil }

// keepStop keeps the world drawn on src as the held picture.
func (g *Game) keepStop(src *ebiten.Image) {
	if !g.stop.capture {
		return
	}
	g.stop.capture = false
	b := src.Bounds()
	if g.stop.img == nil || g.stop.img.Bounds() != b {
		if g.stop.img != nil {
			g.stop.img.Deallocate()
		}
		g.stop.img = ebiten.NewImage(b.Dx(), b.Dy())
	}
	g.stop.img.Clear()
	g.stop.img.DrawImage(src, nil)
}

// drawStopped puts the held picture on screen, thrown by the shake.
func (g *Game) drawStopped(screen *ebiten.Image, scale int) {
	op := &ebiten.DrawImageOptions{}
	sx, sy := shake(g.frame)
	op.GeoM.Translate(math.Round(-sx*TileW*float64(scale)), math.Round(-sy*TileH*float64(scale)))
	screen.DrawImage(g.stop.img, op)
}

// blowStop is how many frames a blow of dmg (of a beast's max) holds the
// picture: heavier, critical and killing blows longer.
func blowStop(dmg int, held sim.ItemKind, crit, kill bool) int {
	n := stopMin
	if dmg >= 25 {
		n++
	}
	if held == sim.ItemBattleAxe || held == sim.ItemIvorySpear {
		n++
	}
	if crit {
		n++
	}
	if kill {
		n++
	}
	return min(stopMax, n)
}

// --- the flash on a struck beast ---------------------------------------------------

// flashFrames is how long a struck beast flashes.
const flashFrames = 9

// hitFlash is the colour a struck thing flashes age frames after the blow:
// white for two frames, then red, fading. a is 0 when it is over.
func hitFlash(age int) (r, g, b, a float64) {
	switch {
	case age < 0:
		return 0, 0, 0, 0
	case age < 2:
		return 1, 1, 1, 0.9
	case age < 4:
		return 1, 0.25, 0.18, 0.55
	case age < flashFrames:
		return 1, 0.22, 0.16, 0.3 * float64(flashFrames-age) / float64(flashFrames-4)
	}
	return 0, 0, 0, 0
}

// drawFlash lays sprite s, placed by op, over itself in one flat colour.
func drawFlash(dst drawTarget, s sprite, op *ebiten.DrawImageOptions, r, g, b, a float64) {
	if a <= 0 {
		return
	}
	var cm colorm.ColorM
	cm.Scale(0, 0, 0, a)
	cm.Translate(r, g, b, 0)
	cop := &colorm.DrawImageOptions{}
	cop.GeoM = op.GeoM
	drawColorMD(dst, s.img, cm, cop, s.density())
}

// beastFlash is how c flashes now, if it was struck lately.
func (g *Game) beastFlash(id sim.CreatureID) (r, gg, b, a float64) {
	at, ok := g.flashAt[id]
	if !ok {
		return 0, 0, 0, 0
	}
	age := g.frame - at
	if age >= flashFrames {
		delete(g.flashAt, id)
		return 0, 0, 0, 0
	}
	return hitFlash(age)
}

// --- what happened since the last snapshot -------------------------------------------

// noteFeel reads the blows of the new snapshot s against the one before:
// beasts struck (and by whom), players struck, blows taken on shields; and
// starts their flashes, hit-stops, shakes and effects.
func (g *Game) noteFeel(old, s protocol.Snapshot) {
	if g.flashAt == nil {
		g.flashAt = make(map[sim.CreatureID]int)
	}
	me, haveMe := playerIn(s, g.myID)
	crit := s.Self.Notice == sim.NoticeCrit && s.Self.NoticeSeq != g.noticeSeq
	bitten := s.Self.Notice == sim.NoticeBitten && s.Self.NoticeSeq != g.noticeSeq
	myBlow := haveMe && me.Acting && g.since(g.swingAt[g.myID]) < 14
	seen := make(map[sim.CreatureID]bool, len(s.Creatures))
	for _, c := range s.Creatures {
		seen[c.ID] = true
		q, ok := g.prevCreatures[c.ID]
		if !ok || c.HP >= q.HP || !shown(c) {
			continue
		}
		g.beastStruck(c, int(q.HP-c.HP), false, haveMe, me, myBlow, crit)
	}
	for id, q := range g.prevCreatures {
		if !seen[id] && q.HP < q.MaxHP && shown(q) && haveMe && math.Hypot(float64(q.X-me.X), float64(q.Y-me.Y)) < 4 {
			g.beastStruck(q, int(q.HP), true, haveMe, me, myBlow, crit) // the killing blow
		}
	}
	for _, p := range s.Players {
		q, ok := g.prevPlayers[p.ID]
		if !ok || p.Dead {
			continue
		}
		if p.GuardSeq != q.GuardSeq {
			g.shieldStruck(p)
		}
		if p.HurtSeq != q.HurtSeq {
			g.playerStruck(s, p, p.ID == g.myID && bitten)
		}
	}
	g.noteDust(old, s)
}

// playerIn finds player id in s.
func playerIn(s protocol.Snapshot, id sim.PlayerID) (protocol.PlayerView, bool) {
	for _, p := range s.Players {
		if p.ID == id {
			return p, true
		}
	}
	return protocol.PlayerView{}, false
}

// beastStruck is c losing dmg health: its flash, and if the blow was mine,
// the stop, the shake and the effect of the weapon.
func (g *Game) beastStruck(c protocol.CreatureView, dmg int, kill, haveMe bool, me protocol.PlayerView, myBlow, crit bool) {
	g.flashAt[c.ID] = g.frame
	x, y := float64(c.X), float64(c.Y)
	z := beastHeight(c.Kind) * TileH * 0.55
	mine := myBlow && math.Hypot(x-float64(me.X), y-float64(me.Y)) < 4.5
	ang := 0.0
	if haveMe {
		ang = math.Atan2((y-float64(me.Y))*TileH, (x-float64(me.X))*TileW)
	}
	k := math.Min(1.4, 0.75+beastHeight(c.Kind)*0.15)
	if g.tileAt(int(math.Floor(x)), int(math.Floor(y))).Swimmable() && !crocLike(c.Kind) {
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 0.6 * k})
	}
	if !mine {
		g.spawnFX(fxInst{kind: fxSpark, x: x, y: y, z: z, ang: ang, k: 0.6})
		return
	}
	held := me.Held
	switch {
	case crit:
		g.spawnFX(fxInst{kind: fxCrit, x: x, y: y, z: z, ang: ang, k: k})
	default:
		g.spawnFX(fxInst{kind: fxSpark, x: x, y: y, z: z, ang: ang, k: k * 0.85})
	}
	switch {
	case held == sim.ItemSpear:
		g.spawnFX(fxInst{kind: fxThrust, x: x - math.Cos(ang)*0.25, y: y, z: z, ang: ang, k: 0.9})
	case held == sim.ItemIvorySpear:
		g.spawnFX(fxInst{kind: fxSweep, x: float64(me.X), y: float64(me.Y), z: 0, ang: ang, k: 1})
		g.spawnFX(fxInst{kind: fxThrust, x: x - math.Cos(ang)*0.25, y: y, z: z, ang: ang, k: 1})
	case held == sim.ItemBattleAxe:
		g.spawnFX(fxInst{kind: fxCleave, x: x - math.Cos(ang)*0.35, y: y, z: 0, ang: ang, k: 1})
		g.spawnFX(fxInst{kind: fxStomp, x: x, y: y, k: 0.6, pal: g.dustAt(x, y)})
	}
	g.spawnFX(fxInst{kind: fxBits, x: x, y: y, z: z, ang: ang, k: 1, pal: g.dustAt(x, y)})
	if g.leafyAt(x, y) {
		g.spawnFX(fxInst{kind: fxLeaves, x: x, y: y, z: z + 4, k: 1})
	}
	g.stopFor(blowStop(dmg, held, crit, kill))
	switch {
	case crit:
		addTrauma(0.38)
	case held == sim.ItemBattleAxe:
		addTrauma(0.34)
	case dmg >= 40:
		addTrauma(0.22)
	}
}

// clawers are the beasts whose blow is a rake of claws.
func clawer(k sim.CreatureKind) bool {
	switch k {
	case sim.CreatureRaptor, sim.CreatureAchero, sim.CreatureBear, sim.CreatureSabre, sim.CreatureMonkey,
		sim.CreatureTroodon, sim.CreatureAnzu:
		return true
	}
	return false
}

// bigBlow reports whether a blow of k's shakes the ground.
func bigBlow(k sim.CreatureKind) bool {
	switch k {
	case sim.CreatureRex, sim.CreatureTrike, sim.CreatureToro, sim.CreatureMammoth, sim.CreatureStego,
		sim.CreatureAnkylo, sim.CreatureBrachio, sim.CreatureMosa, sim.CreatureKraken:
		return true
	}
	return k.Boss()
}

// playerStruck is p jolted by a blow: the lines round them, the claws or
// the bite of what struck them, and for me the stop and the shake.
func (g *Game) playerStruck(s protocol.Snapshot, p protocol.PlayerView, bitten bool) {
	x, y := float64(p.X), float64(p.Y)
	z := float64(playerArtH) * 0.55
	mine := p.ID == g.myID
	kind := sim.CreatureNone
	if mine && bitten {
		kind = sim.CreatureKind(s.Self.NoticeArg)
	}
	// Where the blow came from: the nearest beast of that kind, else the
	// nearest beast.
	ang := -math.Pi / 2
	best := 3.5
	for _, c := range s.Creatures {
		if kind != sim.CreatureNone && c.Kind != kind {
			continue
		}
		if d := math.Hypot(float64(c.X)-x, float64(c.Y)-y); d < best {
			best = d
			ang = math.Atan2((y-float64(c.Y))*TileH, (x-float64(c.X))*TileW)
			if kind == sim.CreatureNone {
				kind = c.Kind
			}
		}
	}
	g.spawnFX(fxInst{kind: fxLines, x: x, y: y, z: z, k: 1})
	switch {
	case p.Swimming:
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 0.8})
	case clawer(kind):
		rake := ang + 0.9 // claws rake down across the blow
		if math.Cos(ang) < 0 {
			rake = ang - 0.9
		}
		g.spawnFX(fxInst{kind: fxClaw, x: x, y: y, z: z + 2, ang: rake, k: 1})
	default:
		g.spawnFX(fxInst{kind: fxSpark, x: x, y: y, z: z, ang: ang, k: 0.7})
	}
	if kind == sim.CreatureMegalodon || kind == sim.CreatureMosa {
		g.spawnFX(fxInst{kind: fxSplash, x: x, y: y, k: 1.8})
	}
	if !mine {
		return
	}
	n := 3
	if bigBlow(kind) {
		n += 2
		addTrauma(0.45)
	} else {
		addTrauma(0.16)
	}
	g.stopFor(n)
}

// shieldStruck is a blow taken on p's raised shield: sparks and splinters
// off its face (a crossed flash for a parry), the knock of it, and for me a
// short stop.
func (g *Game) shieldStruck(p protocol.PlayerView) {
	dx, dy := p.Dir.Delta()
	x, y := float64(p.X)+float64(dx)*0.42, float64(p.Y)+float64(dy)*0.3
	ang := math.Atan2(float64(dy)*TileH, float64(dx)*TileW)
	z := float64(playerArtH) * 0.5
	if p.Parry {
		g.spawnFX(fxInst{kind: fxParry, x: x, y: y, z: z, ang: ang, k: 1})
		snd.playAt("shield_parry", x, y, 0.9, 0.06)
	} else {
		g.spawnFX(fxInst{kind: fxBlock, x: x, y: y, z: z, ang: ang, k: 1})
		snd.playAt("shield_block", x, y, 0.9, 0.08)
	}
	if p.ID != g.myID {
		return
	}
	if p.Parry {
		g.stopFor(4)
		addTrauma(0.2)
	} else {
		g.stopFor(2)
		addTrauma(0.12)
	}
}

// --- dust ---------------------------------------------------------------------------------

// dustAt is the dust that rises from the ground at (x, y).
func (g *Game) dustAt(x, y float64) int {
	switch g.tileAt(int(math.Floor(x)), int(math.Floor(y))) {
	case sim.TileSand, sim.TileDesert:
		return dustSand
	case sim.TileSnow, sim.TileIce:
		return dustSnow
	case sim.TileAsh, sim.TileLava:
		return dustAsh
	case sim.TileGrass, sim.TileForest, sim.TileMarsh:
		return dustGrass
	}
	return dustEarth
}

// leafyAt reports whether a blow at (x, y) shakes leaves loose: in the woods.
func (g *Game) leafyAt(x, y float64) bool {
	return g.tileAt(int(math.Floor(x)), int(math.Floor(y))) == sim.TileForest
}

// sandy reports whether feet on (x, y) kick up dust.
func (g *Game) sandy(x, y float64) bool {
	switch g.tileAt(int(math.Floor(x)), int(math.Floor(y))) {
	case sim.TileSand, sim.TileDesert, sim.TileAsh:
		return true
	}
	return false
}

// stepEvery is how far apart, in tiles, the footsteps that raise dust are.
const stepEvery = 0.55

// noteDust raises dust behind charging beasts and under feet on sand.
func (g *Game) noteDust(old, s protocol.Snapshot) {
	if g.walked == nil {
		g.walked = make(map[sim.PlayerID]float64)
	}
	for _, p := range s.Players {
		q, ok := g.prevPlayers[p.ID]
		if !ok || p.Dead || p.Swimming || p.Boat != 0 || p.Mount != 0 || p.Asleep {
			continue
		}
		d := math.Hypot(float64(p.X-q.X), float64(p.Y-q.Y))
		if d > 1 {
			continue // a jump, not a step
		}
		g.walked[p.ID] += d
		if g.walked[p.ID] < stepEvery {
			continue
		}
		g.walked[p.ID] = 0
		if !g.sandy(float64(p.X), float64(p.Y)) {
			continue
		}
		side := 1.0
		if g.frame/7%2 == 0 {
			side = -1
		}
		ang := math.Atan2(float64(q.Y-p.Y), float64(q.X-p.X)) // blown back along the way they go
		g.spawnFX(fxInst{kind: fxStep, x: float64(p.X) + side*0.08, y: float64(p.Y) + 0.05, ang: ang, k: 1, pal: dustSand})
	}
	for _, c := range s.Creatures {
		q, ok := g.prevCreatures[c.ID]
		if !ok || !bigBlow(c.Kind) || c.Kind.Boss() && c.State == sim.CreatureLurk {
			continue
		}
		d := math.Hypot(float64(c.X-q.X), float64(c.Y-q.Y))
		charging := d > 0.17 && d < 2
		if c.State == sim.CreatureSpecial || charging && (s.Tick+uint64(c.ID))%3 == 0 {
			ang := math.Atan2(float64(q.Y-c.Y), float64(q.X-c.X))
			g.spawnFX(fxInst{kind: fxDust, x: float64(c.X), y: float64(c.Y), ang: ang, k: math.Min(1.6, 0.8+beastHeight(c.Kind)*0.25), pal: g.dustAt(float64(c.X), float64(c.Y))})
		}
		if q.State == sim.CreatureSpecial && c.State != sim.CreatureSpecial && !g.tileAt(int(c.X), int(c.Y)).Swimmable() {
			g.spawnFX(fxInst{kind: fxStomp, x: float64(c.X), y: float64(c.Y), k: 1.6, pal: g.dustAt(float64(c.X), float64(c.Y))})
		}
		if (c.Kind == sim.CreatureMegalodon || c.Kind == sim.CreatureMosa) && c.State != q.State &&
			(c.State == sim.CreatureStrike || q.State == sim.CreatureLurk || c.State == sim.CreatureLurk) {
			g.spawnFX(fxInst{kind: fxSplash, x: float64(c.X), y: float64(c.Y), k: 2.2})
		}
	}
}
