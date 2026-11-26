package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// compareOrder is the species in the order the comparison plays them.
var compareOrder = []string{
	"rex", "spino", "carcharo", "giganoto", "trike", "toro", "stego", "brachio", "thesc", "ornitho",
	"raptor", "achero", "troodon", "ptero", "quetzal", "anzu", "plesio",
	"mammoth", "mammothking", "sabre", "bear", "croc", "brachy", "champso", "mosa", "kraken", "megalodon", "titanoboa",
}

var eventOrder = []string{"roar", "roar_far", "call", "alarm", "herd", "hurt", "growl", "attack", "strike", "grab", "special", "cower", "death"}

// compare writes out.wav: for each species, the first variant of each of
// its sounds in the old dump (VOICE_DUMP of the game before), a blip, then
// the same in the new; and beside it a list of when each begins and how
// loud each is.
func compare(oldDir, newDir, out string) error {
	type snd struct {
		event string
		path  string
	}
	list := func(dir, sp string) []snd {
		fs, _ := filepath.Glob(filepath.Join(dir, sp+"_*_1.wav"))
		var ss []snd
		for _, f := range fs {
			ev := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), sp+"_"), "_1.wav")
			if !slices.Contains(eventOrder, strings.TrimRight(ev, "0123456789")) {
				continue // another species whose name starts like this one's
			}
			ss = append(ss, snd{ev, f})
		}
		slices.SortStableFunc(ss, func(a, b snd) int {
			return slices.Index(eventOrder, strings.TrimRight(a.event, "0123456789")) - slices.Index(eventOrder, strings.TrimRight(b.event, "0123456789"))
		})
		return ss
	}
	blip := make([]float64, secs(0.12))
	for i := range blip {
		t := float64(i) / rate
		blip[i] = 0.12 * math.Sin(2*math.Pi*1320*t) * math.Sin(math.Pi*t/0.12)
	}
	var x []float64
	var idx strings.Builder
	fmt.Fprintf(&idx, "compare_old_new.wav: với mỗi loài, tiếng CŨ (biến thể 1 của từng tiếng), một tiếng bíp, rồi tiếng MỚI.\n")
	fmt.Fprintf(&idx, "Cột: thời điểm | file | độ to (50 ms to nhất, lọc theo tai nghe kiểu BS.1770, dB) | đỉnh (dBFS)\n")
	gap := make([]float64, secs(0.5))
	for _, sp := range compareOrder {
		olds, news := list(oldDir, sp), list(newDir, sp)
		if len(olds) == 0 && len(news) == 0 {
			continue
		}
		fmt.Fprintf(&idx, "\n[%s] %s\n", clockOf(dur(x)), sp)
		for i, set := range [][]snd{olds, news} {
			label := "  cũ:"
			if i == 1 {
				x = append(x, blip...)
				x = append(x, gap...)
				label = "  mới:"
			}
			fmt.Fprintln(&idx, label)
			for _, s := range set {
				y, err := load(s.path)
				if err != nil {
					return err
				}
				fmt.Fprintf(&idx, "    %s  %-26s %6.1f dB %6.1f dB\n", clockOf(dur(x)), filepath.Base(s.path), loudness(y), db(peakOf(y)))
				x = append(x, y...)
				x = append(x, gap...)
			}
			x = append(x, gap...)
		}
		x = append(x, make([]float64, secs(1))...)
	}
	if err := writeWAV(out, x); err != nil {
		return err
	}
	return os.WriteFile(strings.TrimSuffix(out, ".wav")+".txt", []byte(idx.String()), 0o644)
}

func clockOf(s float64) string {
	m := int(s) / 60
	return fmt.Sprintf("%d:%05.2f", m, s-float64(60*m))
}
