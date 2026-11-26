package client

import (
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestDumpDinoReview writes, for each species in $DINO_SPECIES (default
// rex) to the folder $DINO_REVIEW: <species>_poses.png, every pose of its
// rig at the size it is shown at 1080p (the standing poses, then the walk,
// the run and the blow, a frame each); <species>_walk.gif and
// <species>_attack.gif, twice that size. For checking a redrawing (see
// dinos/hd/README.md).
func TestDumpDinoReview(t *testing.T) {
	dir := os.Getenv("DINO_REVIEW")
	if dir == "" {
		t.Skip("set DINO_REVIEW to a folder")
	}
	names := []string{"rex"}
	if s := os.Getenv("DINO_SPECIES"); s != "" {
		names = strings.Split(s, ",")
	}
	hits := map[string]int{"rex": rexBite, "dakota": raptorPounce, "achero": raptorPounce, "trike": trikeGore, "toro": trikeGore,
		"stego": stegoSwipe, "anzu": anzuKick, "quetzal": quetzalStab, "plesio": plesioStrike}
	for _, sp := range names {
		r, ok := dinoRigs[sp]
		if !ok {
			t.Fatalf("no rig %q", sp)
		}
		fam := poseFamily(sp)
		var still []rigPose
		still = append(still, rigPose{})
		var keys []int
		for k := range dinoPoses[fam] {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		for _, k := range keys {
			still = append(still, dinoPoses[fam][k])
		}
		if w, ok := dinoWinds[fam]; ok {
			still = append(still, w.pose)
		}
		walk, run, strike := animFrames(sp, hits[sp])
		rows := [][]rigPose{still, walk, run, strike}

		draw := func(p rigPose) (*canvas, float64, float64) {
			c, ax, ay, _ := rigCanvas(r, p)
			return c, ax, ay
		}
		base := dinoImage(r.file, false)
		k := HDDensity
		if !dinoHD {
			k = 1
		}
		cw, ch := (base.w+50)*k, (base.h+30)*k // a cell, the anchor at the same place in each
		const perRow = 6
		var lines [][]rigPose
		for _, row := range rows {
			for i := 0; i < len(row); i += perRow {
				lines = append(lines, row[i:min(len(row), i+perRow)])
			}
		}
		sheet := image.NewNRGBA(image.Rect(0, 0, cw*perRow, ch*len(lines)))
		reviewGrass(sheet)
		for li, line := range lines {
			for i, p := range line {
				c, ax, ay := draw(p)
				reviewPut(sheet, c, i*cw+cw/2-int(ax), li*ch+ch-6*k-int(ay), 1)
			}
		}
		reviewPNG(t, filepath.Join(dir, sp+"_poses.png"), sheet)

		anim := func(name string, poses []rigPose, delay int) {
			var frames []*image.NRGBA
			for _, p := range poses {
				c, ax, ay := draw(p)
				img := image.NewNRGBA(image.Rect(0, 0, cw*2, ch*2))
				reviewGrass(img)
				reviewPut(img, c, cw-2*int(ax), 2*ch-12*k-2*int(ay), 2)
				frames = append(frames, img)
			}
			reviewGIF(t, filepath.Join(dir, sp+"_"+name+".gif"), frames, delay)
		}
		anim("walk", walk, 6)
		if len(strike) > 0 {
			att := append([]rigPose{{}, {}, {}}, strike...)
			if roar, ok := dinoPoses[fam][rexRoar]; ok && fam == "rex" {
				for range 8 {
					att = append(att, roar)
				}
			}
			anim("attack", att, 5)
		}
	}
}

func reviewGrass(img *image.NRGBA) {
	for j := range img.Pix {
		img.Pix[j] = []byte{88, 128, 70, 255}[j%4]
	}
}

// put draws canvas c on img with its top left at (x, y), k times bigger.
func reviewPut(img *image.NRGBA, c *canvas, x, y, k int) {
	for py := 0; py < c.h*k; py++ {
		for px := 0; px < c.w*k; px++ {
			if q := c.px[(py/k)*c.w+px/k]; q.A >= 128 && image.Pt(x+px, y+py).In(img.Rect) {
				img.SetNRGBA(x+px, y+py, color.NRGBA{q.R, q.G, q.B, 255})
			}
		}
	}
}

func reviewPNG(t *testing.T, path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// writeGIF writes frames as a looping GIF on one palette: the 256 colours
// used most, every other colour drawn as the nearest of them.
func reviewGIF(t *testing.T, path string, frames []*image.NRGBA, delay int) {
	count := map[color.NRGBA]int{}
	for _, f := range frames {
		for i := 0; i < len(f.Pix); i += 4 {
			count[color.NRGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}]++
		}
	}
	type kv struct {
		c color.NRGBA
		n int
	}
	var all []kv
	for c, n := range count {
		all = append(all, kv{c, n})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	var pal color.Palette
	for _, e := range all[:min(256, len(all))] {
		pal = append(pal, e.c)
	}
	near := map[color.NRGBA]uint8{}
	out := &gif.GIF{}
	for _, f := range frames {
		pi := image.NewPaletted(f.Rect, pal)
		for i := 0; i < len(f.Pix); i += 4 {
			c := color.NRGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255}
			idx, ok := near[c]
			if !ok {
				idx = uint8(pal.Index(c))
				near[c] = idx
			}
			pi.Pix[i/4] = idx
		}
		out.Image = append(out.Image, pi)
		out.Delay = append(out.Delay, delay)
	}
	fh, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if err := gif.EncodeAll(fh, out); err != nil {
		t.Fatal(err)
	}
}
