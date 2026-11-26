package dinopaint

import "sort"

// Species is how one animal is painted. Every position and size is in
// pixels of its density-1 drawing, the guide: the painting is that drawing
// three times as fine, so the rig's joints (internal/client/dinorigs.go)
// land on the same anatomy.
type Species struct {
	Name string // the drawing: dinos/<Name>.png

	// Materials are the colours of the hide. Each pixel of the guide is the
	// material whose seed colours are nearest its own, so the regions and
	// markings of the drawing are kept; the painting then lights each with
	// its own ramp.
	Materials []Material
	Ink       float64 // guide pixels darker than this (luminance, 0-255) are outline, not hide

	// Parts are the masses the body is made of, back to front: the far
	// legs, the body, the near legs. A pixel belongs to the frontmost part
	// whose shape holds it; each part is rounded over from its own edge, so
	// a thigh bulges in front of the flank, and casts a soft shadow on what
	// is behind it.
	Parts []Part

	Skin       Skin
	Osteoderms []Row  // raised rows along the back
	Folds      []Fold // creases at the neck and the joints

	// Add and Remove change the silhouette traced from the guide: claws
	// sharpened, a snout made crisper.
	Add, Remove []SDF

	// Details paints what is drawn last, over the lit hide: eyes, nostrils,
	// teeth, claws, horns.
	Details func(p *Painter)
}

// Material is one colour of hide.
type Material struct {
	Name   string
	Seeds  []uint32 // colours of the guide that are this material
	Ramp   []uint32 // its colours from deep shadow to the highlight
	Relief float64  // how strongly the scales show on it (0: 1)
	Region SDF      // where it is, whatever the guide says (nil: from the guide)
	Over   bool     // with Region: only over the guide's pixels of the materials listed in Only
	Only   []string
}

// Part is a mass of the body.
type Part struct {
	Name    string
	Shape   SDF                        // all of it, also where something in front hides it
	Far     bool                       // on the far side of the body: darker
	Round   float64                    // how far in from its edge it rounds over (0: 5)
	RoundAt func(x, y float64) float64 // Round from place to place, when it changes
	Soft    float64                    // how much its form is softened (0: 0.8)
	Muscles []Muscle
	Scale   float64 // its scales' size, times the skin's (0: 1)
	Mat     string  // all of one material ("": from the guide)

	// Onto is the part it grows out of: outside Seam its surface runs
	// smoothly into that one's, as a thigh's top does into the hip. Seam
	// is where its edge over what is behind is a crease, with a line and a
	// shadow (nil: all along its edge).
	Onto string
	Seam SDF
}

// creased reports whether part pt's edge at guide point (x, y) is a crease.
func (pt *Part) creased(x, y float64) bool { return pt.Seam == nil || pt.Seam(x, y) < 0 }

// Muscle is a bulge on a part: an ellipse rx by ry, turned rot, amp high.
type Muscle struct{ X, Y, RX, RY, Rot, Amp float64 }

// Mu is a muscle.
func Mu(x, y, rx, ry, rot, amp float64) Muscle { return Muscle{x, y, rx, ry, rot, amp} }

// Skin is the scales, in image pixels of the painting (three to a pixel of
// the guide): their radius on the flank, along the back and on the belly.
type Skin struct {
	Flank, Back, Belly float64
	BackDepth          float64 // how far below the top edge the back's big scales reach (guide pixels; 0: 5)
	BellyDepth         float64 // and above the bottom edge the belly's small ones (0: 6)
	Relief             float64 // how strongly they show (0: 1)
	Seed               uint32
}

// Row is a row of osteoderms, bony knobs set in the skin, Off below the
// top edge of the silhouette from column X0 to X1, Gap apart and R across
// (guide pixels). Parts limits it to those parts.
type Row struct {
	X0, X1, Off, Gap, R float64
	Crest               float64 // how far the knobs stand out of the silhouette (0: not at all)
	Parts               []string
}

// Fold is a set of creases in the skin: N lines like the polyline Pts,
// each Gap further along (DX, DY), fading towards their ends.
type Fold struct {
	Pts    []float64
	N      int
	DX, DY float64
	Depth  float64 // how dark (0: 1)
}

var registry = map[string]*Species{}

// Register adds a species to those the tool can paint.
func Register(s *Species) { registry[s.Name] = s }

// Lookup is a registered species.
func Lookup(name string) (*Species, bool) { s, ok := registry[name]; return s, ok }

// Names are the registered species, sorted.
func Names() []string {
	var n []string
	for k := range registry {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}
