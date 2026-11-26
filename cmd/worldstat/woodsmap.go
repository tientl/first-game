package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"survivalgame/internal/sim"
)

// writeWoodsMap writes a map of the woods of w in the tiles [x0, x1) ×
// [y0, y1), px pixels to a tile: the thick woods ("rừng dày") dark green,
// ordinary woods mid green, in between shaded; a dot for each tree (darker
// for a grown one, lighter for a young one), saplings and stumps small and
// brown; trails, grass, sand, rock and water in plain colours; the spawn a
// red cross.
func writeWoodsMap(w *sim.World, path string, x0, y0, x1, y1, px int) error {
	W, _ := w.Size()
	H := w.SurfaceHeight()
	x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, W), min(y1, H)
	tiles := w.Tiles()
	thick := w.DevWoodsThickness()
	img := image.NewRGBA(image.Rect(0, 0, (x1-x0)*px, (y1-y0)*px))
	fill := func(tx, ty, ox, oy, s int, c color.RGBA) {
		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				img.SetRGBA((tx-x0)*px+ox+x, (ty-y0)*px+oy+y, c)
			}
		}
	}
	mix := func(a, b color.RGBA, t float64) color.RGBA {
		l := func(p, q uint8) uint8 { return uint8(float64(p) + (float64(q)-float64(p))*t) }
		return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
	}
	ordinary, dense := color.RGBA{96, 150, 70, 255}, color.RGBA{28, 70, 34, 255}
	for ty := y0; ty < y1; ty++ {
		for tx := x0; tx < x1; tx++ {
			i := ty*W + tx
			var c color.RGBA
			switch tiles[i] {
			case sim.TileForest:
				c = mix(ordinary, dense, float64(thick[i]))
			case sim.TileGrass:
				c = color.RGBA{170, 200, 110, 255}
			case sim.TilePath:
				c = color.RGBA{170, 130, 80, 255}
			case sim.TileSand, sim.TileDesert:
				c = color.RGBA{225, 210, 160, 255}
			case sim.TileMarsh:
				c = color.RGBA{110, 120, 80, 255}
			case sim.TileSnow:
				c = mix(color.RGBA{235, 240, 245, 255}, color.RGBA{150, 170, 160, 255}, float64(thick[i]))
			case sim.TileWater, sim.TileShallow, sim.TileRiver:
				c = color.RGBA{70, 120, 180, 255}
			default:
				c = color.RGBA{130, 125, 120, 255}
			}
			fill(tx, ty, 0, 0, px, c)
		}
	}
	w.EachEntityIn(x0, y0, x1-1, y1-1, func(e sim.Entity) {
		s := max(1, px/2)
		o := (px - s) / 2
		switch {
		case e.Kind == sim.EntityTree && e.Ready && e.Big, e.Kind == sim.EntityGiantTree:
			fill(e.TX, e.TY, o, o, s, color.RGBA{10, 30, 12, 255})
		case e.Kind == sim.EntityTree && e.Ready:
			fill(e.TX, e.TY, o, o, s, color.RGBA{40, 110, 40, 255})
		case e.Kind == sim.EntityTree, e.Kind == sim.EntitySapling:
			fill(e.TX, e.TY, o, o, max(1, s/2), color.RGBA{120, 80, 40, 255})
		}
	})
	sx, sy := w.Spawn()
	for d := -3; d <= 3; d++ {
		for _, p := range [][2]int{{int(sx)*px - x0*px + d, int(sy)*px - y0*px}, {int(sx)*px - x0*px, int(sy)*px - y0*px + d}} {
			if image.Pt(p[0], p[1]).In(img.Rect) {
				img.SetRGBA(p[0], p[1], color.RGBA{230, 30, 30, 255})
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
