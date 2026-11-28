package sim

import (
	"testing"
	"time"
)

func TestOnlyBusyEntitiesAreStepped(t *testing.T) {
	w := genOpen(t, 1, SizeLarge)
	if n := len(w.active); n > 50 {
		t.Fatalf("a fresh world steps %d of %d entities, want only the few that change", n, len(w.entities))
	}

	// A twig picked up (E) is stepped until it grows back, then dropped.
	pw := flatWorld()
	id := pw.AddPlayer("a")
	twig := pw.addEntity(EntityTwig, 17, 16)
	pw.Apply(id, Command{Kind: CmdInteract})
	pw.Step()
	if twig.Ready || pw.active[twig.ID] == nil {
		t.Fatalf("picked twig: ready %v, active %v", twig.Ready, pw.active[twig.ID] != nil)
	}
	pw.RemovePlayer(id)
	stepN(pw, TwigRegrowTicks)
	if !twig.Ready {
		t.Fatal("twig did not grow back on time")
	}
	pw.Step()
	if pw.active[twig.ID] != nil {
		t.Error("a grown twig is still stepped")
	}

	// Something changed behind watch's back is caught by the sweep.
	tuft := pw.addEntity(EntityGrassTuft, 10, 10)
	tuft.Ready, tuft.RegrowAt = false, pw.tick+5
	stepN(pw, activeSweepTicks+5)
	if !tuft.Ready {
		t.Error("the sweep missed a tuft made not Ready directly")
	}
}

func TestBigWorldStepsQuickly(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	w := genOpen(t, 1, SizeHuge)
	w.AddPlayer("a")
	stepN(w, 5) // warm up
	t0 := time.Now()
	const n = 100
	stepN(w, n)
	per := time.Since(t0) / n
	t.Logf("1024x1024 open world, one player: %v per tick (%d entities, %d active)", per, len(w.entities), len(w.active))
	// While everyone sleeps the server runs SleepSpeedup steps per 50 ms tick.
	if budget := TickDuration / SleepSpeedup; per > budget {
		t.Errorf("%v per tick is over the %v budget of sped-up sleep", per, budget)
	}
}
