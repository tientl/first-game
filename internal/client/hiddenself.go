package client

import (
	"math"
	"math/bits"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

// The player is never lost under the leaves. Whatever stands in front of
// them fades (see occlude.go), but two or three crowns one behind another
// still hide a figure, faded as each is. So when most of the player's own
// figure is covered, it is drawn again over everything: a soft light
// outline round a ghost of the figure, as in many games of this kind.

// hiddenSelf is what the frame knows of the player's own figure behind
// things: which of its sample points (see seeThrough) something in front
// covers, and the scratch images it is drawn again in.
var hiddenSelf struct {
	cover     uint32
	buf, ring *fineBuf
	shown     bool // the last frame drew the silhouette (for cmd/shot)
}

const (
	hiddenShare   = 0.4  // of the figure's sample points covered before it shows through
	hiddenOutline = 0.85 // how solid the outline is
	hiddenGhost   = 0.5  // and the ghost of the figure inside it
	hiddenBufW    = 96   // the scratch images, in art pixels
	hiddenBufH    = 96
	hiddenBufX    = 48 // where the figure's feet go in them
	hiddenBufY    = 78
)

// SelfSilhouetted reports whether the last frame drew the player's own
// silhouette over what hid them.
func SelfSilhouetted() bool { return hiddenSelf.shown }

// noteSelfCover records that sample point bit of the player's own figure
// is covered by something drawn in front of it.
func noteSelfCover(bit int) { hiddenSelf.cover |= 1 << bit }

// drawHiddenSelf draws the player's own figure again over everything, as a
// silhouette, if enough of it was hidden this frame.
func (g *Game) drawHiddenSelf(dst drawTarget) {
	covered := bits.OnesCount32(hiddenSelf.cover)
	hiddenSelf.cover, hiddenSelf.shown = 0, false
	if float64(covered) < hiddenShare*figSamplesX*figSamplesY {
		return
	}
	for _, d := range g.drawList {
		if !d.isPlayer {
			continue
		}
		p := g.curr.Players[d.idx]
		if p.ID != g.myID || p.Dead || p.Asleep || p.Swimming {
			continue
		}
		if hiddenSelf.buf == nil {
			hiddenSelf.buf = newFineBuf(hiddenBufW, hiddenBufH)
			hiddenSelf.ring = newFineBuf(hiddenBufW, hiddenBufH)
		}
		buf, ring := hiddenSelf.buf, hiddenSelf.ring
		buf.img.Clear()
		ring.img.Clear()
		align(dst, d.sx, d.sy) // on the grid the figure was drawn on
		// The part of the feet's place between art pixels goes into the
		// scratch image, as for a jolted figure (see jolt.go).
		gx, gy := gridOf(dst)
		fx, fy := d.sx-gx-math.Floor(d.sx-gx), d.sy-gy-math.Floor(d.sy-gy)
		at := d
		at.sx, at.sy = hiddenBufX+fx, hiddenBufY+fy
		g.drawStanding(buf, p, at)

		// The outline: the figure's shape in one light colour, stamped a
		// little to each side, with the figure itself cut out of it.
		var cm colorm.ColorM
		cm.Scale(0, 0, 0, 1)
		cm.Translate(1, 0.95, 0.78, 0)
		for _, o := range [8][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-0.7, -0.7}, {0.7, -0.7}, {-0.7, 0.7}, {0.7, 0.7}} {
			op := &colorm.DrawImageOptions{}
			op.GeoM.Translate(o[0]*HDDensity, o[1]*HDDensity)
			colorm.DrawImage(ring.img, buf.img, cm, op)
		}
		cut := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationOut}
		ring.img.DrawImage(buf.img, cut)

		ox, oy := d.sx-fx-hiddenBufX, d.sy-fy-hiddenBufY
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ox, oy)
		op.ColorScale.ScaleAlpha(hiddenOutline)
		drawImageD(dst, ring.img, op, HDDensity)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ox, oy)
		op.ColorScale.ScaleAlpha(hiddenGhost)
		drawImageD(dst, buf.img, op, HDDensity)
		hiddenSelf.shown = true
		return
	}
}
