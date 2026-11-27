// Command shot renders one frame of the client against a fake, frozen server
// and saves it as a PNG. It is a dev tool for checking art changes:
//
//	go run ./cmd/shot -o day.png
//	go run ./cmd/shot -o night.png -time 0.85
//	xvfb-run -a -s "-screen 0 1920x1080x24" go run ./cmd/shot -w 1920 -h 1080 -o hd.png
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
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
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
	overlay := flag.String("overlay", "", "open the map, the bag or the crafting window: map, bag, craft")
	animals := flag.Bool("animals", false, "put one of each animal (fish with -coast) next to the player")
	held := flag.Int("held", -1, "item kind the player holds (default: the selected slot's)")
	fishing := flag.String("fishing", "", "show the player fishing: wait or bite (use with -coast)")
	builds := flag.Bool("builds", false, "put a wooden wall and a bed nearby, with the other player asleep in it")
	farm := flag.Bool("farm", false, "put a row of tilled soil and growing sweet potatoes nearby, and a hoe in hand")
	furnace := flag.Bool("furnace", false, "put a lit furnace and one of each ore nearby, and a copper pickaxe in hand")
	open := flag.Bool("open", false, "an open world instead of the 96x96 island")
	sizes := flag.Bool("sizes", false, "with -menu: show the difficulty picker (it took the place of the world-size picker)")
	size := flag.Int("size", 256, "open world size")
	at := flag.String("at", "", "open world: stand at desert, marsh, pond (a swamp with ponds), cave, mountain, ash or lava; or by the nearest cavemouth, cavein, pyramid, village, path, giant (tree) or hut (at its door); and in the newest worlds at waterfall, ramp (through a cliff), lagoon or beach (see isles.go)")
	female := flag.Bool("female", false, "the player is a woman")
	hair := flag.Int("hair", 0, "the player's hair style, 0 to 6 (6: the shaggy mane)")
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
	warm := flag.Bool("warm", false, "the trial warm ground: ochre earth with patches of grass")
	fx := flag.Bool("fx", false, "turn on the post-processing shaders: glow and colour grading")
	quality := flag.Int("quality", 100, "the render quality: the world drawn at 100, 75 or 50 percent of the screen and blown up (the interface stays sharp; see internal/client/display.go)")
	uiScale := flag.Float64("uiscale", 0, "the interface's scale, screen pixels to a UI pixel (2.5 is 250%); 0 is automatic")
	settingsTab := flag.String("settings", "", "open the settings screen at a tab: audio, display or controls (with -menu, over the main menu; else over the pause menu)")
	fight := flag.Bool("fight", false, "show the beasts' health: a few just hurt, with the damage floating up (use with -dinos)")
	troodon := flag.Bool("troodon", false, "put a pack of night hunters round the player: one stalking, one leaping in, one biting, and two more watching from the dark (use with -time 0.9 for their eyes)")
	woodsFlag := flag.Bool("woods", false, "put the woods round the player: a giant tree and a tree with beehives, banana trees, a troop of monkeys, and bees after the player")
	jungleFlag := flag.Bool("jungle", false, "a clearing in the woods to see their size by: two giants of the old forest in full view, ordinary trees, banana trees and monkeys round the player")
	dung := flag.Bool("dung", false, "with -farm: a fed bed, heaps of dung by a grazing three-horn, and dung in hand")
	sea := flag.Bool("sea", false, "out at sea: the player sailing, a raft, a swimmer, a mosasaur and a giant squid")
	armor := flag.Int("armor", 0, "the armour tier the player wears, 0 (none) to 4")
	wear := flag.Int("wear", 0, "the clothes the player has on, bits added up: 1 tunic, 2 mantle, 4 leg wraps, 8 boots (15: all; 0: the furs everyone starts in)")
	bagFrom := flag.Int("bagfrom", 0, "fill every slot with one item kind each, counting up from this one (to review the icons with -overlay bag)")
	prey := flag.Bool("prey", false, "put a raptor right in front of the player, so -acting shows a blow at a beast (-frames picks the moment)")
	punch := flag.String("punch", "", "a blow by hand at a rabbit put in front of the player: jab, or cross for the other hand; nothing is held unless -held says so")
	hurtFlag := flag.Bool("hurt", false, "the player has just been struck")
	villageIdx := flag.Int("village", 0, "with -at village: which village to stand in, from 0")
	blowAge := flag.Int("age", -1, "with -punch or -hurt: how many frames into the blow or the jolt to show")
	bodies := flag.Bool("bodies", false, "put a fallen player's body and its bag next to the player, and another's gone to bones")
	bossesFlag := flag.Bool("bosses", false, "put the six bosses round the player on a beach (the Megalodon and a lurking Spinosaurus in the sea), the boss bar up and the lairs on the map; see -bosspose")
	bossPose := flag.Int("bosspose", 0, "with -bosses: shift which pose each boss shows")
	bossAge := flag.Int("bossage", -1, "how many frames into its state each boss and sea monster is shown (-1: as many as -frames)")
	flag.BoolVar(&demoBosses, "demobosses", false, "with -bosses: only Titanoboa and the Megalodon, the bar showing Titanoboa's health")
	flag.StringVar(&titanAt, "titanat", "", "with -bosses: dx,dy[,1], where to put Titanoboa, in tiles from the player (default -7,-3.5), facing left with ,1; the Megalodon is left out")
	skillsFlag := flag.String("skills", "", "film every move of these beasts (a comma list of rex, dakota, achero, troodon, ptero, quetzal, plesio, sabre, trike, toro, stego, brachio, ankylo, anzu, thesc, ornitho, mammoth; or all) and write a GIF of each into -o's folder; see skills.go")
	encounterFlag := flag.String("encounter", "", "play a real meeting with a boss of the demo, titan or shark, and write its stills as <o>_approach.png, _lair.png, _intro.png and _arena.png; see encounter.go")
	bossFight := flag.String("bossfight", "", "play a scene at a lair of StyleLairs and write its pictures into -o's folder: titan, shark or map; see bossfight.go")
	encGIF := flag.Bool("encgif", false, "with -encounter: also write every fourth frame of the entrance as <o>_gif_NNN.png")
	king := flag.Bool("king", false, "put the mammoth king by the player, with a mammoth of its herd for scale; -bosspose picks its state (0 idle, 1 chase, 2 strike, 3 special, 4 roar), -bossage how far into it")
	victory := flag.Bool("victory", false, "show the game-beaten (Phá đảo!) screen")
	aimAt := flag.String("aim", "", "dx,dy: hold the mouse pointer this many tiles from the player (the hover brackets, a held torch's ghost)")
	buildID := flag.Int("build", -1, "recipe id of a structure to place (0 campfire, 8 wall, 9 bed…): build mode with its ghost under -aim, the ingredients given")
	herohd := flag.Bool("herohd", false, "draw the players as the trial hero: bigger, with a run, a sword blow and firelight (a trial; see internal/client/herohd.go)")
	chestFlag := flag.String("chest", "", "open a chest next to the player, with things in it: chest, or treasure (an old chest from a cave with its hoard)")
	statusFlag := flag.String("status", "", "status effects on the player, comma-separated: cold, poison, web, starving, hungry, bees")
	running := flag.Bool("run", false, "the player is on the move (with -herohd: mid-stride)")
	winW := flag.Int("w", 1280, "window width; run xvfb with a screen at least this big (xvfb-run -s '-screen 0 1920x1080x24')")
	winH := flag.Int("h", 720, "window height: 720 draws the world at 2 screen pixels per art pixel, 1080 at 3")
	floraFlag := flag.String("flora", "", "set the new plants and the toads round the player: pond (with -open -at pond), coast (with -coast) or land")
	hdTest := flag.Bool("hdtest", false, "set down the HD trial: a mossy stone drawn as the old pixel art (left of the player) and at HD density (right); see internal/client/HD_ART.md")
	envHD := flag.Bool("envhd", true, "the HD art of trees, plants, rocks and buildings (false: the original density-1 art, to compare)")
	devCon := flag.Bool("devconsole", false, "open the dev console (as with -dev in cmd/game), with a few commands run on the world; see dev.go")
	critterHD := flag.Bool("critterhd", true, "the HD art of the code-drawn animals (false: the original density-1 art, to compare)")
	scaleSheet := flag.Bool("scalesheet", false, "write the scale sheet instead of a frame: the player next to every tree, plant, rock and building, with heights (see internal/client/scalesheet.go); the heights are printed too")
	trunkSheet := flag.Bool("trunksheet", false, "write the trunk sheet instead of a frame: the foot of every kind of tree close up, three times the size (see internal/client/trunksheet.go)")
	costFlag := flag.Bool("cost", false, "print what the art drawn in code has cost once the frame is taken: the bytes of its images, the time to draw the start-up set and the trees drawn ahead, and the Go heap")
	swimAt := flag.String("swimat", "", "with -swim: put the player in the water nearest the spawn: deep (open sea), mid (deep water by the shallows), shallow (the shallows), ford or river")
	swimFlag := flag.Bool("swim", false, "the player is swimming where they stand (put them in the water with -coast -move), and the other player, a woman with long hair, swims beside them; with -run both are on the move")
	moveBy := flag.String("move", "", "dx,dy: stand this many tiles from where the other flags put the player (to stand behind a tree, at a door)")
	standAt := flag.String("pos", "", "x,y: stand on this tile, whatever the other flags say (to see one place in worlds of two styles)")
	scaleOnly := flag.String("scaleonly", "", "with -scalesheet: only the groups whose name contains this (trees, conifers, swamp, low, buildings, rocks, creatures)")
	scaleTitle := flag.String("scaletitle", "", "with -scalesheet: a heading for the sheet")
	subPixel := flag.Bool("subpixel", true, "let the camera and moving things sit between art pixels, on screen pixels (false: whole art pixels, as before HD art)")
	reveal := flag.Bool("reveal", false, "the whole map seen: no fog on the map (M) and the minimap")
	mapZoom := flag.Int("mapzoom", 1, "with -overlay map: zoom the map in this many times round the player (as the wheel does)")
	gifN := flag.Int("gif", 0, "write a GIF of this many frames (with -crop: of that region, not enlarged) as <o>.gif, from the frame -frames picks")
	gifStep := flag.Int("gifstep", 2, "with -gif: game frames between the GIF's frames")
	walkFlag := flag.String("walk", "", "dx,dy: the player walks this many tiles a second the whole time (to see the grass bend round them; with -gif)")
	benchN := flag.Int("bench", 0, "time this many frames before the capture, vsync off, and print what they took and the ground's memory")
	motionSheet := flag.String("motionsheet", "", "write the player's motion sheet and GIFs into this folder instead of a frame (see internal/client/motionsheet.go)")
	thin := flag.Float64("thin", 1, "keep about this share of the trees, bushes and rocks round the player (to see the ground)")
	styleFlag := flag.Int("style", -1, "with -open: the world style to make (default: the latest; 7 is the archipelago before the designed islands, 8 the designed islands before the woods were spaced, 9 the woods before the wide beaches)")
	canopyFlag := flag.Bool("canopy", false, "print the share of the view the canopy covers: the drawings of trees and the like, taller than a player (see internal/client/canopyshare.go)")
	oldSave := flag.Bool("oldsave", false, "with -open: save the world as a game from before the woods were thinned (version 19) and load it again, which thins them (see internal/sim/thinwoods.go)")
	loadSave := flag.String("load", "", "load this save file (a copy of one) instead of making a world, an older one thinned as the game would; stand where its first player stood (-move is from there)")
	fellR := flag.Float64("fell", 0, "fell every tree within this many tiles of the player (stumps), before -days")
	days := flag.Float64("days", 0, "let this many days go by for what grows: stumps, saplings, young trees, the woods filling their gaps (see internal/sim/regrow.go)")
	regrowStrip := flag.Bool("regrow", false, "put the stages a felled tree grows back through in a row right of the player: stump, sapling, young tree, grown tree")
	realState := flag.Bool("real", false, "show every entity as it is (by default every ninth shows harvested, to see those states too)")
	exact := flag.Bool("exact", false, "draw on a screen exactly -w by -h pixels, whatever size the window gets (a window taller than the desktop is cut short, and a 1061-pixel screen is not 1080p: the interface is drawn at scale 1, not 2)")
	iconHD := flag.Bool("iconhd", true, "the HD item icons, at the screen's own resolution (false: the old 16x16 icons, scaled up; see internal/client/iconhd.go)")
	iconSheet := flag.Bool("iconsheet", false, "write the icon sheet instead of a frame: every HD item icon next to its old one, in 1080p hotbar slots and three times as big")
	freeBuild := flag.Bool("freebuild", false, "dev mode with free build on: the DEV · XÂY TỰ DO tag, every recipe lit with a DEV badge; -overlay palette opens the build palette (see internal/sim/devbuild.go)")
	devBase := flag.Bool("devbase", false, "build a small base round the player through free build's own commands: the ground cleared, then walls, a door, floors, a bed, fires, a farm, flowers (implies -freebuild; see dev.go)")
	devPiece := flag.Int("devpiece", -1, "a piece of the build palette in hand, by its number (free build): its ghost under -aim")
	combatSheet := flag.String("combatsheet", "", "write the combat sheets into this folder instead of a frame: the shield on the figure in every facing, and every effect frame by frame (see internal/client/combatsheet.go)")
	combatLive := flag.String("combat", "", "play a real fight and write it as <o>.gif: feel (blows with hit-stop, flash, knockback, shake) or shield (raise, block, parry); see combat.go")
	heldSheet := flag.String("heldsheet", "", "write the held-item sheets into this folder instead of a frame: every held item in the hand old and new, and a GIF of the swings (see internal/client/heldsheet.go)")
	heldFit := flag.String("heldfit", "", "write the held-fit sheets into this folder instead of a frame: every tool in hand in every facing, and GIFs of a run from behind and from the side (see internal/client/heldfitsheet.go)")
	heldHD := flag.Bool("heldhd", true, "the HD held items, painted at the figure's density and turned cleanly (false: the old density-1 sprites; see internal/client/heldhd.go)")
	flag.Parse()
	client.SetMeasureCanopy(*canopyFlag)
	client.SetHeroHD(*herohd)
	client.SetScaleOnly(*scaleOnly, *scaleTitle)
	client.SetHDTest(*hdTest)
	client.SetEnvHD(*envHD)
	client.SetCritterHD(*critterHD)
	client.SetIconHD(*iconHD)
	client.SetHeldHD(*heldHD)
	client.SetSubPixel(*subPixel)
	client.SetZoom(*zoom)
	client.SetPostFX(*fx)
	client.OverrideSettings(func(s *client.Settings) { s.Quality = *quality })
	if *uiScale > 0 {
		client.SetUIScale(*uiScale)
	}
	client.SetWarmGround(*warm)
	if *bossesFlag {
		*coast = true
	}
	client.SetBossAge(*bossAge)
	if *skillsFlag != "" {
		runSkills(*skillsFlag, filepath.Dir(*out), *seed, *size, *winW, *winH)
		return
	}
	if *combatLive != "" {
		runCombat(*combatLive, *out, *seed, *winW, *winH, *dayFrac)
		return
	}
	if *bossFight != "" {
		runBossFight(*bossFight, filepath.Dir(*out), *seed, *size, *winW, *winH)
		return
	}
	if *encounterFlag != "" {
		runEncounter(*encounterFlag, strings.TrimSuffix(*out, ".png"), *seed, *size, *winW, *winH, *dayFrac, *encGIF)
		return
	}

	N := 96
	style := sim.StyleIsland
	if *open {
		N, style = *size, sim.StyleLatest
		if *styleFlag >= 0 {
			style = sim.WorldStyle(*styleFlag)
		}
	}
	var w *sim.World
	var err error
	var savedAt []float64 // where the save's first player stood
	switch {
	case *loadSave != "":
		f, ferr := server.ReadSave(*loadSave)
		if ferr != nil {
			log.Fatal(ferr)
		}
		if w, err = sim.Load(f.World); err != nil {
			log.Fatal(err)
		}
		N, _ = w.Size()
		*open = true
		if ps := f.World.Players; len(ps) > 0 {
			savedAt = []float64{ps[0].X, ps[0].Y}
		}
	default:
		if w, err = sim.Generate(sim.GenConfig{Seed: *seed, Width: N, Height: N, Style: style}); err != nil {
			log.Fatal(err)
		}
		if *oldSave {
			d := w.Save()
			d.Version = 19 // the last version before the woods were thinned
			if w, err = sim.Load(d); err != nil {
				log.Fatal(err)
			}
		}
	}
	_, mapH := w.Size() // taller than N when caves lie under the islands
	surfH := 0
	if w.SurfaceHeight() < mapH {
		surfH = w.SurfaceHeight()
	}
	sx, sy := w.Spawn()
	if savedAt != nil {
		sx, sy = savedAt[0], savedAt[1]
	}
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
	if *swimAt != "" {
		sx, sy = swimSpot(w, *swimAt, sx, sy)
	}
	if x, y, ok := islesSpot(w, *at, sx, sy); ok { // see isles.go
		sx, sy, *at = x, y, ""
		log.Printf("standing at %.1f,%.1f", sx, sy)
	}
	if x, y, ok := cavePropSpot(w, *at, sx, sy, -1.5, 2.5); ok { // see caves.go
		sx, sy, *at = x, y, ""
	}
	if x, y, ok := caveSpot(w, *at, sx, sy); ok { // see caves.go
		sx, sy, *at = x, y, ""
		log.Printf("standing at %.1f,%.1f", sx, sy)
	}
	if *at == "cavemouth" || *at == "cavein" || *at == "cavedown" || *at == "caveup" || *at == "giant" || *at == "hut" {
		k := sim.EntityCaveMouth
		switch *at {
		case "cavedown": // by the hole down to the deep caves (natural caves)
			k = sim.EntityCaveDown
		case "caveup": // down in the deep caves, by the rope back up
			k = sim.EntityCaveUp
		case "cavein":
			k = sim.EntityCaveExit
		case "giant": // in front of the nearest giant of the old forest (-move to step aside)
			k = sim.EntityGiantTree
		case "hut": // at the door of the nearest hut of a tribe
			k = sim.EntityHut
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
				if k == sim.EntityCaveDown || k == sim.EntityCaveUp {
					sy = float64(e.TY) + 2.2
				}
				if k == sim.EntityHut {
					sy = float64(e.TY) + 1.45 // on the step before the doorway
				}
			}
		})
		log.Printf("standing at %.1f,%.1f", sx, sy)
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
			v := vs[min(max(*villageIdx, 0), len(vs)-1)]
			sx, sy = v.X, v.Y+3
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
			"pond": sim.TileMarsh, "forest": sim.TileForest, "rocky": sim.TileRocky}[*at]
		// Desert, marsh, snow, woods and rocky hills: stand well inside, where they fill the screen.
		deep := 0.0
		if *at == "desert" || *at == "marsh" || *at == "snow" || *at == "pond" || *at == "forest" || *at == "rocky" {
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
	var px, py float64
	if n, _ := fmt.Sscanf(*standAt, "%g,%g", &px, &py); n == 2 {
		sx, sy = px+0.5, py+0.5
	}
	var mdx, mdy float64
	if n, _ := fmt.Sscanf(*moveBy, "%g,%g", &mdx, &mdy); n == 2 {
		sx, sy = sx+mdx, sy+mdy
	}
	if *fellR > 0 {
		log.Printf("felled %d trees", w.DevFell(sx, sy, *fellR))
	}
	if *days > 0 {
		w.DevPassTime(uint64(*days * sim.DayTicks)) // see internal/sim/regrow.go
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
	if *bagFrom > 0 {
		for i := range snap.Self.Inv {
			if k := sim.ItemKind(*bagFrom + i); k.Valid() && k < sim.ItemHeadBase {
				snap.Self.Inv[i] = sim.Slot{Item: k, Count: min(k.MaxCount(), 1+i%7)}
			} else {
				snap.Self.Inv[i] = sim.Slot{}
			}
		}
	}
	snap.Self.Selected = 1
	snap.Players[0].Held = snap.Self.Inv[snap.Self.Selected].Item
	if *held >= 0 {
		snap.Players[0].Held = sim.ItemKind(*held)
	}
	if *chestFlag != "" {
		kind := sim.EntityChest
		if *chestFlag == "treasure" {
			kind = sim.EntityTreasure
		}
		const id = 1<<30 + 77
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: kind, TX: int16(sx) + 1, TY: int16(sy)})
		snap.Self.Chest = id
		c := &snap.Self.ChestSlots
		c.Add(sim.ItemPotion, 2)
		c.Add(sim.ItemTorch, 3)
		c.Add(sim.ItemCoal, 5)
		c.Add(sim.ItemIronOre, 3)
		if kind == sim.EntityChest {
			c.Add(sim.ItemWood, 64)
			c.Add(sim.ItemStone, 22)
			c.Add(sim.ItemCookedMeat, 6)
		} else {
			c.Add(sim.ItemDiamond, 1)
			c.Add(sim.ItemGlowSeed, 2)
		}
	}
	for _, st := range strings.Split(*statusFlag, ",") {
		switch st {
		case "cold":
			snap.Self.Cold = true
		case "poison":
			snap.Self.Poisoned = true
		case "web":
			snap.Self.Webbed = true
		case "starving":
			snap.Self.Hunger = 0
		case "hungry":
			snap.Self.Hunger = 12
		case "bees":
			snap.Players[0].Bees = true
		}
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
	snap.Players[0].Wear = sim.Wear(*wear)
	snap.Players[1].Held = sim.ItemBerry
	if snap.Players[0].Held == sim.ItemTorch {
		snap.Self.TorchFuel = sim.TorchBurnTicks * 2 / 3
	}
	snap.Self.Inv.Add(sim.ItemTorch, 2)
	if r, ok := sim.RecipeByID(sim.RecipeID(*buildID)); ok && *buildID >= 0 {
		for _, in := range r.Inputs {
			snap.Self.Inv.Add(in.Item, in.Count) // build mode lasts while they are there
		}
	}
	snap.Self.Inv.Add(sim.ItemSpear, 1)
	if *devBase {
		*freeBuild = true
		buildDevBase(w, sx, sy) // see dev.go
	}
	snap.Self.DevBuild = *freeBuild

	cx, cy := int(math.Floor(sx)), int(math.Floor(sy))
	n := 0
	w.EachEntityIn(cx-20, cy-20, cx+20, cy+20, func(e sim.Entity) {
		n++
		ready := e.Ready
		if n%9 == 0 && !*realState { // show some harvested states too
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
	if *floraFlag != "" {
		addFlora(&snap, w, sx, sy, *floraFlag) // see flora.go
	}
	if *woodsFlag {
		addWoods(&snap, sx, sy)
	}
	if *regrowStrip {
		addRegrowStrip(&snap, w, sx, sy) // see regrow.go
	}
	if *jungleFlag {
		addJungle(&snap, sx, sy)
	}
	if *bodies {
		addBodies(&snap, sx, sy)
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
	if *troodon {
		put := func(id sim.CreatureID, dx, dy float64, st sim.CreatureState, left bool) {
			snap.Creatures = append(snap.Creatures, protocol.CreatureView{
				ID: id, Kind: sim.CreatureTroodon, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
				HP: 5, MaxHP: 5,
			})
		}
		put(troodonLeaper, 1.3, 2.1, sim.CreatureStrike, true)     // leaping in
		put(troodonLeaper+1, -1.5, 0.3, sim.CreatureStrike, false) // biting
		put(troodonLeaper+2, -4.2, 1.8, sim.CreatureChase, false)  // crouched, waiting its turn
		put(troodonLeaper+3, -7.5, -2.2, sim.CreatureChase, false) // and two more out in the dark
		put(troodonLeaper+4, 7.5, 3.4, sim.CreatureIdle, true)
	}
	if *king {
		states := []sim.CreatureState{sim.CreatureIdle, sim.CreatureChase, sim.CreatureStrike, sim.CreatureSpecial, sim.CreatureRoar}
		hp := int16(min(sim.CreatureMammothKing.MaxHP(), 32767))
		snap.Creatures = append(snap.Creatures,
			protocol.CreatureView{ID: 1<<25 + 1, Kind: sim.CreatureMammothKing, X: float32(sx - 7.5), Y: float32(sy + 1.2), State: states[*bossPose%len(states)], HP: hp, MaxHP: hp},
			protocol.CreatureView{ID: 1<<25 + 2, Kind: sim.CreatureMammoth, X: float32(sx + 5), Y: float32(sy + 1.6), State: sim.CreatureIdle, FaceLeft: true})
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
		put(sim.CreatureAnkylo, 8.5, -6.5, sim.CreatureIdle, true)
		put(sim.CreatureAnkylo, -10.5, 1, sim.CreatureRoar, false) // its club turned on someone
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
	putDinosA(&snap, sx, sy, *coast) // dinosa.go
	putAnkylo(&snap, sx, sy)         // ankylo.go
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
	if *bossesFlag {
		addBosses(&snap, sx, sy, *bossPose) // see bosses.go
	}
	if *victory {
		addVictory(&snap)
	}
	if *prey {
		dx, dy := sim.Dir(*dir).Delta()
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: 1<<27 + 1, Kind: sim.CreatureRaptor, X: float32(sx + 1.3*float64(dx)), Y: float32(sy + 1.3*float64(dy)), State: sim.CreatureIdle, FaceLeft: dx > 0,
		})
	}
	if *swimFlag {
		snap.Players[0].Swimming, snap.Self.Breath = true, 1
		if len(snap.Players) > 1 {
			o := &snap.Players[1]
			o.X, o.Y, o.Dir, o.Swimming = float32(sx-2.4), float32(sy+0.4), snap.Players[0].Dir, true
			o.Look = sim.Look{Female: true, Hair: sim.HairLong, HairColor: 3, Skin: 2}
			o.Mount, o.Boat, o.Asleep = 0, 0, false
		}
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

	// A campfire next to the spawn, so lighting is visible at night; not
	// in the base built by -devbase, which has its own.
	if !*sea && !*devBase {
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID: 1 << 30, Kind: sim.EntityCampfire, TX: int16(cx + 2), TY: int16(cy), Ready: true,
			Fuel: sim.FireStartFuel,
		})
	}

	conn := &fakeConn{ch: make(chan protocol.ServerMsg, 8)}
	conn.ch <- protocol.ServerMsg{Kind: protocol.ServerWelcome, Welcome: protocol.Welcome{
		PlayerID: 1, Width: N, Height: mapH, Tiles: w.Tiles(), Relief: w.Relief(),
		Explored: shotExplored(N, mapH, sx, sy), SurfaceH: surfH, Caves: w.CaveCells(),
	}}
	if *reveal {
		m := <-conn.ch
		m.Welcome.Explored = revealAll(N, mapH)
		conn.ch <- m
	}
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
	if *running {
		before := snap
		before.Players = slices.Clone(snap.Players)
		before.Tick = snap.Tick - 1
		dx, dy := snap.Players[0].Dir.Delta()
		before.Players[0].X -= 0.04 * float32(dx)
		before.Players[0].Y -= 0.04 * float32(dy)
		if *swimFlag && len(before.Players) > 1 {
			before.Players[1].X -= 0.04 * float32(dx)
			before.Players[1].Y -= 0.04 * float32(dy)
		}
		conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: before}
	}
	if *hurtFlag {
		before := snap
		before.Players = slices.Clone(snap.Players)
		before.Tick = snap.Tick - 1
		conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: before}
		snap.Players[0].HurtSeq++
	}
	cookScene(&snap, sx, sy)   // see cook.go
	hudScene(&snap)            // see hud.go
	tentScene(&snap, sx, sy)   // see tent.go
	thinEntities(&snap, *thin) // see terrain.go
	conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: snap}

	if *audioSettings != "" {
		client.InitAudio(*audioSettings)
	}
	ebiten.SetWindowSize(*winW, *winH)
	ebiten.SetWindowTitle("shot")
	game := client.New(conn, "Tien")
	game.ShowOverlay(*overlay)
	hudGame(game) // see hud.go
	if *settingsTab != "" && !*menu {
		game.ShowSettings(*settingsTab) // over the pause menu
	}
	game.SetMapZoom(*mapZoom)
	var adx, ady float64
	if n, _ := fmt.Sscanf(*aimAt, "%g,%g", &adx, &ady); n == 2 {
		game.SetAim(adx, ady)
	}
	if *buildID >= 0 {
		game.StartBuild(sim.RecipeID(*buildID))
	}
	if *freeBuild {
		game.SetDev(true) // the DEV tag, which names free build
	}
	if *devPiece >= 0 {
		game.StartPiece(*devPiece)
	}
	game.HoldBlows(*blowAge)
	if *devCon {
		showDevConsole(game, w)
	}
	if *troodon {
		game.HoldLeap(troodonLeaper)
	}
	var inner ebiten.Game = game
	if *create {
		m := client.NewMenu(client.SaveInfo{})
		m.ShowCharacterScreen()
		inner = &menuGame{m}
	}
	if *menu {
		m := client.NewMenu(client.SaveInfo{Exists: true, Day: 3, Name: "Tien", SavedAt: time.Now(), Size: 512})
		if *sizes {
			m.ShowDifficultyPicker()
		}
		if *settingsTab != "" {
			m.ShowSettings(*settingsTab)
		}
		inner = &menuGame{m}
	}
	g := &shooter{inner: inner, frames: *frames, out: *out, crop: *crop, cost: *costFlag, gifN: *gifN, gifStep: max(1, *gifStep), bench: *benchN}
	var wdx, wdy float64
	if n, _ := fmt.Sscanf(*walkFlag, "%g,%g", &wdx, &wdy); n == 2 {
		g.walk = &walker{conn: conn, snap: snap, vx: float32(wdx / sim.TicksPerSecond), vy: float32(wdy / sim.TicksPerSecond)}
	}
	if *exact {
		g.fixW, g.fixH = *winW, *winH
	}
	if *benchN > 0 {
		ebiten.SetVsyncEnabled(false)
		ebiten.SetTPS(ebiten.SyncWithFPS)
	}
	if *scaleSheet {
		g.sheet = game
	}
	if *trunkSheet {
		g.trunks = game
	}
	if *motionSheet != "" {
		g.motions, g.motionDir = game, *motionSheet
	}
	if *iconSheet {
		g.icons = game
	}
	if *combatSheet != "" {
		g.combat, g.combatDir = game, *combatSheet
	}
	if *heldSheet != "" {
		g.held, g.heldDir = game, *heldSheet
	}
	if *heldFit != "" {
		g.heldFit, g.heldFitDir = game, *heldFit
	}
	if *tentSheetFlag != "" {
		g.tents, g.tentDir = game, *tentSheetFlag
	}
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if g.err != nil {
		log.Fatal(g.err)
	}
	if *canopyFlag {
		log.Printf("canopy covers %.0f%% of the view; own figure silhouetted: %v", 100*client.CanopyShare(), client.SelfSilhouetted())
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
	inner      ebiten.Game
	frames     int
	n          int
	out        string
	crop       string
	done       bool
	err        error
	sheet      *client.Game // write its scale sheet instead of the frame
	trunks     *client.Game // write its trunk sheet instead of the frame
	motions    *client.Game // write its motion sheets into motionDir instead of the frame
	motionDir  string
	held       *client.Game // write its held-item sheets into heldDir instead of the frame
	combat     *client.Game // write its combat sheets into combatDir instead of the frame
	combatDir  string
	heldDir    string
	heldFit    *client.Game // write its held-fit sheets into heldFitDir instead of the frame
	heldFitDir string
	icons      *client.Game // write its icon sheet instead of the frame
	tents      *client.Game // write the leaf tent's sheet and GIF into tentDir instead of the frame
	tentDir    string       // the folder they go in
	cost       bool         // print what the art has cost

	gifN, gifStep int           // -gif: frames to keep, and game frames between them
	gifFrames     []*image.RGBA // kept so far
	gifAt         int           // the game frame the last was kept at
	bench         int           // -bench: frames to time
	fixW, fixH    int           // -exact: the screen's size, whatever the window's
	benchTimes    []time.Duration
	benchLast     time.Time
	walk          *walker // -walk: the player on the move; see walk.go
}

func (s *shooter) Update() error {
	if s.done {
		return ebiten.Termination
	}
	s.n++
	if s.walk != nil {
		s.walk.step(s.n)
	}
	return s.inner.Update()
}

func (s *shooter) Draw(screen *ebiten.Image) {
	s.inner.Draw(screen)
	if s.bench > 0 && s.n >= s.frames-s.bench && s.n < s.frames && !s.done {
		s.benchFrame(screen) // see terrain.go
	}
	if s.n < s.frames || s.done {
		return
	}
	if s.gifN > 0 {
		if len(s.gifFrames) == 0 || s.n-s.gifAt >= s.gifStep {
			s.gifFrame(screen)
			s.gifAt = s.n
		}
		if len(s.gifFrames) < s.gifN {
			return
		}
		if err := writeGIF(gifPath(s.out), s.gifFrames, max(2, int(math.Round(float64(s.gifStep)*100/60)))); err != nil {
			s.err, s.done = err, true
			return
		}
		log.Printf("wrote %s", gifPath(s.out))
	}
	if s.tents != nil {
		s.err, s.done = s.tents.TentSheets(s.tentDir), true
		if s.err == nil {
			log.Printf("wrote the tent sheets in %s", s.tentDir)
		}
		return
	}
	if s.motions != nil {
		s.err, s.done = s.motions.MotionSheets(s.motionDir), true
		if s.err == nil {
			log.Printf("wrote the motion sheets in %s", s.motionDir)
		}
		return
	}
	if s.combat != nil {
		s.err, s.done = s.combat.CombatSheets(s.combatDir), true
		if s.err == nil {
			log.Printf("wrote the combat sheets in %s", s.combatDir)
		}
		return
	}
	if s.heldFit != nil {
		s.err, s.done = s.heldFit.HeldFitSheets(s.heldFitDir), true
		if s.err == nil {
			log.Printf("wrote the held-fit sheets in %s", s.heldFitDir)
		}
		return
	}
	if s.held != nil {
		s.err, s.done = s.held.HeldSheets(s.heldDir), true
		if s.err == nil {
			log.Printf("wrote the held-item sheets in %s", s.heldDir)
		}
		return
	}
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	var final image.Image = img
	if s.sheet != nil || s.cost {
		printArtCost("at the frame")
	}
	s.printBench()
	if s.trunks != nil {
		final = s.trunks.TrunkSheet()
	}
	if s.icons != nil {
		final = s.icons.IconSheet()
	}
	if s.sheet != nil {
		final = s.sheet.ScaleSheet()
		printArtCost("with the whole sheet drawn")
		for _, r := range s.sheet.ScaleHeights() {
			fmt.Printf("%-30s %-22s %5d px %5.2fx\n", r.Group, r.Name, r.Px, r.Players)
		}
	}
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

func (s *shooter) Layout(w, h int) (int, int) {
	if s.fixW > 0 {
		return s.fixW, s.fixH
	}
	return s.inner.Layout(w, h)
}

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

// addBodies clears the ground round (sx, sy) and lays fallen players there:
// fresh bodies to the right, the same players gone to bones to the left.
func addBodies(snap *protocol.Snapshot, sx, sy float64) {
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < cx-6 || e.TX > cx+6 || e.TY < cy-4 || e.TY > cy+4 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	// fuel packs a look and a colour as sim's packGraveLook does.
	fuel := func(l sim.Look, col int) int32 {
		f := int(l.Hair&7)<<1 | int(l.HairColor&7)<<4 | int(l.Skin&7)<<7 | (col&15)<<10
		if l.Female {
			f |= 1
		}
		if got, c := sim.GraveLook(int32(f)); got != l.Sanitized() || c != col {
			panic("addBodies: sim.GraveLook does not read the look back")
		}
		return int32(f)
	}
	id := sim.EntityID(1 << 26)
	put := func(dx, dy int16, bones uint8, l sim.Look, col int) {
		id++
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: sim.EntityGrave, TX: cx + dx, TY: cy + dy, Ready: true, Var: bones, Fuel: fuel(l, col)})
	}
	a := sim.Look{Hair: sim.HairSpiky, HairColor: 4, Skin: 1}
	b := sim.Look{Female: true, Hair: sim.HairLong, HairColor: 3}
	c := sim.Look{Hair: sim.HairPonytail, HairColor: 1, Skin: 4}
	put(5, 0, 0, a, 1)
	put(3, 2, 0, b, 4)
	put(2, -2, 0, c, 2)
	put(-5, 0, sim.GraveBones, b, 4)
	put(-3, 3, sim.GraveBones, a, 1)
	put(-2, -2, sim.GraveBones, c, 2)
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

// troodonLeaper is the night hunter -troodon shows in mid-leap.
const troodonLeaper = sim.CreatureID(1<<24 + 1)

// printArtCost prints what the art drawn in code has cost so far, and the Go
// heap (after a collection).
func printArtCost(when string) {
	bytes, startup, warm := client.ArtCost()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ahead := "not done yet"
	if warm > 0 {
		ahead = warm.Round(time.Millisecond).String()
	}
	fmt.Printf("art %s: %.1f MB of images drawn in code; start-up set drawn in %v; trees drawn ahead in %s; Go heap %.0f MB in use, %.0f MB from the OS\n",
		when, float64(bytes)/(1<<20), startup.Round(time.Millisecond), ahead, float64(m.HeapInuse)/(1<<20), float64(m.Sys)/(1<<20))
}

// addJungle clears the woods round (sx, sy) and stands two giants of the old
// forest there, far enough down the screen to be seen whole, with ordinary
// trees behind for scale, banana trees and a few monkeys.
func addJungle(snap *protocol.Snapshot, sx, sy float64) {
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < cx-15 || e.TX > cx+15 || e.TY < cy-11 || e.TY > cy+9 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	id := sim.EntityID(1 << 26)
	put := func(k sim.EntityKind, dx, dy int16, big bool, v uint8) {
		id++
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: k, TX: cx + dx, TY: cy + dy, Ready: true, Big: big, Var: v})
	}
	// One giant of each look (picked by the tile, as in the game).
	look := func(dx, dy int16) uint32 { return sim.HashTile(int(cx+dx), int(cy+dy)) / 8 % 2 }
	put(sim.EntityGiantTree, -7, 3, true, 1) // with a hive
	gx, gy := int16(8), int16(4)
	for _, d := range [][2]int16{{8, 4}, {9, 4}, {7, 4}, {8, 5}, {9, 3}, {10, 4}, {7, 3}} {
		if look(d[0], d[1]) != look(-7, 3) {
			gx, gy = d[0], d[1]
			break
		}
	}
	put(sim.EntityGiantTree, gx, gy, true, 0)
	put(sim.EntityTree, -1, -4, true, 0)
	put(sim.EntityTree, 3, -6, true, 0)
	put(sim.EntityTree, -12, -5, false, 0)
	put(sim.EntityTree, 12, -4, true, 0)
	put(sim.EntityBananaTree, 3, 1, true, 0)
	put(sim.EntityBananaTree, -3, 2, true, 0)
	put(sim.EntityBerryBush, 1, 3, false, 0)
	put(sim.EntityGrassTuft, -1, 2, false, 0)
	cid := sim.CreatureID(1 << 28)
	monkey := func(dx, dy float64, st sim.CreatureState, left bool) {
		cid++
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{ID: cid, Kind: sim.CreatureMonkey, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left})
	}
	monkey(4.2, 1.6, sim.CreatureEat, true)
	monkey(-4.5, 4.2, sim.CreatureIdle, false)
}

// swimSpot is the middle of the water tile nearest (sx, sy) of the kind
// asked for (see -swimat), or (sx, sy) if there is none.
func swimSpot(w *sim.World, kind string, sx, sy float64) (float64, float64) {
	N, H := w.Size()
	tiles := w.Tiles()
	at := func(x, y int) sim.Tile {
		if x < 0 || y < 0 || x >= N || y >= H {
			return sim.TileWater
		}
		return tiles[y*N+x]
	}
	all := func(tx, ty, r int, ok func(sim.Tile) bool) bool {
		for y := ty - r; y <= ty+r; y++ {
			for x := tx - r; x <= tx+r; x++ {
				if !ok(at(x, y)) {
					return false
				}
			}
		}
		return true
	}
	water := func(t sim.Tile) bool { return t == sim.TileWater }
	want := func(tx, ty int) bool {
		t := at(tx, ty)
		switch kind {
		case "deep":
			return water(t) && all(tx, ty, 6, water)
		case "mid":
			return water(t) && all(tx, ty, 1, water) && !all(tx, ty, 3, water)
		case "shallow":
			return t == sim.TileShallow && all(tx, ty, 1, func(t sim.Tile) bool { return t == sim.TileShallow })
		case "ford":
			return t == sim.TileFord
		case "river":
			return t == sim.TileRiver && all(tx, ty, 1, func(t sim.Tile) bool { return t == sim.TileRiver })
		}
		return false
	}
	best, bx, by := math.Inf(1), sx, sy
	for ty := 4; ty < H-4; ty++ {
		for tx := 4; tx < N-4; tx++ {
			if d := math.Hypot(float64(tx)-sx, float64(ty)-sy); d < best && want(tx, ty) {
				best, bx, by = d, float64(tx)+0.5, float64(ty)+0.5
			}
		}
	}
	return bx, by
}
