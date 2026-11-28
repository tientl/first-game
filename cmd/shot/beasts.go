package main

import (
	"flag"
	"log"
	"strconv"
	"strings"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

var beastsFlag = flag.String("beasts", "", "put beasts round the player: kind,dx,dy[,state][,left] separated by ';' (kind by number or short name: rabbit, crab, brachio, tribesman...; state: idle, wander, flee, chase, strike, eat, sleep, claw, spit, hurt, cower, roar); a long-neck at 0,1.5 stands in front of the player")

var beastStates = map[string]sim.CreatureState{
	"idle": sim.CreatureIdle, "wander": sim.CreatureWander, "flee": sim.CreatureFlee, "rest": sim.CreatureRest,
	"sleep": sim.CreatureSleep, "chase": sim.CreatureChase, "strike": sim.CreatureStrike, "hurt": sim.CreatureHurt,
	"roar": sim.CreatureRoar, "cower": sim.CreatureCower, "eat": sim.CreatureEat, "drink": sim.CreatureDrink,
	"claw": sim.CreatureClaw, "spit": sim.CreatureSpit, "trapped": sim.CreatureTrapped,
}

// beastNames are the kinds by the short names the code gives them, in order.
var beastNames = strings.Fields("none rabbit snake worm fish crab croc trike stego brachio raptor rex ptero plesio mammoth sabre ornitho thesc anzu toro achero quetzal brachy champso spider bear tribesman mosa kraken scorpion monkey troodon titanoboa spino mammothking carcharo giganoto megalodon toad glowtoad ankylo")

// beastKind is the kind named s: a number, a short name, or a word of its
// name in the game.
func beastKind(s string) sim.CreatureKind {
	if n, err := strconv.Atoi(s); err == nil {
		return sim.CreatureKind(n)
	}
	s = strings.ToLower(s)
	for i, n := range beastNames {
		if n == s {
			return sim.CreatureKind(i)
		}
	}
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		if strings.Contains(strings.ToLower(k.Name()), s) {
			return k
		}
	}
	log.Fatalf("-beasts: no creature called %q", s)
	return 0
}

// putBeasts adds -beasts' creatures round (sx, sy) to snap.
func putBeasts(snap *protocol.Snapshot, sx, sy float64) {
	if *beastsFlag == "" {
		return
	}
	id := sim.CreatureID(1<<28 + 900)
	for _, b := range strings.Split(*beastsFlag, ";") {
		f := strings.Split(strings.TrimSpace(b), ",")
		if len(f) < 3 {
			log.Fatalf("-beasts: %q is not kind,dx,dy", b)
		}
		dx, _ := strconv.ParseFloat(f[1], 64)
		dy, _ := strconv.ParseFloat(f[2], 64)
		c := protocol.CreatureView{Kind: beastKind(f[0]), X: float32(sx + dx), Y: float32(sy + dy)}
		for _, o := range f[3:] {
			if o == "left" {
				c.FaceLeft = true
			} else if st, ok := beastStates[o]; ok {
				c.State = st
			} else {
				log.Fatalf("-beasts: no state %q", o)
			}
		}
		id++
		c.ID = id
		c.HP, c.MaxHP = int16(min(c.Kind.MaxHP(), 32767)), int16(min(c.Kind.MaxHP(), 32767))
		snap.Creatures = append(snap.Creatures, c)
	}
}
