package client

// A warmer ground: ochre earth with grass growing over it in patches, the
// edges of each patch ragged with blades, a scatter of pebbles and cracks in
// the bare earth, all in the warm light of a late afternoon. It is one of
// the two looks of the ground (SetGroundStyle, in groundhd.go); warmEarth in
// groundart.go draws it.

var (
	warmDirtRamp  = mkRamp(0x5a3a22, 0x74492a, 0x8e5c33, 0xa8733f, 0xbf8a4c, 0xd4a35f)
	warmGrassRamp = mkRamp(0x3b4e22, 0x4e6229, 0x627631, 0x77893a, 0x8d9b46, 0xa4ad57)
	warmMossRamp  = mkRamp(0x2c4220, 0x3a5426, 0x4b672c, 0x5e7b34, 0x74903f)
	warmStone     = mkRamp(0x5e5650, 0x7a706a, 0x968a80, 0xb2a698)
)
