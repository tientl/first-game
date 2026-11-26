package sim

// Deeds: the short things a player does that others should see done, though
// the world changes in a tick: eating, drinking, filling a bowl at the
// water, tossing something out of the bag. Each bumps a counter the client
// watches, as HurtSeq is watched for blows, and plays the motion when it
// changes (see client/motion.go).

// Deed names a thing done.
type Deed uint8

const (
	DeedNone  Deed = iota
	DeedEat        // ate (or drank) one of DeedItem
	DeedScoop      // filled a bowl or a flask at the water: DeedItem is what it holds now
	DeedToss       // threw DeedItem out of the bag
)

// did notes that p did d with item k.
func (p *Player) did(d Deed, k ItemKind) {
	p.deed, p.deedItem = d, k
	p.deedSeq++
}

// Deed is the last thing p did, with what, and a count that goes up by one
// with every deed.
func (p *Player) Deed() (Deed, ItemKind, uint8) { return p.deed, p.deedItem, p.deedSeq }
