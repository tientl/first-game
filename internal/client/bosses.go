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
	return [numBosses]float64{60, 56, 66, 54, 62, 0}[i]
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
		case sim.CreatureSpino:
			callAt("roar", x, y, 0.95, 1.12, roarReach)
			callAt("croc_hiss", x, y, 0.6, 0.8, 24)
		case sim.CreatureMammothKing:
			callAt("trumpet", x, y, 1, 0.72, 30)
		case sim.CreatureCarcharo:
			callAt("roar", x, y, 1, 1.06, roarReach)
		case sim.CreatureGiganoto:
			callAt("roar", x, y, 1, 0.82, roarReach)
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
			callAt("trumpet", x, y, 0.9, 0.8, 24)
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
