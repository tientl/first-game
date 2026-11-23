package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestSaveRequestAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "save.json")
	s, err := New(Config{Seed: 42, Width: 64, Height: 64, SavePath: path})
	if err != nil {
		t.Fatal(err)
	}
	c, id := join(t, s, "Tiến")
	s.world.Apply(id, sim.Command{Kind: sim.CmdEat, Slot: 0}) // an empty slot: harmless, just a notice
	send(t, c, protocol.ClientMsg{Kind: protocol.ClientSave})
	s.tick()

	saved := false
	for {
		select {
		case m := <-c.Messages():
			if m.Kind == protocol.ServerSaved {
				if !m.Saved.OK || m.Saved.Auto {
					t.Fatalf("save result = %+v, want OK and not auto", m.Saved)
				}
				saved = true
			}
			continue
		default:
		}
		break
	}
	if !saved {
		t.Fatal("no save result sent")
	}

	f, err := ReadSave(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.SavedAt.IsZero() || len(f.World.Players) != 1 || f.World.Players[0].Name != "Tiến" {
		t.Fatalf("save file = %+v", f)
	}
	s2, err := NewFromSave(Config{}, f.World)
	if err != nil {
		t.Fatal(err)
	}
	if s2.world.Tick() != f.World.Tick {
		t.Errorf("reloaded tick = %d, want %d", s2.world.Tick(), f.World.Tick)
	}
}

func TestShutdownSavesAndPauseStopsTheClock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	s, err := New(Config{Seed: 1, Width: 48, Height: 48, SavePath: path})
	if err != nil {
		t.Fatal(err)
	}
	before := s.world.Tick()
	s.SetPaused(true)
	s.tick()
	if s.world.Tick() != before {
		t.Error("paused server advanced the clock")
	}
	s.SetPaused(false)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSave(path); err != nil {
		t.Fatalf("no save after shutdown: %v", err)
	}
}

func TestReadSaveErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadSave(filepath.Join(dir, "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing file: err = %v, want ErrNotExist", err)
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{not json"), 0o644)
	if _, err := ReadSave(bad); !errors.Is(err, sim.ErrBadSave) {
		t.Errorf("corrupt file: err = %v, want ErrBadSave", err)
	}
}
