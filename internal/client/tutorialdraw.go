package client

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The guide's tracker: a small plate under the vitals and the status chips
// (and under the long line of a status, while that shows) with the goal
// now, the key to use, and a line on how. A goal done is ticked for a
// moment, then the next slides in; the last says the guide is over and
// fades away.

const (
	trackW      = 204 // the plate's width
	trackTitle  = 0.85
	trackHint   = 0.75
	trackPad    = 7
	trackTextX  = 26 // the title's left, from the plate's
	trackLines  = 2  // the most lines of the title, and of the hint
	trackMaxH   = trackPad + trackLines*13.6 + 3 + trackLines*12 + 9
	trackGap    = 5 // under the chips
	trackTick   = 1400 * time.Millisecond
	trackEnd    = 5 * time.Second
	trackEndOut = 1 * time.Second
)

// trackTop is the top of the tracker under nv vitals and nc chips, and the
// long line of a status h tall (0 if none).
func trackTop(nv, nc int, toastH float64) float64 {
	y := chipsBottom(nv, nc) + trackGap
	if toastH > 0 {
		y += 4 + toastH + trackGap
	}
	return y
}

// trackLay is how a goal's text lies on the plate.
type trackLay struct {
	title, hint []string
	keyW        float64 // the key's cap, 0 for none
	h           float64
}

// trackLayout wraps a goal's text to the plate.
func trackLayout(s tutStep) trackLay {
	l := trackLay{}
	l.title = wrapAt(s.title, trackW-trackTextX-trackPad, trackTitle)
	hintX := 0.0
	if s.key != "" {
		l.keyW = textWidth(s.key, trackHint) + 9
		hintX = l.keyW + 4
	}
	l.hint = wrapAt(s.hint, trackW-trackPad-trackTextX-hintX, trackHint)
	l.h = math.Round(trackPad + float64(len(l.title))*lineUI*trackTitle + 3 + max(13, float64(len(l.hint))*lineUI*trackHint) + 9)
	return l
}

// toastHeight is the height of the long line of the status now shouting
// under the chips, 0 if none.
func (g *Game) toastHeight(list []status) float64 {
	var shout *status
	for i := range list {
		s := &list[i]
		if s.long != "" && g.frame-g.statusAt[s.kind] < statusShout && (shout == nil || g.statusAt[s.kind] > g.statusAt[shout.kind]) {
			shout = s
		}
	}
	if shout == nil {
		return 0
	}
	lines := wrapAt(shout.long, statusToastW-34, statusToastScale)
	return math.Round(float64(len(lines))*lineUI*statusToastScale + 6)
}

// drawTutorial draws the tracker under nv vitals and the status chips list.
func (g *Game) drawTutorial(dst *uiDst, nv int, list []status) {
	t := g.tut
	if !g.tutorialOn() || t == nil || g.curr.Self.Dead || g.overlay != overlayNone {
		return
	}
	now := time.Now()
	y := trackTop(nv, len(list), g.toastHeight(list))
	x := float64(vitalX)

	// What to show: a goal just done (ticked), the end, or the goal now.
	var (
		s     tutStep
		done  bool
		alpha = 1.0
		in    = 1.0
		prog  = float64(t.step) / float64(len(tutSteps))
	)
	switch {
	case t.flash >= 0 && now.Sub(t.flashAt) < trackTick && t.step < len(tutSteps):
		s, done = tutSteps[t.flash], true
	case t.over && !t.finishedAt.IsZero():
		since := now.Sub(t.finishedAt)
		if t.flash >= 0 && since < trackTick {
			s, done = tutSteps[t.flash], true
			break
		}
		if since > trackEnd+trackEndOut {
			return
		}
		s, done = tutStep{title: "Hướng dẫn xong!", hint: "F1 xem lại các phím. Chúc bạn sống sót!"}, true
		alpha = min(1, (trackEnd+trackEndOut-since).Seconds()/trackEndOut.Seconds())
		prog = 1
	case t.over:
		return
	default:
		s = tutSteps[t.step]
		if t.flash >= 0 {
			in = smooth(min(1, (now.Sub(t.flashAt)-trackTick).Seconds()/0.25))
		}
	}
	lay := trackLayout(s)
	x -= (1 - in) * 8
	a := alpha * (0.35 + 0.65*in)
	plateA(dst, x, y, trackW, lay.h, 9, 0.9*a)

	// The ring or the tick.
	cx, cy := x+13, y+trackPad+lineUI*trackTitle/2
	if done {
		fillCircle(dst, float32(cx), float32(cy), 6.5, fade(color.RGBA{96, 176, 84, 255}, uint8(255*a)), true)
		w := float32(1.6 * dst.k)
		c := fade(colWhite, uint8(255*a))
		fx, fy := func(v float64) float32 { return float32(dst.X(v)) }, func(v float64) float32 { return float32(dst.Y(v)) }
		vector.StrokeLine(dst.img, fx(cx-3), fy(cy), fx(cx-0.8), fy(cy+2.4), w, c, true)
		vector.StrokeLine(dst.img, fx(cx-0.8), fy(cy+2.4), fx(cx+3.2), fy(cy-2.6), w, c, true)
	} else {
		strokeCircle(dst, float32(cx), float32(cy), 6, 1.4, fade(colKey, uint8(230*a)), true)
	}

	col := colBoneText
	if done {
		col = mix(colGood, colWhite, 0.2)
	}
	ty := y + trackPad
	for _, l := range lay.title {
		uiText(dst, l, x+trackTextX, ty, fade(col, uint8(255*a)), text.AlignStart, trackTitle, fxShadow)
		ty += lineUI * trackTitle
	}
	ty += 3
	hx := x + trackTextX
	if lay.keyW > 0 {
		plateA(dst, hx, ty, lay.keyW, 13, 5, 0.9*a)
		uiText(dst, s.key, hx+lay.keyW/2, ty+(13-lineUI*trackHint)/2, fade(colKey, uint8(255*a)), text.AlignCenter, trackHint, fxShadow)
		hx += lay.keyW + 4
	}
	for _, l := range lay.hint {
		uiText(dst, l, hx, ty+(13-lineUI*trackHint)/2, fade(colMuted, uint8(235*a)), text.AlignStart, trackHint, fxShadow)
		ty += lineUI * trackHint
	}

	// The progress, a thin line along the foot.
	bx, bw := x+trackPad, float64(trackW-2*trackPad)
	fillRect(dst, float32(bx), float32(y+lay.h-6), float32(bw), 2, fade(color.RGBA{0, 0, 0, 255}, uint8(110*a)), false)
	if prog > 0 {
		fillRect(dst, float32(bx), float32(y+lay.h-6), float32(bw*prog), 2, fade(colKey, uint8(220*a)), false)
	}
}
