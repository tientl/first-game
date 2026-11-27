package client

import (
	"math"
)

// The minimap, under the clock in the top right corner, shows the ground
// around the player with north up. It is cut from the same baked map and
// fog as the full map (M), so it only shows what the player has seen.

const (
	miniW, miniH = 188, 141 // the minimap's inside, in screen pixels: 4:3 like a tile
	miniTiles    = 36       // tiles across the minimap
	miniPad      = 6
	miniTop      = 70 // the map's frame, just under the clock in the same block; see hud.go
)

// miniBottom is the screen y just below the minimap, for what goes under it.
func miniBottom() float64 { return miniTop + miniH + 2*miniPad }

// miniScale is the minimap's size of one tile, in screen pixels.
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
