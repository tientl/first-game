package client

import (
	"math"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The server sends state, not events, so sounds are inferred by comparing
// each snapshot with the one before it: a player who just started swinging,
// a tree that became a stump, a fuller bag, a new notice.

// soundMemory is what the client keeps between frames for sound.
type soundMemory struct {
	stepFrame  map[sim.PlayerID]int
	lastHurt   time.Time
	lastShiver time.Time
	ocean      float64 // share of water around the listener, refreshed now and then
	forest     float64
	swimState  // see swimsound.go
}

func (g *Game) snapshotSounds(old, cur protocol.Snapshot) {
	if snd == nil || g.demo {
		return
	}
	g.thunderSounds(cur)
	oldEnts := make(map[sim.EntityID]protocol.EntityView, len(old.Entities))
	oldAt := make(map[[2]int]protocol.EntityView, len(old.Entities))
	for _, e := range old.Entities {
		oldEnts[e.ID] = e
		oldAt[[2]int{int(e.TX), int(e.TY)}] = e
	}
	curEnts := make(map[sim.EntityID]protocol.EntityView, len(cur.Entities))
	for _, e := range cur.Entities {
		curEnts[e.ID] = e
	}
	wasActing := make(map[sim.PlayerID]bool, len(old.Players))
	wasFishing := make(map[sim.PlayerID]bool, len(old.Players))
	for _, p := range old.Players {
		wasActing[p.ID] = p.Acting
		wasFishing[p.ID] = p.Cast != sim.CastNone
	}
	at := func(e protocol.EntityView) (float64, float64) { return float64(e.TX) + 0.5, float64(e.TY) + 0.5 }

	// Swings: each start of the working pose is one hit on what the player
	// was facing, judged by the world as it was before the hit landed.
	for _, p := range cur.Players {
		if !p.Acting || wasActing[p.ID] || wasFishing[p.ID] {
			continue // a strike on a line is heard in fishingSounds
		}
		if pickedUp(old, cur, p) {
			continue // E picked something up: no blow; my own pick-up is heard from the bag below
		}
		hasWood := true
		if p.ID == g.myID {
			hasWood = old.Self.Inv.HasFuel()
		}
		px, py := float64(p.X), float64(p.Y)
		tg, ok := sim.ResolveTarget(px, py, p.Aim, float64(p.AimX), float64(p.AimY), hasWood, func(tx, ty int) (sim.TargetCandidate, bool) {
			e, ok := oldAt[[2]int{tx, ty}]
			return sim.TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready, Var: e.Var}, ok
		}, creatureCandidates(old.Creatures, px, py), sim.ReachBonusFor(p.Held)+sim.RideReach(p.Mount != 0))
		if !ok {
			continue
		}
		if tg.Creature != 0 {
			for _, c := range old.Creatures {
				if c.ID != tg.Creature {
					continue
				}
				swingAtCreatureSound(c)
				switch {
				case c.Kind == sim.CreatureFish && p.Held == sim.ItemRod:
					snd.playAt("cast", float64(p.X), float64(p.Y), 0.45, 0.06)
				case p.Held == sim.ItemSpear || p.Held == sim.ItemIvorySpear:
					snd.playAt("swish", float64(p.X), float64(p.Y), 0.5, 0.1)
				}
			}
			continue
		}
		e := oldEnts[tg.Entity]
		x, y := at(e)
		g.toolHitFX(e, x, y, math.Atan2((y-py)*TileH, (x-px)*TileW), g.swingWhat[p.ID].kind)
		switch e.Kind {
		case sim.EntityTree:
			snd.playAt("chop", x, y, 0.75, 0.06)
		case sim.EntityFigTree, sim.EntityGinkgoTree, sim.EntityPalm, sim.EntityBananaTree:
			if !e.Ready { // bare: the axe; ripe fruit is picked by hand
				snd.playAt("chop", x, y, 0.7, 0.06)
			}
		case sim.EntityBoulder:
			if g.swingWhat[p.ID].kind == swingPick { // the sound of the tool drawn swinging
				snd.playAt("pick_hit", x, y, 0.6, 0.06)
			} else {
				snd.playAt("stone", x, y, 0.6, 0.06)
			}
		case sim.EntityWall, sim.EntityCactus:
			snd.playAt("chop", x, y, 0.6, 0.08)
		case sim.EntityCoal, sim.EntityCopperOre, sim.EntityIronOre, sim.EntityPlatinumOre, sim.EntityDiamondOre, sim.EntityStoneWall:
			snd.playAt("pick_hit", x, y, 0.6, 0.06)
		case sim.EntityReed, sim.EntityMushroom, sim.EntityClam, sim.EntityFlora:
			snd.playAt("rustle", x, y, 0.4, 0.12)
		case sim.EntityBerryBush, sim.EntityGrassTuft:
			snd.playAt("rustle", x, y, 0.5, 0.1)
		case sim.EntityTwig:
			snd.playAt("rustle", x, y, 0.3, 0.15)
		case sim.EntityCampfire, sim.EntityFurnace:
			snd.playAt("stoke", x, y, 0.6, 0.05)
		}
	}

	// What changed in the world. Things merely leaving the snapshot's range
	// are far away, so hearing range already keeps them silent.
	for id, e := range oldEnts {
		c, still := curEnts[id]
		x, y := at(e)
		switch {
		case e.Kind == sim.EntityTree && e.Ready && (!still || !c.Ready):
			snd.playAt("treefall", x, y, bigLouder(e, 0.75), 0.04)
		case still && sim.IsFruitTree(e.Kind) && !sim.IsFruitStump(e.Kind, e.Var) && sim.IsFruitStump(c.Kind, c.Var):
			snd.playAt("treefall", x, y, 0.7, 0.04) // a fruit tree felled
		case (e.Kind == sim.EntityBoulder || e.Kind == sim.EntityCoal || e.Kind == sim.EntityStoneWall || isOre(e.Kind)) && !still:
			snd.playAt("rockbreak", x, y, bigLouder(e, 0.7), 0.05)
		case e.Kind == sim.EntityCampfire && !still && e.Fuel < sim.TicksPerSecond*2:
			snd.playAt("fizzle", x, y, 0.5, 0.05)
		}
	}
	for id, c := range curEnts {
		x, y := at(c)
		if e, seen := oldEnts[id]; seen {
			if e.Kind == sim.EntityFarmland && c.Kind == sim.EntityCrop {
				snd.playAt("rustle", x, y, 0.45, 0.1) // planted
			}
			continue
		}
		switch {
		case c.Kind == sim.EntityCampfire && c.Fuel > sim.FireStartFuel-sim.TicksPerSecond:
			snd.playAt("ignite", x, y, 0.7, 0.03)
		case c.Kind == sim.EntityWall || c.Kind == sim.EntityStoneWall || sim.IsBedLike(c.Kind):
			snd.playAt("build", x, y, 0.6, 0.05)
		case c.Kind == sim.EntityFarmland:
			snd.playAt("step_sand", x, y, 0.9, 0.1) // tilled
		}
	}

	// Torches lighting up and players lying down or getting up.
	wasP := make(map[sim.PlayerID]protocol.PlayerView, len(old.Players))
	for _, p := range old.Players {
		wasP[p.ID] = p
	}
	for _, p := range cur.Players {
		q, ok := wasP[p.ID]
		if !ok {
			continue
		}
		x, y := float64(p.X), float64(p.Y)
		lit := p.Held == sim.ItemTorch && !p.Dead && !p.Asleep
		wasLit := q.Held == sim.ItemTorch && !q.Dead && !q.Asleep
		if lit && !wasLit {
			snd.playAt("torch_on", x, y, 0.5, 0.05)
		}
		switch {
		case p.Asleep && !q.Asleep:
			snd.playAt("sleep", x, y, 0.45, 0)
		case !p.Asleep && q.Asleep && !p.Dead:
			snd.playAt("wake", x, y, 0.4, 0)
		}
	}

	g.creatureSounds(old, cur)
	g.fishingSounds(old, cur)

	// My own state.
	was, now := old.Self, cur.Self
	pickups := map[sim.ItemKind]string{
		sim.ItemWood: "pick_wood", sim.ItemStone: "pick_stone",
		sim.ItemGrass: "pick_grass", sim.ItemBerry: "pick_berry",
		sim.ItemStick: "pick_stick", sim.ItemMeat: "pick_meat",
		sim.ItemWorm: "pick_worm", sim.ItemFish: "pick_meat", sim.ItemCrabMeat: "pick_meat",
		sim.ItemPebble: "pick_stone", sim.ItemPotato: "pick_grass",
		sim.ItemFig: "pick_berry", sim.ItemGinkgoNut: "pick_berry", sim.ItemCoconut: "pick_wood",
		sim.ItemCactus: "pick_grass", sim.ItemMushroom: "pick_berry", sim.ItemCoal: "pick_stone",
		sim.ItemClam:   "pick_stone",
		sim.ItemBamboo: "pick_wood", sim.ItemGourd: "pick_berry", sim.ItemPitcherWater: "pick_berry",
		sim.ItemFlax: "pick_grass", sim.ItemVine: "pick_grass", sim.ItemPoisonMushroom: "pick_berry",
		sim.ItemNightshade: "pick_berry", sim.ItemAloe: "pick_grass", sim.ItemHerb: "pick_grass",
		sim.ItemMoss: "pick_grass", sim.ItemSeaweed: "pick_grass", sim.ItemLily: "pick_berry",
		sim.ItemLotusSeed: "pick_berry", sim.ItemHorsetail: "pick_grass", sim.ItemRice: "pick_grass",
		sim.ItemCorn: "pick_berry", sim.ItemSquash: "pick_berry",
	}
	wasN, nowN := was.Inv.Counts(), now.Inv.Counts()
	for k, name := range pickups {
		if nowN[k] > wasN[k] {
			snd.playVar(name, 0.45, 0, 0.04)
		}
	}
	for k := sim.ItemKind(1); k < sim.NumItems; k++ {
		if k.Edible() && nowN[k] < wasN[k] && now.Hunger > was.Hunger {
			snd.playVar("eat", 0.6, 0, 0.06)
			break
		}
	}
	if now.NoticeSeq != was.NoticeSeq {
		switch now.Notice {
		case sim.NoticeCrafted:
			if r, ok := sim.RecipeByID(sim.RecipeID(now.NoticeArg)); ok && r.NearFire {
				snd.playVar("sizzle", 0.5, 0, 0.06)
			} else {
				snd.play("craft", 0.6, 0, 1)
			}
		case sim.NoticeFertilized:
			snd.playVar("pick_grass", 0.6, 0, 0.08)
		case sim.NoticeTossed:
			tossSound() // see toss.go
		case sim.NoticeCantToss, sim.NoticeNotEnough, sim.NoticeNoSpace, sim.NoticeNotEdible, sim.NoticeNoneLeft, sim.NoticeBagFull,
			sim.NoticeNeedTool, sim.NoticeHaveOne, sim.NoticeNeedBait, sim.NoticeNeedFire, sim.NoticeNotTired, sim.NoticeTooFar, sim.NoticeNothingNear, sim.NoticeHardGround:
			// Holding Space at a tree with no axe repeats the notice twice
			// a second; one buzz now and then is enough.
			if snd.ready("error", 1500*time.Millisecond) {
				snd.play("error", 0.4, 0, 1)
			}
		}
	}
	switch {
	case now.Dead && !was.Dead:
		snd.play("death", 0.7, 0, 1)
	case !now.Dead && was.Dead:
		snd.play("respawn", 0.6, 0, 1)
	case !now.Dead && now.Health < was.Health-0.01 && time.Since(g.sm.lastHurt) > 1800*time.Millisecond:
		g.sm.lastHurt = time.Now()
		snd.playVar("hurt", 0.45, 0, 0.08)
	}
	if now.Cold && !now.Dead && (!was.Cold || time.Since(g.sm.lastShiver) > 6*time.Second) {
		g.sm.lastShiver = time.Now()
		snd.playVar("shiver", 0.4, 0, 0.05)
	}
}

// frameSounds runs every frame: footsteps, ambience and the choice of music.
func (g *Game) frameSounds(alpha float64) {
	if snd == nil || g.demo {
		return
	}
	if g.disconnected || !g.ready || !g.hasCurr {
		snd.silenceLoops()
		return
	}
	if g.sm.stepFrame == nil {
		g.sm.stepFrame = make(map[sim.PlayerID]int)
	}
	g.sm.swimState.init()

	g.swarmSounds() // see jungleart.go
	lx, ly := float64(g.width)/2, float64(g.height)/2
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			lx, ly, _ = g.lerpPlayer(p, alpha)
		}
	}
	snd.setListener(lx, ly)

	// Footsteps land on the walk cycle's contact frames.
	for _, p := range g.curr.Players {
		x, y, moving := g.lerpPlayer(p, alpha)
		f := -1
		if g.swimSounds(p, x, y, moving) || p.Boat != 0 { // see swimsound.go
			moving = false // strokes in the water, no steps; the boat carries them
		}
		if moving && !p.Dead {
			f = walkFrameAt(g.frame)
		}
		if f != g.sm.stepFrame[p.ID] && (f == 1 || f == 5) {
			name := "step_grass"
			switch g.tileAt(int(math.Floor(x)), int(math.Floor(y))) {
			case sim.TileSand:
				name = "step_sand"
			case sim.TileRocky:
				name = "step_rock"
			case sim.TileForest:
				name = "step_forest"
			case sim.TileSnow:
				name = "step_snow"
			}
			if p.ID == g.myID {
				snd.playVar(name, 0.28, 0, 0.08)
			} else {
				snd.playAt(name, x, y, 0.25, 0.08)
			}
		}
		g.sm.stepFrame[p.ID] = f
	}

	// What the surroundings sound like.
	if g.frame%15 == 0 {
		water, forest, n := 0, 0, 0
		cx, cy := int(lx), int(ly)
		for ty := cy - 9; ty <= cy+9; ty += 2 {
			for tx := cx - 12; tx <= cx+12; tx += 2 {
				switch g.tileAt(tx, ty) {
				case sim.TileWater:
					water++
				case sim.TileForest:
					forest++
				}
				n++
			}
		}
		g.sm.ocean = float64(water) / float64(n)
		g.sm.forest = float64(forest) / float64(n)
	}
	amb := sim.Ambient(g.curr.Tick)
	night := 1 - amb

	fire := 0.0
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityCampfire || e.Fuel <= 0 {
			continue
		}
		d := math.Hypot(float64(e.TX)+0.5-lx, (float64(e.TY)+0.5-ly)*1.3)
		strength := 0.4 + 0.6*min(1, float64(e.Fuel)/sim.FireStartFuel)
		fire = max(fire, math.Pow(clamp01(1-d/9), 2)*strength)
	}
	for _, p := range g.curr.Players { // a torch crackles too, more quietly
		if p.Held == sim.ItemTorch && !p.Dead && !p.Asleep {
			d := math.Hypot(float64(p.X)-lx, (float64(p.Y)-ly)*1.3)
			fire = max(fire, math.Pow(clamp01(1-d/6), 2)*0.35)
		}
	}
	// Underground the outside goes quiet: drips and echoes instead.
	out := 1 - g.caveIn()
	snd.setLoop("ocean", min(1, g.sm.ocean*2.5)*0.5*out)
	snd.setLoop("fire", fire*0.7)
	wind := 0.1 + 0.08*night
	if g.curr.Weather.Windy() {
		wind = 0.55
	}
	rain := 0.0
	if g.curr.Weather.Wet() {
		rain = 0.45
		if g.curr.Weather == sim.WeatherStorm {
			rain = 0.7
		}
	}
	snd.setLoop("wind", wind*out)
	snd.setLoop("rain", rain*out)
	snd.setLoop("crickets", math.Pow(night, 1.5)*0.3*(1-g.sm.ocean)*out)
	snd.setLoop("cave", 0.5*g.caveIn())

	// Birdsong by day, more of it near the woods.
	if out > 0.5 && amb > 0.6 && !g.paused && snd.rng.Float64() < (0.004+0.01*g.sm.forest)*(1-g.sm.ocean) {
		snd.play("bird", 0.1+snd.rng.Float64()*0.2, snd.rng.Float64()*1.6-0.8, 0.9+snd.rng.Float64()*0.25)
	}

	g.bossWinSound() // see bossmusic.go
	switch {
	case g.curr.Boss.Kind.Boss() && !g.curr.Self.Dead:
		snd.setMusic(bossTrack(g.curr.Boss.Kind)) // a boss fight; see bossmusic.go
	case g.caveIn() > 0.5:
		snd.setMusic("cave")
	case amb < 0.35:
		snd.setMusic("night")
	default:
		snd.setMusic("day")
	}
	snd.update()
}

// bigLouder raises the volume of a sound made by a big tree or boulder.
func bigLouder(e protocol.EntityView, vol float64) float64 {
	if e.Big {
		return min(1, vol*1.3)
	}
	return vol
}

// thunderSounds rolls thunder for each new strike of lightning, louder and
// sooner the closer it fell.
func (g *Game) thunderSounds(cur protocol.Snapshot) {
	if g.strikesSeen == nil {
		g.strikesSeen = make(map[[2]float32]bool)
	}
	for _, s := range cur.Strikes {
		k := [2]float32{s.X, s.Y}
		if g.strikesSeen[k] {
			continue
		}
		g.strikesSeen[k] = true
		callAt("thunder", float64(s.X), float64(s.Y), 1, 1, 40) // heard 40 tiles off; see roar.go
	}
	if len(g.strikesSeen) > 64 {
		clear(g.strikesSeen)
		for _, s := range cur.Strikes {
			g.strikesSeen[[2]float32{s.X, s.Y}] = true
		}
	}
}
