package sim

// Difficulty is how hard a world is, chosen when it is made and kept in its
// save. It scales what the wild does to players: how hard beasts hit, how
// fast hunger and winter-night cold eat at them, and how many packs of night
// hunters come out. The world itself (its land, its beasts and their health)
// is the same at every level.
type Difficulty uint8

const (
	DiffNormal Difficulty = iota // the zero value: worlds saved before difficulty existed
	DiffEasy
	DiffHard
	NumDifficulties
)

// Name is the difficulty as the game shows it.
func (d Difficulty) Name() string {
	switch d {
	case DiffEasy:
		return "Dễ"
	case DiffHard:
		return "Khó"
	}
	return "Thường"
}

// diffTune is what one difficulty changes.
type diffTune struct {
	harm   float64 // beasts' blows on players are multiplied by this
	hunger float64 // and hunger drains this much faster
	cold   float64 // and the cold bites this much harder
	packs  int     // night-hunter packs about one player at most
}

var diffTunes = [NumDifficulties]diffTune{
	DiffNormal: {harm: 1, hunger: 1, cold: 1, packs: TroodonPacksNear},
	DiffEasy:   {harm: 0.5, hunger: 0.7, cold: 0.5, packs: 1},
	DiffHard:   {harm: 1.5, hunger: 1.3, cold: 1.5, packs: 3},
}

func (d Difficulty) tune() diffTune {
	if d >= NumDifficulties {
		d = DiffNormal
	}
	return diffTunes[d]
}

// Difficulty is the world's difficulty.
func (w *World) Difficulty() Difficulty { return w.difficulty }
