package client

import "sync"

// The six bosses' art, and the icons of the legendary items they drop. This
// file is the contract between the bosses' logic (internal/sim, wired into
// the client elsewhere) and their drawings: the logic only calls bossCanvas
// and bossItemCanvas. The bodies below are placeholders until the art is in.

// bossID names a boss, in this order everywhere.
type bossID uint8

const (
	bossTitanoboa   bossID = iota // a giant snake of the jungle
	bossSpino                     // Spinosaurus, of the swamp
	bossMammothKing               // the mammoth king, of the snow
	bossCarcharo                  // Carcharodontosaurus, of the desert
	bossGiganoto                  // Giganotosaurus, of the volcano
	bossMegalodon                 // a giant shark, of the open sea
	numBosses
)

// bossPose is what a boss's sprite shows.
type bossPose uint8

const (
	bossIdle    bossPose = iota // standing (Titanoboa coiled; Megalodon swimming)
	bossWalkA                   // walking, slithering or swimming: two frames
	bossWalkB                   //
	bossAttack                  // its main blow: a bite (the Mammoth King: a tusk swipe)
	bossSpecial                 // its own move: see bossSpecialMove
	bossRoar                    // roaring, trumpeting or hissing: the fight begins, or it is enraged
	bossLurk                    // hidden: Spinosaurus' sail and eyes above the water, Megalodon's fin; the others crouched or coiled low
	numBossPoses
)

// bossSpecialMove says what bossSpecial shows for each boss.
var bossSpecialMove = [numBosses]string{
	bossTitanoboa:   "constricting: coiled tight around its prey",
	bossSpino:       "lunging out of the water, jaws wide",
	bossMammothKing: "rearing up on its hind legs to stomp",
	bossCarcharo:    "charging at a run, head low",
	bossGiganoto:    "sweeping its tail",
	bossMegalodon:   "breaching: leaping out of the water, jaws open",
}

// bossCanvas draws pose p of boss b facing right, with its anchor in the
// canvas's design units (give them to mk): the middle of its feet on the
// ground, or for the Megalodon and a lurking Spinosaurus the waterline under
// its middle. (Spinosaurus' lunge comes out of the water too, and is anchored
// on the waterline where it breaks it.) The canvas is cut to fit each pose,
// so its size changes from pose to pose; the anchor keeps the feet still.
// The drawings are made once and kept: do not draw on the canvas returned.
func bossCanvas(b bossID, p bossPose) (*canvas, float64, float64) {
	if b >= numBosses || p >= numBossPoses {
		b, p = min(b, numBosses-1), bossIdle
	}
	bossMu.Lock()
	defer bossMu.Unlock()
	d := &bossDrawn[b][p]
	if d.c == nil {
		d.c, d.ax, d.ay = bossDraw(b, p)
	}
	return d.c, d.ax, d.ay
}

var (
	bossMu    sync.Mutex
	bossDrawn [numBosses][numBossPoses]struct {
		c      *canvas
		ax, ay float64
	}
)

func bossDraw(b bossID, p bossPose) (*canvas, float64, float64) {
	switch b {
	case bossTitanoboa:
		return titanoboaCanvas(p)
	case bossSpino:
		if p == bossLurk {
			return spinoLurkCanvas()
		}
		return theroCanvas(&spinoLook, theroPoses(b, p))
	case bossMammothKing:
		return mammothKingCanvas(p)
	case bossCarcharo:
		return theroCanvas(&carchLook, theroPoses(b, p))
	case bossGiganoto:
		return theroCanvas(&gigaLook, theroPoses(b, p))
	}
	return megalodonCanvas(p)
}

// bossItemCanvas is the bag icon of the legendary item boss b drops: 16 by
// 16, like the other item icons (see itemicons.go).
func bossItemCanvas(b bossID) *canvas {
	return bossItemIcon(b)
}
