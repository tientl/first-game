package client

import (
	"testing"

	"survivalgame/internal/sim"
)

func TestCharCreatorChoices(t *testing.T) {
	c := newCharCreator()
	if c.look != (sim.Look{}) {
		t.Fatalf("starts as %+v, want the original survivor", c.look)
	}
	// Until a style is picked, the body brings its usual hair.
	c.change(ccBody, 1)
	if !c.look.Female || c.look.Hair != sim.DefaultHair(true) {
		t.Errorf("after switching body: %+v", c.look)
	}
	c.change(ccHair, -1)
	want := sim.DefaultHair(true) - 1
	if c.look.Hair != want {
		t.Errorf("hair = %d, want %d", c.look.Hair, want)
	}
	c.change(ccBody, 1)
	if c.look.Female || c.look.Hair != want {
		t.Errorf("a picked style changed with the body: %+v", c.look)
	}
	// Choices wrap both ways.
	c.look.HairColor = 0
	c.change(ccColor, -1)
	if c.look.HairColor != sim.NumHairColors-1 {
		t.Errorf("colour = %d, want %d", c.look.HairColor, sim.NumHairColors-1)
	}
	c.look.Hair = sim.NumHairStyles - 1
	c.change(ccHair, 1)
	if c.look.Hair != 0 {
		t.Errorf("hair = %d, want 0 after wrapping", c.look.Hair)
	}
	if c.change(ccStart, 1) {
		t.Error("a button counted as a choice")
	}
}

func TestCharCreatorButtons(t *testing.T) {
	c := newCharCreator()
	for range 20 {
		if c.activate(ccRandom) != createNone || !c.look.Valid() {
			t.Fatalf("random look %+v", c.look)
		}
	}
	if c.activate(ccStart) != createDone || c.activate(ccBack) != createCancel {
		t.Error("start and back do not finish the screen")
	}
	if c.activate(ccHair) != createNone {
		t.Error("a choice finished the screen")
	}
}

func TestTypingAName(t *testing.T) {
	var name []rune
	for _, r := range "  Lan\t  Anh\x07" {
		name = typeRune(name, r)
	}
	if got := string(name); got != "Lan Anh" {
		t.Errorf("typed %q, want %q", got, "Lan Anh")
	}
	var long []rune
	for range 30 {
		long = typeRune(long, 'ồ')
	}
	if len(long) != sim.MaxNameRunes {
		t.Errorf("name of %d letters, want at most %d", len(long), sim.MaxNameRunes)
	}
	// Unikey turns "Ha" into "Hà" with a Backspace and the accented letter.
	name = eraseLast([]rune("Ha"))
	name = typeRune(name, 'à')
	if got := string(name); got != "Hà" {
		t.Errorf("after rewrite %q, want %q", got, "Hà")
	}
	if got := eraseLast(nil); len(got) != 0 {
		t.Errorf("erasing nothing gave %q", string(got))
	}
	c := newCharCreator()
	if c.focus != ccName {
		t.Errorf("opens on row %d, want the name", c.focus)
	}
	c.setName(" Tiến ")
	if got := c.nameText(); got != "Tiến" {
		t.Errorf("name = %q, want %q", got, "Tiến")
	}
}

func TestCharCreatorSkin(t *testing.T) {
	c := newCharCreator()
	if c.look.Skin != 0 {
		t.Fatalf("starts with skin %d, want the original", c.look.Skin)
	}
	c.change(ccSkin, -1)
	if c.look.Skin != sim.NumSkinTones-1 {
		t.Errorf("skin = %d, want %d after wrapping", c.look.Skin, sim.NumSkinTones-1)
	}
	c.change(ccSkin, 1)
	c.change(ccSkin, 1)
	if c.look.Skin != 1 {
		t.Errorf("skin = %d, want 1", c.look.Skin)
	}
	// The skin row does not move the hair colour, nor the other way round.
	if c.look.HairColor != 0 {
		t.Errorf("hair colour moved to %d", c.look.HairColor)
	}
	c.pick(ccColor, 5)
	if c.look.Skin != 1 || c.look.HairColor != 5 {
		t.Errorf("after picking a hair colour: %+v", c.look)
	}
}
