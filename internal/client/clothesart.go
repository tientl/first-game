package client

import (
	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The bag icons and notes of the clothes (sim/clothes.go): a linen tunic, a
// fur mantle, trousers with leg wraps, and boots. How they look when worn
// is tallfig.go's and limbs.go's.

var iiLinen = mkRamp(0x5e6a78, 0x8794a2, 0xaeb9c4, 0xd2dae2, 0xf0f4f8)

func clothesIcons(icons *[sim.NumItems]*ebiten.Image) {
	icons[sim.ItemTunic] = iiArmor(iiLinen, iiRs(iiLinen, iiHide, iiGold),
		"..............",
		"......a.......",
		"......a.......",
		"..............",
		"..............",
		"..............",
		"..bcccAAcccb..",
		"..abbbAAbbba..",
		"........c.....",
		"........b.....",
		"..............",
		"..............").image()
	icons[sim.ItemMantle] = iiIcon(iiRs(iiFur, iiGold),
		"...23....32...",
		".234432234432.",
		"23443344334432",
		"34432.dd.23443",
		"2332..cc..2332",
		"232........232",
		".21........12.",
		".1..........1.").image()
	icons[sim.ItemLegWraps] = iiIcon(iiRs(iiChar, iiPale),
		"..3333333333..",
		"..2344334432..",
		"..234432 432..",
		"..2343..3432..",
		"..2332..2332..",
		"..2332..2332..",
		"..cbdc..cbdc..",
		"..bdcb..bdcb..",
		"..dcbd..dcbd..",
		"..cbdc..cbdc..",
		"..bdcb..bdcb..").image()
	icons[sim.ItemBoots] = iiIcon(iiRs(iiChar, iiFur),
		"..cdcdc.......",
		".bcdcdcb......",
		"..23443.......",
		"..23443.......",
		"..234432......",
		"..2344432.....",
		"..23444433322.",
		"..23344444432.",
		"..11111111111.").image()
}

func init() {
	itemInfo[sim.ItemTunic] = "Mặc khi có trong túi: áo vải lanh thắt đai (nữ: váy dài), nhuộm màu của bạn. Chỉ để đẹp; giáp mặc đè lên trên."
	itemInfo[sim.ItemMantle] = "Mặc khi có trong túi: khăn lông choàng vai, cài trâm. Chỉ để đẹp."
	itemInfo[sim.ItemLegWraps] = "Mặc khi có trong túi: quần và dây da quấn ống chân. Chỉ để đẹp."
	itemInfo[sim.ItemBoots] = "Mặc khi có trong túi: giày da viền lông. Không có thì đi chân đất."
}
