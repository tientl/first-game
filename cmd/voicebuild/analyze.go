package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"
	"slices"
)

// The tools to look at a sound without hearing it: its level over time, the
// calls in it, and a picture of its spectrum.

func db(v float64) float64 {
	if v <= 1e-9 {
		return -180
	}
	return 20 * math.Log10(v)
}

// frameRMS is x's level every 10 ms, over 30 ms windows.
func frameRMS(x []float64) []float64 {
	hop, win := secs(0.01), secs(0.03)
	var out []float64
	for o := 0; o < len(x); o += hop {
		out = append(out, rmsOf(x[o:min(len(x), o+win)]))
	}
	return out
}

// event is a stretch of a recording well above its background.
type event struct {
	t0, t1, peak float64 // seconds, and dB
	f0, centroid float64 // Hz
}

// events finds the calls in x: stretches at least thr dB over the noise
// floor, joined across gaps shorter than 0.15 s.
func events(x []float64, thr float64) (floor float64, evs []event) {
	fr := frameRMS(x)
	s := slices.Clone(fr)
	slices.Sort(s)
	floor = db(s[len(s)/10])
	on := false
	var cur event
	gap := 0
	for i, v := range fr {
		t := float64(i) * 0.01
		loud := db(v) > floor+thr
		switch {
		case loud && !on:
			on, cur, gap = true, event{t0: t, peak: db(v)}, 0
		case loud:
			cur.peak = max(cur.peak, db(v))
			gap = 0
		case on:
			gap++
			if gap > 15 {
				cur.t1 = t - 0.15
				on = false
				if cur.t1-cur.t0 >= 0.08 {
					evs = append(evs, cur)
				}
			}
		}
	}
	if on {
		cur.t1 = float64(len(fr)) * 0.01
		evs = append(evs, cur)
	}
	for i := range evs {
		seg := cut(x, evs[i].t0, evs[i].t1)
		evs[i].f0 = pitchOf(seg)
		evs[i].centroid = centroid(seg)
	}
	return floor, evs
}

// pitchOf is the strongest period of x as a frequency, by autocorrelation
// of its loudest 100 ms, or 0 if it has none.
func pitchOf(x []float64) float64 {
	n := secs(0.1)
	if len(x) < n {
		return 0
	}
	best, bo := 0.0, 0
	for o := 0; o+n <= len(x); o += n / 4 {
		if r := rmsOf(x[o : o+n]); r > best {
			best, bo = r, o
		}
	}
	w := x[bo : bo+n]
	e := 0.0
	for _, v := range w {
		e += v * v
	}
	bestC, bestL := 0.0, 0
	dipped := false // past the first dip, so the peak at lag 0 is not taken
	for lag := rate / 2000; lag < rate/30 && lag < n/2; lag++ {
		c := 0.0
		for i := 0; i+lag < n; i++ {
			c += w[i] * w[i+lag]
		}
		if c < 0.2*e {
			dipped = true
		}
		if dipped && c > bestC {
			bestC, bestL = c, lag
		}
	}
	if bestL == 0 || bestC < 0.3*e {
		return 0
	}
	return float64(rate) / float64(bestL)
}

// centroid is the centre of mass of x's spectrum, in Hz.
func centroid(x []float64) float64 {
	frames := stft(x)
	s, w := 0.0, 0.0
	for _, f := range frames {
		for k, c := range f {
			m := cmplx.Abs(c)
			s += m * float64(k) * rate / stftN
			w += m
		}
	}
	if w == 0 {
		return 0
	}
	return s / w
}

func info(path string, x []float64) {
	floor, evs := events(x, 12)
	fmt.Printf("%s: %.2f s, peak %.1f dBFS, floor %.1f dB, %d calls\n", path, dur(x), db(peakOf(x)), floor, len(evs))
	for _, e := range evs {
		fmt.Printf("  %7.2f–%-7.2f %5.2f s  peak %6.1f dB (+%4.1f)  f0 %5.0f Hz  centroid %5.0f Hz\n",
			e.t0, e.t1, e.t1-e.t0, e.peak, e.peak-floor, e.f0, e.centroid)
	}
}

// spectrogram draws x from t0 to t1 up to fmax Hz: a level strip on top, the
// spectrum below with a line every second and every kHz.
func spectrogram(path string, x []float64, t0, t1, fmax float64) error {
	if t1 <= 0 || t1 > dur(x) {
		t1 = dur(x)
	}
	seg := cut(x, t0, t1)
	const n = 2048
	pxs := 120.0 // pixels a second
	if d := t1 - t0; d*pxs > 1800 {
		pxs = 1800 / d
	}
	w := max(1, int((t1-t0)*pxs))
	const h, top = 300, 60
	img := image.NewRGBA(image.Rect(0, 0, w, h+top))
	win := hann(n)
	maxBin := int(fmax / rate * n)
	cols := make([][]float64, w)
	ref := 0.0
	for px := range w {
		c := int(float64(px)/pxs*rate) - n/2
		f := make([]complex128, n)
		for i := range n {
			if k := c + i; k >= 0 && k < len(seg) {
				f[i] = complex(seg[k]*win[i], 0)
			}
		}
		fft(f, false)
		col := make([]float64, h)
		for y := range h {
			k0 := y * maxBin / h
			k1 := max(k0+1, (y+1)*maxBin/h)
			m := 0.0
			for k := k0; k < k1; k++ {
				m = max(m, cmplx.Abs(f[k]))
			}
			col[y] = m
			ref = max(ref, m)
		}
		cols[px] = col
	}
	for px, col := range cols {
		for y, m := range col {
			v := (db(m/ref) + 80) / 80 // 80 dB of range
			img.Set(px, top+h-1-y, heat(v))
		}
	}
	// The level strip.
	fr := frameRMS(seg)
	for px := range w {
		i := min(len(fr)-1, int(float64(px)/pxs*100))
		v := (db(fr[i]) + 60) / 60
		hh := int(max(0, min(1, v)) * (top - 4))
		for y := range top {
			c := color.RGBA{20, 20, 20, 255}
			if top-2-y < hh {
				c = color.RGBA{120, 200, 120, 255}
			}
			img.Set(px, y, c)
		}
	}
	for s := math.Ceil(t0); s < t1; s++ {
		px := int((s - t0) * pxs)
		for y := range h + top {
			if y%4 < 2 {
				img.Set(px, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	for k := 1000.0; k < fmax; k += 1000 {
		y := top + h - 1 - int(k/fmax*h)
		for px := 0; px < w; px += 3 {
			img.Set(px, y, color.RGBA{90, 160, 255, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// heat is a dark-to-bright colour for v in 0..1.
func heat(v float64) color.RGBA {
	v = max(0, min(1, v))
	r := min(1, v*2.2)
	g := max(0, min(1, v*2.2-0.7))
	b := max(0, min(1, v*3-2)) + 0.25*(1-v)*v*4
	return color.RGBA{uint8(255 * r), uint8(255 * g), uint8(255 * min(1, b)), 255}
}
