package sim

import (
	"math"
)

// Fighting in the caves (StyleCaveFight): the caves are no longer only dark
// and empty. Every beast below is one row of caveSpecs: its numbers, a few
// traits and its moves (skills.go: a wind-up that shows what is coming, a
// blow, a recovery that is the moment to strike back). One step
// (stepCaveBeast) drives them all by their traits: an ambusher keeps still
// till someone is close, a pack circles and goes in by turns, a hunter that
// fears fire keeps out of torchlight, a harmless thing runs.
//
//   - cavespawn.go lays them out when the caves are dressed: encounters in
//     the chambers and ambushes in the tunnels by island kind and level,
//     a queen in her nest, a boss on the hoard at the end of the main route,
//     skeletons in the fossil chambers.
//   - cavehaz.go is what lies on the cave floor and falls from its roof: web,
//     poison and dust clouds, collapses.
//
// The look of each beast is a placeholder in the client (cavecreatures.go).

// caveTrait is how a cave beast behaves.
type caveTrait uint16

const (
	trAmbush   caveTrait = 1 << iota // keeps still, unseen, till its foe is close
	trFearFire                       // keeps out of torchlight
	trPack                           // hunts in a pack that takes turns
	trHarmless                       // runs from people; never fights
	trBoss                           // a cave boss: stands guard, mends
	trSkeleton                       // lies as a pile of bones till someone is near
	trLegShed                        // loses a leg to every hard blow, and slows
	trStill                          // does not move at all (an egg sac)
	trSwarm                          // a bat: roosts, bursts from the torch
	trNoBody                         // falls to nothing worth butchering: its loot drops at once
)

// caveSpec is one kind of cave beast.
type caveSpec struct {
	kind       CreatureKind
	name       string
	hp         int     // in hits, as every creature's
	walk, run  float64 // tiles a second
	radius     float64
	reach      float64
	yield      ItemKind
	yieldN     int
	drops      []Ingredient
	sight      float64 // it notices a player this close
	giveUp     float64 // and gives up this far
	traits     caveTrait
	herd       [2]int // appears in groups of this many, least and most; zero: alone
	leash      float64
	amphibious bool
	tough      bool
	material   Material
	skills     []SkillDef
}

func (s *caveSpec) has(t caveTrait) bool { return s.traits&t != 0 }

// Material is what a beast's body is made of, for the effects of a blow on
// it: sparks, blood, dust or splinters.
type Material uint8

const (
	MatFlesh  Material = iota // bare skin and meat: blood
	MatFur                    // fur over flesh: blood and hair
	MatScale                  // scales and hide
	MatChitin                 // an insect's or spider's shell: pale ichor and chips
	MatBone                   // bone: dust and splinters
)

// Material is what k's body is made of.
func (k CreatureKind) Material() Material {
	if s := caveSpecOf(k); s != nil {
		return s.material
	}
	switch k {
	case CreatureBear, CreatureSabre, CreatureMammoth, CreatureRabbit:
		return MatFur
	case CreatureSpider, CreatureScorpion, CreatureCrab:
		return MatChitin
	case CreatureSnake, CreatureCroc, CreatureRaptor, CreatureRex, CreatureTrike, CreatureStego, CreatureAnkylo:
		return MatScale
	}
	return MatFlesh
}

// SkeletonFlavour is what the player is told about the walking skeletons:
// the one place to change once it is decided what drives them.
const SkeletonFlavour = "Bộ xương cổ tự dựng dậy khi có người lại gần. Búa nặng và lửa làm nó vỡ; giáo và dao trượt qua kẽ xương."

// caveVars is what a cave beast keeps besides what every creature has.
type caveVars struct {
	legs  uint8    // legs a centipede has shed
	guard EntityID // the hoard a boss stands on
	hatch int      // ticks an egg sac has sat with a foe near
	swarm int      // ticks a bat has been bursting about
}

// Cave tuning.
const (
	SkeletonRiseDist = 4.2 // a pile of bones rises when someone comes this close
	skeletonRiseTime = 24  // ticks it takes to stand up
	SkeletonBluntK   = 1.7 // what a heavy, blunt blow does to a skeleton
	SkeletonFireK    = 2.4 // and a torch
	SkeletonSharpK   = 0.55
	ExposedK         = 1.5 // a blow on a beast caught in the recovery of a big move
	LegShedHits      = 15  // a blow this heavy knocks a centipede's leg off
	LegsMax          = 4
	caveBossMendEvery = TicksPerSecond / 2
	eggHatchTicks    = 8 * TicksPerSecond
	eggHatchNear     = 9.0
	caveRespawnTicks = 90 * TicksPerSecond
)

var caveSpecs [NumCreatureKinds]*caveSpec

func caveSpecOf(k CreatureKind) *caveSpec {
	if k >= NumCreatureKinds {
		return nil
	}
	return caveSpecs[k]
}

// IsCaveBeast reports whether k is one of the cave fighting's beasts.
func (k CreatureKind) IsCaveBeast() bool { return caveSpecOf(k) != nil }

// IsCaveBoss reports whether k is a cave boss (a queen's own boss bar).
func (k CreatureKind) IsCaveBoss() bool {
	s := caveSpecOf(k)
	return s != nil && (s.has(trBoss) || k.IsSpiderQueen())
}

// IsSpiderQueen reports whether k is a spider queen.
func (k CreatureKind) IsSpiderQueen() bool {
	return k >= CreatureQueenMeadow && k <= CreatureQueenVolcano
}

// IsSkeleton reports whether k is a walking skeleton.
func (k CreatureKind) IsSkeleton() bool {
	return k == CreatureSkelRaptor || k == CreatureSkelTrike || k == CreatureSkelRex
}

// --- building the moves -------------------------------------------------------

func sk(name, label string, cl SkillClass, wind, act, rec int, shape ZoneShape, reach, width, dmg float64, cd int, minD, maxD float64, weight int) SkillDef {
	return SkillDef{Name: name, Label: label, Class: cl, Windup: wind, Active: act, Recover: rec,
		Shape: shape, Reach: reach, Width: width, Damage: dmg, Cooldown: cd, MinD: minD, MaxD: maxD,
		Weight: weight, Lock: max(2, wind/4)}
}

func (d SkillDef) with(f func(*SkillDef)) SkillDef { f(&d); return d }

// scaled is the moves of a stronger variant: damage times k.
func scaled(defs []SkillDef, k float64) []SkillDef {
	out := append([]SkillDef(nil), defs...)
	for i := range out {
		out[i].Damage = math.Round(out[i].Damage * k)
	}
	return out
}

func ram(dmg, reach float64) SkillDef {
	return sk("ram", "Lao húc", SkillHeavy, 18, 14, 22, ZoneLine, reach, 0.8, dmg, 120, 2.5, reach, 6).with(func(d *SkillDef) {
		d.Zone, d.Knock, d.Motion, d.Speed, d.Lock = true, 2, MoveDash, 0.3, -1
		d.Fx = FxExpose
	})
}

func tailSweep(label string, dmg, reach float64) SkillDef {
	return sk("tail", label, SkillLight, 10, 3, 14, ZoneCone, reach, 1.2, dmg, 60, 0, reach, 8).with(func(d *SkillDef) {
		d.Behind, d.Knock, d.Lock = true, 1.5, -1
	})
}

func venomBite(dmg, reach float64) SkillDef {
	return sk("bite", "Cắn nọc độc", SkillHeavy, 18, 3, 18, ZoneCone, reach, 0.8, dmg, 70, 0, reach, 7).with(func(d *SkillDef) {
		d.Fx, d.Lock = FxVenom, 4
	})
}

func roarStun(r float64, stun int) SkillDef {
	return sk("roar", "Gầm choáng", SkillHeavy, 20, 6, 16, ZoneCircle, r, 0, 0, 300, 0, r, 2).with(func(d *SkillDef) {
		d.Zone, d.Stun, d.Lock, d.Wind = true, stun, -1, CreatureRoar
		d.Need = NeedAngry
	})
}

func init() {
	spiderQueenBase := []SkillDef{
		venomBite(14, 2.4),
		sk("web", "Phun tơ", SkillLight, 12, 4, 14, ZoneCone, 6, 0.3, 0, 90, 2.5, 6, 6).with(func(d *SkillDef) {
			d.Fx, d.Slow, d.Lock = FxEnsnare, 60, 3
		}),
		sk("drop", "Thả mình từ trần", SkillSpecial, 28, 8, 22, ZoneCircle, 1.9, 0, 22, 220, 3, 7, 5).with(func(d *SkillDef) {
			d.Zone, d.Motion, d.Anchor, d.Stun, d.Lock = true, MoveLeap, AtTarget, 12, 6
		}),
		sk("pounce", "Vồ", SkillHeavy, 14, 8, 16, ZoneCircle, 1.0, 0, 12, 140, 2.5, 5, 6).with(func(d *SkillDef) {
			d.Fx, d.Motion, d.Anchor, d.Lock, d.Stagger = FxPin, MoveLeap, AtTarget, 5, false
		}),
		sk("webfloor", "Giăng tơ dưới sàn", SkillSpecial, 24, 4, 16, ZoneCircle, 3, 0, 0, 260, 2, 6, 5).with(func(d *SkillDef) {
			d.Zone, d.Anchor, d.Fx, d.Lock = true, AtTarget, FxWebFloor, -1
		}),
		sk("hatch", "Gọi nhện con", SkillCry, 16, 2, 10, ZoneNone, 0, 0, 0, 600, 0, 99, 14).with(func(d *SkillDef) {
			d.Fx, d.Need, d.HPBelow, d.Lock = FxSummon, NeedAngry, 0.75, -1
		}),
	}
	queen := func(k CreatureKind, name string, mut func(defs []SkillDef) []SkillDef, mat Material, amph bool, drops []Ingredient) *caveSpec {
		return &caveSpec{kind: k, name: name, hp: 70, walk: 1.2, run: 3.4, radius: 0.7, reach: 2.4,
			yield: ItemSilk, yieldN: 4, drops: drops, sight: 8, giveUp: 14, traits: trBoss, leash: 6,
			amphibious: amph, tough: true, material: mat, skills: mut(scaled(spiderQueenBase, 1))}
	}
	gland := []Ingredient{{ItemVenomGland, 1}, {ItemChitin, 2}}
	same := func(d []SkillDef) []SkillDef { return d }
	arthroSkills := []SkillDef{
		ram(20, 6),
		tailSweep("Quật đuôi", 12, 2.8),
		sk("spray", "Phun dịch độc", SkillSpecial, 28, 4, 20, ZoneCircle, 3.2, 0, 0, 300, 0, 4.5, 12).with(func(d *SkillDef) {
			d.Zone, d.Fx, d.HPBelow, d.Need, d.Lock = true, FxCloudPoison, 0.65, NeedAngry, -1
		}),
		sk("coil", "Cuộn tròn", SkillCry, 10, 70, 24, ZoneNone, 0, 0, 0, 500, 0, 5, 14).with(func(d *SkillDef) {
			d.Fx, d.HPBelow, d.Need, d.Lock = FxTurtle|FxExpose, 0.8, NeedAngry, -1
		}),
	}
	queenArthro := append(scaled(arthroSkills, 1.5), sk("stomp", "Dậm nhiều chân", SkillSpecial, 26, 5, 20, ZoneCircle, 3.6, 0, 24, 240, 0, 3.6, 8).with(func(d *SkillDef) {
		d.Zone, d.Stun, d.Lock, d.Fx = true, 14, -1, FxExpose
	}))
	for i := range queenArthro {
		if queenArthro[i].Name == "ram" {
			queenArthro[i].Reach = 7
		}
		if queenArthro[i].Name == "spray" {
			queenArthro[i].Reach, queenArthro[i].HPBelow = 4.2, 0.75
		}
	}
	monitorBase := []SkillDef{
		tailSweep("Quật đuôi", 12, 3),
		sk("bite", "Cắn", SkillHeavy, 16, 3, 18, ZoneCone, 2.4, 0.7, 14, 70, 0, 2.4, 7).with(func(d *SkillDef) { d.Bleed, d.Lock = 60, 4 }),
		sk("breath", "Phun hơi nóng", SkillSpecial, 26, 4, 20, ZoneCone, 5, 0.5, 16, 220, 2, 5, 6).with(func(d *SkillDef) {
			d.Zone, d.Bleed, d.Lock = true, 60, -1
		}),
	}
	lionBase := []SkillDef{
		sk("claws", "Vuốt đôi", SkillLight, 9, 6, 14, ZoneCone, 2.0, 0.9, 9, 50, 0, 2, 7).with(func(d *SkillDef) { d.Hits, d.Every, d.Lock = 2, 4, 3 }),
		sk("pounce", "Vồ rình", SkillHeavy, 14, 8, 16, ZoneCircle, 1.1, 0, 18, 120, 2.5, 5.5, 6).with(func(d *SkillDef) {
			d.Motion, d.Anchor, d.Knock, d.Lock, d.Stagger = MoveLeap, AtTarget, 1.5, 5, true
		}),
		sk("bite", "Cắn xé", SkillHeavy, 16, 2, 16, ZoneCone, 1.9, 0.7, 14, 90, 0, 1.9, 5).with(func(d *SkillDef) { d.Bleed, d.Lock = 80, 4 }),
	}
	vampSkills := []SkillDef{
		sk("swoop", "Sà xuống", SkillHeavy, 14, 8, 20, ZoneCircle, 1.2, 0, 12, 120, 3, 7, 7).with(func(d *SkillDef) {
			d.Motion, d.Anchor, d.Lock, d.Stagger = MoveLeap, AtTarget, 5, true
		}),
		sk("drain", "Hút máu", SkillLight, 9, 2, 16, ZoneCone, 1.8, 0.8, 8, 60, 0, 1.8, 8).with(func(d *SkillDef) { d.Fx, d.Lock = FxDrain, 3 }),
		sk("screech", "Rít chói tai", SkillSpecial, 22, 5, 16, ZoneCircle, 4, 0, 0, 300, 0, 4, 3).with(func(d *SkillDef) {
			d.Zone, d.Stun, d.Lock, d.Need = true, 10, -1, NeedAngry
		}),
	}
	skelRaptor := []SkillDef{
		sk("claw", "Vuốt xương", SkillLight, 8, 6, 12, ZoneCone, 1.9, 0.9, 7, 50, 0, 1.9, 6).with(func(d *SkillDef) { d.Hits, d.Every, d.Lock, d.Stagger = 2, 5, 3, true }),
		sk("pounce", "Vồ", SkillHeavy, 12, 8, 14, ZoneCircle, 1.0, 0, 8, 100, 2.2, 4.2, 6).with(func(d *SkillDef) {
			d.Motion, d.Anchor, d.Knock, d.Lock, d.Stagger = MoveLeap, AtTarget, 1.2, 5, true
		}),
		sk("rattle", "Lọc cọc gọi bầy", SkillCry, 12, 2, 6, ZoneNone, 0, 0, 0, 300, 2.5, 9, 20).with(func(d *SkillDef) {
			d.Fx, d.Need, d.Stagger, d.Lock = FxFlank, NeedLeader, true, -1
		}),
	}
	rexSkills := []SkillDef{
		sk("bite", "Cắn nghiền", SkillHeavy, 16, 2, 16, ZoneCone, 3.4, 0.7, 26, 80, 0, 3.4, 6).with(func(d *SkillDef) { d.Lock = 4 }),
		tailSweep("Quật đuôi xương", 20, 3.8),
		sk("stomp", "Dậm chân", SkillSpecial, 26, 5, 20, ZoneCircle, 4.5, 0, 14, 240, 0, 4.5, 4).with(func(d *SkillDef) {
			d.Zone, d.Stun, d.Lock, d.Fx = true, 14, -1, FxExpose
		}),
		sk("reassemble", "Gom xương lại", SkillCry, 24, 50, 10, ZoneNone, 0, 0, 0, 900, 0, 8, 16).with(func(d *SkillDef) {
			d.Fx, d.HPBelow, d.Need, d.Lock = FxTurtle, 0.35, NeedAngry, -1
		}),
	}
	bear := []SkillDef{
		sk("rear", "Đứng lên vả", SkillSpecial, 26, 4, 20, ZoneCone, 3.2, 1.1, 30, 200, 0, 3.2, 6).with(func(d *SkillDef) {
			d.Zone, d.Knock, d.Lock, d.Fx = true, 2.5, -1, FxExpose
		}),
		roarStun(4.5, 14),
	}
	bear[1].Weight = 3
	alphaBear := append(scaled(bear, 1.3), sk("swipe", "Vả mạnh", SkillLight, 10, 3, 14, ZoneCone, 2.4, 1.0, 18, 50, 0, 2.4, 8).with(func(d *SkillDef) { d.Lock = 3 }),
		sk("maul", "Cắn xé", SkillHeavy, 18, 3, 18, ZoneCone, 2.4, 0.8, 26, 90, 0, 2.4, 6).with(func(d *SkillDef) { d.Lock = 4 }))
	scorpQueen := []SkillDef{
		sk("pinch", "Kẹp", SkillHeavy, 16, 3, 18, ZoneCone, 2.4, 0.9, 16, 70, 0, 2.4, 7).with(func(d *SkillDef) { d.Lock = 4 }),
		sk("sting", "Chích độc", SkillSpecial, 24, 4, 22, ZoneCone, 3.2, 0.5, 20, 160, 0, 3.2, 6).with(func(d *SkillDef) { d.Zone, d.Fx, d.Lock = true, FxVenom, -1 }),
		tailSweep("Quật đuôi", 14, 3),
		sk("brood", "Gọi bọ cạp", SkillCry, 16, 2, 10, ZoneNone, 0, 0, 0, 600, 0, 99, 14).with(func(d *SkillDef) {
			d.Fx, d.Need, d.HPBelow, d.Lock = FxSummon, NeedAngry, 0.65, -1
		}),
	}
	crocBase := []SkillDef{
		sk("bite", "Cắn gãy", SkillHeavy, 18, 3, 20, ZoneCone, 2.6, 0.8, 28, 80, 0, 2.6, 7).with(func(d *SkillDef) { d.Lock = 4 }),
		tailSweep("Quật đuôi", 16, 3),
		sk("lunge", "Lao ra khỏi nước", SkillHeavy, 16, 12, 22, ZoneLine, 5, 0.7, 22, 140, 2.5, 5.5, 6).with(func(d *SkillDef) {
			d.Zone, d.Knock, d.Motion, d.Speed, d.Lock, d.Fx = true, 1.5, MoveDash, 0.4, -1, FxExpose
		}),
		sk("roll", "Cắn kéo xoay", SkillHeavy, 18, 8, 20, ZoneCircle, 1.1, 0, 10, 220, 2, 4.5, 4).with(func(d *SkillDef) {
			d.Fx, d.Motion, d.Anchor, d.Lock = FxPin, MoveLeap, AtTarget, 6
		}),
	}
	trikeSkel := []SkillDef{
		sk("charge", "Lao húc", SkillSpecial, 24, 16, 22, ZoneLine, 7, 0.9, 22, 160, 3, 7, 7).with(func(d *SkillDef) {
			d.Zone, d.Knock, d.Motion, d.Speed, d.Lock, d.Fx = true, 2.5, MoveDash, 0.36, -1, FxExpose
		}),
		sk("toss", "Hất sừng", SkillHeavy, 14, 3, 16, ZoneCone, 2.2, 0.8, 16, 70, 0, 2.2, 6).with(func(d *SkillDef) { d.Knock, d.Lock = 2, 4 }),
	}
	centipede := []SkillDef{
		sk("lunge", "Phóng cắn", SkillLight, 8, 8, 16, ZoneLine, 3.4, 0.5, 6, 80, 1.5, 3.4, 8).with(func(d *SkillDef) {
			d.Fx, d.Motion, d.Speed, d.Lock, d.Stagger = FxVenom, MoveDash, 0.5, 3, true
		}),
		sk("bite", "Cắn độc", SkillLight, 8, 2, 12, ZoneCone, 1.4, 0.8, 5, 30, 0, 1.4, 6).with(func(d *SkillDef) {
			d.Fx, d.Motion, d.Back, d.Lock, d.Stagger = FxVenom, MoveHopBack, 2, 2, true
		}),
	}
	whip := []SkillDef{
		sk("strike", "Phóng ra", SkillLight, 8, 7, 14, ZoneLine, 3, 0.5, 9, 70, 1.5, 3.2, 7).with(func(d *SkillDef) {
			d.Motion, d.Speed, d.Lock, d.Stagger = MoveDash, 0.55, 3, true
		}),
		sk("whip", "Quất roi", SkillLight, 9, 2, 14, ZoneCone, 1.7, 0.8, 7, 40, 0, 1.7, 6).with(func(d *SkillDef) {
			d.Motion, d.Back, d.Lock, d.Stagger = MoveHopBack, 1.5, 2, true
		}),
	}
	hyena := []SkillDef{
		sk("bite", "Cắn xé", SkillLight, 9, 2, 16, ZoneCone, 1.7, 0.8, 9, 40, 0, 1.7, 7).with(func(d *SkillDef) { d.Bleed, d.Lock, d.Stagger = 40, 3, true }),
		sk("leap", "Vồ", SkillHeavy, 12, 8, 14, ZoneCircle, 1.0, 0, 8, 110, 2.2, 4, 5).with(func(d *SkillDef) {
			d.Motion, d.Anchor, d.Knock, d.Lock, d.Stagger = MoveLeap, AtTarget, 1, 5, true
		}),
		sk("howl", "Hú gọi bầy", SkillCry, 12, 2, 6, ZoneNone, 0, 0, 0, 300, 2.5, 9, 20).with(func(d *SkillDef) {
			d.Fx, d.Need, d.Stagger, d.Lock = FxFlank, NeedLeader, true, -1
		}),
	}
	nip := func(dmg float64) []SkillDef {
		return []SkillDef{sk("nip", "Cắn nhẹ", SkillLight, 8, 2, 20, ZoneCone, 1.0, 0.8, dmg, 80, 0, 1.0, 1).with(func(d *SkillDef) {
			d.Motion, d.Back, d.Lock, d.Stagger = MoveHopBack, 1.5, 2, true
		}),
			sk("dart", "Lao tới cắn", SkillLight, 8, 6, 16, ZoneLine, 2.4, 0.4, dmg, 120, 1.2, 2.4, 1).with(func(d *SkillDef) {
				d.Motion, d.Speed, d.Lock, d.Stagger = MoveDash, 0.4, 3, true
			})}
	}

	add := func(s *caveSpec) { caveSpecs[s.kind] = s }
	add(&caveSpec{kind: CreatureArthro, name: "Rết khổng lồ Arthropleura", hp: 40, walk: 1.0, run: 3.0, radius: 0.7, reach: 2.2,
		yield: ItemMeat, yieldN: 4, drops: []Ingredient{{ItemChitin, 3}}, sight: 6, giveUp: 12, leash: 9, tough: true,
		material: MatChitin, skills: arthroSkills})
	add(&caveSpec{kind: CreatureArthroQueen, name: "Chúa rết Arthropleura", hp: 110, walk: 1.0, run: 3.2, radius: 0.9, reach: 2.8,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemChitin, 8}, {ItemVenomGland, 1}}, sight: 8, giveUp: 15, leash: 6, tough: true,
		traits: trBoss, material: MatChitin, skills: queenArthro})
	add(&caveSpec{kind: CreatureCentipede, name: "Rết nhà khổng lồ", hp: 8, walk: 2.0, run: 5.8, radius: 0.3, reach: 1.5,
		yield: ItemMeat, yieldN: 1, drops: []Ingredient{{ItemCentipedeClaw, 1}}, sight: 6, giveUp: 12, leash: 8,
		traits: trAmbush | trFearFire | trLegShed | trNoBody, material: MatChitin, skills: centipede})
	add(queen(CreatureQueenMeadow, "Nhện chúa vằn nâu", same, MatChitin, false, gland))
	add(queen(CreatureQueenDesert, "Nhện chúa lông (tarantula)", func(d []SkillDef) []SkillDef {
		d[1] = sk("flick", "Phất lông ngứa", SkillSpecial, 24, 4, 16, ZoneCircle, 2.4, 0, 0, 160, 1.5, 5, 6).with(func(d *SkillDef) {
			d.Zone, d.Anchor, d.Offset, d.Fx, d.Slow, d.Lock = true, AtAhead, 2.2, FxCloudDust, 80, -1
		})
		return d
	}, MatChitin, false, gland))
	add(queen(CreatureQueenMarsh, "Nhện chúa săn cá", same, MatChitin, true, gland))
	add(queen(CreatureQueenSnow, "Nhện chúa trắng", func(d []SkillDef) []SkillDef { d[0].Slow = 50; return d }, MatChitin, false, gland))
	add(queen(CreatureQueenVolcano, "Nhện chúa đen", func(d []SkillDef) []SkillDef { d[0].Bleed = 60; return d }, MatChitin, false, gland))
	add(&caveSpec{kind: CreatureSpiderling, name: "Nhện con", hp: 2, walk: 1.6, run: 3.6, radius: 0.2, reach: 1.2,
		yield: ItemSilk, yieldN: 1, sight: 7, giveUp: 11, leash: 8, herd: [2]int{2, 3}, traits: trNoBody, material: MatChitin,
		skills: []SkillDef{sk("bite", "Cắn", SkillLight, 8, 2, 14, ZoneCone, 1.1, 0.8, 3, 40, 0, 1.1, 1).with(func(d *SkillDef) { d.Lock, d.Stagger = 2, true }), nip(3)[1]}})
	add(&caveSpec{kind: CreatureEggSac, name: "Ổ trứng nhện", hp: 4, radius: 0.4, reach: 1.5, yield: ItemSilk, yieldN: 1,
		traits: trStill | trNoBody, material: MatChitin})
	add(&caveSpec{kind: CreatureBat, name: "Dơi hang", hp: 1, walk: 2.0, run: 6.0, radius: 0.2, reach: 1.2,
		yield: ItemGuano, yieldN: 1, sight: 5, giveUp: 11, leash: 6, herd: [2]int{4, 6}, traits: trSwarm | trNoBody, material: MatFur, skills: nip(1.5)})
	add(&caveSpec{kind: CreatureOlm, name: "Cá lưỡng cư mù", hp: 1, walk: 0.5, run: 1.6, radius: 0.2, reach: 1.4,
		yield: ItemOlm, yieldN: 1, leash: 4, traits: trHarmless | trNoBody, material: MatFlesh})
	add(&caveSpec{kind: CreatureCaveCrab, name: "Cua hang trắng", hp: 1, walk: 0.8, run: 2.6, radius: 0.2, reach: 1.4,
		yield: ItemCrabMeat, yieldN: 1, leash: 4, traits: trHarmless | trNoBody, material: MatChitin})
	add(&caveSpec{kind: CreatureHyena, name: "Linh cẩu hang", hp: 10, walk: 1.6, run: 4.6, radius: 0.4, reach: 1.6,
		yield: ItemMeat, yieldN: 2, drops: []Ingredient{{ItemFur, 1}}, sight: 8, giveUp: 14, leash: 10, herd: [2]int{2, 3},
		traits: trPack | trFearFire, material: MatFur, skills: hyena})
	add(&caveSpec{kind: CreatureWhipScorp, name: "Bọ cạp roi", hp: 9, walk: 1.4, run: 5.0, radius: 0.35, reach: 1.6,
		yield: ItemMeat, yieldN: 1, drops: []Ingredient{{ItemChitin, 1}}, sight: 6, giveUp: 11, leash: 8,
		traits: trAmbush | trNoBody, material: MatChitin, skills: whip})
	add(&caveSpec{kind: CreatureCaveLion, name: "Sư tử hang", hp: 40, walk: 1.4, run: 4.8, radius: 0.55, reach: 2.0,
		yield: ItemMeat, yieldN: 5, drops: []Ingredient{{ItemFur, 3}, {ItemBone, 2}}, sight: 8, giveUp: 14, leash: 10,
		traits: trAmbush, material: MatFur, skills: lionBase})
	add(&caveSpec{kind: CreatureMonitor, name: "Kỳ đà hang", hp: 30, walk: 1.1, run: 3.4, radius: 0.55, reach: 2.0,
		yield: ItemMeat, yieldN: 4, drops: []Ingredient{{ItemDinoHide, 1}}, sight: 6, giveUp: 12, leash: 9, tough: true, material: MatScale, skills: monitorBase})
	add(&caveSpec{kind: CreatureAnt, name: "Kiến thợ khổng lồ", hp: 3, walk: 1.8, run: 3.6, radius: 0.2, reach: 1.2,
		yield: ItemMeat, yieldN: 1, sight: 5, giveUp: 9, leash: 7, herd: [2]int{3, 5}, traits: trNoBody, material: MatChitin, skills: nip(2)})
	add(&caveSpec{kind: CreatureAntSoldier, name: "Kiến lính khổng lồ", hp: 9, walk: 1.6, run: 3.8, radius: 0.3, reach: 1.5,
		yield: ItemMeat, yieldN: 1, drops: []Ingredient{{ItemChitin, 1}}, sight: 7, giveUp: 12, leash: 7, herd: [2]int{1, 2}, material: MatChitin,
		skills: []SkillDef{
			sk("mandible", "Kẹp hàm", SkillLight, 9, 2, 14, ZoneCone, 1.5, 0.8, 6, 40, 0, 1.5, 6).with(func(d *SkillDef) { d.Lock, d.Stagger = 3, true }),
			sk("acid", "Phun axit", SkillHeavy, 16, 3, 16, ZoneCone, 4, 0.3, 4, 140, 2, 4, 4).with(func(d *SkillDef) { d.Slow, d.Lock = 50, -1 }),
		}})
	add(&caveSpec{kind: CreatureVampBat, name: "Dơi quỷ hút máu", hp: 45, walk: 2.0, run: 5.5, radius: 0.5, reach: 2.0,
		yield: ItemGuano, yieldN: 3, drops: []Ingredient{{ItemBone, 2}, {ItemFur, 2}}, sight: 9, giveUp: 14, leash: 8, tough: true, material: MatFur, skills: vampSkills})
	add(&caveSpec{kind: CreatureBlindCroc, name: "Cá sấu mù khổng lồ", hp: 90, walk: 0.8, run: 2.8, radius: 0.8, reach: 2.6,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemDinoHide, 4}, {ItemBone, 3}}, sight: 6, giveUp: 12, leash: 5, tough: true,
		amphibious: true, traits: trBoss, material: MatScale, skills: crocBase})
	add(&caveSpec{kind: CreatureAlphaBear, name: "Gấu đầu đàn", hp: 90, walk: 1.0, run: 3.6, radius: 0.8, reach: 2.6,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemFur, 6}, {ItemBone, 4}}, sight: 7, giveUp: 14, leash: 6, tough: true,
		traits: trBoss, material: MatFur, skills: alphaBear})
	add(&caveSpec{kind: CreatureMegalania, name: "Kỳ đà lửa Megalania", hp: 100, walk: 1.1, run: 3.5, radius: 0.8, reach: 2.6,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemDinoHide, 5}, {ItemBone, 3}}, sight: 7, giveUp: 14, leash: 6, tough: true,
		traits: trBoss, material: MatScale, skills: append(scaled(monitorBase, 1.6), ram(24, 6))})
	add(&caveSpec{kind: CreatureLionAlpha, name: "Sư tử hang đầu đàn", hp: 85, walk: 1.4, run: 4.6, radius: 0.7, reach: 2.4,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemFur, 6}, {ItemBone, 3}}, sight: 8, giveUp: 15, leash: 6, tough: true,
		traits: trBoss, material: MatFur, skills: append(scaled(lionBase, 1.5), roarStun(4.5, 16))})
	add(&caveSpec{kind: CreatureScorpQueen, name: "Bọ cạp chúa", hp: 75, walk: 1.2, run: 3.6, radius: 0.7, reach: 2.4,
		yield: ItemMeat, yieldN: 6, drops: []Ingredient{{ItemVenomGland, 2}, {ItemChitin, 5}}, sight: 7, giveUp: 14, leash: 6, tough: true,
		traits: trBoss, material: MatChitin, skills: scorpQueen})
	bonesDrop := []Ingredient{{ItemFossilBone, 2}, {ItemBone, 2}}
	add(&caveSpec{kind: CreatureSkelRaptor, name: "Raptor xương", hp: 12, walk: 1.8, run: 5.2, radius: 0.35, reach: 1.6,
		yield: ItemBone, yieldN: 2, drops: []Ingredient{{ItemFossilBone, 1}}, sight: 7, giveUp: 13, leash: 9, herd: [2]int{2, 3},
		traits: trSkeleton | trPack | trNoBody, material: MatBone, skills: skelRaptor})
	add(&caveSpec{kind: CreatureSkelTrike, name: "Tam giác xương", hp: 40, walk: 0.9, run: 3.2, radius: 0.7, reach: 2.2,
		yield: ItemBone, yieldN: 4, drops: bonesDrop, sight: 7, giveUp: 13, leash: 9, tough: true,
		traits: trSkeleton | trNoBody, material: MatBone, skills: trikeSkel})
	add(&caveSpec{kind: CreatureSkelRex, name: "Rex xương", hp: 100, walk: 1.0, run: 3.6, radius: 0.85, reach: 3.0,
		yield: ItemBone, yieldN: 6, drops: []Ingredient{{ItemFossilBone, 4}, {ItemAmber, 1}}, sight: 8, giveUp: 15, leash: 6, tough: true,
		traits: trSkeleton | trBoss | trNoBody, material: MatBone, skills: rexSkills})

	// The cave bear's new moves, used since StyleCaveFight (see stepBear).
	for i := range bear {
		bear[i].Kind = CreatureBear
	}
	registerCaveSkills(CreatureBear, bear)

	for k, s := range caveSpecs {
		if s == nil {
			continue
		}
		creatureDefs[k] = creatureDef{
			name: s.name, hp: s.hp,
			walk: s.walk / TicksPerSecond, run: s.run / TicksPerSecond,
			radius: s.radius, reach: s.reach,
			yield: s.yield, yieldN: s.yieldN, drops: s.drops,
			leash: max(s.leash, 1), idle: [2]int{30, 120}, roam: [2]int{20, 60},
			fights: !s.has(trHarmless), tough: s.tough, amphibious: s.amphibious,
			herd: s.herd[1], herdMin: s.herd[0],
			since: StyleCaveFight, openOnly: true, away: 18, respawn: caveRespawnTicks,
		}
		for _, d := range s.skills {
			d.Kind = CreatureKind(k)
			registerCaveSkills(CreatureKind(k), []SkillDef{d})
		}
	}
	for n, k := range map[string]CreatureKind{
		"arthropleura": CreatureArthro, "millipede": CreatureArthro, "arthroqueen": CreatureArthroQueen,
		"centipede": CreatureCentipede, "scutigera": CreatureCentipede,
		"spiderqueen": CreatureQueenMeadow, "queenmeadow": CreatureQueenMeadow, "queendesert": CreatureQueenDesert,
		"queenmarsh": CreatureQueenMarsh, "queensnow": CreatureQueenSnow, "queenvolcano": CreatureQueenVolcano,
		"spiderling": CreatureSpiderling, "eggsac": CreatureEggSac, "bat": CreatureBat, "olm": CreatureOlm,
		"cavecrab": CreatureCaveCrab, "hyena": CreatureHyena, "whipscorpion": CreatureWhipScorp,
		"cavelion": CreatureCaveLion, "monitor": CreatureMonitor, "ant": CreatureAnt, "antsoldier": CreatureAntSoldier,
		"vampbat": CreatureVampBat, "blindcroc": CreatureBlindCroc, "alphabear": CreatureAlphaBear,
		"megalania": CreatureMegalania, "lionalpha": CreatureLionAlpha, "scorpqueen": CreatureScorpQueen,
		"skelraptor": CreatureSkelRaptor, "skeltrike": CreatureSkelTrike, "skelrex": CreatureSkelRex,
	} {
		devBeastAliases[n] = k
	}
}

// registerCaveSkills adds moves to a kind's list (bears get theirs this
// way, the new beasts theirs).
func registerCaveSkills(k CreatureKind, defs []SkillDef) {
	for _, d := range defs {
		d.Kind = k
		addSkills(d)
	}
}

// caveFights reports whether w has the fighting of the caves.
func (w *World) caveFights() bool { return w.style >= StyleCaveFight }

// --- the steps ------------------------------------------------------------------

// stepCaveKind steps c if it is one of the cave beasts, and reports whether
// it was.
func (w *World) stepCaveKind(c *Creature) bool {
	sp := caveSpecOf(c.Kind)
	if sp == nil {
		return false
	}
	w.stepCaveBeast(c, sp)
	return true
}

func (w *World) stepCaveBeast(c *Creature, sp *caveSpec) {
	d := c.Kind.def()
	if c.cool > 0 {
		c.cool--
	}
	switch {
	case sp.has(trStill):
		w.stepEggSac(c)
		return
	case sp.has(trSkeleton) && w.stepSkeletonRest(c):
		return
	case sp.has(trSwarm) && w.stepBat(c, sp):
		return
	}
	p, dist := w.foeOf(c)
	hadFoe := p != nil
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	if p == nil || p.Asleep {
		w.caveIdle(c, sp)
		return
	}
	if sp.has(trHarmless) {
		if dist < 3 {
			w.fleeFrom(c, p)
		} else {
			w.wander(c)
		}
		return
	}
	if !hadFoe {
		if dist > sp.sight {
			w.caveIdle(c, sp)
			return
		}
		c.foe, c.rage = p.ID, 10*TicksPerSecond // it has noticed them
	}
	if dist > sp.giveUp {
		c.rage = 0
		w.caveIdle(c, sp)
		return
	}
	run := d.run
	if sp.has(trLegShed) {
		run *= 1 - 0.13*float64(c.cv.legs)
	}
	if sp.has(trFearFire) && p.torchFuel > 0 && dist < 5.5 && !(c.cv.legs >= LegsMax) {
		if dist < 4.5 {
			w.fleeFrom(c, p)
		} else {
			w.circleFoe(c, p, 5.2, run*0.7, 0.3, func(dx, dy float64) { w.moveCreature(c, dx, dy) })
		}
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if sp.has(trPack) && w.packFight(c, p, dist, func(dx, dy float64) { w.moveCreature(c, dx, dy) }) {
		return
	}
	if !sp.has(trPack) && w.trySkill(c, p, dist) {
		return
	}
	if !w.runAt(c, p.X, p.Y, run, d.reach*0.55) {
		c.faceToward(p.X)
	}
}

// caveIdle is a cave beast with nobody to fight: an ambusher waits where it
// is, a boss mends and stays by its hoard, the rest wander.
func (w *World) caveIdle(c *Creature, sp *caveSpec) {
	switch {
	case sp.has(trAmbush):
		c.State, c.timer = CreatureIdle, 0
	case sp.has(trBoss):
		if c.HP < c.Kind.MaxHP() && w.tick%caveBossMendEvery == 0 {
			c.HP = min(c.Kind.MaxHP(), c.HP+HealthScale/2)
		}
		if d := math.Hypot(c.homeX-c.X, c.homeY-c.Y); d > sp.leash {
			w.runAt(c, c.homeX, c.homeY, c.Kind.def().walk*1.6, 0.5)
			c.State = CreatureWander
			return
		}
		w.wander(c)
	default:
		w.wander(c)
	}
}

// stepEggSac has a queen's egg sac hatch when someone has stood near it, or
// the queen is fighting, long enough. A sac that is broken first gives only
// silk.
func (w *World) stepEggSac(c *Creature) {
	c.State = CreatureIdle
	near := false
	if _, d := w.nearestPlayer(c.X, c.Y); d <= eggHatchNear {
		near = true
	}
	for _, o := range w.fauna.creatures {
		if o.Kind.IsSpiderQueen() && !o.gone && o.rage > 0 && math.Hypot(o.X-c.X, o.Y-c.Y) < 14 {
			near = true
			break
		}
	}
	switch {
	case near:
		c.cv.hatch++
	case c.cv.hatch > 0:
		c.cv.hatch--
	}
	if c.cv.hatch < eggHatchTicks {
		return
	}
	c.gone = true
	p, _ := w.nearestPlayer(c.X, c.Y)
	w.summon(c, CreatureSpiderling, 2+int(c.ID%2), p)
}

// summon sets n creatures of kind k round c, angry at p if there is one.
func (w *World) summon(c *Creature, k CreatureKind, n int, p *Player) {
	r := w.random()
	for i := 0; i < n; i++ {
		for range 10 {
			a, dd := r.Float64()*2*math.Pi, 0.8+r.Float64()*1.6
			x, y := c.X+math.Cos(a)*dd, c.Y+math.Sin(a)*dd
			if !w.creatureFits(k, x, y) {
				continue
			}
			o := w.addCreature(k, x, y)
			if p != nil {
				o.foe, o.rage, o.State = p.ID, HerdRageTicks, CreatureChase
			}
			break
		}
	}
}

// stepSkeletonRest is a skeleton lying as a pile of bones (asleep) or
// standing up (roaring). It reports whether that was its whole step.
func (w *World) stepSkeletonRest(c *Creature) bool {
	switch c.State {
	case CreatureSleep:
		p, d := w.nearestPlayer(c.X, c.Y)
		if c.HP < c.Kind.MaxHP() || p != nil && !p.Asleep && d <= SkeletonRiseDist {
			c.State, c.timer = CreatureRoar, skeletonRiseTime
			if p != nil {
				c.faceToward(p.X)
			}
			// the rest of its bones in the chamber stir too
			for _, o := range w.fauna.creatures {
				if o != c && o.Kind.IsSkeleton() && o.State == CreatureSleep && math.Hypot(o.X-c.X, o.Y-c.Y) < 7 && o.sk.id == SkillNone {
					o.State, o.timer = CreatureRoar, skeletonRiseTime+int(o.ID%5)*3
				}
			}
		}
		return true
	case CreatureRoar:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
			if p, _ := w.nearestPlayer(c.X, c.Y); p != nil {
				c.foe, c.rage = p.ID, HerdRageTicks
			}
		}
		return true
	}
	return false
}

// stepBat is a bat: it hangs in its roost (asleep) till a torch or a
// footstep comes close, then the whole roost bursts into the air, leaving
// guano, and for a while nips at whoever woke it.
func (w *World) stepBat(c *Creature, sp *caveSpec) bool {
	p, dist := w.nearestPlayer(c.X, c.Y)
	if c.State == CreatureSleep || c.State == CreatureIdle && c.cv.swarm == 0 {
		if p != nil && !p.Asleep && (dist < 1.8 || p.torchFuel > 0 && dist < 3.8) {
			w.burstBats(c, p)
		} else if c.State != CreatureSleep {
			c.State = CreatureSleep
		}
		return true
	}
	if c.cv.swarm > 0 {
		c.cv.swarm--
		r := w.random()
		if c.hx == 0 && c.hy == 0 || r.IntN(18) == 0 {
			a := r.Float64() * 2 * math.Pi
			c.hx, c.hy = math.Cos(a), math.Sin(a)
		}
		run := c.Kind.def().run * 0.8
		if c.rage > 0 && p != nil && !p.Asleep && c.cv.swarm < 36 {
			// after the burst, it comes at its foe, wheeling
			if w.trySkill(c, p, dist) {
				return true
			}
			a := math.Atan2(p.Y-c.Y, p.X-c.X) + math.Sin(float64(w.tick)/5+float64(c.ID))*0.9
			c.hx, c.hy = math.Cos(a), math.Sin(a)
			if dist < 1.2 {
				c.hx, c.hy = -c.hx, -c.hy
			}
		}
		c.State = CreatureChase
		if !w.moveCreature(c, c.hx*run, c.hy*run) {
			c.hx, c.hy = 0, 0
		}
		if c.cv.swarm == 0 {
			c.State, c.rage = CreatureSleep, 0
		}
		return true
	}
	return false
}

// burstBats sends every bat near c into the air, leaving guano.
func (w *World) burstBats(c *Creature, p *Player) {
	for _, o := range w.fauna.creatures {
		if o.Kind != CreatureBat || o.gone || o.State != CreatureSleep && o != c || math.Hypot(o.X-c.X, o.Y-c.Y) > 9 {
			continue
		}
		o.cv.swarm = 5 * TicksPerSecond
		o.State, o.hx, o.hy = CreatureChase, 0, 0
		o.foe, o.rage = p.ID, 5*TicksPerSecond
		if o.ID%3 == 0 {
			w.dropAt(o.X, o.Y, []Ingredient{{ItemGuano, 1}})
		}
	}
}

// --- what a blow does -----------------------------------------------------------

// isBlunt reports whether held strikes with weight: a pickaxe, a battle axe.
func isBlunt(held ItemKind) bool {
	if held == ItemBattleAxe {
		return true
	}
	class, _ := held.ToolKind()
	return class == ToolPickaxe
}

// caveBlow is what is left of a blow of dmg by p at c once the cave beast's
// make has had its say: a skeleton shatters under heavy blows and a torch
// and shrugs off a spear; a beast caught in the recovery of a big move is
// open to everything.
func (w *World) caveBlow(p *Player, c *Creature, dmg int) int {
	sp := caveSpecOf(c.Kind)
	if sp == nil {
		return dmg
	}
	k := 1.0
	if sp.has(trSkeleton) {
		held := p.Held()
		switch {
		case held == ItemTorch && p.torchFuel > 0:
			k = SkeletonFireK
			dmg = max(dmg, FistDamage)
		case isBlunt(held):
			k = SkeletonBluntK
		case held == ItemSpear || held == ItemIvorySpear:
			k = SkeletonSharpK
		default:
			if _, ok := SwordTier(held); ok {
				k = SkeletonSharpK
			}
		}
	}
	if s := &c.sk; s.id != SkillNone && s.phase == PhaseRecover && Skill(s.id).Fx&FxExpose != 0 {
		k *= ExposedK
	}
	if k == 1 {
		return dmg
	}
	return max(1, int(float64(dmg)*k+0.5))
}

// caveWounded is c just hit for dmg by someone: a centipede loses a leg to a
// hard blow, a skeleton's pile wakes, and the nests and packs take notice.
func (w *World) caveWounded(c *Creature, dmg int, by *Player) {
	sp := caveSpecOf(c.Kind)
	if sp == nil {
		return
	}
	if sp.has(trLegShed) && dmg >= LegShedHits && c.cv.legs < LegsMax {
		c.cv.legs++
	}
	if by != nil && !sp.has(trHarmless) && !sp.has(trStill) && c.State != CreatureSleep && c.rage < HerdRageTicks/2 {
		c.foe, c.rage = by.ID, HerdRageTicks
	}
	if by != nil && w.Underground(int(by.Y)) {
		w.caveNoise(by.X, by.Y, 14)
	}
}

// caveFelled is c just killed: a boss's hoard opens to the world for good, a
// skeleton falls apart in bones.
func (w *World) caveFelled(c *Creature) {
	if c.cv.guard != 0 {
		if e := w.entities[c.cv.guard]; e != nil {
			e.Fuel = 1 // the guardian has fallen: a load must not raise it again
		}
	}
	if sp := caveSpecOf(c.Kind); sp != nil && sp.has(trSkeleton) {
		// ancient amber, now and then, among the bones
		if w.random().IntN(6) == 0 {
			w.dropAt(c.X, c.Y, []Ingredient{{ItemAmber, 1}})
		}
	}
}

// hoardGuardian is the cave boss standing guard over old chest e, if one is
// still alive.
func (w *World) hoardGuardian(e *Entity) *Creature {
	if !w.caveFights() {
		return nil
	}
	for _, c := range w.fauna.creatures {
		if c.cv.guard == e.ID && !c.gone {
			return c
		}
	}
	return nil
}

// afterLoad takes away the cave bosses whose hoard was opened before the
// save: the fallen stay fallen.
func (w *World) afterLoadCaves() {
	for _, c := range w.fauna.creatures {
		if c.cv.guard == 0 {
			continue
		}
		if e := w.entities[c.cv.guard]; e == nil || e.Fuel == 1 {
			c.gone = true
		}
	}
}

// caveHit is what move d of c leaves on p besides damage.
func (w *World) caveHit(c *Creature, d *SkillDef, p *Player) {
	if d.Fx&FxVenom != 0 {
		p.poison = max(p.poison, PoisonLong)
	}
	if d.Fx&FxEnsnare != 0 {
		p.webbed = WebTicks
		p.notify(NoticeWebbed, 0)
	}
	if d.Fx&FxDrain != 0 && d.Damage > 0 {
		c.HP = min(c.Kind.MaxHP(), c.HP+int(d.Damage*0.8))
	}
}

// caveCry is what move d of c does where it lands: clouds, floor webs,
// helpers called, bones gathered.
func (w *World) caveCry(c *Creature, d *SkillDef) {
	s := &c.sk
	if d.Damage >= 14 {
		w.caveNoise(c.X, c.Y, 8) // a big blow shakes the roof
	}
	switch {
	case d.Fx&FxCloudPoison != 0:
		w.addHazard(HazPoison, s.x, s.y, d.Reach, 6*TicksPerSecond)
	case d.Fx&FxCloudDust != 0:
		w.addHazard(HazDust, s.x, s.y, d.Reach, 5*TicksPerSecond)
	}
	if d.Fx&FxWebFloor != 0 {
		w.addHazard(HazWeb, s.x, s.y, d.Reach, 30*TicksPerSecond)
	}
	if d.Fx&FxSummon != 0 {
		p := w.players[s.foe]
		switch {
		case c.Kind.IsSpiderQueen():
			w.summon(c, CreatureSpiderling, 3, p)
		case c.Kind == CreatureScorpQueen:
			w.summon(c, CreatureScorpion, 2, p)
		}
	}
	if d.Name == "reassemble" {
		c.HP = min(c.Kind.MaxHP(), c.HP+c.Kind.MaxHP()/5)
	}
}
