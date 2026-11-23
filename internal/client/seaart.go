package client

// TEMPORARY stand-in for the sea art being drawn separately; replaced on merge.

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

type seaArt struct {
	raftBack, raftFront sprite
	sailBack, sailFront sprite
	sail                [3]sprite
	mosaRise            sprite
	mosaBite            sprite
	mosaBack            sprite
	mosaShadow          sprite
	krakenArm           [3]sprite
	krakenHead          sprite
	krakenShadow        sprite
}

const (
	raftSeatX, raftSeatY = 0, -2
	sailSeatX, sailSeatY = -12, -4
	mastX, mastY         = 4, -5
)

func blk(w, h int, ax, ay float64, col color.RGBA) sprite {
	c := newCanvas(w, h)
	c.rect(0, 0, w, h, solid(col))
	return mk(c, ax, ay)
}

func genSeaArt() seaArt {
	var a seaArt
	a.raftBack, a.raftFront = blk(36, 6, 18, 4, woodRamp.at(1)), blk(36, 4, 18, 2, woodRamp.at(0))
	a.sailBack, a.sailFront = blk(60, 8, 30, 5, woodRamp.at(1)), blk(60, 5, 30, 2, woodRamp.at(0))
	for i := range a.sail {
		a.sail[i] = blk(18+i*2, 32, 2, 32, hex(0xd8c8a0))
	}
	a.mosaRise, a.mosaBite = blk(14, 30, 7, 30, hex(0x406070)), blk(18, 30, 9, 30, hex(0x406070))
	a.mosaBack = blk(30, 4, 15, 4, hex(0x406070))
	a.mosaShadow = blk(90, 16, 45, 8, hex(0x102030))
	for i := range a.krakenArm {
		a.krakenArm[i] = blk(6, 30+i*3, 3, 30+float64(i*3), hex(0x9a3050))
	}
	a.krakenHead, a.krakenShadow = blk(20, 14, 10, 14, hex(0x9a3050)), blk(60, 40, 30, 20, hex(0x200810))
	return a
}

func genSeaIcons(icons *[sim.NumItems]*ebiten.Image) {
	r := newCanvas(16, 16)
	r.rect(1, 7, 14, 5, solid(woodRamp.at(1)))
	icons[sim.ItemRaft] = r.image()
	s := newCanvas(16, 16)
	s.rect(1, 11, 14, 3, solid(woodRamp.at(1)))
	s.rect(7, 2, 6, 9, solid(hex(0xd8c8a0)))
	icons[sim.ItemSailboat] = s.image()
}
