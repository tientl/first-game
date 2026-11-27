package client

import (
	"image/color"
	"math"
	"sync"

	"survivalgame/internal/sim"
)

// The HD icons of the tools and weapons that have held painters
// (heldhd_art.go): each icon is its held painter run on the icon's canvas,
// so what is in the bag is what is in the hand, to the scar and the turn of
// the lashing. The tool lies from the bottom left to the top right, its
// working end up, mirrored so the edge of the head faces the light, and is
// scaled to fill the box.

func init() {
	for t := sim.TierStone; t < sim.NumTiers; t++ {
		regIcon(sim.ToolItem(sim.ToolAxe, t), fromHeld(sim.ToolItem(sim.ToolAxe, t), 0.66, 1))
		regIcon(sim.ToolItem(sim.ToolPickaxe, t), fromHeld(sim.ToolItem(sim.ToolPickaxe, t), 0.7, 1))
		regIcon(sim.ToolItem(sim.ToolHoe, t), fromHeld(sim.ToolItem(sim.ToolHoe, t), 0.66, 1))
		if t > sim.TierStone {
			regIcon(sim.SwordItem(t), fromHeld(sim.SwordItem(t), math.Pi/4, 0.8))
		}
	}
	regIcon(sim.ItemSpear, fromHeld(sim.ItemSpear, math.Pi/4, 1.45))
	regIcon(sim.ItemIvorySpear, fromHeld(sim.ItemIvorySpear, math.Pi/4, 1.5))
	regIcon(sim.ItemBattleAxe, fromHeld(sim.ItemBattleAxe, 0.62, 1))
	regIcon(sim.ItemRod, icRod)
	regIcon(sim.ItemTorch, icTorchHeld)
}

// heldFit is where a held item's painting reaches, in its own frame turned
// by ang (and mirrored), su times as thick: x0, y0, x1, y1 in art pixels
// from the grip.
type heldFit struct{ x0, y0, x1, y1 float64 }

var heldFits = struct {
	sync.Mutex
	m map[heldFitKey]heldFit
}{m: map[heldFitKey]heldFit{}}

type heldFitKey struct {
	d       *heldDef
	ang, su float64
}

// fitOf measures what d paints turned by ang, su times as thick: it is
// painted once, coarsely, on a scratch canvas, and the box of what it
// covered kept.
func fitOf(d *heldDef, ang, su float64) heldFit {
	key := heldFitKey{d, ang, su}
	heldFits.Lock()
	f, ok := heldFits.m[key]
	heldFits.Unlock()
	if ok {
		return f
	}
	const k, n = 4.0, 200 // pixels to an art pixel; the canvas, room for 50 art pixels
	p := &icv{n: n, k: k, px: make([]color.RGBA, n*n), part: make([]uint8, n*n)}
	h := &hpv{icv: p, cs: math.Cos(ang), sn: math.Sin(ang), ox: n / k / 2, oy: n / k / 2, su: su, sv: 1, flip: true, icon: true}
	d.paint(h)
	x0, y0, x1, y1 := n, n, -1, -1
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if p.px[y*n+x].A >= 128 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	c := n / k / 2
	f = heldFit{float64(x0)/k - c, float64(y0)/k - c, float64(x1+1)/k - c, float64(y1+1)/k - c}
	heldFits.Lock()
	heldFits.m[key] = f
	heldFits.Unlock()
	return f
}

// iconLo and iconHi bound what an icon fills of its design box.
const iconLo, iconHi = 0.55, 15.45

// heldOn paints held item d on p turned by ang from upright (clockwise,
// radians), mirrored, su times as thick for its length (a long spear would
// be a hair at a slot's size), scaled to fit the box x0, y0, x1, y1 (design
// units) and centred in it. It returns the frame it painted in with the icon still
// at the item's scale: call done when finished with it.
func heldOn(p *icv, d *heldDef, ang, su, x0, y0, x1, y1 float64) *heldPlace {
	f := fitOf(d, ang, su)
	sc := min((x1-x0)/(f.x1-f.x0), (y1-y0)/(f.y1-f.y0))
	gx := (x0+x1)/2 - (f.x0+f.x1)/2*sc
	gy := (y0+y1)/2 - (f.y0+f.y1)/2*sc
	k := p.k
	p.k *= sc
	hp := &hpv{icv: p, cs: math.Cos(ang), sn: math.Sin(ang), ox: gx / sc, oy: gy / sc, su: su, sv: 1, flip: true, icon: true}
	d.paint(hp)
	return &heldPlace{hpv: hp, k: k, sc: sc}
}

// heldPlace is a held item painted on an icon, the icon still scaled to it.
type heldPlace struct {
	*hpv
	k, sc float64
}

// done sets the icon back to its own design units.
func (h *heldPlace) done() { h.icv.k = h.k }

// design is item point (u, v) in the icon's design units.
func (h *heldPlace) design(u, v float64) (float64, float64) {
	x, y := h.at(u, v)
	return x * h.sc, y * h.sc
}

// fromHeld is the icon painter of item k from its held painter, turned by
// ang, su times as thick, filling the box.
func fromHeld(k sim.ItemKind, ang, su float64) iconPainter {
	return func(p *icv) { heldOn(p, heldDefs[k], ang, su, iconLo, iconLo, iconHi, iconHi).done() }
}

// icTorchHeld is the held torch with its flame burning upright on it.
func icTorchHeld(p *icv) {
	h := heldOn(p, heldTorchDef, 0.62, 1.1, iconLo, 5.4, 11.4, iconHi)
	// The flame and its glow, in the torch's scale, rooted in its head.
	x, y := h.at(0, torchFlameV-0.5)
	paintTorchFlame(p, x, y, 0.35)
	h.done()
}

// icRod is the fishing rod with its line hanging from the tip to a float
// and a bone hook.
func icRod(p *icv) {
	h := heldOn(p, heldRodDef, 0.72, 1.3, iconLo, iconLo, 12.6, iconHi)
	tx, ty := h.design(rodTipU, rodTipV+0.4)
	h.done()
	// The line from the tip, slack, down to the float.
	fx, fy := tx+0.6, ty+6.4
	p.fill(sLock(tx, ty, tx+1.4, ty+3.0, fx, fy, 0.11, 0.11), solidC(hex(0xe8e0cc)))
	p.bareLast()
	// The float: a red and white bob.
	p.fill(sEll(fx, fy+0.9, 0.95, 1.2), func(f *ifrag) color.RGBA {
		l := litOf((f.x-fx)/0.95*0.9, (f.y-fy-0.9)/1.2*0.9)
		if f.y < fy+0.75 {
			return pick(iiRed, l+0.05)
		}
		return pick(iiWhite, l)
	})
	p.on(sEll(fx-0.35, fy+0.35, 0.22, 0.16), solidC(icGlint))
	// The hook under it, a curl of bone.
	p.fill(sPath(0.11, 0.11, fx, fy+2.1, fx, fy+3.2), solidC(hex(0xe8e0cc)))
	p.bareLast()
	p.fill(sPath(0.16, 0.1, fx, fy+3.1, fx, fy+4.4, fx-0.4, fy+5.0, fx-1.0, fy+4.8, fx-1.1, fy+4.1), shBulge(icBoneR, 0.15, 0.1))
}
