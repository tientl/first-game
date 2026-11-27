package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The six bosses on screen: their sprites (from bossart.go, mirrored for
// facing left), labels, shadows and voices. The bar, the victory screen and
// the map marks are in bossui.go.

// bossSprites holds every pose of every boss; the last index is 0 facing
// right, 1 facing left.
type bossSprites [numBosses][numBossPoses][2]sprite

func genBossSprites() (s bossSprites) {
	for b := range numBosses {
		if bossID(b) == bossTitanoboa || bossID(b) == bossMegalodon || bossID(b) == bossMammothKing {
			continue // drawn at HDDensity, frame by frame as they are shown: see bossHDSprite
		}
		for p := range numBossPoses {
			c, ax, ay := bossCanvas(bossID(b), bossPose(p))
			s[b][p][0], s[b][p][1] = sidedPair(mk(c, ax, ay))
		}
	}
	return s
}

// bossItemIcons puts the legendary items' bag icons in.
func bossItemIcons(icons *[sim.NumItems]*ebiten.Image) {
	for b := range numBosses {
		icons[sim.LegendaryItem(sim.BossKind(int(b)))] = bossItemCanvas(bossID(b)).image()
	}
}

// bossPoseOf is the pose boss c shows this frame.
func bossPoseOf(c protocol.CreatureView, moving bool, t int) bossPose {
	switch c.State {
	case sim.CreatureRoar:
		return bossRoar
	case sim.CreatureStrike, sim.CreatureClaw:
		return bossAttack
	case sim.CreatureSpecial, sim.CreatureGrab:
		return bossSpecial
	case sim.CreatureLurk:
		return bossLurk
	}
	if moving {
		period := 20
		if c.State == sim.CreatureChase {
			period = 12
		}
		if (t/period)%2 == 0 {
			return bossWalkA
		}
		return bossWalkB
	}
	return bossIdle
}

// bossSprite is creatureSprite for the bosses.
func (g *Game) bossSprite(c protocol.CreatureView, moving bool) (sprite, float64, bool) {
	i, ok := sim.BossIndex(c.Kind)
	if !ok || g.art == nil {
		return sprite{}, 0, false
	}
	if s, ok := g.bossHDSprite(c, moving); ok {
		return s, 0, true
	}
	side := 0
	if c.FaceLeft {
		side = 1
	}
	return g.art.bosses[i][bossPoseOf(c, moving, g.frame+int(c.ID)*7)][side], 0, true
}

// bossShadow is the shadow width under boss k; 0 for anything else.
func bossShadow(k sim.CreatureKind) float64 {
	i, ok := sim.BossIndex(k)
	if !ok {
		return 0
	}
	return [numBosses]float64{0, 56, 108, 54, 62, 0}[i] // Titanoboa's shadow is in its drawing
}

// bossHDSprite is the frame of Titanoboa, the mammoth king or the Megalodon to show for c:
// each state of theirs (see internal/sim/bossfight.go) has its own run of
// frames, timed from the moment the state began.
func (g *Game) bossHDSprite(c protocol.CreatureView, moving bool) (sprite, bool) {
	left := c.FaceLeft
	t := g.frame + int(c.ID)*7
	age, prev := g.stateAge(c)
	if bossAgeShot >= 0 {
		t = bossAgeShot
	}
	if s, ok := g.bossSkillSprite(c, t, left); ok { // its moves in its temple or at its atoll: see lairfight.go
		return s, true
	}
	switch c.Kind {
	case sim.CreatureTitanoboa:
		hdWarm(titanIdle, titanRoar, titanSlither, titanStrike, titanGrab)
		switch c.State {
		case sim.CreatureRoar: // reared, hissing
			return titanRoar.sprite(t/6, left), true
		case sim.CreatureStrike, sim.CreatureClaw: // drawn back, then the lunge
			return titanStrike.sprite(titanStrikeFrame(age), left), true
		case sim.CreatureGrab, sim.CreatureSpecial: // its coils round its prey
			return titanGrab.sprite(t/6, left), true
		case sim.CreatureChase:
			if moving {
				return titanSlither.sprite(t/3, left), true
			}
			return titanSlither.sprite(t/7, left), true // gathering itself, in reach
		}
		if moving { // going back to its lair
			return titanSlither.sprite(t/5, left), true
		}
		return titanIdle.sprite(t/13, left), true // coiled in the undergrowth
	case sim.CreatureMammothKing:
		return kingSprite(c, moving, t, age, left), true
	case sim.CreatureMegalodon:
		hdWarm(megaLurk, megaSwim, megaSurface, megaStrike, megaBreach, megaSlam, megaRoar)
		switch c.State {
		case sim.CreatureRoar:
			return megaRoar.sprite(t/5, left), true
		case sim.CreatureStrike, sim.CreatureClaw:
			if prev == sim.CreatureLurk { // the ram: it is on them at once
				age += 30
			}
			return megaStrike.sprite(megaFrameAt(megaStrikeAges[:], age), left), true
		case sim.CreatureSpecial, sim.CreatureGrab: // the leap
			return megaBreach.sprite(megaFrameAt(megaBreachAges[:], age), left), true
		case sim.CreatureLurk: // circling, only its fin showing
			return megaLurk.sprite(t/5, left), true
		case sim.CreatureChase: // up at the surface
			if prev == sim.CreatureSpecial && age < megaSlamN*megaSlamAge {
				return megaSlam.sprite(age/megaSlamAge, left), true // it has just come down
			}
			return megaSurface.sprite(t/5, left), true
		}
		return megaSwim.sprite(t/7, left), true // going back to its deep
	}
	return sprite{}, false
}

// bossPortrait is a picture of boss i for the screens that list them.
func (g *Game) bossPortrait(i int, pose bossPose) sprite {
	switch bossID(i) {
	case bossTitanoboa:
		return titanRoar.sprite(0, false)
	case bossMegalodon:
		return megaBreach.sprite(5, false)
	case bossMammothKing:
		return kingRoar.sprite(0, false)
	}
	return g.art.bosses[i][pose][0]
}

// bossName is a boss's name and epithet: "Spinosaurus — Chúa đầm lầy".
func bossName(k sim.CreatureKind) string { return k.Name() + " — " + sim.BossEpithet(k) }

// bossLabel is the label of a boss pointed at.
func bossLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	if !c.Kind.Boss() {
		return "", colWhite, false
	}
	n := bossName(c.Kind)
	switch c.State {
	case sim.CreatureGrab:
		return n + " — đang siết! giữ phím di chuyển để vùng ra", colBad, true
	case sim.CreatureSpecial:
		return n + " — coi chừng, lùi lại!", colBad, true
	}
	return n + " (trùm!)", colBad, true
}

// bossSound plays what a boss did between two snapshots.
func bossSound(o, c protocol.CreatureView) bool {
	if !c.Kind.Boss() {
		return false
	}
	x, y := float64(c.X), float64(c.Y)
	near := func() float64 { return math.Hypot(x-snd.lx, (y-snd.ly)*1.3) }
	switch {
	case c.State == sim.CreatureRoar: // a fight begins, or its rage
		switch c.Kind {
		case sim.CreatureTitanoboa:
			callAt("hiss", x, y, 1, 0.5, 28)
			callAt("croc_hiss", x, y, 0.8, 0.7, 28)
		case sim.CreatureSpino: // a crocodile's roar with its hiss in it
			callAt("spino_roar", x, y, 0.95, 1, roarReach)
		case sim.CreatureMammothKing:
			callAt("mammothking_roar", x, y, 1, 1, 30)
		case sim.CreatureCarcharo:
			callAt("carcharo_roar", x, y, 1, 1, roarReach)
		case sim.CreatureGiganoto:
			callAt("giganoto_roar", x, y, 1, 1, roarReach)
		case sim.CreatureMegalodon:
			callAt("splash", x, y, 1, 0.6, 24)
		}
		roarShake(near())
	case c.State == sim.CreatureStrike:
		snd.playAt("chomp", x, y, 1, 0.05)
	case c.State == sim.CreatureGrab:
		snd.playAt("hiss", x, y, 0.8, 0.1)
	case c.State == sim.CreatureSpecial:
		if c.Kind == sim.CreatureMammothKing {
			callAt("mammothking_special", x, y, 0.9, 1, 24)
		} else {
			snd.playAt("splash", x, y, 0.5, 0.1)
		}
	case o.State == sim.CreatureSpecial: // the blow lands
		snd.playAt("stomp", x, y, 1, 0.05)
		roarShake(near() + 4)
	case o.State == sim.CreatureLurk || c.State == sim.CreatureLurk:
		snd.playAt("splash", x, y, 0.6, 0.1)
	}
	return true
}
