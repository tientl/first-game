package client

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Animals: their sprites (drawn side-on facing right, mirrored for left),
// the icons of what they give, their sounds, and how they are drawn. Fish
// are drawn in their own pass under everything else, since they are under
// water.

var (
	wormRamp  = mkRamp(0x7d3343, 0xb4566a, 0xd98393, 0xf1b3bc)
	fishRamp  = mkRamp(0x24485e, 0x356a82, 0x4f8fa6, 0x86bfcf, 0xd2eef2)
	finRamp   = mkRamp(0x2a5468, 0x3f7890, 0x5a96ac)
	meatRamp  = mkRamp(0x6b1a22, 0x9e2c35, 0xc94a4f, 0xe57b77, 0xf6b1a6)
	colPink   = hex(0xe8a0a0)
	colBone   = hex(0xece4d2)
	colBoneSh = hex(0xc9bea6)
)

const (
	rabbitSit = iota
	rabbitHop
	rabbitSleep
	rabbitPoses

	wormFrames = 2
	fishFrames = 2
)

// creatureArt holds every animal sprite; the first index is 0 facing right,
// 1 facing left.
type creatureArt struct {
	rabbit [2][rabbitPoses]sprite
	snake  [2][snakeSprites]sprite // see snake.go
	crab   [crabPoses]sprite       // walks sideways, so no left and right; see crab.go
	croc   [2][crocPoses]sprite    // see croc.go
	burrow sprite                  // a rabbit burrow; see burrows.go
	den    sprite                  // a snake den
	worm   [2][wormFrames]sprite
	fish   [2][fishFrames]sprite

	// The great beasts: see dinoart.go, hunterart.go and skyseaart.go.
	trike       [2][trikePoses]sprite
	stego       [2][stegoPoses]sprite
	brachio     [2][brachPoses]sprite
	raptor      [2][raptorPoses]sprite
	rex         [2][rexPoses]sprite
	ptero       [2][pteroPoses]sprite
	plesio      [2][plesioPoses]sprite
	mammoth     [2][mammothPoses]sprite // see iceart.go
	sabre       [2][sabrePoses]sprite
	hell        hellArt // the tyrant's neighbours: see hellart.go and hellwire.go
	pteroShadow sprite
}

func genCreatureArt() creatureArt {
	var a creatureArt
	for p := range rabbitPoses {
		a.rabbit[0][p] = genRabbit(p)
		a.rabbit[1][p] = mirrorSprite(a.rabbit[0][p])
	}
	for f := range snakeSprites {
		a.snake[0][f] = genSnake(f)
		a.snake[1][f] = mirrorSprite(a.snake[0][f])
	}
	for f := range wormFrames {
		a.worm[0][f] = genWorm(f)
		a.worm[1][f] = mirrorSprite(a.worm[0][f])
	}
	for f := range fishFrames {
		a.fish[0][f] = genFish(f)
		a.fish[1][f] = mirrorSprite(a.fish[0][f])
	}
	for p := range crabPoses {
		a.crab[p] = genCrab(p)
	}
	for p := range crocPoses {
		a.croc[0][p] = genCroc(p)
		a.croc[1][p] = mirrorSprite(a.croc[0][p])
	}
	a.burrow, a.den = genBurrow(), genDen()
	for p := range trikePoses {
		a.trike[0][p], a.trike[1][p] = sidedPair(genTrike(p))
	}
	for p := range stegoPoses {
		a.stego[0][p], a.stego[1][p] = sidedPair(genStego(p))
	}
	for p := range brachPoses {
		a.brachio[0][p], a.brachio[1][p] = sidedPair(genBrachio(p))
	}
	for p := range raptorPoses {
		a.raptor[0][p], a.raptor[1][p] = sidedPair(genRaptor(p))
	}
	for p := range rexPoses {
		a.rex[0][p], a.rex[1][p] = sidedPair(genRex(p))
	}
	for p := range pteroPoses {
		a.ptero[0][p], a.ptero[1][p] = sidedPair(genPtero(p))
	}
	for p := range plesioPoses {
		a.plesio[0][p], a.plesio[1][p] = sidedPair(genPlesio(p))
	}
	for p := range mammothPoses {
		a.mammoth[0][p], a.mammoth[1][p] = sidedPair(genMammoth(p))
	}
	for p := range sabrePoses {
		a.sabre[0][p], a.sabre[1][p] = sidedPair(genSabre(p))
	}
	a.hell = genHellArt()
	a.pteroShadow = mk(pteroShadow(), 10, 3)
	return a
}

func sidedPair(s sprite) (sprite, sprite) { return s, mirrorSprite(s) }

// --- sprites ----------------------------------------------------------------

type ell struct{ x, y, rx, ry float64 }

// roundShape fills e with a ramp lit like a sphere; bias brightens or darkens.
func roundShape(c *canvas, e ell, rp ramp, bias float64) {
	c.ellipse(e.x, e.y, e.rx, e.ry, func(x, y int, nx, ny float64) color.RGBA {
		return rp.pick(lum(nx, ny)*0.95+bias, x, y)
	})
}

// wormStamps draws a worm as discs along a wave from x0 to x1.
func wormStamps(c *canvas, x0, x1, cy, amp, freq, ph, r float64) {
	for s := 0; ; s++ {
		x := x0 + float64(s)*0.25
		if x > x1 {
			break
		}
		y := cy + amp*math.Sin((x-x0)*freq-ph)
		saddle := (x-x0)/(x1-x0) > 0.55 && (x-x0)/(x1-x0) < 0.7
		ring := int(math.Floor(x*1.6))%3 == 0
		c.ellipse(x, y, r, r, func(px, py int, nx, ny float64) color.RGBA {
			l := lum(nx, ny)
			if ring {
				l -= 0.25
			}
			col := wormRamp.pick(l, px, py)
			if saddle {
				col = mix(col, hex(0xf0a070), 0.4)
			}
			return col
		})
	}
}

func genWorm(frame int) sprite {
	c := newCanvas(13, 7)
	wormStamps(c, 2, 10, 4, 0.8, 0.95, float64(frame)*math.Pi, 0.85)
	c.outline(0.6)
	return mk(c, 6, 5)
}

func genFish(frame int) sprite {
	c := newCanvas(17, 9)
	tail := []float64{4.5, 4.5, 0.5, 1.6, 0.5, 7.4}
	if frame == 1 {
		tail = []float64{4.5, 4.5, 1.2, 2.4, 0.2, 6.8}
	}
	c.polygon(tail, func(x, y int) color.RGBA { return finRamp.pick(0.7-float64(y)*0.08, x, y) })
	c.polygon([]float64{7, 2.8, 10.5, 1, 11, 3}, solid(finRamp[1])) // dorsal fin
	c.ellipse(9, 4.5, 5.2, 2.5, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if ny > 0.35 {
			l += 0.3 // pale belly
		}
		return fishRamp.pick(l, x, y)
	})
	c.set(12, 4, colInk)
	c.outline(0.6)
	return mk(c, 8, 6)
}

// genCreatureIcons draws the bag icons for what animals give.
func genCreatureIcons(icons *[sim.NumItems]*ebiten.Image) {
	meat := newCanvas(16, 16)
	meat.line(8, 8, 13, 13, func(int, int, float64) color.RGBA { return colBone })
	meat.line(8, 9, 13, 14, func(int, int, float64) color.RGBA { return colBoneSh })
	roundShape(meat, ell{13.6, 14.3, 1.3, 1.3}, ramp{colBoneSh, colBone}, 0)
	roundShape(meat, ell{14.4, 13.2, 1.3, 1.3}, ramp{colBoneSh, colBone}, 0)
	meat.ellipse(6.5, 6.5, 5.5, 5, func(x, y int, nx, ny float64) color.RGBA {
		if math.Abs(nx*0.8+ny*0.6-0.1) < 0.09 {
			return hex(0xf4d8cc) // a streak of fat
		}
		return meatRamp.pick(lum(nx, ny), x, y)
	})
	meat.outline(0.8)
	icons[sim.ItemMeat] = meat.image()

	worm := newCanvas(16, 16)
	wormStamps(worm, 2, 14, 8, 3.2, 0.52, 0, 1.4)
	worm.outline(0.8)
	icons[sim.ItemWorm] = worm.image()

	fish := newCanvas(16, 16)
	fish.polygon([]float64{5.5, 8, 0.5, 3.6, 0.5, 12.4}, func(x, y int) color.RGBA { return finRamp.pick(0.8-float64(y)*0.05, x, y) })
	fish.polygon([]float64{7, 5.2, 11, 2.6, 11.6, 5.4}, solid(finRamp[1]))
	fish.ellipse(9.6, 8, 5.8, 3.4, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if ny > 0.35 {
			l += 0.3
		}
		return fishRamp.pick(l, x, y)
	})
	fish.set(13, 7, colInk)
	fish.outline(0.8)
	icons[sim.ItemFish] = fish.image()

	// And what they become on the fire (drawn in craftart.go).
	icons[sim.ItemCookedMeat] = iconCookedMeat()
	icons[sim.ItemCookedFish] = iconCookedFish()
	icons[sim.ItemCrabMeat], icons[sim.ItemCookedCrab] = iconCrabMeat(), iconCookedCrab() // crab.go
	icons[sim.ItemHide] = iconHide().image()                                              // dinoart.go
	icons[sim.ItemBone] = iconBone().image()
	icons[sim.ItemRexTooth] = iconRexTooth().image()
	icons[sim.ItemSaddle] = iconSaddle().image() // ride.go
	icons[sim.ItemFur] = iconFur().image()       // iceart.go
	icons[sim.ItemIvory] = iconIvory().image()
	icons[sim.ItemFurCoat] = iconFurCoat().image()
	icons[sim.ItemIvorySpear] = iconIvorySpear().image()
}

// --- sounds -----------------------------------------------------------------

// genCreatureSFX adds the animal sounds to the effect bank.
func genCreatureSFX(r *rand.Rand, add func(name string, peak float64, x []float64)) {
	{
		flt := newFilter(highpass, 3200, 0.7)
		x := render(0.7, func(t float64, _ int) float64 {
			return flt.do(r.Float64()*2-1) * env(t, 0.05, 0.2)
		})
		add("hiss", 1, x)
	}
	{
		flt := newFilter(lowpass, 3000, 0.7)
		x := render(0.6, func(t float64, _ int) float64 {
			flt.set(lowpass, max(400, 3000-t*4000), 0.7)
			return flt.do(r.Float64()*2-1) * env(t, 0.005, 0.12)
		})
		for range 6 {
			f := 900 + r.Float64()*1500
			addAt(x, tone(0.08, f, f*1.6, 0.05, 0.002, 0.02), 0.05+r.Float64()*0.35, 0.25)
		}
		add("splash", 1, x)
	}
	{
		x := make([]float64, samples(0.3))
		addAt(x, tone(0.12, 1900, 2700, 0.06, 0.004, 0.035), 0, 1)
		addAt(x, tone(0.1, 2200, 3000, 0.05, 0.004, 0.03), 0.11, 0.7)
		add("squeak", 1, x)
	}
	{ // a wet squelch
		x := make([]float64, samples(0.3))
		addAt(x, grains(r, 0.18, 400, 0.008, bandpass, 650, 1.4, func(t float64) float64 { return env(t, 0.01, 0.05) }), 0, 1)
		addAt(x, tone(0.15, 240, 150, 0.08, 0.004, 0.04), 0, 0.6)
		add("pick_worm", 1, x)
	}
	{ // a soft thump into the bag
		x := make([]float64, samples(0.3))
		addAt(x, tone(0.2, 150, 90, 0.06, 0.002, 0.05), 0, 1)
		addAt(x, noiseBurst(r, 0.08, lowpass, 900, 0.7, 0.002, 0.02), 0, 0.5)
		add("pick_meat", 1, x)
	}
	{ // a fish tugging the float under: a hollow plop
		x := make([]float64, samples(0.35))
		addAt(x, tone(0.22, 520, 180, 0.07, 0.002, 0.05), 0, 1)
		addAt(x, noiseBurst(r, 0.12, bandpass, 1400, 1.2, 0.002, 0.03), 0.01, 0.5)
		add("plop", 1, x)
	}
	{ // a crab's claw snapping shut: two hard clicks
		x := make([]float64, samples(0.25))
		addAt(x, noiseBurst(r, 0.03, bandpass, 3200, 3, 0.0002, 0.004), 0, 1)
		addAt(x, noiseBurst(r, 0.03, bandpass, 2400, 3, 0.0002, 0.005), 0.05, 0.9)
		addAt(x, tone(0.06, 900, 600, 0.03, 0.001, 0.015), 0.05, 0.4)
		add("pinch", 1, x)
	}
	{ // a crocodile's jaws slamming shut: a heavy, wet thud
		x := make([]float64, samples(0.45))
		addAt(x, tone(0.3, 120, 55, 0.08, 0.002, 0.08), 0, 1)
		addAt(x, noiseBurst(r, 0.12, lowpass, 1200, 0.8, 0.001, 0.03), 0, 0.8)
		addAt(x, noiseBurst(r, 0.04, bandpass, 2200, 2, 0.0003, 0.006), 0.01, 0.6)
		add("chomp", 1, x)
	}
	genDinoSFX(r, add) // last, so the rng gives the sounds above as before
	genIceSFX(r, add)
}

// creatureSounds plays what animals did between two snapshots.
func (g *Game) creatureSounds(old, cur protocol.Snapshot) {
	was := make(map[sim.CreatureID]protocol.CreatureView, len(old.Creatures))
	for _, c := range old.Creatures {
		was[c.ID] = c
	}
	for _, c := range cur.Creatures {
		o, seen := was[c.ID]
		delete(was, c.ID)
		if !seen || o.State == c.State {
			continue
		}
		if g.feedSound(o, c) || dinoSound(o, c) || iceSound(o, c) {
			continue // see dinosound.go and iceart.go
		}
		if c.State == sim.CreatureHidden || o.State == sim.CreatureHidden {
			if c.Kind == sim.CreatureCrab {
				snd.playAt("splash", float64(c.X), float64(c.Y), 0.25, 0.2) // under the water, or up again
			} else {
				snd.playAt("rustle", float64(c.X), float64(c.Y), 0.3, 0.15) // a scramble at the mouth of a hole
			}
			continue
		}
		if hellSound(o, c) {
			continue // see hellwire.go
		}
		if c.Kind == sim.CreatureCroc {
			switch {
			case o.State == sim.CreatureLurk && c.State == sim.CreatureChase:
				snd.playAt("splash", float64(c.X), float64(c.Y), 0.7, 0.05) // bursting out of the water
				callAt("croc_hiss", float64(c.X), float64(c.Y), 0.75, 1, 16)
			case c.State == sim.CreatureStrike:
				snd.playAt("chomp", float64(c.X), float64(c.Y), 0.8, 0.06)
			case c.State == sim.CreatureLurk:
				snd.playAt("splash", float64(c.X), float64(c.Y), 0.25, 0.15) // sliding back under
			}
			continue
		}
		if c.Kind == sim.CreatureCrab && c.State == sim.CreatureStrike {
			snd.playAt("pinch", float64(c.X), float64(c.Y), 0.5, 0.1)
			continue
		}
		if c.Kind != sim.CreatureSnake {
			continue
		}
		switch c.State {
		case sim.CreatureChase, sim.CreatureStrike, sim.CreatureHurt:
			if o.State != sim.CreatureStrike && snd.ready("hiss", 400*time.Millisecond) {
				snd.playAt("hiss", float64(c.X), float64(c.Y), 0.35, 0.1)
			}
		}
	}
	// What vanished right next to a player was caught; anything else just
	// went out of view.
	for _, c := range was {
		if !nearAPlayer(cur.Players, float64(c.X), float64(c.Y), c.Kind.Reach()+0.6) {
			continue
		}
		switch c.Kind {
		case sim.CreatureRabbit:
			snd.playAt("squeak", float64(c.X), float64(c.Y), 0.45, 0.1)
		case sim.CreatureSnake:
			snd.playAt("hiss", float64(c.X), float64(c.Y), 0.4, 0.15)
		case sim.CreatureTrike, sim.CreatureStego, sim.CreatureBrachio, sim.CreatureRex, sim.CreatureMammoth, sim.CreatureToro, sim.CreatureQuetzal:
			snd.playAt("treefall", float64(c.X), float64(c.Y), 0.5, 0.1) // the great body comes down
		}
	}
	g.beastSteps(cur)
	g.tameEffects(old, cur) // see ride.go
	g.chewSounds(cur)       // see hellwire.go
	g.beastCalls(cur)       // see roar.go
	g.predatorGrowls(cur)   // see predsound.go
}

// swingAtCreatureSound is the sound of an Action landing on c.
func swingAtCreatureSound(c protocol.CreatureView) {
	x, y := float64(c.X), float64(c.Y)
	switch c.Kind {
	case sim.CreatureFish:
		snd.playAt("splash", x, y, 0.55, 0.08)
	case sim.CreatureWorm:
		snd.playAt("rustle", x, y, 0.25, 0.15)
	case sim.CreatureTrike, sim.CreatureStego, sim.CreatureBrachio, sim.CreatureRex,
		sim.CreatureRaptor, sim.CreaturePtero, sim.CreaturePlesio, sim.CreatureMammoth, sim.CreatureSabre,
		sim.CreatureToro, sim.CreatureAchero, sim.CreatureQuetzal, sim.CreatureBrachy, sim.CreatureChampso, sim.CreatureAnzu, sim.CreatureThesc:
		snd.playAt("whack", x, y, 0.6, 0.1)
	default:
		snd.playAt("rustle", x, y, 0.45, 0.1)
	}
}

func nearAPlayer(ps []protocol.PlayerView, x, y, r float64) bool {
	for _, p := range ps {
		if math.Hypot(float64(p.X)-x, float64(p.Y)-y) <= r {
			return true
		}
	}
	return false
}

// --- targeting --------------------------------------------------------------

// targetView is what the Action key would work on right now.
type targetView struct {
	ok       bool
	creature bool // c is the target, else e
	drop     bool // a pile on the ground, d
	e        protocol.EntityView
	c        protocol.CreatureView
	d        protocol.DropView
}

// creatureCandidates lists the creatures in cs near (x, y), for sim.PickTarget.
func creatureCandidates(cs []protocol.CreatureView, x, y float64) []sim.CreatureCandidate {
	const far = sim.CandidateRange // more than any creature's reach
	var out []sim.CreatureCandidate
	for _, c := range cs {
		cx, cy := float64(c.X), float64(c.Y)
		if c.State.OutOfReach() || c.Ridden {
			continue // under someone's float, down a hole, under water, up in the air or ridden: as on the server
		}
		if math.Abs(cx-x) <= far && math.Abs(cy-y) <= far {
			out = append(out, sim.CreatureCandidate{ID: c.ID, Kind: c.Kind, X: cx, Y: cy})
		}
	}
	return out
}

// creatureLabel is what the target marker says over c.
// creatureLabel is creatureLabelBase with what a beast is busy at, if
// anything: eating, drinking or hunting.
func (g *Game) creatureLabel(c protocol.CreatureView) (string, color.RGBA) {
	label, col := g.creatureLabelBase(c)
	busy := ""
	switch {
	case c.State == sim.CreatureEat:
		busy = " (đang ăn)"
	case c.State == sim.CreatureDrink:
		busy = " (đang uống)"
	case c.State == sim.CreatureChase && c.Kind.Hunts():
		busy = " (đang săn)"
	}
	if name := c.Kind.Name(); busy != "" && strings.HasPrefix(label, name) {
		label = name + busy + label[len(name):]
	}
	return label + healthText(c, g.myHeld()), col
}

func (g *Game) creatureLabelBase(c protocol.CreatureView) (string, color.RGBA) {
	name := c.Kind.Name()
	if c.Tamed { // see ride.go
		if c.Owner != g.myID {
			return name + " — thú của " + g.playerName(c.Owner), colWhite
		}
		if food := c.Kind.TameFood(); g.myHeld() == food {
			return name + " của bạn — cho ăn để hồi máu", colGood
		}
		verb := "bảo đứng chờ"
		if c.Stay {
			verb = "gọi đi theo"
		}
		if c.Kind.Rideable() {
			return name + " của bạn — Space: " + verb + ", R: cưỡi", colGood
		}
		return name + " của bạn — Space: " + verb, colGood
	}
	if food := c.Kind.TameFood(); food.Valid() && g.myHeld() == food {
		progress := fmt.Sprintf(" (%d/%d)", c.Tame, c.Kind.TameMeals())
		switch {
		case c.Kind.MustCower() && c.State != sim.CreatureCower:
			return name + " — đánh cho gục rồi mới cho ăn được", colBad
		case c.State == sim.CreatureChase || c.State == sim.CreatureStrike:
			return name + " — đang nổi giận, không chịu ăn", colBad
		}
		return name + " — cho " + strings.ToLower(food.Def().Name) + " để thuần" + progress, colGood
	}
	if c.State == sim.CreatureCower {
		return name + " (đã gục) — cầm " + strings.ToLower(c.Kind.TameFood().Def().Name) + " để thuần", colGood
	}
	if l, col, ok := seaLabel(c); ok {
		return l, col // see searender.go
	}
	if l, col, ok := tribalLabel(c); ok {
		return l, col // see tribalrender.go
	}
	if l, col, ok := hellLabel(c); ok {
		return l, col // see hellwire.go
	}
	switch c.Kind {
	case sim.CreatureRabbit:
		if c.State == sim.CreatureSleep {
			return name + " (đang ngủ) — bắt", colWhite
		}
		return name + " — bắt", colWhite
	case sim.CreatureSnake:
		return name + " — đánh", colWhite
	case sim.CreatureWorm:
		return name + " — nhặt làm mồi", colWhite
	case sim.CreatureCrab:
		if c.State == sim.CreatureChase || c.State == sim.CreatureStrike {
			return name + " — bắt (coi chừng bị kẹp)", colWhite
		}
		return name + " — bắt", colWhite
	case sim.CreatureCroc:
		return name + " — đánh (cắn rất đau!)", colBad
	case sim.CreatureTrike:
		if c.State == sim.CreatureChase || c.State == sim.CreatureStrike {
			return name + " — đang húc! chạy hoặc đánh", colBad
		}
		return name + " — săn (cả đàn sẽ húc lại)", colWhite
	case sim.CreatureStego:
		if c.State == sim.CreatureChase || c.State == sim.CreatureStrike {
			return name + " — coi chừng đuôi gai!", colBad
		}
		return name + " — săn (đuôi gai quất người đứng gần)", colWhite
	case sim.CreatureBrachio:
		return name + " — săn (hiền, chỉ bỏ đi)", colWhite
	case sim.CreatureRaptor:
		return name + " — đánh (săn theo bầy!)", colBad
	case sim.CreatureRex:
		return name + " — đánh (cực kỳ nguy hiểm!)", colBad
	case sim.CreaturePtero:
		if c.State == sim.CreatureIdle {
			return name + " (đang đậu) — đánh", colWhite
		}
		return name + " — đánh khi nó bổ nhào", colBad
	case sim.CreaturePlesio:
		return name + " — đánh vào cổ (cắn người trên bờ)", colBad
	case sim.CreatureMammoth:
		if c.State == sim.CreatureChase || c.State == sim.CreatureStrike {
			return name + " — đang húc ngà! chạy hoặc đánh", colBad
		}
		return name + " — săn (cả đàn sẽ húc lại)", colWhite
	case sim.CreatureSabre:
		return name + " — đánh (rình rồi vồ!)", colBad
	case sim.CreatureFish:
		if tool := c.Kind.Tool(); tool.Valid() && g.myHeld() != tool {
			return name + " — cầm " + tool.Def().Name + " trên tay để câu", colBad
		}
		bait := c.Kind.Bait()
		if g.curr.Self.Inv.Count(bait) == 0 {
			return name + " — cần " + bait.Def().Name + " làm mồi", colBad
		}
		return name + " — quăng câu (cá cắn thì nhấn Space)", colWhite
	}
	return name, colWhite
}

// --- drawing ----------------------------------------------------------------

// lerpCreature returns where to draw c and whether it is moving.
func (g *Game) lerpCreature(c protocol.CreatureView, alpha float64) (x, y float64, moving bool) {
	x, y = float64(c.X), float64(c.Y)
	q, ok := g.prevCreatures[c.ID]
	if !ok || (q.X == c.X && q.Y == c.Y) {
		return x, y, false
	}
	px, py := float64(q.X), float64(q.Y)
	if math.Abs(px-x) > teleportTiles || math.Abs(py-y) > teleportTiles {
		return x, y, false
	}
	return px + (x-px)*alpha, py + (y-py)*alpha, true
}

// creatureSprite returns the sprite for c this frame and how far above the
// ground to draw it.
func (g *Game) creatureSprite(c protocol.CreatureView, moving bool) (sprite, float64) {
	a := &g.art.beasts
	side := 0
	if c.FaceLeft {
		side = 1
	}
	t := g.frame + int(c.ID)*7 // so a group does not move in lockstep
	if s, bob, ok := g.seaSprite(c); ok {
		return s, bob // see searender.go
	}
	if s, bob, ok := g.tribalSprite(c, moving); ok {
		return s, bob // see tribalrender.go
	}
	if s, bob, ok := g.woodsSprite(c, moving); ok {
		return s, bob // see jungleart.go
	}
	if s, bob, ok := g.hellSprite(c, moving); ok {
		return s, bob // see hellwire.go
	}
	switch c.Kind {
	case sim.CreatureRabbit:
		switch {
		case c.State == sim.CreatureSleep:
			return a.rabbit[side][rabbitSleep], 0
		case moving:
			period := 16
			if c.State == sim.CreatureFlee {
				period = 9
			}
			ph := float64(t%period) / float64(period)
			if bob := math.Round(math.Sin(ph*math.Pi) * 3); bob >= 1 {
				return a.rabbit[side][rabbitHop], bob
			}
		}
		return a.rabbit[side][rabbitSit], 0
	case sim.CreatureSnake:
		switch {
		case c.State == sim.CreatureStrike:
			return a.snake[side][snakeStrike], 0
		case c.State == sim.CreatureIdle && !moving: // resting in a coil, tasting the air now and then
			if (t/9)%9 == 0 {
				return a.snake[side][snakeCoilLik], 0
			}
			return a.snake[side][snakeCoil], 0
		case moving:
			return a.snake[side][(t/6)%snakeSlither], 0
		}
		return a.snake[side][0], 0
	case sim.CreatureWorm:
		return a.worm[side][(t/20)%wormFrames], 0
	case sim.CreatureCrab:
		switch {
		case c.State == sim.CreatureChase || c.State == sim.CreatureStrike:
			return a.crab[crabPinch], 0 // claws up
		case moving:
			return a.crab[crabWalkA+(t/4)%2], 0
		}
		return a.crab[crabWalkA], 0
	case sim.CreatureCroc:
		switch {
		case c.State == sim.CreatureLurk:
			return a.croc[side][crocLurk], 0
		case c.State == sim.CreatureStrike:
			return a.croc[side][crocBite], 0
		case moving && c.State == sim.CreatureChase:
			return a.croc[side][crocWalkA+(t/3)%2], 0 // a scrambling rush
		case moving:
			return a.croc[side][crocWalkA+(t/9)%2], 0
		}
		return a.croc[side][crocWalkA], 0
	case sim.CreatureTrike:
		switch {
		case feeding(c):
			return a.trike[side][trikeEatA+chew(c, t)], 0
		case c.State == sim.CreatureStrike, calling(c): // a gore, or a bellow with the head thrown up
			return a.trike[side][trikeGore], 0
		case c.State == sim.CreatureChase && moving: // a pounding charge
			return a.trike[side][trikeCharge], float64((t / 4) % 2)
		case c.State == sim.CreatureChase:
			return a.trike[side][trikeCharge], 0
		case moving:
			return a.trike[side][trikeWalkA+(t/12)%2], 0
		}
		return a.trike[side][trikeWalkA], 0
	case sim.CreatureStego:
		switch {
		case feeding(c):
			return a.stego[side][stegoEatA+chew(c, t)], 0
		case c.State == sim.CreatureStrike:
			return a.stego[side][stegoSwipe], 0
		case moving:
			return a.stego[side][stegoWalkA+(t/14)%2], 0
		}
		return a.stego[side][stegoWalkA], 0
	case sim.CreatureBrachio:
		if feeding(c) {
			return a.brachio[side][brachEatA+chew(c, t)], 0
		}
		if moving {
			return a.brachio[side][brachWalkA+(t/16)%2], 0
		}
		return a.brachio[side][brachWalkA], 0
	case sim.CreatureRaptor:
		switch {
		case feeding(c):
			return a.raptor[side][raptorEatA+chew(c, t)], 0
		case c.State == sim.CreatureStrike:
			return a.raptor[side][raptorPounce], 3
		case c.State == sim.CreatureCower:
			return a.raptor[side][raptorCower], 0
		case moving && (c.State == sim.CreatureChase || c.State == sim.CreatureFlee):
			return a.raptor[side][raptorRunA+(t/4)%2], 0
		case moving:
			return a.raptor[side][raptorRunA+(t/8)%2], 0
		}
		return a.raptor[side][raptorStand], 0
	case sim.CreatureRex:
		switch {
		case feeding(c):
			return a.rex[side][rexEatA+chew(c, t)], 0
		case c.State == sim.CreatureRoar, calling(c):
			return a.rex[side][rexRoar], 0
		case c.State == sim.CreatureStrike:
			return a.rex[side][rexBite], 0
		case moving && c.State == sim.CreatureChase:
			return a.rex[side][rexWalkA+(t/7)%2], 0
		case moving:
			return a.rex[side][rexWalkA+(t/11)%2], 0
		}
		return a.rex[side][rexWalkA], 0
	case sim.CreaturePtero:
		flap := [4]int{pteroUp, pteroGlide, pteroDown, pteroGlide}
		switch c.State {
		case sim.CreatureIdle:
			return a.ptero[side][pteroPerch], 0
		case sim.CreatureSoar:
			if (t/90)%3 == 0 { // now and then it flaps; mostly it glides
				return a.ptero[side][flap[(t/6)%4]], pteroSoarAlt
			}
			return a.ptero[side][pteroGlide], pteroSoarAlt
		case sim.CreatureFlee:
			return a.ptero[side][flap[(t/4)%4]], pteroClimbAlt
		case sim.CreatureStrike:
			return a.ptero[side][pteroGlide], pteroDiveAlt - 3
		}
		return a.ptero[side][pteroDive], pteroDiveAlt
	case sim.CreaturePlesio:
		switch c.State {
		case sim.CreatureLurk:
			return a.plesio[side][plesioUnder], 0
		case sim.CreatureStrike:
			return a.plesio[side][plesioStrike], 0
		}
		return a.plesio[side][plesioUp], 0
	case sim.CreatureMammoth:
		switch {
		case feeding(c):
			return a.mammoth[side][mammothEatA+chew(c, t)], 0
		case c.State == sim.CreatureStrike, calling(c): // a gore, or trumpeting with the trunk raised
			return a.mammoth[side][mammothGore], 0
		case c.State == sim.CreatureChase:
			bob := 0.0
			if moving {
				bob = float64((t / 5) % 2)
			}
			return a.mammoth[side][mammothCharge], bob
		case moving:
			return a.mammoth[side][mammothWalkA+(t/14)%2], 0
		}
		return a.mammoth[side][mammothWalkA], 0
	case sim.CreatureSabre:
		switch {
		case feeding(c):
			return a.sabre[side][sabreEatA+chew(c, t)], 0
		case c.State == sim.CreatureStrike:
			return a.sabre[side][sabreSpring], 3
		case c.State == sim.CreatureCower:
			return a.sabre[side][sabreCower], 0
		case c.State == sim.CreatureChase:
			return a.sabre[side][sabreStalk], 0
		case moving && c.State == sim.CreatureFlee:
			return a.sabre[side][sabreWalkA+(t/4)%2], 0
		case moving:
			return a.sabre[side][sabreWalkA+(t/9)%2], 0
		}
		return a.sabre[side][sabreStand], 0
	default:
		return a.fish[side][(t/9)%fishFrames], 0
	}
}

// inTheSky reports whether c is drawn high in the air, above everything on
// the ground, by drawSky.
func inTheSky(c protocol.CreatureView) bool {
	return c.Kind == sim.CreaturePtero && (c.State == sim.CreatureSoar || c.State == sim.CreatureFlee) ||
		c.Kind == sim.CreatureQuetzal && c.State == sim.CreatureSoar
}

// drawSky draws the flying reptiles high overhead, after everything on the
// ground, with their shadows far below. Inside a cave there is no sky.
func (g *Game) drawSky(dst *ebiten.Image, cam camera, alpha float64) {
	if g.caveIn() > 0.5 {
		return
	}
	for _, c := range g.curr.Creatures {
		if !inTheSky(c) {
			continue
		}
		x, y, moving := g.lerpCreature(c, alpha)
		sx, sy := cam.toScreen(x, y)
		s, alt := g.creatureSprite(c, moving)
		sh := &ebiten.DrawImageOptions{}
		sh.GeoM.Translate(sx-g.art.beasts.pteroShadow.ax, sy-g.art.beasts.pteroShadow.ay)
		if c.Kind == sim.CreatureQuetzal { // a far bigger shadow
			sh.GeoM.Translate(-sx, -sy)
			sh.GeoM.Scale(2.4, 2.4)
			sh.GeoM.Translate(sx, sy)
		}
		sh.ColorScale.ScaleAlpha(0.6)
		dst.DrawImage(g.art.beasts.pteroShadow.img, sh)
		alt += math.Round(math.Sin(float64(g.frame)*0.05+float64(c.ID)) * 1.5)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-s.ax, sy-alt-s.ay)
		dst.DrawImage(s.img, op)
	}
}

// creatureShadow is the width of the shadow under a creature of kind k, in
// art pixels; the great beasts' grow with how big they are drawn.
func creatureShadow(k sim.CreatureKind) float64 { return creatureShadowBase(k) * beastScale(k) }

func creatureShadowBase(k sim.CreatureKind) float64 {
	switch k {
	case sim.CreatureToro:
		return 30
	case sim.CreatureOrnitho, sim.CreatureAnzu:
		return 13
	case sim.CreatureThesc:
		return 18
	case sim.CreatureAchero:
		return 12
	case sim.CreatureQuetzal:
		return 20
	case sim.CreatureRabbit, sim.CreatureMonkey:
		return 12
	case sim.CreatureSnake:
		return 20
	case sim.CreatureCrab:
		return 14
	case sim.CreatureTrike, sim.CreatureStego, sim.CreatureRex:
		return 30
	case sim.CreatureMammoth:
		return 38
	case sim.CreatureSabre:
		return 18
	case sim.CreatureBrachio:
		return 36
	case sim.CreatureRaptor:
		return 15
	case sim.CreatureSpider:
		return 18
	case sim.CreatureBear:
		return 30
	case sim.CreatureTribesman:
		return 12
	case sim.CreatureScorpion:
		return 30
	default:
		return 0 // fish and sea reptiles are under water; flying ones cast their own
	}
}

func (g *Game) drawCreature(dst *ebiten.Image, c protocol.CreatureView, d drawable) {
	if c.State == sim.CreatureLurk {
		return // only its eyes show, drawn with the water by drawFish
	}
	s, bob := g.creatureSprite(c, d.moving)
	x := d.sx
	if c.Kind == sim.CreaturePtero && bob > 0 { // swooping low: its shadow right under it
		sh := &ebiten.DrawImageOptions{}
		sh.GeoM.Translate(d.sx-g.art.beasts.pteroShadow.ax, d.sy-g.art.beasts.pteroShadow.ay)
		dst.DrawImage(g.art.beasts.pteroShadow.img, sh)
	}
	op := &ebiten.DrawImageOptions{}
	if c.State == sim.CreatureHurt {
		x += math.Round(math.Sin(float64(g.frame) * 1.3)) // flinch
		op.ColorScale.Scale(1, 0.55, 0.55, 1)
	}
	if c.State == sim.CreatureTrapped {
		x += math.Round(math.Sin(float64(g.frame) * 0.9)) // struggling in the snare
	}
	wades := c.Kind == sim.CreatureCrab || crocLike(c.Kind)
	if wades && g.tileAt(int(math.Floor(float64(c.X))), int(math.Floor(float64(c.Y)))) == sim.TileWater {
		op.ColorScale.Scale(0.72, 0.86, 0.95, 1) // wading: seen through the shallows
	}
	op.GeoM.Translate(x-s.ax, d.sy-bob-s.ay)
	dst.DrawImage(s.img, op)
	if c.State == sim.CreatureSleep {
		g.drawZzz(dst, d.sx+2, d.sy-10, float64(c.ID))
	}
	if c.Kind == sim.CreatureKraken {
		g.drawKrakenArms(dst, c, d)
	}
}

// drawZzz floats two little z's up from a sleeper.
func (g *Game) drawZzz(dst *ebiten.Image, x, y, seed float64) {
	z := [7][2]float32{{0, 0}, {1, 0}, {2, 0}, {1, 1}, {0, 2}, {1, 2}, {2, 2}}
	for i := range 2 {
		ph := math.Mod(float64(g.frame)/110+float64(i)*0.5+seed*0.13, 1)
		a := uint8(220 * (1 - ph))
		zx, zy := float32(math.Round(x+ph*5)), float32(math.Round(y-ph*10))
		for _, p := range z {
			vector.FillRect(dst, zx+p[0], zy+p[1], 1, 1, color.RGBA{a, a, a, a}, false)
		}
	}
}

// drawBubbles sends a few bubbles up from a crab that went under.
func (g *Game) drawBubbles(dst *ebiten.Image, cam camera, c protocol.CreatureView) {
	sx, sy := cam.toScreen(float64(c.X), float64(c.Y))
	for i := range 3 {
		ph := math.Mod(float64(g.frame)/50+float64(i)*0.37+float64(c.ID)*0.11, 1)
		if ph > 0.6 {
			continue
		}
		a := uint8(200 * (1 - ph/0.6))
		x := math.Round(sx + float64(i*3-3) + math.Sin(ph*9)*1.2)
		y := math.Round(sy - 2 - ph*6)
		vector.FillRect(dst, float32(x), float32(y), 1, 1, color.RGBA{a, a, a, a}, false)
	}
}

// drawLurker draws a crocodile waiting under water: its eyes and nostrils
// riding the surface, bobbing a little, with a faint ring around them.
func (g *Game) drawLurker(dst *ebiten.Image, cam camera, c protocol.CreatureView) {
	sx, sy := cam.toScreen(float64(c.X), float64(c.Y))
	s, _ := g.creatureSprite(c, false)
	bob := math.Round(math.Sin(float64(g.frame)*0.04 + float64(c.ID)))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-s.ax, sy+bob-s.ay)
	op.ColorScale.Scale(0.85, 0.92, 0.95, 1)
	dst.DrawImage(s.img, op)
	if ph := math.Mod(float64(g.frame)/200+float64(c.ID)*0.29, 1); ph < 0.25 {
		r, a := 4+ph/0.25*6, uint8(110*(1-ph/0.25))
		for k := range 24 {
			ang := 2 * math.Pi * float64(k) / 24
			px, py := math.Round(sx+r*math.Cos(ang)*1.4), math.Round(sy-1+r*0.45*math.Sin(ang))
			vector.FillRect(dst, float32(px), float32(py), 1, 1, color.RGBA{a, a, a, a}, false)
		}
	}
}

// drawSeaReptile draws a sea reptile in the water: a dark shape gliding
// below the surface, or reared up with rings spreading round its neck.
func (g *Game) drawSeaReptile(dst *ebiten.Image, cam camera, c protocol.CreatureView, alpha float64) {
	x, y, moving := g.lerpCreature(c, alpha)
	sx, sy := cam.toScreen(x, y)
	s, _ := g.creatureSprite(c, moving)
	bob := math.Round(math.Sin(float64(g.frame)*0.05 + float64(c.ID)))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx-s.ax, sy+bob-s.ay)
	if c.State == sim.CreatureLurk {
		op.ColorScale.ScaleAlpha(0.45) // seen through the water
		dst.DrawImage(s.img, op)
		return
	}
	if c.State == sim.CreatureHurt {
		op.ColorScale.Scale(1, 0.55, 0.55, 1)
	}
	dst.DrawImage(s.img, op)
	ph := math.Mod(float64(g.frame)/90+float64(c.ID)*0.31, 1)
	r, a := 5+ph*9, uint8(130*(1-ph))
	for k := range 28 {
		ang := 2 * math.Pi * float64(k) / 28
		px, py := math.Round(sx+r*math.Cos(ang)*1.3), math.Round(sy+r*0.4*math.Sin(ang))
		vector.FillRect(dst, float32(px), float32(py), 1, 1, color.RGBA{a, a, a, a}, false)
	}
}

// drawFish draws the fish under the water's surface, before anything that
// stands on land.
func (g *Game) drawFish(dst *ebiten.Image, cam camera, alpha float64) {
	for _, c := range g.curr.Creatures {
		if c.Kind == sim.CreatureCrab && c.State == sim.CreatureHidden {
			g.drawBubbles(dst, cam, c) // a crab gone under
			continue
		}
		if c.Kind == sim.CreaturePlesio {
			g.drawSeaReptile(dst, cam, c, alpha)
			continue
		}
		if seaMonster(c.Kind) {
			if c.State == sim.CreatureLurk {
				g.drawSeaShadow(dst, cam, c, alpha)
			}
			continue
		}
		if c.State == sim.CreatureLurk {
			if !g.underIce(c) {
				g.drawLurker(dst, cam, c)
			}
			continue
		}
		if c.Kind != sim.CreatureFish || g.underIce(c) {
			continue
		}
		x, y, moving := g.lerpCreature(c, alpha)
		sx, sy := cam.toScreen(x, y)
		s, _ := g.creatureSprite(c, moving)
		bob := math.Round(math.Sin(float64(g.frame)*0.05 + float64(c.ID)))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-s.ax, sy+bob-s.ay)
		op.ColorScale.Scale(0.86, 0.94, 1, 1) // seen through the water, colours and all
		op.ColorScale.ScaleAlpha(0.9)
		dst.DrawImage(s.img, op)

		// Now and then a ring spreads on the surface above it.
		ph := math.Mod(float64(g.frame)/170+float64(c.ID)*0.37, 1)
		if ph > 0.3 {
			continue
		}
		r := 2 + ph/0.3*7
		a := uint8(150 * (1 - ph/0.3))
		n := int(r * 5)
		for k := range n {
			ang := 2 * math.Pi * float64(k) / float64(n)
			px, py := math.Round(sx+r*math.Cos(ang)), math.Round(sy-3+r*0.45*math.Sin(ang))
			vector.FillRect(dst, float32(px), float32(py), 1, 1, color.RGBA{a, a, a, a}, false)
		}
	}
}
