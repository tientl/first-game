package sim

// Look is how a player's character looks: chosen on the character screen when
// a new game starts, kept in the save, and shown to everyone. The zero Look
// is the original survivor: a man with short dark-brown hair and the first skin
// tone, so players from saves made before looks existed keep their appearance.
type Look struct {
	Female    bool
	Hair      HairStyle
	HairColor uint8 // index into the client's hair palette, below NumHairColors
	Skin      uint8 // index into the client's skin tones, below NumSkinTones
}

// HairStyle is a haircut. Any body can wear any style.
type HairStyle uint8

const (
	HairShort    HairStyle = iota // short, with a fringe
	HairSpiky                     // spiked up
	HairBob                       // chin length, straight fringe
	HairLong                      // long, down the back
	HairPonytail                  // pulled back into a tail
	HairBun                       // tied up in a bun on top
	HairShaggy                    // long and unkempt, to the shoulders
	NumHairStyles
)

// NumHairColors is how many hair colours there are to pick from.
const NumHairColors = 8

// NumSkinTones is how many skin colours there are to pick from.
const NumSkinTones = 6

// Valid reports whether every field of l is in range.
func (l Look) Valid() bool {
	return l.Hair < NumHairStyles && l.HairColor < NumHairColors && l.Skin < NumSkinTones
}

// Sanitized returns l with anything out of range (from a newer client or a
// hand-edited save) set back to the default.
func (l Look) Sanitized() Look {
	if l.Hair >= NumHairStyles {
		l.Hair = HairShort
	}
	if l.HairColor >= NumHairColors {
		l.HairColor = 0
	}
	if l.Skin >= NumSkinTones {
		l.Skin = 0
	}
	return l
}

// DefaultHair is the style a new character of the given body starts with.
func DefaultHair(female bool) HairStyle {
	if female {
		return HairLong
	}
	return HairShort
}
