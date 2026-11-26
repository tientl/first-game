package client

import (
	"image/color"
	"math"
)

// The rabbit, side-on and facing right: a round fluffy body in warm agouti
// brown, a cream chest and chin, long ears with pink insides, a white puff
// of a tail and a bright dark eye. It sits up with its ears high, stretches
// out flat in a leap with the ears swept back, and sleeps tucked into a loaf.

var (
	furRamp  = mkRamp(0x3f2a1a, 0x644629, 0x8a6540, 0xad875a, 0xcdab7b, 0xe8cfa2)
	paleRamp = mkRamp(0xa8937a, 0xd2bf9f, 0xefe2c6, 0xfdf7e6) // chest, chin and feet
	tailRamp = mkRamp(0xa9a8b4, 0xd6d4d6, 0xf4f1ea, 0xffffff)
	earPink  = hex(0xe59a94)
	earDeep  = hex(0xb96a6c)
	colNose  = hex(0xd9777c)
)

// furLit is the brightness of a furry lump at (nx, ny) on its own ball, part
// of an animal whose whole body is the ellipse vol: the lump's own roundness
// and the body's, so the parts read as one lit volume.
func furLit(vol ell, x, y int, nx, ny float64) float64 {
	gx, gy := (float64(x)+0.5-vol.x)/vol.rx, (float64(y)+0.5-vol.y)/vol.ry
	l := lum(nx*0.85, ny*0.85)*0.5 + lum(gx*0.85, gy*0.85)*0.5
	return (l-0.5)*1.35 + 0.47
}

// furPick takes a fur colour from rp for brightness l, flecked with darker
// and lighter hairs.
func furPick(rp ramp, l float64, x, y int) color.RGBA {
	i := rp.idx(l, x, y)
	switch h := hashf(x, y, 4242); {
	case h < 0.09:
		i--
	case h > 0.93:
		i++
	}
	return rp.at(i)
}

// furBall fills e with fur.
func furBall(c *canvas, vol, e ell, rp ramp, bias float64) {
	c.ellipse(e.x, e.y, e.rx, e.ry, func(x, y int, nx, ny float64) color.RGBA {
		return furPick(rp, furLit(vol, x, y, nx, ny)+bias, x, y)
	})
}

// furLimb is a furry limb from (x0, y0) to (x1, y1), r0 thick at the start
// and r1 at the end.
func furLimb(c *canvas, vol ell, x0, y0, x1, y1, r0, r1 float64, rp ramp, bias float64) {
	n := int(math.Hypot(x1-x0, y1-y0)*2) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		r := lerpf(r0, r1, t)
		furBall(c, vol, ell{lerpf(x0, x1, t), lerpf(y0, y1, t), r, r}, rp, bias)
	}
}

func genRabbit(pose int) sprite { return critter("rabbit", 0, pose, 0) }

// rabbitCanvas draws rabbit pose on a canvas and returns its anchor.
func rabbitCanvas(pose int) (*canvas, float64, float64) {
	c := newCanvas(20, 18)
	px := func(x, y int, col color.RGBA) { c.set(x, y, col) }
	switch pose {
	case rabbitHop: // stretched out mid-leap, ears swept back
		vol := ell{9.5, 9.5, 9, 5}
		furLimb(c, vol, 14.8, 6, 12, 2.8, 0.9, 0.8, furRamp, -0.22)    // far ear
		furLimb(c, vol, 4.6, 11, 1.6, 13.2, 1.3, 0.8, furRamp, -0.2)   // far hind leg
		furLimb(c, vol, 12.4, 11, 15.6, 13.4, 0.9, 0.7, furRamp, -0.2) // far fore leg
		furBall(c, vol, ell{2.6, 8.4, 1.7, 1.6}, tailRamp, 0.2)        // tail
		furBall(c, vol, ell{6.6, 10, 4.3, 3.1}, furRamp, 0)            // haunch
		furBall(c, vol, ell{10.6, 9.7, 4, 2.7}, furRamp, 0.04)         // chest
		furBall(c, vol, ell{10, 11.3, 3.4, 1.3}, paleRamp, 0.2)        // belly
		furLimb(c, vol, 5.4, 11.6, 2.2, 14.2, 1.5, 0.8, furRamp, 0)    // near hind leg
		furLimb(c, vol, 2.4, 14.4, 0.8, 15.2, 0.8, 0.7, paleRamp, 0.3) // and its long foot
		furLimb(c, vol, 13.2, 11.2, 16.8, 13.2, 1, 0.7, furRamp, 0.05) // near fore leg
		px(17, 13, paleRamp[2])
		furBall(c, vol, ell{15.2, 7.9, 2.7, 2.3}, furRamp, 0.08) // head
		furBall(c, vol, ell{17.1, 8.7, 1.3, 1.1}, furRamp, 0.16) // muzzle
		furBall(c, vol, ell{16, 9.5, 1.5, 0.7}, paleRamp, 0.3)   // chin
		furLimb(c, vol, 14, 6, 9.6, 4.2, 1, 0.9, furRamp, 0.02)  // near ear
		for x := 11; x <= 13; x++ {
			px(x, 5, earPink)
		}
		px(2, 7, tailRamp[3])
		px(1, 8, tailRamp[3])
		px(16, 7, hex(0x2a1c1c))
		px(16, 8, colInk)
		px(15, 7, hex(0xffffff))
		px(18, 8, colNose)
		c.outline(0.72)
		return c, 9, 16
	case rabbitSleep: // tucked into a loaf, ears flat along the back
		vol := ell{9, 12.5, 8, 4.5}
		furBall(c, vol, ell{2.4, 12.8, 1.6, 1.5}, tailRamp, 0.15)        // tail
		furLimb(c, vol, 12.6, 10.8, 8.2, 10.2, 0.9, 0.8, furRamp, -0.22) // far ear
		furBall(c, vol, ell{7.4, 12.6, 5.2, 3.5}, furRamp, 0)            // the loaf
		furBall(c, vol, ell{10.6, 13, 3.4, 3}, furRamp, 0.03)            // shoulder
		furBall(c, vol, ell{8.6, 15.5, 3, 0.8}, paleRamp, 0.25)          // hind foot peeping out
		furBall(c, vol, ell{14, 12.4, 2.8, 2.4}, furRamp, 0.08)          // head, low
		furBall(c, vol, ell{15.9, 13.3, 1.3, 1.1}, furRamp, 0.16)        // muzzle
		furBall(c, vol, ell{14.2, 14.6, 2.2, 1}, paleRamp, 0.3)          // chin on the paws
		furLimb(c, vol, 13, 10.4, 8, 9.2, 1, 0.9, furRamp, 0.04)         // near ear
		for x := 10; x <= 12; x++ {
			px(x, 10, earPink)
		}
		px(13, 12, furRamp[0]) // a closed eye
		px(14, 12, furRamp[0])
		px(17, 13, colNose)
		px(1, 12, tailRamp[3])
		c.outline(0.72)
		return c, 9, 16
	}
	// Sitting up, ears high.
	vol := ell{9, 10, 8, 7}
	furLimb(c, vol, 11.8, 5.6, 10.6, 1.6, 1, 0.9, furRamp, -0.22) // far ear
	furBall(c, vol, ell{2.2, 11.8, 1.8, 1.7}, tailRamp, 0.2)      // tail
	furBall(c, vol, ell{6.6, 12.1, 4.5, 3.8}, furRamp, 0)         // haunch
	furBall(c, vol, ell{9.8, 11.3, 3.6, 3.5}, furRamp, 0.04)      // back and shoulder
	furBall(c, vol, ell{12, 12.3, 1.9, 2.7}, paleRamp, 0.2)       // cream chest
	furBall(c, vol, ell{8.4, 15.5, 3, 0.9}, paleRamp, 0.2)        // the long hind foot
	c.rect(12, 13, 2, 3, func(x, y int) color.RGBA { return furPick(furRamp, 0.62, x, y) })
	px(13, 15, paleRamp[2]) // fore paw
	px(14, 15, paleRamp[2])
	furBall(c, vol, ell{13.4, 7.9, 3, 2.7}, furRamp, 0.06)       // head
	furBall(c, vol, ell{15.5, 8.8, 1.4, 1.2}, furRamp, 0.16)     // muzzle
	furBall(c, vol, ell{14.4, 10, 1.7, 0.8}, paleRamp, 0.3)      // chin and cheek
	furLimb(c, vol, 13.4, 5.4, 12.8, 1.4, 1.2, 1, furRamp, 0.04) // near ear
	for y := 2; y <= 4; y++ {
		px(13, y, earPink)
	}
	px(13, 5, earDeep)
	px(1, 11, tailRamp[3]) // fluff
	px(2, 10, tailRamp[3])
	px(14, 7, colInk)
	px(14, 8, colInk)
	px(14, 7, hex(0x2a1c1c))
	px(13, 7, hex(0xffffff)) // the glint
	px(16, 8, colNose)
	c.outline(0.72)
	return c, 9, 16
}
