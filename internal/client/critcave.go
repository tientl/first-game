package client

// The cave beasts and the tribe at HD density: the cave spider is in
// critspider.go, the giant scorpion in critscorp.go, the warrior in
// crittribe.go.

func init() {
	set := func(name string, hd func(pose, f int) (*canvas, float64, float64), anim map[int]int) {
		critKinds[name].hd, critKinds[name].anim = hd, anim
	}
	set("spider", spiderHD, map[int]int{spiderWalkA: cycFrames, spiderBite: blowFrames, spiderSpit: blowFrames})
	set("scorpion", scorpionHD, map[int]int{scorpionWalkA: cycFrames, scorpionPinch: blowFrames, scorpionSting: blowFrames})
	set("tribe", tribeHD, map[int]int{tribeWalkA: cycFrames, tribeWalkB: cycFrames, tribeThrust: blowFrames})
}
