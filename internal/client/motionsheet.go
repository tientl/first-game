package client

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The motion sheets: every motion of the player played through the game's
// own drawing (drawPlayer, drawRider, drawSwimmer, drawSleeper), frame by
// frame on a scripted clock, and written as a sheet of frames and as GIFs.
// For a look at the motions as they are in the game:
//
//	go run ./cmd/shot -w 1920 -h 1080 -motionsheet dir
//
// It must run inside the game loop (Draw), where images can be drawn.

// reel is a scene of the player for the sheets: n frames at 60 a second,
// set up by at before each one (it may move the figure, which stands at
// (x, y) in art pixels of a cell).
type reel struct {
	name string
	n    int
	dirs []sim.Dir
	at   func(g *Game, t int, p *protocol.PlayerView, d *drawable)
	bg   color.RGBA
	draw func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) // what to draw instead of drawPlayer
}

// A cell of the sheets, in art pixels, and where the feet stand in it.
const (
	cellW, cellH = 84, 64
	cellX, cellY = 42, 56
)

var (
	reelGrass = color.RGBA{92, 132, 74, 255}
	reelWater = color.RGBA{52, 112, 140, 255}
	reelFloor = color.RGBA{122, 104, 80, 255}
	allDirs   = []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight, sim.DirLeft}
)

// sheetID is the player the sheets show: someone other than me, so nothing
// of my own (the hover, the ailments) is drawn on them.
const sheetID sim.PlayerID = 7

// render draws one frame of r facing dir at time t into an image.
func (g *Game) render(r reel, dir sim.Dir, t int, buf *fineBuf) *image.RGBA {
	buf.img.Fill(r.bg)
	p := protocol.PlayerView{ID: sheetID, Dir: dir, Look: sim.Look{Hair: sim.HairShaggy, HairColor: 2, Skin: 1}, X: 40, Y: 40}
	d := drawable{sx: cellX, sy: cellY, id: uint32(sheetID), isPlayer: true}
	g.frame = 100000 + t
	if r.at != nil {
		r.at(g, t, &p, &d)
	}
	d.wx, d.wy = float64(p.X), float64(p.Y)
	align(buf, d.sx, d.sy)
	if r.draw != nil {
		r.draw(g, buf, p, d)
	} else {
		g.drawPlayer(buf, p, d)
	}
	b := buf.img.Bounds()
	img := image.NewRGBA(b)
	buf.img.ReadPixels(img.Pix)
	return img
}

// MotionSheets writes the motion sheets and GIFs into dir.
func (g *Game) MotionSheets(dir string) error {
	keepFrame := g.frame
	defer func() { g.frame = keepFrame }()
	buf := newFineBuf(cellW, cellH)
	reels := g.motionReels()
	// The sheet: every reel in every facing a row, a frame every so often.
	const cols = 12
	cw, ch := cellW*HDDensity, cellH*HDDensity
	rows := 0
	for _, r := range reels {
		rows += len(r.dirs)
	}
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cw, rows*ch))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{40, 44, 40, 255}}, image.Point{}, draw.Src)
	row := 0
	for _, r := range reels {
		for _, d := range r.dirs {
			g.resetMotions()
			step := max(1, r.n/cols)
			for t := 0; t < r.n && t/step < cols; t++ { // every frame drawn, for the clocks that count them
				img := g.render(r, d, t, buf)
				if t%step == 0 {
					i := t / step
					draw.Draw(sheet, image.Rect(i*cw, row*ch, (i+1)*cw, (row+1)*ch), img, image.Point{}, draw.Src)
				}
			}
			label(sheet, 4, row*ch+4, r.name)
			row++
		}
	}
	if err := sheetPNG(filepath.Join(dir, "motion_sheet.png"), sheet); err != nil {
		return err
	}
	if err := g.compareStrips(filepath.Join(dir, "compare")); err != nil {
		return err
	}
	// The GIFs: reels side by side, every facing, playing together.
	gifs := map[string][]string{}
	for _, r := range reels {
		gifs[gifOf(r.name)] = append(gifs[gifOf(r.name)], r.name)
	}
	names := make([]string, 0, len(gifs))
	for n := range gifs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		var group []reel
		for _, r := range reels {
			if gifOf(r.name) == name {
				group = append(group, r)
			}
		}
		if err := g.writeReelGIF(filepath.Join(dir, name), group, buf); err != nil {
			return err
		}
	}
	return nil
}

// resetMotions forgets what every player was doing, between reels.
func (g *Game) resetMotions() {
	g.moves, g.gaits, g.swingAt, g.swingWhat, g.hurtAt = nil, nil, nil, nil, nil
	g.heldAge = 0
}

// writeReelGIF writes reels as one GIF: a row for each, a column for each
// facing, at 30 frames a second.
func (g *Game) writeReelGIF(path string, reels []reel, buf *fineBuf) error {
	cw, ch := cellW*HDDensity, cellH*HDDensity
	n := 0
	for _, r := range reels {
		n = max(n, r.n)
	}
	frames := make([]*image.RGBA, 0, n/2+1)
	for t := 0; t < n; t += 2 {
		frames = append(frames, image.NewRGBA(image.Rect(0, 0, 4*cw, len(reels)*ch)))
	}
	for ri, r := range reels {
		for di, d := range r.dirs {
			g.resetMotions()
			for t := 0; t < n; t++ {
				img := g.render(r, d, t%r.n, buf)
				if t%2 != 0 {
					continue
				}
				k := t / 2
				draw.Draw(frames[k], image.Rect(di*cw, ri*ch, (di+1)*cw, (ri+1)*ch), img, image.Point{}, draw.Src)
				if di == 0 {
					label(frames[k], 4, ri*ch+4, r.name)
				}
			}
		}
	}
	return sheetGIF(path, frames, 3)
}

// gifOf is the GIF reel name goes in.
func gifOf(name string) string {
	switch name {
	case "walk", "run":
		return "motion_walk_run.gif"
	case "chop", "slash", "backslash", "thrust", "punch", "cleave", "shoot", "throw", "guard":
		return "motion_attacks.gif"
	case "idle", "sit by the fire":
		return "motion_idle_sit.gif"
	case "gather", "reach up", "scoop":
		return "motion_gather_eat.gif"
	case "eat meat", "eat berries", "eat soup", "drink", "drink at water":
		return "motion_eat_drink.gif"
	case "hurt", "die":
		return "motion_hurt_death.gif"
	case "swim", "tread water", "wade in", "ford":
		return "motion_swim.gif"
	case "sleep", "sleep on the ground":
		return "motion_sleep.gif"
	}
	return "motion_ride.gif"
}

// motionReels is every scene of the sheets.
func (g *Game) motionReels() []reel {
	side := []sim.Dir{sim.DirRight, sim.DirLeft}
	still := func(held sim.ItemKind) func(*Game, int, *protocol.PlayerView, *drawable) {
		return func(g *Game, t int, p *protocol.PlayerView, d *drawable) { p.Held = held }
	}
	walking := func(speed float64) func(*Game, int, *protocol.PlayerView, *drawable) {
		return func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			dx, dy := p.Dir.Delta()
			along := float32(speed * float64(t) / TileW)
			p.X += float32(dx) * along
			p.Y += float32(dy) * along
			d.moving = true
		}
	}
	// blow is a blow begun at frame 0 with held, at what w says, again
	// every period frames; struck is how many blows went before.
	blow := func(held sim.ItemKind, w swingInfo, period, before int) func(*Game, int, *protocol.PlayerView, *drawable) {
		return func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Held = held
			start := g.frame - t%period
			p.Acting = g.frame-start < sim.ActingTicks*3
			if g.swingAt == nil {
				g.swingAt, g.swingWhat = map[sim.PlayerID]int{}, map[sim.PlayerID]swingInfo{}
			}
			g.swingAt[p.ID], g.swingWhat[p.ID] = start, w
			st := g.moveState(p.ID)
			st.blows = before + t/period
			st.blowAt, st.movedAt = start, g.frame-1000
		}
	}
	deed := func(dd sim.Deed, k sim.ItemKind, period int) func(*Game, int, *protocol.PlayerView, *drawable) {
		return func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			st := g.moveState(p.ID)
			st.deed, st.deedItem, st.deedAt = dd, k, g.frame-t%period
			st.movedAt, st.blowAt = g.frame-30, never
		}
	}
	reels := []reel{
		{name: "walk", n: 48, dirs: allDirs, at: walking(0.9)},
		{name: "run", n: 48, dirs: allDirs, at: walking(1.6)},
		{name: "idle", n: 140, dirs: allDirs, at: still(sim.ItemNone)},
		{name: "guard", n: 48, dirs: allDirs, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Held = sim.ItemCopperSword
			st := g.moveState(p.ID)
			st.blowAt, st.movedAt = g.frame-30, g.frame-30
		}},
		{name: "chop", n: 36, dirs: allDirs, at: blow(sim.ItemAxe, swingInfo{kind: swingAxe}, 36, 0)},
		{name: "slash", n: 36, dirs: allDirs, at: blow(sim.ItemCopperSword, swingInfo{kind: swingHand, beast: true}, 36, 1)},
		{name: "backslash", n: 36, dirs: allDirs, at: blow(sim.ItemCopperSword, swingInfo{kind: swingHand, beast: true}, 36, 0)},
		{name: "thrust", n: 30, dirs: allDirs, at: blow(sim.ItemSpear, swingInfo{kind: swingThrust}, 30, 0)},
		{name: "cleave", n: 44, dirs: allDirs, at: blow(sim.ItemBattleAxe, swingInfo{kind: swingCleave}, 44, 0)},
		{name: "shoot", n: int(moShoot.span()), dirs: allDirs, draw: drawShootReel},
		{name: "throw", n: 40, dirs: allDirs, at: deed(sim.DeedToss, sim.ItemStone, 40)},
		{name: "hurt", n: 30, dirs: allDirs, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			if g.hurtAt == nil {
				g.hurtAt = map[sim.PlayerID]int{}
			}
			g.hurtAt[p.ID] = g.frame - t%30
		}},
		{name: "die", n: int(moDie.span()) + 20, dirs: allDirs, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Dead = true
			st := g.moveState(p.ID)
			st.deadAt, st.deadDir = g.frame-t, p.Dir
		}, draw: func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
			if !g.drawDying(dst, p, d) { // down: as the body the server leaves lies
				g.drawLaidBody(dst, p, d)
			}
		}},
		{name: "gather", n: int(moGather.span()) + 6, dirs: allDirs, at: blow(sim.ItemNone, swingInfo{kind: swingHand}, int(moGather.span())+6, 0)},
		{name: "reach up", n: int(moReachUp.span()) + 6, dirs: allDirs, at: blow(sim.ItemNone, swingInfo{kind: swingHand, high: true}, int(moReachUp.span())+6, 0)},
		{name: "scoop", n: int(moScoop.span()) + 10, dirs: allDirs, at: deed(sim.DeedScoop, sim.ItemSeaWater, int(moScoop.span())+10), bg: reelGrass},
		{name: "eat meat", n: int(moEatBite.span()) + 8, dirs: allDirs, at: deed(sim.DeedEat, sim.ItemCookedMeat, int(moEatBite.span())+8)},
		{name: "eat berries", n: int(moEatHand.span()) + 8, dirs: allDirs, at: deed(sim.DeedEat, sim.ItemBerry, int(moEatHand.span())+8)},
		{name: "eat soup", n: int(moEatBowl.span()) + 8, dirs: allDirs, at: deed(sim.DeedEat, sim.ItemStew, int(moEatBowl.span())+8)},
		{name: "drink", n: int(moDrink.span()) + 8, dirs: allDirs, at: deed(sim.DeedEat, sim.ItemCleanWater, int(moDrink.span())+8)},
		{name: "drink at water", n: int(moDrinkWater.span()) + 8, dirs: allDirs, draw: drawMotionReel(moDrinkWater)},
		{name: "sit by the fire", n: 300, dirs: allDirs, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			st := g.moveState(p.ID)
			st.movedAt, st.blowAt = g.frame-sitAfter-1-t, never
			if t < 250 {
				st.sitAt = g.frame - t
			} else if t == 250 {
				st.sitAt, st.standAt = never, g.frame
			}
			g.sheetFire(p)
		}},
	}
	reels = append(reels, g.waterReels()...)
	reels = append(reels, g.sleepReels()...)
	reels = append(reels, g.rideReels()...)
	for i := range reels {
		if reels[i].bg == (color.RGBA{}) {
			reels[i].bg = reelGrass
		}
		if len(reels[i].dirs) == 0 {
			reels[i].dirs = side
		}
	}
	return reels
}

// sheetFire puts a burning campfire by p, for sitting by.
func (g *Game) sheetFire(p *protocol.PlayerView) {
	for _, e := range g.curr.Entities {
		if e.ID == 1<<30+77 {
			return
		}
	}
	g.curr.Entities = append(g.curr.Entities, protocol.EntityView{ID: 1<<30 + 77, Kind: sim.EntityCampfire,
		TX: int16(p.X) + 1, TY: int16(p.Y), Fuel: sim.FireStartFuel})
}

// drawMotionReel draws motion m as it plays, on its own clock, with nothing
// in hand: for motions nothing in the game sets off yet.
func drawMotionReel(m motionID) func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
	return func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
		lo, up := m.keys(p.Dir&3, m.frameAt(float64(g.frame-100000)))
		drawFig(dst, figDraw{b: g.art.bodyIn(p.Look, p.Wear), up: up, lo: lo, tint: playerColors[int(p.ID)%len(playerColors)]}, d.sx, d.sy)
	}
}

// drawShootReel draws LPC's shoot with a bow drawn in code: there is no bow
// in the game yet.
func drawShootReel(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
	fr := moShoot.frameAt(float64(g.frame - 100000))
	lo, up := moShoot.keys(p.Dir&3, fr)
	f := figDraw{b: g.art.bodyIn(p.Look, p.Wear), up: up, lo: lo, tint: playerColors[int(p.ID)%len(playerColors)]}
	draw := [13]float64{0, 0, 0, 0.1, 0.6, 1, 1, 1, 1, 0, 0, 0, 0}[fr]
	f.pair = func(dst drawTarget, hx, hy, ox, oy float64) {
		drawBow(dst, p.Dir&3, ox, oy, hx, hy, draw, fr >= 3 && fr < 9)
	}
	drawFig(dst, f, d.sx, d.sy)
}

// drawBow draws a bow held at (bx, by), its string drawn back draw of the
// way to (hx, hy), with an arrow on it if nocked.
func drawBow(dst drawTarget, dir sim.Dir, bx, by, hx, hy, draw float64, nocked bool) {
	wood := color.RGBA{120, 80, 44, 255}
	str := color.RGBA{230, 226, 210, 255}
	side := facingSide(dir)
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	const half = 9.0
	// The bow's limbs: an arc through the grip, bent back at the tips.
	tip := func(s float64) (float64, float64) {
		if !sideways {
			return bx + s*half*0.9, by + 1.5
		}
		return bx - side*2.2*(1+draw*0.4)*s*s, by + s*half
	}
	px, py := tip(-1)
	for s := -1.0; s <= 1.01; s += 0.1 {
		x, y := tip(s)
		strokeLine(dst, float32(px), float32(py), float32(x), float32(y), 1, wood, false)
		px, py = x, y
	}
	ax, ay := tip(-1)
	cx, cy := tip(1)
	nx, ny := lerpf(bx, hx, draw), lerpf(by, hy, draw)
	if draw < 0.05 {
		nx, ny = (ax+cx)/2, (ay+cy)/2
	}
	strokeLine(dst, float32(ax), float32(ay), float32(nx), float32(ny), 0.5, str, false)
	strokeLine(dst, float32(nx), float32(ny), float32(cx), float32(cy), 0.5, str, false)
	if nocked {
		ex, ey := bx+side*4, by
		if !sideways {
			ex, ey = bx, by-4
		}
		strokeLine(dst, float32(nx), float32(ny), float32(ex), float32(ey), 0.5, color.RGBA{200, 190, 170, 255}, false)
	}
}

// label writes s small at (x, y) on img, in a plain 3x5 hand.
func label(img *image.RGBA, x, y int, s string) {
	for i, r := range s {
		glyph(img, x+i*8, y, r)
	}
}

func glyph(img *image.RGBA, x, y int, r rune) {
	g, ok := font3x5[r]
	if !ok {
		return
	}
	for row := 0; row < 5; row++ {
		for col := 0; col < 3; col++ {
			if g[row]>>(2-col)&1 == 1 {
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						img.SetRGBA(x+col*2+dx, y+row*2+dy, color.RGBA{255, 255, 255, 255})
					}
				}
			}
		}
	}
}

var font3x5 = map[rune][5]uint8{
	'a': {2, 5, 7, 5, 5}, 'b': {6, 5, 6, 5, 6}, 'c': {3, 4, 4, 4, 3}, 'd': {6, 5, 5, 5, 6},
	'e': {7, 4, 6, 4, 7}, 'f': {7, 4, 6, 4, 4}, 'g': {3, 4, 5, 5, 3}, 'h': {5, 5, 7, 5, 5},
	'i': {7, 2, 2, 2, 7}, 'j': {1, 1, 1, 5, 2}, 'k': {5, 5, 6, 5, 5}, 'l': {4, 4, 4, 4, 7},
	'm': {5, 7, 7, 5, 5}, 'n': {6, 5, 5, 5, 5}, 'o': {2, 5, 5, 5, 2}, 'p': {6, 5, 6, 4, 4},
	'q': {2, 5, 5, 6, 3}, 'r': {6, 5, 6, 5, 5}, 's': {3, 4, 2, 1, 6}, 't': {7, 2, 2, 2, 2},
	'u': {5, 5, 5, 5, 7}, 'v': {5, 5, 5, 5, 2}, 'w': {5, 5, 7, 7, 5}, 'x': {5, 5, 2, 5, 5},
	'y': {5, 5, 2, 2, 2}, 'z': {7, 1, 2, 4, 7}, '(': {1, 2, 2, 2, 1}, ')': {4, 2, 2, 2, 4},
}

func sheetPNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// sheetGIF writes frames as a GIF, delay hundredths of a second apiece,
// on one palette of their commonest colours.
func sheetGIF(path string, frames []*image.RGBA, delay int) error {
	count := map[color.RGBA]int{}
	for _, f := range frames {
		for i := 0; i < len(f.Pix); i += 16 {
			count[color.RGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}]++
		}
	}
	cols := make([]color.RGBA, 0, len(count))
	for c := range count {
		cols = append(cols, c)
	}
	sort.Slice(cols, func(i, j int) bool {
		if count[cols[i]] != count[cols[j]] {
			return count[cols[i]] > count[cols[j]]
		}
		a, b := cols[i], cols[j]
		return int(a.R)<<16|int(a.G)<<8|int(a.B) < int(b.R)<<16|int(b.G)<<8|int(b.B)
	})
	var pal color.Palette
	for _, c := range cols[:min(len(cols), 255)] {
		pal = append(pal, c)
	}
	idx := map[color.RGBA]uint8{}
	out := &gif.GIF{}
	for _, f := range frames {
		p := image.NewPaletted(f.Rect, pal)
		for i := 0; i < len(f.Pix); i += 4 {
			c := color.RGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}
			v, ok := idx[c]
			if !ok {
				v = uint8(pal.Index(c))
				idx[c] = v
			}
			p.Pix[i/4] = v
		}
		out.Image = append(out.Image, p)
		out.Delay = append(out.Delay, delay)
	}
	fl, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fl.Close()
	return gif.EncodeAll(fl, out)
}

var (
	_ = math.Pi
	_ = fmt.Sprint
	_ = ebiten.NewImage
)

// compareStrips writes, for each LPC animation a motion follows, the
// motion's key poses facing right as a strip (one cell a pose) and their
// times in milliseconds, for setting beside LPC's own (an internal review,
// made outside the game: no LPC art is in the game or its repository).
func (g *Game) compareStrips(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	buf := newFineBuf(cellW, cellH)
	cw, ch := cellW*HDDensity, cellH*HDDensity
	look := sim.Look{Hair: sim.HairShaggy, HairColor: 2, Skin: 1}
	b := g.art.bodyIn(look, 0)
	tint := playerColors[int(sheetID)%len(playerColors)]
	type strip struct {
		name   string
		frames func() ([]figDraw, []int)
	}
	motionStrip := func(m motionID) func() ([]figDraw, []int) {
		return func() ([]figDraw, []int) {
			var fs []figDraw
			for f := 0; f < m.frames(); f++ {
				lo, up := m.keys(sim.DirRight, f)
				fs = append(fs, figDraw{b: b, up: up, lo: lo, tint: tint})
			}
			return fs, motions[m].ms
		}
	}
	gait := func(lo loAnim, up upAnim, cycleMs int) func() ([]figDraw, []int) {
		return func() ([]figDraw, []int) {
			var fs []figDraw
			var ms []int
			for f := 0; f < loFrames; f++ {
				fs = append(fs, figDraw{b: b, up: upKey{sim.DirRight, up, uint8(f * upFrames / loFrames)}, lo: loKey{sim.DirRight, lo, uint8(f)}, tint: tint})
				ms = append(ms, cycleMs/loFrames)
			}
			return fs, ms
		}
	}
	strips := []strip{
		{"walk", gait(loWalk, upWalk, 1000)}, {"run", gait(loRun, upRun, 1000)},
		{"idle", motionStrip(moIdle)}, {"combat_idle", motionStrip(moReady)},
		{"slash", motionStrip(moChop)}, {"backslash", motionStrip(moBackslash)},
		{"thrust", motionStrip(moThrust)}, {"shoot", motionStrip(moShoot)},
		{"hurt", motionStrip(moDie)}, {"sit", motionStrip(moSitDown)},
		{"spellcast", motionStrip(moDrink)}, {"emote", motionStrip(moEatBite)},
	}
	for _, s := range strips {
		fs, ms := s.frames()
		img := image.NewRGBA(image.Rect(0, 0, len(fs)*cw, ch))
		for i, f := range fs {
			buf.img.Fill(reelGrass)
			drawFig(buf, f, cellX, cellY)
			cell := image.NewRGBA(image.Rect(0, 0, cw, ch))
			buf.img.ReadPixels(cell.Pix)
			draw.Draw(img, image.Rect(i*cw, 0, (i+1)*cw, ch), cell, image.Point{}, draw.Src)
		}
		if err := sheetPNG(filepath.Join(dir, s.name+".png"), img); err != nil {
			return err
		}
		txt := ""
		for _, m := range ms {
			txt += fmt.Sprintf("%d\n", m)
		}
		if err := os.WriteFile(filepath.Join(dir, s.name+".txt"), []byte(txt), 0o644); err != nil {
			return err
		}
	}
	return nil
}
