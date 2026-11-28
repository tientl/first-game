package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The cave fighting's beasts (sim/cavefight.go): PLACEHOLDER art. Every kind
// has one function below that returns its sprite, each marked
// "placeholder art: replaced by the HD painter later". Until then a beast is
// an existing sprite (spider, bear, snake, raptor...) scaled and tinted, and
// its moves show as a stretched, reddened pose in the wind-up, a lunge in the
// blow and a dim one in the recovery, so every fight can be read and tested.
// To replace one, change only its function: it gets the creature as the
// snapshot has it (state, skill, phase) and moving, and returns the sprite.

// phTint recolours a placeholder: hue-neutral multipliers, and how much of
// its colour it keeps (sat) and how bright it is (val).
type phTint struct {
	r, g, b  float32
	sat, val float64
}

var (
	tintBrown  = phTint{1.0, 0.72, 0.5, 0.9, 0.95}
	tintPale   = phTint{1.0, 1.0, 1.0, 0.15, 1.45}
	tintBone   = phTint{1.15, 1.08, 0.9, 0.0, 2.3}
	tintDark   = phTint{0.7, 0.7, 0.8, 0.6, 0.55}
	tintEmber  = phTint{1.3, 0.7, 0.55, 0.9, 0.6}
	tintSand   = phTint{1.15, 0.95, 0.65, 0.7, 1.1}
	tintGreen  = phTint{0.75, 1.1, 0.7, 0.9, 1.0}
	tintPlain  = phTint{1, 1, 1, 1, 1}
	tintBlind  = phTint{1.0, 0.95, 0.95, 0.25, 1.4}
	tintIce    = phTint{0.85, 0.95, 1.2, 0.3, 1.4}
	tintAmber  = phTint{1.15, 0.85, 0.35, 1.1, 1.0}
	tintMarshy = phTint{0.7, 0.85, 0.7, 0.6, 0.8}
)

type phKey struct {
	img    *ebiten.Image
	sx, sy float32
	t      phTint
}

var phCache = map[phKey]sprite{}

// phMake is s scaled by (sx, sy) and tinted.
func phMake(s sprite, sx, sy float64, t phTint) sprite {
	key := phKey{s.img, float32(sx), float32(sy), t}
	if r, ok := phCache[key]; ok {
		return r
	}
	w, h := s.img.Bounds().Dx(), s.img.Bounds().Dy()
	img := ebiten.NewImage(max(1, int(math.Round(float64(w)*sx))), max(1, int(math.Round(float64(h)*sy))))
	var cm colorm.ColorM
	cm.ChangeHSV(0, t.sat, t.val)
	cm.Scale(float64(t.r), float64(t.g), float64(t.b), 1)
	op := &colorm.DrawImageOptions{}
	op.GeoM.Scale(sx, sy)
	colorm.DrawImage(img, s.img, cm, op)
	r := sprite{img: img, ax: s.ax * sx, ay: s.ay * sy, dens: s.dens}
	phCache[key] = r
	return r
}

// placeholder draws c as existing kind base, scaled and tinted, posed by
// the move it is making.
func (g *Game) placeholder(c protocol.CreatureView, moving bool, base sim.CreatureKind, sx, sy float64, t phTint) sprite {
	v := c
	v.Kind, v.Skill, v.Phase, v.Tamed = base, sim.SkillNone, sim.PhaseNone, false
	if v.State == sim.CreatureRoar || v.State == sim.CreatureClaw || v.State == sim.CreatureSpit {
		v.State = sim.CreatureStrike
	}
	if v.State == sim.CreatureSleep && base != sim.CreatureRabbit {
		v.State = sim.CreatureIdle
	}
	s, _ := g.creatureSprite(v, moving)
	switch {
	case c.Skill != sim.SkillNone && c.Phase == sim.PhaseWindup:
		sx, sy = sx*0.94, sy*1.14 // drawn back, rearing
		t.r, t.g, t.b = t.r*1.25, t.g*0.8, t.b*0.8
	case c.Skill != sim.SkillNone && c.Phase == sim.PhaseActive:
		sx, sy = sx*1.12, sy*0.94 // the blow: stretched out
	case c.Skill != sim.SkillNone && c.Phase == sim.PhaseRecover:
		t.val *= 0.8 // spent: open to a counter
	}
	if look := sim.Skill(c.Skill); c.Skill != sim.SkillNone && look.Fx&sim.FxTurtle != 0 && c.Phase != sim.PhaseNone {
		sx, sy = sx*0.6, sy*1.3 // curled into a coil
	}
	return phMake(s, sx, sy, t)
}

// bonePile is a skeleton asleep: its bones in a low heap.
func (g *Game) bonePile(c protocol.CreatureView, base sim.CreatureKind, sx float64) sprite {
	v := c
	v.Kind, v.Skill, v.Phase, v.State = base, sim.SkillNone, sim.PhaseNone, sim.CreatureIdle
	s, _ := g.creatureSprite(v, false)
	return phMake(s, sx, 0.28, tintBone)
}

// skeleton is a walking skeleton: the dinosaur rig in pale bone.
func (g *Game) skeleton(c protocol.CreatureView, moving bool, base sim.CreatureKind, sx float64) sprite {
	if c.State == sim.CreatureSleep || c.State == sim.CreatureRoar {
		if c.State == sim.CreatureRoar { // standing up: half way
			v := c
			v.Kind, v.Skill, v.Phase, v.State = base, sim.SkillNone, sim.PhaseNone, sim.CreatureIdle
			s, _ := g.creatureSprite(v, false)
			return phMake(s, sx, 0.65, tintBone)
		}
		return g.bonePile(c, base, sx)
	}
	return g.placeholder(c, moving, base, sx, sx, tintBone)
}

// --- placeholder art: replaced by the HD painter later (one function a kind) ---

func (g *Game) artArthro(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSnake, 2.6, 1.9, tintBrown)
}
func (g *Game) artArthroQueen(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSnake, 3.6, 2.6, tintAmber)
}
func (g *Game) artCentipede(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSnake, 1.5, 0.9, tintEmber)
}
func (g *Game) artQueenMeadow(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSpider, 2.4, 2.4, tintBrown)
}
func (g *Game) artQueenDesert(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSpider, 2.6, 2.6, tintSand)
}
func (g *Game) artQueenMarsh(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSpider, 2.4, 2.4, tintMarshy)
}
func (g *Game) artQueenSnow(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSpider, 2.4, 2.4, tintPale)
}
func (g *Game) artQueenVolcano(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSpider, 2.5, 2.5, tintEmber)
}
func (g *Game) artSpiderling(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSpider, 0.55, 0.55, tintBrown)
}
func (g *Game) artEggSac(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, false, sim.CreatureRabbit, 0.9, 0.9, tintPale) // a pale lump
}
func (g *Game) artBat(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureRabbit, 0.6, 0.45, tintDark)
}
func (g *Game) artOlm(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSnake, 0.7, 0.6, tintBlind)
}
func (g *Game) artCaveCrab(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureCrab, 1.0, 1.0, tintPale)
}
func (g *Game) artHyena(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureRaptor, 0.95, 0.85, tintSand)
}
func (g *Game) artWhipScorp(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureScorpion, 0.85, 0.85, tintDark)
}
func (g *Game) artCaveLion(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSabre, 0.95, 0.95, tintSand)
}
func (g *Game) artMonitor(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureCroc, 0.95, 0.95, tintGreen)
}
func (g *Game) artAnt(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureCrab, 0.7, 0.7, tintDark)
}
func (g *Game) artAntSoldier(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureCrab, 1.1, 1.1, tintEmber)
}
func (g *Game) artVampBat(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreaturePtero, 0.8, 0.8, tintDark)
}
func (g *Game) artBlindCroc(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureCroc, 1.7, 1.7, tintBlind)
}
func (g *Game) artAlphaBear(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureBear, 1.5, 1.5, tintDark)
}
func (g *Game) artMegalania(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureCroc, 1.8, 1.7, tintEmber)
}
func (g *Game) artLionAlpha(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureSabre, 1.5, 1.5, tintIce)
}
func (g *Game) artScorpQueen(c protocol.CreatureView, m bool) sprite {
	return g.placeholder(c, m, sim.CreatureScorpion, 1.9, 1.9, tintAmber)
}
func (g *Game) artSkelRaptor(c protocol.CreatureView, m bool) sprite {
	return g.skeleton(c, m, sim.CreatureRaptor, 1.0)
}
func (g *Game) artSkelTrike(c protocol.CreatureView, m bool) sprite {
	return g.skeleton(c, m, sim.CreatureTrike, 1.0)
}
func (g *Game) artSkelRex(c protocol.CreatureView, m bool) sprite {
	return g.skeleton(c, m, sim.CreatureRex, 1.0)
}

var caveBeastArt map[sim.CreatureKind]func(*Game, protocol.CreatureView, bool) sprite

func init() {
	caveBeastArt = map[sim.CreatureKind]func(*Game, protocol.CreatureView, bool) sprite{
		sim.CreatureArthro: (*Game).artArthro, sim.CreatureArthroQueen: (*Game).artArthroQueen,
		sim.CreatureCentipede: (*Game).artCentipede, sim.CreatureQueenMeadow: (*Game).artQueenMeadow,
		sim.CreatureQueenDesert: (*Game).artQueenDesert, sim.CreatureQueenMarsh: (*Game).artQueenMarsh,
		sim.CreatureQueenSnow: (*Game).artQueenSnow, sim.CreatureQueenVolcano: (*Game).artQueenVolcano,
		sim.CreatureSpiderling: (*Game).artSpiderling, sim.CreatureEggSac: (*Game).artEggSac,
		sim.CreatureBat: (*Game).artBat, sim.CreatureOlm: (*Game).artOlm, sim.CreatureCaveCrab: (*Game).artCaveCrab,
		sim.CreatureHyena: (*Game).artHyena, sim.CreatureWhipScorp: (*Game).artWhipScorp,
		sim.CreatureCaveLion: (*Game).artCaveLion, sim.CreatureMonitor: (*Game).artMonitor,
		sim.CreatureAnt: (*Game).artAnt, sim.CreatureAntSoldier: (*Game).artAntSoldier,
		sim.CreatureVampBat: (*Game).artVampBat, sim.CreatureBlindCroc: (*Game).artBlindCroc,
		sim.CreatureAlphaBear: (*Game).artAlphaBear, sim.CreatureMegalania: (*Game).artMegalania,
		sim.CreatureLionAlpha: (*Game).artLionAlpha, sim.CreatureScorpQueen: (*Game).artScorpQueen,
		sim.CreatureSkelRaptor: (*Game).artSkelRaptor, sim.CreatureSkelTrike: (*Game).artSkelTrike,
		sim.CreatureSkelRex: (*Game).artSkelRex,
	}
}

// caveSprite is the sprite of a cave beast; ok is false for any other kind.
func (g *Game) caveSprite(c protocol.CreatureView, moving bool) (sprite, float64, bool) {
	f, ok := caveBeastArt[c.Kind]
	if !ok {
		return sprite{}, 0, false
	}
	bob := 0.0
	if c.Kind == sim.CreatureBat || c.Kind == sim.CreatureVampBat {
		bob = 8 + math.Round(math.Sin(float64(g.frame)*0.3+float64(c.ID))*2) // flying
		if c.State == sim.CreatureSleep {
			bob = 0
		}
	}
	return f(g, c, moving), bob, true
}

// caveBeastShadow is the width of the shadow under a cave beast; 0 for others.
func caveBeastShadow(k sim.CreatureKind) float64 {
	switch k {
	case sim.CreatureArthroQueen, sim.CreatureBlindCroc, sim.CreatureMegalania, sim.CreatureSkelRex:
		return 44
	case sim.CreatureArthro, sim.CreatureAlphaBear, sim.CreatureQueenMeadow, sim.CreatureQueenDesert,
		sim.CreatureQueenMarsh, sim.CreatureQueenSnow, sim.CreatureQueenVolcano, sim.CreatureScorpQueen, sim.CreatureLionAlpha, sim.CreatureSkelTrike:
		return 34
	case sim.CreatureBat, sim.CreatureOlm, sim.CreatureSpiderling:
		return 8
	}
	if k.IsCaveBeast() {
		return 16
	}
	return 0
}

// caveLabel is the target label of a cave beast.
func caveLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	if !c.Kind.IsCaveBeast() {
		return "", colWhite, false
	}
	name := c.Kind.Name()
	switch {
	case c.Kind.IsSkeleton():
		if c.State == sim.CreatureSleep {
			return "Đống xương cổ — đừng lại gần", colBad, true
		}
		return name + " — búa nặng và lửa làm nó vỡ; giáo trượt qua. " + sim.SkeletonFlavour, colBad, true
	case c.Kind == sim.CreatureOlm || c.Kind == sim.CreatureCaveCrab:
		return name + " — hiền, bắt để ăn", colGood, true
	case c.Kind == sim.CreatureEggSac:
		return name + " — đập vỡ trước khi nở!", colBad, true
	case c.Kind == sim.CreatureArthro || c.Kind == sim.CreatureArthroQueen:
		return name + " — đánh lúc nó duỗi ra, khi cuộn tròn thì giáp cứng", colBad, true
	case c.Kind == sim.CreatureCentipede:
		return name + " — nhanh, sợ lửa đuốc; cắn có độc", colBad, true
	case c.Kind == sim.CreatureBat:
		return name + " — bay tán loạn khi có đuốc lại gần", colWhite, true
	case c.Kind.IsCaveBoss():
		return name + " — chúa hang: đánh khi nó vừa ra chiêu xong", colBad, true
	}
	return name + " — đánh", colBad, true
}

// caveBeastEyes is how the eyes of a cave beast shine in the dark; nil for
// none (the bones have none).
func caveBeastEyes(k sim.CreatureKind) *caveEyes {
	switch {
	case !k.IsCaveBeast(), k.IsSkeleton(), k == sim.CreatureOlm, k == sim.CreatureCaveCrab, k == sim.CreatureEggSac:
		return nil
	case k == sim.CreatureBat || k == sim.CreatureVampBat:
		return &batEyes
	case k.Material() == sim.MatFur:
		return &bearEyes
	}
	return &spiderEyes
}

// caveVoices are the voices of the cave beasts, from the existing bank.
func caveVoices(k sim.CreatureKind) kindVoices {
	switch {
	case k.IsSkeleton():
		return kindVoices{alarm: say("whack", 0.5, 1.6), hurt: say("whack", 0.45, 1.9), death: say("whack", 0.6, 0.7)} // a clatter of bones
	case k == sim.CreatureBat || k == sim.CreatureVampBat:
		return kindVoices{alarm: say("spider_chitter", 0.4, 1.8), hurt: say("spider_chitter", 0.4, 2.0), death: say("spider_chitter", 0.4, 1.4)}
	case k == sim.CreatureAlphaBear:
		return kindVoices{hurt: say("bellow_hurt", 0.6, 0.8), death: say("beast_death", 0.8, 0.8)}
	case k == sim.CreatureHyena || k == sim.CreatureCaveLion || k == sim.CreatureLionAlpha:
		return kindVoices{hurt: say("bellow_hurt", 0.5, 1.3), death: say("beast_death", 0.6, 1.2)}
	case k == sim.CreatureMonitor || k == sim.CreatureMegalania || k == sim.CreatureBlindCroc:
		return voicesOf(sim.CreatureCroc)
	case k == sim.CreatureWhipScorp || k == sim.CreatureScorpQueen:
		return voicesOf(sim.CreatureScorpion)
	case k.IsCaveBeast() && k.Material() == sim.MatChitin:
		return voicesOf(sim.CreatureSpider)
	}
	return kindVoices{}
}

// --- hazards and the boss's bar --------------------------------------------------

// drawHazards draws the clouds, webs and collapses of the snapshot.
func (g *Game) drawHazards(dst drawTarget, cam camera, alpha float64) {
	if len(g.curr.Hazards) == 0 {
		return
	}
	alignCamera(dst)
	for _, h := range g.curr.Hazards {
		z := protocol.ZoneView{Shape: sim.ZoneCircle, X: h.X, Y: h.Y, Reach: h.R}
		switch h.Kind {
		case sim.HazPoison:
			pulse := 0.5 + 0.5*math.Sin(float64(g.frame)*0.08+float64(h.X))
			zoneShape(dst, cam, z, 1, scaleAlpha(color.RGBA{70, 170, 50, 255}, 0.26+0.12*pulse))
			zoneShape(dst, cam, z, 0.6, scaleAlpha(color.RGBA{110, 200, 70, 255}, 0.2))
		case sim.HazDust:
			zoneShape(dst, cam, z, 1, scaleAlpha(color.RGBA{190, 160, 110, 255}, 0.3))
		case sim.HazWeb:
			zoneShape(dst, cam, z, 1, scaleAlpha(color.RGBA{230, 230, 235, 255}, 0.22))
			zoneEdgeLine(dst, cam, z, scaleAlpha(color.RGBA{240, 240, 245, 255}, 0.55))
			zoneShape(dst, cam, z, 0.55, scaleAlpha(color.RGBA{240, 240, 245, 255}, 0.18))
		case sim.HazCollapse:
			total := max(1, float64(h.Total))
			k := 1 - max(0, float64(h.Left)-alpha)/total
			zoneShape(dst, cam, z, 1, scaleAlpha(zoneFaint, 0.26+0.1*k))
			zoneShape(dst, cam, z, easeInOut(k), scaleAlpha(zoneDeep, 0.3+0.2*k))
			zoneEdgeLine(dst, cam, z, scaleAlpha(zoneEdge, 0.5+0.4*k))
			// dust trickling from the roof
			for i := range 9 {
				ph := math.Mod(float64(g.frame)/20+float64(i)*0.37, 1)
				a := float64(i)*2.4 + float64(h.X)
				px := (float64(h.X) + math.Cos(a)*float64(h.R)*0.8*math.Mod(float64(i)*0.31, 1)) * TileW
				py := (float64(h.Y)+math.Sin(a)*float64(h.R)*0.5)*TileH - 14 + ph*14
				fillRect(dst, float32(math.Round(px-cam.ox)), float32(math.Round(py-cam.oy)), 1, 2, color.RGBA{170, 150, 120, 200}, false)
			}
		}
	}
}

// drawCaveBossBar draws a small health bar over a cave boss that is fighting.
func (g *Game) drawCaveBossBar(dst drawTarget, c protocol.CreatureView, d drawable, s sprite) {
	if !c.Kind.IsCaveBoss() || c.HP >= c.MaxHP && c.State != sim.CreatureChase && c.Skill == sim.SkillNone {
		return
	}
	w := float32(34)
	x, y := float32(math.Round(d.sx))-w/2, float32(math.Round(d.sy-s.ay))-5
	fillRect(dst, x-1, y-1, w+2, 4, color.RGBA{10, 8, 8, 230}, false)
	fillRect(dst, x, y, w*float32(c.HP)/float32(max(1, int(c.MaxHP))), 2, color.RGBA{200, 50, 40, 255}, false)
}
