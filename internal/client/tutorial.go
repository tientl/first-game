package client

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The first minutes' guide: a short chain of goals, one at a time in a
// small tracker under the vitals (tutorialdraw.go), and a few tips said
// once each when the moment comes (a notice, see hudtoasts.go).
//
// It is worked out from what the client is already told (the bag, the
// things standing near, the player's food, the level underground): no part
// of the game's rules or protocol knows of it. How far each world's guide
// has got is kept in tutorial.json next to settings.json, under the world's
// seed and the player's name, so "Chơi tiếp" carries on where it stopped and
// a finished guide never comes back. The tips already said are kept there
// too, for every world. Only a new game starts a guide; a save that has no
// record (a game from before the guide, or played with it off) has none.
// The guide never shows in a dev game, and the settings can turn it off.

// tutStep is one goal.
type tutStep struct {
	title string // the goal, in two lines at most
	key   string // the key or button to use, in a little cap
	hint  string // how, in two lines at most
	done  func(v *tutView) bool
}

// tutView is what a step is judged by: one snapshot, and what the guide
// has seen of the player's food.
type tutView struct {
	inv      [sim.NumItems]int
	me       protocol.PlayerView
	haveMe   bool
	hunger   float32
	cold     bool
	under    bool // in the natural caves
	fireNear bool // a burning campfire close by
	tentNear bool // a leaf tent standing close by
	ate      bool // the player has eaten since the guide began
	torchLit bool
}

// tutNear is how close (in tiles) a campfire or tent counts as the
// player's.
const tutNear = 14

var tutSteps = []tutStep{
	{"Nhặt cành cây và đá cuội", "E", "đứng cạnh rồi nhặt: mỗi thứ vài cái",
		func(v *tutView) bool {
			return v.inv[sim.ItemStick] >= 2 && v.inv[sim.ItemPebble] >= 1 || v.has(sim.ItemAxe)
		}},
	{"Làm Rìu đá", "C", "mở bảng chế tạo, nhấp Rìu đá",
		func(v *tutView) bool { return v.has(sim.ItemAxe) }},
	{"Chặt cây lấy gỗ", "Chuột trái", "giữ chuột vào thân cây: lấy 2 gỗ",
		func(v *tutView) bool { return v.inv[sim.ItemWood] >= 2 || v.fireNear }},
	{"Gom 3 đá và 2 cỏ khô", "Chuột trái", "làm Cúp đá để đập tảng đá; cắt bụi cỏ",
		func(v *tutView) bool { return v.inv[sim.ItemStone] >= 3 && v.inv[sim.ItemGrass] >= 2 || v.fireNear }},
	{"Đốt lửa trại", "C", "chọn Lửa trại, nhấp chỗ đặt. Đêm lạnh: đứng gần lửa",
		func(v *tutView) bool { return v.fireNear }},
	{"Ăn gì đó", "Chuột phải", "cầm quả mọng hoặc thịt nướng trên lửa",
		func(v *tutView) bool { return v.ate }},
	{"Làm đuốc", "C", "cầm đuốc khi vào hang tối",
		func(v *tutView) bool { return v.inv[sim.ItemTorch] > 0 || v.torchLit || v.under }},
	{"Làm Lều lá để ngủ qua đêm", "V", "làm trong C, rồi đặt xuống đất",
		func(v *tutView) bool { return v.inv[sim.ItemLeafTent] > 0 || v.tentNear }},
	{"Tìm cửa hang và bước vào", "", "hang có đá quý, nhưng tối và nguy hiểm",
		func(v *tutView) bool { return v.under }},
}

func (v *tutView) has(k sim.ItemKind) bool { return v.inv[k] > 0 }

// The tips, in the order they are looked for.
const (
	tipWindup   = "windup"
	tipCold     = "cold"
	tipHungry   = "hungry"
	tipCaveDark = "cavedark"
	tipLair     = "lair"
	tipNight    = "night"
)

// tutTipGap is the least time between two tips.
const tutTipGap = 8 * time.Second

// tutorialFile is tutorial.json.
type tutorialFile struct {
	Worlds map[string]tutorialWorld `json:"worlds,omitempty"`
	Tips   []string                 `json:"tips,omitempty"` // said once, in any world
}

// tutorialWorld is one world's guide: the goal it is at, or that it is over.
type tutorialWorld struct {
	Step int  `json:"step"`
	Done bool `json:"done,omitempty"`
}

// tutorial is a game's guide.
type tutorial struct {
	path string // where it is kept; "" keeps it nowhere
	key  string
	file tutorialFile

	step int  // the goal now; len(tutSteps) when all are done
	over bool // the guide is finished (or never begun): no tracker

	lastHunger float32
	haveHunger bool
	ate        bool

	// For the drawing.
	flash      int // the goal last done, shown ticked for a moment (-1: none)
	flashAt    time.Time
	finishedAt time.Time

	lastTip time.Time

	still bool // held where it is, for the screenshot tool
}

// tutorialPath is where the guide is kept: next to the settings.
func tutorialPath() string {
	if settingsPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(settingsPath), "tutorial.json")
}

// worldKey names a world's guide: the world's seed and the player.
func worldKey(seed uint64, player string) string {
	b, _ := json.Marshal([]any{seed, player})
	return string(b)
}

// loadTutorial opens the guide of world key at path. A world with no record
// begins its guide if fresh (a new game), and has none otherwise.
func loadTutorial(path, key string, fresh bool) *tutorial {
	t := &tutorial{path: path, key: key, flash: -1}
	if b, err := os.ReadFile(path); err == nil && path != "" {
		if err := json.Unmarshal(b, &t.file); err != nil {
			log.Printf("tutorial: %s: %v (starting over)", path, err)
			t.file = tutorialFile{}
		}
	}
	w, ok := t.file.Worlds[key]
	switch {
	case ok:
		t.step, t.over = max(0, min(w.Step, len(tutSteps))), w.Done
	case fresh:
		t.save()
	default:
		t.over = true
		t.save()
	}
	return t
}

// save writes the guide's progress.
func (t *tutorial) save() {
	if t.file.Worlds == nil {
		t.file.Worlds = map[string]tutorialWorld{}
	}
	t.file.Worlds[t.key] = tutorialWorld{Step: t.step, Done: t.over}
	t.flush()
}

// flush writes the file, through a temporary one.
func (t *tutorial) flush() {
	if t.path == "" {
		return
	}
	b, err := json.MarshalIndent(t.file, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		log.Printf("tutorial: %v", err)
		return
	}
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		log.Printf("tutorial: saving: %v", err)
		return
	}
	if err := os.Rename(tmp, t.path); err != nil {
		log.Printf("tutorial: saving: %v", err)
		_ = os.Remove(tmp)
	}
}

// viewOf works out what the steps judge from a snapshot, and notes the
// player's meals.
func (t *tutorial) viewOf(s *protocol.Snapshot, me sim.PlayerID, under bool) *tutView {
	v := &tutView{inv: s.Self.Inv.Counts(), hunger: s.Self.Hunger, cold: s.Self.Cold, under: under}
	for _, p := range s.Players {
		if p.ID == me {
			v.me, v.haveMe = p, true
		}
	}
	if t.haveHunger && s.Self.Hunger > t.lastHunger+2 {
		t.ate = true
	}
	t.lastHunger, t.haveHunger = s.Self.Hunger, true
	v.ate = t.ate
	v.torchLit = s.Self.TorchFuel > 0 || v.me.Held == sim.ItemTorch
	if v.haveMe {
		for _, e := range s.Entities {
			dx, dy := float64(e.TX)+0.5-float64(v.me.X), float64(e.TY)+0.5-float64(v.me.Y)
			if dx*dx+dy*dy > tutNear*tutNear {
				continue
			}
			switch e.Kind {
			case sim.EntityCampfire:
				v.fireNear = v.fireNear || e.Fuel > 0
			case sim.EntityLeafTent:
				v.tentNear = true
			}
		}
	}
	return v
}

// observe takes in a snapshot: it moves the guide on past the goals now
// done (several at once if they are), and says which tip, if any, is due.
// advanced reports a goal done, for its sound.
func (t *tutorial) observe(s *protocol.Snapshot, me sim.PlayerID, under bool, now time.Time) (advanced bool, tip string) {
	v := t.viewOf(s, me, under)
	if !t.over {
		for t.step < len(tutSteps) && tutSteps[t.step].done(v) {
			t.flash, t.flashAt = t.step, now
			t.step++
			advanced = true
		}
		if t.step >= len(tutSteps) {
			t.over = true
			t.finishedAt = now
		}
		if advanced {
			t.save()
		}
	}
	if v.haveMe && !v.me.Dead && now.Sub(t.lastTip) >= tutTipGap {
		if id, text := t.tipDue(s, v); id != "" {
			t.file.Tips = append(t.file.Tips, id)
			t.lastTip = now
			t.flush()
			tip = text
		}
	}
	return advanced, tip
}

// said reports whether tip id has been said.
func (t *tutorial) said(id string) bool { return slices.Contains(t.file.Tips, id) }

// tipDue is the tip to say now, if any has not been said and its moment is
// here.
func (t *tutorial) tipDue(s *protocol.Snapshot, v *tutView) (id, text string) {
	switch {
	case !t.said(tipWindup) && inRedZone(s, v.me):
		text = "Vùng đỏ là chỗ thú sắp đánh trúng: chạy ra khỏi đó ngay."
		if v.me.Shield {
			text = "Vùng đỏ là chỗ thú sắp đánh trúng: chạy ra, hoặc giữ Q giơ khiên đỡ đòn."
		}
		return tipWindup, text
	case !t.said(tipCold) && s.Self.Cold:
		return tipCold, "Lạnh rồi: đứng gần lửa trại hoặc cầm đuốc cho ấm."
	case !t.said(tipHungry) && float64(v.hunger) < sim.MaxHunger*0.2:
		return tipHungry, "Đói rồi: cầm quả mọng và bấm chuột phải để ăn, hoặc nướng thịt trên lửa."
	case !t.said(tipCaveDark) && v.under && !v.torchLit:
		return tipCaveDark, "Trong hang tối om: chọn đuốc trên thanh đồ để thấy đường."
	case !t.said(tipLair) && lairInSight(s):
		return tipLair, "Ổ của một con trùm! Mang đồ tốt và nhiều thức ăn rồi hãy lại gần."
	case !t.said(tipNight) && !v.under && sim.PhaseOf(s.Tick) != sim.PhaseDay:
		return tipNight, "Trời sắp tối: nhóm lửa hoặc cầm đuốc cho ấm. Đêm thú săn mồi ra ngoài."
	}
	return "", ""
}

// inRedZone reports whether the player stands in a warning on the ground
// that has not yet landed (near enough: the zones' shapes are only judged
// by their reach).
func inRedZone(s *protocol.Snapshot, me protocol.PlayerView) bool {
	for _, z := range s.Zones {
		if z.Left == 0 {
			continue
		}
		reach := float64(z.Reach)
		if z.Shape != sim.ZoneCircle {
			reach += float64(z.Width)
		}
		if math.Hypot(float64(z.X-me.X), float64(z.Y-me.Y)) <= reach+0.5 {
			return true
		}
	}
	return false
}

// lairInSight reports whether a boss's lair not yet beaten is near.
func lairInSight(s *protocol.Snapshot) bool {
	for _, l := range s.Lairs {
		if !l.Defeated {
			return true
		}
	}
	return false
}

// --- the game's side ---------------------------------------------------------

// SetTutorial opens the guide of the world with seed and the player's name:
// fresh for a new game. Only cmd/game calls it.
func (g *Game) SetTutorial(seed uint64, player string, fresh bool) {
	g.tut = loadTutorial(tutorialPath(), worldKey(seed, player), fresh)
}

// tutorialOn reports whether the guide runs: not in a dev game, and not
// when the player turned it off.
func (g *Game) tutorialOn() bool {
	return g.tut != nil && g.dev == nil && !settings.NoTutorial
}

// observeTutorial feeds a snapshot to the guide.
func (g *Game) observeTutorial(s *protocol.Snapshot) {
	if !g.tutorialOn() || g.demo || g.tut.still {
		return
	}
	under := false
	if g.naturalCaves() {
		for _, p := range s.Players {
			if p.ID == g.myID {
				_, _, under = sim.CaveLevelAt(g.height, g.surfaceH, int(p.Y))
			}
		}
	}
	advanced, tip := g.tut.observe(s, g.myID, under, time.Now())
	if advanced {
		snd.play("ui_select", 0.35, 0, 1.35) // soft, a little higher than a click
	}
	if tip != "" {
		g.showNotice(tip)
	}
}

// ShowTutorial puts the tracker at goal n (the last: the end), for the
// screenshot tool; nothing is kept.
func (g *Game) ShowTutorial(n int) {
	t := &tutorial{flash: -1, still: true}
	if n >= 100 { // 100 + k: goal k just done, ticked
		n, t.flash, t.flashAt = n-99, n-100, time.Now().Add(time.Hour)
	}
	t.step = max(0, min(n, len(tutSteps)))
	if t.step == len(tutSteps) {
		t.over, t.finishedAt = true, time.Now()
	}
	g.tut = t
}
