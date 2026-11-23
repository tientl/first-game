package client

// TEMPORARY stand-in for the desert art being drawn separately; replaced on merge.

type desertArt struct {
	pyramid  sprite
	scorpion [scorpionPoses]sprite
}

const (
	scorpionWalkA = iota
	scorpionWalkB
	scorpionPinch
	scorpionSting
	scorpionPoses
)

func genDesertArt() desertArt {
	var a desertArt
	c := newCanvas(168, 170)
	c.polygon([]float64{0, 170, 168, 170, 84, 0}, solid(hex(0xd8b070)))
	a.pyramid = mk(c, 84, 170)
	for p := range a.scorpion {
		sc := newCanvas(44, 20)
		sc.rect(0, 0, 44, 20, solid(hex(0x4a2a10)))
		a.scorpion[p] = mk(sc, 22, 20)
	}
	return a
}
