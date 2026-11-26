package client

import (
	"image/color"
	"strings"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Putting the new plants (sim/flora.go) in the world: their sprites, their
// shadows, what is drawn under them, and what the labels say.

// floraEntitySprite is entitySprite for a wild plant: its sprite and how
// much it sways in the wind.
func floraEntitySprite(e protocol.EntityView) (sprite, float64) {
	f := sim.FloraKind(e.Var) % sim.NumFlora
	sway := 0.0
	if e.Ready {
		switch f {
		case sim.FloraBamboo:
			sway = 0.025
		case sim.FloraFlax, sim.FloraWildRice, sim.FloraHorsetail:
			sway = 0.09
		case sim.FloraWildCorn, sim.FloraHerb:
			sway = 0.04
		case sim.FloraLotus:
			sway = 0.02
		case sim.FloraMangrove:
			sway = 0.008
		}
	}
	return floraSprite(f, e.Ready), sway
}

// floraShadow is the width of the shadow under a wild plant; what floats
// casts its own into the water.
func (g *Game) floraShadow(e protocol.EntityView) float64 {
	f := sim.FloraKind(e.Var) % sim.NumFlora
	if g.tileAt(int(e.TX), int(e.TY)).Sea() {
		return 0
	}
	if !e.Ready {
		return [sim.NumFlora]float64{sim.FloraGourd: 30, sim.FloraPitcher: 20, sim.FloraVine: 14, sim.FloraNightshade: 26,
			sim.FloraAloe: 14, sim.FloraMoss: 28, sim.FloraMangrove: 30 * mangroveS, sim.FloraWildSquash: 26}[f]
	}
	return [sim.NumFlora]float64{
		sim.FloraBamboo: 30, sim.FloraGourd: 30, sim.FloraPitcher: 20, sim.FloraFlax: 14, sim.FloraVine: 22,
		sim.FloraPoisonMushroom: 14, sim.FloraNightshade: 26, sim.FloraAloe: 20, sim.FloraHerb: 16, sim.FloraMoss: 28,
		sim.FloraMangrove: 34 * mangroveS, sim.FloraHorsetail: 16, sim.FloraWildRice: 16, sim.FloraWildCorn: 20, sim.FloraWildSquash: 28,
	}[f]
}

var mangroveRipples sprite

// drawFloraUnder draws what lies under plant e before it: the rings round a
// mangrove's roots where it stands in the water.
func (g *Game) drawFloraUnder(dst drawTarget, e protocol.EntityView, d drawable) {
	if e.Kind != sim.EntityFlora || sim.FloraKind(e.Var) != sim.FloraMangrove || !g.tileAt(int(e.TX), int(e.TY)).Sea() {
		return
	}
	if mangroveRipples.img == nil {
		mangroveRipples = mkUp(genMangroveRipples())
	}
	k := 1 + 0.04*float64((g.frame/20+int(e.ID))%3)
	drawAt(dst, mangroveRipples, d.sx, d.sy, k, k, 0, nil)
}

// floraHidden reports whether plant e is not to be seen: a water plant
// under the ice of a frozen pond.
func (g *Game) floraHidden(e protocol.EntityView) bool {
	return e.Kind == sim.EntityFlora && sim.FloraKind(e.Var).InWater() && g.frozenAt(int(e.TX), int(e.TY))
}

// floraLabel finishes the target label of a wild plant.
func (g *Game) floraLabel(e protocol.EntityView, label string, col color.RGBA) (string, color.RGBA) {
	f := sim.FloraKind(e.Var) % sim.NumFlora
	item, _ := f.Item()
	what := strings.ToLower(item.Def().Name)
	switch {
	case f.NeedsAxe() && !g.has(sim.ToolAxe):
		return label + " — cần rìu (C: chế tạo)", colBad
	case f.NeedsAxe():
		return label + " — chặt lấy " + what, col
	case f == sim.FloraPitcher:
		return label + " — lấy nước mưa trong ấm", col
	case f == sim.FloraPoisonMushroom || f == sim.FloraNightshade:
		return label + " — hái (có độc, đừng ăn!)", col
	}
	return label + " — hái " + what, col
}

// farmLabel finishes the label of an empty bed or a crop.
func (g *Game) farmLabel(e protocol.EntityView, label string, col color.RGBA) (string, color.RGBA) {
	if e.Kind == sim.EntityCrop {
		c := sim.CropKind(e.Var) % sim.NumCrops
		if c == sim.CropPotato {
			return label + " — đào khoai", col
		}
		return label + " — thu hoạch", col
	}
	if e.Big {
		label += " (đã bón phân)"
	}
	inv := g.curr.Self.Inv
	if c, ok := sim.SeedToSow(g.myHeld(), &inv); ok {
		return label + " — gieo " + strings.ToLower(c.Seed().Def().Name), col
	}
	return label + " — cần giống để trồng: khoai lang, thóc, ngô hoặc hạt bí", colBad
}

func init() {
	for k, s := range map[sim.ItemKind]string{
		sim.ItemBamboo:         "Chặt bụi tre trong rừng (cần rìu). Dựng Tường tre, Sàn tre, đóng Bè tre; sau này làm thân tên và ống đựng nước.",
		sim.ItemGourd:          "Hái ở giàn bầu trên đồng cỏ. Ăn được; sau này phơi khô làm bình nước.",
		sim.ItemPitcherWater:   "Nước mưa trong ấm của cây nắp ấm ở đầm lầy: uống được ngay. Trời mưa thì ấm đầy lại rất nhanh.",
		sim.ItemFlax:           "Hái cây lanh hoa xanh trên đồng cỏ. 3 sợi bện thành Dây thừng; cùng Rêu làm Băng rêu.",
		sim.ItemVine:           "Gỡ dây leo trên thân cây khô trong rừng. 2 dây bện thành Dây thừng thô.",
		sim.ItemRope:           "Bện từ sợi lanh hoặc dây leo. Đóng Bè tre; sau này làm dây cung, bẫy.",
		sim.ItemPoisonMushroom: "Nấm tím đốm vàng: CÓ ĐỘC. Ăn vào là trúng độc. Sau này dùng tẩm tên độc.",
		sim.ItemNightshade:     "Quả đen bóng của bụi cà độc trong rừng: CÓ ĐỘC. Sau này dùng tẩm tên độc.",
		sim.ItemAloe:           "Cắt lá lô hội ở sa mạc. Bôi lên là hồi máu, dịu vết bỏng; cùng Thảo dược nấu Bình máu.",
		sim.ItemHerb:           "Hái trên đồng cỏ và trong rừng. Nhai là giải độc cóc, nhện, bọ cạp, nấm độc; nấu Bình máu thảo dược.",
		sim.ItemMoss:           "Gỡ rêu trên đá. Đắp vết thương thay băng (hồi máu, không giải độc); cùng Sợi lanh làm Băng rêu.",
		sim.ItemSeaweed:        "Vớt ở vùng nước nông ven biển. Ăn được.",
		sim.ItemLily:           "Hoa súng hái trên ao. Lá súng là chỗ cóc ngồi.",
		sim.ItemLotusSeed:      "Hạt trong đài sen trên ao. Ăn được, bùi và mát.",
		sim.ItemHorsetail:      "Cỏ đuôi ngựa mọc sát mép nước, thân ráp. Sau này làm thuốc và đánh bóng đồ.",
		sim.ItemRice:           "Tuốt từ lúa hoang ven ao. Gieo vào luống đất: 1 ngày sau gặt được 5. Rang 3 nắm cạnh lửa thành Cốm rang.",
		sim.ItemRoastRice:      "Cầm thóc (3 nắm), đứng gần lửa trại đang cháy, nhấp chuột phải để rang.",
		sim.ItemCorn:           "Bẻ ở ngô hoang trên đồng cỏ. Ăn sống được, nướng ngon hơn; gieo vào luống đất thì hơn 1 ngày sau bẻ được 3 bắp.",
		sim.ItemRoastCorn:      "Cầm ngô, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",
		sim.ItemSquash:         "Hái ở dây bí. Ăn sống được, nướng thì rất no.",
		sim.ItemRoastSquash:    "Cầm bí, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",
		sim.ItemSquashSeed:     "Lấy khi hái bí. Gieo vào luống đất: một ngày rưỡi sau được 2 quả bí và thêm hạt.",
		sim.ItemToadSkin:       "Da cóc độc xanh. Sau này dùng làm thuốc độc.",
		sim.ItemPoisonGland:    "Tuyến độc sau mắt cóc xanh. Sau này dùng tẩm tên độc.",
		sim.ItemGlowGland:      "Tuyến của cóc phát sáng: vẫn sáng lờ mờ trong đêm. Sau này làm đèn.",
	} {
		itemInfo[k] = s
	}
}
