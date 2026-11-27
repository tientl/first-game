package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
)

// The chips for what the beasts' moves leave on the player (sim/status.go),
// beside the poison and the web's (hud.go).

var (
	colStun   = color.RGBA{255, 222, 110, 255}
	colBleed  = color.RGBA{214, 40, 40, 255}
	colSlow   = color.RGBA{150, 200, 90, 255}
	colPinned = color.RGBA{236, 150, 70, 255}
)

// skillStatuses are the chips for self's statuses from the beasts' moves.
func skillStatuses(self protocol.SelfState) []status {
	var out []status
	if self.Pinned {
		out = append(out, status{statusPinned, "Bị đè", "Bị đè xuống! Bấm đánh liên tục để vùng ra", colPinned})
	}
	if self.Stunned {
		out = append(out, status{statusStun, "Choáng", "Choáng váng — không đi, không đánh được", colStun})
	}
	if self.Bleeding {
		out = append(out, status{statusBleed, "Chảy máu", "Chảy máu! Máu tụt dần một lúc", colBleed})
	}
	if self.Slowed {
		out = append(out, status{statusSlow, "Chậm", "Trúng nọc Troodon — đi chậm lại", colSlow})
	}
	return out
}

// drawSkillStatusIcon draws the little picture of status k centred on (x, y).
func drawSkillStatusIcon(dst *ebiten.Image, k statusKind, x, y float32, c color.RGBA) {
	switch k {
	case statusStun: // stars round a head
		for i := range 3 {
			a := float64(i)*2*math.Pi/3 - math.Pi/2
			sx, sy := x+float32(math.Cos(a))*5, y+float32(math.Sin(a))*5
			vector.FillCircle(dst, sx, sy, 2.2, c, true)
		}
		vector.StrokeCircle(dst, x, y, 5, 1, fade(c, 140), true)
	case statusBleed: // a drop of blood
		drawDrop(dst, x, y, 13, c)
	case statusSlow: // a snail's shell: a spiral
		var px, py float32
		for i := 0; i <= 24; i++ {
			a := float64(i) * 0.5
			r := float32(1 + a*0.55)
			qx, qy := x+r*float32(math.Cos(a)), y+r*float32(math.Sin(a))
			if i > 0 {
				vector.StrokeLine(dst, px, py, qx, qy, 1.6, c, true)
			}
			px, py = qx, qy
		}
	case statusPinned: // a claw over a bar
		vector.StrokeLine(dst, x-7, y+5, x+7, y+5, 2, c, true)
		for i := range 3 {
			dx := float32(i-1) * 4
			vector.StrokeLine(dst, x+dx-1, y-6, x+dx+1, y+3, 1.8, c, true)
		}
	}
}
