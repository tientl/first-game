package sim

// The demo. A build for people to try the game has only two of the six
// bosses: Titanoboa in the deepest jungle and Megalodon in the deep sea, one
// fight on land and one at sea. The others are not raised, have no lair on
// the map, and the game is beaten when these two have fallen. Everything
// else plays as in the full game.
//
// It is on for a binary built with `-tags demo` (see demo_tag.go), or with
// SetDemo, e.g. cmd/game's -demo flag.

var demo = demoBuild

// demoBosses are the bosses of the demo.
var demoBosses = []CreatureKind{CreatureTitanoboa, CreatureMegalodon}

// SetDemo turns the demo on or off for worlds made or loaded from now on.
func SetDemo(on bool) { demo = on }

// Demo reports whether this is the demo.
func Demo() bool { return demo }

// bossActive reports whether boss k is in this game.
func bossActive(k CreatureKind) bool {
	if !demo {
		return true
	}
	for _, d := range demoBosses {
		if d == k {
			return true
		}
	}
	return false
}
