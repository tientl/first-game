package sim

import (
	"math"
	"testing"
)

// The themes of the natural caves: see cavethemes.go.

func TestCavesTakeTheirIslandsTheme(t *testing.T) {
	w := genCaves(t, 42, SizeHuge)
	zones := map[CaveZone]int{}
	for _, s := range w.caves.systems {
		k := w.islands[s.island].Kind
		d := s.chambers[s.dest]
		if !d.hall && d.zone != themeOf(k) {
			t.Errorf("island %d (%d): the destination is zone %d, want its own %d", s.island, k, d.zone, themeOf(k))
		}
		for _, c := range s.chambers {
			zones[c.zone]++
		}
	}
	t.Logf("chambers by zone: %v", zones)
	for z := ZoneNest; z < NumCaveZones; z++ {
		if zones[z] == 0 {
			t.Errorf("no chamber of zone %d in a world of 1024", z)
		}
	}
	// Their waters, lava and ice, and what lives in the waters.
	var water, lava, ice, fish, crocs int
	for i, tl := range w.tiles[w.width*w.surfaceH:] {
		_ = i
		switch tl {
		case TileRiver:
			water++
		case TileLava:
			lava++
		case TileIce:
			ice++
		}
	}
	for _, c := range w.fauna.creatures {
		if !w.Underground(int(c.Y)) {
			continue
		}
		switch c.Kind {
		case CreatureFish:
			fish++
		case CreatureCroc:
			crocs++
		}
	}
	t.Logf("water %d, lava %d, ice %d; fish %d, crocodiles %d", water, lava, ice, fish, crocs)
	if water == 0 || lava == 0 || ice == 0 || fish == 0 || crocs == 0 {
		t.Error("the caves lack water, lava, ice, fish or crocodiles")
	}
}

func TestCavesPropsAreWhereTheyBelong(t *testing.T) {
	w := genCaves(t, 42, SizeHuge)
	count := map[CaveProp]int{}
	for _, e := range w.entities {
		if e.Kind != EntityCaveProp {
			continue
		}
		p := CaveProp(e.Var)
		count[p]++
		if !w.TileAt(e.TX, e.TY).Walkable() || w.caveLevel(e.TY) < 0 {
			t.Fatalf("a prop %d at (%d,%d) off the caves' floor", p, e.TX, e.TY)
		}
		z := w.caveCell(e.TX, e.TY).Zone()
		switch p {
		case PropWeb, PropEggSac:
			if z != ZoneNest {
				t.Errorf("a web or egg sac in zone %d", z)
			}
		case PropFossil:
			if z != ZoneFossil || !e.Ready {
				t.Errorf("a fossil in zone %d, ready %v", z, e.Ready)
			}
		case PropPainting:
			if z != ZonePaintings || w.TileAt(e.TX, e.TY-1) != TileMountain {
				t.Errorf("a painting in zone %d with no wall above it", z)
			}
		case PropSarcophagus:
			if z != ZoneTomb {
				t.Errorf("the coffin in zone %d", z)
			}
		}
	}
	t.Logf("props: %v", count)
	for p := PropCrystal; p < NumCaveProps; p++ {
		if count[p] == 0 {
			t.Errorf("no prop %d in a world of 1024", p)
		}
	}
}

// caveTestPlayer puts a player in the caves at (x, y) of w.
func caveTestPlayer(w *World, x, y float64) *Player {
	id := w.AddPlayer("caver")
	p := w.players[id]
	p.X, p.Y = x, y
	return p
}

func TestCavesLavaBurnsWebsHoldBonesDigOut(t *testing.T) {
	w := genStyle(t, StyleCaves, 42, SizeLarge)
	// Lava: a player standing against it is burnt.
	var lavaAt, webAt, fossilAt, paintAt [2]int
	var found [4]bool
	for i, tl := range w.tiles {
		x, y := i%w.width, i/w.width
		if !found[0] && tl == TileLava && w.caveLevel(y) >= 0 && w.TileAt(x, y+1).Walkable() && !w.blocked(x, y+1) {
			lavaAt, found[0] = [2]int{x, y + 1}, true
		}
	}
	for _, e := range w.entities {
		if e.Kind != EntityCaveProp {
			continue
		}
		switch CaveProp(e.Var) {
		case PropWeb:
			webAt, found[1] = [2]int{e.TX, e.TY}, true
		case PropFossil:
			fossilAt, found[2] = [2]int{e.TX, e.TY}, true
		case PropPainting:
			paintAt, found[3] = [2]int{e.TX, e.TY}, true
		}
	}
	if found != [4]bool{true, true, true, true} {
		t.Fatalf("lava, a web, a fossil, a painting: found %v", found)
	}
	p := caveTestPlayer(w, float64(lavaAt[0])+0.5, float64(lavaAt[1])+0.35)
	before := p.Health
	stepN(w, TicksPerSecond)
	if p.Health >= before {
		t.Errorf("health %.1f after a second against lava, %.1f before", p.Health, before)
	}
	// A web: walked into, it holds the player and is torn.
	p.X, p.Y, p.Health = float64(webAt[0])+0.5, float64(webAt[1])+0.5, MaxHealth
	stepN(w, 1)
	if !p.Webbed() {
		t.Error("a web did not hold the player")
	}
	if e, ok := w.EntityAt(webAt[0], webAt[1]); ok && e.Kind == EntityCaveProp {
		t.Error("the web was not torn")
	}
	// A fossil gives its bones once.
	fe := w.entityAt(fossilAt[0], fossilAt[1])
	if !w.touchProp(p, fe) || p.Notice != NoticeFossil {
		t.Fatal("the fossil gave nothing")
	}
	if w.touchProp(p, fe) {
		t.Error("the fossil gave its bones twice")
	}
	bones := 0
	for _, d := range w.drops {
		if d.Item == ItemBone && math.Hypot(d.X-float64(fossilAt[0]), d.Y-float64(fossilAt[1])) < 3 {
			bones += d.Count
		}
	}
	if bones < caveFossilBones {
		t.Errorf("%d bones dug out, want %d", bones, caveFossilBones)
	}
	// A painting tells what it shows.
	pe := w.entityAt(paintAt[0], paintAt[1])
	if !w.touchProp(p, pe) || p.Notice != NoticePainting {
		t.Error("the painting showed nothing")
	}
	if pic, dir := PaintingOf(int(p.NoticeArg)); pic >= numPaintings || dir > 7 {
		t.Errorf("painting %d facing %d", pic, dir)
	}
}
