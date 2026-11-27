package client

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The held sheets: every held item in the caveman's hand, old against new,
// and GIFs of the swings, played through the game's own drawing as the
// motion sheets are (motionsheet.go):
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -heldsheet dir
//
// It must run inside the game loop (Draw).

// heldSheetItems is what the comparison shows, in order.
var heldSheetItems = func() []sim.ItemKind {
	var ks []sim.ItemKind
	for t := sim.TierStone; t < sim.NumTiers; t++ {
		ks = append(ks, sim.ToolItem(sim.ToolAxe, t))
	}
	for t := sim.TierStone; t < sim.NumTiers; t++ {
		ks = append(ks, sim.ToolItem(sim.ToolPickaxe, t))
	}
	for t := sim.TierStone; t < sim.NumTiers; t++ {
		ks = append(ks, sim.ToolItem(sim.ToolHoe, t))
	}
	for t := sim.TierCopper; t < sim.NumTiers; t++ {
		ks = append(ks, sim.SwordItem(t))
	}
	return append(ks, sim.ItemSpear, sim.ItemIvorySpear, sim.ItemBattleAxe, sim.ItemTorch, sim.ItemRod,
		sim.ItemClayBowl, sim.ItemFlask, sim.ItemGourd, sim.ItemCookedFish, sim.ItemWood, sim.ItemBerry)
}()

// HeldSheets writes held_compare.png and held_swings.gif into dir, and
// says what the held items cost.
func (g *Game) HeldSheets(dir string) error {
	keepFrame, keepHD, keepWarm := g.frame, heldHD, heldWarmAhead
	defer func() { g.frame, heldHD, heldWarmAhead = keepFrame, keepHD, keepWarm }()
	heldWarmAhead = false // so heldCost times the painting alone
	if err := g.heldCompare(filepath.Join(dir, "held_compare.png")); err != nil {
		return err
	}
	heldHD = true
	if err := g.heldSwings(filepath.Join(dir, "held_swings.gif")); err != nil {
		return err
	}
	if err := g.heldMotions(filepath.Join(dir, "held_motions_axe.png"), sim.ItemAxe,
		"walk", "run", "idle", "guard", "gather", "reach up", "throw", "eat meat", "sit by the fire", "hurt",
		"swim", "ford", "ride three-horn"); err != nil {
		return err
	}
	if err := g.heldMotions(filepath.Join(dir, "held_motions_long.png"), sim.ItemSpear, "walk", "idle", "ride three-horn"); err != nil {
		return err
	}
	if err := g.heldMotions(filepath.Join(dir, "held_motions_torch.png"), sim.ItemTorch, "walk", "run", "idle", "sit by the fire"); err != nil {
		return err
	}
	g.heldCost()
	return nil
}

// heldCompare draws each item of heldSheetItems in the hand of a caveman
// facing right, old and new, at 1080p and three times as big about the
// hand.
func (g *Game) heldCompare(path string) error {
	const (
		cw, ch = cellW * HDDensity, cellH * HDDensity // a cell at 1080p
		zw     = 38                                   // the close-up, art pixels
		zk     = 3                                    // enlarged so many times
		zp     = zw * HDDensity * zk
		bw     = 2*cw + 2*zp + 24
		bh     = zp + 26
		cols   = 2
	)
	items := heldSheetItems
	rows := (len(items) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*bw, rows*bh+30))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{36, 40, 36, 255}}, image.Point{}, draw.Src)
	label(sheet, 8, 8, "old (left) and new (right): at 1080p, and the hand x3")
	buf := newFineBuf(cellW, cellH)
	for i, k := range items {
		bx, by := (i%cols)*bw, (i/cols)*bh+30
		for j, hd := range []bool{false, true} {
			heldHD = hd
			g.resetMotions()
			img := g.render(reel{bg: reelGrass, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
				p.Held = k
			}}, sim.DirRight, 0, buf)
			draw.Draw(sheet, image.Rect(bx+j*cw, by+20, bx+(j+1)*cw, by+20+ch), img, image.Point{}, draw.Src)
			// The close-up: about the hand, which is about 11 art pixels up
			// from the feet and a little ahead.
			x0, y0 := (cellX-zw/2+2)*HDDensity, (cellY-11-zw/2-4)*HDDensity
			ox, oy := bx+2*cw+12+j*(zp+12), by+20
			for y := 0; y < zw*HDDensity; y++ {
				for x := 0; x < zw*HDDensity; x++ {
					c := img.RGBAAt(x0+x, y0+y)
					for dy := 0; dy < zk; dy++ {
						for dx := 0; dx < zk; dx++ {
							sheet.SetRGBA(ox+x*zk+dx, oy+y*zk+dy, c)
						}
					}
				}
			}
		}
		label(sheet, bx+4, by+4, asciiName(k))
	}
	return sheetPNG(path, sheet)
}

// asciiName is k's name for the sheets' small hand, which has no accents.
func asciiName(k sim.ItemKind) string {
	if class, t := k.ToolKind(); class != sim.NoTool {
		return fmt.Sprintf("%s %s", [...]string{"", "axe", "pick", "hoe"}[class], tierWord[t])
	}
	if t, ok := sim.SwordTier(k); ok {
		return "knife " + tierWord[t]
	}
	switch k {
	case sim.ItemSpear:
		return "spear"
	case sim.ItemIvorySpear:
		return "ivory spear"
	case sim.ItemBattleAxe:
		return "war axe"
	case sim.ItemTorch:
		return "torch"
	case sim.ItemRod:
		return "rod"
	case sim.ItemClayBowl:
		return "bowl (icon)"
	case sim.ItemFlask:
		return "flask (icon)"
	case sim.ItemGourd:
		return "gourd (icon)"
	case sim.ItemCookedFish:
		return "cooked fish (icon)"
	case sim.ItemWood:
		return "wood (hd icon)"
	case sim.ItemBerry:
		return "berry (hd icon)"
	}
	return fmt.Sprintf("item %d", k)
}

var tierWord = [sim.NumTiers]string{"stone", "flint", "jade", "agate", "obsidian"}

// heldSwings writes a GIF of the chop, the slash, the thrust and a walk
// with a torch, every facing, with the HD held items.
func (g *Game) heldSwings(path string) error {
	blow := func(held sim.ItemKind, w swingInfo, period, before int) func(*Game, int, *protocol.PlayerView, *drawable) {
		return func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Held = held
			start := g.frame - t%period
			p.Acting = g.frame-start < sim.ActingTicks*3
			if g.swingAt == nil {
				g.swingAt, g.swingWhat = map[sim.PlayerID]int{}, map[sim.PlayerID]swingInfo{}
			}
			g.swingAt[p.ID], g.swingWhat[p.ID] = start, w
			st := g.moveState(p.ID)
			st.blows = before + t/period
			st.blowAt, st.movedAt = start, g.frame-1000
		}
	}
	reels := []reel{
		{name: "chop", n: 36, dirs: allDirs, bg: reelGrass, at: blow(sim.ItemAxe, swingInfo{kind: swingAxe}, 36, 0)},
		{name: "slash", n: 72, dirs: allDirs, bg: reelGrass, at: blow(sim.ItemIronSword, swingInfo{kind: swingHand, beast: true}, 36, 1)},
		{name: "thrust", n: 30, dirs: allDirs, bg: reelGrass, at: blow(sim.ItemSpear, swingInfo{kind: swingThrust}, 30, 0)},
		{name: "torch walk", n: 48, dirs: allDirs, bg: color.RGBA{52, 70, 60, 255}, at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Held = sim.ItemTorch
			dx, dy := p.Dir.Delta()
			along := float32(0.9 * float64(t) / TileW)
			p.X += float32(dx) * along
			p.Y += float32(dy) * along
			d.moving = true
		}},
	}
	return g.writeReelGIF(path, reels, newFineBuf(cellW, cellH))
}

// heldCost prints what painting the held items costs: each angle's time and
// size, and every angle of every item.
func (g *Game) heldCost() {
	n, px, start := 0, 0, time.Now()
	for _, d := range heldDefs {
		if d == nil {
			continue
		}
		for i := range heldTurns {
			p := bakeHeld(d, i)
			n++
			px += p.w * p.h
		}
	}
	el := time.Since(start)
	fmt.Printf("held items: %d angles painted in %v (%.2f ms each), %.1f MB if every angle of every item were shown\n",
		n, el.Round(time.Millisecond), float64(el.Microseconds())/1000/float64(n), float64(4*px)/(1<<20))
	shown, bytes := 0, 0
	for _, s := range heldSets {
		for _, f := range s.frame {
			if f != nil {
				shown++
				b := f.img.Bounds()
				bytes += 4 * b.Dx() * b.Dy()
			}
		}
	}
	fmt.Printf("held items shown for these sheets: %d angles of %d items, %.2f MB\n", shown, len(heldSets), float64(bytes)/(1<<20))
}

// heldMotions draws the motions in which something stays in the hand,
// with held in it, every facing, a few frames of each: to see that the
// grip stays in the fist and what is held goes in front of the body or
// behind it as it should.
func (g *Game) heldMotions(path string, held sim.ItemKind, names ...string) error {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var reels []reel
	for _, r := range g.motionReels() {
		if !want[r.name] {
			continue
		}
		at := r.at
		r.at = func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
			p.Held = held
			if at != nil {
				at(g, t, p, d)
			}
			if p.Held == sim.ItemNone || p.Held == sim.ItemStone { // a gather's or a throw's own
				p.Held = held
			}
		}
		reels = append(reels, r)
	}
	const cols = 8
	cw, ch := cellW*HDDensity, cellH*HDDensity
	rows := 0
	for _, r := range reels {
		rows += len(r.dirs)
	}
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cw, rows*ch))
	buf := newFineBuf(cellW, cellH)
	row := 0
	for _, r := range reels {
		for _, d := range r.dirs {
			g.resetMotions()
			step := max(1, r.n/cols)
			for t := 0; t < r.n && t/step < cols; t++ {
				img := g.render(r, d, t, buf)
				if t%step == 0 {
					i := t / step
					draw.Draw(sheet, image.Rect(i*cw, row*ch, (i+1)*cw, (row+1)*ch), img, image.Point{}, draw.Src)
				}
			}
			label(sheet, 4, row*ch+4, r.name)
			row++
		}
	}
	return sheetPNG(path, sheet)
}
