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
	"slices"
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
	at := flag.String("at", "", "open world: stand at desert, marsh, pond (a swamp with ponds), cave, mountain, ash or lava")
	female := flag.Bool("female", false, "the player is a woman")
	hair := flag.Int("hair", 0, "the player's hair style, 0 to 5")
	hairColor := flag.Int("haircolor", 0, "the player's hair colour, 0 to 7")
	skin := flag.Int("skin", 0, "the player's skin tone, 0 to 5")
	create := flag.Bool("create", false, "show the character screen of a new game")
	croc := flag.Bool("croc", false, "put a crocodile basking next to the player, and one lurking in the nearest pond")
	dinos := flag.Bool("dinos", false, "put one of each dinosaur around the player, in different poses (sea reptiles with -coast)")
	ice := flag.Bool("ice", false, "put a mammoth herd, a charging mammoth and a stalking sabre-tooth around the player")
	hell := flag.Bool("hell", false, "put one of each of the tyrant's neighbours around the player (pond crocodiles with -croc)")
	eat := flag.Bool("eat", false, "with -dinos, -hell or -ice: the idle beasts are eating instead")
	ride := flag.Bool("ride", false, "the player rides a tamed three-horn, with a pet raptor, someone else's waiting plate-back and a beaten raptor nearby")
	tribal := flag.Bool("tribal", false, "put the tribal age's things round the player: cave beasts, a warrior, a body, a trophy, flowers, glowing plants, a man-eater, buildings, a chest and piles on the ground")
	weather := flag.String("weather", "", "the weather: rain, wind, storm, tornado, snow")
	season := flag.String("season", "", "the season: spring, summer, autumn, winter")
	zoom := flag.Int("zoom", 0, "the view's zoom, -3 to 3")
	fight := flag.Bool("fight", false, "show the beasts' health: a few just hurt, with the damage floating up (use with -dinos)")
	woodsFlag := flag.Bool("woods", false, "put the woods round the player: a giant tree and a tree with beehives, banana trees, a troop of monkeys, and bees after the player")
	dung := flag.Bool("dung", false, "with -farm: a fed bed, heaps of dung by a grazing three-horn, and dung in hand")
	sea := flag.Bool("sea", false, "out at sea: the player sailing, a raft, a swimmer, a mosasaur and a giant squid")
	armor := flag.Int("armor", 0, "the armour tier the player wears, 0 (none) to 4")
	punch := flag.String("punch", "", "a blow by hand at a rabbit put in front of the player: jab, or cross for the other hand; nothing is held unless -held says so")
	hurtFlag := flag.Bool("hurt", false, "the player has just been struck")
	blowAge := flag.Int("age", -1, "with -punch or -hurt: how many frames into the blow or the jolt to show")
	flag.Parse()
	client.SetZoom(*zoom)

	N := 96
	style := sim.StyleIsland
	if *open {
		N, style = *size, sim.StyleLatest
	}
	w, err := sim.Generate(sim.GenConfig{Seed: *seed, Width: N, Height: N, Style: style})
	if err != nil {
		log.Fatal(err)
	}
	_, mapH := w.Size() // taller than N when caves lie under the islands
	surfH := 0
	if w.SurfaceHeight() < mapH {
		surfH = w.SurfaceHeight()
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
	if *sea {
		// Open water a few tiles off the home island.
		best := math.Inf(1)
		tiles := w.Tiles()
		for ty := 8; ty < N-8; ty++ {
			for tx := 8; tx < N-8; tx++ {
				open := true
				for y := ty - 5; y <= ty+5 && open; y++ {
					for x := tx - 9; x <= tx+9; x++ {
						if tiles[y*N+x] != sim.TileWater {
							open = false
							break
						}
					}
				}
				if d := math.Hypot(float64(tx)-sx, float64(ty)-sy); open && d < best {
					best, *at = d, ""
					sx, sy = float64(tx)+0.5, float64(ty)+0.5
				}
			}
		}
	}
	if *at == "cavemouth" || *at == "cavein" {
		k := sim.EntityCaveMouth
		if *at == "cavein" {
			k = sim.EntityCaveExit
		}
		best := math.Inf(1)
		px, py := sx, sy
		w.EachEntityIn(0, 0, 1<<14, 1<<14, func(e sim.Entity) {
			if d := math.Hypot(float64(e.TX)-px, float64(e.TY)-py); e.Kind == k && d < best {
				best = d
				sx, sy = float64(e.TX)+0.5, float64(e.TY)+2.5
				if k == sim.EntityCaveExit {
					sy = float64(e.TY) + 1.5 // where the way down arrives
				}
			}
		})
		*at = ""
	}
	if *at == "pyramid" {
		if tx, ty, ok := w.Pyramid(); ok {
			sx, sy = float64(tx)+0.5, float64(ty)+4.5
		}
		*at = ""
	}
	if *at == "village" {
		if vs := w.Villages(); len(vs) > 0 {
			sx, sy = vs[0].X, vs[0].Y+3
		}
		*at = ""
	}
	if *at == "path" {
		best := math.Inf(1)
		for i, t := range w.Tiles() {
			if t != sim.TilePath {
				continue
			}
			x, y := float64(i%N)+0.5, float64(i/N)+0.5
			if d := math.Hypot(x-sx, y-sy); d > 12 && d < best {
				best, *at = d, ""
				sx, sy = x, y
			}
		}
		*at = ""
	}
	if *at != "" {
		want := map[string]sim.Tile{"desert": sim.TileDesert, "marsh": sim.TileMarsh, "cave": sim.TileCave,
			"mountain": sim.TileMountain, "ash": sim.TileAsh, "lava": sim.TileLava, "mouth": sim.TileCave, "snow": sim.TileSnow,
			"pond": sim.TileMarsh}[*at]
		// Desert and marsh: stand well inside, where they fill the screen.
		deep := 0.0
		if *at == "desert" || *at == "marsh" || *at == "snow" || *at == "pond" {
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
		ponds := func(tx, ty int) float64 {
			n, in := 0, 0
			for y := ty - 8; y <= ty+8; y++ {
				for x := tx - 12; x <= tx+12; x++ {
					n++
					switch w.TileAt(x, y) {
					case sim.TileWater:
						in++
					case sim.TileSand:
						return 0 // a beach: that is the sea
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
				if *at == "pond" && ponds(tx, ty) < 0.12 {
					continue // a swamp with ponds all around
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
	snap.Players[0].Armor = sim.Tier(*armor)
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
			ID: e.ID, Kind: e.Kind, TX: int16(e.TX), TY: int16(e.TY), Ready: ready, Big: e.Big, Var: e.Var, Fuel: int32(e.Fuel),
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
		if *dung {
			snap.Entities[len(snap.Entities)-sim.CropStages-2].Big = true // the empty bed, fed
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: 1 << 29, Kind: sim.CreatureTrike, X: float32(sx + 3.2), Y: float32(sy + 2.4), State: sim.CreatureEat, FaceLeft: true,
			})
			for i, o := range [][2]float64{{4.6, 2.7}, {2.2, 3.5}, {1.2, 1.4}} {
				snap.Drops = append(snap.Drops, protocol.DropView{ID: sim.DropID(900 + i), Item: sim.ItemDung, Count: 2,
					X: float32(sx + o[0]), Y: float32(sy + o[1]), Age: 999})
			}
			snap.Players[0].Held = sim.ItemDung
			snap.Self.Inv.Add(sim.ItemDung, 4)
		}
	}
	if *woodsFlag {
		addWoods(&snap, sx, sy)
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

	if *ride {
		m := sim.CreatureID(1<<27 + 1)
		snap.Creatures = append(snap.Creatures,
			protocol.CreatureView{ID: m, Kind: sim.CreatureTrike, X: float32(sx), Y: float32(sy), Owner: 1, Tamed: true, Ridden: true},
			protocol.CreatureView{ID: m + 1, Kind: sim.CreatureRaptor, X: float32(sx - 3), Y: float32(sy + 0.6), Owner: 1, Tamed: true},
			protocol.CreatureView{ID: m + 2, Kind: sim.CreatureStego, X: float32(sx + 4.5), Y: float32(sy - 2.5), Owner: 2, Tamed: true, Stay: true, FaceLeft: true},
			protocol.CreatureView{ID: m + 3, Kind: sim.CreatureRaptor, X: float32(sx + 3), Y: float32(sy + 2.5), State: sim.CreatureCower, FaceLeft: true})
		snap.Players[0].Mount = m
		snap.Self.Inv.Add(sim.ItemSaddle, 1)
	}
	if *hell {
		id := sim.CreatureID(1 << 25)
		put := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
			id++
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: id, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
			})
		}
		put(sim.CreatureQuetzal, -7.5, -2.5, sim.CreatureIdle, false)
		put(sim.CreatureToro, 4, -4.5, sim.CreatureIdle, true)
		put(sim.CreatureThesc, -3, -5, sim.CreatureIdle, false)
		put(sim.CreatureThesc, -1, -6, sim.CreatureIdle, true)
		put(sim.CreatureAnzu, 6.5, 1.5, sim.CreatureRoar, true)
		put(sim.CreatureOrnitho, -5.5, 3.2, sim.CreatureIdle, false)
		put(sim.CreatureOrnitho, -3.8, 4.2, sim.CreatureRest, false)
		put(sim.CreatureAchero, 2.5, 3.5, sim.CreatureChase, true)
		put(sim.CreatureAchero, 3.6, 4.4, sim.CreatureStrike, true)
		put(sim.CreatureAchero, 4.8, 3.2, sim.CreatureIdle, true)
		put(sim.CreatureQuetzal, 1.5, -2.5, sim.CreatureSoar, false)
	}
	if *ice {
		id := sim.CreatureID(1 << 26)
		put := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
			id++
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: id, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
			})
		}
		put(sim.CreatureMammoth, -6, -3.5, sim.CreatureIdle, false)
		put(sim.CreatureMammoth, -3, -5, sim.CreatureIdle, false)
		put(sim.CreatureMammoth, 4.5, -2.5, sim.CreatureChase, true)
		put(sim.CreatureSabre, -5, 3.2, sim.CreatureChase, false)
		put(sim.CreatureSabre, 5, 3.5, sim.CreatureStrike, true)
	}
	if *dinos {
		id := sim.CreatureID(1 << 28)
		put := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
			id++
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: id, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
			})
		}
		put(sim.CreatureBrachio, -8, -3.5, sim.CreatureIdle, false)
		put(sim.CreatureTrike, -3.5, -5.5, sim.CreatureIdle, false)
		put(sim.CreatureTrike, -1, -6.8, sim.CreatureIdle, true)
		put(sim.CreatureTrike, 1.5, -1.5, sim.CreatureChase, true) // charging, head down
		put(sim.CreatureStego, 5.5, -5.5, sim.CreatureIdle, true)
		put(sim.CreatureRex, 8, 1.5, sim.CreatureRoar, true)
		put(sim.CreatureRaptor, -6.5, 3, sim.CreatureIdle, false)
		put(sim.CreatureRaptor, -4.8, 4.2, sim.CreatureStrike, false)
		put(sim.CreatureRaptor, -7.4, 4.8, sim.CreatureChase, false)
		put(sim.CreaturePtero, 3.5, -3, sim.CreatureSoar, false)
		put(sim.CreaturePtero, 3, 3.5, sim.CreatureIdle, true)
		if *coast {
			put(sim.CreaturePlesio, 1.5, 4.6, sim.CreatureStrike, true)
		}
	}
	if *eat {
		for i := range snap.Creatures {
			if c := &snap.Creatures[i]; c.State == sim.CreatureIdle && c.Kind.Feeds() {
				c.State = sim.CreatureEat
			}
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

	if *tribal {
		addTribal(&snap, sx, sy)
	}
	if *sea {
		snap.Boats = []protocol.BoatView{
			{ID: 1, Kind: sim.BoatSail, X: float32(sx), Y: float32(sy), Heading: 0.2, Speed: 0.25, HP: 110, Rider: 1},
			{ID: 2, Kind: sim.BoatRaft, X: float32(sx - 6), Y: float32(sy + 2.5), Heading: math.Pi, Speed: 0, HP: 40, Held: true},
		}
		snap.Players[0].Boat, snap.Self.Boat = 1, 1
		snap.Players[1].X, snap.Players[1].Y, snap.Players[1].Swimming = float32(sx-3), float32(sy-2), true
		snap.Self.Breath = 1
		snap.WindDir, snap.Wind = 0.3, 0.9
		snap.Creatures = append(snap.Creatures,
			protocol.CreatureView{ID: 1<<28 + 1, Kind: sim.CreatureMosa, X: float32(sx + 4.5), Y: float32(sy + 1), State: sim.CreatureStrike, FaceLeft: true},
			protocol.CreatureView{ID: 1<<28 + 2, Kind: sim.CreatureMosa, X: float32(sx + 2), Y: float32(sy - 4), State: sim.CreatureLurk},
			protocol.CreatureView{ID: 1<<28 + 3, Kind: sim.CreatureKraken, X: float32(sx - 6), Y: float32(sy + 2.6), State: sim.CreatureGrab, FaceLeft: false},
		)
	}
	snap.Season = map[string]sim.Season{"summer": sim.Summer, "autumn": sim.Autumn, "winter": sim.Winter}[*season]
	snap.Weather = map[string]sim.Weather{"rain": sim.WeatherRain, "wind": sim.WeatherWind, "storm": sim.WeatherStorm,
		"tornado": sim.WeatherTornado, "snow": sim.WeatherSnow}[*weather]
	if snap.Weather == sim.WeatherStorm {
		snap.Strikes = []protocol.StrikeView{{X: float32(sx + 5), Y: float32(sy - 2), Age: 1}}
	}
	if snap.Weather == sim.WeatherTornado {
		snap.Tornadoes = []protocol.TornadoView{{ID: 1, X: float32(sx - 6), Y: float32(sy + 1)}}
	}

	// A campfire next to the spawn, so lighting is visible at night.
	if !*sea {
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID: 1 << 30, Kind: sim.EntityCampfire, TX: int16(cx + 2), TY: int16(cy), Ready: true,
			Fuel: sim.FireStartFuel,
		})
	}

	conn := &fakeConn{ch: make(chan protocol.ServerMsg, 8)}
	conn.ch <- protocol.ServerMsg{Kind: protocol.ServerWelcome, Welcome: protocol.Welcome{
		PlayerID: 1, Width: N, Height: mapH, Tiles: w.Tiles(),
		Explored: shotExplored(N, mapH, sx, sy), SurfaceH: surfH,
	}}
	if *fight {
		for i := range snap.Creatures {
			m := int16(snap.Creatures[i].Kind.MaxHP())
			snap.Creatures[i].HP, snap.Creatures[i].MaxHP = m, m
		}
		conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: snap}
		hurt := snap
		hurt.Creatures = slices.Clone(snap.Creatures)
		for i := range hurt.Creatures {
			c := &hurt.Creatures[i]
			switch i % 3 {
			case 0:
				c.HP -= int16(float64(c.MaxHP) * 0.22)
			case 1:
				c.HP -= int16(float64(c.MaxHP) * 0.71)
			}
		}
		hurt.Self.Notice, hurt.Self.NoticeSeq = sim.NoticeCrit, snap.Self.NoticeSeq+1
		hurt.Tick++
		snap = hurt
	}
	if *punch != "" {
		me := &snap.Players[0]
		me.Acting = true
		if *held < 0 {
			me.Held = sim.ItemNone
		}
		dx, dy := me.Dir.Delta()
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: 1<<30 + 99, Kind: sim.CreatureRabbit, X: me.X + 1.1*float32(dx), Y: me.Y + 1.0*float32(dy),
			HP: 6, MaxHP: 10, FaceLeft: dx > 0,
		})
		if *punch == "cross" { // a blow before this one, and a rest between
			first, rest := snap, snap
			first.Players, rest.Players = slices.Clone(snap.Players), slices.Clone(snap.Players)
			rest.Players[0].Acting = false
			first.Tick, rest.Tick = snap.Tick-2, snap.Tick-1
			conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: first}
			conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: rest}
		}
	}
	if *hurtFlag {
		before := snap
		before.Players = slices.Clone(snap.Players)
		before.Tick = snap.Tick - 1
		conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: before}
		snap.Players[0].HurtSeq++
	}
	conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: snap}

	if *audioSettings != "" {
		client.InitAudio(*audioSettings)
	}
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("shot")
	game := client.New(conn, "Tien")
	game.ShowOverlay(*overlay)
	game.HoldBlows(*blowAge)
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
func shotExplored(n, h int, x, y float64) []byte {
	seen := sim.NewExplored(n, h)
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

// addTribal puts one of each of the tribal age's things round (sx, sy).
func addTribal(snap *protocol.Snapshot, sx, sy float64) {
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < cx-9 || e.TX > cx+9 || e.TY < cy-6 || e.TY > cy+6 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	id := sim.EntityID(1 << 28)
	put := func(k sim.EntityKind, dx, dy int16, v uint8, ready bool) {
		id++
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: k, TX: cx + dx, TY: cy + dy, Ready: ready, Var: v, Fuel: sim.TorchStandTicks})
	}
	put(sim.EntityCorpse, 3, 1, uint8(sim.CreatureRaptor), true)
	put(sim.EntityCorpse, -4, 3, uint8(sim.CreatureTrike), true)
	put(sim.EntityTrophy, 2, -2, uint8(sim.CreatureRex), true)
	put(sim.EntityTrophy, 4, -2, uint8(sim.CreatureBear), true)
	put(sim.EntityTrophy, 6, -2, uint8(sim.CreatureMammoth), true)
	for f := range sim.NumFlowers {
		put(sim.EntityFlower, -8+int16(f), -4, uint8(f), true)
	}
	put(sim.EntityGlowPlant, -6, 0, 0, true)
	put(sim.EntityGlowPlant, -8, 1, 1, true)
	put(sim.EntityGlowPlant, -7, -2, 2, true)
	put(sim.EntityManEater, 7, 2, 0, true)
	put(sim.EntityDoor, -2, -3, 0, true)
	put(sim.EntityWall, -3, -3, 0, true)
	put(sim.EntityWall, -1, -3, 0, true)
	put(sim.EntitySpikes, 0, -3, 0, true)
	put(sim.EntityFloor, -3, -2, 0, true)
	put(sim.EntityFloor, -2, -2, 0, true)
	put(sim.EntityTrap, 5, 3, 0, true)
	put(sim.EntityTorchStand, 1, 2, 0, true)
	put(sim.EntityTreasure, 8, 0, 0, true)
	put(sim.EntitySapling, -1, 4, uint8(sim.SeedGlow), false)
	put(sim.EntityHut, -6, 5, 0, true)
	put(sim.EntityTotem, -3, 5, 0, true)
	cid := sim.CreatureID(1 << 28)
	beast := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
		cid++
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{ID: cid, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left})
	}
	beast(sim.CreatureSpider, 4.5, 4.5, sim.CreatureChase, true)
	beast(sim.CreatureSpider, 6.5, 5.2, sim.CreatureSpit, true)
	beast(sim.CreatureBear, -6, -1, sim.CreatureClaw, false)
	beast(sim.CreatureBear, 9, 4, sim.CreatureIdle, true)
	beast(sim.CreatureTribesman, 2.5, 4.2, sim.CreatureStrike, true)
	beast(sim.CreatureTribesman, -1.5, 3.2, sim.CreatureIdle, false)
	for i, k := range []sim.ItemKind{sim.ItemWood, sim.ItemStick, sim.ItemDinoHide, sim.ItemPotion, sim.HeadItem(sim.CreatureRaptor)} {
		snap.Drops = append(snap.Drops, protocol.DropView{ID: sim.DropID(i + 1), Item: k, Count: uint8(1 + i%2),
			X: float32(sx + 1 + float64(i)*0.6), Y: float32(sy + 1.6 + float64(i%2)*0.4), Age: 999})
	}
	snap.Webs = []protocol.WebView{{ID: 1, X: float32(sx + 2), Y: float32(sy + 3)}}
}

// addWoods clears the ground round (sx, sy) and puts the woods there.
func addWoods(snap *protocol.Snapshot, sx, sy float64) {
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < cx-9 || e.TX > cx+9 || e.TY < cy-7 || e.TY > cy+4 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	id := sim.EntityID(1 << 27)
	put := func(k sim.EntityKind, dx, dy int16, ready bool, v uint8, fuel int32) {
		id++
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: k, TX: cx + dx, TY: cy + dy, Ready: ready, Var: v, Fuel: fuel})
	}
	put(sim.EntityGiantTree, -5, -2, true, 1, 0)
	put(sim.EntityTree, 4, -3, true, 1, 0)
	put(sim.EntityTree, 6, -1, true, 0, 0)
	put(sim.EntityBananaTree, 1, -4, true, 0, 0)
	put(sim.EntityBananaTree, 3, 1, true, 0, 0)
	put(sim.EntityBananaTree, 7, 2, false, 0, 0)
	cid := sim.CreatureID(1 << 29)
	monkey := func(dx, dy float64, st sim.CreatureState, left bool) {
		cid++
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{ID: cid, Kind: sim.CreatureMonkey, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left})
	}
	monkey(2.2, -2.2, sim.CreatureIdle, true)
	monkey(1.2, -1.5, sim.CreatureEat, false)
	monkey(3.6, -0.6, sim.CreatureWander, true)
	monkey(5.2, 0.6, sim.CreatureIdle, false)
	snap.Players[0].Bees = true
}
