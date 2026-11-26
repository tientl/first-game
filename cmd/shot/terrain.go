package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Tools for looking at the ground of the newest worlds: a GIF of the water
// moving (-gif), the time a frame takes (-bench), the ground's memory (with
// -cost), and fewer trees in the way (-thin).

// thinEntities keeps about keep of the things standing round the player
// (trees, bushes, rocks), so the ground under them shows; creatures stay.
func thinEntities(snap *protocol.Snapshot, keep float64) {
	if keep >= 1 {
		return
	}
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		h := float64(sim.HashTile(int(e.TX), int(e.TY))%1000) / 1000
		if h < keep || e.ID >= 1<<26 { // the scene's own props stay
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
}

// benchFrame notes how long the frame just drawn took, the graphics card's
// work included (reading a pixel back waits for it).
func (s *shooter) benchFrame(screen *ebiten.Image) {
	px := make([]byte, 4)
	screen.SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image).ReadPixels(px)
	now := time.Now()
	if !s.benchLast.IsZero() {
		s.benchTimes = append(s.benchTimes, now.Sub(s.benchLast))
	}
	s.benchLast = now
}

// printBench prints what the frames measured by benchFrame took.
func (s *shooter) printBench() {
	if len(s.benchTimes) == 0 {
		return
	}
	ts := append([]time.Duration(nil), s.benchTimes...)
	sort.Slice(ts, func(a, b int) bool { return ts[a] < ts[b] })
	var sum time.Duration
	for _, t := range ts {
		sum += t
	}
	avg := sum / time.Duration(len(ts))
	fmt.Printf("frames: %d timed; mean %.2f ms, median %.2f ms, 95th %.2f ms (%.0f fps)\n", len(ts),
		ms(avg), ms(ts[len(ts)/2]), ms(ts[len(ts)*95/100]), 1/avg.Seconds())
	chunks, bytes, water := client.GroundCost()
	fmt.Printf("ground: %d chunks cached, %.1f MB of images, of which %.1f MB water maps\n", chunks, float64(bytes)/(1<<20), float64(water)/(1<<20))
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// gifFrame keeps the frame just drawn for the GIF, cropped as -crop says
// (not enlarged).
func (s *shooter) gifFrame(screen *ebiten.Image) {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	var cx, cy, cw, ch int
	if n, _ := fmt.Sscanf(s.crop, "%d,%d,%d,%d", &cx, &cy, &cw, &ch); n == 4 {
		sub := image.NewRGBA(image.Rect(0, 0, cw, ch))
		draw.Draw(sub, sub.Bounds(), img, image.Pt(cx, cy), draw.Src)
		img = sub
	}
	s.gifFrames = append(s.gifFrames, img)
}

// writeGIF writes the frames kept as a looping GIF, delay hundredths of a
// second each, in one palette made from all of them.
func writeGIF(path string, frames []*image.RGBA, delay int) error {
	pal := gifPalette(frames)
	g := &gif.GIF{}
	for _, f := range frames {
		p := image.NewPaletted(f.Bounds(), pal)
		draw.Draw(p, p.Bounds(), f, f.Bounds().Min, draw.Src)
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, delay)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gif.EncodeAll(out, g); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// gifPalette picks 256 colours for the frames: a median cut of their
// pixels.
func gifPalette(frames []*image.RGBA) color.Palette {
	count := map[uint32]int{}
	for _, f := range frames[:min(len(frames), 4)] {
		for i := 0; i < len(f.Pix); i += 4 {
			k := uint32(f.Pix[i])>>2<<12 | uint32(f.Pix[i+1])>>2<<6 | uint32(f.Pix[i+2])>>2
			count[k]++
		}
	}
	type px struct {
		c [3]float64
		n int
	}
	all := make([]px, 0, len(count))
	for k, n := range count {
		all = append(all, px{[3]float64{float64(k>>12&63)*4 + 2, float64(k>>6&63)*4 + 2, float64(k&63)*4 + 2}, n})
	}
	boxes := [][]px{all}
	for len(boxes) < 256 {
		// Split the box with the widest spread of colour, weighted by pixels.
		bi, best, axis := -1, -1.0, 0
		for i, b := range boxes {
			if len(b) < 2 {
				continue
			}
			for a := range 3 {
				lo, hi := math.Inf(1), math.Inf(-1)
				w := 0
				for _, p := range b {
					lo, hi = min(lo, p.c[a]), max(hi, p.c[a])
					w += p.n
				}
				if s := (hi - lo) * math.Sqrt(float64(w)); s > best {
					bi, best, axis = i, s, a
				}
			}
		}
		if bi < 0 {
			break
		}
		b := boxes[bi]
		sort.Slice(b, func(x, y int) bool { return b[x].c[axis] < b[y].c[axis] })
		total := 0
		for _, p := range b {
			total += p.n
		}
		half, acc, cut := total/2, 0, 1
		for i, p := range b {
			acc += p.n
			if acc >= half {
				cut = max(1, min(len(b)-1, i+1))
				break
			}
		}
		boxes[bi] = b[:cut]
		boxes = append(boxes, b[cut:])
	}
	pal := make(color.Palette, 0, len(boxes))
	for _, b := range boxes {
		var r, g, bl, w float64
		for _, p := range b {
			r += p.c[0] * float64(p.n)
			g += p.c[1] * float64(p.n)
			bl += p.c[2] * float64(p.n)
			w += float64(p.n)
		}
		pal = append(pal, color.RGBA{uint8(r / w), uint8(g / w), uint8(bl / w), 255})
	}
	return pal
}

// gifPath is where the GIF goes: out with .gif for .png.
func gifPath(out string) string { return strings.TrimSuffix(out, ".png") + ".gif" }
