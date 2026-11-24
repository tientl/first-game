package dino3d

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"survivalgame/internal/r3d"
)

// Dirs is how many ways a baked creature can face: 0 is right, then
// anticlockwise seen from above in eighths of a turn, so 2 is away from the
// viewer, 4 left and 6 towards the viewer.
const Dirs = 8

const (
	DirRight = 0
	DirLeft  = 4
)

// Bake settings for the tyrant.
const (
	// The game draws its ground 24 wide by 18 deep to the tile, which is a
	// camera 48.6 degrees up. Standing things are drawn nearer side-on than
	// that, as the drawn sprites are; this is in between, enough to show
	// the back and the far-side legs.
	rexElev  = 27 * math.Pi / 180
	rexScale = 9.2 // pixels to the model's unit: as long as the drawn tyrant
	rexSS    = 4   // rendered this many times larger, then averaged down
	rexW     = 176 // the sprite's canvas before cropping
	rexH     = 184
	rexOX    = 88
	rexOY    = 136
)

// rexLight is the game's light (client/pix.go: up, left and out of the
// screen), with cool shadows and a faint rim along the back.
var rexLight = r3d.Light{
	Dir:  r3d.V(-0.55, -0.7, 0.55),
	Wrap: 0.22, Dark: 0.34, Bright: 1.22,
	Shadow:   r3d.RGB{R: 0.96, G: 0.86, B: 0.84},
	Rim:      0.5,
	RimColor: r3d.Hex(0xe8d8a8),
}

// RexPalette is every colour a baked tyrant uses, the outline aside.
var RexPalette = sync.OnceValue(func() []r3d.RGB {
	mats := []r3d.RGB{rexBack, rexStripe, rexFlank, rexBelly, rexShank, rexToe}
	keep := []r3d.RGB{rexTooth, rexEye, rexSocket, rexMouth, rexTongue, rexClaw, r3d.Hex(0xc8bc9c)}
	return r3d.ShadePalette(rexLight, mats, 6, 24, keep)
})

// Sprites is a baked creature: Frames[dir][anim][frame]. A direction that
// was not asked for is left empty.
type Sprites struct {
	Frames [Dirs][RexAnims][]r3d.Sprite
}

var theRex = sync.OnceValue(NewRex)

// BakeRex renders the tyrant's every frame facing each of dirs.
func BakeRex(dirs ...int) *Sprites {
	r := theRex()
	out := &Sprites{}
	pal := RexPalette()
	type job struct{ dir, anim, i int }
	var jobs []job
	for _, d := range dirs {
		for a := Anim(0); a < RexAnims; a++ {
			out.Frames[d][a] = make([]r3d.Sprite, RexFrames[a])
			for i := range RexFrames[a] {
				jobs = append(jobs, job{d, int(a), i})
			}
		}
	}
	ch := make(chan job)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := r3d.NewTarget(rexW*rexSS, rexH*rexSS)
			var posed r3d.Mesh
			for j := range ch {
				t.Clear()
				out.Frames[j.dir][j.anim][j.i] = r.render(t, &posed, pal, j.dir, rexFrame(Anim(j.anim), j.i))
			}
		}()
	}
	for _, j := range jobs {
		ch <- j
	}
	close(ch)
	wg.Wait()
	return out
}

func (r *Rex) render(t *r3d.Target, posed *r3d.Mesh, pal []r3d.RGB, dir int, p rexPose) r3d.Sprite {
	mats := r.apply(p)
	r.mesh.Skin(mats, posed)
	cam := r3d.Camera{
		Yaw: float64(dir) * math.Pi / 4, Elev: rexElev, Scale: rexScale * rexSS,
		OX: rexOX * rexSS, OY: rexOY * rexSS,
	}
	r3d.Render(t, posed, cam, rexLight)
	var dots []r3d.Dot
	for _, e := range r.eyes {
		dots = append(dots, r3d.Dot{P: mats[r.head].Point(e), C: rexEye, Bias: 0.12})
	}
	return r3d.Resolve(t, cam, r3d.SpriteOpts{
		SS: rexSS, Cover: 0.42, Palette: pal, Ink: rexInk, Outline: true, Edge: 0.24, EdgeDark: 0.6, Grain: 0.07,
	}, dots)
}

// RexSheet lays every frame of every direction out on the game's grass,
// enlarged zoom times: a row to a direction, for looking the model over.
func RexSheet(s *Sprites, zoom int) *image.NRGBA {
	const cw, ch, ax, ay = 168, 176, 84, 128
	cols := 0
	for _, n := range RexFrames {
		cols += n
	}
	small := image.NewNRGBA(image.Rect(0, 0, cols*cw, Dirs*ch))
	grass := [2]color.NRGBA{{0x5d, 0x9e, 0x45, 255}, {0x57, 0x97, 0x42, 255}}
	for d := 0; d < Dirs; d++ {
		col := 0
		for a := Anim(0); a < RexAnims; a++ {
			for i := 0; i < RexFrames[a]; i++ {
				cell := image.Rect(col*cw, d*ch, (col+1)*cw, (d+1)*ch)
				draw.Draw(small, cell, image.NewUniform(grass[(col+d)%2]), image.Point{}, draw.Src)
				if fr := s.Frames[d][a]; i < len(fr) && fr[i].Img != nil {
					sp := fr[i]
					at := image.Pt(cell.Min.X+ax-int(math.Round(sp.AX)), cell.Min.Y+ay-int(math.Round(sp.AY)))
					to := sp.Img.Bounds().Add(at).Intersect(cell)
					draw.Draw(small, to, sp.Img, to.Min.Sub(at), draw.Over)
				}
				col++
			}
		}
	}
	big := image.NewNRGBA(image.Rect(0, 0, small.Rect.Dx()*zoom, small.Rect.Dy()*zoom))
	for y := 0; y < big.Rect.Dy(); y++ {
		for x := 0; x < big.Rect.Dx(); x++ {
			big.SetNRGBA(x, y, small.NRGBAAt(x/zoom, y/zoom))
		}
	}
	return big
}
