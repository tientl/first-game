package sim

import (
	"encoding/json"
	"fmt"
	"testing"
)

// There are no save files of the old versions to keep (the game has only
// ever had the one save format, JSON of SaveData, and no fixtures were
// kept), so this makes each version's save from a world of the day's style by
// taking away what that version did not have: the fields and the kinds of
// things added later, and the inventory as a count per item before version
// 4. It is the best stand-in for the real files, not a copy of them: what a
// version kept that no longer exists in the code cannot be rebuilt here.

// versionStyle is the newest world style that existed at save version v.
func versionStyle(v int) WorldStyle {
	switch {
	case v >= 28:
		return StyleCaveWays
	case v >= 27:
		return StyleFlint
	case v >= 26:
		return StyleCaves
	case v >= 25:
		return StyleGround
	case v >= 23:
		return StyleWilds
	case v >= 21:
		return StyleBeaches
	case v >= 20:
		return StyleWoods
	case v >= 19:
		return StyleIsles
	}
	return StyleIsland
}

// kindsSince is the save version that added each kind of entity that was
// not there from version 1.
var kindsSince = map[EntityKind]int{
	EntityTwig: 2, EntityPebble: 5, EntityFarmland: 5, EntityCrop: 5,
	EntityFurnace: 6, EntityCopperOre: 6, EntityIronOre: 6,
	EntityFigTree: 9, EntityGinkgoTree: 9, EntityPalm: 9,
	EntityCorpse: 10, EntityTrophy: 10, EntityGrave: 10,
	EntityTreasure: 12, EntityLairProp: 15,
	EntityClay: 16, EntityMortar: 16, EntityOven: 16, EntityDryRack: 16,
	EntityFlora: 17, EntityLeafTent: 24,
	EntityCaveDown: 26, EntityCaveUp: 26, EntityStalagmite: 26, EntityCaveProp: 26,
}

// asVersion rewrites a save of the latest version the way version v would
// have kept it.
func asVersion(d SaveData, v int) SaveData {
	d.Version = v
	if v < 8 {
		d.Pets = nil
	}
	if v < 10 {
		d.Drops = nil
	}
	if v < 12 {
		d.Bosses = nil
	}
	if v < 16 {
		d.Stations = nil
	}
	if v < 19 {
		d.Difficulty = 0
	}
	kept := make([]Entity, 0, len(d.Entities))
	for _, e := range d.Entities {
		if since, ok := kindsSince[e.Kind]; ok && v < since {
			continue
		}
		if v < 3 {
			e.Big = false
		}
		if v < 31 {
			e.Face = DirDown // nothing turned: see facing.go
		}
		kept = append(kept, e)
	}
	d.Entities = kept
	if v < 2 {
		d.Entities = d.Entities[:len(d.Entities):len(d.Entities)]
	}
	players := make([]PlayerSave, len(d.Players))
	copy(players, d.Players)
	for i := range players {
		p := &players[i]
		if v < 4 { // the inventory was a count per item kind
			inv := make([]int, NumItems)
			for _, s := range p.Slots {
				if !s.Empty() {
					inv[s.Item] += s.Count
				}
			}
			p.Inv, p.Slots = inv, Inventory{}
		}
		if v < 18 {
			p.Look = Look{}
		}
		if v < 16 {
			p.Buffs = nil
		}
	}
	d.Players = players
	return d
}

func TestEverySaveVersionLoadsTicksAndSavesAgain(t *testing.T) {
	for v := 1; v <= SaveVersion; v++ {
		styles := []WorldStyle{versionStyle(v)}
		if v >= 7 && v <= 18 {
			styles = append(styles, StyleSeas) // the archipelago came in these years, too
		}
		for _, style := range styles {
			t.Run(fmt.Sprintf("v%d/style%d", v, style), func(t *testing.T) {
				size := 64
				if style >= StyleIsles {
					size = 256
				}
				w, err := Generate(GenConfig{Seed: 11, Width: size, Height: size, Style: style})
				if err != nil {
					t.Fatal(err)
				}
				id := w.AddPlayer("Cũ")
				p := w.players[id]
				p.Inv.Add(ItemWood, 7)
				p.Inv.Add(ItemBerry, 3)
				p.Health, p.Hunger = 61, 44
				stepN(w, 20)

				d := asVersion(w.Save(), v)
				// Through the file format and back, as a real save is read.
				b, err := json.Marshal(d)
				if err != nil {
					t.Fatal(err)
				}
				var read SaveData
				if err := json.Unmarshal(b, &read); err != nil {
					t.Fatal(err)
				}
				old, err := Load(read)
				if err != nil {
					t.Fatalf("version %d does not load: %v", v, err)
				}
				if old.Tick() < StartTick {
					t.Errorf("tick = %d, before the start %d", old.Tick(), StartTick)
				}
				id2 := old.AddPlayer("Cũ")
				p2 := mustPlayer(t, old, id2)
				if p2.Inv.Count(ItemWood) != 7 || p2.Inv.Count(ItemBerry) != 3 || p2.Health != 61 {
					t.Errorf("the player came back as %d wood, %d berries, health %v", p2.Inv.Count(ItemWood), p2.Inv.Count(ItemBerry), p2.Health)
				}
				stepN(old, 200) // a few seconds of the world, creatures and all

				// Saved now, it is a current save and loads as such.
				now := old.Save()
				if now.Version != SaveVersion {
					t.Fatalf("saved as version %d, want %d", now.Version, SaveVersion)
				}
				b, err = json.Marshal(now)
				if err != nil {
					t.Fatal(err)
				}
				var again SaveData
				if err := json.Unmarshal(b, &again); err != nil {
					t.Fatal(err)
				}
				w3, err := Load(again)
				if err != nil {
					t.Fatalf("the upgraded save does not load: %v", err)
				}
				if len(w3.entities) != len(old.entities) {
					t.Errorf("entities = %d after the second load, want %d", len(w3.entities), len(old.entities))
				}
				stepN(w3, 20)
			})
		}
	}
}
