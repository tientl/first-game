package client

import (
	"image/color"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// How the tyrant's neighbours (hellart.go) are drawn, named and heard; the
// hooks are called first from creatureSprite, creatureLabel and
// creatureSounds in creatures.go.

// hellArt holds their sprites; the first index is 0 facing right, 1 left.
type hellArt struct {
	toro       [2][trikePoses]sprite
	achero     [2][raptorPoses]sprite
	brachy     [2][crocPoses]sprite
	champso    [2][crocPoses]sprite
	ornitho    [2][ornithoPoses]sprite
	thesc      [2][thescPoses]sprite
	anzu       [2][anzuPoses]sprite
	quetzal    [2][quetzalPoses]sprite
	quetzalFly [2][pteroPoses]sprite
}

func genHellArt() hellArt {
	var a hellArt
	both := func(gen func(int) sprite, n int, dst func(p int, r, l sprite)) {
		for p := range n {
			r, l := sidedPair(gen(p))
			dst(p, r, l)
		}
	}
	both(genToro, trikePoses, func(p int, r, l sprite) { a.toro[0][p], a.toro[1][p] = r, l })
	both(genAchero, raptorPoses, func(p int, r, l sprite) { a.achero[0][p], a.achero[1][p] = r, l })
	both(genBrachy, crocPoses, func(p int, r, l sprite) { a.brachy[0][p], a.brachy[1][p] = r, l })
	both(genChampso, crocPoses, func(p int, r, l sprite) { a.champso[0][p], a.champso[1][p] = r, l })
	both(genOrnitho, ornithoPoses, func(p int, r, l sprite) { a.ornitho[0][p], a.ornitho[1][p] = r, l })
	both(genThesc, thescPoses, func(p int, r, l sprite) { a.thesc[0][p], a.thesc[1][p] = r, l })
	both(genAnzu, anzuPoses, func(p int, r, l sprite) { a.anzu[0][p], a.anzu[1][p] = r, l })
	both(genQuetzal, quetzalPoses, func(p int, r, l sprite) { a.quetzal[0][p], a.quetzal[1][p] = r, l })
	for p := range pteroPerch { // the flight poses only
		a.quetzalFly[0][p], a.quetzalFly[1][p] = sidedPair(genQuetzalFly(p))
	}
	return a
}

// crocLike reports whether k lurks in the water like the crocodile.
func crocLike(k sim.CreatureKind) bool {
	return k == sim.CreatureCroc || k == sim.CreatureBrachy || k == sim.CreatureChampso
}

// hellSprite is creatureSprite for the tyrant's neighbours; ok is false for
// any other kind.
func (g *Game) hellSprite(c protocol.CreatureView, moving bool) (sprite, float64, bool) {
	a := &g.art.beasts.hell
	side := 0
	if c.FaceLeft {
		side = 1
	}
	t := g.frame + int(c.ID)*7
	switch c.Kind {
	case sim.CreatureToro: // as the three-horn
		switch {
		case feeding(c):
			return a.toro[side][trikeEatA+chew(c, t)], 0, true
		case c.State == sim.CreatureStrike, calling(c):
			return a.toro[side][trikeGore], 0, true
		case c.State == sim.CreatureChase && moving:
			return a.toro[side][trikeCharge], float64((t / 4) % 2), true
		case c.State == sim.CreatureChase:
			return a.toro[side][trikeCharge], 0, true
		case moving:
			return a.toro[side][trikeWalkA+(t/12)%2], 0, true
		}
		return a.toro[side][trikeWalkA], 0, true
	case sim.CreatureAchero: // as the raptor
		switch {
		case feeding(c):
			return a.achero[side][raptorEatA+chew(c, t)], 0, true
		case c.State == sim.CreatureStrike:
			return a.achero[side][raptorPounce], 2, true
		case c.State == sim.CreatureCower:
			return a.achero[side][raptorCower], 0, true
		case moving && (c.State == sim.CreatureChase || c.State == sim.CreatureFlee):
			return a.achero[side][raptorRunA+(t/4)%2], 0, true
		case moving:
			return a.achero[side][raptorRunA+(t/8)%2], 0, true
		}
		return a.achero[side][raptorStand], 0, true
	case sim.CreatureBrachy, sim.CreatureChampso: // as the crocodile
		arr := &a.brachy
		if c.Kind == sim.CreatureChampso {
			arr = &a.champso
		}
		switch {
		case c.State == sim.CreatureLurk:
			return arr[side][crocLurk], 0, true
		case c.State == sim.CreatureStrike:
			return arr[side][crocBite], 0, true
		case moving && c.State == sim.CreatureChase:
			return arr[side][crocWalkA+(t/3)%2], 0, true
		case moving:
			return arr[side][crocWalkA+(t/9)%2], 0, true
		}
		return arr[side][crocWalkA], 0, true
	case sim.CreatureOrnitho:
		switch {
		case feeding(c):
			return a.ornitho[side][ornithoEatA+chew(c, t)], 0, true
		case c.State == sim.CreatureRest:
			return a.ornitho[side][ornithoRest], 0, true
		case moving && c.State == sim.CreatureFlee:
			return a.ornitho[side][ornithoRunA+(t/3)%2], 0, true
		case moving:
			return a.ornitho[side][ornithoRunA+(t/7)%2], 0, true
		}
		return a.ornitho[side][ornithoStand], 0, true
	case sim.CreatureThesc:
		switch {
		case feeding(c):
			return a.thesc[side][thescEatA+chew(c, t)], 0, true
		case moving && c.State == sim.CreatureFlee:
			return a.thesc[side][thescWalkA+(t/6)%2], 0, true
		case moving:
			return a.thesc[side][thescWalkA+(t/11)%2], 0, true
		}
		return a.thesc[side][thescStand], 0, true
	case sim.CreatureAnzu:
		switch {
		case feeding(c):
			return a.anzu[side][anzuEatA+chew(c, t)], 0, true
		case c.State == sim.CreatureRoar, calling(c): // crest up to warn, or to boom
			return a.anzu[side][anzuDisplay], 0, true
		case c.State == sim.CreatureStrike:
			return a.anzu[side][anzuKick], 0, true
		case moving:
			return a.anzu[side][anzuWalkA+(t/8)%2], 0, true
		}
		return a.anzu[side][anzuStand], 0, true
	case sim.CreatureQuetzal:
		switch {
		case feeding(c):
			return a.quetzal[side][quetzalEatA+chew(c, t)], 0, true
		case c.State == sim.CreatureSoar:
			flap := [4]int{pteroUp, pteroGlide, pteroDown, pteroGlide}
			return a.quetzalFly[side][flap[(t/8)%4]], pteroSoarAlt * 1.4, true
		case c.State == sim.CreatureStrike:
			return a.quetzal[side][quetzalStab], 0, true
		case moving:
			return a.quetzal[side][quetzalWalkA+(t/12)%2], 0, true
		}
		return a.quetzal[side][quetzalWalkA], 0, true
	}
	return sprite{}, 0, false
}

// hellLabel is creatureLabel for the tyrant's neighbours (wild ones; tamed
// ones are labelled as pets before this).
func hellLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	name := c.Kind.Name()
	angry := c.State == sim.CreatureChase || c.State == sim.CreatureStrike
	switch c.Kind {
	case sim.CreatureOrnitho:
		if c.State == sim.CreatureRest {
			return name + " (đang thở dốc) — bắt, hoặc cho quả mọng để thuần", colGood, true
		}
		return name + " — chạy rất nhanh: chờ nó dừng thở", colWhite, true
	case sim.CreatureThesc:
		return name + " — săn (hiền, chỉ bỏ chạy)", colWhite, true
	case sim.CreatureAnzu:
		if angry || c.State == sim.CreatureRoar {
			return name + " — đang xù mào! lùi ra kẻo bị đá", colBad, true
		}
		return name + " — săn (lại gần là nó đá)", colWhite, true
	case sim.CreatureToro:
		if angry {
			return name + " — đang húc! chạy hoặc đánh", colBad, true
		}
		return name + " — săn (cả đàn sẽ húc lại)", colWhite, true
	case sim.CreatureAchero:
		return name + " — đánh (săn theo bầy đông!)", colBad, true
	case sim.CreatureQuetzal:
		return name + " — đánh (mổ rất đau, bị thương thì bay mất)", colBad, true
	case sim.CreatureBrachy, sim.CreatureChampso:
		return name + " — đánh (rình dưới ao!)", colBad, true
	}
	return "", color.RGBA{}, false
}

// hellSound plays what one of the tyrant's neighbours did when its state
// changed from o to c, and reports whether c is one of them.
func hellSound(o, c protocol.CreatureView) bool {
	x, y := float64(c.X), float64(c.Y)
	switch c.Kind {
	case sim.CreatureBrachy, sim.CreatureChampso:
		switch {
		case o.State == sim.CreatureLurk && c.State == sim.CreatureChase:
			snd.playAt("splash", x, y, 0.6, 0.08)
			rate := 1.15
			if c.Kind == sim.CreatureChampso {
				rate = 1.35 // the slender snout, a thinner hiss
			}
			callAt("croc_hiss", x, y, 0.6, rate, 16)
		case c.State == sim.CreatureStrike:
			snd.playAt("chomp", x, y, 0.65, 0.1)
		case c.State == sim.CreatureLurk:
			snd.playAt("splash", x, y, 0.2, 0.15)
		}
	case sim.CreatureToro:
		switch {
		case c.State == sim.CreatureChase && o.State != sim.CreatureStrike:
			if snd.ready("bellow", 700*time.Millisecond) {
				snd.playAt("bellow", x, y, 0.9, 0.08)
			}
		case c.State == sim.CreatureStrike:
			snd.playAt("whack", x, y, 0.95, 0.08)
		}
	case sim.CreatureThesc:
		if c.State == sim.CreatureFlee && o.State != sim.CreatureHurt && snd.ready("bellow", 700*time.Millisecond) {
			snd.playAt("bellow", x, y, 0.5, 0.25)
		}
	case sim.CreatureOrnitho:
		if c.State == sim.CreatureFlee && snd.ready("squeak", 500*time.Millisecond) {
			snd.playAt("squeak", x, y, 0.4, 0.3)
		}
	case sim.CreatureAnzu:
		switch c.State {
		case sim.CreatureRoar:
			snd.playAt("screech", x, y, 0.55, 0.3)
		case sim.CreatureStrike:
			snd.playAt("whack", x, y, 0.6, 0.12)
		}
	case sim.CreatureAchero:
		switch {
		case c.State == sim.CreatureChase && (o.State == sim.CreatureIdle || o.State == sim.CreatureWander):
			if snd.ready("screech", 300*time.Millisecond) {
				callAt("raptor_screech", x, y, 0.6, 1.2, 18) // a smaller throat, higher
			}
		case c.State == sim.CreatureStrike:
			snd.playAt("hiss", x, y, 0.4, 0.2)
		case c.State == sim.CreatureEat && o.State == sim.CreatureChase:
			callAt("chirp", x, y, 0.45, 1.22, 16) // the pack calls over its kill
		}
	case sim.CreatureQuetzal:
		switch c.State {
		case sim.CreatureChase:
			if o.State == sim.CreatureIdle || o.State == sim.CreatureWander {
				snd.playAt("shriek", x, y, 0.7, 0.02)
			}
		case sim.CreatureStrike:
			snd.playAt("whack", x, y, 0.8, 0.08)
		case sim.CreatureSoar:
			snd.playAt("shriek", x, y, 0.9, 0.05)
		}
	default:
		return false
	}
	return true
}

// feeding reports whether c is eating or drinking.
func feeding(c protocol.CreatureView) bool {
	return c.State == sim.CreatureEat || c.State == sim.CreatureDrink
}

// chew is which of a beast's two eating poses to show at frame t: its jaws
// work quickly over a meal, slowly lapping at water.
func chew(c protocol.CreatureView, t int) int {
	if c.State == sim.CreatureDrink {
		return (t / 18) % 2
	}
	return (t / 9) % 2
}

// feedSound plays a beast starting to eat, drink or chew, and reports
// whether it played anything.
func (g *Game) feedSound(o, c protocol.CreatureView) bool {
	x, y := float64(c.X), float64(c.Y)
	switch {
	case c.State == sim.CreatureEat && o.State != sim.CreatureEat:
		if c.Kind.Hunts() {
			snd.playAt("chomp", x, y, 0.6, 0.1) // into the kill
		} else {
			snd.playAt("rustle", x, y, 0.35, 0.15) // a mouthful of leaves
		}
		return true
	case c.State == sim.CreatureDrink && o.State != sim.CreatureDrink:
		snd.playAt("plop", x, y, 0.3, 0.2)
		return true
	}
	return false
}

// chewSounds plays the jaws of beasts eating near the player now and then.
func (g *Game) chewSounds(cur protocol.Snapshot) {
	for _, c := range cur.Creatures {
		if c.State != sim.CreatureEat {
			continue
		}
		if snd.ready("chew", 700*time.Millisecond) {
			snd.playAt("eat", float64(c.X), float64(c.Y), 0.22, 0.25)
		}
		return
	}
}
