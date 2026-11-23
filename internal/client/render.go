package client

import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

var playerColors = []color.RGBA{
	{214, 84, 70, 255}, {70, 130, 214, 255}, {232, 190, 60, 255},
	{120, 190, 110, 255}, {170, 110, 200, 255}, {230, 140, 70, 255},
}

// camera maps world coordinates (tiles) to whole screen pixels, so tiles
// never show seams and nothing shimmers while scrolling.
type camera struct{ ox, oy float64 }

func newCamera(cx, cy float64, sw, sh int) camera {
	return camera{
		ox: math.Round(cx*TileW) - float64(sw/2),
		oy: math.Round(cy*TileH) - float64(sh/2),
	}
}

func (c camera) toScreen(x, y float64) (float64, float64) {
	return math.Round(x*TileW) - c.ox, math.Round(y*TileH) - c.oy
}

// drawable is one upright thing in the depth-sorted pass.
type drawable struct {
	y        float64 // world y of its ground point: larger draws later (in front)
	id       uint32
	isPlayer bool
	idx      int // index into curr.Players or curr.Entities
	sx, sy   float64
	moving   bool
}

// Draw renders one frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 22, 30, 255})
	if g.art == nil {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if !g.ready || !g.hasCurr {
		msg := "Đang kết nối…"
		if g.disconnected {
			msg = "Mất kết nối tới máy chủ"
		}
		g.text(screen, msg, float64(sw/2), float64(sh/2-16), colWhite, text.AlignCenter, 2)
		return
	}

	alpha := g.alpha()
	cx, cy := float64(g.width)/2, float64(g.height)/2
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			cx, cy, _ = g.lerpPlayer(p, alpha)
		}
	}
	cam := newCamera(cx, cy, sw, sh)

	g.drawGround(screen, cam, sw, sh)
	g.drawSprites(screen, cam, alpha)
	g.drawLighting(screen, cam)
	g.drawWorldLabels(screen, cam)
	g.drawHUD(screen, sw, sh)

	if g.disconnected {
		g.dim(screen, sw, sh, 0.6)
		g.text(screen, "Mất kết nối tới máy chủ", float64(sw/2), float64(sh/2-16), colWhite, text.AlignCenter, 2)
	}
}

func (g *Game) drawGround(screen *ebiten.Image, cam camera, sw, sh int) {
	tx0 := int(math.Floor(cam.ox / TileW))
	tx1 := int(math.Floor((cam.ox + float64(sw)) / TileW))
	ty0 := int(math.Floor(cam.oy/TileH)) - 1 // one row up: its cliff may be visible
	ty1 := int(math.Floor((cam.oy + float64(sh)) / TileH))

	waterFrame := []int{0, 1, 2, 3, 2, 1}
	op := &ebiten.DrawImageOptions{}
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			t := g.tileAt(tx, ty)
			h := sim.HashTile(tx, ty)
			variant := int(h % tileVariants)
			if t == sim.TileWater {
				variant = waterFrame[(g.frame/20+int(h%6))%len(waterFrame)]
			}
			op.GeoM.Reset()
			op.GeoM.Translate(float64(tx*TileW)-cam.ox, float64(ty*TileH)-cam.oy)
			screen.DrawImage(g.art.tiles[t][variant], op)
		}
	}
	// Land that borders the sea to the south gets an earth wall, which gives
	// the island height.
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			if g.tileAt(tx, ty) != sim.TileWater && g.tileAt(tx, ty+1) == sim.TileWater {
				op.GeoM.Reset()
				op.GeoM.Translate(float64(tx*TileW)-cam.ox, float64((ty+1)*TileH)-cam.oy)
				screen.DrawImage(g.art.cliff, op)
			}
		}
	}
}

func (g *Game) drawSprites(screen *ebiten.Image, cam camera, alpha float64) {
	list := g.drawList[:0]
	for i, e := range g.curr.Entities {
		wx, wy := float64(e.TX)+0.5, float64(e.TY)+0.5
		sx, sy := cam.toScreen(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(e.ID), idx: i, sx: sx, sy: sy})
	}
	for i, p := range g.curr.Players {
		wx, wy, moving := g.lerpPlayer(p, alpha)
		sx, sy := cam.toScreen(wx, wy)
		list = append(list, drawable{y: wy, id: uint32(p.ID), isPlayer: true, idx: i, sx: sx, sy: sy, moving: moving})
	}
	slices.SortFunc(list, func(a, b drawable) int {
		if c := cmp.Compare(a.y, b.y); c != 0 {
			return c
		}
		if a.isPlayer != b.isPlayer {
			if a.isPlayer {
				return 1
			}
			return -1
		}
		return cmp.Compare(a.id, b.id)
	})
	g.drawList = list

	for _, d := range list {
		w := 26.0
		if !d.isPlayer {
			w = shadowWidth(g.curr.Entities[d.idx])
		}
		if w > 0 {
			g.drawShadow(screen, d.sx, d.sy, w)
		}
	}
	for _, d := range list {
		if d.isPlayer {
			g.drawPlayer(screen, g.curr.Players[d.idx], d)
		} else {
			g.drawEntity(screen, g.curr.Entities[d.idx], d)
		}
	}
}

func shadowWidth(e protocol.EntityView) float64 {
	switch e.Kind {
	case sim.EntityTree:
		if e.Ready {
			return 44
		}
		return 20
	case sim.EntityBoulder:
		return 44
	case sim.EntityBerryBush:
		return 38
	case sim.EntityGrassTuft:
		return 18
	default:
		return 0
	}
}

func (g *Game) drawShadow(screen *ebiten.Image, sx, sy, w float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-shadowSize/2, -shadowSize/2)
	op.GeoM.Scale(w/shadowSize, w*0.4/shadowSize)
	op.GeoM.Translate(sx, sy)
	op.ColorScale.ScaleAlpha(0.28)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(g.art.shadow, op)
}

// drawAt draws s with its anchor at (x, y), scaled and rotated about the anchor.
func drawAt(screen *ebiten.Image, s sprite, x, y, scaleX, scaleY, rot float64, tint color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Scale(scaleX, scaleY)
	if rot != 0 {
		op.GeoM.Rotate(rot)
	}
	op.GeoM.Translate(x, y)
	if tint != nil {
		op.ColorScale.ScaleWithColor(tint)
	}
	if scaleX != 1 || scaleY != 1 || rot != 0 {
		op.Filter = ebiten.FilterLinear
	}
	screen.DrawImage(s.img, op)
}

func (g *Game) entitySprite(e protocol.EntityView) sprite {
	a := g.art
	switch e.Kind {
	case sim.EntityTree:
		if e.Ready {
			return a.tree
		}
		return a.stump
	case sim.EntityBoulder:
		return a.boulder
	case sim.EntityBerryBush:
		if e.Ready {
			return a.bushFull
		}
		return a.bushPicked
	case sim.EntityGrassTuft:
		if e.Ready {
			return a.tuftFull
		}
		return a.tuftPicked
	default:
		return a.campfire
	}
}

func (g *Game) drawEntity(screen *ebiten.Image, e protocol.EntityView, d drawable) {
	drawAt(screen, g.entitySprite(e), d.sx, d.sy, 1, 1, 0, nil)
	if e.Kind == sim.EntityCampfire {
		size := 0.45 + 0.55*min(1, float64(e.Fuel)/sim.FireStartFuel)
		flicker := 1 + 0.12*math.Sin(float64(g.frame)*0.37+float64(e.ID))
		sway := 1 + 0.06*math.Sin(float64(g.frame)*0.23+float64(e.ID)*1.7)
		drawAt(screen, g.art.flame, d.sx, d.sy+1, size*sway, size*flicker, 0, nil)
		drawAt(screen, g.art.flameCore, d.sx, d.sy+1, size*sway, size*flicker, 0, nil)
	}
}

func (g *Game) drawPlayer(screen *ebiten.Image, p protocol.PlayerView, d drawable) {
	if p.Dead {
		drawAt(screen, g.art.tombstone, d.sx, d.sy, 1, 1, 0, nil)
		return
	}
	t := float64(g.frame)
	bob := 0.0
	if d.moving {
		bob = math.Round(math.Abs(math.Sin(t*0.3)) * 2)
	}
	tint := playerColors[int(p.ID)%len(playerColors)]

	// When facing away, the tool is behind the body.
	if p.Acting && p.Dir == sim.DirUp {
		g.drawTool(screen, p, d, t, bob)
	}
	drawAt(screen, g.art.legs, d.sx, d.sy, 1, 1, 0, nil)
	drawAt(screen, g.art.body, d.sx, d.sy-6-bob, 1, 1, 0, tint)
	drawAt(screen, g.art.heads[p.Dir&3], d.sx, d.sy-20-bob, 1, 1, 0, nil)
	if p.Acting && p.Dir != sim.DirUp {
		g.drawTool(screen, p, d, t, bob)
	}
}

func (g *Game) drawTool(screen *ebiten.Image, p protocol.PlayerView, d drawable, t, bob float64) {
	side := 1.0
	if p.Dir == sim.DirLeft {
		side = -1
	}
	swing := (0.9 + 0.7*math.Sin(t*0.5)) * side
	drawAt(screen, g.art.tool, d.sx+11*side, d.sy-12-bob, 1, 1, swing, nil)
}

// target returns the entity the Action key would hit right now, using the
// same rule as the server.
func (g *Game) target() (protocol.EntityView, bool) {
	if g.curr.Self.Dead {
		return protocol.EntityView{}, false
	}
	for _, p := range g.curr.Players {
		if p.ID != g.myID {
			continue
		}
		hasWood := g.curr.Self.Inv[sim.ItemWood] > 0
		id, ok := sim.NearestTarget(float64(p.X), float64(p.Y), hasWood, g.candidateAt)
		if !ok {
			return protocol.EntityView{}, false
		}
		for _, e := range g.curr.Entities {
			if e.ID == id {
				return e, true
			}
		}
	}
	return protocol.EntityView{}, false
}

func (g *Game) candidateAt(tx, ty int) (sim.TargetCandidate, bool) {
	i, ok := g.entAt[[2]int{tx, ty}]
	if !ok {
		return sim.TargetCandidate{}, false
	}
	e := g.curr.Entities[i]
	return sim.TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready}, true
}

// drawWorldLabels draws what must stay readable at night: the target marker
// and other players' names.
func (g *Game) drawWorldLabels(screen *ebiten.Image, cam camera) {
	if e, ok := g.target(); ok {
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		top := sy - g.entitySprite(e).ay - 12 + math.Round(2*math.Sin(float64(g.frame)*0.12))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-6, top)
		screen.DrawImage(g.art.marker, op)
		label := e.Kind.Name()
		if e.Kind == sim.EntityCampfire {
			label += " — thêm củi"
		}
		g.text(screen, label, sx, top-18, colWhite, text.AlignCenter, 1)
	}
	alpha := g.alpha()
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			continue
		}
		wx, wy, _ := g.lerpPlayer(p, alpha)
		sx, sy := cam.toScreen(wx, wy)
		g.text(screen, p.Name, sx, sy-62, colWhite, text.AlignCenter, 1)
	}
}
