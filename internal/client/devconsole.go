package client

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The dev console: only in a game started with -dev (SetDev). The ` key
// (or F12) opens a line at the bottom of the screen; what is typed there
// goes to the server as a ClientDev, which runs it only in dev mode, and
// the answer is shown above the line.

const (
	devKeepLines = 200 // output lines remembered
	devShowLines = 14  // output lines shown
	devKeepHist  = 50  // commands remembered for Up and Down
	devLineH     = 16
	devMaxInput  = 160 // runes
)

var (
	colDevTag  = color.RGBA{255, 120, 60, 255}
	colDevIn   = color.RGBA{255, 236, 150, 255}
	colDevBack = color.RGBA{8, 10, 16, 238}
	colDevErr  = color.RGBA{255, 130, 120, 255}
)

type devLine struct {
	s   string
	col color.Color
}

// devConsole is the console's state.
type devConsole struct {
	open   bool
	input  []rune
	hist   []string
	histAt int // index into hist while browsing with Up and Down; len(hist) when not
	out    []devLine
	runes  []rune // scratch for typed characters
}

// SetDev turns the dev console and the DEV tag on. Only cmd/game's -dev
// calls it.
func (g *Game) SetDev(on bool) {
	if !on {
		g.dev = nil
		return
	}
	g.dev = &devConsole{}
	g.dev.say("Chế độ dev. Gõ help để xem các lệnh; ` hoặc Esc để đóng.", colMuted)
}

// ShowDevConsole opens the console with what was typed and answered, for
// cmd/shot.
func (g *Game) ShowDevConsole(typed []string, answers [][]string) {
	if g.dev == nil {
		g.SetDev(true)
	}
	g.dev.open = true
	for i, t := range typed {
		g.dev.echo(t)
		if i < len(answers) {
			for _, l := range answers[i] {
				g.dev.say(l, colWhite)
			}
		}
	}
}

func (d *devConsole) say(s string, col color.Color) {
	d.out = append(d.out, devLine{s, col})
	if n := len(d.out) - devKeepLines; n > 0 {
		d.out = append(d.out[:0], d.out[n:]...)
	}
}

func (d *devConsole) echo(cmd string) {
	d.say("> "+cmd, colDevIn)
	if len(d.hist) == 0 || d.hist[len(d.hist)-1] != cmd {
		d.hist = append(d.hist, cmd)
		if n := len(d.hist) - devKeepHist; n > 0 {
			d.hist = append(d.hist[:0], d.hist[n:]...)
		}
	}
	d.histAt = len(d.hist)
}

// devToggle reports whether the console key was just pressed.
func devToggle() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyBackquote) || inpututil.IsKeyJustPressed(ebiten.KeyF12)
}

// updateDev handles the console, and reports whether it has the keyboard
// this frame (so the game must not read it).
func (g *Game) updateDev() bool {
	d := g.dev
	if d == nil || !g.ready || g.paused || g.disconnected {
		return false
	}
	if !d.open {
		if devToggle() {
			d.open = true
			d.histAt = len(d.hist)
			ebiten.AppendInputChars(d.runes[:0]) // the ` itself is not typed
			return true
		}
		return false
	}
	// The player stands still while typing.
	if g.sentInput != (sim.Input{}) {
		if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput}) == nil {
			g.sentInput = sim.Input{}
		}
	}
	if devToggle() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.open = false
		return true
	}
	d.runes = ebiten.AppendInputChars(d.runes[:0])
	for _, r := range d.runes {
		if r == '`' || r == '~' || r < ' ' || len(d.input) >= devMaxInput {
			continue
		}
		d.input = append(d.input, r)
	}
	if repeatKey(ebiten.KeyBackspace) && len(d.input) > 0 {
		d.input = d.input[:len(d.input)-1]
	}
	if repeatKey(ebiten.KeyArrowUp) && d.histAt > 0 {
		d.histAt--
		d.input = []rune(d.hist[d.histAt])
	}
	if repeatKey(ebiten.KeyArrowDown) && d.histAt < len(d.hist) {
		d.histAt++
		d.input = d.input[:0]
		if d.histAt < len(d.hist) {
			d.input = []rune(d.hist[d.histAt])
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		line := strings.TrimSpace(string(d.input))
		d.input = d.input[:0]
		if line != "" {
			d.echo(line)
			if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientDev, Dev: line}) != nil {
				d.say("Không gửi được lệnh tới máy chủ", colDevErr)
			}
		}
	}
	return true
}

// repeatKey is true when k was just pressed, and then again and again while
// it is held.
func repeatKey(k ebiten.Key) bool {
	n := inpututil.KeyPressDuration(k)
	return n == 1 || (n > 30 && n%4 == 0)
}

// onDev shows the server's answer to a console line.
func (g *Game) onDev(r protocol.DevReply) {
	if g.dev == nil {
		return
	}
	col := color.Color(colWhite)
	if r.Rejected {
		col = colDevErr
	}
	for _, l := range r.Lines {
		g.dev.say(l, col)
	}
	if r.Explored != nil && g.width > 0 && g.height > 0 {
		g.wmap.explored = sim.ExploredFrom(g.width, g.height, r.Explored)
		g.wmap.fogDirty = image.Rect(0, 0, g.width, g.height)
	}
}

// drawDev draws the DEV tag and, when open, the console.
func (g *Game) drawDev(screen *ebiten.Image, sw, sh int) {
	d := g.dev
	if d == nil {
		return
	}
	tag := "DEV"
	tw := float32(g.textWidth(tag))
	panel(screen, 250, 12, tw+16, 24)
	g.text(screen, tag, 258, 17, colDevTag, text.AlignStart, 1)
	if !d.open {
		return
	}
	n := min(devShowLines, len(d.out))
	h := float32((n+1)*devLineH + 14)
	y0 := float32(sh) - h - 8
	vector.FillRect(screen, 8, y0, float32(sw-16), h, colDevBack, false)
	vector.StrokeRect(screen, 8, y0, float32(sw-16), h, 1, colDevTag, false)
	y := float64(y0) + 6
	for _, l := range d.out[len(d.out)-n:] {
		g.text(screen, l.s, 16, y, l.col, text.AlignStart, 1)
		y += devLineH
	}
	in := "> " + string(d.input)
	if g.frame/30%2 == 0 {
		in += "_"
	}
	g.text(screen, in, 16, y+2, colDevIn, text.AlignStart, 1)
}
