package client

import (
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"survivalgame/internal/sim"
)

// skillKinds is the order the sheet shows the beasts in.
var skillKinds = []sim.CreatureKind{
	sim.CreatureRex, sim.CreatureRaptor, sim.CreatureAchero, sim.CreatureTroodon, sim.CreaturePtero,
	sim.CreatureQuetzal, sim.CreaturePlesio, sim.CreatureSabre, sim.CreatureTrike, sim.CreatureToro,
	sim.CreatureStego, sim.CreatureBrachio, sim.CreatureAnkylo, sim.CreatureAnzu, sim.CreatureThesc,
	sim.CreatureOrnitho, sim.CreatureMammoth,
}

type keyPose struct {
	c      *canvas
	ax, ay float64
	label  string
}

// skillKeys are the key poses of move id, drawn on the CPU: wound up (both
// ends of a fidget), struck, and the recovery's start.
func skillKeys(id sim.SkillID) []keyPose {
	d := sim.Skill(id)
	var out []keyPose
	add := func(c *canvas, ax, ay float64, label string) {
		if c != nil {
			out = append(out, keyPose{c, ax, ay, label})
		}
	}
	rig := func(name string, p rigPose, label string) {
		c, ax, ay, _ := rigCanvas(dinoRigs[name], p)
		add(c, ax, ay, label)
	}
	crit := func(hd func(pose, f int) (*canvas, float64, float64), pose, f int, label string) {
		c, ax, ay := hd(pose, f)
		ax, ay = ax*cK, ay*cK // a critter's anchor is in art pixels
		add(c, ax, ay, label)
	}
	switch id {
	case sim.SkillMammothCharge:
		crit(mammothHD, mammothSkHeadDown, 0, "lấy đà")
		crit(mammothHD, mammothSkHeadDown, 2, "cào đất")
		crit(mammothHD, mammothCharge, 3, "lao")
		return out
	case sim.SkillMammothSwipe:
		crit(mammothHD, mammothSkCurl, 0, "lấy đà")
		crit(mammothHD, mammothSkSwipe, 0, "ra đòn")
		return out
	case sim.SkillMammothTrumpet:
		crit(mammothHD, mammothSkTrumpet, 0, "rống")
		return out
	case sim.SkillSabrePounce:
		crit(sabreHD, sabreSkLow, 0, "lấy đà")
		crit(sabreHD, sabreSkLow, 1, "giật đuôi")
		crit(sabreHD, sabreSpring, 4, "vồ")
		crit(sabreHD, sabreSkBite, 0, "hồi")
		return out
	case sim.SkillSabreBite:
		crit(sabreHD, sabreSkGape, 0, "lấy đà")
		crit(sabreHD, sabreSkBite, 0, "ra đòn")
		return out
	case sim.SkillPteroDive, sim.SkillPteroSnatch:
		rig("pteroFly", rigPose{squash: 0.8, wings: 0.12}, "lượn chờ")
		rig("pteroFly", dinoPoses["pteroFly"][pteroDive], "bổ nhào")
		rig("pteroFly", flapPose(2), "bay lên")
		return out
	case sim.SkillAnkyloClub:
		rig("ankylo", ankyloWind, "lấy đà")
		rig("ankylo", ankyloSwingPose(2), "vung")
		rig("ankylo", dinoPoses["ankylo"][ankyloSwing], "ra đòn")
		return out
	}
	look, ok := skillLooks[id]
	name := skillRigs[d.Kind]
	if !ok || name == "" {
		return out
	}
	n := float64(d.Windup)
	if id != sim.SkillPlesioAmbush {
		p, _ := skillPose(look, sim.PhaseWindup, n-1, n, 0, d.Every)
		rig(name, p, "lấy đà")
		if look.wind2 != (rigPose{}) {
			rig(name, look.wind2, "nhịp chờ")
		}
	}
	rig(name, look.strike, "ra đòn")
	if look.strike2 != (rigPose{}) {
		rig(name, look.strike2, "đòn 2")
	}
	if look.recover != (rigPose{}) {
		rig(name, look.recover, "hồi")
	}
	return out
}

// TestDumpSkillSheet writes $SKILL_SHEET: the key poses of every beast's
// every move, a row to a beast, at the size they are shown at 1080p.
func TestDumpSkillSheet(t *testing.T) {
	path := os.Getenv("SKILL_SHEET")
	if path == "" {
		t.Skip("set SKILL_SHEET to a PNG path")
	}
	only := os.Getenv("SKILL_KINDS")
	type row struct {
		title string
		cells []keyPose
		moves []string // the move over each cell
		w, h  int
	}
	var rows []row
	for _, k := range skillKinds {
		if only != "" && !strings.Contains(","+only+",", ","+skillRigs[k]+",") && !strings.Contains(only, k.Name()) {
			continue
		}
		r := row{title: k.Name()}
		std := rigPose{}
		if name, ok := skillRigs[k]; ok {
			c, ax, ay, _ := rigCanvas(dinoRigs[name], std)
			r.cells = append(r.cells, keyPose{c, ax, ay, "đứng"})
			r.moves = append(r.moves, "")
		}
		for _, id := range sim.SkillsOf(k) {
			d := sim.Skill(id)
			for i, kp := range skillKeys(id) {
				r.cells = append(r.cells, kp)
				m := ""
				if i == 0 {
					m = d.Label
				}
				r.moves = append(r.moves, m)
			}
		}
		for _, c := range r.cells {
			r.w = max(r.w, c.c.w)
			r.h = max(r.h, c.c.h)
		}
		r.w, r.h = r.w+24, r.h+60
		rows = append(rows, r)
	}
	W, H := 0, 0
	for _, r := range rows {
		W = max(W, 160+r.w*len(r.cells))
		H += r.h
	}
	sheet := image.NewNRGBA(image.Rect(0, 0, W, H))
	reviewGrass(sheet)
	text := func(x, y int, s string, col color.Color) {
		d := font.Drawer{Dst: sheet, Src: image.NewUniform(col), Face: bitmapfont.Face, Dot: fixed.P(x, y)}
		d.DrawString(s)
	}
	y := 0
	for _, r := range rows {
		text(8, y+r.h/2, r.title, color.White)
		for i, c := range r.cells {
			x := 160 + i*r.w
			ground := y + r.h - 22
			reviewPut(sheet, c.c, x+r.w/2-int(c.ax), ground-int(c.ay), 1)
			if r.moves[i] != "" {
				text(x+4, y+16, r.moves[i], color.RGBA{255, 230, 120, 255})
			}
			text(x+4, y+r.h-6, c.label, color.RGBA{230, 240, 230, 255})
		}
		y += r.h
	}
	reviewPNG(t, path, sheet)
}
