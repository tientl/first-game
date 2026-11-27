package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The round wooden shield (sim.ItemShield, see sim/shield.go): boards set
// edge to edge, two quarters daubed with red ochre, a rim of rawhide laced
// on, and a domed boss over the grip. It hangs on the back of anyone who
// carries one, and is raised on the other arm with Q or the right button
// (moGuard): before the chest from the front, held out at the foe seen from
// the side, from behind only its back and rim show past the body.
//
// The keys: Q held raises it, always; the right button held raises it when
// what is in the hand has no use for the right button (a weapon, a tool,
// nothing at all), so it never takes the place of setting down a torch,
// eating, feeding a beast or launching a boat. The left button, the aim and
// Space are untouched: behind the shield no blow is struck.

// shieldR is the shield's radius in art pixels: about the width of the
// trunk, a little under half the figure's height across.
const shieldR = 6.4

// shieldLayer is where in a figure's layers the shield goes.
type shieldLayer uint8

const (
	shieldBehind shieldLayer = iota // behind the whole figure
	shieldOnBody                    // over the trunk, under the near arm
	shieldOver                      // over everything
)

// The ways the shield is seen.
const (
	shieldFace     = iota // its face, square on
	shieldBack            // its back: boards, the grip bar, the rim folded over
	shieldSideFace        // its face turned away, seen at a slant, its edge showing
	shieldSideBack        // its face turned the other way: slimmer
	numShieldViews
)

var shieldSprites [numShieldViews]*sprite

// Its materials, dark to light.
var (
	shWoodR  = mkRamp(0x2e1d10, 0x4a2f19, 0x6b4526, 0x8c5f34, 0xa97a46, 0xc49a62)
	shOchreR = mkRamp(0x3e140c, 0x5e1f12, 0x80301a, 0x9c4426, 0xb45c34)
	shHideR  = mkRamp(0x3a2a1a, 0x5a4228, 0x7a5c3a, 0x9c7c52, 0xbca070)
	shBossR  = mkRamp(0x40342a, 0x6a5a48, 0x958268, 0xbfae90, 0xe4d8bc)
	shInk    = hex(0x1c120a)
)

// shieldSprite is the shield seen as view, painted the first time.
func shieldSprite(view int) sprite {
	if s := shieldSprites[view]; s != nil {
		return *s
	}
	sx, front := 1.0, true
	switch view {
	case shieldBack:
		front = false
	case shieldSideFace:
		sx = 0.5
	case shieldSideBack:
		sx = 0.3
	}
	s := paintShield(shieldR, sx, front)
	shieldSprites[view] = &s
	return s
}

// paintShield paints the shield, radius r art pixels, squashed across to sx
// of its width (turned away), its face or its back, at HDDensity, anchored
// at its middle. Turned, its face looks to the right and the thickness of
// its boards shows on the left.
func paintShield(r, sx float64, front bool) sprite {
	const K = HDDensity
	edge := (1 - sx) * 1.5 // art pixels of the boards' edge seen
	wArt := 2*r*sx + edge + 2
	hArt := 2*r + 2
	W, H := int(math.Ceil(wArt*K)), int(math.Ceil(hArt*K))
	cx, cy := 1+edge+r*sx, 1+r
	px := make([]color.RGBA, W*H)
	sample := func(ax, ay float64) (color.RGBA, bool) {
		du, dv := ax/(r*sx), ay/r
		d := math.Hypot(du, dv)
		if d <= 1 {
			return shieldPx(du, dv, d, sx, front), true
		}
		if edge > 0 && ax < 0 {
			if math.Hypot((ax+edge)/(r*sx), dv) <= 1 { // the boards' edge, wrapped in hide
				l := 0.35 - dv*0.2
				return pick(shHideR, l), true
			}
		}
		return color.RGBA{}, false
	}
	const ss = 3
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			var cr, cg, cb, n float64
			var mid color.RGBA
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					ax := (float64(x)+(float64(i)+0.5)/ss)/K - cx
					ay := (float64(y)+(float64(j)+0.5)/ss)/K - cy
					if c, ok := sample(ax, ay); ok {
						cr, cg, cb, n = cr+float64(c.R), cg+float64(c.G), cb+float64(c.B), n+1
						if i == 1 && j == 1 {
							mid = c
						}
					}
				}
			}
			if n == 0 {
				continue
			}
			if mid.A == 0 {
				mid = color.RGBA{uint8(cr / n), uint8(cg / n), uint8(cb / n), 255}
			}
			mid.A = uint8(255 * n / (ss * ss))
			px[y*W+x] = mid
		}
	}
	// A dark line round it, a pixel wide.
	out := make([]color.RGBA, len(px))
	copy(out, px)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			c := px[y*W+x]
			if c.A < 128 {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H || px[ny*W+nx].A < 128 {
					out[y*W+x] = mix(c, shInk, 0.75)
					out[y*W+x].A = c.A
					break
				}
			}
		}
	}
	img := ebiten.NewImage(W, H)
	img.WritePixels(straightBytes(out))
	artBytes.Add(int64(4 * W * H))
	return spriteHD(img, cx, cy)
}

// shieldPx is the colour of the shield at (du, dv) in its own unit disc, d
// from the middle; sx is how far it is turned.
func shieldPx(du, dv, d, sx float64, front bool) color.RGBA {
	// The light from the top left on a face very slightly domed; turned
	// away, it is in its own shade.
	l := 0.55 - 0.18*du - 0.24*dv
	if sx < 1 {
		l -= 0.12
	}
	if !front {
		l -= 0.1
	}
	ang := math.Atan2(dv, du)
	switch {
	case d > 0.85: // the rawhide rim, laced on
		ll := l + 0.05 - (d-0.92)*1.2
		if front {
			lace := math.Mod(ang/(2*math.Pi)*18+18, 1)
			if lace < 0.16 && d > 0.875 && d < 0.975 {
				return pick(shHideR, ll-0.35)
			}
		}
		return pick(shHideR, ll)
	case front && d < 0.25: // the boss, a dome of hardwood
		nx, ny := du/0.25, dv/0.25
		ll := lum(nx*0.85, ny*0.85)
		if d > 0.21 {
			ll -= 0.35
		}
		if sx < 1 {
			ll -= 0.1
		}
		return pick(shBossR, ll)
	case !front && math.Abs(dv) < 0.085 && math.Abs(du) < 0.62: // the grip bar across the back
		ll := 0.6 - math.Abs(dv)*4
		if math.Abs(du) < 0.2 {
			return pick(shHideR, ll-0.1) // wrapped where the fist goes
		}
		return pick(shWoodR, ll)
	case !front && d < 0.2: // the hollow behind the boss
		return pick(shWoodR, 0.05)
	}
	// The boards: seven, edge to edge, each its own tone, the grain along it.
	b := (du + 1) * 3.5
	i := math.Floor(b)
	f := b - i
	tone := hashf(int(i), 7, 501)*0.16 - 0.08
	grain := 0.05 * math.Sin(dv*38+hashf(int(i), 3, 502)*9+math.Sin(dv*9+i)*1.5)
	ll := l + tone + grain
	if f < 0.07 || f > 0.95 {
		return pick(shWoodR, ll-0.32) // the seam
	}
	if front && (du > 0) != (dv > 0) { // two quarters daubed with ochre, worn through in places
		worn := hashf(int(du*40+50), int(dv*40+50), 503)
		if worn > 0.1 && math.Abs(du) > 0.04 && math.Abs(dv) > 0.04 {
			return pick(shOchreR, ll+0.05)
		}
	}
	return pick(shWoodR, ll)
}

// drawShieldAt draws the shield seen as view with its middle at (x, y), art
// pixels, mirrored for side -1.
func drawShieldAt(dst drawTarget, view int, x, y, side float64) {
	s := shieldSprite(view)
	if side < 0 {
		s = mirrorSprite(s)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round((x-s.ax)*HDDensity)/HDDensity, math.Round((y-s.ay)*HDDensity)/HDDensity)
	drawSprite(dst, s, op)
}

// Where the shield is on the figure, in art pixels from the upper half's
// point: hung on the back, and how high.
const (
	shieldBackY    = -17.5 // its middle, on the back between the shoulders
	shieldBackSide = 3.2   // seen from the side, how far behind the trunk's middle
)

// shieldDrawer is the figure's shield hook for p facing dir: raised on the
// other arm while they guard, else hung on the back.
func shieldDrawer(p protocol.PlayerView, dir sim.Dir) func(dst drawTarget, at shieldLayer, ux, uy, skew float64, u *upperArt) {
	side := facingSide(dir)
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	return func(dst drawTarget, at shieldLayer, ux, uy, skew float64, u *upperArt) {
		if p.Guard {
			h := u.hand[1]
			hx, hy := handPoint(h, ux, uy, skew)
			switch {
			case sideways && at == shieldOnBody: // held out at the foe, before the chest
				drawShieldAt(dst, shieldSideFace, hx+side*0.6, hy+3.5, side)
			case dir == sim.DirDown && at == shieldOver: // square before them, the fist at its grip
				drawShieldAt(dst, shieldFace, hx+1.5, hy+1, 1)
			case dir == sim.DirUp && at == shieldBehind: // before them, away from us: its back past the body
				drawShieldAt(dst, shieldBack, ux-2.5-skew*(shieldBackY+hipHeight), uy+shieldBackY+1, 1)
			}
			return
		}
		switch {
		case sideways && at == shieldBehind: // hung on the back, behind it
			drawShieldAt(dst, shieldSideBack, ux-side*shieldBackSide-skew*(shieldBackY+hipHeight), uy+shieldBackY, -side)
		case dir == sim.DirDown && at == shieldBehind: // behind them: its rim over and past the shoulders
			drawShieldAt(dst, shieldBack, ux+0.5, uy+shieldBackY-2.5, 1)
		case dir == sim.DirUp && at == shieldOnBody: // their back to us: its face, slung there
			drawShieldAt(dst, shieldFace, ux, uy+shieldBackY+0.5, 1)
		}
	}
}

// --- the keys ------------------------------------------------------------------

// guardHeld reports whether the player holds the shield up this frame: Q,
// or the right button when the hand has nothing else to do with it.
func (g *Game) guardHeld() bool {
	if g.curr.Self.Inv.Count(sim.ItemShield) == 0 {
		return false
	}
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return true
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) || g.build.on || (g.freeBuild() && shiftHeld()) {
		return false
	}
	return g.rightGuards()
}

// rightGuards reports whether the right button raises the shield now: the
// player carries one and what is in the hand has no other use for it.
func (g *Game) rightGuards() bool {
	if g.curr.Self.Inv.Count(sim.ItemShield) == 0 {
		return false
	}
	cmd, _ := rightClickUse(g.myHeld(), g.hotSel, g.useContextNow())
	return cmd.Kind == sim.CmdNone
}

// guardPressed reports a fresh press of the right button that raises the
// shield, which then says nothing about the hand's use.
func (g *Game) guardPressed() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && g.rightGuards()
}

// --- the bag ---------------------------------------------------------------------

func init() {
	regIcon(sim.ItemShield, icShield)
	itemInfo[sim.ItemShield] = "Khiên gỗ tròn: ván gỗ ghép, vành da sống, núm gỗ giữa. Có trong túi là đeo sau lưng. " +
		"Giữ Q (hoặc chuột phải khi tay cầm vũ khí, dụng cụ hay tay không) để giơ khiên: đòn đánh từ phía trước mất 3/4 sức, " +
		"giơ đúng lúc (¼ giây trước khi trúng) thì gạt hẳn đòn của thú nhỏ và làm nó loạng choạng. Giơ khiên thì đi chậm và không đánh được."
}

// icShield paints the shield's HD icon: its face square on, slightly
// turned, as it hangs on a back.
func icShield(p *icv) {
	const cx, cy, r = 8.0, 8.0, 7.2
	p.fill(sEll(cx, cy, r, r), func(f *ifrag) color.RGBA {
		du, dv := (f.x-cx)/r, (f.y-cy)/r
		return shieldPx(du, dv, math.Hypot(du, dv), 1, true)
	})
}

// shieldIcons draws the shield's old 16x16 icon, for the small sizes.
func shieldIcons(icons *[sim.NumItems]*ebiten.Image) {
	c := newCanvas(16, 16)
	c.ellipse(8, 8, 7, 7, func(x, y int, nx, ny float64) color.RGBA {
		return shieldPx(nx, ny, math.Hypot(nx, ny), 1, true)
	})
	c.outline(0.8)
	icons[sim.ItemShield] = c.image()
}
