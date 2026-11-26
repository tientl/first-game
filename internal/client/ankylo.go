package client

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The Ankylosaurus, the armoured one (sim/ankylo.go): drawn from
// dinos/ankylo.png by its rig ("ankylo" in dinorigs.go), every pose drawn the
// first time it is shown. Turned on a threat it holds its club raised; its
// blow is a crouch with the club drawn back away from the viewer, then a
// sideways swing round towards them (ankylorig.go), blurred as it goes;
// backing towards whoever hit it, it walks backwards. Here too are its
// name, its voice, and the icon and note of its bony plates.

// ankyloScale is how big it is drawn against its design, for its shadow
// and the seat of its rider: as big as the plate-back.
const ankyloScale = 1.6

// Its poses, in dinoPoses["ankylo"].
const (
	ankyloStand = iota
	ankyloWalkB
	ankyloGuard // its back to a threat, the club raised
	ankyloSwing // the club swung round: the blow
	ankyloHurt  // head up, beak open: hit, or honking
	ankyloEatA  // head down to the ground: beak shut...
	ankyloEatB  // ...and open
)

// ankyloWind is the crouch before the blow: the club drawn back, away from
// the viewer and up, the head down.
var ankyloWind = rigPose{squash: 0.93, head: 0.18, tail: 0.22, sweep: -0.75}

func ankyloPose(side, p int) sprite { return rigSprite("ankylo", side, dinoPoses["ankylo"][p]) }

// ankyloSprite is creatureSprite for the Ankylosaurus; ok is false for any
// other kind.
func (g *Game) ankyloSprite(c protocol.CreatureView, moving bool, side, t int) (sprite, float64, bool) {
	if c.Kind != sim.CreatureAnkylo {
		return sprite{}, 0, false
	}
	switch {
	case feeding(c):
		return ankyloPose(side, ankyloEatA+chew(c, t)), 0, true
	case c.State == sim.CreatureStrike:
		return ankyloBlow(side, g.strikeAge(c), c.Tamed), 0, true
	case c.State == sim.CreatureHurt, calling(c):
		return ankyloPose(side, ankyloHurt), 0, true
	case moving:
		base := rigPose{}
		if c.State == sim.CreatureChase {
			base = dinoPoses["ankylo"][ankyloGuard]
		}
		f := t * gaitFrames / (2 * 16)
		if g.backingUp(c) { // the stride played backwards
			f = -f - 1
		}
		f = (f%gaitFrames + gaitFrames) % gaitFrames
		return rigSprite("ankylo", side, stridePose(base, f, 1)), 0, true
	case c.State == sim.CreatureRoar || c.State == sim.CreatureChase:
		return rigSprite("ankylo", side, ankyloGuardPose(t)), 0, true
	}
	return ankyloPose(side, ankyloStand), 0, true
}

// backingUp reports whether c moved towards its tail since the last snapshot.
func (g *Game) backingUp(c protocol.CreatureView) bool {
	q, ok := g.prevCreatures[c.ID]
	if !ok {
		return false
	}
	dx := float64(c.X - q.X)
	return (dx > 0 && c.FaceLeft) || (dx < 0 && !c.FaceLeft)
}

// ankyloGuardPose is its guard at frame t: the club raised and swaying a
// little from side to side, slowly, in eight steps.
func ankyloGuardPose(t int) rigPose {
	p := dinoPoses["ankylo"][ankyloGuard]
	ph := 2 * math.Pi * float64((t/9)%8) / 8
	p.sweep = math.Round(0.14*math.Sin(ph)*100) / 100
	p.tail += math.Round(0.03*math.Cos(ph)*100) / 100
	return p
}

// The blow, frame by frame (60 to a second), as the sim times it: drawn
// back as long as the club is wound up (sim.WindupTicks), swung round in
// ankyloSwingFrames frames to land as the sim lands it, held, then brought
// back to its guard. A pet's blow has no wind-up to speak of.
const (
	ankyloSwingFrames = 5
	ankyloHold        = 8
	ankyloBack        = 10
)

// ankyloBlow is the armoured one's blow, age frames in (see strikeAge).
func ankyloBlow(side, age int, quick bool) sprite {
	p, smear := ankyloBlowFrame(age, quick)
	if smear >= 0 {
		return ankyloSmear(side, smear)
	}
	return rigSprite("ankylo", side, p)
}

// ankyloBlowFrame is frame age of the blow: a pose of the rig, or the
// blurred frame smear of the swing (smear < 0: none).
func ankyloBlowFrame(age int, quick bool) (p rigPose, smear int) {
	wind := sim.WindupTicks(sim.CreatureAnkylo) * 60 / sim.TicksPerSecond
	if quick {
		wind = 8
	}
	guard := dinoPoses["ankylo"][ankyloGuard]
	swing := dinoPoses["ankylo"][ankyloSwing]
	start := wind - ankyloSwingFrames + 2 // the swing's middle meets the sim's blow
	switch {
	case age < 0:
		return guard, -1
	case age < start:
		// Into the crouch over the first frames, two to a step, then held.
		return lerpPose(guard, ankyloWind, min(1, float64(age/2+1)/3)), -1
	case age < start+ankyloSwingFrames:
		return rigPose{}, age - start
	case age < start+ankyloSwingFrames+ankyloHold:
		return swing, -1
	case age < start+ankyloSwingFrames+ankyloHold+ankyloBack:
		k := age - (start + ankyloSwingFrames + ankyloHold)
		return lerpPose(swing, guard, float64(k/3+1)/4), -1
	}
	return guard, -1
}

// ankyloSwingPose is frame k of the swing, eased: slow out of the crouch,
// fastest through the middle.
func ankyloSwingPose(k int) rigPose {
	t := float64(k+1) / ankyloSwingFrames
	t = t * t * (3 - 2*t)
	return lerpPose(ankyloWind, dinoPoses["ankylo"][ankyloSwing], math.Round(t*100)/100)
}

type smearKey struct{ side, k int }

var ankyloSmears = map[smearKey]sprite{}

// ankyloSmear is frame k of the swing with the club blurred behind it: the
// poses it has just swung through, faint and pale, under the pose itself.
func ankyloSmear(side, k int) sprite {
	key := smearKey{side, k}
	if s, ok := ankyloSmears[key]; ok {
		return s
	}
	c, ax, ay, ok := ankyloSmearCanvas(k)
	if !ok {
		return rigSprite("ankylo", side, ankyloSwingPose(k))
	}
	ankyloSmears[smearKey{0, k}], ankyloSmears[smearKey{1, k}] = sidedPair(mk(c, ax, ay))
	return ankyloSmears[key]
}

// ankyloSmearCanvas draws frame k of the swing, blurred, facing right.
func ankyloSmearCanvas(k int) (*canvas, float64, float64, bool) {
	r := dinoRigs["ankylo"]
	main, ax, ay, ok := rigCanvas(r, ankyloSwingPose(k))
	if !ok || k == 0 {
		return main, ax, ay, ok
	}
	type layer struct {
		c      *canvas
		ax, ay float64
		a      float64
	}
	layers := []layer{}
	// The way the club came since the frame before, in four steps, the
	// older the fainter.
	from := ankyloWind
	if k > 1 {
		from = ankyloSwingPose(k - 2)
	}
	to := ankyloSwingPose(k)
	for j, a := range []float64{0.14, 0.22, 0.3, 0.4, 0.5} {
		if c, gx, gy, ok := rigCanvas(r, lerpPose(from, to, float64(j+1)/6)); ok {
			layers = append(layers, layer{c, gx, gy, a})
		}
	}
	layers = append(layers, layer{main, ax, ay, 1})
	// One canvas round them all, their anchors on one point.
	left, top, right, bottom := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, l := range layers {
		left, top = min(left, -l.ax), min(top, -l.ay)
		right, bottom = max(right, float64(l.c.w)-l.ax), max(bottom, float64(l.c.h)-l.ay)
	}
	out := newCanvas(int(math.Ceil(right-left)), int(math.Ceil(bottom-top)))
	out.dens = main.dens
	dust := color.RGBA{0xd8, 0xcc, 0xb0, 255}
	for _, l := range layers {
		ox, oy := int(math.Round(-l.ax-left)), int(math.Round(-l.ay-top))
		for y := 0; y < l.c.h; y++ {
			for x := 0; x < l.c.w; x++ {
				q := l.c.px[y*l.c.w+x]
				if q.A < 128 {
					continue
				}
				i := (y+oy)*out.w + x + ox
				if l.a >= 1 {
					out.px[i] = q
					continue
				}
				if out.px[i].A >= 128 {
					continue
				}
				m := mix(q, dust, 0.45)
				out.px[i] = color.RGBA{m.R, m.G, m.B, uint8(255 * l.a)}
			}
		}
	}
	return out, -left, -top, true
}

// ankyloLabel is creatureLabel for a wild Ankylosaurus; ok is false for any
// other kind.
func ankyloLabel(c protocol.CreatureView) (string, color.RGBA, bool) {
	if c.Kind != sim.CreatureAnkylo {
		return "", color.RGBA{}, false
	}
	name := c.Kind.Name()
	switch c.State {
	case sim.CreatureStrike:
		return name + " — sắp quật chùy đuôi! lùi ra hoặc vòng lên đầu", colBad, true
	case sim.CreatureRoar, sim.CreatureChase:
		return name + " — đang quay đuôi về phía bạn: tránh xa chùy", colBad, true
	}
	return name + " — săn (giáp dày: đánh vào đầu, coi chừng chùy đuôi)", colWhite, true
}

// --- its voice ----------------------------------------------------------------------

// ankyloVoices is when the armoured ones last grunted, and which have swung
// their club in the blow they are striking.
var ankyloVoices = struct {
	grunt map[sim.CreatureID]time.Time
	swung map[sim.CreatureID]bool
}{map[sim.CreatureID]time.Time{}, map[sim.CreatureID]bool{}}

// ankyloSounds plays what the armoured ones did between two snapshots: a
// warning grunt as one turns its club on someone, an effort grunt as it
// crouches to strike, the whoosh of the club as it swings. Their cries
// (hit, dying, honking) are voicesOf's and callOf's.
func (g *Game) ankyloSounds(old, cur protocol.Snapshot) {
	was := map[sim.CreatureID]protocol.CreatureView{}
	for _, c := range old.Creatures {
		if c.Kind == sim.CreatureAnkylo {
			was[c.ID] = c
		}
	}
	now := time.Now()
	for _, c := range cur.Creatures {
		if c.Kind != sim.CreatureAnkylo {
			continue
		}
		x, y := float64(c.X), float64(c.Y)
		if c.State != sim.CreatureStrike {
			delete(ankyloVoices.swung, c.ID)
		}
		o, seen := was[c.ID]
		if seen && o.State != c.State {
			switch {
			case c.State == sim.CreatureRoar && calm(o.State):
				if now.Sub(ankyloVoices.grunt[c.ID]) > 4*time.Second {
					ankyloVoices.grunt[c.ID] = now
					callAt("ankylo_grunt", x, y, 0.75, 1, 20)
				}
			case c.State == sim.CreatureStrike:
				callAt("ankylo_grunt", x, y, 0.6, 1.18, 18) // the effort of the crouch
			}
		}
		if c.State == sim.CreatureStrike && !ankyloVoices.swung[c.ID] {
			wind := sim.WindupTicks(sim.CreatureAnkylo) * 60 / sim.TicksPerSecond
			if c.Tamed {
				wind = 8
			}
			if age := g.strikeAge(c); age >= wind-ankyloSwingFrames {
				ankyloVoices.swung[c.ID] = true
				snd.playAt("club_swing", x, y, 0.8, 0.06)
				snd.playAt("whack", x, y, 0.5, 0.05)
			}
		}
	}
	if len(ankyloVoices.grunt) > 256 {
		clear(ankyloVoices.grunt)
	}
}

// genAnkyloSFX makes its voice: a low, nasal honk through the long passages
// of its snout; a deep huffing grunt; a bleat when hit; a long sinking moan
// as it dies; and the whoosh of its club.
func genAnkyloSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0xa4c1, 0x10))
	// The honk, twice: "hronk-hronk", nasal and buzzing.
	for _, k := range []float64{1, 0.92, 1.07} {
		x := make([]float64, samples(0.95))
		for i := range 2 {
			kk := k * (1 - 0.06*float64(i))
			addAt(x, beastVoice(r, voiceSpec{
				dur: 0.34, pitch: glide(pt{0, 92 * kk}, pt{0.25, 118 * kk}, pt{1, 84 * kk}), amp: swell(0.06, 0.45),
				rough: 0.45, rattle: 24, formants: [2]float64{330 * kk, 1900 * kk}, breath: 0.25, sub: 0.45, harmonics: 18,
			}), 0.05+0.4*float64(i), 1-0.2*float64(i))
		}
		add("ankylo_honk", 0.85, echo(saturate(x, 1.5), pt{0.14, 0.2}, pt{0.3, 0.08}))
	}
	// The grunt: a deep huff through the nose, rough, short.
	for _, k := range []float64{1, 0.9} {
		x := make([]float64, samples(0.6))
		addAt(x, noiseBurst(r, 0.25, bandpass, 420*k, 0.9, 0.02, 0.12), 0, 0.45) // the breath out
		addAt(x, beastVoice(r, voiceSpec{
			dur: 0.45, pitch: glide(pt{0, 74 * k}, pt{0.3, 82 * k}, pt{1, 58 * k}), amp: swell(0.04, 0.4),
			rough: 0.85, rattle: 16, formants: [2]float64{280 * k, 820 * k}, breath: 0.55, sub: 0.7, harmonics: 16,
		}), 0.04, 1)
		add("ankylo_grunt", 0.9, saturate(x, 1.8))
	}
	// Hit: a sharp nasal bleat that cracks.
	for _, k := range []float64{1, 1.08, 0.94} {
		x := beastVoice(r, voiceSpec{
			dur: 0.55, pitch: glide(pt{0, 120 * k}, pt{0.18, 196 * k}, pt{1, 100 * k}), amp: swell(0.04, 0.5),
			rough: 0.65, rattle: 28, formants: [2]float64{460 * k, 2100 * k}, breath: 0.35, sub: 0.35, harmonics: 16,
		})
		add("ankylo_hurt", 0.9, saturate(x, 1.9))
	}
	// Dying: a long nasal moan that sinks and runs out of breath.
	for _, k := range []float64{1, 0.9} {
		x := make([]float64, samples(2.4))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 2.0, pitch: glide(pt{0, 112 * k}, pt{0.2, 124 * k}, pt{0.7, 70 * k}, pt{1, 48 * k}),
			amp:   func(u float64) float64 { return swell(0.06, 0.5)(u) * (1 - 0.5*u) },
			rough: 0.55, rattle: 19, formants: [2]float64{340 * k, 1700 * k}, breath: 0.6, sub: 0.6, harmonics: 16,
		}), 0, 1)
		addAt(x, noiseBurst(r, 0.8, lowpass, 700, 0.7, 0.2, 0.3), 1.5, 0.35) // the last breath
		add("ankylo_death", 0.9, echo(saturate(x, 1.4), pt{0.16, 0.22}, pt{0.38, 0.1}))
	}
	{ // The club through the air: a heavy, low whoosh, rising as it comes round.
		flt := newFilter(bandpass, 250, 1)
		x := render(0.42, func(t float64, _ int) float64 {
			flt.set(bandpass, 220+1300*t/0.42, 1.3)
			return flt.do(r.Float64()*2-1) * math.Pow(math.Sin(math.Pi*t/0.42), 1.5)
		})
		add("club_swing", 1, x)
	}
}

// --- its plates ------------------------------------------------------------------

var (
	osteoPlate = mkRamp(0x1a1b1c, 0x2b2c28, 0x3c3e33, 0x535444, 0x6c6c58)
	osteoBone  = mkRamp(0x6a5640, 0x8f7a5c, 0xb59f7c, 0xd4c29e, 0xeadcbc)
)

func init() {
	itemInfo[sim.ItemOsteoderm] = "Tấm xương bọc lưng khủng long bọc giáp, cứng như đá. Vật liệu quý, để dành cho đồ sau này."
	regIcon(sim.ItemOsteoderm, icOsteoderm)
}

// ankyloIcons draws the bag icon of the plates for the old 16 x 16 art (a
// drop on the ground uses it).
func ankyloIcons(icons *[sim.NumItems]*ebiten.Image) {
	c := newCanvas(16, 16)
	c.ellipse(8, 8.5, 6.2, 4.8, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx*0.8, ny*0.8)
		switch {
		case nx*nx+ny*ny > 0.7 && nx+ny < 0:
			return osteoBone.pick(l+0.15, x, y) // the pale rim
		case math.Abs(float64(y)-8.2+float64(x-8)*0.2) < 0.7:
			return osteoPlate.pick(l+0.25, x, y) // the keel
		}
		return osteoPlate.pick(l, x, y)
	})
	c.outline(0.8)
	icons[sim.ItemOsteoderm] = c.image()
}

// icOsteoderm is an osteoderm: an oval plate of bone, domed, with a keel
// along it, a pale rim of bare bone and a pitted face.
func icOsteoderm(p *icv) {
	a := mkAxis(3.2, 10.8, 12.8, 5.6)
	plate := iEll(8, 8.2, 6.6, 4.6, math.Atan2(5.6-10.8, 12.8-3.2))
	p.fill(plate, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		_ = u
		l := bulge(f, 1.6) - 0.04
		// The keel: a ridge along the plate, lit on its upper face.
		if k := math.Abs(v + 0.2); k < 0.9 {
			if v < -0.2 {
				l += 0.22 * (1 - k/0.9)
			} else {
				l -= 0.14 * (1 - k/0.9)
			}
		}
		if fbm2(f.x*1.7, f.y*1.7, 211, 2) < 0.3 {
			l -= 0.12 // pits in the bone
		}
		if -f.d < 0.75 && toward(f.gx, f.gy) > 0.1 {
			return pick(osteoBone, 0.55+0.35*toward(f.gx, f.gy)) // the rim of bare bone
		}
		return pick(osteoPlate, l)
	})
}
