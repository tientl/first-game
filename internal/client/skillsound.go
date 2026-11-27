package client

import (
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The beasts' moves heard (sim/skills.go): a cue of its own as each wind-up
// begins, from the recorded voices (realvoice.go) pitched and chosen per
// move, so a heavy move is heard coming as well as seen; then the sound of
// the blow as it lands.

// skillCue is how one move sounds.
type skillCue struct {
	voice     string  // as the wind-up begins
	vol, rate float64 // its loudness and pitch
	hit       string  // as it lands (a synthesised sound)
	hitVol    float64
	roar      bool // the tyrant's roar, with its shaking ground (rexroar.go)
	landVoice string
	landVol   float64
	landRate  float64
}

var skillCues = map[sim.SkillID]skillCue{
	sim.SkillRexBite:   {voice: "rex_growl", vol: 0.9, rate: 1.1, hit: "chomp", hitVol: 1},
	sim.SkillRexRoar:   {roar: true},
	sim.SkillRexTail:   {voice: "rex_growl", vol: 0.8, rate: 0.85, hit: "swish", hitVol: 0.9},
	sim.SkillRexCharge: {voice: "rex_hurt", vol: 0.9, rate: 0.8, hit: "stomp", hitVol: 1},

	sim.SkillDakotaPounce:  {voice: "raptor_growl", vol: 0.7, rate: 1, hit: "raptor_attack", hitVol: 0.7},
	sim.SkillDakotaSickle:  {voice: "raptor_attack", vol: 0.5, rate: 1.15, hit: "swish", hitVol: 0.6},
	sim.SkillDakotaScreech: {voice: "raptor_alarm", vol: 0.9, rate: 1.05},
	sim.SkillAcheroSnap:    {voice: "achero_growl", vol: 0.5, rate: 1.1, hit: "chomp", hitVol: 0.5},

	sim.SkillTroodonAmbush: {voice: "troodon_growl", vol: 0.6, rate: 0.9, hit: "troodon_attack", hitVol: 0.6},
	sim.SkillTroodonVenom:  {voice: "hiss", vol: 0.5, rate: 1.2, hit: "chomp", hitVol: 0.4},
	sim.SkillTroodonCall:   {voice: "troodon_alarm", vol: 0.9, rate: 1.1},

	sim.SkillPteroDive:   {voice: "ptero_alarm", vol: 0.7, rate: 1, hit: "swish", hitVol: 0.6},
	sim.SkillPteroSnatch: {voice: "ptero_call", vol: 0.6, rate: 1.1, hit: "swish", hitVol: 0.5},

	sim.SkillQuetzalStab:   {voice: "quetzal_alarm", vol: 0.7, rate: 1, hit: "whack", hitVol: 0.7},
	sim.SkillQuetzalBuffet: {voice: "quetzal_call", vol: 0.8, rate: 0.9, hit: "swish", hitVol: 1},

	sim.SkillPlesioNeck:   {voice: "plesio_alarm", vol: 0.7, rate: 1, hit: "chomp", hitVol: 0.8},
	sim.SkillPlesioAmbush: {voice: "splash", vol: 0.35, rate: 0.8, hit: "splash", hitVol: 1, landVoice: "plesio_alarm", landVol: 0.8, landRate: 0.9},

	sim.SkillSabrePounce: {voice: "sabre_growl", vol: 0.7, rate: 0.95, hit: "snarl", hitVol: 0.7},
	sim.SkillSabreBite:   {voice: "sabre_snarl", vol: 0.7, rate: 1, hit: "chomp", hitVol: 0.8},

	sim.SkillTrikeCharge: {voice: "trike_alarm", vol: 0.8, rate: 0.95, hit: "stomp", hitVol: 0.8},
	sim.SkillTrikeToss:   {voice: "trike_hurt", vol: 0.6, rate: 1.2, hit: "whack", hitVol: 0.9},
	sim.SkillTrikeCall:   {voice: "trike_call", vol: 0.9, rate: 1.05},

	sim.SkillToroDisplay: {voice: "toro_alarm", vol: 0.8, rate: 0.95},
	sim.SkillToroCharge:  {voice: "toro_alarm", vol: 0.9, rate: 0.85, hit: "stomp", hitVol: 0.9},

	sim.SkillStegoFlush: {voice: "stego_alarm", vol: 0.7, rate: 1},
	sim.SkillStegoTail:  {voice: "stego_alarm", vol: 0.7, rate: 0.85, hit: "swish", hitVol: 0.9},

	sim.SkillBrachioStomp: {voice: "brachio_alarm", vol: 0.9, rate: 0.9, hit: "boom", hitVol: 1},
	sim.SkillBrachioWhip:  {voice: "brachio_call", vol: 0.7, rate: 1.2, hit: "swish", hitVol: 1},

	sim.SkillAnkyloSlam:   {voice: "ankylo_alarm", vol: 0.8, rate: 0.95, hit: "boom", hitVol: 0.9},
	sim.SkillAnkyloTurtle: {voice: "ankylo_hurt", vol: 0.6, rate: 0.9},

	sim.SkillAnzuKick:    {voice: "anzu_call", vol: 0.5, rate: 1.2, hit: "whack", hitVol: 0.6},
	sim.SkillAnzuDisplay: {voice: "anzu_roar", vol: 0.7, rate: 1, hit: "pinch", hitVol: 0.6},

	sim.SkillThescKick:   {voice: "thesc_alarm", vol: 0.5, rate: 1.1, hit: "whack", hitVol: 0.5},
	sim.SkillOrnithoKick: {voice: "ornitho_alarm", vol: 0.5, rate: 1.1, hit: "whack", hitVol: 0.5},

	sim.SkillMammothCharge:  {voice: "mammoth_alarm", vol: 0.9, rate: 0.9, hit: "stomp", hitVol: 0.9},
	sim.SkillMammothSwipe:   {voice: "mammoth_call", vol: 0.6, rate: 1.25, hit: "swish", hitVol: 0.9},
	sim.SkillMammothTrumpet: {voice: "mammoth_call", vol: 1, rate: 1},
}

// skillSound plays what a beast's move did between two snapshots, o then c,
// and reports whether the change was its move's (so nothing else plays for
// it).
func skillSound(o, c protocol.CreatureView) bool {
	if c.Skill == o.Skill && c.Phase == o.Phase {
		return c.Skill != sim.SkillNone // the same beat of the same move: nothing new
	}
	cue, ok := skillCues[c.Skill]
	x, y := float64(c.X), float64(c.Y)
	switch {
	case c.Skill == sim.SkillNone:
		return o.Skill != sim.SkillNone // the move is over
	case !ok:
		return true
	case c.Phase == sim.PhaseWindup && (o.Skill != c.Skill || o.Phase != sim.PhaseWindup):
		if cue.roar {
			roarAt(x, y, 1) // the growl in its chest is the wind-up, the jaws open as it lands
			return true
		}
		if cue.voice != "" {
			callAt(cue.voice, x, y, cue.vol, cue.rate, 24)
		}
	case c.Phase == sim.PhaseActive:
		if cue.hit != "" {
			snd.playAt(cue.hit, x, y, cue.hitVol, 0.08)
		}
		if cue.landVoice != "" {
			callAt(cue.landVoice, x, y, cue.landVol, cue.landRate, 24)
		}
	}
	return true
}
