package main

import (
	"strings"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The cave fighting's beasts for -cavebeasts: a row of each named kind (or
// all of them) beside the player, each in the pose -cavepose says, with the
// red zone of the move it is making and, with -cavehaz, the clouds, webs and
// a collapse about to come down.

var caveBeastNames = map[string]sim.CreatureKind{}

func init() {
	for k := sim.CreatureArthro; k <= sim.CreatureSkelRex; k++ {
		_ = k
	}
}

// addCaveBeasts puts the beasts of list round (sx, sy).
func addCaveBeasts(snap *protocol.Snapshot, sx, sy float64, list, pose string, haz bool) {
	var kinds []sim.CreatureKind
	if list == "all" {
		for k := sim.CreatureArthro; k <= sim.CreatureSkelRex; k++ {
			kinds = append(kinds, k)
		}
	} else {
		for _, n := range strings.Split(list, ",") {
			if k, ok := sim.FindCreature(strings.TrimSpace(n)); ok && k.IsCaveBeast() {
				kinds = append(kinds, k)
			}
		}
	}
	id := sim.CreatureID(1 << 27)
	for i, k := range kinds {
		id++
		col, row := i%6, i/6
		v := protocol.CreatureView{ID: id, Kind: k, X: float32(sx - 7 + float64(col)*2.9), Y: float32(sy + 2.5 + float64(row)*3.1),
			State: sim.CreatureIdle, HP: int16(k.MaxHP()), MaxHP: int16(k.MaxHP())}
		if row%2 == 1 {
			v.FaceLeft = true
		}
		skills := sim.SkillsOf(k)
		pick := func(special bool) sim.SkillID {
			var first sim.SkillID
			for _, s := range skills {
				if first == 0 {
					first = s
				}
				if sim.Skill(s).Class == sim.SkillSpecial == special && sim.Skill(s).Windup > 0 {
					return s
				}
			}
			return first
		}
		switch pose {
		case "sleep":
			v.State = sim.CreatureSleep
		case "chase":
			v.State = sim.CreatureChase
		case "windup", "special", "active", "recover":
			if len(skills) > 0 {
				s := pick(pose == "special" || k.IsCaveBoss())
				d := sim.Skill(s)
				v.State = sim.CreatureStrike
				v.Skill = s
				v.Phase = map[string]sim.SkillPhase{"windup": sim.PhaseWindup, "special": sim.PhaseWindup, "active": sim.PhaseActive, "recover": sim.PhaseRecover}[pose]
				v.PhaseLen = uint8(d.PhaseLen(v.Phase))
				v.PhaseT = v.PhaseLen / 2
				if v.Phase == sim.PhaseWindup && d.Zone && d.Shape != sim.ZoneNone {
					snap.Zones = append(snap.Zones, protocol.ZoneView{Creature: id, Kind: k, Skill: s, Shape: d.Shape,
						X: v.X, Y: v.Y, Angle: 0, Reach: float32(d.Reach), Width: float32(d.Width),
						Left: v.PhaseLen - v.PhaseT, Total: v.PhaseLen})
				}
			}
		}
		if k.IsCaveBoss() {
			v.HP = v.HP * 2 / 3
		}
		snap.Creatures = append(snap.Creatures, v)
	}
	if haz {
		x, y := float32(sx), float32(sy+4.5)
		snap.Hazards = append(snap.Hazards,
			protocol.HazardView{Kind: sim.HazPoison, X: x - 5, Y: y, R: 2.4, Left: 100, Total: 120},
			protocol.HazardView{Kind: sim.HazWeb, X: x, Y: y, R: 1.8, Left: 600, Total: 600},
			protocol.HazardView{Kind: sim.HazDust, X: x + 4.5, Y: y, R: 2.2, Left: 100, Total: 100},
			protocol.HazardView{Kind: sim.HazCollapse, X: x + 9, Y: y, R: 2, Left: 10, Total: 24})
	}
}
