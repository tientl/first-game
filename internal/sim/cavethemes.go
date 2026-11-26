package sim

// The themes of the natural caves' chambers and what stands in them.

// CaveProp is which thing of the caves an EntityCaveProp is.
type CaveProp uint8

// touchProp is a player's look at a cave prop.
func (w *World) touchProp(p *Player, e *Entity) bool { return false }

// caveHazards is what the caves' webs and lava do to p.
func (w *World) caveHazards(p *Player) {}
