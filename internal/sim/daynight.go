package sim

// Phase is the part of the day.
type Phase uint8

const (
	PhaseDay Phase = iota
	PhaseDusk
	PhaseNight
)

// Fractions of DayTicks at which each phase starts. A day begins with a short
// dawn ramp, so it reads: dawn → day → dusk → night → (next day's dawn).
const (
	DuskStart  = 0.55
	NightStart = 0.70
	transition = 0.04 // length of the dawn and nightfall ramps

	duskAmbient = 0.4 // ambient light at the end of dusk
)

// StartTick is where a new world's clock starts: just after dawn on day 1, so
// a fresh player never spawns into darkness.
const StartTick = uint64(DayTicks / 10)

// DayOf returns the 1-based day number at tick.
func DayOf(tick uint64) int { return int(tick/DayTicks) + 1 }

// DayFraction returns how far through the current day tick is, in [0, 1).
func DayFraction(tick uint64) float64 {
	return float64(tick%DayTicks) / DayTicks
}

// PhaseOf returns the phase of the day at tick.
func PhaseOf(tick uint64) Phase {
	f := DayFraction(tick)
	switch {
	case f >= NightStart:
		return PhaseNight
	case f >= DuskStart:
		return PhaseDusk
	default:
		return PhaseDay
	}
}

// Ambient returns the ambient light level at tick: 1 in full day, 0 in deep
// night, with smooth ramps between.
func Ambient(tick uint64) float64 {
	f := DayFraction(tick)
	switch {
	case f < transition: // dawn
		return f / transition
	case f < DuskStart:
		return 1
	case f < NightStart: // dusk
		return 1 - (1-duskAmbient)*(f-DuskStart)/(NightStart-DuskStart)
	case f < NightStart+transition: // nightfall
		return duskAmbient * (1 - (f-NightStart)/transition)
	default:
		return 0
	}
}
