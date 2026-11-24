package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Fishing lines: a line from the rod's tip to a float on the water, the
// float dipping with a "!" when the fish bites, a hint for the angler, and
// the plop. The rules are in sim/fishing.go.

var (
	colLine    = color.RGBA{226, 230, 236, 190}
	colFloatHi = hex(0xe8463c)
	colFloatLo = hex(0xf4f0e6)
)

// rodTip is where the tip of a held rod is, in art pixels, for a player
// standing at (sx, sy). It follows drawHeld: the rod sprite's tip is 23.5
// pixels above its grip, mirrored by the facing, then tilted.
func rodTip(dir sim.Dir, sx, sy float64) (float64, float64) {
	side := facingSide(dir)
	hx, hy := handAt(dir, 0)
	x0, y0 := math.Round(sx+hx), math.Round(sy+hy)+1
	tilt := 0.5
	if dir == sim.DirLeft || dir == sim.DirRight {
		tilt = 0.9
	}
	th := tilt * side
	px, py := -0.5*side, -23.5
	return x0 + px*math.Cos(th) - py*math.Sin(th), y0 + px*math.Sin(th) + py*math.Cos(th)
}

// floatAt is where p's float is drawn, in art pixels: bobbing while it
// waits, pulled under while the fish bites.
func (g *Game) floatAt(p protocol.PlayerView, cam camera) (float64, float64) {
	x, y := cam.toScreen(float64(p.BobX), float64(p.BobY))
	if p.Cast == sim.CastBite {
		return x + math.Round(math.Sin(float64(g.frame)*0.9)), y + 2
	}
	return x, y + math.Round(math.Sin(float64(g.frame)*0.08+float64(p.ID))*0.6)
}

// drawLines draws every fishing line that is out.
func (g *Game) drawLines(dst *ebiten.Image, cam camera, alpha float64) {
	for _, p := range g.curr.Players {
		if p.Cast == sim.CastNone || p.Dead {
			continue
		}
		wx, wy, _ := g.lerpPlayer(p, alpha)
		px, py := cam.toScreen(wx, wy)
		tx, ty := rodTip(p.Dir&3, px, py)
		fx, fy := g.floatAt(p, cam)

		// The line sags between the tip and the float, more while slack.
		sag := 6.0
		if p.Cast == sim.CastBite {
			sag = 1 // pulled taut
		}
		n := 2*int(math.Hypot(fx-tx, fy-ty)+sag) + 2 // dense enough for no gaps
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(n)
			x := tx + (fx-tx)*t
			y := ty + (fy-ty)*t + sag*4*t*(1-t)
			vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 1, 1, colLine, false)
		}

		// The float: red over white, half under water when the fish bites.
		fx, fy = math.Round(fx), math.Round(fy)
		vector.FillRect(dst, float32(fx-1), float32(fy-3), 3, 2, colFloatHi, false)
		if p.Cast != sim.CastBite {
			vector.FillRect(dst, float32(fx-1), float32(fy-1), 3, 2, colFloatLo, false)
		}
		if p.Cast == sim.CastBite { // rings spreading from the tug
			ph := math.Mod(float64(g.frame)/14, 1)
			r := 2 + ph*6
			a := uint8(170 * (1 - ph))
			for k := range 16 {
				ang := 2 * math.Pi * float64(k) / 16
				x, y := math.Round(fx+r*math.Cos(ang)), math.Round(fy+r*0.45*math.Sin(ang))
				vector.FillRect(dst, float32(x), float32(y), 1, 1, color.RGBA{a, a, a, a}, false)
			}
		}
	}
}

// drawFishingLabels draws, at screen resolution, the "!" over a float the
// fish is tugging, and what the local angler should do now.
func (g *Game) drawFishingLabels(screen *ebiten.Image, cam camera, scale float64, sw, sh int) {
	blink := (g.frame/6)%2 == 0
	for _, p := range g.curr.Players {
		if p.Cast != sim.CastBite || p.Dead {
			continue
		}
		fx, fy := g.floatAt(p, cam)
		c := colKey
		if blink {
			c = colWhite
		}
		g.text(screen, "!", (fx+7)*scale, (fy-20)*scale-12, c, text.AlignCenter, 2) // beside the line, not on it
	}

	me, ok := g.me()
	if !ok || me.Cast == sim.CastNone || me.Dead || g.overlay != overlayNone {
		return
	}
	msg, col := "Đợi cá cắn câu…   (đi chỗ khác để thu dây)", color.Color(colMuted)
	if me.Cast == sim.CastBite {
		msg, col = "Cá cắn câu! Nhấp chuột!", colKey
		if blink {
			col = colWhite
		}
	}
	w := float32(g.textWidth(msg)) + 28
	y := float32(sh) - 128
	panel(screen, float32(sw)/2-w/2, y, w, 28)
	g.text(screen, msg, float64(sw)/2, float64(y)+6, col, text.AlignCenter, 1)
}

// me returns the local player's view in the current snapshot.
func (g *Game) me() (protocol.PlayerView, bool) {
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			return p, true
		}
	}
	return protocol.PlayerView{}, false
}

// fishingSounds plays the plop of a bite and the outcome of a strike.
func (g *Game) fishingSounds(old, cur protocol.Snapshot) {
	was := make(map[sim.PlayerID]sim.CastState, len(old.Players))
	for _, p := range old.Players {
		was[p.ID] = p.Cast
	}
	for _, p := range cur.Players {
		if p.Cast == sim.CastBite && was[p.ID] != sim.CastBite {
			snd.playAt("plop", float64(p.BobX), float64(p.BobY), 0.6, 0.08)
		}
	}
	if cur.Self.NoticeSeq == old.Self.NoticeSeq {
		return
	}
	switch cur.Self.Notice {
	case sim.NoticeCaught:
		snd.playVar("splash", 0.6, 0, 0.06)
	case sim.NoticeTooEarly, sim.NoticeBaitStolen:
		snd.playVar("splash", 0.3, 0, 0.1)
	}
}
