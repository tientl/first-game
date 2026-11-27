package server

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// snapshotFor builds the view of the world for player id: its own state plus
// the players and entities within the interest radius. players is the
// current player list (shared across all snapshots of one tick). Every
// returned slice is freshly allocated, because the snapshot is handed to
// another goroutine.
func (s *Server) snapshotFor(id sim.PlayerID, players []sim.Player) (protocol.Snapshot, bool) {
	var self *sim.Player
	for i := range players {
		if players[i].ID == id {
			self = &players[i]
			break
		}
	}
	if self == nil {
		return protocol.Snapshot{}, false
	}

	tick := s.world.Tick()
	snap := protocol.Snapshot{
		Tick: tick,
		Self: protocol.SelfState{
			Health:    float32(self.Health),
			TorchFuel: int32(self.TorchFuel()),
			Hunger:    float32(self.Hunger),
			Inv:       self.Inv,
			Selected:  uint8(self.Selected),
			Dead:      self.Dead,
			Cold:      self.Cold,
			Poisoned:  self.Poisoned(),
			Webbed:    self.Webbed(),
			HomeBed:   self.HomeBed() != 0,
			Boat:      self.BoatID(),
			Breath:    float32(self.Breath()),
			Notice:    self.Notice,
			NoticeArg: self.NoticeArg,
			NoticeSeq: self.NoticeSeq,
			DevBuild:  s.world.DevFreeBuild(id), // only ever true in dev mode
		},
	}
	if id := self.OpenChest(); id != 0 {
		if inv, ok := s.world.ChestContents(id); ok {
			snap.Self.Chest, snap.Self.ChestSlots = id, inv
		}
	}

	if id := self.OpenCook(); id != 0 {
		if c, ok := s.world.CookContents(id); ok {
			snap.Self.Cook = protocol.CookView{ID: id, Kind: c.Kind, Slots: c.Slots, Dish: c.Dish, State: c.State,
				Progress: int32(c.Progress), Total: int32(c.Total), Fuel: int32(c.Fuel)}
		}
	}
	for i, t := range self.Buffs() {
		snap.Self.Cook.Buffs[i] = int32(t)
	}

	r := float64(s.cfg.InterestRadius)
	for i := range players {
		p := &players[i]
		cast, bobX, bobY := p.Cast()
		deed, deedItem, deedSeq := p.Deed()
		if math.Abs(p.X-self.X) > r || math.Abs(p.Y-self.Y) > r {
			continue
		}
		axeTier, hasAxe := p.Inv.BestTool(sim.ToolAxe)
		armor, _ := p.Inv.BestArmor() // stone (0) when none
		ax, ay, aim := p.Aim()
		snap.Players = append(snap.Players, protocol.PlayerView{
			ID:        p.ID,
			Name:      p.Name,
			X:         float32(p.X),
			Y:         float32(p.Y),
			Dir:       p.Dir,
			Dead:      p.Dead,
			Acting:    p.Acting(tick),
			Asleep:    p.Asleep,
			Look:      p.Look,
			Mount:     p.Mount(),
			DinoArmor: p.Inv.Count(sim.ItemDinoArmor) > 0,
			Boat:      p.BoatID(),
			Swimming:  s.world.Swimming(p),
			Bees:      p.Swarmed(),
			HurtSeq:   p.HurtSeq(),
			DeedSeq:   deedSeq,
			Deed:      deed,
			DeedItem:  deedItem,
			HasAxe:    hasAxe,
			AxeTier:   axeTier,
			Armor:     armor,
			Wear:      p.Inv.Worn(),
			Held:      p.Held(),
			Cast:      cast,
			BobX:      float32(bobX),
			BobY:      float32(bobY),
			Aim:       aim,
			AimX:      float32(ax),
			AimY:      float32(ay),
			Shield:    p.HasShield(),
			Guard:     p.Guarding(),
			GuardSeq:  p.GuardSeq(),
			Parry:     p.Parried(),
		})
	}

	cx, cy := int(math.Floor(self.X)), int(math.Floor(self.Y))
	ri := s.cfg.InterestRadius
	s.world.EachEntityIn(cx-ri, cy-ri, cx+ri, cy+ri, func(e sim.Entity) {
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID:    e.ID,
			Kind:  e.Kind,
			TX:    int16(e.TX), // world size is capped at sim.MaxWorldSize
			TY:    int16(e.TY),
			Ready: e.Ready,
			Big:   e.Big,
			Stage: uint8(e.GrowthStage(tick)),
			Fuel:  entityFuel(e, tick),
			Var:   e.Var,
			Cook:  e.Cook,
		})
	})
	snap.Season, snap.Weather = sim.SeasonOf(tick), s.world.Weather()
	s.world.EachDropIn(self.X-r, self.Y-r, self.X+r, self.Y+r, func(d sim.Drop) {
		snap.Drops = append(snap.Drops, protocol.DropView{
			ID: d.ID, Item: d.Item, Count: uint8(min(d.Count, 255)),
			X: float32(d.X), Y: float32(d.Y), FromX: float32(d.FromX), FromY: float32(d.FromY),
			Age: uint16(min(tick-d.Born, 60000)),
		})
	})
	for _, b := range s.world.Webs() {
		if math.Abs(b.X-self.X) <= r && math.Abs(b.Y-self.Y) <= r {
			snap.Webs = append(snap.Webs, protocol.WebView{ID: b.ID, X: float32(b.X), Y: float32(b.Y), VX: float32(b.VX), VY: float32(b.VY)})
		}
	}
	for _, st := range s.world.Strikes() {
		if math.Abs(st.X-self.X) <= 2*r && math.Abs(st.Y-self.Y) <= 2*r {
			snap.Strikes = append(snap.Strikes, protocol.StrikeView{X: float32(st.X), Y: float32(st.Y), Age: uint8(min(tick-st.Tick, 255))})
		}
	}
	for _, b := range s.world.Boats() {
		if math.Abs(b.X-self.X) <= r && math.Abs(b.Y-self.Y) <= r {
			snap.Boats = append(snap.Boats, protocol.BoatView{ID: b.ID, Kind: b.Kind, X: float32(b.X), Y: float32(b.Y),
				Heading: float32(b.Heading), Speed: float32(b.Speed), HP: int16(b.HP), Rider: b.Rider, Held: b.Held > 0})
		}
	}
	wd, ws := s.world.Wind()
	snap.WindDir, snap.Wind = float32(wd), float32(ws)
	for _, t := range s.world.Tornadoes() {
		if math.Abs(t.X-self.X) <= 2*r && math.Abs(t.Y-self.Y) <= 2*r {
			snap.Tornadoes = append(snap.Tornadoes, protocol.TornadoView{ID: t.ID, X: float32(t.X), Y: float32(t.Y)})
		}
	}
	s.world.EachCreatureIn(self.X-r, self.Y-r, self.X+r, self.Y+r, func(c sim.Creature) {
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID:       c.ID,
			Kind:     c.Kind,
			X:        float32(c.X),
			Y:        float32(c.Y),
			State:    c.State,
			FaceLeft: c.FaceLeft,
			Owner:    c.Owner,
			Tamed:    c.Tamed(),
			Tame:     uint8(min(c.Tame, 255)),
			Stay:     c.Stay,
			Ridden:   c.Ridden(),
			HP:       int16(min(c.HP, 32767)),
			MaxHP:    int16(min(c.Kind.MaxHP(), 32767)),
		})
	})
	s.bossViews(&snap, id) // see bosses.go
	return snap, true
}

// entityFuel is what a client is told of e's fuel: for a man-eater, the
// ticks until it may snap again (the client shows the snap from it).
func entityFuel(e sim.Entity, tick uint64) int32 {
	if e.Kind == sim.EntityManEater {
		if e.RegrowAt > tick {
			return int32(min(e.RegrowAt-tick, 1<<20))
		}
		return 0
	}
	if e.HasHive() { // ticks till the hive has honey again; 0 when it has
		return int32(min(e.HoneyAt-min(e.HoneyAt, tick), 1<<20))
	}
	return int32(e.Fuel)
}
