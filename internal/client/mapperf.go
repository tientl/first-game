package client

import (
	"image"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Keeping the map (M) and the minimap smooth on big worlds: a world of 2048
// tiles across has eight and a half million tiles, caves and all. The map
// is baked on every core at a size that keeps detail at 1080p; the fog's
// first full image is made off the game loop (after that only the ground
// round newly seen tiles is repainted); and the ground the season changes
// is worked out off the loop too.

// bigGround is the most tiles a map can have for its ground to be worked
// out again on the game loop when the season turns.
const bigGround = 1 << 21

// mapImagePx is the longest the baked map's width may be, in pixels, and
// mapImageMax the most either side may be.
const (
	mapImagePx  = 2048
	mapImageMax = 4000
)

// mapScale is how many map pixels a tile is across in the baked map of a
// world w by h tiles (h counting the caves).
func mapScale(w, h int) float64 {
	return min(mapPxPerTile, float64(mapImagePx)/float64(w), float64(mapImageMax)/(float64(h)*TileH/TileW))
}

// inRows calls fn for every row from 0 to h-1, shared out among the cores.
func inRows(h int, fn func(y int)) {
	n := max(1, min(runtime.GOMAXPROCS(0), h))
	var wg sync.WaitGroup
	for k := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := k; y < h; y += n {
				fn(y)
			}
		}()
	}
	wg.Wait()
}

// fogPixels is the fog over the tiles of r, one pixel a tile.
func fogPixels(e *sim.Explored, r image.Rectangle) []byte {
	pix := make([]byte, r.Dx()*r.Dy()*4)
	inRows(r.Dy(), func(y int) {
		ty := r.Min.Y + y
		for x := range r.Dx() {
			tx := r.Min.X + x
			a := uint32(fogAlpha(e, tx, ty))
			c := mix(colFog, colFogLite, hashf(tx, ty, 500))
			i := (y*r.Dx() + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = uint8(uint32(c.R)*a/255), uint8(uint32(c.G)*a/255), uint8(uint32(c.B)*a/255), uint8(a)
		}
	})
	return pix
}

// firstFog makes the fog image the first time it is wanted. Its pixels are
// worked out off the game loop, from a copy of what was seen; until they
// are, there is no image and the map and the minimap wait. What is seen
// meanwhile is in fogDirty, painted over it once it is up. It reports
// whether the image is up.
func (m *worldMap) firstFog(w, h int) bool {
	if m.fogFirst == nil {
		ch := make(chan []byte, 1)
		m.fogFirst = ch
		seen := sim.ExploredFrom(w, h, m.explored.Bytes())
		go func() { ch <- fogPixels(seen, image.Rect(0, 0, w, h)) }()
		return false
	}
	select {
	case pix := <-m.fogFirst:
		m.fog = ebiten.NewImage(w, h)
		m.fog.WritePixels(pix)
		return true
	default:
		return false
	}
}

// seasonGround is the ground for a new season, being worked out off the
// game loop.
type seasonGround struct {
	season sim.Season
	ready  chan *terrain
}

// startSeasonGround begins working out the ground as season s shows it.
func (g *Game) startSeasonGround(s sim.Season) {
	ch := make(chan *terrain, 1)
	g.seasonNext = &seasonGround{season: s, ready: ch}
	w, h, tiles, pond, relief, under := g.width, g.height, g.tiles, g.pondMask, g.relief, g.caveRow()
	go func() {
		terr := newTerrain(w, h, seasonTiles(w, h, tiles, s, pond))
		terr.setRelief(relief)
		terr.under = under
		ch <- terr
	}()
}

// takeSeasonGround puts the new season's ground in place once it is ready.
func (g *Game) takeSeasonGround() {
	if g.seasonNext == nil {
		return
	}
	select {
	case terr := <-g.seasonNext.ready:
		g.ground.reset(terr)
		g.seasonNext = nil
	default:
	}
}
