package sim

import (
	"math"
	"strconv"
	"strings"
)

// The dev console's boss command, for trying a boss's fight at its lair:
//
//	boss wake           the nearest boss wakes and turns on you
//	boss phase <1-3>    its health set to where that stage of its fight begins
//	boss skill <chiêu>  it makes that move at you now (boss skill: list them)

func init() {
	for _, n := range []string{"boss", "trum"} {
		devCommands[n] = "boss"
	}
	devMore["boss"] = func(w *World, r *DevResult, p *Player, args []string) { w.devBoss(r, p, args) }
	DevHelp = append(DevHelp,
		"boss wake | phase <1-3> | skill [chiêu] | calm [on|off]   trùm gần nhất: thức dậy, sang giai đoạn, ra chiêu; ngủ say")
}

// devNearBoss is the boss nearest p, within 60 tiles of its lair, and its lair.
func (w *World) devNearBoss(p *Player) (*Creature, *bossLair) {
	var best *bossLair
	bd := 60.0
	for i := range w.bosses.lairs {
		l := &w.bosses.lairs[i]
		if l.c == nil || l.c.gone {
			continue
		}
		if d := math.Hypot(l.x-p.X, l.y-p.Y); d < bd {
			best, bd = l, d
		}
	}
	if best == nil {
		return nil, nil
	}
	return best.c, best
}

// devWake wakes the boss of l onto p, without its roar.
func (w *World) devWake(c *Creature, l *bossLair, p *Player) {
	l.awake, l.roared, l.foe, l.lost = true, true, p.ID, 0
	if c.State == CreatureIdle || c.State == CreatureRoar || c.State == CreatureWander {
		c.State, c.timer = CreatureChase, 0
		if c.Kind == CreatureMegalodon {
			c.State = CreatureLurk
		}
	}
}

func (w *World) devBoss(r *DevResult, p *Player, args []string) {
	c, l := w.devNearBoss(p)
	if c == nil && (len(args) == 0 || fold(args[0]) != "calm") {
		r.say("Không có trùm nào gần đây (tp trùm 1 hoặc 2)")
		return
	}
	if len(args) == 0 {
		r.say("boss wake | phase <số> | skill [chiêu] — %s, máu %d/%d, giai đoạn %d", c.Kind.Name(), c.HP, c.Kind.MaxHP(), c.SkillStage()+1)
		return
	}
	switch fold(args[0]) {
	case "calm", "ngu", "sleep":
		w.dev.calm = !w.dev.calm
		if len(args) > 1 {
			w.dev.calm = fold(args[1]) == "on" || fold(args[1]) == "bat" || args[1] == "1"
		}
		if w.dev.calm {
			for i := range w.bosses.lairs {
				if l := &w.bosses.lairs[i]; l.c != nil {
					l.awake, l.foe, l.roared = false, 0, false
				}
			}
			r.say("Trùm ngủ say, không ai đánh thức được")
		} else {
			r.say("Trùm thức như thường")
		}
		return
	case "wake", "day", "thuc":
		w.devWake(c, l, p)
		r.say("%s thức dậy", c.Kind.Name())
	case "phase", "stage", "giaidoan", "gd":
		n := 0
		if len(args) > 1 {
			n, _ = strconv.Atoi(args[1])
		}
		at := skillStages[c.Kind]
		if n < 1 || n > len(at)+1 {
			r.say("phase <1-%d>", len(at)+1)
			return
		}
		hp := c.Kind.MaxHP()
		if n > 1 {
			hp = int(at[n-2] * float64(c.Kind.MaxHP()))
		}
		c.HP, l.hp = hp, hp
		l.enraged = c.HP*2 < c.Kind.MaxHP()
		w.devWake(c, l, p)
		r.say("%s: giai đoạn %d, máu %d/%d", c.Kind.Name(), c.SkillStage()+1, c.HP, c.Kind.MaxHP())
	case "skill", "chieu", "move":
		if len(args) < 2 {
			var names []string
			for _, id := range SkillsOf(c.Kind) {
				names = append(names, Skill(id).Name+" ("+Skill(id).Label+")")
			}
			r.say("Chiêu của %s: %s", c.Kind.Name(), strings.Join(names, ", "))
			return
		}
		id, ok := FindSkill(c.Kind, strings.Join(args[1:], " "))
		if !ok {
			r.say("%s không có chiêu %q", c.Kind.Name(), strings.Join(args[1:], " "))
			return
		}
		w.devWake(c, l, p)
		if !w.ForceSkill(c.ID, id, p.ID) {
			r.say("%s không ra được chiêu %s", c.Kind.Name(), Skill(id).Label)
			return
		}
		r.say("%s: %s (lấy đà %.1f s)", c.Kind.Name(), Skill(id).Label, float64(c.sk.wind)/TicksPerSecond)
	default:
		r.say("boss wake | phase <số> | skill [chiêu]")
	}
}
