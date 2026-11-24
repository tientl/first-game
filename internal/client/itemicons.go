package client

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The bag icons, redrawn as one set. Each is a 16x16 pixel map, placed by
// hand: a row of text per row of pixels, each character a shade of one of
// the icon's materials. All of them share the palettes below, are lit from
// the top left like the rest of the art, and get the same selective outline.
//
// In a map:
//
//	'.' or ' '  nothing
//	'0'..'4'    the first material, dark to light
//	'a'..'e'    the second
//	'A'..'E'    the third
//	'p'..'t'    the fourth
//	'P'..'T'    the fifth
//	'#'         ink
//	'+'         a glint

// The shared palettes, five shades each, dark to light.
var (
	iiWood   = mkRamp(0x3a2418, 0x5c3a22, 0x87592f, 0xb07d44, 0xd6a868)
	iiPale   = mkRamp(0x7a5530, 0xa67b45, 0xcfa468, 0xe8c88c, 0xf7e4b4)
	iiStone  = mkRamp(0x34323f, 0x55535f, 0x7c7a84, 0xa5a2a6, 0xd0cdc6)
	iiFlint  = mkRamp(0x2a2523, 0x494039, 0x6f6358, 0xa09282, 0xd8ccb6)
	iiJade   = mkRamp(0x0f3b34, 0x1c6b55, 0x2fa07a, 0x62cf9e, 0xb4f0c8)
	iiAgate  = mkRamp(0x5a1f14, 0xa23c1e, 0xdc6a32, 0xf29a5c, 0xfbd0a0)
	iiObsid  = mkRamp(0x15101f, 0x2a2140, 0x46386a, 0x7a68a8, 0xc0b4e8)
	iiHide   = mkRamp(0x55301c, 0x8a5230, 0xb7783f, 0xd9a25e, 0xf0cf94)
	iiBone   = mkRamp(0x6e5e50, 0xa39480, 0xd0c4a8, 0xece4cc, 0xfffaea)
	iiMeat   = mkRamp(0x6a1a28, 0xa82e3c, 0xd8545a, 0xf08a88, 0xfcc4b8)
	iiCooked = mkRamp(0x4a2010, 0x7c3a18, 0xb05c24, 0xd98a3c, 0xf4bc6c)
	iiChar   = mkRamp(0x1e120e, 0x3a2216, 0x5c3820, 0x845430, 0xb07c48)
	iiLeaf   = mkRamp(0x16361e, 0x2a5c2a, 0x4a8a34, 0x7cb648, 0xb8dc70)
	iiStraw  = mkRamp(0x5a4a1c, 0x8c7a2c, 0xbca844, 0xdccc6c, 0xf2e8a0)
	iiBerry  = mkRamp(0x5a1030, 0x941c40, 0xcc3050, 0xee6070, 0xffa8a8)
	iiRoast  = mkRamp(0x4a1420, 0x7c2430, 0xb03c40, 0xd86c50, 0xf8b080)
	iiFish   = mkRamp(0x1c3050, 0x2c5880, 0x4088b0, 0x74b8d4, 0xb8e4ec)
	iiFur    = mkRamp(0x3a2c28, 0x5c463c, 0x846858, 0xac8e78, 0xd0b8a0)
	iiScale  = mkRamp(0x1c3a2c, 0x2e5a3c, 0x4a7c4c, 0x74a060, 0xa4c480)
	iiFire   = mkRamp(0xa02810, 0xe05018, 0xf88820, 0xfcc040, 0xfff0a0)
	iiCoal   = mkRamp(0x14121c, 0x26242e, 0x3c3c4a, 0x5a5a6c, 0x9a9ab4)
	iiGold   = mkRamp(0x7a4a10, 0xb87818, 0xe4a828, 0xf8d048, 0xfff0a0)
	iiNut    = mkRamp(0x7a6430, 0xa89048, 0xd0bc70, 0xe8dca0, 0xfaf4d0)
	iiPurple = mkRamp(0x3a1838, 0x642858, 0x96407c, 0xc0689c, 0xe4a4c4)
	iiTuber  = mkRamp(0x4a1830, 0x7c2c48, 0xa8485c, 0xcc7078, 0xe8a8a0)
	iiRed    = mkRamp(0x6a1018, 0xa81c24, 0xdc3830, 0xf86c50, 0xffb090)
	iiBlue   = mkRamp(0x1c2a78, 0x2c48b8, 0x4c78e4, 0x84acf4, 0xc4dcfc)
	iiWhite  = mkRamp(0x7c7c90, 0xa8a8b8, 0xd0d0d8, 0xececf0, 0xffffff)
	iiPink   = mkRamp(0x7c2050, 0xb03c74, 0xdc6498, 0xf498b8, 0xfcd0dc)
	iiCyan   = mkRamp(0x0c4a58, 0x1488a0, 0x2cc4d0, 0x7ce8e4, 0xd4fcf4)
	iiShell  = mkRamp(0x6a4c44, 0xa07c6c, 0xccaa94, 0xe8d0bc, 0xfcf0e0)
	iiDung   = mkRamp(0x2e2010, 0x4a3418, 0x6c4c20, 0x8c6830, 0xac8848)
	iiGourd  = mkRamp(0x6a4a20, 0x9a7434, 0xc4a050, 0xe0c478, 0xf4e4a8)
	iiWorm   = mkRamp(0x7a3040, 0xb05060, 0xd87c84, 0xf0a8a4, 0xfcd4c8)
	iiCactus = mkRamp(0x1c4030, 0x2c6444, 0x4c9058, 0x84bc74, 0xc4e4a0)
	iiCrab   = mkRamp(0x2a3048, 0x445470, 0x6c80a0, 0x9cb0c4, 0xd0dce4)

	// iiFlintCore is flint freshly broken: darker and glassier than a worked edge.
	iiFlintCore = mkRamp(0x1c1a1e, 0x2e2a2c, 0x494440, 0x6f6660, 0xa39a8c)

	iiGlint = hex(0xfffdf0)
)

// iiTier is the stone each tier's tools are headed with, indexed by sim.Tier.
var iiTier = [sim.NumTiers]ramp{iiStone, iiFlint, iiJade, iiAgate, iiObsid}

// iiPut stamps a pixel map onto c with its top-left corner at (ox, oy).
func iiPut(c *canvas, ox, oy int, rs []ramp, rows ...string) {
	for y, row := range rows {
		if len(row) != len(rows[0]) {
			panic(fmt.Sprintf("itemicons: ragged map, row %d %q", y, row))
		}
		for x := 0; x < len(row); x++ {
			var col color.RGBA
			switch ch := row[x]; {
			case ch == '.' || ch == ' ':
				continue
			case ch == '#':
				col = colInk
			case ch == '+':
				col = iiGlint
			case ch >= '0' && ch <= '4':
				col = rs[0][ch-'0']
			case ch >= 'a' && ch <= 'e':
				col = rs[1][ch-'a']
			case ch >= 'A' && ch <= 'E':
				col = rs[2][ch-'A']
			case ch >= 'p' && ch <= 't':
				col = rs[3][ch-'p']
			case ch >= 'P' && ch <= 'T':
				col = rs[4][ch-'P']
			default:
				panic(fmt.Sprintf("itemicons: bad pixel %q in row %d %q", ch, y, row))
			}
			c.put(ox+x, oy+y, col)
		}
	}
}

// iiIcon is a finished icon of one map, 14 pixels wide at most, outlined.
func iiIcon(rs []ramp, rows ...string) *canvas {
	c := newCanvas(16, 16)
	iiPut(c, 1, 1+(14-len(rows))/2, rs, rows...)
	c.outline(0.8)
	return c
}

func iiRs(rs ...ramp) []ramp { return rs }

// --- tools --------------------------------------------------------------------

// The tools lie the same way: haft from the bottom left, head at the top
// right, the working edge towards the light.

func iiAxe(head ramp) *canvas {
	return iiIcon(iiRs(iiWood, head, iiHide),
		"....eddc......",
		"..eedddcc..31.",
		".eedddccbb31b.",
		".eeddccbbDBbb.",
		".eeddccbDBba..",
		".eedccb31a....",
		"..eedc31......",
		"...ee31.......",
		"....31........",
		"...31.........",
		"..31..........",
		".31...........",
		"31............")
}

func iiPick(head ramp) *canvas {
	return iiIcon(iiRs(iiWood, head, iiHide),
		"...deeddcc....",
		".dddcccccbbb..",
		".c......baDBb.",
		".........DBbcb",
		"........31..cb",
		".......31...cb",
		"......31....cb",
		".....31.....cb",
		"....31......ba",
		"...31.......ba",
		"..31........a.",
		".31...........",
		"31............")
}

func iiHoe(head ramp) *canvas {
	return iiIcon(iiRs(iiWood, head, iiHide),
		".......cdddc..",
		"......cDBddddc",
		"......DBbccddc",
		"......31..cddc",
		".....31...cdcb",
		".....31..cddcb",
		"....31...cdccb",
		"....31...deeed",
		"...31.........",
		"...31.........",
		"..31..........",
		"..31..........",
		".31...........",
		".31...........")
}

func iiKnife(blade ramp) *canvas {
	return iiIcon(iiRs(iiBone, blade, iiHide),
		"............de",
		"..........cdec",
		".........cdecb",
		"........cdecba",
		".......cdecba.",
		"......cdecba..",
		".....cdecb....",
		"....EDB.......",
		"...DCA........",
		"..EDB.........",
		".DCA..........",
		"343...........",
		"32............")
}

func iiSpear(tip, tassel ramp) *canvas {
	return iiIcon(iiRs(iiWood, tip, tassel),
		"............de",
		"..........cdeb",
		".........cdeb.",
		".........deba.",
		"........DBb...",
		".......DB.C...",
		"......31..CB..",
		".....31....B..",
		"....31........",
		"...31.........",
		"..31..........",
		".31...........",
		"31............")
}

// iiBattleAxe is the war axe: a double-bitted jade head on a long haft.
func iiBattleAxe() *canvas {
	return iiIcon(iiRs(iiWood, iiJade, iiHide),
		"......ed......",
		"....eddc....31",
		"...eddc....31.",
		"...edcc...31..",
		"..eddcbaDDB...",
		"..ed..baDB....",
		".......DBa..bd",
		"......DBaabbbd",
		".....31..bbbd.",
		"....31...bbbd.",
		"...31...bbbd..",
		"..31....bd....",
		".DB...........",
		"DB............")
}

func iiTorch() *canvas {
	return iiIcon(iiRs(iiWood, iiFire, iiHide),
		".......b......",
		"......bc......",
		"......bdc.....",
		".....bcdcb....",
		".....cdedcb...",
		"....bcdeedc...",
		"....bcdeedb...",
		".....bcddb....",
		".....DCDCB....",
		".....CBCBA....",
		"......32......",
		"......32......",
		"......31......",
		"......31......")
}

func iiRod() *canvas {
	c := newCanvas(16, 16)
	iiPut(c, 1, 1, iiRs(iiWood, iiBone, iiRed),
		"...........2..",
		"..........32..",
		".........32...",
		"........32....",
		".......32.....",
		"......32......",
		".....32.......",
		"....32.....CDC",
		"...32......+++",
		"..432.........",
		".CD2........d.",
		"CDB.......c.d.",
		"DB........cdc.")
	c.outline(0.8)
	// The line, drawn after the outline so it stays a hair.
	for y := 1; y <= 7; y++ {
		c.put(13, y, hex(0xcfd3dc))
	}
	c.put(13, 10, hex(0xcfd3dc))
	return c
}

// --- gear ---------------------------------------------------------------------

// iiTunic is the body every armour is drawn on.
func iiTunic(c *canvas, body ramp) {
	iiPut(c, 1, 2, iiRs(body),
		"..2332..2332..",
		".234431134432.",
		"23444321344432",
		"23344432443322",
		".223444444322.",
		"..2344444432..",
		"..2344444332..",
		"..2334443322..",
		"..2333433322..",
		"..1223332211..",
		"..1122222111..",
		"..11.11.11.1..")
}

func iiArmor(body ramp, over []ramp, rows ...string) *canvas {
	c := newCanvas(16, 16)
	iiTunic(c, body)
	if len(rows) > 0 {
		iiPut(c, 1, 2, over, rows...)
	}
	c.outline(0.8)
	return c
}

// --- flowers and seeds ----------------------------------------------------------

// iiFlower is a blossom, seven pixels square, on a leafy stem.
func iiFlower(rs []ramp, blossom ...string) *canvas {
	c := newCanvas(16, 16)
	iiPut(c, 1, 8, iiRs(iiLeaf),
		"......2.......",
		"..33..2..33...",
		".2343.2.3432..",
		"..232323232...",
		"......2.......",
		"......1.......")
	iiPut(c, 4, 1, rs, blossom...)
	c.outline(0.8)
	return c
}

// redrawItemIcons replaces the bag icon of every item (the trophy heads
// apart) with the set drawn here.
func redrawItemIcons(icons *[sim.NumItems]*ebiten.Image) {
	set := func(k sim.ItemKind, c *canvas) { icons[k] = c.image() }

	// --- raw materials ---
	set(sim.ItemWood, iiIcon(iiRs(iiWood, iiPale),
		"...cdc33333332",
		"..cdedc4434331",
		"..dcbcd3323221",
		"..cdedc2212110",
		"...cdc11111100",
		"..............",
		".cdc33333332..",
		"cdedc4434331..",
		"dcbcd3323221..",
		"cdedc2212110..",
		".cdc11111100.."))
	set(sim.ItemStone, iiIcon(iiRs(iiStone),
		"....234432....",
		"..2344444332..",
		".23444443322..",
		".23344433222..",
		"1233343222211.",
		"12232232221110",
		"12222122211100",
		"11221112111000",
		".111111101000.",
		"..1100000000.."))
	{
		c := newCanvas(16, 16)
		iiPut(c, 4, 2, iiRs(iiFlint), ".3443.", "344332", "233221", "122110", ".1100.")
		iiPut(c, 1, 8, iiRs(iiStone), ".344443.", "34444332", "33443322", "23332221", "12222110", ".111100.")
		iiPut(c, 10, 7, iiRs(iiShell), ".343.", "34432", "33321", "22211", ".110.")
		c.outline(0.8)
		set(sim.ItemPebble, c)
	}
	set(sim.ItemGrass, iiIcon(iiRs(iiStraw, iiHide),
		".2..3..4.3..2.",
		".23.3.34.3.32.",
		"..2334343332..",
		"..2334343432..",
		"...23434332...",
		"...12343432...",
		"....234343....",
		"....bcdcdc....",
		"....abcbcb....",
		"....134343....",
		"...12343432...",
		"..1223434322..",
		"..12.2343.21.."))
	set(sim.ItemStick, iiIcon(iiRs(iiWood, iiLeaf),
		"...........cd.",
		"....cd....3b..",
		"....bc...32...",
		"....3...32....",
		"....32.32.....",
		".....3321.....",
		".....321......",
		"....321.......",
		"...4321.......",
		"...321........",
		"..321.........",
		".321..........",
		".21..........."))
	set(sim.ItemCoal, iiIcon(iiRs(iiCoal),
		"....2332......",
		"..23443221....",
		".2344322110...",
		".233221211100.",
		"1232211121100.",
		"12211112332100",
		".1110123432110",
		"..001233221100",
		"...01222111000",
		"....0111100000",
		"......0000...."))

	// --- the stones of each tier: raw, then worked ---
	set(sim.ItemCopperOre, iiIcon(iiRs(iiFlintCore, iiBone),
		"...cddc.......",
		".cdeeedc33....",
		"cdeebeed3332..",
		"cdeeeed333322.",
		"cdeeed+4444321",
		"bdeed222222211",
		"bcdd2232222110",
		".bcd222322110.",
		"..bcc22211cb..",
		"...bbc111cb...",
		".....bbbb....."))
	set(sim.ItemIronOre, iiIcon(iiRs(iiStone, iiJade),
		"....233432....",
		"..23344443c2..",
		".233444bcdc22.",
		".2334bcdedb21.",
		"12333cdedcb211",
		"122bcdedcb2110",
		"12bcddcb221100",
		"11abccb2111000",
		".11ab11110000.",
		"..1100000000.."))
	set(sim.ItemPlatinumOre, iiIcon(iiRs(iiStone, iiAgate),
		"....23332.....",
		"..233444332...",
		".2344443bbb2..",
		".234443bcccb2.",
		"1233442bcdddcb",
		"122333bcde+edc",
		"12222abcdeeedc",
		"112221abcdddcb",
		".111111abcccb.",
		"..1100001bbb.."))
	set(sim.ItemDiamond, iiIcon(iiRs(iiObsid),
		"......2.......",
		".....243......",
		"....2443..2...",
		"....24332.32..",
		"...2433212432.",
		"...2+32112332.",
		"..243321123321",
		"..243211012321",
		".2432211012211",
		".2332110011110",
		".1221100.1100.",
		"..1100........"))
	set(sim.ItemCopperBar, iiIcon(iiRs(iiFlint, iiFlint),
		"...........de.",
		".........cdee.",
		".......bcdeed.",
		"......bcddedc.",
		"....abcdcedcb.",
		"...abcdcdedc..",
		"..abcdcdddcb..",
		".abccdcdccb...",
		".abbcdccbba...",
		"..aabccbaa....",
		"...aabbaa....."))
	set(sim.ItemIronBar, iiIcon(iiRs(iiJade, iiJade),
		"........cddc..",
		"......cdeeedc.",
		".....cde+eddc.",
		"....cdeeedddcb",
		"....cdeedddccb",
		"...cdeedddccb.",
		"...cdedddccb..",
		"..bcdddccbb...",
		"..bcddccbb....",
		".abcccbba.....",
		".abbbbaa......",
		"..aaaa........"))
	set(sim.ItemPlatinumBar, iiIcon(iiRs(iiAgate, iiAgate),
		"....bcccb.....",
		"..bcdddddcb...",
		".bcdeeeeedcb..",
		".cde+cccedcb..",
		"bcdecddcedcba.",
		"bcdecd+cedcba.",
		"bcdecccedccba.",
		".bcdeeeddcba..",
		".abcdddccba...",
		"...abbbbaa...."))

	// --- tools and weapons ---
	set(sim.ItemAxe, iiAxe(iiStone))
	set(sim.ItemPickaxe, iiPick(iiStone))
	set(sim.ItemHoe, iiHoe(iiStone))
	for i, k := range [...]sim.ItemKind{sim.ItemCopperAxe, sim.ItemIronAxe, sim.ItemPlatinumAxe, sim.ItemDiamondAxe} {
		set(k, iiAxe(iiTier[i+1]))
	}
	for i, k := range [...]sim.ItemKind{sim.ItemCopperPickaxe, sim.ItemIronPickaxe, sim.ItemPlatinumPickaxe, sim.ItemDiamondPickaxe} {
		set(k, iiPick(iiTier[i+1]))
	}
	for i, k := range [...]sim.ItemKind{sim.ItemCopperHoe, sim.ItemIronHoe, sim.ItemPlatinumHoe, sim.ItemDiamondHoe} {
		set(k, iiHoe(iiTier[i+1]))
	}
	for i, k := range [...]sim.ItemKind{sim.ItemCopperSword, sim.ItemIronSword, sim.ItemPlatinumSword, sim.ItemDiamondSword} {
		set(k, iiKnife(iiTier[i+1]))
	}
	set(sim.ItemSpear, iiSpear(iiStone, iiHide))
	set(sim.ItemIvorySpear, iiSpear(iiBone, iiFur))
	set(sim.ItemBattleAxe, iiBattleAxe())
	set(sim.ItemTorch, iiTorch())
	set(sim.ItemRod, iiRod())

	// --- armour and what is worn ---
	set(sim.ItemCopperArmor, iiArmor(iiHide, iiRs(iiBone, iiFur),
		"..............",
		"......bb......",
		"......0.......",
		".......0......",
		"......0.......",
		"..............",
		"..............",
		"..............",
		"..............",
		"..............",
		"..bcbbcbcbbc..",
		"..ab.ba.bc.a.."))
	set(sim.ItemIronArmor, iiArmor(iiHide, iiRs(iiBone),
		"..............",
		"..............",
		"..............",
		"..............",
		"...34444443...",
		"...22222222...",
		"...34444443...",
		"...22222222...",
		"...34444443...",
		"...22222222...",
		"....344443....",
		".............."))
	set(sim.ItemPlatinumArmor, iiArmor(iiHide, iiRs(iiAgate),
		"..............",
		"..............",
		".34........43.",
		".12........21.",
		"...34.34.34...",
		"...12.12.12...",
		"....34.34.3...",
		"....12.12.1...",
		"...34.34.34...",
		"...12.12.12...",
		"..............",
		".............."))
	set(sim.ItemDiamondArmor, iiArmor(iiChar, iiRs(iiBone, iiRed),
		".4..........4.",
		"..............",
		"..............",
		"...4......4...",
		"...3.4..4.3...",
		".....3443.....",
		"......33......",
		"..............",
		"..aaaaaaaaaa..",
		"..4.4.4.4.4...",
		"..............",
		".............."))
	set(sim.ItemFurCoat, iiArmor(iiFur, iiRs(iiBone, iiHide),
		"..4443..3444..",
		".4434....4344.",
		"..............",
		"..............",
		"......a.......",
		"......3.......",
		"......a.......",
		"......3.......",
		"......a.......",
		"..............",
		"..4343434343..",
		"..34.43.34.3.."))
	set(sim.ItemDinoArmor, iiArmor(iiScale, iiRs(iiScale, iiBone),
		"..............",
		"..............",
		"..............",
		"..1.1.1..1.1..",
		"...1.1.11.1...",
		"..1.1.1..1.1..",
		"...1.1.11.1...",
		"..1.1.1..1.1..",
		"...1.1.11.1...",
		"..............",
		"..d.d.d.d.d...",
		".............."))
	set(sim.ItemSaddle, iiIcon(iiRs(iiHide, iiBone, iiBone, iiChar),
		"..........rr..",
		".rr......rssr.",
		"qrsr....rssqp.",
		"qrssrrrrssqp..",
		".qrrssssrqp...",
		".2pqqqqqqp321.",
		"2344444444321.",
		"2344444443321.",
		"2333444433221.",
		".22333332221..",
		"...2C222211...",
		"...C.C........",
		"..C...C.......",
		"..CDDDC......."))

	// --- what the beasts give ---
	hide := []string{
		"22....33....22",
		"232..2442..232",
		".233234432332.",
		"..3344444433..",
		"...34444443...",
		"...34444432...",
		"..3344444432..",
		".233444443322.",
		"232.234432.232",
		"22...2332...22",
		"......22......",
	}
	set(sim.ItemHide, iiIcon(iiRs(iiHide), hide...))
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 2, iiRs(iiScale), hide...)
		iiPut(c, 1, 2, iiRs(iiScale),
			"..............",
			"..............",
			"......2.2.....",
			"....2.2.2.2...",
			".....2.2.2....",
			"....2.2.2.....",
			".....2.2.2.2..",
			"....2.2.2.2...",
			"......2.2.....",
			"..............",
			"..............")
		c.outline(0.8)
		set(sim.ItemDinoHide, c)
	}
	set(sim.ItemFur, iiIcon(iiRs(iiFur),
		"23...3223...32",
		"23323421243332",
		".3343421243433",
		".2343432134342",
		"..34343213434.",
		".2343432134342",
		".3343421243433",
		"23323421243332",
		"23...3223...32",
		"1.....22.....1"))
	set(sim.ItemBone, iiIcon(iiRs(iiBone),
		".........34...",
		"........344...",
		"........3443..",
		"........344443",
		".......342.332",
		"......342..22.",
		".22..342......",
		"233.342.......",
		"344432........",
		"..3432........",
		"...332........",
		"...22........."))
	set(sim.ItemRexTooth, iiIcon(iiRs(iiBone, iiHide),
		"......abbba...",
		".....abcccba..",
		".....bcdccb...",
		"....3444432...",
		"....3444431...",
		"...34444432...",
		"...3444431....",
		"...3444432....",
		"..34444321....",
		"..3444431.....",
		"..344432......",
		".34431........",
		".3432.........",
		".32..........."))
	set(sim.ItemIvory, iiIcon(iiRs(iiBone, iiHide),
		"...........ab.",
		"..........3443",
		"..........3443",
		".........34432",
		".........3443.",
		"........34432.",
		".......34432..",
		".....334432...",
		"...3344432....",
		".33444332.....",
		"3444332.......",
		".222.........."))
	{
		// A skein wound on a stick: the turns run across it.
		c := newCanvas(16, 16)
		iiPut(c, 1, 2, iiRs(iiWood, iiWhite),
			"...........32.",
			".......ddd32..",
			".....ddddddd..",
			"....ddddddddd.",
			"...ddddddddd..",
			"..ddddddddd...",
			"..dddddddd....",
			"..ddddddd.....",
			"..dddddd......",
			"..32ddd.......",
			".32...........")
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if c.at(x, y) == iiWhite[3] {
					i := []int{2, 3, 4, 3}[(x-y+16)%4]
					if x+y > 17 {
						i--
					}
					c.put(x, y, iiWhite[i])
				}
			}
		}
		c.outline(0.8)
		set(sim.ItemSilk, c)
	}
	set(sim.ItemDung, iiIcon(iiRs(iiDung, iiStraw),
		"......32....+.",
		".....3432..#..",
		"....234321....",
		"...23443321...",
		"...12222110...",
		"..2344433321..",
		".234443333221.",
		".122222221110.",
		"23444333343321",
		"2333333222221c",
		".111111111100."))

	// --- food ---
	set(sim.ItemMeat, iiIcon(iiRs(iiMeat, iiBone, iiBone),
		"...ddeeeedd...",
		"..d344444ddc..",
		".d34444443322.",
		".d344444433221",
		".d344d44332221",
		".c33d443322211",
		".c233333222110",
		"..c2222221110.",
		"...1111110D...",
		".....000...DDE",
		"...........EDD",
		"............E."))
	set(sim.ItemCookedMeat, iiIcon(iiRs(iiCooked, iiChar, iiBone),
		"...23333322...",
		"..2344444332..",
		".234a444a3322.",
		".23a444a433221",
		".2a444a43a2221",
		".2334a433a2211",
		".123a333a22110",
		"..12222221110.",
		"...1111110D...",
		".....000...DDE",
		"...........EDD",
		"............E."))
	fish := []string{
		".....12221....",
		"...2333333...2",
		".2344433333.21",
		"234#4233333221",
		"23444233333221",
		".cddddcccb..21",
		"...cccbb1....2",
		".....1b.......",
	}
	set(sim.ItemFish, iiIcon(iiRs(iiFish, iiWhite), fish...))
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 4, iiRs(iiCooked, iiGold), fish...)
		iiPut(c, 1, 4, iiRs(iiChar),
			"..............",
			"........1..1..",
			".......1..1...",
			"......1..1....",
			".....1..1.....",
			"..............",
			"..............",
			"..............")
		c.outline(0.8)
		set(sim.ItemCookedFish, c)
	}
	set(sim.ItemWorm, iiIcon(iiRs(iiWorm, iiWorm),
		".......2332...",
		"......234432..",
		"......232.232.",
		"..........232.",
		"...2332..2332.",
		"..234432e332..",
		".232.23ee32...",
		".232..2222....",
		".2332.........",
		"..232........."))
	{
		c := newCanvas(16, 16)
		for _, b := range [][2]int{{2, 5}, {9, 6}, {5, 10}} {
			iiPut(c, b[0], b[1], iiRs(iiBerry), ".232.", "24432", "23332", "12221", ".111.")
		}
		iiPut(c, 1, 1, iiRs(iiLeaf, iiLeaf, iiWood),
			".......B.bcd..",
			"......B.bcdec.",
			"......B.abcc..",
			".....BB..ab...")
		c.outline(0.8)
		set(sim.ItemBerry, c)
	}
	{
		c := newCanvas(16, 16)
		for i := 0; i < 13; i++ {
			c.put(1+i, 14-i, iiWood[3])
			c.put(2+i, 14-i, iiWood[1])
		}
		for _, b := range [][2]int{{2, 9}, {6, 5}, {10, 1}} {
			iiPut(c, b[0], b[1], iiRs(iiRoast), ".232.", "24432", "23332", "12221", ".111.")
		}
		c.outline(0.8)
		set(sim.ItemRoastBerry, c)
	}
	set(sim.ItemPotato, iiIcon(iiRs(iiTuber),
		"...........21.",
		".........2332.",
		".......234432.",
		".....23444332.",
		"...2344143321.",
		"..23444333221.",
		".23143332221..",
		".2333222211...",
		"1222211110....",
		"11111100......",
		".100.........."))
	set(sim.ItemRoastPotato, iiIcon(iiRs(iiChar, iiGold),
		"...........21.",
		".........2332.",
		".......2bcd32.",
		".....22cdedc2.",
		"...22bcdeedc1.",
		"..22bcdeedc21.",
		".22bcdddcb21..",
		".21bbcccb11...",
		"1221111110....",
		"11111100......",
		".100.........."))
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 1, iiRs(iiCactus, iiNut, iiPink),
			"......CD......",
			".....2BC2.....",
			"...23444332...",
			"..2344444332..",
			"..2344344332..",
			".234444443321.",
			".234344434321.",
			".234444443321.",
			".2344344ddc21.",
			"..23444deedc..",
			"..23344deedc..",
			"...23333ddc...",
			"....222211....")
		c.outline(0.8)
		for _, p := range [][2]int{{3, 2}, {1, 5}, {0, 8}, {2, 12}, {5, 14}, {12, 2}, {14, 5}, {15, 8}, {6, 5}, {10, 6}, {4, 8}, {8, 9}, {5, 11}} {
			c.put(p[0], p[1], iiNut[4])
		}
		set(sim.ItemCactus, c)
	}
	set(sim.ItemMushroom, iiIcon(iiRs(iiRed, iiBone),
		"....233332....",
		"..2344+44332..",
		".234444443+21.",
		".34+4444433221",
		"2344443+433221",
		"23333333332211",
		".11abbbbbba11.",
		".....cdcb.....",
		".....cdcb.....",
		"....cddcb.....",
		"....cddccb....",
		".....bbbb....."))
	set(sim.ItemClam, iiIcon(iiRs(iiShell, iiShell),
		"....cdddc.....",
		"..cdedededc...",
		".cdedededecc..",
		".dedededededc.",
		"cdecdedcdedcdb",
		"cdcdedcdedcdcb",
		"bcdcdcdcdcdcb.",
		".bcdcdcdcdcb..",
		"..bbcccccbb...",
		"...aabbbaa....",
		"....abbba....."))
	set(sim.ItemRoastClam, iiIcon(iiRs(iiCooked, iiShell),
		"...bcdddcb....",
		"..cdedededc...",
		".cdedededecc..",
		".cdddddddddc..",
		".b233443322b..",
		"cc234443322cc.",
		"cdc2333222cdc.",
		"bcdccccccdcb..",
		".bcdddddcb....",
		"..bbcccbb....."))
	set(sim.ItemCrabMeat, iiIcon(iiRs(iiCrab, iiWhite),
		".23........32.",
		"2342......2432",
		"232.#....#.232",
		".22.3....3.22.",
		"..2233443322..",
		".234444444432.",
		".234444443332.",
		".233344333322.",
		"2.2233332222.2",
		".2.11222211.2.",
		"2..1......1..2"))
	set(sim.ItemCookedCrab, iiIcon(iiRs(iiRed),
		".23........32.",
		"2342......2432",
		"232.#....#.232",
		".22.3....3.22.",
		"..2233443322..",
		".234444444432.",
		".234444443332.",
		".233344333322.",
		"2.2233332222.2",
		".2.11222211.2.",
		"2..1......1..2"))

	// --- fruit ---
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 1, iiRs(iiPurple, iiLeaf),
			".....cb...",
			"....2b....",
			"...2342...",
			"..234432..",
			".23444332.",
			".234443321",
			".234433221",
			".233332211",
			"..2222111.",
			"...11111..")
		iiPut(c, 8, 8, iiRs(iiPurple, iiPink, iiNut),
			"..232..",
			".2ded2.",
			"2dcEdc2",
			"2dEcEd2",
			"2dcdEd1",
			".2ddd1.",
			"..111..")
		c.outline(0.8)
		set(sim.ItemFig, c)
	}
	nut := []string{".343.", "34432", "23321", ".221."}
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 1, iiRs(iiWood, iiGold),
			"....cdd.ddc...",
			"...cdeedeedc..",
			"..cdedeededec.",
			"..cdededededc.",
			"...cdedededc..",
			"....cdeddc....",
			".....cdc......",
			"......c.......",
			"......b.......",
			".....b........")
		iiPut(c, 1, 10, iiRs(iiNut), nut...)
		iiPut(c, 9, 9, iiRs(iiNut), nut...)
		iiPut(c, 6, 11, iiRs(iiNut), nut...)
		c.outline(0.8)
		set(sim.ItemGinkgoNut, c)
	}
	{
		c := newCanvas(16, 16)
		roast := []string{".343.", "34a32", "23b21", ".221."}
		for _, p := range [][2]int{{5, 2}, {1, 6}, {9, 6}, {5, 10}} {
			iiPut(c, p[0], p[1], iiRs(iiCooked, iiLeaf), roast...)
		}
		c.outline(0.8)
		set(sim.ItemRoastGinkgo, c)
	}
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 1, iiRs(iiWood),
			"...2332...",
			".23444332.",
			".34430332.",
			"2304303321",
			"2344333221",
			"2333332221",
			"1233322211",
			".12222211.",
			".11111110.",
			"...1100...")
		iiPut(c, 7, 8, iiRs(iiWood, iiWhite),
			"..2332..",
			".2deed2.",
			"2de++ed2",
			"2deeedd1",
			"2dedddc1",
			".1dddc1.",
			"..1111..")
		c.outline(0.8)
		set(sim.ItemCoconut, c)
	}
	{
		// Two bananas on one stalk: crescents, the near one rimmed dark.
		c := newCanvas(16, 16)
		moon := func(ox, oy, grow float64, f func(x, y int, t float64) color.RGBA) {
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					px, py := float64(x)+0.5-ox, float64(y)+0.5-oy
					out := math.Hypot(px-9, py-3)
					in := math.Hypot(px-11, py-0.5)
					if out <= 8.6+grow && in >= 8.2-grow && px > 0.5-grow {
						c.put(x, y, f(x, y, (8.6-out)/3))
					}
				}
			}
		}
		skin := func(x, y int, t float64) color.RGBA { return iiGold.at(1 + int(t*4.2)) }
		moon(5, 0, 0, skin)
		moon(1, 2, 1, func(int, int, float64) color.RGBA { return iiGold[0] })
		moon(1, 2, 0, skin)
		iiPut(c, 2, 1, iiRs(iiWood), ".32.", ".21.")
		c.put(14, 9, iiWood[1])
		c.put(11, 13, iiWood[1])
		c.outline(0.8)
		set(sim.ItemBanana, c)
	}
	set(sim.ItemHoney, iiIcon(iiRs(iiGold, iiGold),
		"....cdddddc...",
		"..cdebddebddc.",
		".cddddddddddc.",
		".debddebddebd.",
		".dddddddddddd.",
		".cdebddebddec.",
		".cddddddddddc.",
		"..cebddebddc..",
		"...cdddddcde..",
		".........de...",
		".........de...",
		"..........+..."))

	// --- medicine ---
	set(sim.ItemPotion, iiIcon(iiRs(iiGourd, iiHide, iiWood, iiRed),
		".....AB.....",
		"....3443....",
		"....3432....",
		"...234432...",
		"...bcdcba...",
		"..23444432..",
		".2344444432.",
		".234rs44332.",
		".23rsss3322.",
		".233rr33221.",
		"..23333221..",
		"...222211..."))
	set(sim.ItemBigPotion, iiIcon(iiRs(iiHide, iiBone, iiWood, iiRed),
		"....rsssr.....",
		"...3rssssr3...",
		"...23333332...",
		"....bcdcba....",
		"..2334444332..",
		"32344444443323",
		"3.34rs444332.2",
		"2234rss4433212",
		".233rsr433221.",
		".223rr3332211.",
		"..2233332211..",
		"...11222110..."))
	set(sim.ItemBandage, iiIcon(iiRs(iiLeaf, iiWhite),
		"..........34..",
		"...cdddddd32c.",
		"..deeeeeee2ed.",
		".dcccdeeeeeeec",
		".dcbcdeeeeeedc",
		".dcccdeeedddcb",
		".cdddcdddccccb",
		"..cccbccccbbb.",
		".........dedc.",
		"..........ddc."))
	{
		c := newCanvas(16, 16)
		iiPut(c, 1, 5, iiRs(iiCyan), ".2332.", "234432", "24+432", "234432", "123321", ".1221.")
		iiPut(c, 8, 2, iiRs(iiPurple), ".343.", "34+43", "34443", "23432", ".232.")
		iiPut(c, 9, 10, iiRs(iiCyan), ".34.", "34+3", "2432", ".22.")
		c.outline(0.8)
		c.put(5, 2, iiCyan[4])
		c.put(7, 13, iiCyan[4])
		c.put(14, 8, iiPurple[4])
		set(sim.ItemGlowSpore, c)
	}

	// --- flowers ---
	set(sim.ItemFlowerRed, iiFlower(iiRs(iiRed),
		".33.33.",
		"3443443",
		"344#443",
		"23#+#32",
		"233#332",
		".22322.",
		"......."))
	daisy := []string{
		"..343..",
		".43434.",
		"343a343",
		"44aba44",
		"343a343",
		".43434.",
		"..343..",
	}
	set(sim.ItemFlowerYellow, iiFlower(iiRs(iiGold, iiCooked), daisy...))
	set(sim.ItemFlowerWhite, iiFlower(iiRs(iiWhite, iiGold), daisy...))
	set(sim.ItemFlowerBlue, iiFlower(iiRs(iiBlue, iiGold),
		"..343..",
		".34443.",
		"3343433",
		"44ede44",
		"3443443",
		".33.33.",
		"......."))
	set(sim.ItemFlowerPink, iiFlower(iiRs(iiPink, iiGold),
		".23432.",
		"2344332",
		"3432443",
		"342d243",
		"2344432",
		".22332.",
		"......."))
	set(sim.ItemFlowerPurple, iiIcon(iiRs(iiPurple, iiLeaf),
		"......3.......",
		".....343......",
		"......32......",
		".....343......",
		"....3432......",
		".....343......",
		"....3432......",
		".....232......",
		"......c.......",
		"..dd..c..dd...",
		".bcdc.c.cdcb..",
		"..bcbcbcbcb...",
		"......c.......",
		"......b......."))

	// --- seeds ---
	{
		c := newCanvas(16, 16)
		seed := []string{".34.", "3432", "2321", ".21."}
		iiPut(c, 2, 8, iiRs(iiNut), seed...)
		iiPut(c, 7, 10, iiRs(iiNut), seed...)
		iiPut(c, 10, 5, iiRs(iiNut), seed...)
		iiPut(c, 3, 2, iiRs(iiPink, iiGold), ".3.3.", "34a43", ".aba.", "34a43", ".3.3.")
		c.outline(0.8)
		set(sim.ItemFlowerSeed, c)
	}
	{
		c := newCanvas(16, 16)
		pip := []string{".3.", "342", "232", ".1."}
		for _, p := range [][2]int{{2, 8}, {6, 10}, {10, 8}, {11, 3}} {
			iiPut(c, p[0], p[1], iiRs(iiRoast), pip...)
		}
		iiPut(c, 3, 2, iiRs(iiBerry, iiLeaf), "..c..", ".232.", "24432", "12321", ".111.")
		c.outline(0.8)
		set(sim.ItemBerrySeed, c)
	}
	set(sim.ItemTreeSeed, iiIcon(iiRs(iiWood, iiLeaf, iiPale),
		".......dc.....",
		"......dcb.....",
		"......c.......",
		"....A1c1A.....",
		"...B1A1A1B....",
		"...23444332...",
		"..2344443321..",
		"..2344433221..",
		"..2334433221..",
		"...23332221...",
		"...1222211....",
		"....11110.....",
		"......0......."))
	{
		c := newCanvas(16, 16)
		seed := []string{".34.", "34+3", "2432", ".22."}
		iiPut(c, 2, 3, iiRs(iiCyan), seed...)
		iiPut(c, 9, 4, iiRs(iiCyan), seed...)
		iiPut(c, 5, 9, iiRs(iiCyan), seed...)
		iiPut(c, 10, 10, iiRs(iiPurple), ".3.", "343", ".2.")
		c.outline(0.8)
		c.put(7, 2, iiCyan[4])
		c.put(13, 9, iiCyan[4])
		c.put(3, 13, iiCyan[4])
		set(sim.ItemGlowSeed, c)
	}
	set(sim.ItemManEaterSeed, iiIcon(iiRs(iiPurple, iiBone, iiRed, iiLeaf),
		"......r.r.....",
		".....rsrsr....",
		"....23343 2...",
		"...2344443 2..",
		"..234443332...",
		"..e3e3e3e3e2..",
		"..1e1e1e1e11..",
		"..BCBCBCBCB1..",
		"..1d1d1d1d11..",
		"..d2d2d2d2d1..",
		"...2333222 1..",
		"....222211....",
		".....1110....."))

	// --- boats ---
	set(sim.ItemRaft, iiIcon(iiRs(iiWood, iiPale, iiStraw),
		"..23E3333E3cd.",
		"..11C1111C1bc.",
		"2333E3333E3cd.",
		"1111C1111C1bc.",
		".233E3334E33cd",
		".111C1111C11bc",
		"2333E3333Ecd..",
		"1111C1111Cbc..",
		"..23E4333E33cd",
		"..11C1111C11bc"))
	set(sim.ItemSailboat, iiIcon(iiRs(iiWood, iiBone, iiFish),
		"......1.......",
		"...3333333....",
		"...cdeeedc....",
		"...cdeeedc....",
		"..cdeeeeedc...",
		"..cdddddddc...",
		"..cdeeeeedc...",
		".cdeeeeeeedc..",
		".bccccccccb...",
		"......1.......",
		"13....1.....31",
		".13333333331..",
		"..122222221...",
		"D.CD..DC.D.C.."))
}
