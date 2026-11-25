package client

import "math"

// The rigs of the drawn dinosaurs (see dinorig.go), measured on the
// drawings in dinos/, in their pixels, and how each pose moves them. Pose
// numbers are the species' pose constants; anything a table leaves out is
// drawn as it stands.

// headUp is the side of the neck the head is on, as a rigPart's dir: up and
// forward at the given slope (radians above the horizontal).
func headUp(slope float64) float64 { return -slope }

// walkGait is when each leg of a four-legged rig steps, its legs listed
// near hind, far hind, far fore, near fore: one foot at a time, each fore
// foot a quarter of a stride after the hind foot on its side.
var walkGait = []float64{0, 0.5, 0.75, 0.25}

var dinoRigs = map[string]dinoRig{
	"trike": {
		file: "trike", stride: 2,
		// the near hind leg, the far hind, the far fore, the near fore: each pair steps in turn
		legs: []rigLimb{{15, 25, 43}, {26, 34, 43}, {35, 43, 43}, {44, 53, 43}},
		head: rigPart{pivot: rigPt{44.6, 31.5}, dir: headUp(0.75), maxY: 56},
		jaw:  [2]rigPt{{62, 39.5}, {71.5, 41.6}},
		gait: walkGait, far: 0b0110,
		tail: rigPart{pivot: rigPt{15, 31}, dir: math.Pi, maxY: 43},
	},
	"toro": {
		file: "toro", stride: 2,
		legs: []rigLimb{{16, 27, 49}, {28, 37, 49}, {38, 47, 49}, {48, 58, 49}},
		head: rigPart{pivot: rigPt{46.5, 35.5}, dir: headUp(0.75), maxY: 62},
		jaw:  [2]rigPt{{67.6, 44.2}, {77.9, 46.5}},
		gait: walkGait, far: 0b0110,
		tail: rigPart{pivot: rigPt{16, 36}, dir: math.Pi, maxY: 49},
	},
	"stego": {
		file: "stego", stride: 2,
		legs: []rigLimb{{35, 45, 46}, {46, 54, 46}, {55, 61, 46}, {62, 69, 46}},
		head: rigPart{pivot: rigPt{73, 37.5}, dir: headUp(0.1), maxY: 56},
		jaw:  [2]rigPt{{82, 42.5}, {89.5, 43}},
		tail: rigPart{pivot: rigPt{17.5, 30}, dir: math.Pi, maxY: 44}, // the bare end of it, with the spikes
		gait: walkGait, far: 0b0110,
	},
	"brachio": {
		file: "brachio", stride: 3,
		legs: []rigLimb{{22, 35, 84}, {36, 45, 84}, {50, 59, 84}, {60, 71, 84}},
		head: rigPart{pivot: rigPt{83.5, 25.5}, dir: headUp(0.98), maxY: 60},
		neck: rigPart{pivot: rigPt{72, 44}, dir: headUp(0.98), maxY: 80},
		jaw:  [2]rigPt{{99, 12.5}, {107, 13.5}},
		gait: walkGait, far: 0b0110,
		tail: rigPart{pivot: rigPt{22, 66}, dir: math.Pi, maxY: 92},
	},
	"dakota": {
		file: "dakota", stride: 3,
		legs:  []rigLimb{{22, 31, 23}, {12, 21, 23}}, // the near leg, then the far one behind it
		head:  rigPart{pivot: rigPt{31, 10}, dir: headUp(0.5), maxY: 14},
		jaw:   [2]rigPt{{40.5, 8}, {48, 8.4}},
		teeth: true,
		far:   0b10,
		tail:  rigPart{pivot: rigPt{14, 10}, dir: math.Pi, maxY: 18},
	},
	"achero": {
		file: "achero", stride: 2.5,
		legs:  []rigLimb{{17, 24, 17}, {10, 16, 17}},
		head:  rigPart{pivot: rigPt{23.6, 6.8}, dir: headUp(0.5), maxY: 10},
		jaw:   [2]rigPt{{31, 5}, {36.5, 5.2}},
		teeth: true,
		far:   0b10,
		tail:  rigPart{pivot: rigPt{11, 8}, dir: math.Pi, maxY: 13},
	},
	"troodon": {
		file: "troodon", stride: 3,
		legs:  []rigLimb{{23, 32, 22}, {13, 22, 22}}, // the near leg, then the far one behind it
		head:  rigPart{pivot: rigPt{34.5, 10.5}, dir: headUp(0.5), maxY: 13},
		jaw:   [2]rigPt{{40.5, 9}, {48.5, 9.4}},
		tail:  rigPart{pivot: rigPt{17, 14.5}, dir: math.Pi, maxY: 19},
		teeth: true,
		far:   0b10,
	},
	"rex": {
		file: "rex", stride: 4,
		legs:  []rigLimb{{45, 64, 62}, {27, 44, 60}}, // the near leg, then the far one behind it
		head:  rigPart{pivot: rigPt{70, 39}, dir: headUp(math.Pi / 4), maxY: 56},
		jaw:   [2]rigPt{{84, 33}, {103, 33}},
		teeth: true,
		far:   0b10,
		tail:  rigPart{pivot: rigPt{38, 39}, dir: math.Pi, maxY: 50},
	},
	"ornitho": {
		file: "ornitho", stride: 3,
		legs: []rigLimb{{23, 31, 31}, {14, 22, 31}},
		head: rigPart{pivot: rigPt{35.2, 3.6}, dir: headUp(0.3), maxY: 13},
		neck: rigPart{pivot: rigPt{31, 18}, dir: headUp(0.9), maxY: 20},
		jaw:  [2]rigPt{{41, 7}, {46.5, 7.2}},
		far:  0b10,
		tail: rigPart{pivot: rigPt{13, 19}, dir: math.Pi, maxY: 25},
	},
	"thesc": {
		file: "thesc", stride: 2.5,
		legs: []rigLimb{{23, 31, 25}, {13, 22, 25}},
		head: rigPart{pivot: rigPt{33, 10.5}, dir: headUp(0.6), maxY: 15},
		jaw:  [2]rigPt{{41, 11}, {47, 11.4}},
		far:  0b10,
		tail: rigPart{pivot: rigPt{12, 13}, dir: math.Pi, maxY: 20},
	},
	"anzu": {
		file: "anzu", stride: 3,
		legs: []rigLimb{{23, 36, 36}, {13, 22, 36}}, // the near leg, then the far one behind it
		head: rigPart{pivot: rigPt{30.5, 24.5}, dir: headUp(1.1), maxY: 26},
		jaw:  [2]rigPt{{39.5, 11.5}, {46, 11.8}},
		far:  0b10,
		tail: rigPart{pivot: rigPt{11, 24}, dir: math.Pi, maxY: 33},
	},
	"quetzal": {
		file: "quetzal", stride: 3,
		// The near hind leg, the near wing it walks on, the far wing, the far hind leg.
		legs: []rigLimb{{30, 40, 86}, {49, 61, 86}, {62, 71, 86}, {22, 29, 86}},
		head: rigPart{pivot: rigPt{63, 16}, dir: headUp(math.Pi / 4), maxY: 40},
		neck: rigPart{pivot: rigPt{50, 55}, dir: headUp(1.0), maxY: 60},
		jaw:  [2]rigPt{{71.4, 19.6}, {100.5, 24.7}},
		gait: []float64{0, 0.25, 0.75, 0.5}, far: 0b1100,
	},
	"plesio": {
		file: "plesio", ax: 23, ay: 45,
		head:  rigPart{pivot: rigPt{34, 40}, dir: headUp(0.9), maxY: 45},
		jaw:   [2]rigPt{{51.5, 9.5}, {60.5, 9.6}},
		teeth: true,
	},
	"pteroPerch": {file: "ptero_perch", ax: 28, ay: 38},
	// The fliers are seen from above, heading right: the far wing up the
	// screen, the near one down it, each swinging about its shoulder.
	"pteroFly": {
		file: "ptero_fly", ax: 18, ay: 28,
		wings: [2]rigPart{
			{pivot: rigPt{18.5, 22.2}, dir: -math.Pi / 2, maxY: 999}, // the far wing
			{pivot: rigPt{18.5, 33.8}, dir: math.Pi / 2, maxY: 999},  // the near one
		},
	},
	"quetzalFly": {
		file: "quetzal_fly", ax: 29, ay: 46,
		wings: [2]rigPart{
			{pivot: rigPt{30.2, 36.1}, dir: -math.Pi / 2, maxY: 999},
			{pivot: rigPt{30.2, 55.9}, dir: math.Pi / 2, maxY: 999},
		},
	},
}

// dinoPoses is how each pose of a species moves its rig.
var dinoPoses = map[string]map[int]rigPose{
	"trike": {
		trikeWalkB:  {step: 1},
		trikeCharge: {step: 1, head: 0.35},
		trikeGore:   {head: -0.4, open: 0.35}, // also its call: head up, beak open
		trikeEatA:   {head: 0.55},
		trikeEatB:   {head: 0.55, open: 0.3},
	},
	"stego": {
		stegoWalkB: {step: 1},
		stegoSwipe: {tail: 0.55},
		stegoEatA:  {head: 0.5},
		stegoEatB:  {head: 0.5, open: 0.2},
	},
	"brachio": {
		brachWalkB: {step: 1},
		brachEatA:  {neck: 0.8, head: 0.7},
		brachEatB:  {neck: 0.8, head: 0.7, open: 0.3},
	},
	"raptor": {
		raptorRunA:   {step: 1},
		raptorRunB:   {step: -1},
		raptorPounce: {step: 1, tilt: -0.3, open: 0.45, lift: 3},
		raptorCower:  {squash: 0.75, head: 0.4},
		raptorEatA:   {head: 0.7},
		raptorEatB:   {head: 0.7, open: 0.4},
	},
	"troodon": {
		troodonWalkA: {step: 0.8},
		troodonWalkB: {step: -0.8},
		troodonRunA:  {step: 1.5, tilt: 0.08, head: 0.1, tail: 0.06},
		troodonRunB:  {step: -1.3, tilt: 0.08, head: 0.1},
		troodonStalk: {squash: 0.7, head: 0.4, tail: 0.14, step: 0.6},
		troodonHiss:  {squash: 0.7, head: 0.4, tail: 0.14, step: 0.6, open: 0.3},
		troodonLeap:  {step: 1.2, tilt: -0.22, head: 0.15, open: 0.45, tail: -0.1},
		troodonBite:  {step: 1.2, tilt: 0.16, head: 0.3, open: 0.5, tail: 0.12},
		troodonPeer:  {head: -0.3},
		troodonHurt:  {squash: 0.9, head: -0.45, open: 0.35, tail: -0.15},
	},
	"rex": {
		rexWalkB: {step: 1},
		rexRoar:  {head: -0.5, open: 0.45},
		rexBite:  {head: 0.25, open: 0.4},
		rexEatA:  {head: 0.6, open: 0.1},
		rexEatB:  {head: 0.6, open: 0.4},
	},
	"ornitho": {
		ornithoRunA: {step: 1},
		ornithoRunB: {step: -1},
		ornithoRest: {squash: 0.85, neck: 0.25},
		ornithoEatA: {neck: 1.5, head: -0.7}, // the neck swings down, the head levels out again
		ornithoEatB: {neck: 1.5, head: -0.7, open: 0.3},
	},
	"thesc": {
		thescWalkA: {step: 1},
		thescWalkB: {step: -1},
		thescEatA:  {head: 0.35, tilt: 0.25}, // short-necked: it leans its whole body down
		thescEatB:  {head: 0.35, tilt: 0.25, open: 0.3},
	},
	"anzu": {
		anzuWalkA:   {step: 1},
		anzuWalkB:   {step: -1},
		anzuDisplay: {head: -0.3, open: 0.4}, // also its call
		anzuKick:    {step: 2},
		anzuEatA:    {head: 1.45},
		anzuEatB:    {head: 1.45, open: 0.3},
	},
	"quetzal": {
		quetzalWalkB: {step: 1},
		quetzalStab:  {neck: 0.4, head: 0.3, open: 0.2},
		quetzalEatA:  {neck: 1.35, head: 0.1},
		quetzalEatB:  {neck: 1.35, head: 0.1, open: 0.25},
	},
	"plesio": {
		plesioStrike: {head: 0.7, open: 0.4},
	},
	// A wingbeat seen from above: the wings shorten as they rise and fall,
	// and sweep back on the way down; + wings is forward.
	"pteroFly": {
		pteroUp:   {squash: 0.7, wings: 0.1},
		pteroDown: {squash: 0.86, wings: -0.2},
		pteroDive: {tilt: 0.45, wings: -0.5},
	},
}

// poseFamily is whose poses (in dinoPoses and dinoWinds) species moves by.
func poseFamily(species string) string {
	switch species {
	case "toro":
		return "trike"
	case "quetzalFly":
		return "pteroFly"
	case "dakota", "achero":
		return "raptor"
	}
	return species
}

// drawnDino is pose of species drawn from its drawing, or ok false if the
// drawing is missing.
func drawnDino(species string, pose int) (*canvas, float64, float64, bool) {
	r, ok := dinoRigs[species]
	if !ok {
		return nil, 0, 0, false
	}
	return rigCanvas(r, dinoPoses[poseFamily(species)][pose])
}
