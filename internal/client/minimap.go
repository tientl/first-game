package client

import (
	"math"
)

// The minimap, round in the top right corner in the ring of the day (see
// hud.go), shows the ground around the player with north up. It is cut
// from the same baked map and fog as the full map (M), so it only shows
// what the player has seen.
//
// It is laid out in its own pixels, miniPx to a tile as the map's pieces
// are baked (terrainmap.go), on a canvas miniW by miniH, and drawn miniZoom
// UI pixels to one of them: a little finer than the full map, so the small
// circle (miniD across, hudlayout.go) still shows a good way round.

const (
	miniZoom     = 0.8      // UI pixels to a minimap pixel
	miniW, miniH = 140, 140 // the minimap's canvas, in its own pixels: miniD / miniZoom
)

// miniTop is the screen y of the minimap's top, and miniBottom the y just
// below the corner it sits in, for what goes under it.
const miniTop = hudM + miniRing

func miniBottom() float64 { return cornerBottom(true, false) }

// miniScale is the minimap's size of one tile, in its own pixels.
func miniScale() (tw, th float64) {
	tw = miniPx // whole pixels, so the minimap's pieces are drawn one to one: see terrainmap.go
	return tw, tw * TileH / TileW
}

// miniOrigin is where map tile (0, 0) lands on the minimap's canvas when it
// is centred on (cx, cy), in tiles. It is whole pixels, so nothing shimmers.
func miniOrigin(cx, cy float64) (ox, oy float64) {
	tw, th := miniScale()
	return math.Round(miniW/2 - cx*tw), math.Round(miniH/2 - cy*th)
}
