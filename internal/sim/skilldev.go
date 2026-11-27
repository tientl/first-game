package sim

import (
	"math"
	"strings"
)

// The dev console's commands for trying the beasts' moves (skills.go):
//
//	fight <thú>           one of them, angry, a few tiles off
//	skill <thú> [chiêu]   the nearest one (or a new one) set where the move
//	                      strikes the player, and made to do it now; with no
//	                      move named, its moves are listed

func init() {
	for _, n := range []string{"fight", "danh", "dau", "aggro"} {
		devCommands[n] = "fight"
	}
	for _, n := range []string{"skill", "chieu", "move"} {
		devCommands[n] = "skill"
	}
	devMore["fight"] = func(w *World, r *DevResult, p *Player, args []string) { w.devFight(r, p, args) }
	devMore["skill"] = func(w *World, r *DevResult, p *Player, args []string) { w.devSkill(r, p, args) }
	DevHelp = append(DevHelp,
		"fight <thú>           gọi một con thú đang giận ra gần mình (fight rex)",
		"skill <thú> [chiêu]   bắt thú gần nhất ra chiêu vào mình ngay (skill rex tail); không ghi chiêu: liệt kê")
	devBeastAliases["dakota"] = CreatureRaptor
	devBeastAliases["dakotaraptor"] = CreatureRaptor
	devBeastAliases["troodon"] = CreatureTroodon
	devBeastAliases["anzu"] = CreatureAnzu
}

func (w *World) devFight(r *DevResult, p *Player, args []string) {
	k, ok := w.devBeast(r, args, "fight")
	if !ok {
		return
	}
	c := w.devPlace(p, k, 5)
	if c == nil {
		r.say("Không có chỗ cho %s quanh đây", k.Name())
		return
	}
	enrage(c, p)
	r.say("%s đang giận, cách %.0f ô", k.Name(), math.Hypot(c.X-p.X, c.Y-p.Y))
}

func (w *World) devSkill(r *DevResult, p *Player, args []string) {
	k, ok := w.devBeast(r, args, "skill")
	if !ok {
		return
	}
	if len(SkillsOf(k)) == 0 {
		r.say("%s không có chiêu nào", k.Name())
		return
	}
	if len(args) < 2 {
		var names []string
		for _, id := range SkillsOf(k) {
			names = append(names, Skill(id).Name+" ("+Skill(id).Label+")")
		}
		r.say("Chiêu của %s: %s", k.Name(), strings.Join(names, ", "))
		return
	}
	id, ok := FindSkill(k, strings.Join(args[1:], " "))
	if !ok {
		r.say("%s không có chiêu %q", k.Name(), strings.Join(args[1:], " "))
		return
	}
	d := Skill(id)
	c := w.devPlace(p, k, skillDistance(d))
	if c == nil {
		r.say("Không có chỗ cho %s quanh đây", k.Name())
		return
	}
	enrage(c, p)
	c.FaceLeft = c.X > p.X
	if !w.ForceSkill(c.ID, id, p.ID) {
		r.say("%s không ra được chiêu %s", k.Name(), d.Label)
		return
	}
	r.say("%s: %s (lấy đà %.1f s)", k.Name(), d.Label, float64(c.sk.wind)/TicksPerSecond)
}

// devBeast reads the beast named in args for command cmd.
func (w *World) devBeast(r *DevResult, args []string, cmd string) (CreatureKind, bool) {
	if len(args) == 0 {
		r.say("%s <thú>; list beasts để xem tên thú", cmd)
		return CreatureNone, false
	}
	k, ok := FindCreature(args[0])
	if !ok || k.Boss() {
		r.say("Không biết thú %q%s", args[0], suggest(devCreatureNames(), args[0]))
		return CreatureNone, false
	}
	return k, true
}

// skillDistance is how far from its foe a beast stands to make move d.
func skillDistance(d *SkillDef) float64 {
	switch {
	case d.Motion == MoveLeap:
		return (d.MinD + d.MaxD) / 2
	case d.Motion == MoveDash:
		return min(d.MinD+2, d.MaxD)
	case d.Motion == MoveSwim:
		return 3
	case d.Motion == MoveHover:
		return 0
	case d.Anchor == AtAhead || d.Anchor == AtBehind:
		return d.Offset
	}
	return min(max(d.MinD+0.3, d.Kind.def().radius+1), d.MaxD)
}

// devPlace sets a beast of kind k dist tiles from p, to the side with room
// (east first): the nearest of its kind within 20 tiles, or a new one.
func (w *World) devPlace(p *Player, k CreatureKind, dist float64) *Creature {
	var c *Creature
	best := 20.0
	for _, o := range w.fauna.creatures {
		if d := math.Hypot(o.X-p.X, o.Y-p.Y); o.Kind == k && !o.gone && !o.Tamed() && d < best {
			c, best = o, d
		}
	}
	for _, step := range []float64{0, 0.5, 1, 1.5, 2, 3} {
		for _, a := range []float64{0, math.Pi, math.Pi / 2, -math.Pi / 2, math.Pi / 4, 3 * math.Pi / 4} {
			x, y := p.X+math.Cos(a)*(dist+step), p.Y+math.Sin(a)*(dist+step)
			if !w.creatureFits(k, x, y) {
				continue
			}
			if c == nil {
				c = w.addCreature(k, x, y)
			}
			w.cancelSkill(c)
			c.X, c.Y, c.homeX, c.homeY = x, y, x, y
			c.FaceLeft = x > p.X
			return c
		}
	}
	return nil
}

// enrage turns c on p, as if p had hit it.
func enrage(c *Creature, p *Player) {
	c.foe, c.rage, c.cool = p.ID, HerdRageTicks, 0
	c.hunter, c.huntedT = 0, 0
	switch c.State {
	case CreatureIdle, CreatureWander, CreatureEat, CreatureDrink, CreatureSleep, CreatureRest, CreatureHurt:
		c.State, c.timer = CreatureChase, RexChaseTicks
	}
	if c.Kind == CreaturePtero {
		c.State = CreatureSoar
	}
	if c.Kind == CreaturePlesio && c.State == CreatureChase {
		c.timer = PlesioSurfaceTicks
	}
}
