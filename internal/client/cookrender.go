package client

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Drawing the cooking gear in the world: the pot, grill or pan over a
// campfire (between its flames and its front stones), the built stations,
// their steam and smoke, the hints over them, and the chips of the buffs
// dishes give. The art is in cookart.go, the window in cookui.go.

// cookFlags are the sim.CookFlag bits of e.
func cookFlags(e protocol.EntityView) (loaded, working, done bool) {
	return e.Cook&sim.CookFlagLoaded != 0, e.Cook&sim.CookFlagWorking != 0, e.Cook&sim.CookFlagDone != 0
}

// cookSprite is the sprite of a built station or a clay bed.
func (g *Game) cookSprite(e protocol.EntityView) sprite {
	a := &g.art.cook
	loaded, _, done := cookFlags(e)
	switch e.Kind {
	case sim.EntityClay:
		if e.Ready {
			return a.clay[0]
		}
		return a.clay[1]
	case sim.EntityMortar:
		if loaded || done {
			return a.mortar[1]
		}
		return a.mortar[0]
	case sim.EntityOven:
		i := 0
		if e.Fuel > 0 {
			i = 1
		}
		if done {
			i += 2
		}
		return a.oven[i]
	}
	switch {
	case loaded && e.Cook&sim.CookFlagSalt != 0:
		return a.rack[3]
	case loaded && e.Cook&sim.CookFlagFish != 0:
		return a.rack[2]
	case loaded:
		return a.rack[1]
	case done:
		return a.rack[4]
	}
	return a.rack[0]
}

// drawCookEntity draws e if it is a campfire with gear on it, and reports
// whether it was. (The built stations are plain sprites: see cookSprite.)
func (g *Game) drawCookEntity(dst drawTarget, e protocol.EntityView, d drawable) bool {
	if e.Kind != sim.EntityCampfire || sim.StationOf(e.Kind, e.Var) == sim.StationNone {
		return false
	}
	a := &g.art.cook
	loaded, working, done := cookFlags(e)
	var back, front sprite
	switch sim.CookGear(e.Var) {
	case sim.GearPot:
		back, front = a.potBack, a.pot[0]
		if loaded || done {
			front = a.pot[1]
		}
		if working {
			front = a.pot[1+(g.frame/14+int(e.ID))%3] // the soup turning over
		}
	case sim.GearGrill:
		back, front = a.grillBack, a.grill[0]
		if loaded || done {
			front = a.grill[1]
		}
	default:
		back, front = a.panBack, a.pan[0]
		if loaded || done {
			front = a.pan[1]
		}
	}
	drawAt(dst, g.art.campBack, d.sx, d.sy, 1, 1, 0, nil)
	drawAt(dst, back, d.sx, d.sy, 1, 1, 0, nil)
	if e.Fuel > 0 {
		size := 0.6 + 0.4*min(1, float64(e.Fuel)/sim.FireStartFuel)
		f := g.art.flames[(g.frame/5+int(e.ID))%flameFrames]
		drawAt(dst, f, d.sx, d.sy+1, size, size*0.8, 0, nil) // kept low, under what cooks
	}
	drawAt(dst, front, d.sx, d.sy, 1, 1, 0, nil)
	drawAt(dst, g.art.campFront, d.sx, d.sy, 1, 1, 0, nil)
	return true
}

// drawOvenFire is the fire in a lit oven's mouth.
func (g *Game) drawOvenFire(dst drawTarget, e protocol.EntityView, d drawable) {
	if e.Kind != sim.EntityOven || e.Fuel <= 0 {
		return
	}
	f := g.art.flames[(g.frame/4+int(e.ID))%flameFrames]
	drawAt(dst, f, d.sx-1.5, d.sy-0.5, 0.5, 0.36, 0, nil)
}

// puffs draws n puffs rising from (x, y) one after another: each climbs
// rise art pixels over its life, drifts with the wind, grows from r0 to r1
// art pixels across and fades. tint is their colour at their thickest.
func (g *Game) puffs(dst drawTarget, x, y float64, id sim.EntityID, n int, period, rise, r0, r1 float64, tint color.RGBA) {
	t := float64(g.frame)
	wx := math.Cos(float64(g.curr.WindDir)) * (0.25 + float64(g.curr.Wind))
	for i := 0; i < n; i++ {
		seed := float64(id)*0.731 + float64(i)*1.618
		ph := math.Mod(t/period+seed, 1)
		px := x + math.Sin(seed*9.1)*2 + math.Sin(ph*4+seed*3)*2.2*ph + wx*ph*ph*14
		py := y - ph*rise
		a := math.Sin(math.Min(1, ph*4)*math.Pi/2) * (1 - ph) * (1 - ph)
		s := g.art.cook.puff[(i+int(id))%len(g.art.cook.puff)]
		k := (r0 + (r1-r0)*ph) / s.w()
		c := color.RGBA{uint8(float64(tint.R) * a), uint8(float64(tint.G) * a), uint8(float64(tint.B) * a), uint8(float64(tint.A) * a)}
		drawAt(dst, s, px, py, k, k, seed*2+ph, c)
	}
}

// drawCookFX draws the steam of simmering pots, the smoke of grills, pans
// and lit ovens, and the sparkle of salt forming. Like the embers it is a
// pure function of time.
func (g *Game) drawCookFX(dst drawTarget, cam camera) {
	steam := color.RGBA{236, 240, 246, 150}
	smoke := color.RGBA{150, 150, 158, 120}
	for _, e := range g.curr.Entities {
		st := sim.StationOf(e.Kind, e.Var)
		if st == sim.StationNone {
			continue
		}
		loaded, working, _ := cookFlags(e)
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		switch st {
		case sim.StationPot:
			if working {
				g.puffs(dst, sx, sy-29, e.ID, 6, 150, 30, 5, 15, steam)
			} else if loaded && e.Fuel > 0 {
				g.puffs(dst, sx, sy-29, e.ID, 2, 190, 18, 4, 9, steam)
			}
		case sim.StationGrill:
			if working {
				g.puffs(dst, sx-3, sy-12, e.ID, 4, 130, 30, 4, 13, smoke)
				g.puffs(dst, sx+5, sy-12, e.ID+7, 2, 170, 24, 3, 9, steam)
			}
		case sim.StationPan:
			if working {
				g.puffs(dst, sx, sy-12, e.ID, 4, 110, 22, 3, 10, steam)
				g.sizzle(dst, sx, sy-11, e.ID)
			}
		case sim.StationOven:
			if e.Fuel > 0 {
				n := 3
				if working {
					n = 6 // baking: the fire is fed and draws well
				}
				g.puffs(dst, sx+3.5, sy-18, e.ID, n, 200, 40, 4, 16, smoke)
			}
		case sim.StationRack:
			if working && e.Cook&sim.CookFlagSalt != 0 {
				g.puffs(dst, sx, sy-17, e.ID, 2, 260, 14, 3, 8, color.RGBA{220, 232, 244, 90})
			}
		}
	}
}

// sizzle spits a few drops of fat off a hot pan at (x, y).
func (g *Game) sizzle(dst drawTarget, x, y float64, id sim.EntityID) {
	t := float64(g.frame)
	for i := 0; i < 5; i++ {
		seed := float64(id)*0.37 + float64(i)*2.618
		ph := math.Mod(t/(26+float64(i)*5)+seed, 1)
		px := x + math.Sin(seed*7)*9 + math.Sin(seed*3)*4*ph
		py := y - math.Sin(ph*math.Pi)*(4+math.Mod(seed, 1)*5)
		a := uint8(230 * (1 - ph))
		fillRect(dst, float32(math.Round(px)), float32(math.Round(py)), 1, 1, color.RGBA{a, uint8(int(a) * 220 / 255), uint8(int(a) * 150 / 255), a}, false)
	}
}

// cookLabel is the hint over cooking gear e, "" if e is none.
func (g *Game) cookLabel(e protocol.EntityView) string {
	if e.Kind == sim.EntityClay {
		return "Bãi đất sét — đào lấy đất sét"
	}
	st := sim.StationOf(e.Kind, e.Var)
	if st == sim.StationNone {
		if e.Kind == sim.EntityCampfire {
			if gear := sim.GearOf(g.myHeld()); gear != sim.GearNone {
				return "Lửa trại — thêm củi · V: đặt " + g.myHeld().Def().Name + " lên"
			}
		}
		return ""
	}
	loaded, working, done := cookFlags(e)
	label := st.Name()
	switch {
	case done:
		label += " — có món đã xong, mở lấy ra"
	case working:
		label += " — đang nấu · mở"
	case loaded && st.NeedsHeat() && e.Fuel <= 0:
		label += " — lửa tắt, mở để thêm củi"
	default:
		label += " — mở"
	}
	return label
}

// isCookRecipe reports whether r makes cooking gear: it is listed with the
// cooking in the crafting window.
func isCookRecipe(r sim.Recipe) bool {
	return r.ID >= sim.RecipeClayPot && r.ID <= sim.RecipeSaltMeat && r.ID != sim.RecipeFlask
}

// --- the buffs of dishes ------------------------------------------------------------

var buffColors = [sim.NumBuffs]color.RGBA{
	sim.BuffRegen: {250, 120, 130, 255},
	sim.BuffWarm:  {255, 176, 84, 255},
	sim.BuffSwift: {130, 226, 240, 255},
	sim.BuffMight: {255, 210, 90, 255},
	sim.BuffSated: {186, 226, 120, 255},
}

var buffLongs = [sim.NumBuffs]string{
	sim.BuffRegen: "Canh nóng vào bụng — máu hồi nhanh",
	sim.BuffWarm:  "Ấm bụng — không sợ lạnh",
	sim.BuffSwift: "Nhẹ người — chạy nhanh hơn",
	sim.BuffMight: "Sung sức — đánh mạnh hơn",
	sim.BuffSated: "No lâu — bụng đói chậm lại",
}

// buffStatuses are the chips of the buffs I have, each with its time left.
func (g *Game) buffStatuses() []status {
	var out []status
	for b, t := range g.curr.Self.Cook.Buffs {
		if t <= 0 {
			continue
		}
		k := sim.BuffKind(b)
		out = append(out, status{statusBuff + statusKind(b), fmt.Sprintf("%s %s", k.Name(), clockText(int(t))), buffLongs[b], buffColors[b]})
	}
	return out
}

// drawBuffIcon draws buff b's little picture centred on (x, y).
func drawBuffIcon(dst *ebiten.Image, b sim.BuffKind, x, y float32, c color.RGBA) {
	line := func(x0, y0, x1, y1, w float32) { vector.StrokeLine(dst, x+x0, y+y0, x+x1, y+y1, w, c, true) }
	switch b {
	case sim.BuffRegen: // a heart with a plus
		vector.FillCircle(dst, x-2.6, y-1.6, 3.2, c, true)
		vector.FillCircle(dst, x+2.6, y-1.6, 3.2, c, true)
		var p vector.Path
		p.MoveTo(x-5.6, y-0.4)
		p.LineTo(x+5.6, y-0.4)
		p.LineTo(x, y+6)
		p.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(c)
		vector.FillPath(dst, &p, nil, op)
		vector.StrokeLine(dst, x-2.4, y, x+2.4, y, 1.5, colWhite, true)
		vector.StrokeLine(dst, x, y-2.4, x, y+2.4, 1.5, colWhite, true)
	case sim.BuffWarm: // a flame
		drawDrop(dst, x, y, 13, c)
		drawDrop(dst, x, y+2, 7, color.RGBA{255, 236, 170, 255})
	case sim.BuffSwift: // a boot's wing: three streaks and an arrowhead
		line(-6, -4, 1, -4, 1.6)
		line(-7, 0, 2, 0, 1.6)
		line(-6, 4, 1, 4, 1.6)
		line(2, -5, 7, 0, 1.8)
		line(2, 5, 7, 0, 1.8)
	case sim.BuffMight: // a fist: a blade crossed
		line(-5, 5, 5, -5, 2.2)
		line(2, -6, 6, -2, 1.6)
		line(-6, 2, -2, 6, 2.6)
		vector.FillCircle(dst, x+5, y-5, 1.4, colWhite, true)
	case sim.BuffSated: // a full bowl
		var p vector.Path
		p.MoveTo(x-6.5, y-1)
		p.LineTo(x+6.5, y-1)
		p.ArcTo(x+6.5, y+6, x, y+6, 6)
		p.ArcTo(x-6.5, y+6, x-6.5, y-1, 6)
		p.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(c)
		vector.FillPath(dst, &p, nil, op)
		vector.FillCircle(dst, x-2.5, y-2.5, 2.4, c, true)
		vector.FillCircle(dst, x+2, y-3, 2.8, c, true)
	}
}

// cookPlaced reports whether held is cooking gear used with V or the right
// button: a pot, grill or pan set on a fire, a bowl or flask filled.
func cookPlaced(held sim.ItemKind) bool {
	return sim.GearOf(held) != sim.GearNone || held == sim.ItemClayBowl || held == sim.ItemFlask
}

// cookUseLabel names what the right button does with held, for the help.
func cookUseLabel(held sim.ItemKind) string {
	switch held {
	case sim.ItemClayBowl:
		return "múc nước biển (đứng ở mép biển)"
	case sim.ItemFlask:
		return "múc nước ngọt (đứng ở mép ao, hồ)"
	}
	return "đặt lên lửa trại gần nhất"
}

// cookNoticeText is noticeText for the notices of cooking.
func cookNoticeText(kind sim.NoticeKind, arg uint8) string {
	name := sim.ItemKind(arg).Def().Name
	switch kind {
	case sim.NoticeCantCook:
		return name + " không dùng được ở đây"
	case sim.NoticeCookFull:
		return "Hết ô nguyên liệu cho " + name
	case sim.NoticeCookNotEmpty:
		return "Lấy hết đồ ra trước rồi mới dỡ được"
	case sim.NoticeNeedHearth:
		return name + ": đến cạnh một lửa trại chưa bắc gì lên rồi nhấn V"
	case sim.NoticeNoWater:
		return "Đứng sát mép nước rồi nhấn V để múc"
	case sim.NoticeWrongWater:
		if sim.ItemKind(arg) == sim.ItemFlask {
			return "Nước biển mặn không uống được — Bình da chỉ múc nước ao, hồ"
		}
		return "Bát đất múc nước biển để phơi muối — nước ngọt thì dùng Bình da"
	case sim.NoticeScooped:
		return "Đã múc: " + name
	}
	return ""
}
