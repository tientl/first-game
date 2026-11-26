package client

// critKinds is every code-drawn animal, by name, with its density-1
// drawing; the HD drawings are hung on in the init of the file that draws
// them (critsmall.go, critcroc.go, critbeast.go, critcave.go).
var critKinds = func() map[string]*critKind {
	m := map[string]*critKind{}
	reg := func(name string, poses int, old func(int) (*canvas, float64, float64)) {
		m[name] = &critKind{poses: poses, old: old, anim: map[int]int{}}
	}
	reg("rabbit", rabbitPoses, rabbitCanvas)
	reg("snake", snakeSprites, snakeCanvas)
	reg("crab", crabPoses, crabCanvas)
	reg("croc", crocPoses, muggerCanvas)
	reg("brachy", crocPoses, func(p int) (*canvas, float64, float64) { return crocCanvasOf(p, brachyLook) })
	reg("champso", crocPoses, func(p int) (*canvas, float64, float64) { return crocCanvasOf(p, champsoLook) })
	reg("worm", wormFrames, wormCanvas)
	reg("fish", fishFrames, fishCanvas)
	for v := range fishLooks {
		reg(fishKind(v), fishFrames, func(p int) (*canvas, float64, float64) { return fishLookCanvas(v, p) })
	}
	reg("mammoth", mammothPoses, mammothCanvas)
	reg("sabre", sabrePoses, sabreCanvas)
	reg("bear", bearPoses, bearCanvas)
	reg("spider", spiderPoses, spiderCanvas)
	reg("tribe", tribePoses, tribesmanCanvas)
	reg("monkey", monkeyPoses, monkeyCanvas)
	reg("scorpion", scorpionPoses, scorpionCanvas)
	return m
}()
