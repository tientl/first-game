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
		legs: []rigLimb{
			hind(15, 25, 43, rigPt{20, 34}, rigPt{21, 44}, rigPt{20.5, 51}, rigPt{22.5, 55.5}),
			hind(26, 34, 43, rigPt{30, 35}, rigPt{30.5, 44}, rigPt{30, 49.5}, rigPt{32, 53}),
			fore(35, 43, 43, rigPt{39, 35}, rigPt{38.5, 44}, rigPt{39, 50}, rigPt{40.5, 53.5}),
			fore(44, 53, 43, rigPt{48.5, 34}, rigPt{48, 44}, rigPt{48.5, 51}, rigPt{50, 55.5}),
		},
		head: rigPart{pivot: rigPt{44.6, 31.5}, dir: headUp(0.75), maxY: 56},
		jaw:  [2]rigPt{{62, 39.5}, {71.5, 41.6}},
		gait: walkGait, far: 0b0110,
		tail: rigPart{pivot: rigPt{15, 31}, dir: math.Pi, maxY: 43},
	},
	"toro": {
		file: "toro", stride: 2,
		legs: []rigLimb{
			hind(16, 27, 49, rigPt{21.5, 39}, rigPt{22.5, 49.5}, rigPt{22, 56.5}, rigPt{24, 61.5}),
			hind(28, 37, 49, rigPt{32.5, 40}, rigPt{33, 49.5}, rigPt{32.5, 54.5}, rigPt{34.5, 58.5}),
			fore(38, 47, 49, rigPt{42, 40}, rigPt{41.5, 49.5}, rigPt{42, 55}, rigPt{43.5, 58.5}),
			fore(48, 58, 49, rigPt{53, 39}, rigPt{52.5, 49.5}, rigPt{53, 56.5}, rigPt{54.5, 61.5}),
		},
		head: rigPart{pivot: rigPt{46.5, 35.5}, dir: headUp(0.75), maxY: 62},
		jaw:  [2]rigPt{{67.6, 44.2}, {77.9, 46.5}},
		gait: walkGait, far: 0b0110,
		tail: rigPart{pivot: rigPt{16, 36}, dir: math.Pi, maxY: 49},
	},
	"stego": {
		file: "stego", stride: 2,
		legs: []rigLimb{
			hind(35, 45, 46, rigPt{40.5, 36}, rigPt{41.5, 45}, rigPt{41, 51}, rigPt{43, 55.5}),
			hind(46, 54, 46, rigPt{50, 37}, rigPt{50.5, 45}, rigPt{50, 50}, rigPt{52, 53.5}),
			fore(55, 61, 46, rigPt{58, 38}, rigPt{57.5, 45}, rigPt{58, 50}, rigPt{59.5, 53.5}),
			fore(62, 71, 46, rigPt{66, 37}, rigPt{65.5, 45}, rigPt{66, 51}, rigPt{67.5, 55.5}),
		},
		head: rigPart{pivot: rigPt{73, 37.5}, dir: headUp(0.1), maxY: 56},
		jaw:  [2]rigPt{{82, 42.5}, {89.5, 43}},
		tail: rigPart{pivot: rigPt{17.5, 30}, dir: math.Pi, maxY: 44}, // the bare end of it, with the spikes
		gait: walkGait, far: 0b0110,
	},
	"brachio": {
		file: "brachio", stride: 3,
		legs: []rigLimb{
			// the near legs take only their own flank as they swing, not the chest
			{x0: 21, x1: 35, top: 80, r: 7, j: []rigPt{{29, 72}, {30, 90}, {29.5, 102}, {32, 111.5}}},
			hind(36, 45, 81, rigPt{41, 74}, rigPt{41.5, 90}, rigPt{41, 100}, rigPt{43, 108.5}),
			fore(50, 59, 81, rigPt{55, 74}, rigPt{54.5, 90}, rigPt{55, 100}, rigPt{56.5, 108.5}),
			{x0: 60, x1: 73, top: 80, r: 7, j: []rigPt{{66, 72}, {65.5, 90}, {66, 102}, {67.5, 111.5}}, fore: true},
		},
		// The neck and the head bend about the top of the neck, so lowered they
		// fold over the throat and open no gap along the top.
		head: rigPart{pivot: rigPt{78.8, 25.1}, dir: headUp(0.98), maxY: 60},
		neck: rigPart{pivot: rigPt{65.5, 39.5}, dir: headUp(0.98), maxY: 80},
		jaw:  [2]rigPt{{99, 12.5}, {107, 13.5}},
		gait: walkGait, far: 0b0110,
		tail: rigPart{pivot: rigPt{22, 66}, dir: math.Pi, maxY: 92},
	},
	"dakota": {
		file: "dakota", stride: 3, runner: true,
		legs: []rigLimb{ // the near leg, then the far one behind it
			hind(22, 31, 23, rigPt{25, 19}, rigPt{27.5, 24}, rigPt{24.5, 28.5}, rigPt{25.5, 34}, rigPt{31.5, 35.5}),
			hind(12, 21, 23, rigPt{15, 19}, rigPt{17.5, 24}, rigPt{14.5, 28.5}, rigPt{15.5, 34}, rigPt{21.5, 35.5}),
		},
		head:  rigPart{pivot: rigPt{31, 10}, dir: headUp(0.5), maxY: 14},
		jaw:   [2]rigPt{{40.5, 8}, {48, 8.4}},
		teeth: true,
		far:   0b10,
		tail:  rigPart{pivot: rigPt{14, 10}, dir: math.Pi, maxY: 18},
	},
	"achero": {
		file: "achero", stride: 2.5, runner: true,
		legs: []rigLimb{
			hind(17, 24, 17, rigPt{20, 14}, rigPt{21.5, 18}, rigPt{19, 21}, rigPt{20, 25.5}, rigPt{24.5, 26}),
			hind(9, 16, 17, rigPt{12.5, 14}, rigPt{13.5, 18}, rigPt{11.5, 21}, rigPt{12.5, 25.5}, rigPt{16.5, 26}),
		},
		head:  rigPart{pivot: rigPt{23.6, 6.8}, dir: headUp(0.5), maxY: 10},
		jaw:   [2]rigPt{{31, 5}, {36.5, 5.2}},
		teeth: true,
		far:   0b10,
		tail:  rigPart{pivot: rigPt{11, 8}, dir: math.Pi, maxY: 13},
	},
	"troodon": {
		file: "troodon", stride: 3, runner: true,
		legs: []rigLimb{ // the near leg, then the far one behind it
			hind(23, 32, 22, rigPt{28, 19}, rigPt{29.5, 24}, rigPt{25.5, 28}, rigPt{26.5, 33.5}, rigPt{32, 34.5}),
			hind(13, 22, 22, rigPt{20, 19}, rigPt{20, 23.5}, rigPt{15.5, 27.5}, rigPt{16.5, 33.5}, rigPt{21.5, 34.5}),
		},
		head:  rigPart{pivot: rigPt{34.5, 10.5}, dir: headUp(0.5), maxY: 13},
		jaw:   [2]rigPt{{40.5, 9}, {48.5, 9.4}},
		tail:  rigPart{pivot: rigPt{17, 14.5}, dir: math.Pi, maxY: 19},
		teeth: true,
		far:   0b10,
	},
	"rex": {
		file: "rex", stride: 4,
		legs: []rigLimb{ // the near leg, then the far one behind it
			{x0: 45, x1: 64, top: 56, r: 9, j: []rigPt{{53, 47}, {60, 65.5}, {49.5, 72}, {52, 83}, {65, 85}}},
			{x0: 27, x1: 44, top: 56, r: 7, j: []rigPt{{40, 48}, {41, 60}, {33, 71}, {35, 83}, {45, 85}}},
		},
		head:       rigPart{pivot: rigPt{70, 39}, dir: headUp(math.Pi / 4), maxY: 56},
		jaw:        [2]rigPt{{84, 34}, {103, 34}}, // the upper teeth are above this line, in the drawing
		drawnTeeth: true,
		teeth:      true,
		far:        0b10,
		tail:       rigPart{pivot: rigPt{38, 39}, dir: math.Pi, maxY: 50},
	},
	"ornitho": {
		file: "ornitho", stride: 3, runner: true,
		legs: []rigLimb{
			hind(23, 32, 31, rigPt{27, 26}, rigPt{28.5, 32.5}, rigPt{25, 37.5}, rigPt{25.5, 44.5}, rigPt{32, 46}),
			hind(14, 22, 31, rigPt{18, 26}, rigPt{19.5, 32.5}, rigPt{16, 37.5}, rigPt{16.5, 44.5}, rigPt{22.5, 46}),
		},
		head: rigPart{pivot: rigPt{35.2, 3.6}, dir: headUp(0.3), maxY: 13},
		neck: rigPart{pivot: rigPt{31, 18}, dir: headUp(0.9), maxY: 20},
		jaw:  [2]rigPt{{41, 7}, {46.5, 7.2}},
		far:  0b10,
		tail: rigPart{pivot: rigPt{13, 19}, dir: math.Pi, maxY: 25},
	},
	"thesc": {
		file: "thesc", stride: 2.5,
		legs: []rigLimb{
			hind(23, 31, 25, rigPt{27, 20}, rigPt{28.5, 25}, rigPt{25, 28}, rigPt{25.5, 31.5}, rigPt{31, 32.5}),
			hind(13, 22, 25, rigPt{17.5, 20}, rigPt{18.5, 25}, rigPt{15, 28}, rigPt{15.5, 31.5}, rigPt{21, 32.5}),
		},
		head: rigPart{pivot: rigPt{33, 10.5}, dir: headUp(0.6), maxY: 15},
		jaw:  [2]rigPt{{41, 11}, {47, 11.4}},
		far:  0b10,
		tail: rigPart{pivot: rigPt{12, 13}, dir: math.Pi, maxY: 20},
	},
	"anzu": {
		file: "anzu", stride: 3,
		legs: []rigLimb{ // the near leg, then the far one behind it
			hind(23, 36, 36, rigPt{27, 31}, rigPt{27.5, 40}, rigPt{25.5, 46.5}, rigPt{27, 52}, rigPt{35.5, 54}),
			hind(13, 22, 36, rigPt{18, 31}, rigPt{17.5, 40}, rigPt{16, 45}, rigPt{17.5, 52}, rigPt{22.5, 54}),
		},
		head: rigPart{pivot: rigPt{30.5, 24.5}, dir: headUp(1.1), maxY: 26},
		jaw:  [2]rigPt{{39.5, 11.5}, {46, 11.8}},
		far:  0b10,
		tail: rigPart{pivot: rigPt{11, 24}, dir: math.Pi, maxY: 33},
	},
	"quetzal": {
		file: "quetzal", stride: 3,
		// The near hind leg, the near wing it walks on, the far wing, the far hind leg.
		legs: []rigLimb{{x0: 30, x1: 40, top: 86}, {x0: 49, x1: 61, top: 86}, {x0: 62, x1: 71, top: 86}, {x0: 22, x1: 29, top: 86}},
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
