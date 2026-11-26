package client

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The world map (M) shows the whole island. It is baked once per map, off the
// game loop, from the same coast and biome fields as the ground so its shapes
// match what the player walks on. On top go the player, the other players in
// view and the campfires this client has seen.

const (
	mapPxPerTile = 8    // baked pixels per tile across, before the cap
	mapMaxPx     = 1024 // longest side of the baked image
	fireForget   = 16   // tiles: a remembered fire this close but not in the snapshot is gone
)

var (
	mapWater = ramp{colShallow, mix(colShallow, colWaterBG, 0.5), colWaterBG, mix(colWaterBG, colDeep, 0.5), colDeep}
	colFire  = color.RGBA{255, 176, 64, 255}
)

// mapTiles is every land tile kind the map draws, in legend order, with its
// colours and its name in the legend. A new tile kind goes here, and only
// here, to show on the map. Water is not a land tile: the coast decides it.
var mapTiles = []struct {
	tile  sim.Tile
	ramp  ramp
	label string
}{
	{sim.TileGrass, grassRamp, "Cỏ"},
	{sim.TileForest, forestRamp, "Rừng"},
	{sim.TileSand, sandRamp, "Cát"},
	{sim.TileRocky, rockRamp, "Đá"},
	{sim.TileDesert, desertRamp, "Hoang mạc"},
	{sim.TileMarsh, marshRamp, "Đầm lầy"},
	{sim.TileMountain, mtnRamp, "Núi"},
	{sim.TileCave, caveRamp, "Hang"},
	{sim.TileAsh, ashRamp2, "Tro núi lửa"},
	{sim.TileLava, lavaRamp, "Dung nham"},
	{sim.TileSnow, snowRamp, "Tuyết"},
	{sim.TilePath, pathRamp, "Đường mòn"},
	{sim.TileShallow, mkRamp(0x4ec0c6), "Biển nông"}, // the new waters and cliffs: see terrainfeatures.go
	{sim.TileRiver, mkRamp(0x3f8fc0), "Sông, hồ"},
	{sim.TileFord, mkRamp(0x6ab4ca), "Chỗ lội"},
	{sim.TileWaterfall, mkRamp(0xeaf7fa), "Thác"},
	{sim.TileCliff, mkRamp(0x3a2a22), "Vách đá"},
}

// mapRamp is the palette the map paints land tile t with. Kinds missing from
// mapTiles (say, from a newer server) are painted as grass, never crash.
func mapRamp(t sim.Tile) ramp {
	for _, m := range mapTiles {
		if m.tile == t {
			return m.ramp
		}
	}
	return grassRamp
}

type mapResult struct {
	gen  int
	w, h int
	data []byte
}

// worldMap holds the baked map image, or is waiting for it, and the fog
// over the parts the player has not seen.
type worldMap struct {
	gen     int
	img     *ebiten.Image
	results chan mapResult

	explored *sim.Explored   // what this player has seen, kept up to date
	fog      *ebiten.Image   // one pixel per tile, opaque where unseen
	fogFirst chan []byte     // the first fog, made off the game loop: see mapperf.go
	fogDirty image.Rectangle // tiles whose fog may be stale; empty when none
	mini     *ebiten.Image   // the minimap's canvas

	present [sim.NumTiles]bool // tile kinds this world has, for the legend
}

// reset forgets the old image and starts baking terr in the background.
// explored is what the player had seen, from the server.
func (m *worldMap) reset(terr *terrain, explored *sim.Explored) {
	m.explored, m.fogDirty = explored, image.Rectangle{}
	clear(m.present[:])
	for _, t := range terr.tiles {
		if t < sim.NumTiles {
			m.present[t] = true
		}
	}
	if m.fog != nil {
		m.fog.Deallocate()
		m.fog = nil
	}
	m.fogFirst = nil
	m.gen++
	if m.img != nil {
		m.img.Deallocate()
		m.img = nil
	}
	if m.results == nil {
		m.results = make(chan mapResult, 4)
	}
	gen, out := m.gen, m.results
	go func() {
		w, h, data := terr.bakeMap()
		out <- mapResult{gen, w, h, data}
	}()
}

// collect turns a finished bake into an image. It must run on the game loop.
func (m *worldMap) collect() {
	for {
		select {
		case r := <-m.results:
			if r.gen != m.gen {
				continue
			}
			m.img = ebiten.NewImage(r.w, r.h)
			m.img.WritePixels(r.data)
		default:
			return
		}
	}
}

// bakeMap draws the whole map, keeping the TileW:TileH shape of a tile,
// its rows shared out among the cores.
func (t *terrain) bakeMap() (w, h int, data []byte) {
	sx := mapScale(t.w, t.h)
	sy := sx * TileH / TileW
	w, h = max(1, int(float64(t.w)*sx)), max(1, int(float64(t.h)*sy))
	c := newCanvas(w, h)
	inRows(h, func(y int) {
		for x := 0; x < w; x++ {
			c.px[y*w+x] = t.mapPixel((float64(x)+0.5)/sx, (float64(y)+0.5)/sy, x, y)
		}
	})
	return w, h, c.bytes()
}

// mapPixel colours the map at (u, v) in tiles; (px, py) only drives dithering.
func (t *terrain) mapPixel(u, v float64, px, py int) color.RGBA {
	s := t.coast(u, v)
	if s < 0 {
		if s > -0.12 {
			return mix(colFoam, colShallow, 0.35)
		}
		return t.mapShallow(mapWater.pick(math.Sqrt(clamp01(-s/5)), px, py), u, v)
	}
	best, second, margin := t.landType(u, v)
	c := mapRamp(best).pick(0.5+(fbm2(u*0.25, v*0.3, 90, 2)-0.5)*1.2, px, py)
	if margin < 0.04 && second != best {
		c = mix(c, colInk, 0.25) // inked border between biomes
	}
	if s < 0.12 {
		c = mix(cliffRamp.at(1), c, s/0.12*0.6) // inked coastline
	}
	return t.mapRelief(c, u, v)
}

// --- fog of war -------------------------------------------------------------

var (
	colFog     = hex(0x16141f)
	colFogLite = hex(0x24202f)
)

// fogAlpha is how thick the fog is over tile (tx, ty): none on seen ground,
// thin next to it and thicker one tile out, so with linear scaling the
// explored land fades out softly at its edge.
func fogAlpha(e *sim.Explored, tx, ty int) uint8 {
	if e.Has(tx, ty) {
		return 0
	}
	near := 3 // Chebyshev distance to the nearest seen tile, capped
	for dy := -2; dy <= 2 && near > 1; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if e.Has(tx+dx, ty+dy) {
				near = min(near, max(dx, -dx, dy, -dy))
			}
		}
	}
	return [4]uint8{0, 130, 205, 255}[near]
}

// fogPad is how far past the revealed circle the fog can change: the
// fading edge reaches two tiles out.
const fogPad = sim.RevealRadius + 3

// markSeen notes that the ground around tile (tx, ty) was just revealed.
func (m *worldMap) markSeen(tx, ty int) {
	r := image.Rect(tx-fogPad, ty-fogPad, tx+fogPad+1, ty+fogPad+1)
	m.fogDirty = m.fogDirty.Union(r)
}

// fogImage returns the fog, one pixel per tile. It is built whole once, then
// only the part around newly seen ground is repainted, which keeps it cheap
// on big maps.
func (m *worldMap) fogImage() *ebiten.Image {
	e := m.explored
	w, h := e.Size()
	if w == 0 || h == 0 {
		return nil
	}
	all := image.Rect(0, 0, w, h)
	if m.fog == nil && !m.firstFog(w, h) {
		return nil // still being made: see mapperf.go
	}
	r := m.fogDirty.Intersect(all)
	m.fogDirty = image.Rectangle{}
	if r.Empty() {
		return m.fog
	}
	m.fog.SubImage(r).(*ebiten.Image).WritePixels(fogPixels(e, r))
	return m.fog
}

// mapSnapshot keeps the map's memory up to date with a new snapshot: the
// ground around the player is now seen, and the campfires in view are noted.
func (g *Game) mapSnapshot(s protocol.Snapshot) {
	for _, p := range s.Players {
		x, y := float64(p.X), float64(p.Y)
		if p.ID == g.myID && g.wmap.explored.Reveal(x, y) {
			g.wmap.markSeen(int(math.Floor(x)), int(math.Floor(y)))
		}
	}
	g.rememberFires(s)
}

// --- remembered campfires ---------------------------------------------------

// knownFire is a campfire this client has seen. The server only sends what is
// near the player, so the map remembers fires once they go out of view.
type knownFire struct {
	tx, ty int16
	seenAt uint64 // tick of the last snapshot that held it
	outAt  uint64 // tick it burns out unless someone adds wood
}

func (g *Game) rememberFires(s protocol.Snapshot) {
	if g.fires == nil {
		g.fires = make(map[sim.EntityID]knownFire)
	}
	for _, e := range s.Entities {
		if e.Kind == sim.EntityCampfire {
			g.fires[e.ID] = knownFire{e.TX, e.TY, s.Tick, s.Tick + uint64(max(0, e.Fuel))}
		}
	}
	px, py, ok := 0.0, 0.0, false
	for _, p := range s.Players {
		if p.ID == g.myID {
			px, py, ok = float64(p.X), float64(p.Y), true
		}
	}
	for id, f := range g.fires {
		near := ok && math.Abs(float64(f.tx)+0.5-px) <= fireForget && math.Abs(float64(f.ty)+0.5-py) <= fireForget
		if s.Tick >= f.outAt || (near && f.seenAt != s.Tick) {
			delete(g.fires, id)
		}
	}
}

// --- drawing ----------------------------------------------------------------

// maxMapZoom is how far the world map zooms in.
const maxMapZoom = 6

func (g *Game) drawMap(screen *ebiten.Image, sw, sh int) {
	full := screen
	g.dim(screen, sw, sh, 0.65)
	g.text(screen, "Bản đồ", float64(sw)/2, 18, colWhite, text.AlignCenter, 2)

	// Fit the map between the title and the legend, keeping its shape.
	const top, bottom, side = 60.0, 74.0, 28.0
	// With caves under the islands, the map shows the layer the player is
	// in: the islands, or the caves.
	r0, rh := g.mapRows()
	mw, mh := float64(g.width*TileW), rh*TileH
	k := min((float64(sw)-2*side)/mw, (float64(sh)-top-bottom)/mh)
	dw, dh := mw*k, mh*k
	x0, y0 := math.Round((float64(sw)-dw)/2), math.Round(top+(float64(sh)-top-bottom-dh)/2)
	panel(screen, float32(x0)-8, float32(y0)-8, float32(dw)+16, float32(dh)+16)
	legendY := y0 + dh + 22

	// Zoomed in (the wheel, = and -), the map is bigger than its frame and
	// centred on the player; everything is drawn clipped to the frame.
	z := float64(max(1, g.mapZoom))
	if z > 1 {
		px, py := float64(g.width)/2, rh/2
		for _, p := range g.curr.Players {
			if p.ID == g.myID {
				px, py = float64(p.X), float64(p.Y)-r0
			}
		}
		fw, fh := dw, dh
		dw, dh = fw*z, fh*z
		vx0, vy0 := x0, y0
		x0 = math.Round(min(vx0, max(vx0+fw-dw, vx0+fw/2-px/float64(g.width)*dw)))
		y0 = math.Round(min(vy0, max(vy0+fh-dh, vy0+fh/2-py/rh*dh)))
		screen = screen.SubImage(image.Rect(int(vx0), int(vy0), int(vx0+fw), int(vy0+fh))).(*ebiten.Image)
	}

	fog := g.mapLayer(g.wmap.fogImage(), r0, rh)
	if img := g.mapLayer(g.wmap.img, r0, rh); img != nil && fog != nil {
		b := img.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(dw/float64(b.Dx()), dh/float64(b.Dy()))
		op.GeoM.Translate(x0, y0)
		if dw < float64(b.Dx()) {
			op.Filter = ebiten.FilterLinear // shrinking: blend instead of dropping pixels
		}
		screen.DrawImage(img, op)
	} else {
		vector.FillRect(screen, float32(x0), float32(y0), float32(dw), float32(dh), colWaterBG, false)
		g.text(screen, "Đang vẽ bản đồ…", float64(sw)/2, y0+dh/2-8, colWhite, text.AlignCenter, 1)
	}
	// Only what the player has seen shows; the rest is under fog.
	if fog != nil {
		b := fog.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(dw/float64(b.Dx()), dh/float64(b.Dy()))
		op.GeoM.Translate(x0, y0)
		op.Filter = ebiten.FilterLinear // soft edges where seen land fades out
		screen.DrawImage(fog, op)
	}

	at := func(x, y float64) (float32, float32) {
		return float32(x0 + x/float64(g.width)*dw), float32(y0 + (y-r0)/rh*dh)
	}
	pulse := float32(g.frame%60) / 60

	for _, f := range g.fires {
		fx, fy := at(float64(f.tx)+0.5, float64(f.ty)+0.5)
		flick := float32(math.Sin(float64(g.frame)*0.3+float64(f.tx))) * 0.8
		vector.FillCircle(screen, fx, fy, 8+flick, withAlpha(colFire, 70), true)
		drawFireMark(screen, fx, fy)
	}

	g.drawBossMarks(screen, at, r0) // see bossui.go

	alpha := g.alpha()
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			continue
		}
		wx, wy, _ := g.lerpPlayer(p, alpha)
		x, y := at(wx, wy)
		g.drawHeadMark(screen, x, y, 1.5, playerColors[int(p.ID)%len(playerColors)], p.Look)
		g.text(screen, p.Name, float64(x), float64(y)-30, colWhite, text.AlignCenter, 1)
	}
	for _, p := range g.curr.Players {
		if p.ID != g.myID {
			continue
		}
		wx, wy, _ := g.lerpPlayer(p, alpha)
		x, y := at(wx, wy)
		vector.StrokeCircle(screen, x, y, 16+pulse*14, 2, color.RGBA{255, 255, 255, uint8(200 * (1 - pulse))}, true)
		g.drawHeadMark(screen, x, y, 2, playerColors[int(p.ID)%len(playerColors)], p.Look)
	}

	g.drawMapLegend(full, float64(sw)/2, legendY)
	g.text(full, fmt.Sprintf("M / Esc  đóng bản đồ · lăn chuột hoặc = / −  phóng to, thu nhỏ (×%d)", max(1, g.mapZoom)), float64(sw)/2, float64(sh)-26, colMuted, text.AlignCenter, 1)
}

// drawHeadMark draws a player on the map: their head (with look l) on a disc
// of their shirt colour, centred on (x, y), at scale times the art size.
func (g *Game) drawHeadMark(dst *ebiten.Image, x, y float32, scale float64, c color.RGBA, l sim.Look) {
	r := float32(8 * scale)
	vector.FillCircle(dst, x, y, r+1.5, colInk, true)
	vector.FillCircle(dst, x, y, r, c, true)
	head := g.art.body(l).head
	b := head.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(math.Round(float64(x)), math.Round(float64(y)))
	dst.DrawImage(head, op)
}

func drawFireMark(dst *ebiten.Image, x, y float32) {
	vector.FillCircle(dst, x, y, 5, colInk, true)
	vector.FillCircle(dst, x, y, 4, colFire, true)
	vector.FillCircle(dst, x, y+1, 2, color.RGBA{255, 240, 170, 255}, true)
}

// drawMapLegend draws one centred row explaining the marks and colours.
func (g *Game) drawMapLegend(screen *ebiten.Image, cx, y float64) {
	type entry struct {
		label string
		draw  func(x, y float32)
	}
	swatch := func(c color.RGBA) func(x, y float32) {
		return func(x, y float32) {
			fillRoundRect(screen, x-6, y-6, 12, 12, 3, colInk)
			fillRoundRect(screen, x-5, y-5, 10, 10, 2, c)
		}
	}
	me := playerColors[int(g.myID)%len(playerColors)]
	entries := []entry{
		{"Bạn ở đây", func(x, y float32) { g.drawHeadMark(screen, x, y, 1, me, g.myLook()) }},
		{"Lửa trại", func(x, y float32) { drawFireMark(screen, x, y) }},
	}
	for _, m := range mapTiles {
		if g.wmap.present[m.tile] { // only what this world has
			entries = append(entries, entry{m.label, swatch(m.ramp[len(m.ramp)/2])})
		}
	}
	entries = append(entries,
		entry{"Nước", swatch(colWaterBG)},
		entry{"Chưa đi tới", swatch(colFogLite)})
	const mark, gap = 18.0, 22.0
	width := func(e entry) float64 { return mark + g.textWidth(e.label) + gap }
	maxW := math.Max(240, 2*cx-80)
	var rows [][]entry
	rowW := 0.0
	for _, e := range entries {
		if len(rows) == 0 || rowW+width(e)-gap > maxW {
			rows = append(rows, nil)
			rowW = 0
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], e)
		rowW += width(e)
	}
	for r, row := range rows {
		total := -gap
		for _, e := range row {
			total += width(e)
		}
		x, ry := cx-total/2, y+float64(r)*20
		for _, e := range row {
			e.draw(float32(x+6), float32(ry+8))
			g.text(screen, e.label, x+mark, ry, colWhite, text.AlignStart, 1)
			x += width(e)
		}
	}
}

// mapRows is the band of map rows the map shows: the islands, or with the
// player underground, the caves.
func (g *Game) mapRows() (r0, rows float64) {
	if g.surfaceH <= 0 {
		return 0, float64(g.height)
	}
	for _, p := range g.curr.Players {
		if p.ID == g.myID && int(p.Y) >= g.surfaceH {
			top := g.surfaceH + sim.UndergroundGap
			return float64(top), float64(min(g.surfaceH, g.height-top))
		}
	}
	return 0, float64(g.surfaceH)
}

// mapLayer cuts rows r0 to r0+rows of a whole-map image.
func (g *Game) mapLayer(img *ebiten.Image, r0, rows float64) *ebiten.Image {
	if img == nil {
		return nil
	}
	if r0 == 0 && int(rows) == g.height {
		return img
	}
	b := img.Bounds()
	k := float64(b.Dy()) / float64(g.height)
	return img.SubImage(image.Rect(b.Min.X, b.Min.Y+int(r0*k), b.Max.X, b.Min.Y+int((r0+rows)*k))).(*ebiten.Image)
}
