package sim

// Difficulty is how hard a world is, chosen when it is made and kept in its
// save. It scales what the wild does to players (how hard beasts hit, how
// fast hunger and winter-night cold eat at them, how many packs of night
// hunters come out) and what dying costs: at the basic level a player comes
// back with everything; at the harder two their things stay with their body
// where they fell, to be fetched (grave.go). The world itself (its land, its
// beasts and their health) is the same at every level.
type Difficulty uint8

const (
	DiffBasic   Difficulty = iota // Cơ bản; the zero value, so worlds saved before difficulty existed
	DiffSkilled                   // Thành thạo
	DiffLord                      // Chúa tể
	NumDifficulties
)

// Name is the difficulty as the game shows it.
func (d Difficulty) Name() string {
	switch d {
	case DiffSkilled:
		return "Thành thạo"
	case DiffLord:
		return "Chúa tể"
	}
	return "Cơ bản"
}

// diffTune is what one difficulty changes.
type diffTune struct {
	harm      float64 // beasts' blows on players are multiplied by this
	hunger    float64 // and hunger drains this much faster
	cold      float64 // and the cold bites this much harder
	packs     int     // night-hunter packs about one player at most
	keepItems bool    // a player who dies keeps what they carried
}

var diffTunes = [NumDifficulties]diffTune{
	DiffBasic:   {harm: 1, hunger: 1, cold: 1, packs: TroodonPacksNear, keepItems: true},
	DiffSkilled: {harm: 1.5, hunger: 1.3, cold: 1.5, packs: 3},
	DiffLord:    {harm: 2, hunger: 1.6, cold: 2, packs: 4},
}

func (d Difficulty) tune() diffTune {
	if d >= NumDifficulties {
		d = DiffBasic
	}
	return diffTunes[d]
}

// KeepsItems reports whether a player who dies at difficulty d keeps what
// they carried.
func (d Difficulty) KeepsItems() bool { return d.tune().keepItems }

// Difficulty is the world's difficulty.
func (w *World) Difficulty() Difficulty { return w.difficulty }
