package client

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The scale sheet: a dev picture of the player standing next to every tree,
// plant, rock and building (and a few beasts), each on the same ground line
// and labelled with its height in screen pixels at 1080p and in player
// heights. It is how the sizes of the world are checked against the figure:
//
//	go run ./cmd/shot -w 1920 -h 1080 -scalesheet -o sheet.png
//
// See PLAN.md, "Tỉ lệ".

// scaleEntry is one thing on the sheet.
type scaleEntry struct {
	name string
	s    sprite
	// creature: its height is from the ground to its top, as for the rest,
	// but it is not part of the environment's sizes.
	creature bool
}

// playerArtH is how tall the standing player is, in art pixels (93 screen
// pixels at 1080p): the yardstick everything in the world is sized by.
const playerArtH = 31.0

// ScaleRow is one line of ScaleHeights.
type ScaleRow struct {
	Group, Name string
	Px          int     // height on a 1080p screen
	Players     float64 // in player heights
}

// scaleGroups lists what goes on the sheet, group by group.
func (g *Game) scaleGroups() (names []string, groups [][]scaleEntry) {
	a := g.art
	add := func(name string, es ...scaleEntry) {
		if scaleOnly != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(scaleOnly)) {
			return
		}
		names, groups = append(names, name), append(groups, es)
	}
	ent := func(name string, e protocol.EntityView) scaleEntry {
		s, _ := g.entitySprite(e)
		return scaleEntry{name: name, s: s}
	}
	ready := func(k sim.EntityKind, v uint8) protocol.EntityView {
		return protocol.EntityView{Kind: k, Var: v, Ready: true}
	}
	nShape := len(leafRamps)
	sets := [2]treeSet{a.treesOf(0), a.treesOf(1)}
	anc := [2]ancientTrees{a.ancientOf(0), a.ancientOf(1)}
	fruit := a.fruitOf()
	tree := func(sh treeShape, size int) sprite { return sets[size].trees[int(sh)*nShape] }

	add("Trees",
		scaleEntry{name: "round (small)", s: tree(shapeRound, 0)},
		scaleEntry{name: "round", s: tree(shapeRound, 1)},
		scaleEntry{name: "tall", s: tree(shapeTall, 1)},
		scaleEntry{name: "fork", s: sets[1].vined[int(shapeFork)*nShape+1]},
		scaleEntry{name: "spread", s: tree(shapeSpread, 1)},
		scaleEntry{name: "oak", s: sets[1].trees[int(shapeOak)*nShape+2]},
		scaleEntry{name: "weeping", s: tree(shapeWeeping, 1)},
	)
	add("Conifers and ancient trees",
		scaleEntry{name: "pine (small)", s: sets[0].pines[1]},
		scaleEntry{name: "pine", s: sets[1].pines[1]},
		scaleEntry{name: "snow pine", s: a.snowPinesOf(1)[2]},
		scaleEntry{name: "tree fern", s: anc[1].ferns[0]},
		scaleEntry{name: "cycad", s: anc[1].cycads[0]},
		scaleEntry{name: "araucaria", s: anc[1].araucarias[0]},
		scaleEntry{name: "ginkgo", s: anc[1].ginkgos[0]},
		scaleEntry{name: "redwood (small)", s: anc[0].redwoods[0]},
		scaleEntry{name: "redwood", s: anc[1].redwoods[0]},
	)
	woods := woodsArtOf()
	add("Swamp, fruit and jungle trees",
		scaleEntry{name: "cypress", s: swampTree(1, 9)},
		scaleEntry{name: "snag", s: swampTree(1, 0)},
		scaleEntry{name: "fig", s: fruit.figs[0]},
		scaleEntry{name: "fruit ginkgo", s: fruit.ginkgos[0]},
		scaleEntry{name: "palm", s: fruit.palms[0]},
		scaleEntry{name: "banana", s: woods.banana[0]},
		scaleEntry{name: "glow tree", s: a.tribal.glow[2]},
		scaleEntry{name: "giant tree", s: woods.giants[0]},
		scaleEntry{name: "giant stump", s: woods.giantStump},
	)
	low := []scaleEntry{
		{name: "stump (small)", s: a.stumps[0]},
		{name: "stump", s: a.stumps[1]},
		{name: "sapling", s: a.tribal.saplings[0]},
		{name: "berry bush", s: a.bushFull},
		{name: "grass tuft", s: a.tuftFull},
		{name: "reeds", s: a.open.reed},
		{name: "cactus", s: a.open.cactus},
		{name: "mushrooms", s: a.open.mushroom},
		{name: "flower", s: a.tribal.flowers[0]},
		{name: "glow fern", s: a.tribal.glow[1]},
		{name: "man-eater", s: a.tribal.manEater[manEaterWait]},
		{name: "sweet potato", s: a.farm.crops[len(a.farm.crops)-1]},
	}
	for k := sim.FloraKind(0); k < sim.NumFlora; k++ {
		low = append(low, scaleEntry{name: k.Name(), s: floraSprite(k, true)})
	}
	for k := plantKind(0); k < numPlantKinds; k++ {
		low = append(low, scaleEntry{name: fmt.Sprintf("undergrowth %d", k), s: plantSprite(k, 0)})
	}
	add("Low plants", low...)

	add("Buildings",
		ent("wooden wall", ready(sim.EntityWall, 0)),
		ent("door", ready(sim.EntityDoor, 0)),
		ent("stone wall", ready(sim.EntityStoneWall, 0)),
		ent("spikes", ready(sim.EntitySpikes, 0)),
		ent("hut", ready(sim.EntityHut, 0)),
		ent("totem", ready(sim.EntityTotem, 0)),
		ent("bonfire", ready(sim.EntityBonfire, 0)),
		scaleEntry{name: "campfire", s: a.campFront},
		scaleEntry{name: "flame", s: a.flames[0]},
		ent("torch", ready(sim.EntityTorchStand, 0)),
		ent("trophy post", ready(sim.EntityTrophy, 0)),
		scaleEntry{name: "furnace", s: a.ore.furnace[0]},
		scaleEntry{name: "oven", s: a.cook.oven[0]},
		scaleEntry{name: "drying rack", s: a.cook.rack[0]},
		scaleEntry{name: "mortar", s: a.cook.mortar[0]},
		scaleEntry{name: "pot", s: a.cook.pot[0]},
		scaleEntry{name: "chest", s: a.chest},
		scaleEntry{name: "treasure", s: a.tribal.treasure[0]},
		scaleEntry{name: "bed", s: a.bed},
		scaleEntry{name: "trap", s: a.tribal.trapSet},
		scaleEntry{name: "tombstone", s: a.tombstone},
	)
	add("Rocks",
		scaleEntry{name: "boulder (small)", s: a.boulders[0][0]},
		scaleEntry{name: "boulder", s: a.boulders[1][0]},
		scaleEntry{name: "copper ore", s: a.ore.rocks[0][0]},
		scaleEntry{name: "coal", s: a.open.coal},
		scaleEntry{name: "den", s: a.beasts.den},
		scaleEntry{name: "cave mouth", s: a.cave.mouth},
		scaleEntry{name: "cave exit", s: a.cave.exit},
		scaleEntry{name: "pyramid", s: a.desert.pyramid},
	)
	beast := func(name string, k sim.CreatureKind) scaleEntry {
		s, _ := g.creatureSprite(protocol.CreatureView{Kind: k, State: sim.CreatureIdle}, false)
		return scaleEntry{name: name, s: s, creature: true}
	}
	add("Creatures (not rescaled)",
		beast("raptor", sim.CreatureRaptor),
		beast("tribesman", sim.CreatureTribesman),
		beast("sabre-tooth", sim.CreatureSabre),
		beast("bear", sim.CreatureBear),
		beast("three-horn", sim.CreatureTrike),
		beast("plate-back", sim.CreatureStego),
		beast("mammoth", sim.CreatureMammoth),
		beast("rex", sim.CreatureRex),
		beast("long-neck", sim.CreatureBrachio),
	)
	return names, groups
}

// ScaleHeights is the height of everything on the scale sheet.
func (g *Game) ScaleHeights() []ScaleRow {
	var rows []ScaleRow
	names, groups := g.scaleGroups()
	for i, es := range groups {
		for _, e := range es {
			if e.s.img == nil {
				continue
			}
			h := standH(e.s)
			rows = append(rows, ScaleRow{names[i], e.name, int(math.Round(h * HDDensity)), h / playerArtH})
		}
	}
	return rows
}

// ScaleSheet draws the scale sheet, one image pixel to a 1080p screen pixel.
// It must run inside the game loop (it reads an image back).
func (g *Game) ScaleSheet() image.Image {
	if g.art == nil {
		g.art = getArt()
	}
	const (
		sheetW = 640.0 // art pixels: 1920 image pixels
		gap    = 7.0
		labelH = 14.0 // art pixels under the ground line for the two lines of a label
		titleH = 12.0
	)
	names, groups := g.scaleGroups()
	type placed struct {
		e    scaleEntry
		x, y float64 // the ground point, art pixels
	}
	type title struct {
		s string
		y float64
	}
	var all []placed
	var titles []title
	var marks [][2]float64 // where a player stands
	y := 6.0
	if scaleTitle != "" {
		y += 16
	}
	for gi, es := range groups {
		titles = append(titles, title{names[gi], y})
		y += titleH
		// Lay the group out in rows, each led by a player.
		for i := 0; i < len(es); {
			x, top := 34.0, 40.0
			j := i
			for ; j < len(es); j++ {
				s := es[j].s
				if s.img == nil {
					continue
				}
				w := max(s.w(), 30)
				if j > i && x+w+gap > sheetW-4 {
					break
				}
				x += w + gap
				top = max(top, s.ay)
			}
			ground := y + top + 4
			marks = append(marks, [2]float64{16, ground})
			x = 34.0
			for k := i; k < j; k++ {
				s := es[k].s
				if s.img == nil {
					continue
				}
				w := max(s.w(), 30)
				// The sprite sits in its slot by its own left edge.
				all = append(all, placed{es[k], x + (w-s.w())/2 + s.ax, ground})
				x += w + gap
			}
			y = ground + labelH + 8
			i = j
		}
	}
	h := int(y) + 4
	buf := newFineBuf(int(sheetW), h)
	buf.img.Fill(color.RGBA{0x5c, 0x86, 0x4c, 0xff})
	me := protocol.PlayerView{ID: 1, Dir: sim.DirDown}
	for _, m := range marks {
		fillRect(buf, 0, float32(m[1]), float32(sheetW), 0.34, color.RGBA{0x2c, 0x48, 0x2a, 0xff}, false)
		fillRect(buf, 2, float32(m[1]-playerArtH), float32(sheetW-4), 0.34, color.RGBA{0xff, 0xff, 0xff, 0x38}, false)
	}
	for _, p := range all {
		drawAt(buf, p.e.s, p.x, p.y, 1, 1, 0, nil)
	}
	for _, m := range marks {
		g.drawFigureJ(buf, me, sim.DirDown, loKey{dir: sim.DirDown}, upKey{dir: sim.DirDown}, m[0], m[1], 0)
	}
	const K = HDDensity
	if scaleTitle != "" {
		drawText(buf.img, scaleTitle, 8*K, 5*K, color.RGBA{0xff, 0xe9, 0x9a, 0xff}, text.AlignStart, 3)
	}
	for _, t := range titles {
		drawText(buf.img, t.s, 8*K, t.y*K, colWhite, text.AlignStart, 2)
	}
	for _, p := range all {
		cx := (p.x - p.e.s.ax + p.e.s.w()/2) * K
		sh := standH(p.e.s)
		px := int(math.Round(sh * K))
		drawText(buf.img, p.e.name, cx, (p.y+1.5)*K, colWhite, text.AlignCenter, 1)
		drawText(buf.img, fmt.Sprintf("%d px  %.1fx", px, sh/playerArtH), cx, (p.y+7.5)*K, color.RGBA{0xff, 0xe9, 0x9a, 0xff}, text.AlignCenter, 1)
	}
	b := buf.img.Bounds()
	img := image.NewRGBA(b)
	buf.img.ReadPixels(img.Pix)
	buf.img.Deallocate()
	return img
}

// artBytes counts the bytes of every image made from a canvas (the sprites
// drawn in code), and artTime is how long the art made at start-up took.
var (
	artBytes atomic.Int64
	artTime  time.Duration
)

// ArtCost is what the art drawn in code has cost so far: the bytes of its
// images, how long the start-up set took to draw, and how long the trees
// drawn ahead after it took (0 until they are done; see lazyart.go).
func ArtCost() (bytes int64, startup, warm time.Duration) {
	return artBytes.Load(), artTime, time.Duration(warmTime.Load())
}

// scaleOnly, when set, keeps only the groups of the sheet whose name
// contains it (shot -scaleonly); scaleTitle heads the sheet.
var scaleOnly, scaleTitle string

// SetScaleOnly keeps only the groups of the scale sheet whose name contains
// only, and heads the sheet with title.
func SetScaleOnly(only, title string) { scaleOnly, scaleTitle = only, title }

// standH is how tall s stands above its ground point, in art pixels: from
// its anchor up to its highest solid pixel, not to the top of its canvas
// (which may leave room for a swing or a sway). It reads the image back, so
// it runs inside the game loop, like ScaleSheet.
func standH(s sprite) float64 {
	if h, ok := standHs[s.img]; ok {
		return h
	}
	b := s.img.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	s.img.ReadPixels(pix)
	top := b.Dy()
	for y := 0; y < b.Dy() && top == b.Dy(); y++ {
		for x := range b.Dx() {
			if pix[4*(y*b.Dx()+x)+3] >= 128 {
				top = y
				break
			}
		}
	}
	h := s.ay - float64(top)/float64(s.density())
	standHs[s.img] = h
	return h
}

var standHs = map[*ebiten.Image]float64{}
