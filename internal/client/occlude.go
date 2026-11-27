package client

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Seeing through what stands in front. Trees, walls and huts are sized by
// the player now, several times their height, so a crown or a roof drawn in
// front of someone would hide them. Whatever tall thing covers a player or a
// beast standing behind it fades until they step out: most for the player
// themself, less for the others.
//
// What a sprite covers is read from a coarse mask of its opaque pixels, kept
// when the sprite is drawn (noteCover), so a figure beside a trunk under the
// open air is not taken for hidden.

const (
	coverCell   = 12             // image pixels to a cell of a mask
	coverMinH   = 30 * HDDensity // sprites shorter than a player hide nobody: no mask
	fadeMine    = 0.42           // how solid what hides the player themself stays
	fadeOthers  = 0.55           // and what hides another player or a beast
	fadeRate    = 0.16           // of the way there each frame
	figSamplesX = 3
	figSamplesY = 6
)

// coverMask is which cells of a sprite's image are mostly opaque.
type coverMask struct {
	w, h  int // in cells
	flipW int // the image is the mirror of the one the mask was read from, this wide
	on    []bool
}

var coverMasks sync.Map // *ebiten.Image -> *coverMask

// noteCover keeps the mask of img, the image of canvas c.
func noteCover(img *ebiten.Image, c *canvas) {
	if c.h < coverMinH {
		return
	}
	m := &coverMask{w: (c.w + coverCell - 1) / coverCell, h: (c.h + coverCell - 1) / coverCell}
	m.on = make([]bool, m.w*m.h)
	for cy := range m.h {
		for cx := range m.w {
			n, all := 0, 0
			for y := cy * coverCell; y < min(c.h, (cy+1)*coverCell); y += 3 {
				for x := cx * coverCell; x < min(c.w, (cx+1)*coverCell); x += 3 {
					all++
					if c.px[y*c.w+x].A >= 128 {
						n++
					}
				}
			}
			m.on[cy*m.w+cx] = n*2 > all
		}
	}
	coverMasks.Store(img, m)
}

// mirrorCover gives img, the mirror image of src (w image pixels wide),
// the mirror of src's mask.
func mirrorCover(src, img *ebiten.Image, w int) {
	if v, ok := coverMasks.Load(src); ok {
		m := *v.(*coverMask)
		m.flipW = w
		if v.(*coverMask).flipW > 0 {
			m.flipW = 0 // mirrored back
		}
		coverMasks.Store(img, &m)
	}
}

// at reports whether the mask is opaque at image pixel (x, y).
func (m *coverMask) at(x, y float64) bool {
	if m.flipW > 0 {
		x = float64(m.flipW) - x
	}
	if x < 0 || y < 0 {
		return false
	}
	cx, cy := int(x)/coverCell, int(y)/coverCell
	return cx < m.w && cy < m.h && m.on[cy*m.w+cx]
}

// figure is someone who may be hidden: their ground point on the screen (art
// pixels), their depth, and the box of their body.
type figure struct {
	sx, sy, y float64
	hw, h     float64
	mine      bool
	beast     bool // a beast, which a beast in front of it does not fade for
}

// noteFigures lists the players and the beasts of the draw list, for
// seeThrough.
func (g *Game) noteFigures(list []drawable) {
	g.figures = g.figures[:0]
	hiddenSelf.cover = 0 // see hiddenself.go
	for _, d := range list {
		switch {
		case d.isPlayer:
			p := g.curr.Players[d.idx]
			if p.Dead {
				continue
			}
			g.figures = append(g.figures, figure{d.sx, d.sy, d.y, 5, playerArtH, p.ID == g.myID, false})
		case d.creature:
			c := g.curr.Creatures[d.idx]
			s, _ := g.creatureSprite(c, d.moving)
			if s.img == nil || s.ay < 14 {
				continue // small game is lost in the grass anyway
			}
			g.figures = append(g.figures, figure{d.sx, d.sy, d.y, s.w() * 0.3, s.ay * 0.9, false, true})
		}
	}
}

// seeThrough is how solid entity e, drawn as s at d, is this frame: 1, or
// less while it hides a figure standing behind it.
func (g *Game) seeThrough(e protocol.EntityView, s sprite, d drawable) float32 {
	cur, fading := g.fades[e.ID]
	cur, keep := fadeTo(g.coverTarget(s, d, false), cur, fading)
	if !keep {
		delete(g.fades, e.ID)
		return cur
	}
	if g.fades == nil {
		g.fades = make(map[sim.EntityID]float32)
	}
	g.fades[e.ID] = cur
	return cur
}

// beastFades is how solid each great beast fading in front of someone is.
var beastFades = map[sim.CreatureID]float32{}

// seeThroughBeast is seeThrough for a beast: a long-neck or a tyrant
// standing between the camera and someone fades as a tree does.
func (g *Game) seeThroughBeast(c protocol.CreatureView, s sprite, d drawable) float32 {
	if c.ID == 0 {
		return 1
	}
	cur, fading := beastFades[c.ID]
	cur, keep := fadeTo(g.coverTarget(s, d, true), cur, fading)
	if !keep {
		delete(beastFades, c.ID)
		return cur
	}
	if len(beastFades) > 256 {
		clear(beastFades)
	}
	beastFades[c.ID] = cur
	return cur
}

// fadeTo eases a fade at cur (fading: under way) towards target; keep is
// false once it is solid again.
func fadeTo(target, cur float32, fading bool) (float32, bool) {
	if !fading {
		if target == 1 {
			return 1, false
		}
		cur = 1
	}
	cur += (target - cur) * fadeRate
	if target == 1 && cur > 0.985 {
		return 1, false
	}
	return cur, true
}

// coverTarget is how solid s, drawn at d, should be: less while it hides a
// figure standing behind it; a beast (beast) fades only for a player.
func (g *Game) coverTarget(s sprite, d drawable, beast bool) float32 {
	target := float32(1)
	if s.img != nil && s.ay >= 30 {
		if v, ok := coverMasks.Load(s.img); ok {
			m := v.(*coverMask)
			k := float64(s.density())
			left, top := d.sx-s.ax, d.sy-s.ay
			for _, f := range g.figures {
				if f.y >= d.y || beast && f.beast || f.sx+f.hw < left || f.sx-f.hw > left+s.w() || f.sy < top || f.sy-f.h > d.sy {
					continue // in front of it, or clear of it
				}
				hit := 0
				for j := range figSamplesY {
					fy := f.sy - f.h*(float64(j)+0.5)/figSamplesY
					for i := range figSamplesX {
						fx := f.sx + f.hw*(float64(i)-1)
						if m.at((fx-left)*k, (fy-top)*k) {
							hit++
							if f.mine {
								noteSelfCover(j*figSamplesX + i) // see hiddenself.go
							}
						}
					}
				}
				if hit*5 < figSamplesX*figSamplesY*2 {
					continue // less than two fifths of them is behind it
				}
				if f.mine {
					target = fadeMine
					break
				}
				target = fadeOthers
			}
		}
	}
	return target
}
