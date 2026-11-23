package client

import "math"

// The rigs of the drawn dinosaurs (see dinorig.go), measured on the
// drawings in dinos/, in their pixels, and how each pose moves them. Pose
// numbers are the species' pose constants; anything a table leaves out is
// drawn as it stands.

// headUp is the side of the neck the head is on, as a rigPart's dir: up and
// forward at the given slope (radians above the horizontal).
func headUp(slope float64) float64 { return -slope }

var dinoRigs = map[string]dinoRig{
	"trike": {
		file: "trike", stride: 2,
		legs: []rigLimb{{19, 30, 42}, {43, 57, 42}},
		head: rigPart{pivot: rigPt{47, 31}, dir: headUp(0.6), maxY: 44},
		jaw:  [2]rigPt{{62, 31}, {72, 32}},
	},
	"toro": {
		file: "toro", stride: 2,
		legs: []rigLimb{{20, 32, 44}, {46, 63, 44}},
		head: rigPart{pivot: rigPt{50, 34}, dir: headUp(0.6), maxY: 45},
		jaw:  [2]rigPt{{66, 34}, {80, 33}},
	},
	"stego": {
		file: "stego", stride: 2,
		legs: []rigLimb{{26, 38, 43}, {48, 64, 43}},
		head: rigPart{pivot: rigPt{60, 38}, dir: headUp(0.25), maxY: 45},
		jaw:  [2]rigPt{{72, 41}, {80, 41}},
		tail: rigPart{pivot: rigPt{16, 27}, dir: math.Pi, maxY: 34},
	},
	"brachio": {
		file: "brachio", stride: 2,
		legs: []rigLimb{{23, 44, 84}, {59, 80, 84}},
		head: rigPart{pivot: rigPt{85, 30}, dir: headUp(1.1), maxY: 40},
		neck: rigPart{pivot: rigPt{74, 56}, dir: headUp(1.0), maxY: 64},
		jaw:  [2]rigPt{{100, 11}, {107, 12}},
	},
	"dakota": {
		file: "dakota", stride: 2,
		legs:  []rigLimb{{16, 27, 22}},
		head:  rigPart{pivot: rigPt{25, 12}, dir: headUp(0.4), maxY: 16},
		jaw:   [2]rigPt{{33, 12}, {40, 12}},
		teeth: true,
	},
	"achero": {
		file: "achero", stride: 1.5,
		legs:  []rigLimb{{12, 22, 16}},
		head:  rigPart{pivot: rigPt{19, 9}, dir: headUp(0.4), maxY: 11},
		jaw:   [2]rigPt{{25, 9}, {31, 8}},
		teeth: true,
	},
	"rex": {
		file: "rex", stride: 4,
		legs:  []rigLimb{{45, 64, 62}, {27, 44, 60}}, // the near leg, then the far one behind it
		head:  rigPart{pivot: rigPt{70, 39}, dir: headUp(math.Pi / 4), maxY: 56},
		jaw:   [2]rigPt{{84, 33}, {103, 33}},
		teeth: true,
	},
	"ornitho": {
		file: "ornitho", stride: 2,
		legs: []rigLimb{{19, 29, 33}},
		head: rigPart{pivot: rigPt{31, 17}, dir: headUp(1.3), maxY: 20},
		jaw:  [2]rigPt{{38, 7}, {46, 7}},
	},
	"thesc": {
		file: "thesc", stride: 2,
		legs: []rigLimb{{14, 29, 24}},
		head: rigPart{pivot: rigPt{33, 13}, dir: headUp(0.3), maxY: 17},
		jaw:  [2]rigPt{{40, 13}, {46, 13}},
	},
	"anzu": {
		file: "anzu", stride: 2,
		legs: []rigLimb{{17, 27, 36}},
		head: rigPart{pivot: rigPt{27, 20}, dir: headUp(1.2), maxY: 21},
		jaw:  [2]rigPt{{37, 9}, {41, 11}},
	},
	"quetzal": {
		file: "quetzal", stride: 2,
		legs: []rigLimb{{19, 30, 79}, {38, 50, 79}},
		head: rigPart{pivot: rigPt{44, 34}, dir: headUp(1.26), maxY: 40},
		neck: rigPart{pivot: rigPt{40, 53}, dir: headUp(1.2), maxY: 56},
		jaw:  [2]rigPt{{55, 20}, {80, 33}},
	},
	"plesio": {
		file: "plesio", ax: 21, ay: 42,
		head: rigPart{pivot: rigPt{31, 37}, dir: headUp(0.9), maxY: 42},
		jaw:  [2]rigPt{{50, 9}, {57, 9}},
	},
	"pteroPerch": {file: "ptero_perch", ax: 14, ay: 28},
	"pteroFly": {
		file: "ptero_fly", ax: 23, ay: 12,
		wings: [2]rigPart{
			{pivot: rigPt{21, 9}, dir: math.Pi, maxY: 99}, // the left wing
			{pivot: rigPt{27, 8}, dir: 0, maxY: 99},       // the right
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
		stegoEatA:  {head: 0.35},
		stegoEatB:  {head: 0.35, open: 0.2},
	},
	"brachio": {
		brachWalkB: {step: 1},
		brachEatA:  {neck: 1.5, head: 0.6},
		brachEatB:  {neck: 1.5, head: 0.6, open: 0.3},
	},
	"raptor": {
		raptorRunA:   {step: 1},
		raptorRunB:   {step: -1},
		raptorPounce: {step: 1, tilt: -0.3, open: 0.45, lift: 3},
		raptorCower:  {squash: 0.75, head: 0.4},
		raptorEatA:   {head: 0.7},
		raptorEatB:   {head: 0.7, open: 0.4},
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
		ornithoRest: {squash: 0.85, head: 0.2},
		ornithoEatA: {head: 1.1},
		ornithoEatB: {head: 1.1, open: 0.3},
	},
	"thesc": {
		thescWalkA: {step: 1},
		thescWalkB: {step: -1},
		thescEatA:  {head: 0.7},
		thescEatB:  {head: 0.7, open: 0.3},
	},
	"anzu": {
		anzuWalkA:   {step: 1},
		anzuWalkB:   {step: -1},
		anzuDisplay: {head: -0.3, open: 0.4}, // also its call
		anzuKick:    {step: 2},
		anzuEatA:    {head: 1.2},
		anzuEatB:    {head: 1.2, open: 0.3},
	},
	"quetzal": {
		quetzalWalkB: {step: 1},
		quetzalStab:  {neck: 0.4, head: 0.3, open: 0.2},
		quetzalEatA:  {neck: 0.8, head: 0.5},
		quetzalEatB:  {neck: 0.8, head: 0.5, open: 0.3},
	},
	"plesio": {
		plesioStrike: {head: 0.7, open: 0.4},
	},
	"pteroFly": {
		pteroUp:   {wings: 0.35},
		pteroDown: {wings: -0.35},
		pteroDive: {tilt: 0.45, wings: -0.2},
	},
}

// drawnDino is pose of species drawn from its drawing, or ok false if the
// drawing is missing.
func drawnDino(species string, pose int) (*canvas, float64, float64, bool) {
	r, ok := dinoRigs[species]
	if !ok {
		return nil, 0, 0, false
	}
	poses := dinoPoses[species]
	if species == "toro" {
		poses = dinoPoses["trike"]
	}
	if species == "dakota" || species == "achero" {
		poses = dinoPoses["raptor"]
	}
	return rigCanvas(r, poses[pose])
}
