package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"survivalgame/internal/sim"
)

// A tool more than a test: it writes every voice of every dinosaur and great
// beast to WAV files, to be listened to outside the game.
//
//	VOICE_DUMP=dir go test ./internal/client -run DumpCreatureVoices -v
//
// Each sound is played through the game's own mixer (audio.go), so it is
// what a player standing right by the beast hears: the bank's sound
// resampled at the voice's rate (pitch and speed together; a recorded voice
// kept at half rate is played at half speed, see realvoice.go), at its volume,
// times the effects bus (the effects slider squared; VOICE_EFFECTS, 0 to 1,
// default 0.8 as a new game has it), times the mixer's 0.9 and its soft clip.
// The few per-play random pitch shifts (±3–4 %) are left out. The bank's
// sounds are mono and a beast right by the listener is panned to the
// centre, so the left and right channels are the same: the files are mono.
//
// It writes, into dir:
//   - <species>_<event>_<variant>.wav for every sound of every species, one
//     per variant of the bank's sound (an event heard with the same sound at
//     the same rate as another, only at another volume, shares its file, at
//     the louder volume; index.txt lists every use);
//   - all_dinosaurs.wav and all_beasts.wav: each species' sounds back to
//     back, 0.6 s apart, 1.5 s between species;
//   - index.txt: each species and sound, its start in the long file, its
//     file, its peak level and when the game plays it; then which voices of
//     the bank the species share, and which sounds of different species
//     are the same within the game's random pitch shift.
//
// Only voices: bites, blows, footfalls and splashes are left out, but for
// the Megalodon's opening splash, which is all it gives for a roar.

// dumpPart is one sound of the bank played at a volume and rate.
type dumpPart struct {
	name      string
	vol, rate float64
}

// dumpUse is one thing a creature says: one or more sounds played together.
type dumpUse struct {
	event string // the file name's event: roar, call, alarm, hurt, ...
	label string // what it is, for the index
	parts []dumpPart
}

func dp(name string, vol, rate float64) []dumpPart { return []dumpPart{{name, vol, rate}} }

// dumpSpecies is a creature and the sounds the state-change code plays for
// it (dinosound.go, hellwire.go, iceart.go, bosses.go, predsound.go,
// troodonsound.go, creatures.go). Its calls (callOf, moreCalls, seaCall) and
// its cries (voicesOf) are added from the game's own tables.
type dumpSpecies struct {
	key    string
	kind   sim.CreatureKind
	script []dumpUse
}

// roarFar is the volume a tyrant's roar from 16 tiles off is heard at:
// roarAt's 1.4 times callAt's falling off over roarReach.
var roarFar = 1.4 * math.Pow(1-16/roarReach, 2)

var dumpDinos = []dumpSpecies{
	{"rex", sim.CreatureRex, []dumpUse{
		{"roar", "gầm khi thấy người chơi", dp("rex_roar", 1, 1)},
		{"roar", "gầm khi lao theo con mồi", dp("rex_roar", 0.85, 1)},
		{"roar", "gầm bên con mồi vừa hạ", dp("rex_roar", 0.75, 1)},
		{"roar_far", "tiếng gầm nghe từ xa (16 ô, bị nghẹt)", dp("rex_roar_far", roarFar, 1)},
		{"hurt", "gầm đứt quãng khi bị đánh", dp("rex_hurt", 0.75, 1)},
		{"growl", "gầm gừ khi đuổi sát", dp("rex_growl", 0.9, 1)},
	}},
	{"trike", sim.CreatureTrike, []dumpUse{{"alarm", "rống khi lao tới / bỏ chạy", dp("trike_alarm", 0.8, 1)}}},
	{"toro", sim.CreatureToro, []dumpUse{{"alarm", "rống khi lao tới", dp("toro_alarm", 0.9, 1)}}},
	{"stego", sim.CreatureStego, []dumpUse{{"alarm", "rống khi lao tới / bỏ chạy", dp("stego_alarm", 0.8, 1)}}},
	{"ankylo", sim.CreatureAnkylo, []dumpUse{
		{"alarm", "gầm gừ khi quay chùy đuôi về phía người lại gần", dp("ankylo_grunt", 0.75, 1)},
		{"strike", "gầm khi khuỵu xuống lấy đà quật chùy", dp("ankylo_grunt", 0.6, 1.18)},
	}},
	{"brachio", sim.CreatureBrachio, []dumpUse{{"alarm", "rống khi lao tới / bỏ chạy", dp("brachio_alarm", 0.8, 1)}}},
	{"thesc", sim.CreatureThesc, []dumpUse{{"alarm", "kêu khi bỏ chạy", dp("thesc_alarm", 0.5, 1)}}},
	{"ornitho", sim.CreatureOrnitho, []dumpUse{{"alarm", "kêu khi bỏ chạy", dp("ornitho_alarm", 0.4, 1)}}},
	{"raptor", sim.CreatureRaptor, []dumpUse{
		{"alarm", "rít khi lao vào săn", dp("raptor_alarm", 0.75, 1)},
		{"call", "gọi đàn bên con mồi vừa hạ", dp("raptor_call", 0.5, 1)},
		{"growl", "lục khục, lách cách khi đuổi sát", dp("raptor_growl", 0.55, 1)},
		{"attack", "gầm rít khi vồ", dp("raptor_attack", 0.5, 1)},
	}},
	{"achero", sim.CreatureAchero, []dumpUse{
		{"alarm", "rít khi lao vào săn", dp("achero_alarm", 0.6, 1)},
		{"call", "gọi đàn bên con mồi vừa hạ", dp("achero_call", 0.45, 1)},
		{"growl", "lục khục, lách cách khi đuổi sát", dp("achero_growl", 0.45, 1)},
		{"attack", "gầm rít khi vồ", dp("achero_attack", 0.4, 1)},
	}},
	{"troodon", sim.CreatureTroodon, []dumpUse{
		{"alarm", "rít khi phát hiện người", dp("troodon_alarm", 0.7, 1)},
		{"growl", "lách cách trong họng khi bầy áp sát", dp("troodon_growl", 0.5, 1)},
		{"attack", "rít vút lên khi cắn", dp("troodon_attack", 0.5, 1)},
	}},
	{"ptero", sim.CreaturePtero, []dumpUse{{"alarm", "rít khi lao xuống", dp("ptero_alarm", 0.7, 1)}}},
	{"quetzal", sim.CreatureQuetzal, []dumpUse{
		{"alarm", "rít khi lao vào", dp("quetzal_alarm", 0.7, 1)},
		{"alarm", "rít khi cất cánh bay lên", dp("quetzal_alarm", 0.9, 1)},
	}},
	{"anzu", sim.CreatureAnzu, []dumpUse{{"roar", "rít khi xòe mào nổi giận", dp("anzu_roar", 0.55, 1)}}},
	{"plesio", sim.CreaturePlesio, nil},
	{"spino", sim.CreatureSpino, []dumpUse{{"roar", "gầm mở màn trận / nổi điên (gầm lẫn tiếng khè)", dp("spino_roar", 0.95, 1)}}},
	{"carcharo", sim.CreatureCarcharo, []dumpUse{{"roar", "gầm mở màn trận / nổi điên", dp("carcharo_roar", 1, 1)}}},
	{"giganoto", sim.CreatureGiganoto, []dumpUse{{"roar", "gầm mở màn trận / nổi điên", dp("giganoto_roar", 1, 1)}}},
}

var dumpBeasts = []dumpSpecies{
	{"mammoth", sim.CreatureMammoth, []dumpUse{{"alarm", "rống vòi khi lao tới", dp("mammoth_alarm", 0.8, 1)}}},
	{"sabre", sim.CreatureSabre, []dumpUse{
		{"alarm", "gầm khi chọn con mồi", dp("sabre_roar", 0.85, 1)},
		{"strike", "gầm gừ khi vồ", dp("sabre_snarl", 0.9, 1)},
		{"growl", "gầm gừ khi rình đuổi", dp("sabre_growl", 0.7, 1)},
		{"cower", "khè khi sợ", dp("sabre_cower", 0.4, 1)},
	}},
	{"bear", sim.CreatureBear, []dumpUse{{"growl", "gầm gừ khi vả", dp("bear_growl", 0.55, 1.1)}}},
	{"croc", sim.CreatureCroc, []dumpUse{{"alarm", "khè khi lao lên khỏi nước", dp("croc_alarm", 0.75, 1)}}},
	{"brachy", sim.CreatureBrachy, []dumpUse{{"alarm", "khè khi lao lên khỏi ao", dp("croc_alarm", 0.6, 1.15)}}},
	{"champso", sim.CreatureChampso, []dumpUse{{"alarm", "khè khi lao lên khỏi ao", dp("croc_alarm", 0.6, 1.35)}}},
	{"mosa", sim.CreatureMosa, nil},
	{"kraken", sim.CreatureKraken, nil},
	{"titanoboa", sim.CreatureTitanoboa, []dumpUse{
		{"roar", "khè mở màn trận / nổi điên (hai tiếng khè cùng lúc)", []dumpPart{{"hiss", 1, 0.5}, {"croc_hiss", 0.8, 0.7}}},
		{"grab", "khè khi siết mồi", dp("hiss", 0.8, 1)},
	}},
	{"megalodon", sim.CreatureMegalodon, []dumpUse{{"roar", "quẫy nước mở màn trận (không có tiếng kêu)", dp("splash", 1, 0.6)}}},
	{"mammothking", sim.CreatureMammothKing, []dumpUse{
		{"roar", "rống vòi mở màn trận / nổi điên", dp("mammothking_roar", 1, 1)},
		{"special", "rống vòi khi chồm lên ra đòn", dp("mammothking_special", 0.9, 1)},
	}},
}

// uses is everything sp says: its scripted sounds, its call on its own and
// its cries from voicesOf.
func (sp dumpSpecies) uses() []dumpUse {
	us := slices.Clone(sp.script)
	if sp.kind == sim.CreatureNone {
		return us // not a creature of the game yet
	}
	if v, ok := callOf(sp.kind); ok {
		us = append(us, dumpUse{"call", "kêu tự do khi đi lang thang", dp(v.name, v.vol, v.rate)})
	}
	if v, ok := seaCall[sp.kind]; ok {
		us = append(us, dumpUse{"call", "kêu tự do dưới biển", dp(v.name, v.vol, v.rate)})
	}
	kv := voicesOf(sp.kind)
	for _, u := range []struct {
		event, label string
		v            voiceUse
	}{
		{"alarm", "báo động (trồi lên / nổi giận / bỏ chạy)", kv.alarm},
		{"hurt", "kêu khi bị đánh", kv.hurt},
		{"herd", "con cùng đàn đáp lại khi một con bị đánh", kv.herd},
		{"death", "kêu khi chết", kv.death},
	} {
		if u.v.name != "" {
			us = append(us, dumpUse{u.event, u.label, dp(u.v.name, u.v.vol, u.v.rate)})
		}
	}
	return us
}

// dumpOrder is the order of a species' events, the first of a group naming
// its file.
var dumpOrder = []string{"roar", "roar_far", "call", "alarm", "herd", "hurt", "growl", "attack", "strike", "grab", "special", "cower", "death"}

// dumpGroup is the uses of one species that are the same sound at the same
// rate: one set of files.
type dumpGroup struct {
	event string
	parts []dumpPart // at the loudest volume of the uses
	uses  []dumpUse
}

// sameSound is the key of the sound in ps: its names and rates.
func sameSound(ps []dumpPart) string {
	var b strings.Builder
	for _, p := range ps {
		fmt.Fprintf(&b, "%s×%.3f+", p.name, p.rate)
	}
	return b.String()
}

func (sp dumpSpecies) groups() []dumpGroup {
	us := sp.uses()
	slices.SortStableFunc(us, func(a, b dumpUse) int {
		return slices.Index(dumpOrder, a.event) - slices.Index(dumpOrder, b.event)
	})
	var gs []dumpGroup
	at := map[string]int{}
	for _, u := range us {
		k := sameSound(u.parts)
		i, ok := at[k]
		if !ok {
			i = len(gs)
			at[k] = i
			gs = append(gs, dumpGroup{event: u.event, parts: slices.Clone(u.parts)})
		}
		g := &gs[i]
		g.uses = append(g.uses, u)
		for j := range g.parts {
			g.parts[j].vol = max(g.parts[j].vol, u.parts[j].vol)
		}
	}
	// Two groups of one event: alarm, alarm2, ...
	seen := map[string]int{}
	for i := range gs {
		e := gs[i].event
		if seen[e]++; seen[e] > 1 {
			gs[i].event = e + strconv.Itoa(seen[e])
		}
	}
	return gs
}

// variants is how many variants the bank has of the sound of ps.
func variants(bank map[string][][]float32, ps []dumpPart) int {
	n := 0
	for _, p := range ps {
		n = max(n, len(bank[p.name]))
	}
	return n
}

// renderVoice plays variant i of ps through the game's mixer, centred, with
// the effects bus at gain bus, and returns the left channel.
func renderVoice(bank map[string][][]float32, ps []dumpPart, i int, bus float32) []float32 {
	m := &mixer{}
	m.busGain[busSFX], m.busTarget[busSFX] = bus, bus
	frames := 0
	for _, p := range ps {
		vs := bank[p.name]
		data := vs[i%len(vs)]
		rate := p.rate * speedOf(p.name) // as play does: see realvoice.go
		m.add(&voice{data: data, rate: rate, gain: float32(p.vol), target: float32(p.vol), bus: busSFX})
		frames = max(frames, int(math.Ceil(float64(len(data)-1)/rate))+1)
	}
	buf := make([]byte, frames*8)
	m.Read(buf)
	out := make([]float32, frames)
	for j := range out {
		out[j] = math.Float32frombits(binary.LittleEndian.Uint32(buf[8*j:]))
	}
	for len(out) > 0 && out[len(out)-1] == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// writeWAV writes x as 16-bit mono PCM at the game's sample rate.
func writeWAV(path string, x []float32) error {
	var b bytes.Buffer
	n := uint32(len(x))
	le := binary.LittleEndian
	b.WriteString("RIFF")
	b.Write(le.AppendUint32(nil, 36+2*n))
	b.WriteString("WAVEfmt ")
	b.Write(le.AppendUint32(nil, 16))           // the format chunk's size
	b.Write(le.AppendUint16(nil, 1))            // PCM
	b.Write(le.AppendUint16(nil, 1))            // mono
	b.Write(le.AppendUint32(nil, sampleRate))   // frames a second
	b.Write(le.AppendUint32(nil, sampleRate*2)) // bytes a second
	b.Write(le.AppendUint16(nil, 2))            // bytes a frame
	b.Write(le.AppendUint16(nil, 16))           // bits a sample
	b.WriteString("data")
	b.Write(le.AppendUint32(nil, 2*n))
	for _, v := range x {
		b.Write(le.AppendUint16(nil, uint16(int16(math.Round(float64(max(-1, min(1, v)))*32767)))))
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

func dB(v float64) float64 {
	if v <= 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(v)
}

func clock(sec float64) string {
	m := int(sec) / 60
	return fmt.Sprintf("%d:%05.2f", m, sec-float64(60*m))
}

// soundStats is a rendered sound's peak and RMS.
func soundStats(x []float32) (peak, rms float64) {
	for _, v := range x {
		a := math.Abs(float64(v))
		peak = max(peak, a)
		rms += a * a
	}
	if len(x) > 0 {
		rms = math.Sqrt(rms / float64(len(x)))
	}
	return
}

// softClipped is the share of variant i of ps that goes through the
// mixer's soft clip (audio.go: above 0.6 after its 0.9). Played a hundred
// times quieter nothing clips, and the mix is what the clip is given, over
// a hundred.
func softClipped(bank map[string][][]float32, ps []dumpPart, i int, bus float32) float64 {
	x := renderVoice(bank, ps, i, bus/100)
	n := 0
	for _, v := range x {
		if math.Abs(float64(v))*100 > 0.6 {
			n++
		}
	}
	return float64(n) / float64(max(1, len(x)))
}

// dumpedSound is one file written.
type dumpedSound struct {
	species, file string
	group         dumpGroup
	variant, of   int
	x             []float32
	clip          float64 // the share of it through the mixer's soft clip
}

// partsText is the sounds of ps with their rates, and volumes if vol.
func partsText(ps []dumpPart, vol bool) string {
	var s []string
	for _, p := range ps {
		if vol {
			s = append(s, fmt.Sprintf("%s ×%.2f vol %.2f", p.name, p.rate, p.vol))
		} else {
			s = append(s, fmt.Sprintf("%s ×%.2f", p.name, p.rate))
		}
	}
	return strings.Join(s, " + ")
}

// alike reports whether a and c are the same sounds of the bank at rates
// less than 4 % apart: within the game's own random pitch shift, so no one
// can tell them apart in play.
func alike(a, c []dumpPart) bool {
	if len(a) != len(c) {
		return false
	}
	for j, p := range a {
		if q := c[j]; p.name != q.name || math.Abs(p.rate/q.rate-1) >= 0.04 {
			return false
		}
	}
	return true
}

func TestDumpCreatureVoices(t *testing.T) {
	dir := os.Getenv("VOICE_DUMP")
	if dir == "" {
		t.Skip("set VOICE_DUMP=dir to write the creatures' voices there")
	}
	effects := 0.8
	if s := os.Getenv("VOICE_EFFECTS"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v <= 0 || v > 1 {
			t.Fatalf("VOICE_EFFECTS=%q: want a number in (0, 1]", s)
		}
		effects = v
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	b := sharedBank()
	bus := float32(effects * effects)

	var index strings.Builder
	fmt.Fprintf(&index, "Tiếng kêu của khủng long và thú lớn trong Dino Isles\n")
	fmt.Fprintf(&index, "Mỗi tiếng được phát qua bộ trộn của game như khi đứng sát con vật: âm lượng hiệu ứng %.1f, ", effects)
	fmt.Fprintf(&index, "đã đổi tốc độ (×rate: đổi cả cao độ lẫn độ dài), không có độ lệch cao độ ngẫu nhiên (±3–4%%). Mono, 16-bit, 44,1 kHz.\n")
	fmt.Fprintf(&index, "Cột: thời điểm trong file dài | file | đỉnh (dBFS) | tiếng gốc ×rate, âm lượng — dùng khi nào (âm lượng trong game).\n")
	fmt.Fprintf(&index, "Một tiếng dùng cho nhiều việc (cùng tiếng gốc, cùng rate, chỉ khác âm lượng) chỉ có một bộ file, ở âm lượng lớn nhất.\n")

	var all []dumpedSound
	for _, set := range []struct {
		file, title string
		species     []dumpSpecies
	}{
		{"all_dinosaurs.wav", "KHỦNG LONG VÀ THẰN LẰN BAY", dumpDinos},
		{"all_beasts.wav", "THÚ LỚN KHÁC VÀ TRÙM", dumpBeasts},
	} {
		const lead, between, gapSpecies = 0.5, 0.6, 1.5
		long := make([]float32, int(lead*sampleRate))
		fmt.Fprintf(&index, "\n=== %s — %s ===\n", set.file, set.title)
		for si, sp := range set.species {
			if si > 0 {
				long = append(long, make([]float32, int((gapSpecies-between)*sampleRate))...)
			}
			fmt.Fprintf(&index, "\n[%s] %s (%s)\n", clock(float64(len(long))/sampleRate), sp.title(), sp.key)
			for _, g := range sp.groups() {
				n := variants(b, g.parts)
				if n == 0 {
					t.Errorf("%s %s: no sound %s", sp.key, g.event, partsText(g.parts, true))
					continue
				}
				var uses []string
				for _, u := range g.uses {
					uses = append(uses, fmt.Sprintf("%s (%.2f)", u.label, u.parts[0].vol))
				}
				for i := range n {
					x := renderVoice(b, g.parts, i, bus)
					file := fmt.Sprintf("%s_%s_%d.wav", sp.key, g.event, i+1)
					if err := writeWAV(filepath.Join(dir, file), x); err != nil {
						t.Fatal(err)
					}
					peak, _ := soundStats(x)
					what := fmt.Sprintf("biến thể %d/%d", i+1, n)
					if i == 0 {
						what = partsText(g.parts, true) + " — " + strings.Join(uses, "; ")
						if n > 1 {
							what = fmt.Sprintf("[%d biến thể] %s", n, what)
						}
					}
					fmt.Fprintf(&index, "  %s  %-26s %6.1f dB  %s\n", clock(float64(len(long))/sampleRate), file, dB(peak), what)
					all = append(all, dumpedSound{sp.key, file, g, i + 1, n, x, softClipped(b, g.parts, i, bus)})
					long = append(long, x...)
					long = append(long, make([]float32, int(between*sampleRate))...)
				}
			}
		}
		if err := writeWAV(filepath.Join(dir, set.file), long); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&index, "\n(%s dài %s)\n", set.file, clock(float64(len(long))/sampleRate))
		t.Logf("%s: %s", set.file, clock(float64(len(long))/sampleRate))
	}
	var firsts []dumpedSound
	for _, s := range all {
		if s.variant == 1 {
			firsts = append(firsts, s)
		}
	}

	// Which species share a voice of the bank, and at what rates.
	fmt.Fprintf(&index, "\n=== Giọng gốc dùng chung (tiếng trong bank ← loài_sự kiện ×rate) ===\n")
	shared := map[string][]string{}
	for _, s := range firsts {
		for _, p := range s.group.parts {
			shared[p.name] = append(shared[p.name], fmt.Sprintf("%s_%s ×%.2f", s.species, s.group.event, p.rate))
		}
	}
	names := make([]string, 0, len(shared))
	for k := range shared {
		names = append(names, k)
	}
	slices.SortFunc(names, func(a, b string) int {
		if d := len(shared[b]) - len(shared[a]); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	for _, k := range names {
		fmt.Fprintf(&index, "%-15s %d biến thể ← %s\n", k, len(b[k]), strings.Join(shared[k], ", "))
	}

	// Sounds of different species that no one can tell apart in play, in
	// clusters of sounds alike one to the next.
	fmt.Fprintf(&index, "\n=== Khó phân biệt giữa các loài: cùng tiếng gốc, rate lệch dưới 4%% (game tự lệch ±3–4%% mỗi lần phát) ===\n")
	root := make([]int, len(firsts))
	for i := range root {
		root[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if root[i] != i {
			root[i] = find(root[i])
		}
		return root[i]
	}
	for i, a := range firsts {
		for j := i + 1; j < len(firsts); j++ {
			if alike(a.group.parts, firsts[j].group.parts) {
				root[find(j)] = find(i)
			}
		}
	}
	for i := range firsts {
		if find(i) != i {
			continue
		}
		var members []string
		species := map[string]bool{}
		for j, s := range firsts {
			if find(j) == i {
				members = append(members, fmt.Sprintf("%s_%s (%s)", s.species, s.group.event, partsText(s.group.parts, false)))
				species[s.species] = true
			}
		}
		if len(species) > 1 {
			fmt.Fprintf(&index, "- %s\n", strings.Join(members, " ≈ "))
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "index.txt"), []byte(index.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	// Every file is a sound: not silent, not cut off.
	for _, s := range all {
		peak, rms := soundStats(s.x)
		note := ""
		switch {
		case peak < 0.02 || rms < 0.002:
			note = "  << QUÁ NHỎ / IM LẶNG"
			t.Errorf("%s: peak %.4f rms %.4f, silent", s.file, peak, rms)
		case peak >= 0.999:
			note = "  << CHẠM TRẦN"
			t.Errorf("%s: peak %.4f, clipped", s.file, peak)
		case s.clip > 0:
			note = fmt.Sprintf("  (soft clip của mixer: %.2f%% mẫu)", 100*s.clip)
		}
		t.Logf("%-28s %5.2f s  peak %6.1f dBFS  rms %6.1f dBFS%s", s.file, float64(len(s.x))/sampleRate, dB(peak), dB(rms), note)
	}
	t.Logf("%d files in %s", len(all)+3, dir)
}

// The dump's table names only sounds the bank has, so it keeps up with the
// game: a voice renamed in the game fails here, not in the dump.
func TestVoiceDumpTableIsValid(t *testing.T) {
	b := sharedBank()
	for _, sp := range append(slices.Clone(dumpDinos), dumpBeasts...) {
		gs := sp.groups()
		if len(gs) == 0 {
			t.Errorf("%s: no sounds", sp.key)
		}
		for _, g := range gs {
			for _, p := range g.parts {
				if len(b[p.name]) == 0 || !(p.rate > 0.01 && p.rate < 100) || p.vol <= 0 {
					t.Errorf("%s %s: %s is not a sound the game can play", sp.key, g.event, partsText([]dumpPart{p}, true))
				}
			}
		}
	}
}

// title is the species' name in the index.
func (sp dumpSpecies) title() string {
	if sp.kind == sim.CreatureNone {
		return "Ankylosaurus (chưa có trong game ở nhánh này)"
	}
	return sp.kind.Name()
}
