package client

import (
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"survivalgame/internal/sim"
)

// Every recorded voice decodes, has its variants in order, sounds, and is
// played at a speed that makes sense; and every one has a synthesised
// voice to stand in for it.
func TestRecordedVoicesLoad(t *testing.T) {
	t0 := time.Now()
	rec := loadRecordedVoices(voiceFiles)
	t.Logf("%d recorded voices decoded in %v", len(rec.sfx), time.Since(t0).Round(time.Millisecond))
	if len(rec.sfx) < 100 {
		t.Fatalf("%d recorded voices, want them all", len(rec.sfx))
	}
	samples := 0
	for name, vs := range rec.sfx {
		if len(vs) < 2 {
			t.Errorf("%s: %d variants, want several", name, len(vs))
		}
		if s := rec.speed[name]; s != 1 && s != 0.5 {
			t.Errorf("%s: played at speed %v", name, s)
		}
		for i, x := range vs {
			checkFinite(t, name, x)
			peak, rms := stats(x)
			if peak > 1 || peak < 0.1 || rms < 0.005 {
				t.Errorf("%s variant %d: peak %.3f rms %.4f", name, i+1, peak, rms)
			}
			samples += len(x)
		}
		if _, ok := voiceFallback[name]; !ok {
			t.Errorf("%s: no synthesised voice to stand in for it", name)
		}
	}
	t.Logf("%.1f MB of samples", float64(samples*4)/(1<<20))
	for name := range voiceFallback {
		if len(rec.sfx[name]) == 0 {
			t.Errorf("%s: a stand-in for a voice that is not there", name)
		}
	}
}

// A voice that will not load is played with the synthesised voice it
// replaced, at that voice's old speed; one that loads replaces a
// synthesised voice of its name.
func TestRecordedVoiceFallsBack(t *testing.T) {
	fsys := fstest.MapFS{
		"voices/rex_roar_1.qoa":   {Data: []byte("not a QOA file")},
		"voices/bear_growl_1.qoa": {Data: mustRead(t, "voices/bear_growl_1.qoa")},
		"voices/bear_growl_2.qoa": {Data: mustRead(t, "voices/bear_growl_2.qoa")},
	}
	rec := loadRecordedVoices(fsys)
	if len(rec.sfx["rex_roar"]) != 0 || len(rec.sfx["bear_growl"]) != 2 {
		t.Fatalf("loaded %d rex roars and %d bear growls, want 0 and 2", len(rec.sfx["rex_roar"]), len(rec.sfx["bear_growl"]))
	}
	synthRoar := [][]float32{make([]float32, 100)}
	synthGrowl := [][]float32{make([]float32, 50)}
	s := map[string][][]float32{"roar": synthRoar, "bear_growl": synthGrowl, "beast_death": synthRoar}
	t.Cleanup(func() {
		addRecordedVoices(map[string][][]float32{}, loadRecordedVoices(voiceFiles)) // as the game has them
	})
	addRecordedVoices(s, rec)
	if len(s["rex_roar"]) != 1 || &s["rex_roar"][0][0] != &synthRoar[0][0] || speedOf("rex_roar") != 1 {
		t.Error("the tyrant's roar should fall back to the synthesised roar, as it was")
	}
	if len(s["rex_death"]) != 1 || speedOf("rex_death") != 0.75 {
		t.Errorf("the tyrant's death cry should fall back to the beasts' death at 0.75, not %v", speedOf("rex_death"))
	}
	if len(s["bear_growl"]) != 2 || speedOf("bear_growl") != rec.speed["bear_growl"] {
		t.Error("the recorded bear's growl should replace the synthesised one")
	}
}

func mustRead(t *testing.T, name string) []byte {
	b, err := voiceFiles.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Every voice the game plays for a creature is in the bank.
func TestEveryCreatureVoiceIsInTheBank(t *testing.T) {
	b := sharedBank()
	have := func(name string) bool { return len(b[name]) > 0 }
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		kv := voicesOf(k)
		for _, u := range []voiceUse{kv.alarm, kv.hurt, kv.herd, kv.death} {
			if u.name != "" && !have(u.name) {
				t.Errorf("%s: no voice %s", k.Name(), u.name)
			}
		}
		if v, ok := callOf(k); ok && !have(v.name) {
			t.Errorf("%s: no call %s", k.Name(), v.name)
		}
		if v, ok := seaCall[k]; ok && !have(v.name) {
			t.Errorf("%s: no sea call %s", k.Name(), v.name)
		}
	}
	for _, name := range []string{"rex_roar_far", "rex_hurt", "rex_growl", "trike_alarm", "stego_alarm", "brachio_alarm",
		"toro_alarm", "thesc_alarm", "ornitho_alarm", "anzu_roar", "raptor_alarm", "raptor_attack", "raptor_growl",
		"achero_alarm", "achero_attack", "achero_growl", "troodon_alarm", "troodon_attack", "troodon_growl",
		"ptero_alarm", "quetzal_alarm", "spino_roar", "carcharo_roar", "giganoto_roar", "mammothking_roar",
		"mammothking_special", "mammoth_alarm", "sabre_snarl", "sabre_growl", "sabre_cower", "croc_alarm",
		"ankylo_call", "ankylo_alarm", "ankylo_hurt", "ankylo_death"} {
		if !have(name) {
			t.Errorf("no voice %s", name)
		}
	}
	// No two creatures share a voice but the crocodiles, who are one
	// family heard at three sizes.
	who := map[string][]string{}
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		kv := voicesOf(k)
		for _, u := range []voiceUse{kv.hurt, kv.death} {
			if _, ok := voiceFallback[u.name]; ok && !slices.Contains(who[u.name], k.Name()) {
				who[u.name] = append(who[u.name], k.Name())
			}
		}
	}
	for name, ks := range who {
		if len(ks) > 1 && name != "croc_hurt" && name != "croc_death" {
			t.Errorf("%s is the voice of %v", name, ks)
		}
	}
}
