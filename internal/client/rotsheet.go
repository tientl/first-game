package client

import (
	"image"
	"image/color"
	"slices"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The sheet of what turns (facing.go), for a look at it outside the game:
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -rotsheet sheet.png
//
// draws, through the game's own drawing at 1080p's density, every piece
// that turns in each way it faces (south, west, north, east: the order R
// turns them), sleepers in beds and tents each way, and walls, stone walls
// and fences joined into lines, corners, T's and crosses, with doors in
// them both ways. It must run inside the game loop (Draw).

// rotCell is one picture of the sheet: things standing round tile (0, 0).
type rotCell struct {
	name    string
	ents    []protocol.EntityView // TX, TY from the cell's middle tile
	sleeper bool                  // someone asleep on the middle tile
	lit     bool
}

// rotRow is a row of the sheet.
type rotRow struct {
	name  string
	cells []rotCell
}

// facingNames are the facings in the order R turns them, as the sheet
// labels them.
var rotOrder = []struct {
	d    sim.Dir
	name string
}{{sim.DirDown, "south"}, {sim.DirLeft, "west"}, {sim.DirUp, "north"}, {sim.DirRight, "east"}}

// rotRows lays out the sheet.
func rotRows() []rotRow {
	one := func(k sim.EntityKind, v uint8, d sim.Dir, mod func(*protocol.EntityView)) []protocol.EntityView {
		e := protocol.EntityView{Kind: k, Var: v, Face: d, Ready: true}
		if mod != nil {
			mod(&e)
		}
		return []protocol.EntityView{e}
	}
	four := func(name string, k sim.EntityKind, v uint8, mod func(*protocol.EntityView)) rotRow {
		r := rotRow{name: name}
		for _, o := range rotOrder {
			if sim.FacingOf(k, v, o.d) != o.d {
				continue
			}
			r.cells = append(r.cells, rotCell{name: o.name, ents: one(k, v, o.d, mod)})
		}
		return r
	}
	lit := func(e *protocol.EntityView) { e.Fuel = 1000 }
	meat := func(e *protocol.EntityView) { e.Cook = sim.CookFlagLoaded }
	fuel := func(e *protocol.EntityView) { e.Fuel = sim.FireStartFuel; e.Cook = sim.CookFlagLoaded }
	rows := []rotRow{
		four("door", sim.EntityDoor, 0, nil),
		four("bed", sim.EntityBed, 0, nil),
		four("leaf tent", sim.EntityLeafTent, 0, nil),
		four("chest", sim.EntityChest, 0, nil),
		four("furnace (lit)", sim.EntityFurnace, 0, lit),
		four("oven (lit)", sim.EntityOven, 0, lit),
		four("mortar", sim.EntityMortar, 0, meat),
		four("drying rack", sim.EntityDryRack, 0, meat),
	}
	r := four("grill", sim.EntityCampfire, uint8(sim.GearGrill), fuel)
	t := four("head on a post", sim.EntityTrophy, uint8(sim.CreatureTrike), nil)
	r.name, r.cells = "grill, head", append(r.cells, t.cells...)
	rows = append(rows, r)
	// Sleepers, each way.
	sl := rotRow{name: "asleep"}
	for _, k := range []sim.EntityKind{sim.EntityBed, sim.EntityLeafTent} {
		for _, o := range rotOrder {
			if k == sim.EntityLeafTent && (o.d == sim.DirUp) {
				continue // shut in: nothing of them shows
			}
			sl.cells = append(sl.cells, rotCell{name: o.name, ents: one(k, 0, o.d, nil), sleeper: true})
		}
	}
	rows = append(rows, sl)
	// Lines.
	for _, k := range []sim.EntityKind{sim.EntityWall, sim.EntityStoneWall, sim.EntitySpikes} {
		name := map[sim.EntityKind]string{sim.EntityWall: "palisade", sim.EntityStoneWall: "stone wall", sim.EntitySpikes: "stake fence"}[k]
		at := func(es *[]protocol.EntityView, kk sim.EntityKind, d sim.Dir, pts ...[2]int16) {
			for _, p := range pts {
				*es = append(*es, protocol.EntityView{Kind: kk, Face: d, TX: p[0], TY: p[1], Ready: true})
			}
		}
		row := rotRow{name: name}
		var a, b, c, d, e, f, h, i []protocol.EntityView
		at(&a, k, sim.DirDown, [2]int16{0, 0})
		at(&b, k, sim.DirRight, [2]int16{0, 0})
		at(&c, k, sim.DirDown, [2]int16{-1, 0}, [2]int16{0, 0}, [2]int16{1, 0})
		at(&d, k, sim.DirDown, [2]int16{0, -1}, [2]int16{0, 0}, [2]int16{0, 1})
		at(&e, k, sim.DirDown, [2]int16{0, -1}, [2]int16{0, 0}, [2]int16{1, 0}, [2]int16{-1, 1}, [2]int16{-1, 0})
		at(&f, k, sim.DirDown, [2]int16{-1, 0}, [2]int16{0, 0}, [2]int16{1, 0}, [2]int16{0, -1}, [2]int16{0, 1})
		at(&h, k, sim.DirDown, [2]int16{-1, 0}, [2]int16{1, 0})
		at(&h, sim.EntityDoor, sim.DirDown, [2]int16{0, 0})
		at(&i, k, sim.DirDown, [2]int16{0, -1}, [2]int16{0, 1})
		at(&i, sim.EntityDoor, sim.DirRight, [2]int16{0, 0})
		row.cells = []rotCell{{name: "alone", ents: a}, {name: "alone turned", ents: b}, {name: "a line", ents: c},
			{name: "going back", ents: d}, {name: "corners", ents: e}, {name: "a cross", ents: f},
			{name: "door", ents: h}, {name: "door turned", ents: i}}
		rows = append(rows, row)
	}
	return rows
}

// The sheet's cells, art pixels.
const (
	rotCellW, rotCellH = 96, 104
	rotLabelW          = 70
	rotGround          = 70 // the middle tile's ground point, down from a cell's top
)

// RotateSheet draws the sheet of what turns.
func (g *Game) RotateSheet() *image.RGBA {
	if g.art == nil {
		g.art = getArt()
	}
	rows := rotRows()
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r.cells))
	}
	w, h := rotLabelW+cols*rotCellW, len(rows)*rotCellH+12
	buf := newFineBuf(w, h)
	bg := color.RGBA{86, 124, 70, 255}
	fillRect(buf, 0, 0, float32(w), float32(h), bg, false)
	keepEnts, keepAt, keepPlayers := g.curr.Entities, g.entAt, g.curr.Players
	defer func() { g.curr.Entities, g.entAt, g.curr.Players = keepEnts, keepAt, keepPlayers }()
	const ox, oy = 30, 30 // the tile cells stand round, in the world
	var labels []struct {
		x, y int
		s    string
	}
	for ri, r := range rows {
		y0 := float64(ri*rotCellH + 12)
		labels = append(labels, struct {
			x, y int
			s    string
		}{4 * HDDensity, int(y0+rotGround-20) * HDDensity, r.name})
		for ci, c := range r.cells {
			x0 := float64(rotLabelW + ci*rotCellW)
			fillRect(buf, float32(x0)+1, float32(y0)+1, rotCellW-2, rotCellH-2, color.RGBA{96, 136, 78, 255}, false)
			labels = append(labels, struct {
				x, y int
				s    string
			}{int(x0+3) * HDDensity, int(y0+3) * HDDensity, c.name})
			g.drawRotCell(buf, c, x0+rotCellW/2, y0+rotGround, ox, oy)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, w*HDDensity, h*HDDensity))
	buf.img.ReadPixels(img.Pix)
	for _, l := range labels {
		label(img, l.x, l.y, l.s)
	}
	return img
}

// drawRotCell draws cell c with its middle tile's ground point at (cx, cy)
// on dst, standing it on tile (ox, oy) of the world.
func (g *Game) drawRotCell(dst drawTarget, c rotCell, cx, cy float64, ox, oy int) {
	ents := slices.Clone(c.ents)
	g.curr.Entities = g.curr.Entities[:0:0]
	g.entAt = map[[2]int]int{}
	for i := range ents {
		e := &ents[i]
		e.ID = sim.EntityID(1<<29 + i)
		e.TX, e.TY = e.TX+int16(ox), e.TY+int16(oy)
		g.curr.Entities = append(g.curr.Entities, *e)
		g.entAt[[2]int{int(e.TX), int(e.TY)}] = i
		if ti := int(e.TY)*g.width + int(e.TX); ti >= 0 && ti < len(g.tiles) {
			g.tiles[ti] = sim.TileGrass
		}
	}
	slices.SortStableFunc(ents, func(a, b protocol.EntityView) int { return int(a.TY) - int(b.TY) })
	pos := func(e protocol.EntityView) (float64, float64) {
		return cx + float64(int(e.TX)-ox)*TileW, cy + float64(int(e.TY)-oy)*TileH
	}
	for _, e := range ents {
		x, y := pos(e)
		if w := g.shadowWidth(e); w > 0 {
			g.drawShadow(dst, x, y, w)
		}
	}
	var sleeper protocol.PlayerView
	if c.sleeper {
		sleeper = protocol.PlayerView{ID: 9001, Name: "Ngủ", X: float32(ox) + 0.5, Y: float32(oy) + 0.5, Asleep: true,
			Look: sim.Look{Hair: 2, HairColor: 1}}
		g.curr.Players = []protocol.PlayerView{sleeper}
		st := g.moveState(sleeper.ID)
		st.asleepAt, st.wokeAt = g.frame-5000, never
	} else {
		g.curr.Players = nil
	}
	for _, e := range ents {
		x, y := pos(e)
		g.drawEntity(dst, e, drawable{y: float64(e.TY) + 0.5, id: uint32(e.ID), sx: x, sy: y})
		if c.sleeper && int(e.TX) == ox && int(e.TY) == oy {
			g.drawPlayer(dst, sleeper, drawable{y: float64(e.TY) + 0.5, isPlayer: true, sx: x, sy: y, wx: float64(ox) + 0.5, wy: float64(oy) + 0.5})
		}
	}
}
