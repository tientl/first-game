package server

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"survivalgame/internal/sim"
)

// TestLoadTimes prints how long a world takes to make, save, read and load.
// It is slow and big, so it runs only when LOADTIMES names a size:
//
//	LOADTIMES=2048 go test -count=1 -run TestLoadTimes -v ./internal/server
func TestLoadTimes(t *testing.T) {
	size, _ := strconv.Atoi(os.Getenv("LOADTIMES"))
	if size == 0 {
		t.Skip("set LOADTIMES=1024 or 2048")
	}
	mem := func() float64 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.HeapAlloc) / (1 << 20)
	}
	t0 := time.Now()
	srv, err := New(Config{Seed: 20240601, Width: size, Height: size, Style: sim.StyleLatest})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("New (generate) %d: %.1f s, live heap %.0f MB", size, time.Since(t0).Seconds(), mem())
	path := filepath.Join(t.TempDir(), "save.json")
	srv.world.AddPlayer("Tester")
	t0 = time.Now()
	if err := srv.save(); err == nil {
		t.Log("save() without a path unexpectedly worked")
	}
	f := SaveFile{SavedAt: time.Now(), World: srv.world.Save()}
	t.Logf("world.Save: %.2f s", time.Since(t0).Seconds())
	t0 = time.Now()
	if err := writeSave(path, f); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(path)
	t.Logf("writeSave (JSON): %.2f s, %.1f MB on disk", time.Since(t0).Seconds(), float64(st.Size())/(1<<20))
	t0 = time.Now()
	g, err := ReadSave(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ReadSave: %.2f s", time.Since(t0).Seconds())
	t0 = time.Now()
	if _, err := NewFromSave(Config{}, g.World); err != nil {
		t.Fatal(err)
	}
	t.Logf("NewFromSave (Load = generate again + entities): %.1f s, live heap %.0f MB", time.Since(t0).Seconds(), mem())
}
