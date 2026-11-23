// Command shot renders one frame of the client against a fake, frozen server
// and saves it as a PNG. It is a dev tool for checking art changes:
//
//	go run ./cmd/shot -o day.png
//	go run ./cmd/shot -o night.png -time 0.85
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func main() {
	out := flag.String("o", "shot.png", "output PNG")
	seed := flag.Uint64("seed", 42, "world seed")
	dayFrac := flag.Float64("time", 0.3, "time of day in [0, 1)")
	frames := flag.Int("frames", 40, "frames to run before the capture")
	acting := flag.Bool("acting", true, "show the player swinging a tool")
	axe := flag.Bool("axe", true, "give the player an axe")
	coast := flag.Bool("coast", false, "stand on the shore nearest to the spawn (open world: a long sandy beach with clams)")
	rocks := flag.Bool("rocks", false, "stand among the most boulders")
	dir := flag.Int("dir", int(sim.DirRight), "facing: 0 down, 1 up, 2 left, 3 right")
	menu := flag.Bool("menu", false, "show the main menu instead of the game")
	audioSettings := flag.String("audio", "", "start the sound system with this settings file")
	crop := flag.String("crop", "", "x,y,w,h region to keep, enlarged 4x")
	overlay := flag.String("overlay", "", "open the map or bag: map, bag")
	animals := flag.Bool("animals", false, "put one of each animal (fish with -coast) next to the player")
	held := flag.Int("held", -1, "item kind the player holds (default: the selected slot's)")
	fishing := flag.String("fishing", "", "show the player fishing: wait or bite (use with -coast)")
	builds := flag.Bool("builds", false, "put a wooden wall and a bed nearby, with the other player asleep in it")
	farm := flag.Bool("farm", false, "put a row of tilled soil and growing sweet potatoes nearby, and a hoe in hand")
	furnace := flag.Bool("furnace", false, "put a lit furnace and one of each ore nearby, and a copper pickaxe in hand")
	open := flag.Bool("open", false, "an open world instead of the 96x96 island")
	sizes := flag.Bool("sizes", false, "with -menu: show the world-size picker")
	size := flag.Int("size", 256, "open world size")
	at := flag.String("at", "", "open world: stand at desert, marsh, cave, mountain, ash or lava")
	female := flag.Bool("female", false, "the player is a woman")
	hair := flag.Int("hair", 0, "the player's hair style, 0 to 5")
	hairColor := flag.Int("haircolor", 0, "the player's hair colour, 0 to 7")
	skin := flag.Int("skin", 0, "the player's skin tone, 0 to 5")
	create := flag.Bool("create", false, "show the character screen of a new game")
	croc := flag.Bool("croc", false, "put a crocodile basking next to the player, and one lurking in the nearest pond")
	flag.Parse()

	N := 96
	style := sim.StyleIsland
	if *open {
		N, style = *size, sim.StyleOpen
	}
	w, err := sim.Generate(sim.GenConfig{Seed: *seed, Width: N, Height: N, Style: style})
	if err != nil {
		log.Fatal(err)
	}
	sx, sy := w.Spawn()
	if *coast {
		tiles := w.Tiles()
		// In an open world, a beach by the sea rather than a marsh pond:
		// sand, open water below, and a few clam beds in view if any.
		var clams [][2]int
		w.EachEntityIn(0, 0, N-1, N-1, func(e sim.Entity) {
			if e.Kind == sim.EntityClam {
				clams = append(clams, [2]int{e.TX, e.TY})
			}
		})
		seaside := func(tx, ty int) bool {
			if !*open {
				return true
			}
			if tiles[ty*N+tx] != sim.TileSand || ty+4 >= N || tx < 10 || tx >= N-10 {
				return false
			}
			for x := tx - 10; x <= tx+10; x++ { // a long shore across the screen
				if tiles[(ty+4)*N+x] != sim.TileWater || tiles[(ty-1)*N+x] == sim.TileWater {
					return false
				}
			}
			return tiles[(ty+2)*N+tx] == sim.TileWater
		}
		clamsNear := func(tx, ty int) int {
			n := 0
			for _, c := range clams {
				if abs(c[0]-tx) <= 10 && abs(c[1]-ty) <= 5 {
					n++
				}
			}
			return n
		}
		best := math.Inf(1)
		bx, by := sx, sy
		for _, minClams := range []int{3, 0} {
			for ty := 1; ty < N-1; ty++ {
				for tx := 0; tx < N; tx++ {
					if tiles[ty*N+tx] != sim.TileWater && tiles[(ty+1)*N+tx] == sim.TileWater && seaside(tx, ty) {
						d := math.Hypot(float64(tx)-sx, float64(ty)-sy)
						if d < best && (!*open || clamsNear(tx, ty) >= minClams) {
							best, bx, by = d, float64(tx)+0.5, float64(ty)-1.5
						}
					}
				}
			}
			if !math.IsInf(best, 1) {
				break
			}
		}
		sx, sy = bx, by
	}
	if *rocks {
		var boulders []sim.Entity
		w.EachEntityIn(0, 0, N-1, N-1, func(e sim.Entity) {
			if e.Kind == sim.EntityBoulder {
				boulders = append(boulders, e)
			}
		})
		most := -1
		for _, b := range boulders {
			n := 0
			for _, o := range boulders {
				if abs(o.TX-b.TX) <= 6 && abs(o.TY-b.TY) <= 4 {
					n++
				}
			}
			if n > most {
				most, sx, sy = n, float64(b.TX)+0.5, float64(b.TY)+1.5
			}
		}
	}
	if *at != "" {
		want := map[string]sim.Tile{"desert": sim.TileDesert, "marsh": sim.TileMarsh, "cave": sim.TileCave,
			"mountain": sim.TileMountain, "ash": sim.TileAsh, "lava": sim.TileLava, "mouth": sim.TileCave}[*at]
		// Desert and marsh: stand well inside, where they fill the screen.
		deep := 0.0
		if *at == "desert" || *at == "marsh" {
			deep = 0.75
		}
		share := func(tx, ty int) float64 {
			n, in := 0, 0
			for y := ty - 8; y <= ty+8; y++ {
				for x := tx - 12; x <= tx+12; x++ {
					n++
					if t := w.TileAt(x, y); t == want || (want == sim.TileMarsh && t == sim.TileWater) {
						in++
					}
				}
			}
			return float64(in) / float64(n)
		}
		best := math.Inf(1)
		bx, by := sx, sy
		for ty := 2; ty < N-4; ty++ {
			for tx := 2; tx < N-2; tx++ {
				if w.TileAt(tx, ty) != want {
					continue
				}
				if deep > 0 && share(tx, ty) < deep {
					continue
				}
				// Stand on the tile, or for impassable ground a little south
				// of it, where it can be seen from.
				px, py := tx, ty
				if *at == "mouth" { // outside, looking at a cave's mouth
					if w.TileAt(tx, ty+1) == sim.TileCave || !w.TileAt(tx, ty+1).Walkable() {
						continue
					}
					py = ty + 3
				}
				if !want.Walkable() {
					py = ty + 3
					if !w.TileAt(px, py).Walkable() || w.TileAt(px, py) == sim.TileMountain {
						continue
					}
				}
				if d := math.Hypot(float64(px)-sx, float64(py)-sy); d < best {
					best, bx, by = d, float64(px)+0.5, float64(py)+0.5
				}
			}
		}
		sx, sy = bx, by
	}
	tick := uint64(sim.DayTicks) + uint64(*dayFrac*sim.DayTicks)

	snap := protocol.Snapshot{
		Tick: tick,
		Self: protocol.SelfState{Health: 72, Hunger: 55, Inv: sim.Inventory{}},
		Players: []protocol.PlayerView{
			{ID: 1, Name: "Tien", X: float32(sx), Y: float32(sy), Dir: sim.Dir(*dir), Acting: *acting, HasAxe: *axe},
			{ID: 2, Name: "Bạn", X: float32(sx) - 2.2, Y: float32(sy) + 1.4, Dir: sim.DirDown},
		},
	}
	if *axe {
		snap.Self.Inv.Add(sim.ItemAxe, 1)
	}
	snap.Self.Inv.Add(sim.ItemWood, 5)
	snap.Self.Inv.Add(sim.ItemStone, 3)
	snap.Self.Inv.Add(sim.ItemBerry, 2)
	snap.Self.Inv.Add(sim.ItemStick, 2)
	snap.Self.Inv[sim.HotbarSlots] = sim.Slot{Item: sim.ItemGrass, Count: 1} // something in the bag
	snap.Self.Inv[sim.HotbarSlots+3] = sim.Slot{Item: sim.ItemMeat, Count: 3}
	snap.Self.Selected = 1
	snap.Players[0].Held = snap.Self.Inv[snap.Self.Selected].Item
	if *held >= 0 {
		snap.Players[0].Held = sim.ItemKind(*held)
	}
	switch *fishing { // the float 3 tiles below: in the sea with -coast
	case "wait", "bite":
		me := &snap.Players[0]
		me.Held, me.Acting, me.Cast = sim.ItemRod, false, sim.CastWaiting
		me.BobX, me.BobY = float32(sx)+0.4, float32(sy)+3.1
		if *fishing == "bite" {
			me.Cast = sim.CastBite
		}
	}
	snap.Players[0].Look = sim.Look{Female: *female, Hair: sim.HairStyle(*hair), HairColor: uint8(*hairColor), Skin: uint8(*skin)}
	snap.Players[1].Held = sim.ItemBerry
	if snap.Players[0].Held == sim.ItemTorch {
		snap.Self.TorchFuel = sim.TorchBurnTicks * 2 / 3
	}
	snap.Self.Inv.Add(sim.ItemTorch, 2)
	snap.Self.Inv.Add(sim.ItemSpear, 1)

	cx, cy := int(math.Floor(sx)), int(math.Floor(sy))
	n := 0
	w.EachEntityIn(cx-20, cy-20, cx+20, cy+20, func(e sim.Entity) {
		n++
		ready := e.Ready
		if n%9 == 0 { // show some harvested states too
			ready = false
		}
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID: e.ID, Kind: e.Kind, TX: int16(e.TX), TY: int16(e.TY), Ready: ready, Big: e.Big,
		})
	})
	w.EachCreatureIn(sx-20, sy-20, sx+20, sy+20, func(c sim.Creature) {
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: c.ID, Kind: c.Kind, X: float32(c.X), Y: float32(c.Y), State: c.State, FaceLeft: c.FaceLeft,
		})
	})
	if *builds {
		cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
		id := sim.EntityID(1 << 29)
		for i := int16(-4); i <= -1; i++ {
			id++
			snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: sim.EntityWall, TX: cx + i, TY: cy - 3, Ready: true})
		}
		// Clear room around the bed, so nothing generated stands in front of it.
		bx, by := cx+1, cy+2
		kept := snap.Entities[:0]
		for _, e := range snap.Entities {
			if e.Kind == sim.EntityWall || e.TX < bx-2 || e.TX > bx+2 || e.TY < by-1 || e.TY > by+3 {
				kept = append(kept, e)
			}
		}
		snap.Entities = kept
		id++
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: sim.EntityBed, TX: bx, TY: by, Ready: true})
		snap.Players[1].X, snap.Players[1].Y = float32(bx)+0.5, float32(by)+0.5
		snap.Players[1].Asleep = true
	}
	if *furnace {
		cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
		kept := snap.Entities[:0]
		for _, e := range snap.Entities {
			if e.TX < cx-6 || e.TX > cx+1 || e.TY < cy-3 || e.TY > cy+3 {
				kept = append(kept, e)
			}
		}
		snap.Entities = kept
		id := sim.EntityID(1 << 27)
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: sim.EntityFurnace, TX: cx - 2, TY: cy - 2, Ready: true, Fuel: 600})
		for i, k := range []sim.EntityKind{sim.EntityCopperOre, sim.EntityIronOre, sim.EntityPlatinumOre, sim.EntityDiamondOre} {
			id++
			snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: k, TX: cx - 6 + 2*int16(i), TY: cy + 2, Ready: true})
		}
		snap.Players[0].Held, snap.Players[0].Acting = sim.ItemCopperPickaxe, *acting
		snap.Self.Inv.Add(sim.ItemCopperPickaxe, 1)
		snap.Self.Inv.Add(sim.ItemIronOre, 3)
	}
	if *farm {
		cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
		kept := snap.Entities[:0]
		for _, e := range snap.Entities {
			if e.TX < cx-6 || e.TX > cx+1 || e.TY < cy-3 || e.TY > cy+1 {
				kept = append(kept, e)
			}
		}
		snap.Entities = kept
		id := sim.EntityID(1 << 28)
		for i := range int16(sim.CropStages + 2) {
			id++
			e := protocol.EntityView{ID: id, Kind: sim.EntityCrop, TX: cx - 5 + i, TY: cy - 2}
			switch {
			case i == 0:
				e.Kind, e.Ready = sim.EntityFarmland, true
			case i == sim.CropStages+1:
				e.Ready = true // ripe
			default:
				e.Stage = uint8(i - 1)
			}
			snap.Entities = append(snap.Entities, e)
		}
		snap.Players[0].Held, snap.Players[0].Acting = sim.ItemHoe, *acting
		snap.Self.Inv.Add(sim.ItemPotato, 2)
		snap.Self.Inv.Add(sim.ItemHoe, 1)
		snap.Self.Inv.Add(sim.ItemPebble, 3)
		snap.Self.Inv.Add(sim.ItemRoastPotato, 1)
	}
	if *animals {
		id := sim.CreatureID(1 << 30)
		put := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
			id++
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: id, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
			})
		}
		put(sim.CreatureRabbit, 1.4, -1.5, sim.CreatureIdle, false)
		put(sim.CreatureRabbit, -1.4, -1.1, sim.CreatureSleep, true)
		put(sim.CreatureSnake, 2.4, 1.6, sim.CreatureChase, true)
		put(sim.CreatureSnake, 3.6, -0.6, sim.CreatureStrike, true)
		put(sim.CreatureWorm, 0.7, 1.3, sim.CreatureWander, false)
		put(sim.CreatureSnake, 4.3, 0.9, sim.CreatureIdle, false) // coiled by its den
		// A burrow behind the sleeping rabbit and a den by the coiled snake.
		snap.Entities = append(snap.Entities,
			protocol.EntityView{ID: 1<<30 + 1, Kind: sim.EntityBurrow, TX: int16(cx - 2), TY: int16(cy - 2), Ready: true},
			protocol.EntityView{ID: 1<<30 + 2, Kind: sim.EntityDen, TX: int16(cx + 5), TY: int16(cy + 1), Ready: true})
		if *coast { // the sea starts 2.5 tiles below the player
			put(sim.CreatureFish, 0.4, 3.1, sim.CreatureWander, false)
			put(sim.CreatureFish, -1.9, 3.4, sim.CreatureIdle, true)
			put(sim.CreatureCrab, -0.8, 2.1, sim.CreatureIdle, false)   // on the beach
			put(sim.CreatureCrab, 1.6, 2.0, sim.CreatureChase, false)   // claws up
			put(sim.CreatureCrab, -2.6, 2.9, sim.CreatureHidden, false) // gone under: bubbles
		}
	}

	if *croc {
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: 1<<29 + 1, Kind: sim.CreatureCroc, X: float32(sx + 4.4), Y: float32(sy + 2.3), State: sim.CreatureIdle, FaceLeft: false,
		})
		// The lurker: open water nearest the player, with water all round.
		best := math.Inf(1)
		var lx, ly float64
		for ty := cy - 6; ty <= cy+6; ty++ {
			for tx := cx - 10; tx <= cx+10; tx++ {
				open := true
				for _, d := range [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					open = open && w.TileAt(tx+d[0], ty+d[1]) == sim.TileWater
				}
				if d := math.Hypot(float64(tx-cx), float64(ty-cy)); open && d < best {
					best, lx, ly = d, float64(tx)+0.5, float64(ty)+0.5
				}
			}
		}
		if !math.IsInf(best, 1) {
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: 1<<29 + 2, Kind: sim.CreatureCroc, X: float32(lx), Y: float32(ly), State: sim.CreatureLurk, FaceLeft: true,
			})
		}
	}

	// A campfire next to the spawn, so lighting is visible at night.
	snap.Entities = append(snap.Entities, protocol.EntityView{
		ID: 1 << 30, Kind: sim.EntityCampfire, TX: int16(cx + 2), TY: int16(cy), Ready: true,
		Fuel: sim.FireStartFuel,
	})

	conn := &fakeConn{ch: make(chan protocol.ServerMsg, 4)}
	conn.ch <- protocol.ServerMsg{Kind: protocol.ServerWelcome, Welcome: protocol.Welcome{
		PlayerID: 1, Width: N, Height: N, Tiles: w.Tiles(),
		Explored: shotExplored(N, sx, sy),
	}}
	conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: snap}

	if *audioSettings != "" {
		client.InitAudio(*audioSettings)
	}
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("shot")
	game := client.New(conn, "Tien")
	game.ShowOverlay(*overlay)
	var inner ebiten.Game = game
	if *create {
		m := client.NewMenu(client.SaveInfo{})
		m.ShowCharacterScreen()
		inner = &menuGame{m}
	}
	if *menu {
		m := client.NewMenu(client.SaveInfo{Exists: true, Day: 3, Name: "Tien", SavedAt: time.Now(), Size: 512})
		if *sizes {
			m.ShowSizePicker()
		}
		inner = &menuGame{m}
	}
	g := &shooter{inner: inner, frames: *frames, out: *out, crop: *crop}
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if g.err != nil {
		log.Fatal(g.err)
	}
	log.Printf("wrote %s", *out)
}

// shotExplored is a wander out from (x, y), so the map shows some fog.
func shotExplored(n int, x, y float64) []byte {
	seen := sim.NewExplored(n, n)
	for i := range 70 {
		a := float64(i) * 0.19
		seen.Reveal(x+math.Sin(a)*float64(i)*0.32, y+math.Cos(a*0.8)*float64(i)*0.22)
	}
	return seen.Bytes()
}

type fakeConn struct{ ch chan protocol.ServerMsg }

func (c *fakeConn) Send(protocol.ClientMsg) error       { return nil }
func (c *fakeConn) Messages() <-chan protocol.ServerMsg { return c.ch }
func (c *fakeConn) Close() error                        { return nil }

type shooter struct {
	inner  ebiten.Game
	frames int
	n      int
	out    string
	crop   string
	done   bool
	err    error
}

func (s *shooter) Update() error {
	if s.done {
		return ebiten.Termination
	}
	s.n++
	return s.inner.Update()
}

func (s *shooter) Draw(screen *ebiten.Image) {
	s.inner.Draw(screen)
	if s.n < s.frames || s.done {
		return
	}
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	var final image.Image = img
	var cx, cy, cw, ch int
	if n, _ := fmt.Sscanf(s.crop, "%d,%d,%d,%d", &cx, &cy, &cw, &ch); n == 4 {
		const z = 4
		big := image.NewRGBA(image.Rect(0, 0, cw*z, ch*z))
		for y := 0; y < ch*z; y++ {
			for x := 0; x < cw*z; x++ {
				big.Set(x, y, img.At(cx+x/z, cy+y/z))
			}
		}
		final = big
	}
	f, err := os.Create(s.out)
	if err == nil {
		err = png.Encode(f, final)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	s.err, s.done = err, true
}

func (s *shooter) Layout(w, h int) (int, int) { return s.inner.Layout(w, h) }

type menuGame struct{ m *client.Menu }

func (m *menuGame) Update() error              { m.m.Update(); return nil }
func (m *menuGame) Draw(s *ebiten.Image)       { m.m.Draw(s) }
func (m *menuGame) Layout(w, h int) (int, int) { return m.m.Layout(w, h) }

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
