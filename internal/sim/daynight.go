package sim

import "math"

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
	case f >= NightAt(tick):
		return PhaseNight
	case f >= DuskAt(tick):
		return PhaseDusk
	default:
		return PhaseDay
	}
}

// Ambient returns the ambient light level at tick: 1 in full day, 0 in deep
// night, with smooth ramps between.
func Ambient(tick uint64) float64 {
	f := DayFraction(tick)
	dusk, night := DuskAt(tick), NightAt(tick)
	switch {
	case f < transition: // dawn
		return f / transition
	case f < dusk:
		return 1
	case f < night: // dusk
		return 1 - (1-duskAmbient)*(f-dusk)/(night-dusk)
	case f < night+transition: // nightfall
		return duskAmbient * (1 - (f-night)/transition)
	default:
		return 0
	}
}

// The days draw out towards midsummer and in towards midwinter: dusk and
// nightfall come up to dayShift of a day later in summer, earlier in winter.
// DuskStart and NightStart are the equinox's.
const (
	dayShift   = 0.07
	nightShift = 0.06
	midsummer  = 0.375 // through the year: the middle of summer
)

// seasonSwing is +1 at midsummer, -1 at midwinter, 0 at the equinoxes.
func seasonSwing(tick uint64) float64 {
	year := float64(SeasonDays*int(NumSeasons)) * DayTicks
	y := math.Mod(float64(tick), year) / year
	return math.Cos(2 * math.Pi * (y - midsummer))
}

// DuskAt is the fraction of the day dusk starts at on the day of tick.
func DuskAt(tick uint64) float64 {
	return DuskStart + dayShift*seasonSwing(tick-tick%DayTicks)
}

// NightAt is the fraction of the day night falls at on the day of tick.
func NightAt(tick uint64) float64 {
	return NightStart + nightShift*seasonSwing(tick-tick%DayTicks)
}
