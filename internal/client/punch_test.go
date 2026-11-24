package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// punchAt shows me idle and then starting a blow, with held in hand and a
// rabbit in reach if beast.
func punchAt(g *Game, held sim.ItemKind, beast bool) protocol.PlayerView {
	var beasts []protocol.CreatureView
	if beast {
		beasts = []protocol.CreatureView{{ID: 3, Kind: sim.CreatureRabbit, X: 11.3, Y: 10.5}}
	}
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight, Held: held}
	g.onSnapshot(protocol.Snapshot{Tick: g.curr.Tick + 1, Players: []protocol.PlayerView{me}, Creatures: beasts})
	me.Acting = true
	g.onSnapshot(protocol.Snapshot{Tick: g.curr.Tick + 1, Players: []protocol.PlayerView{me}, Creatures: beasts})
	return me
}

func TestPunchIsThrownAndComesBack(t *testing.T) {
	g := readyGame()
	me := punchAt(g, sim.ItemNone, true)
	want := map[int]armPose{0: poseWind, 1: poseWind, 2: poseHalf, punchHit: poseFull, punchHold - 1: poseFull, punchHold: poseHalf, punchEnd - 1: poseHalf}
	start := g.frame
	for age := 0; age < punchEnd; age++ {
		g.frame = start + age
		fp, ok := g.punchPose(me)
		if !ok {
			t.Fatalf("frame %d: no punch", age)
		}
		if w, set := want[age]; set && fp.hand != w {
			t.Errorf("frame %d: the arm is in pose %d, want %d", age, fp.hand, w)
		}
		if fp.off != poseRest {
			t.Errorf("frame %d: the first punch is thrown with the other hand", age)
		}
		if (fp.spark >= 0) != (age >= punchHit && age < punchHit+sparkLife) {
			t.Errorf("frame %d: spark %d", age, fp.spark)
		}
	}
	g.frame = start + punchEnd
	if _, ok := g.punchPose(me); ok {
		t.Error("the arm is still out when the punch is over")
	}

	// The next one comes from the other hand, the one after from the first.
	me = punchAt(g, sim.ItemNone, true)
	g.frame += punchHit
	if fp, _ := g.punchPose(me); fp.off != poseFull || fp.hand != poseRest {
		t.Errorf("the second punch: hand %d, other hand %d", fp.hand, fp.off)
	}
	me = punchAt(g, sim.ItemNone, true)
	g.frame += punchHit
	if fp, _ := g.punchPose(me); fp.hand != poseFull || fp.off != poseRest {
		t.Errorf("the third punch: hand %d, other hand %d", fp.hand, fp.off)
	}
}

func TestReachAndToolsAreNotPunches(t *testing.T) {
	// Nothing to hit: a reach, with no wind-up, lean or spark.
	g := readyGame()
	me := punchAt(g, sim.ItemBerry, false)
	fp, ok := g.punchPose(me)
	if !ok || fp.hand != poseHalf || fp.lean != 0 {
		t.Errorf("a reach starts with %+v, %v", fp, ok)
	}
	g.frame += punchHit
	if fp, _ := g.punchPose(me); fp.hand != poseFull || fp.spark >= 0 || fp.lean != 0 {
		t.Errorf("a reach at its end is %+v", fp)
	}
	// Something held is always in the hand that holds things.
	g = readyGame()
	punchAt(g, sim.ItemStone, true)
	me = punchAt(g, sim.ItemStone, true)
	g.frame += punchHit
	if fp, _ := g.punchPose(me); fp.hand != poseFull || fp.off != poseRest {
		t.Errorf("a punch with a stone in the fist: %+v", fp)
	}
	// Tools and weapons keep their own swings.
	for _, k := range []sim.ItemKind{sim.ItemAxe, sim.ItemSpear, sim.ItemTorch, sim.ItemRod} {
		g = readyGame()
		me = punchAt(g, k, true)
		if _, ok := g.punchPose(me); ok {
			t.Errorf("item %d is swung like a fist", k)
		}
	}
}

func TestJoltStartsWhenStruck(t *testing.T) {
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, HurtSeq: 7}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}})
	if _, ok := g.jolting(1); ok {
		t.Fatal("jolts on first sight")
	}
	g.frame += 100
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}})
	if _, ok := g.jolting(1); ok {
		t.Fatal("jolts without a blow")
	}
	me.HurtSeq++
	g.onSnapshot(protocol.Snapshot{Tick: 3, Players: []protocol.PlayerView{me}})
	if age, ok := g.jolting(1); !ok || age != 0 {
		t.Fatalf("struck: jolting %d, %v", age, ok)
	}
	if dx, dy := joltShift(2, sim.DirRight); dx >= 0 || dy >= 0 {
		t.Errorf("facing right, thrown (%v, %v): want back and up", dx, dy)
	}
	g.frame += joltFrames
	if _, ok := g.jolting(1); ok {
		t.Error("still jolting when it is over")
	}
}

// Every arm of every look is drawn inside its canvas, and facing down or up
// the two arms mirror each other.
func TestArmsFitTheirCanvas(t *testing.T) {
	for _, female := range []bool{false, true} {
		l := sim.Look{Female: female}
		for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirRight} {
			for _, side := range []armSide{armHand, armOff} {
				for pose := poseWind; pose < numArmPoses; pose++ {
					S, F, sh := armCanvases(l, d, side, pose)
					if sh.none {
						continue
					}
					for _, c := range []*canvas{S, F} {
						for y := 0; y < c.h; y++ {
							for x := 0; x < c.w; x++ {
								if c.opaque(x, y) && (x == 0 || y == 0 || x == c.w-1 || y == c.h-1) {
									t.Fatalf("female %v dir %d side %d pose %d: drawn on the edge at (%d, %d)", female, d, side, pose, x, y)
								}
							}
						}
					}
				}
			}
		}
	}
}
