package dino3d

import (
	"image/color"
	"testing"
)

// Every frame of every direction comes out as a sprite of a sensible size,
// standing on its anchor, in no more colours than the palette and the ink.
func TestBakeRex(t *testing.T) {
	dirs := make([]int, Dirs)
	for i := range dirs {
		dirs[i] = i
	}
	s := BakeRex(dirs...)
	seen := map[color.NRGBA]bool{}
	for d := 0; d < Dirs; d++ {
		for a := Anim(0); a < RexAnims; a++ {
			if len(s.Frames[d][a]) != RexFrames[a] {
				t.Fatalf("dir %d anim %d: %d frames, want %d", d, a, len(s.Frames[d][a]), RexFrames[a])
			}
			for i, f := range s.Frames[d][a] {
				b := f.Img.Bounds()
				if b.Dx() < 30 || b.Dy() < 50 || b.Dx() >= rexW || b.Dy() >= rexH {
					t.Errorf("dir %d anim %d frame %d: %dx%d: empty, or cut off by the canvas", d, a, i, b.Dx(), b.Dy())
				}
				if f.AX < 0 || f.AX > float64(b.Dx()) || f.AY < float64(b.Dy())/2 || f.AY > float64(b.Dy())+8 {
					t.Errorf("dir %d anim %d frame %d: anchor %.0f,%.0f outside its %dx%d feet", d, a, i, f.AX, f.AY, b.Dx(), b.Dy())
				}
				for y := b.Min.Y; y < b.Max.Y; y++ {
					for x := b.Min.X; x < b.Max.X; x++ {
						if c := f.Img.NRGBAAt(x, y); c.A != 0 {
							seen[c] = true
						}
					}
				}
			}
		}
	}
	if max := len(RexPalette()) + 1; len(seen) > max {
		t.Errorf("%d colours, want at most %d", len(seen), max)
	}
}

func BenchmarkBakeRexLeftRight(b *testing.B) {
	for b.Loop() {
		BakeRex(DirRight, DirLeft)
	}
}
