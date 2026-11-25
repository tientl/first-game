package client

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Fighting, as the player sees it: a health bar over a beast that has been
// hurt, and the damage of each blow rising from it, bigger and orange for a
// critical one. See sim/combat.go.

// hitPop is the damage of one blow, floating up from where it landed.
type hitPop struct {
	x, y float64 // world tiles
	n    int
	crit bool
	at   time.Time
}

const (
	popLife  = 900 * time.Millisecond
	barShown = 6 * time.Second // a hurt beast's bar stays this long after a blow
)

var (
	pops    []hitPop
	critAt  time.Time                        // when my last blow was a critical one
	hurtAt  = map[sim.CreatureID]time.Time{} // when each beast was last hurt
	barBack = color.RGBA{0x18, 0x10, 0x10, 0xd0}
)

// noteHits finds the beasts that lost health since the last snapshot, and
// floats up what each lost. A beast that is gone lost what it had left.
func (g *Game) noteHits(s protocol.Snapshot) {
	now := time.Now()
	if s.Self.Notice == sim.NoticeCrit && s.Self.NoticeSeq != g.noticeSeq {
		critAt = now
	}
	seen := make(map[sim.CreatureID]bool, len(s.Creatures))
	for _, c := range s.Creatures {
		seen[c.ID] = true
		q, ok := g.prevCreatures[c.ID]
		if !ok || c.HP >= q.HP || !shown(c) {
			continue
		}
		g.pop(float64(c.X), float64(c.Y), c.Kind, int(q.HP-c.HP), now)
		hurtAt[c.ID] = now
	}
	for id, q := range g.prevCreatures {
		if !seen[id] && q.HP < q.MaxHP && now.Sub(hurtAt[id]) < 2*time.Second {
			g.pop(float64(q.X), float64(q.Y), q.Kind, int(q.HP), now) // the killing blow
		}
	}
	for id, t := range hurtAt {
		if now.Sub(t) > barShown {
			delete(hurtAt, id)
		}
	}
}

func (g *Game) pop(x, y float64, k sim.CreatureKind, n int, now time.Time) {
	if n <= 0 {
		return
	}
	crit := now.Sub(critAt) < 400*time.Millisecond
	if crit {
		critAt = time.Time{} // one blow
	}
	pops = append(pops, hitPop{x: x, y: y - beastHeight(k), n: n, crit: crit, at: now})
}

// shown reports whether c is where a blow could land and a bar be seen: not
// down a hole, under the water or up in the air.
func shown(c protocol.CreatureView) bool {
	return !c.State.OutOfReach() && c.State != sim.CreatureLurk && c.Kind != sim.CreatureFish && c.Kind != sim.CreatureWorm
}

// beastHeight is roughly how tall a beast stands, in tiles.
func beastHeight(k sim.CreatureKind) float64 {
	return math.Max(0.8, creatureShadowBase(k)*beastScale(k)/18)
}

// drawCombat draws the bars over hurt beasts and the floating damage.
func (g *Game) drawCombat(screen *ebiten.Image, cam camera, scale, alpha float64) {
	now := time.Now()
	for _, c := range g.curr.Creatures {
		if c.MaxHP <= 0 || c.HP >= c.MaxHP {
			continue
		}
		if t, ok := hurtAt[c.ID]; !ok && !c.Tamed || ok && now.Sub(t) > barShown && !c.Tamed {
			continue
		}
		if !shown(c) {
			continue
		}
		x, y, moving := g.lerpCreature(c, alpha)
		if g.hiddenInCave(x, y) {
			continue
		}
		s, bob := g.creatureSprite(c, moving)
		sx, sy := cam.toScreenFine(x, y)
		w := math.Max(22, math.Min(48, s.w()*0.6)) * scale
		h := 2.5 * scale
		bx, by := sx*scale-w/2, (sy-bob-s.ay-5)*scale
		vector.DrawFilledRect(screen, float32(bx-1), float32(by-1), float32(w+2), float32(h+2), barBack, false)
		f := float64(c.HP) / float64(c.MaxHP)
		col := color.RGBA{0x5c, 0xd0, 0x4a, 0xff}
		switch {
		case f < 0.3:
			col = color.RGBA{0xe0, 0x40, 0x34, 0xff}
		case f < 0.6:
			col = color.RGBA{0xe8, 0xb8, 0x30, 0xff}
		}
		vector.DrawFilledRect(screen, float32(bx), float32(by), float32(w*f), float32(h), col, false)
	}
	kept := pops[:0]
	for _, p := range pops {
		age := now.Sub(p.at)
		if age > popLife {
			continue
		}
		kept = append(kept, p)
		u := float64(age) / float64(popLife)
		sx, sy := cam.toScreen(p.x, p.y)
		size, col := 2.0, color.RGBA{0xff, 0xf2, 0xd0, 0xff}
		label := fmt.Sprint(p.n)
		if p.crit {
			size, col, label = 3, color.RGBA{0xff, 0x9a, 0x2a, 0xff}, label+"!"
		}
		col.A = uint8(255 * math.Min(1, (1-u)*2.5))
		g.text(screen, label, sx*scale, (sy-u*14)*scale-10*size, col, text.AlignCenter, size)
	}
	pops = kept
}

// healthText is a beast's health, for its label.
// healthText is a beast's health, and what a blow with held does, for its
// label.
func healthText(c protocol.CreatureView, held sim.ItemKind) string {
	if c.MaxHP <= 0 || c.Kind == sim.CreatureFish || c.Kind == sim.CreatureWorm {
		return ""
	}
	s := fmt.Sprintf(" · máu %d/%d · đòn ~%d", max(0, c.HP), c.MaxHP, sim.BlowDamage(held, c.Kind))
	if sim.Punches(held) && sim.FistShare(c.Kind) < 1 {
		s += " (da dày: đấm tay gần như vô ích)"
	}
	return s
}
