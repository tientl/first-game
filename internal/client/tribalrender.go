package client

import (
	"image"
	"image/color"
	"math"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Drawing the tribal age: its creatures and entities, piles on the ground,
// bodies and heads, the weather and the seasons.

// addTribalArt makes the tribal age's sprites and icons. It runs after the
// animals' art exists, since trophy heads are cut from it.
func (a *art) addTribalArt() {
	a.tribal = genTribalArt()
	genTribalIcons(&a.icons)
	a.heads = make(map[sim.CreatureKind]sprite)
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		h := sim.HeadItem(k)
		if !h.Valid() {
			continue
		}
		s := a.headOf(k)
		a.heads[k] = s
		a.icons[h] = iconOf(s)
	}
}

// headOf cuts the head of a creature of kind k from its standing sprite: the
// front part of the side view, which faces right.
func (a *art) headOf(k sim.CreatureKind) sprite {
	g := &Game{art: a}
	s, _ := g.creatureSprite(protocol.CreatureView{Kind: k, State: sim.CreatureIdle}, false)
	b := s.img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := make([]byte, 4*w*h)
	s.img.ReadPixels(pix)
	opaque := func(x, y int) bool { return pix[4*(y*w+x)+3] > 100 }
	right := -1
	for x := w - 1; x >= 0 && right < 0; x-- {
		for y := 0; y < h; y++ {
			if opaque(x, y) {
				right = x
				break
			}
		}
	}
	if right < 0 {
		return s
	}
	cw := max(10, int(float64(w)*0.36))
	if w < 22 {
		cw = w // small animals: the whole of them
	}
	left := max(0, right-cw+1)
	top, bottom := h, -1
	for y := 0; y < h; y++ {
		for x := left; x <= right; x++ {
			if opaque(x, y) {
				top, bottom = min(top, y), max(bottom, y)
				break
			}
		}
	}
	if bottom < top {
		return s
	}
	bottom = min(bottom, top+int(float64(cw)*1.1)) // the head and neck, not the legs
	sub := s.img.SubImage(image.Rect(b.Min.X+left, b.Min.Y+top, b.Min.X+right+1, b.Min.Y+bottom+1)).(*ebiten.Image)
	img := ebiten.NewImage(right-left+1, bottom-top+1)
	img.DrawImage(sub, nil)
	return sprite{img, float64(right-left+1) / 2, float64(bottom - top + 1)}
}

// iconOf fits s in a 16x16 bag icon.
func iconOf(s sprite) *ebiten.Image {
	icon := ebiten.NewImage(16, 16)
	b := s.img.Bounds()
	k := min(1, 15/math.Max(float64(b.Dx()), float64(b.Dy())))
	if k < 1 {
		k = math.Max(k, 0.25)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate((16-float64(b.Dx())*k)/2, (16-float64(b.Dy())*k)/2)
	op.Filter = ebiten.FilterLinear
	icon.DrawImage(s.img, op)
	return icon
}

// --- creatures ------------------------------------------------------------------

// tribalSprite is creatureSprite for the tribal age's creatures and the
// colourful fish; ok is false for any other kind.
func (g *Game) tribalSprite(c protocol.CreatureView, moving bool) (sprite, float64, bool) {
	a := &g.art.tribal
	side := 0
	if c.FaceLeft {
		side = 1
	}
	t := g.frame + int(c.ID)*7
	switch c.Kind {
	case sim.CreatureSpider:
		switch {
		case c.State == sim.CreatureSpit:
			return a.spider[side][spiderSpit], 0, true
		case c.State == sim.CreatureStrike:
			return a.spider[side][spiderBite], 0, true
		case moving:
			return a.spider[side][spiderWalkA+(t/3)%2], 0, true
		}
		return a.spider[side][spiderWalkA], 0, true
	case sim.CreatureBear:
		switch {
		case c.State == sim.CreatureClaw:
			return a.bear[side][bearClaw], 0, true
		case c.State == sim.CreatureStrike:
			return a.bear[side][bearBite], 0, true
		case moving && c.State == sim.CreatureChase:
			return a.bear[side][bearWalkA+(t/5)%2], float64((t / 5) % 2), true
		case moving:
			return a.bear[side][bearWalkA+(t/10)%2], 0, true
		}
		return a.bear[side][bearWalkA], 0, true
	case sim.CreatureTribesman:
		switch {
		case c.State == sim.CreatureStrike:
			return a.tribe[side][tribeThrust], 0, true
		case moving && (c.State == sim.CreatureChase || c.State == sim.CreatureFlee):
			return a.tribe[side][tribeWalkA+(t/4)%2], float64((t / 4) % 2), true
		case moving:
			return a.tribe[side][tribeWalkA+(t/8)%2], 0, true
		}
		return a.tribe[side][tribeStand], 0, true
	case sim.CreatureFish:
		return a.fish[int(c.ID)%len(a.fish)][side][(t/9)%fishFrames], 0, true
	case sim.CreatureScorpion:
		sc := g.art.desert.scorpion
		pose := scorpionWalkA
		switch {
		case c.State == sim.CreatureClaw:
			pose = scorpionPinch
		case c.State == sim.CreatureStrike:
			pose = scorpionSting
		case moving && c.State == sim.CreatureChase:
			pose = scorpionWalkA + (t/4)%2
		case moving:
			pose = scorpionWalkA + (t/8)%2
		}
		s := sc[pose]
		if side == 1 {
			s = flipped(s)
		}
		return s, 0, true
	}
	return sprite{}, 0, false
}

// tribalLabel is the target label of a tribal-age creature.
func tribalLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	name := c.Kind.Name()
	switch c.Kind {
	case sim.CreatureSpider:
		return name + " — đánh (nhả tơ, cắn có độc!)", colBad, true
	case sim.CreatureBear:
		return name + " — đánh (cào và cắn rất đau!)", colBad, true
	case sim.CreatureScorpion:
		return name + " — đánh (kẹp rồi chích độc!)", colBad, true
	case sim.CreatureTribesman:
		if c.State == sim.CreatureFlee {
			return name + " (bị thương, đang chạy về làng)", colWhite, true
		}
		return name + " — đánh (cầm giáo, hung dữ!)", colBad, true
	}
	if c.State == sim.CreatureTrapped {
		return name + " (mắc bẫy) — đánh", colGood, true
	}
	return "", colWhite, false
}

// --- entities ---------------------------------------------------------------------

// tribalEntitySprite is entitySprite for the tribal age's entities.
func (g *Game) tribalEntitySprite(e protocol.EntityView) (sprite, float64, bool) {
	a := &g.art.tribal
	switch e.Kind {
	case sim.EntityCorpse:
		if s, ok := g.bodySprite(e); ok {
			return s, 0, true
		}
		return a.blood[1], 0, true
	case sim.EntityTrophy:
		return a.post, 0, true
	case sim.EntityDoor:
		return a.door, 0, true
	case sim.EntityFloor:
		return a.floor, 0, true
	case sim.EntitySpikes:
		return a.spikes, 0, true
	case sim.EntityTrap:
		if e.Ready {
			return a.trapSet, 0, true
		}
		return a.trapSprung, 0, true
	case sim.EntityTorchStand:
		return a.torchStick, 0, true
	case sim.EntityFlower:
		if e.Ready {
			return a.flowers[int(e.Var)%len(a.flowers)], 0.08, true
		}
		return a.flowerPicked, 0, true
	case sim.EntitySapling:
		return a.saplings[int(e.Var)%len(a.saplings)], 0.06, true
	case sim.EntityGlowPlant:
		v := int(e.Var) % 3
		if e.Ready {
			return a.glow[v], 0.02, true
		}
		return a.glowPicked[v], 0.02, true
	case sim.EntityManEater:
		if sim.ManEaterSnapping(uint64(e.Fuel), 0) {
			return a.manEater[1], 0, true
		}
		return a.manEater[0], 0.03, true
	case sim.EntityTreasure:
		if e.Ready {
			return a.treasure[0], 0, true
		}
		return a.treasure[1], 0, true
	case sim.EntityHut:
		return a.hut, 0, true
	case sim.EntityTotem:
		return a.totem, 0, true
	case sim.EntityBonfire:
		return a.bonfire, 0, true
	case sim.EntityPyramid, sim.EntityPyramidStone:
		return g.art.desert.pyramid, 0, true
	case sim.EntityCaveMouth, sim.EntityCaveRock:
		return g.art.cave.mouth, 0, true
	case sim.EntityCaveExit:
		return g.art.cave.exit, 0, true
	}
	return sprite{}, 0, false
}

// bodySprite is the standing sprite of a body's creature, before it is laid down.
func (g *Game) bodySprite(e protocol.EntityView) (sprite, bool) {
	k := sim.CreatureKind(e.Var)
	if k == sim.CreatureNone || k >= sim.NumCreatureKinds {
		return sprite{}, false
	}
	s, _ := g.creatureSprite(protocol.CreatureView{Kind: k, State: sim.CreatureIdle, FaceLeft: e.Big}, false)
	return s, true
}

func tribalShadow(e protocol.EntityView) (float64, bool) {
	switch e.Kind {
	case sim.EntityCorpse, sim.EntityFloor, sim.EntityTrap, sim.EntitySapling:
		return 0, true
	case sim.EntityTrophy, sim.EntityTorchStand:
		return 8, true
	case sim.EntityDoor, sim.EntitySpikes:
		return 26, true
	case sim.EntityFlower:
		return 10, true
	case sim.EntityGlowPlant:
		return []float64{14, 16, 28}[e.Var%3], true
	case sim.EntityManEater:
		return 26, true
	case sim.EntityTreasure:
		return 22, true
	case sim.EntityHut:
		return 54, true
	case sim.EntityTotem:
		return 14, true
	case sim.EntityBonfire, sim.EntityPyramid, sim.EntityPyramidStone, sim.EntityCaveMouth, sim.EntityCaveRock, sim.EntityCaveExit:
		return 0, true
	}
	return 0, false
}

// drawTribalEntity draws the entities that need more than one sprite, and
// reports whether it drew e.
func (g *Game) drawTribalEntity(dst *ebiten.Image, e protocol.EntityView, d drawable) bool {
	a := &g.art.tribal
	switch e.Kind {
	case sim.EntityFloor, sim.EntityPyramidStone, sim.EntityCaveRock:
		return true // flat, or part of the pyramid drawn whole from its doorway
	case sim.EntityCorpse:
		g.drawBody(dst, e, d)
		return true
	case sim.EntityTrophy:
		drawAt(dst, a.post, d.sx, d.sy, 1, 1, 0, nil)
		if h, ok := g.art.heads[sim.CreatureKind(e.Var)]; ok {
			b := h.img.Bounds()
			k := min(1, 22/float64(max(b.Dx(), b.Dy())))
			drawAt(dst, h, d.sx, d.sy-a.post.ay+5, k, k, 0, nil)
		}
		return true
	case sim.EntityTorchStand:
		drawAt(dst, a.torchStick, d.sx, d.sy, 1, 1, 0, nil)
		f := g.art.flames[(g.frame/4+int(e.ID))%flameFrames]
		drawAt(dst, f, d.sx, d.sy-a.torchStick.ay+4, 0.42, 0.5, 0, nil)
		return true
	case sim.EntityBonfire:
		drawAt(dst, a.bonfire, d.sx, d.sy, 1, 1, 0, nil)
		f := g.art.flames[(g.frame/4+int(e.ID))%flameFrames]
		drawAt(dst, f, d.sx, d.sy+1, 1.35, 1.5, 0, nil)
		f2 := g.art.flames[(g.frame/5+int(e.ID)+3)%flameFrames]
		drawAt(dst, f2, d.sx-6, d.sy+2, 0.8, 0.9, 0, nil)
		return true
	}
	return false
}

// drawBody draws a kill lying in its blood: a beast on its back, legs in
// the air; a person on their side.
func (g *Game) drawBody(dst *ebiten.Image, e protocol.EntityView, d drawable) {
	k := sim.CreatureKind(e.Var)
	pool := 1
	switch w := creatureShadow(k); {
	case w < 14:
		pool = 0
	case w > 26:
		pool = 2
	}
	drawAt(dst, g.art.tribal.blood[pool], d.sx, d.sy, 1, 1, 0, nil)
	s, ok := g.bodySprite(e)
	if !ok {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	if k == sim.CreatureTribesman {
		op.GeoM.Rotate(-math.Pi / 2)
		op.GeoM.Translate(d.sx-6, d.sy-2)
	} else {
		const squash = 0.85
		op.GeoM.Scale(1, -squash)
		op.GeoM.Translate(d.sx, d.sy-s.ay*squash+3)
	}
	op.ColorScale.Scale(0.78, 0.72, 0.72, 1) // the colour gone out of it
	dst.DrawImage(s.img, op)
}

// drawFloors draws wooden floors flat on the ground, under everything.
func (g *Game) drawFloors(dst *ebiten.Image, cam camera) {
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityFloor {
			continue
		}
		sx, sy := cam.toScreen(float64(e.TX), float64(e.TY))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx, sy)
		dst.DrawImage(g.art.tribal.floor.img, op)
	}
}

// --- piles on the ground --------------------------------------------------------------

// dropPos is where to draw pile d now: falling from where it dropped in an
// arc for its first moments, then lying still.
func (g *Game) dropPos(d protocol.DropView) (x, y, lift float64) {
	const fall = 8.0 // ticks
	age := float64(d.Age) + g.alpha()
	if age >= fall || (d.FromX == 0 && d.FromY == 0) {
		return float64(d.X), float64(d.Y), 0
	}
	t := age / fall
	x = float64(d.FromX) + (float64(d.X)-float64(d.FromX))*t
	y = float64(d.FromY) + (float64(d.Y)-float64(d.FromY))*t
	return x, y, 10 * math.Sin(t*math.Pi)
}

func (g *Game) drawDrop(dst *ebiten.Image, d protocol.DropView, sx, sy, lift float64) {
	icon := g.art.icons[d.Item]
	if icon == nil {
		return
	}
	bob := 0.0
	if lift == 0 {
		bob = math.Round(math.Sin(float64(g.frame)*0.07+float64(d.ID)) * 1)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-8, -14)
	op.GeoM.Scale(0.8, 0.8)
	op.GeoM.Translate(math.Round(sx), math.Round(sy-lift-bob))
	dst.DrawImage(icon, op)
	if d.Count > 1 {
		// A little stack: a second icon behind, offset.
		op2 := &ebiten.DrawImageOptions{}
		op2.GeoM.Translate(-8, -14)
		op2.GeoM.Scale(0.8, 0.8)
		op2.GeoM.Translate(math.Round(sx)+2, math.Round(sy-lift-bob)-1)
		op2.ColorScale.Scale(0.7, 0.7, 0.7, 1)
		dst.DrawImage(icon, op2)
		dst.DrawImage(icon, op)
	}
}

// dropTarget is the pile Space would pick up now, if any.
func (g *Game) dropTarget(x, y float64) (protocol.DropView, bool) {
	var cands []sim.DropCandidate
	for _, d := range g.curr.Drops {
		cands = append(cands, sim.DropCandidate{ID: d.ID, X: float64(d.X), Y: float64(d.Y)})
	}
	id, ok := sim.NearestDrop(x, y, cands)
	if !ok {
		return protocol.DropView{}, false
	}
	for _, d := range g.curr.Drops {
		if d.ID == id {
			return d, true
		}
	}
	return protocol.DropView{}, false
}

// --- weather and the seasons -----------------------------------------------------------

// seasonTiles is the ground as the season shows it: in winter the meadows,
// woods and swamps lie under snow and the ponds are ice.
func seasonTiles(w, h int, tiles []sim.Tile, s sim.Season, pond []bool) []sim.Tile {
	if s != sim.Winter {
		return tiles
	}
	out := make([]sim.Tile, len(tiles))
	for i, t := range tiles {
		switch {
		case t == sim.TileWater && pond != nil && pond[i]:
			t = sim.TileIce
		case t == sim.TileGrass || t == sim.TileForest || t == sim.TileMarsh:
			t = sim.TileSnow
		}
		out[i] = t
	}
	return out
}

// frozenAt reports whether tile (tx, ty) is pond ice now.
func (g *Game) frozenAt(tx, ty int) bool {
	if !sim.Frozen(g.curr.Season) || g.pondMask == nil || tx < 0 || ty < 0 || tx >= g.width || ty >= g.height {
		return false
	}
	return g.pondMask[ty*g.width+tx]
}

// underIce reports whether c swims under pond ice: not drawn.
func (g *Game) underIce(c protocol.CreatureView) bool {
	switch c.Kind {
	case sim.CreatureFish, sim.CreatureCroc, sim.CreatureBrachy, sim.CreatureChampso, sim.CreatureCrab:
		return g.frozenAt(int(math.Floor(float64(c.X))), int(math.Floor(float64(c.Y))))
	}
	return false
}

// treeTint is how the season colours a tree's leaves: gold in autumn, cold
// in winter; nil when there is nothing to change.
func (g *Game) treeTint() color.Color {
	switch g.curr.Season {
	case sim.Autumn:
		return color.RGBA{255, 196, 120, 255}
	case sim.Winter:
		return color.RGBA{214, 226, 240, 255}
	}
	return nil
}

// weatherDim is how much the sky darkens for the weather.
func (g *Game) weatherDim() float64 {
	switch g.curr.Weather {
	case sim.WeatherStorm:
		return 0.34
	case sim.WeatherRain:
		return 0.2
	case sim.WeatherSnow, sim.WeatherTornado:
		return 0.16
	case sim.WeatherWind:
		return 0.06
	}
	return 0
}

// drawWeather draws rain, snow and blowing leaves over the world, the
// tornadoes, and webs in flight. Inside a cave there is no weather.
func (g *Game) drawWeather(dst *ebiten.Image, cam camera, alpha float64) {
	a := &g.art.tribal
	for _, t := range g.curr.Tornadoes {
		sx, sy := cam.toScreen(float64(t.X), float64(t.Y))
		s := a.tornado[(g.frame/4)%len(a.tornado)]
		g.drawShadow(dst, sx, sy, 40)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-s.ax, sy-s.ay)
		op.ColorScale.ScaleAlpha(0.88)
		dst.DrawImage(s.img, op)
		// Debris whirling round it.
		for k := range 10 {
			ph := float64(g.frame)*0.12 + float64(k)*0.63
			r := 6 + float64(k%4)*4
			x := sx + math.Cos(ph)*r
			y := sy - 6 - float64(k)*5 + math.Sin(ph)*r*0.3
			vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 2, 1, color.RGBA{70, 58, 40, 220}, false)
		}
	}
	for _, w := range g.curr.Webs {
		x, y := float64(w.X)+float64(w.VX)*alpha, float64(w.Y)+float64(w.VY)*alpha
		sx, sy := cam.toScreen(x, y)
		drawAt(dst, a.web, sx, sy-10, 1, 1, float64(g.frame)*0.3, nil)
	}
	if g.caveIn() > 0.5 {
		return
	}
	b := dst.Bounds()
	W, H := float64(b.Dx()), float64(b.Dy())
	secs := float64(g.frame) / 60
	windy := g.curr.Weather.Windy()
	switch {
	case g.curr.Weather.Wet():
		n := int(W * H / 900)
		if g.curr.Weather == sim.WeatherStorm {
			n *= 2
		}
		slant := 0.25
		if windy {
			slant = 0.7
		}
		for i := range n {
			h1, h2 := hashf(i, 1, 501), hashf(i, 2, 502)
			fall := math.Mod(h2*H+secs*(300+h1*80), H+20) - 10
			x := math.Mod(h1*W+fall*slant, W)
			if x < 0 {
				x += W
			}
			col := color.RGBA{170, 196, 230, 150}
			vector.StrokeLine(dst, float32(x), float32(fall), float32(x-4*slant), float32(fall-6), 1, col, false)
		}
	case g.curr.Weather == sim.WeatherSnow:
		n := int(W * H / 700)
		for i := range n {
			h1, h2, h3 := hashf(i, 1, 511), hashf(i, 2, 512), hashf(i, 3, 513)
			y := math.Mod(h2*H+secs*(14+h3*16), H)
			x := math.Mod(h1*W+math.Sin(secs*0.8+h3*9)*6+secs*8, W)
			size := float32(1)
			if h3 > 0.8 {
				size = 2
			}
			vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), size, size, color.RGBA{246, 248, 255, 230}, false)
		}
	}
	if windy {
		// Leaves and dust tearing past.
		n := int(W * H / 6000)
		leaf := []color.RGBA{{120, 150, 60, 230}, {170, 130, 50, 230}, {200, 90, 40, 230}}
		for i := range n {
			h1, h2 := hashf(i, 4, 521), hashf(i, 5, 522)
			x := math.Mod(h1*W+secs*(160+h2*120), W+40) - 20
			y := math.Mod(h2*H+math.Sin(secs*3+h1*20)*8, H)
			c := leaf[i%len(leaf)]
			if g.curr.Season == sim.Winter {
				c = color.RGBA{240, 244, 255, 220}
			}
			vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 2, 1, c, false)
		}
	}
}

// drawLightning draws the bolts of the last moments and the flash they
// light the world with. It goes after the darkness, so night lights up.
func (g *Game) drawLightning(dst *ebiten.Image, cam camera) {
	if len(g.curr.Strikes) == 0 || g.caveIn() > 0.5 {
		return
	}
	flash := 0.0
	for _, s := range g.curr.Strikes {
		age := float64(s.Age)
		if age > 8 {
			continue
		}
		flash = math.Max(flash, 1-age/8)
		if age > 4 {
			continue
		}
		sx, sy := cam.toScreen(float64(s.X), float64(s.Y))
		// A jagged bolt from the top of the view down to where it struck.
		x, y := sx+float64(int(s.X*7)%40-20), -10.0
		seed := int(s.X*13 + s.Y*7)
		for k := 0; y < sy; k++ {
			ny := math.Min(sy, y+8+hashf(seed, k, 531)*10)
			nx := x + (hashf(seed, k, 532)-0.5)*14 + (sx-x)*0.25
			if ny >= sy {
				nx = sx
			}
			vector.StrokeLine(dst, float32(x), float32(y), float32(nx), float32(ny), 3, color.RGBA{140, 170, 255, 120}, false)
			vector.StrokeLine(dst, float32(x), float32(y), float32(nx), float32(ny), 1, color.RGBA{250, 252, 255, 255}, false)
			x, y = nx, ny
		}
		vector.FillCircle(dst, float32(sx), float32(sy), 5, color.RGBA{255, 255, 255, 200}, false)
	}
	if flash > 0 {
		b := dst.Bounds()
		a := uint8(105 * flash)
		vector.FillRect(dst, 0, 0, float32(b.Dx()), float32(b.Dy()), color.RGBA{a, a, a, a}, false)
	}
}

// glowLights adds the light of the plants that glow at night: holes in the
// darkness when punch is set, else a coloured glow over the scene.
func (g *Game) glowLights(dst *ebiten.Image, cam camera, punch bool, dark float64) {
	if dark < 0.05 {
		return
	}
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityGlowPlant || !e.Ready {
			continue
		}
		r := sim.GlowRadius(e.Var)
		pulse := 1 + 0.08*math.Sin(float64(g.frame)*0.03+float64(e.ID))
		op := g.lightOp(cam, e.TX, e.TY, r*pulse)
		if punch {
			op.ColorScale.ScaleAlpha(float32(0.75))
			op.Blend = ebiten.BlendDestinationOut
		} else {
			k := float32(0.35 * dark * pulse)
			if e.Var%2 == 0 {
				op.ColorScale.Scale(k*0.25, k*0.85, k, k)
			} else {
				op.ColorScale.Scale(k*0.7, k*0.35, k, k)
			}
			op.Blend = ebiten.BlendLighter
		}
		dst.DrawImage(g.art.light, op)
	}
}

// drawGlowMasks draws the glowing parts of the plants bright over the dark.
func (g *Game) drawGlowMasks(dst *ebiten.Image, cam camera, dark float64) {
	if dark < 0.05 {
		return
	}
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityGlowPlant || !e.Ready {
			continue
		}
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		if g.hiddenInCave(float64(e.TX)+0.5, float64(e.TY)+0.5) {
			continue
		}
		m := g.art.tribal.glowMask[e.Var%3]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-m.ax, sy-m.ay)
		k := float32(math.Min(1, dark*1.2) * (0.85 + 0.15*math.Sin(float64(g.frame)*0.03+float64(e.ID))))
		op.ColorScale.Scale(k, k, k, k)
		op.Blend = ebiten.BlendLighter
		dst.DrawImage(m.img, op)
	}
}

// darknessNow is how dark the scene is: night, the cave, an overcast sky.
func (g *Game) darknessNow() float64 {
	night := (1 - sim.Ambient(g.curr.Tick)) * maxDarkness
	if g.caveIn() < 0.5 {
		night = math.Max(night, g.weatherDim())
	}
	return math.Max(night, g.caveDarkness())
}

// tribalEntityLabel finishes the target label of a tribal-age entity.
func (g *Game) tribalEntityLabel(e protocol.EntityView, label string, col color.RGBA) (string, color.RGBA) {
	switch e.Kind {
	case sim.EntityCorpse:
		k := sim.CreatureKind(e.Var)
		if k == sim.CreatureTribesman {
			return "Xác thổ dân — lục soát", col
		}
		verb := " — mổ xác"
		if _, ok := sim.SwordTier(g.myHeld()); ok {
			verb += " (dao: nhanh gấp đôi)"
		}
		return "Xác " + lowerName(k.Name()) + verb, col
	case sim.EntityTrophy:
		return sim.HeadItem(sim.CreatureKind(e.Var)).Def().Name + " — gỡ xuống", col
	case sim.EntityFlower:
		if e.Ready {
			return sim.FlowerKind(e.Var).Item().Def().Name + " — hái", col
		}
	case sim.EntitySapling:
		return label + " — đang lớn", colMuted
	case sim.EntityGlowPlant:
		if e.Ready {
			return label + " — lấy bào tử", col
		}
	case sim.EntityManEater:
		if e.Var != 0 {
			return label + " (canh nhà) — chặt", colGood
		}
		return label + " — chặt (coi chừng bị đớp!)", colBad
	case sim.EntityTreasure:
		if e.Ready {
			return label + " — mở", colGood
		}
		return label + " (đã mở)", colMuted
	case sim.EntityTrap:
		if !e.Ready {
			return label + " (đã sập) — đặt lại", col
		}
		return label + " — dỡ bằng rìu", col
	case sim.EntityTorchStand:
		return label + " — nhổ lên", col
	case sim.EntityDoor, sim.EntityFloor, sim.EntitySpikes:
		if !g.has(sim.ToolAxe) {
			return label + " — cần rìu để dỡ", colBad
		}
		return label + " — dỡ", col
	}
	return label, col
}

// lowerName lower-cases the first letter of a common name for use mid-sentence.
func lowerName(s string) string {
	r := []rune(s)
	if len(r) == 0 || !strings.ContainsRune(s, ' ') && !strings.HasPrefix(s, "Thỏ") && !strings.HasPrefix(s, "Rắn") &&
		!strings.HasPrefix(s, "Cua") && !strings.HasPrefix(s, "Gấu") && !strings.HasPrefix(s, "Nhện") {
		return s
	}
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
