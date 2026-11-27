package client

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// HD item icons: drawn in code at the screen's own resolution, so at 1080p
// one pixel of an icon is one pixel of the screen, as in the world, where
// the old 16x16 icons became blocks of 4 to 6 screen pixels across.
//
// An HD icon is a painter (iconpaint.go) that draws in a 16 x 16 design
// box, the box the old icon filled, so it takes the old icon's place and
// size exactly. It is baked the first time it is shown at a size, one image
// per item and size in screen pixels: 96 for a bag slot at 1080p, 64 for a
// chest's, 32 for a recipe's ingredients, half that at 720p, twice at 4K.
// Below 40 pixels it is painted bigger and shrunk, so small icons stay clean;
// at 16 or less the old icon is drawn (it was made for 16). Every HD icon is
// baked ahead on another goroutine at the sizes of the screen's UI scale.
//
// The interface is drawn at the screen's own resolution (uiscale.go), so an
// HD icon goes straight on the screen in its place, in the order it is
// drawn: what covers it (a stack's count, a key badge) is simply drawn
// after it.
//
// Items without a painter keep their old icon, scaled up as before: HD
// icons can be added one item at a time. To add one, see HD_ART.md ("Item
// icons") and iconhd_pilot.go.

// iconHD turns the HD icons on; off, every item shows its old icon.
var iconHD = true

// SetIconHD turns the HD item icons on or off (shot -iconhd=false).
func SetIconHD(on bool) { iconHD = on }

// iconPainter paints one item's icon on p, in the 16 x 16 design box.
type iconPainter func(p *icv)

// hdIcons holds the painter of every item that has an HD icon.
var hdIcons [sim.NumItems]iconPainter

// regIcon gives item k an HD icon painted by f. Call it from an init
// function in the file that holds the painter.
func regIcon(k sim.ItemKind, f iconPainter) {
	if k <= sim.ItemNone || k >= sim.NumItems {
		panic("regIcon: no such item")
	}
	if hdIcons[k] != nil {
		panic("regIcon: " + k.Def().Name + " has an HD icon already")
	}
	hdIcons[k] = f
}

// hasHDIcon reports whether k is drawn with an HD icon.
func hasHDIcon(k sim.ItemKind) bool {
	return iconHD && k > sim.ItemNone && k < sim.NumItems && hdIcons[k] != nil
}

type iconKey struct {
	k  sim.ItemKind
	px int
}

// iconBaked holds every HD icon baked so far, by item and size. It is only
// touched on the game's own thread.
var iconBaked = map[iconKey]*ebiten.Image{}

// hdIcon is item k's HD icon px pixels across, or nil if it has none.
func hdIcon(k sim.ItemKind, px int) *ebiten.Image {
	if !hasHDIcon(k) || px <= 0 {
		return nil
	}
	key := iconKey{k, px}
	if img, ok := iconBaked[key]; ok {
		return img
	}
	iconWarm.Lock()
	c := iconWarm.ready[key]
	delete(iconWarm.ready, key)
	iconWarm.made[key] = true
	iconWarm.Unlock()
	if c == nil {
		c = bakeIcon(hdIcons[k], px)
	}
	img := ebiten.NewImage(px, px)
	img.WritePixels(c.bytes())
	artBytes.Add(int64(4 * px * px))
	iconBaked[key] = img
	return img
}

// iconWarm holds the icons baked ahead on another goroutine (warmIcons),
// waiting to be made images when first drawn.
var iconWarm = struct {
	sync.Mutex
	ready  map[iconKey]*canvas
	made   map[iconKey]bool // made images already: not to be baked again
	scales map[int]bool     // the UI scales warmed for
}{ready: map[iconKey]*canvas{}, made: map[iconKey]bool{}, scales: map[int]bool{}}

// warmIcons bakes every HD icon, on another goroutine, at the sizes the
// interface shows icons at on a screen of UI scale s (48, 32 and 16 UI
// pixels), so opening the bag the first time does not stall a frame. An
// icon drawn before its turn is baked there and then, as without it.
func warmIcons(s int) {
	iconWarm.Lock()
	defer iconWarm.Unlock()
	if iconWarm.scales[s] {
		return
	}
	iconWarm.scales[s] = true
	go func() {
		for _, size := range []int{48, 32, 16} {
			px := size * s
			if px <= iconOldPx {
				continue
			}
			for k, f := range hdIcons {
				key := iconKey{sim.ItemKind(k), px}
				iconWarm.Lock()
				skip := f == nil || iconWarm.made[key] || iconWarm.ready[key] != nil
				iconWarm.Unlock()
				if skip {
					continue
				}
				c := bakeIcon(f, px)
				iconWarm.Lock()
				if !iconWarm.made[key] {
					iconWarm.ready[key] = c
				}
				iconWarm.Unlock()
			}
		}
	}()
}

// iconWarmedAt is the UI scale warmIcons was last asked for.
var iconWarmedAt int

// drawItem draws item k's icon size UI pixels across with its top-left at
// (x, y) on dst. At 16 screen pixels or less (a recipe's ingredients at
// 720p) the old icon is drawn even if there is an HD one: it was made for
// that size, one of its pixels to one of the screen's, and a shrunk HD icon
// is softer.
func (g *Game) drawItem(dst *uiDst, k sim.ItemKind, x, y, size float64, alpha float32) {
	s := dst.k
	if si := int(math.Round(s)); si != iconWarmedAt && iconHD {
		iconWarmedAt = si
		warmIcons(si)
	}
	old := g.art.icons[k]
	px := int(math.Round(size * s))
	if px > iconOldPx || old == nil {
		if img := hdIcon(k, px); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(math.Round(dst.X(x)), math.Round(dst.Y(y)))
			op.ColorScale.ScaleAlpha(alpha)
			dst.img.DrawImage(img, op)
			return
		}
	}
	if old == nil {
		return
	}
	// Whole times bigger, as the old icons have always been drawn.
	w := float64(old.Bounds().Dx())
	sc := max(1, math.Floor(size/w))
	off := (size - w*sc) / 2
	drawIcon(dst, old, x+off, y+off, sc, alpha)
}

// iconOldPx is the size the old icons were drawn for.
const iconOldPx = 16
