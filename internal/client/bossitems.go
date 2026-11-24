package client

// The bag icons of the six legendary items, one from each boss. They are
// drawn like the other icons (see itemicons.go): a pixel map of 14 columns
// at most, outlined. Each carries gold and a glint, which nothing ordinary
// in the bag does.

var (
	iiTitan = mkRamp(0x0e2a16, 0x1a4622, 0x2c6832, 0x468c42, 0x70b45a)
	iiKing  = mkRamp(0x120d10, 0x24181a, 0x3c2826, 0x5a3e34, 0x7c5a48)
	iiEmber = mkRamp(0x7a1a0c, 0xb8300f, 0xe85a14, 0xff8f1e, 0xffc440)
	iiRust  = mkRamp(0x4a140c, 0x742010, 0x9c3416, 0xbe4c20, 0xd96c30)
	iiShark = mkRamp(0x182634, 0x2c4458, 0x4a687c, 0x7090a4, 0xa4c0cc)
)

func bossItemIcon(b bossID) *canvas {
	switch b {
	case bossTitanoboa:
		// Giáp vảy Titanoboa: a coat of green scales, collared and hemmed in gold.
		return iiArmor(iiTitan, iiRs(iiGold, iiTitan),
			"..3.......3...",
			".34.43..34.43.",
			"....3433433...",
			"..a...34...a..",
			".a.a.a.+a.a.a.",
			"..a.a.aa.a.a..",
			"...a.a..a.a...",
			"..a.a.aa.a.a..",
			"...a.a..a.a...",
			"..2343443432..",
			"..1.2.11.2.1..",
			"..............")
	case bossSpino:
		// Giáo răng Spinosaurus: a great tooth lashed to a haft with gold.
		return iiIcon(iiRs(iiWood, iiBone, iiGold, iiRed),
			".............e",
			"...........dee",
			".........cd+ec",
			"........cdeedb",
			".......bcdedb.",
			"......CDEcdb..",
			"......BCDD....",
			".....31BCq....",
			"....31...rq...",
			"...31.....q...",
			"..31..........",
			".31...........",
			"31............")
	case bossMammothKing:
		// Áo lông Voi Chúa: black fur, a collar of frost, a gold clasp.
		return iiArmor(iiKing, iiRs(iiWhite, iiGold),
			"..2332..2332..",
			".234432234432.",
			"....234432....",
			"......de......",
			"......cd......",
			"..............",
			"..............",
			"..............",
			"..............",
			"..2323232323..",
			"..3232323232..",
			"..2.32.23.2...")
	case bossCarcharo:
		// Kiếm răng cưa Carcharo: a blade set with teeth, hilted in gold.
		return iiIcon(iiRs(iiBone, iiGold, iiRust),
			"...........234",
			"..........2341",
			".........23432",
			"........2341..",
			".......23+32..",
			"......2341....",
			".....23432....",
			"..d.2341......",
			"..cd3432......",
			"...cd1........",
			"..CBcd........",
			".CB..c........",
			"de............")
	case bossGiganoto:
		// Rìu nanh Giganoto: a fang for an axe-head, still smouldering.
		return iiIcon(iiRs(iiWood, iiBone, iiGold, iiEmber),
			"....eedc......",
			"..eeddddc..31.",
			".ee+ddccbb31..",
			".edddccbCDCb..",
			".eddrccCDCa...",
			".edsrcb31a....",
			"..edsr31......",
			"..eds31.......",
			"..ed31........",
			"..e31.........",
			"..31..........",
			".BD...........",
			"DB............")
	}
	// Vòng răng Megalodon: shark's teeth on a cord, gold between them.
	return iiIcon(iiRs(iiHide, iiBone, iiGold, iiShark),
		"...11111111...",
		"..1........1..",
		".1..........1.",
		".1..........1.",
		".1..........1.",
		".D1........1D.",
		".dd1......1dd.",
		".cdD1....1Ddc.",
		"..c.dD11Dd.c..",
		"...ddeeeedd...",
		"....cd+edc....",
		".....cddc.....",
		"......cd......",
		"......c.......")
}
