package client

import (
	"slices"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestFKeyCooksWhatThereIs(t *testing.T) {
	g := readyGame()
	if got := g.cookRecipe(); got != sim.RecipeCookMeat || g.hasRawFood() {
		t.Fatalf("empty bag: cooks %d (raw food %v), want the meat recipe so the server says what is missing", got, g.hasRawFood())
	}
	g.curr.Self.Inv.Add(sim.ItemFish, 1)
	if got := g.cookRecipe(); got != sim.RecipeCookFish || !g.hasRawFood() {
		t.Fatalf("cooks %d, want the fish", got)
	}
	g.curr.Self.Inv.Add(sim.ItemMeat, 1)
	if got := g.cookRecipe(); got != sim.RecipeCookMeat {
		t.Fatalf("cooks %d, want the meat first", got)
	}
}

func TestByFire(t *testing.T) {
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5}
	fire := protocol.EntityView{ID: 7, Kind: sim.EntityCampfire, TX: 12, TY: 10, Ready: true, Fuel: sim.FireStartFuel}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{fire}})
	if !g.byFire() {
		t.Fatal("two tiles from a fresh fire should count as by the fire")
	}
	me.X = 20
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{fire}})
	if g.byFire() {
		t.Fatal("ten tiles away is not by the fire")
	}
}

func TestEatsCookedFoodBeforeRaw(t *testing.T) {
	g := readyGame()
	inv := &g.curr.Self.Inv
	inv[2] = sim.Slot{Item: sim.ItemMeat, Count: 3}
	inv[4] = sim.Slot{Item: sim.ItemBerry, Count: 1}
	inv[sim.HotbarSlots+3] = sim.Slot{Item: sim.ItemCookedFish, Count: 1}
	if got, ok := g.slotToEat(); !ok || got != sim.HotbarSlots+3 {
		t.Fatalf("eats slot %d, want the cooked fish", got)
	}
	inv[sim.HotbarSlots+3] = sim.Slot{}
	if got, _ := g.slotToEat(); got != 4 {
		t.Fatalf("eats slot %d, want berries before raw meat", got)
	}
}

func TestCookingSizzles(t *testing.T) {
	played := testSound(t)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}})
	s := protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}}
	s.Self.Notice, s.Self.NoticeArg, s.Self.NoticeSeq = sim.NoticeCrafted, uint8(sim.RecipeCookFish), 1
	g.onSnapshot(s)
	if !slices.Contains(*played, "sizzle") || slices.Contains(*played, "craft") {
		t.Fatalf("played %v, want a sizzle instead of the craft chime", *played)
	}
	if noticeText(sim.NoticeCrafted, uint8(sim.RecipeCookFish)) != "Đã nướng xong 1 Cá nướng" {
		t.Errorf("notice %q", noticeText(sim.NoticeCrafted, uint8(sim.RecipeCookFish)))
	}
}
