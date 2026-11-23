package sim

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLookSanitized(t *testing.T) {
	ok := Look{Female: true, Hair: HairBun, HairColor: NumHairColors - 1, Skin: NumSkinTones - 1}
	if got := ok.Sanitized(); got != ok || !ok.Valid() {
		t.Errorf("valid look %+v sanitized to %+v", ok, got)
	}
	bad := Look{Female: true, Hair: NumHairStyles, HairColor: NumHairColors, Skin: NumSkinTones}
	if bad.Valid() {
		t.Errorf("%+v reported valid", bad)
	}
	if got, want := bad.Sanitized(), (Look{Female: true}); got != want {
		t.Errorf("sanitized = %+v, want %+v", got, want)
	}
}

func TestNewPlayerTakesTheChosenLook(t *testing.T) {
	w := flatWorld()
	l := Look{Female: true, Hair: HairPonytail, HairColor: 4, Skin: 3}
	if p := mustPlayer(t, w, w.AddPlayerWithLook("An", l)); p.Look != l {
		t.Errorf("look = %+v, want %+v", p.Look, l)
	}
	if p := mustPlayer(t, w, w.AddPlayerWithLook("Bình", Look{Hair: 200, HairColor: 200})); p.Look != (Look{}) {
		t.Errorf("out-of-range look kept as %+v", p.Look)
	}
	if p := mustPlayer(t, w, w.AddPlayer("Chi")); p.Look != (Look{}) {
		t.Errorf("AddPlayer look = %+v, want the default", p.Look)
	}
}

func TestReturningPlayerKeepsTheSavedLook(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	l := Look{Female: true, Hair: HairLong, HairColor: 3, Skin: 5}
	w.AddPlayerWithLook("Tiến", l)
	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	// Continuing a game sends no look; the save's wins.
	if p := mustPlayer(t, w2, w2.AddPlayer("Tiến")); p.Look != l {
		t.Errorf("after load look = %+v, want %+v", p.Look, l)
	}
	// Leaving and rejoining in the same session keeps it too.
	id := w2.AddPlayerWithLook("Lan", Look{Hair: HairSpiky})
	w2.RemovePlayer(id)
	if p := mustPlayer(t, w2, w2.AddPlayerWithLook("Lan", Look{Hair: HairBun})); p.Look.Hair != HairSpiky {
		t.Errorf("rejoined hair = %d, want the first look's %d", p.Look.Hair, HairSpiky)
	}
}

func TestSavesWithoutALookGetTheOriginalSurvivor(t *testing.T) {
	var ps PlayerSave
	if err := json.Unmarshal([]byte(`{"Name":"Cũ","X":3,"Y":4}`), &ps); err != nil {
		t.Fatal(err)
	}
	if ps.Look != (Look{}) {
		t.Errorf("old save look = %+v, want zero", ps.Look)
	}
	// And the default look is not written at all.
	b, err := json.Marshal(PlayerSave{Name: "Mới"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "Look") {
		t.Errorf("default look written: %s", b)
	}
}
