package main

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"slices"
)

// --- time and length --------------------------------------------------------------

func secs(t float64) int { return int(math.Round(t * rate)) }

func dur(x []float64) float64 { return float64(len(x)) / rate }

// cut is x from t0 to t1 seconds (t1 <= 0 means to the end), a copy.
func cut(x []float64, t0, t1 float64) []float64 {
	a := max(0, min(len(x), secs(t0)))
	b := len(x)
	if t1 > 0 {
		b = max(a, min(len(x), secs(t1)))
	}
	return slices.Clone(x[a:b])
}

// --- resampling -----------------------------------------------------------------

const (
	sincZeros = 24  // zero crossings each side of the kernel
	sincRes   = 512 // table points between two zero crossings
)

var sincTab = func() []float64 {
	n := sincZeros * sincRes
	t := make([]float64, n+2)
	for i := range t {
		x := float64(i) / sincRes
		s := 1.0
		if x > 0 {
			s = math.Sin(math.Pi*x) / (math.Pi * x)
		}
		// A Blackman-Harris window.
		u := min(1, 0.5+0.5*x/sincZeros)
		w := 0.35875 - 0.48829*math.Cos(2*math.Pi*u) + 0.14128*math.Cos(4*math.Pi*u) - 0.01168*math.Cos(6*math.Pi*u)
		t[i] = s * w
	}
	return t
}()

// kernel is the windowed sinc at distance d (in input samples) for a
// cutoff of fc (1 = Nyquist).
func kernel(d, fc float64) float64 {
	p := math.Abs(d) * fc * sincRes
	i := int(p)
	if i >= sincZeros*sincRes {
		return 0
	}
	f := p - float64(i)
	return (sincTab[i] + (sincTab[i+1]-sincTab[i])*f) * fc
}

// sampleAt is x read at fractional position t with cutoff fc.
func sampleAt(x []float64, t, fc float64) float64 {
	w := float64(sincZeros) / fc
	lo, hi := int(math.Ceil(t-w)), int(math.Floor(t+w))
	s := 0.0
	for k := max(0, lo); k <= min(len(x)-1, hi); k++ {
		s += x[k] * kernel(t-float64(k), fc)
	}
	return s
}

// resample plays x at speed step (2 = twice as fast, an octave up, half as
// long), band-limited so nothing aliases.
func resample(x []float64, step float64) []float64 {
	return varispeed(x, func(float64) float64 { return step })
}

// varispeed plays x at a speed that changes along the way: step(u) at the
// share u (0 to 1) of x already played.
func varispeed(x []float64, step func(u float64) float64) []float64 {
	var out []float64
	n := float64(len(x))
	for t := 0.0; t < n-1; {
		s := step(t / n)
		fc := min(1, 0.97/s)
		out = append(out, sampleAt(x, t, fc))
		t += s
	}
	return out
}

// pitch is x shifted by k (2 = an octave up) at the same length.
func pitch(x []float64, k float64) []float64 {
	return stretchTo(resample(x, k), len(x))
}

// --- WSOLA time stretch ---------------------------------------------------------------

// stretch makes x f times as long without changing its pitch.
func stretch(x []float64, f float64) []float64 {
	return stretchTo(x, int(float64(len(x))*f))
}

// stretchTo makes x n samples long, keeping its pitch: overlapping windows
// of it laid down at the new pace, each nudged to where it best matches what
// came before.
func stretchTo(x []float64, n int) []float64 {
	const win, hop, search = 2048, 512, 384
	if n <= 0 || len(x) < win+2*search {
		return resampleTo(x, n)
	}
	f := float64(len(x)-win) / float64(max(1, n-win)) // input samples per output sample
	out := make([]float64, n+win)
	nrm := make([]float64, n+win)
	w := hann(win)
	prev := -1 // where the last window was read from
	for o := 0; o < n; o += hop {
		at := int(float64(o) * f)
		if prev >= 0 {
			want := prev + hop // the natural continuation of the last window
			if want+win <= len(x) {
				best, bestC := at, math.Inf(-1)
				for d := -search; d <= search; d += 2 {
					p := at + d
					if p < 0 || p+win > len(x) {
						continue
					}
					c := 0.0
					for i := 0; i < win; i += 4 {
						c += x[p+i] * x[want+i]
					}
					if c > bestC {
						best, bestC = p, c
					}
				}
				at = best
			}
		}
		at = max(0, min(len(x)-win, at))
		for i := range win {
			out[o+i] += x[at+i] * w[i]
			nrm[o+i] += w[i]
		}
		prev = at
	}
	for i := range out {
		if nrm[i] > 1e-3 {
			out[i] /= nrm[i]
		}
	}
	return out[:n]
}

// resampleTo stretches x to n samples by changing its speed (and pitch).
func resampleTo(x []float64, n int) []float64 {
	if n <= 0 || len(x) < 2 {
		return make([]float64, max(0, n))
	}
	y := resample(x, float64(len(x))/float64(n))
	if len(y) < n {
		y = append(y, make([]float64, n-len(y))...)
	}
	return y[:n]
}

// --- filters ----------------------------------------------------------------------

type ftype int

const (
	lp ftype = iota
	hp
	bp
	peak
	lowshelf
	highshelf
)

// biquad is an RBJ cookbook filter.
type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func newBiquad(t ftype, f, q, gainDB float64) *biquad {
	f = max(10, min(rate*0.49, f))
	w := 2 * math.Pi * f / rate
	cs, sn := math.Cos(w), math.Sin(w)
	alpha := sn / (2 * q)
	A := math.Pow(10, gainDB/40)
	var b0, b1, b2, a0, a1, a2 float64
	switch t {
	case lp:
		b0, b1, b2 = (1-cs)/2, 1-cs, (1-cs)/2
		a0, a1, a2 = 1+alpha, -2*cs, 1-alpha
	case hp:
		b0, b1, b2 = (1+cs)/2, -(1 + cs), (1+cs)/2
		a0, a1, a2 = 1+alpha, -2*cs, 1-alpha
	case bp:
		b0, b1, b2 = alpha, 0, -alpha
		a0, a1, a2 = 1+alpha, -2*cs, 1-alpha
	case peak:
		b0, b1, b2 = 1+alpha*A, -2*cs, 1-alpha*A
		a0, a1, a2 = 1+alpha/A, -2*cs, 1-alpha/A
	case lowshelf:
		sq := 2 * math.Sqrt(A) * alpha
		b0 = A * ((A + 1) - (A-1)*cs + sq)
		b1 = 2 * A * ((A - 1) - (A+1)*cs)
		b2 = A * ((A + 1) - (A-1)*cs - sq)
		a0 = (A + 1) + (A-1)*cs + sq
		a1 = -2 * ((A - 1) + (A+1)*cs)
		a2 = (A + 1) + (A-1)*cs - sq
	case highshelf:
		sq := 2 * math.Sqrt(A) * alpha
		b0 = A * ((A + 1) + (A-1)*cs + sq)
		b1 = -2 * A * ((A - 1) + (A+1)*cs)
		b2 = A * ((A + 1) + (A-1)*cs - sq)
		a0 = (A + 1) - (A-1)*cs + sq
		a1 = 2 * ((A - 1) - (A+1)*cs)
		a2 = (A + 1) - (A-1)*cs - sq
	}
	return &biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}
}

func (f *biquad) do(x float64) float64 {
	y := f.b0*x + f.b1*f.x1 + f.b2*f.x2 - f.a1*f.y1 - f.a2*f.y2
	f.x2, f.x1, f.y2, f.y1 = f.x1, x, f.y1, y
	return y
}

// filter runs x through a biquad, returning a new slice.
func filter(x []float64, t ftype, f, q, gainDB float64) []float64 {
	b := newBiquad(t, f, q, gainDB)
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = b.do(v)
	}
	return out
}

// lowpass and highpass are two Butterworth sections: 24 dB an octave.
func lowpass(x []float64, f float64) []float64 {
	return filter(filter(x, lp, f, 0.5412, 0), lp, f, 1.3066, 0)
}

func highpass(x []float64, f float64) []float64 {
	return filter(filter(x, hp, f, 0.5412, 0), hp, f, 1.3066, 0)
}

// eq is a peaking boost or cut of db at f.
func eq(x []float64, f, q, db float64) []float64 { return filter(x, peak, f, q, db) }

// shelf boosts or cuts below (low) or above f.
func shelf(x []float64, low bool, f, db float64) []float64 {
	t := highshelf
	if low {
		t = lowshelf
	}
	return filter(x, t, f, 0.7071, db)
}

// --- levels and shapes --------------------------------------------------------------------

func peakOf(x []float64) float64 {
	m := 0.0
	for _, v := range x {
		m = max(m, math.Abs(v))
	}
	return m
}

func rmsOf(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(max(1, len(x))))
}

// norm scales x so its peak is p.
func norm(x []float64, p float64) []float64 {
	m := peakOf(x)
	out := make([]float64, len(x))
	if m == 0 {
		return out
	}
	for i, v := range x {
		out[i] = v * p / m
	}
	return out
}

// normRMS scales x so its loudest 50 ms has RMS r.
func normRMS(x []float64, r float64) []float64 {
	m := loudest(x, 0.05)
	if m == 0 {
		return slices.Clone(x)
	}
	return gain(x, r/m)
}

// loudest is the highest RMS of x over windows of w seconds.
func loudest(x []float64, w float64) float64 {
	n := max(1, secs(w))
	best := 0.0
	for o := 0; o+n <= len(x); o += n / 2 {
		best = max(best, rmsOf(x[o:o+n]))
	}
	if len(x) < n {
		best = rmsOf(x)
	}
	return best
}

func gain(x []float64, g float64) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = v * g
	}
	return out
}

// fade fades x in over a seconds and out over b, with quarter-sine curves.
func fade(x []float64, a, b float64) []float64 {
	out := slices.Clone(x)
	na, nb := min(len(x), secs(a)), min(len(x), secs(b))
	for i := range na {
		out[i] *= math.Sin(math.Pi / 2 * float64(i) / float64(na))
	}
	for i := range nb {
		out[len(x)-1-i] *= math.Sin(math.Pi / 2 * float64(i) / float64(nb))
	}
	return out
}

// shape multiplies x by an envelope through the points {u, gain}, u the
// share of x.
func shape(x []float64, pts ...[2]float64) []float64 {
	out := slices.Clone(x)
	n := float64(len(x))
	for i := range out {
		out[i] *= curve(pts, float64(i)/n)
	}
	return out
}

// curve is the line through the points {u, v} at u.
func curve(pts [][2]float64, u float64) float64 {
	if u <= pts[0][0] {
		return pts[0][1]
	}
	for i := 1; i < len(pts); i++ {
		if a, b := pts[i-1], pts[i]; u <= b[0] {
			return a[1] + (b[1]-a[1])*(u-a[0])/(b[0]-a[0])
		}
	}
	return pts[len(pts)-1][1]
}

// line is curve as a function.
func line(pts ...[2]float64) func(u float64) float64 {
	return func(u float64) float64 { return curve(pts, u) }
}

// drive pushes x into a soft clip: rougher, denser, louder.
func drive(x []float64, d float64) []float64 {
	m := peakOf(x)
	out := make([]float64, len(x))
	if m == 0 {
		return out
	}
	k := math.Tanh(d)
	for i, v := range x {
		out[i] = math.Tanh(d*v/m) / k * m
	}
	return out
}

// mixAt adds y into x at t seconds, times g, growing x if needed.
func mixAt(x, y []float64, t, g float64) []float64 {
	o := secs(t)
	if o < 0 {
		y = y[min(len(y), -o):]
		o = 0
	}
	if need := o + len(y); need > len(x) {
		x = append(x, make([]float64, need-len(x))...)
	}
	for i, v := range y {
		x[o+i] += v * g
	}
	return x
}

// rattle beats x with a throat's flutter: its level wobbling hz times a
// second (drifting a little) by depth.
func rattle(x []float64, hz, depth float64, r *rand.Rand) []float64 {
	out := make([]float64, len(x))
	ph, drift := 0.0, 0.0
	for i, v := range x {
		if i%400 == 0 {
			drift = drift*0.8 + (r.Float64()*2-1)*0.25
		}
		ph += hz * (1 + drift*0.3) / rate
		out[i] = v * (1 - depth*0.5*(1+math.Sin(2*math.Pi*ph)))
	}
	return out
}

// follow is x's envelope: its level smoothed over about ms milliseconds.
func follow(x []float64, ms float64) []float64 {
	rel := math.Exp(-1 / (ms / 1000 * rate))
	att := math.Exp(-1 / (0.004 * rate))
	out := make([]float64, len(x))
	e := 0.0
	for i, v := range x {
		v = math.Abs(v)
		if v > e {
			e = v + (e-v)*att
		} else {
			e = v + (e-v)*rel
		}
		out[i] = e
	}
	return out
}

// subTone is a deep tone gliding through f (Hz along the share of the
// sound) that swells and dies with env, the envelope of what it lies under.
func subTone(env []float64, f func(u float64) float64) []float64 {
	out := make([]float64, len(env))
	ph := 0.0
	n := float64(len(env))
	for i, e := range env {
		ph += f(float64(i)/n) / rate
		out[i] = (math.Sin(2*math.Pi*ph) + 0.25*math.Sin(4*math.Pi*ph+0.5)) * e
	}
	return out
}

// noise is d seconds of white noise.
func noise(r *rand.Rand, d float64) []float64 {
	out := make([]float64, secs(d))
	for i := range out {
		out[i] = r.Float64()*2 - 1
	}
	return out
}

func reverse(x []float64) []float64 {
	out := slices.Clone(x)
	slices.Reverse(out)
	return out
}

// envOnto shapes y (noise, a tone) by x's envelope, so it rises and falls
// with x.
func envOnto(x, y []float64, ms float64) []float64 {
	e := follow(x, ms)
	out := make([]float64, len(x))
	for i := range out {
		if i < len(y) {
			out[i] = y[i] * e[i]
		}
	}
	return out
}

// --- space -----------------------------------------------------------------------

// reverb adds a diffuse tail of decay seconds (to -60 dB) at wet, darker as
// it goes by damp (0 to 1): as off rock and trees.
func reverb(x []float64, decay, wet, damp float64) []float64 {
	tail := secs(decay)
	out := make([]float64, len(x)+tail)
	copy(out, x)
	combs := []float64{0.0297, 0.0371, 0.0411, 0.0437, 0.0513, 0.0587}
	sum := make([]float64, len(out))
	for _, d := range combs {
		n := secs(d)
		g := math.Pow(10, -3*d/decay)
		buf := make([]float64, n)
		lps := 0.0
		j := 0
		for i := range out {
			in := 0.0
			if i < len(x) {
				in = x[i]
			}
			y := buf[j]
			lps = y*(1-damp) + lps*damp
			buf[j] = in + lps*g
			j = (j + 1) % n
			sum[i] += y
		}
	}
	for _, d := range []float64{0.005, 0.0017} {
		n := secs(d)
		buf := make([]float64, n)
		j := 0
		for i := range sum {
			b := buf[j]
			y := -0.7*sum[i] + b
			buf[j] = sum[i] + 0.7*y
			j = (j + 1) % n
			sum[i] = y
		}
	}
	g := wet / math.Sqrt(float64(len(combs)))
	for i := range out {
		out[i] += sum[i] * g
	}
	return out
}

// trimSilence drops the quiet before and after x's sound (below thr times
// its peak), keeping pad seconds each side.
func trimSilence(x []float64, thr, pad float64) []float64 {
	e := follow(x, 20)
	m := peakOf(e) * thr
	a, b := 0, len(x)
	for a < len(x) && e[a] < m {
		a++
	}
	for b > a && e[b-1] < m {
		b--
	}
	a, b = max(0, a-secs(pad)), min(len(x), b+secs(pad))
	return slices.Clone(x[a:b])
}

// --- spectra -----------------------------------------------------------------------

func fft(a []complex128, inverse bool) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for l := 2; l <= n; l <<= 1 {
		ang := -2 * math.Pi / float64(l)
		if inverse {
			ang = -ang
		}
		wl := cmplx.Rect(1, ang)
		for i := 0; i < n; i += l {
			w := complex(1, 0)
			for k := range l / 2 {
				u, v := a[i+k], a[i+k+l/2]*w
				a[i+k], a[i+k+l/2] = u+v, u-v
				w *= wl
			}
		}
	}
	if inverse {
		for i := range a {
			a[i] /= complex(float64(n), 0)
		}
	}
}

const stftN, stftHop = 2048, 512

func hann(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n))
	}
	return w
}

// stft is x's short-time spectra, a frame every stftHop samples.
func stft(x []float64) [][]complex128 {
	w := hann(stftN)
	var frames [][]complex128
	for o := -stftN / 2; o < len(x); o += stftHop {
		f := make([]complex128, stftN)
		for i := range stftN {
			if k := o + i; k >= 0 && k < len(x) {
				f[i] = complex(x[k]*w[i], 0)
			}
		}
		fft(f, false)
		frames = append(frames, f[:stftN/2+1])
	}
	return frames
}

// istft puts spectra back together as n samples.
func istft(frames [][]complex128, n int) []float64 {
	w := hann(stftN)
	out := make([]float64, n+len(frames)*stftHop+2*stftN)
	nrm := make([]float64, len(out))
	full := make([]complex128, stftN)
	for fi, f := range frames {
		copy(full, f)
		for k := 1; k < stftN/2; k++ {
			full[stftN-k] = cmplx.Conj(f[k])
		}
		fft(full, true)
		o := fi*stftHop - stftN/2 + stftN
		for i := range stftN {
			out[o+i] += real(full[i]) * w[i]
			nrm[o+i] += w[i] * w[i]
		}
	}
	res := make([]float64, n)
	for i := range res {
		if v := nrm[i+stftN]; v > 1e-6 {
			res[i] = out[i+stftN] / v
		}
	}
	return res
}

// denoise takes the steady background out of x: each band is turned down
// where it is not well above that band's level in the quietest frames.
// amount is how hard (1 = to the noise floor), floor the least gain kept.
func denoise(x []float64, amount, floor float64) []float64 {
	return denoiseWith(x, noiseProfile(x), amount, floor)
}

// noiseProfile is the level of x's background in each band: its
// 20th-percentile magnitude over the frames.
func noiseProfile(x []float64) []float64 {
	frames := stft(x)
	bins := stftN/2 + 1
	nf := make([]float64, bins)
	if len(frames) == 0 {
		return nf
	}
	col := make([]float64, len(frames))
	for k := range bins {
		for i, f := range frames {
			col[i] = cmplx.Abs(f[k])
		}
		slices.Sort(col)
		nf[k] = col[len(col)*20/100]
	}
	return nf
}

// denoiseWith is denoise against the background nf.
func denoiseWith(x, nf []float64, amount, floor float64) []float64 {
	frames := stft(x)
	if len(frames) < 4 {
		return slices.Clone(x)
	}
	bins := stftN/2 + 1
	gains := make([][]float64, len(frames))
	for i, f := range frames {
		g := make([]float64, bins)
		for k := range bins {
			m := cmplx.Abs(f[k])
			n := nf[k] * amount * 2
			v := 0.0
			if m > 0 {
				v = max(0, 1-(n*n)/(m*m))
			}
			g[k] = max(floor, math.Sqrt(v))
		}
		gains[i] = g
	}
	// Smooth the gains across neighbouring frames and bands, so what noise
	// is left does not twinkle.
	for i, f := range frames {
		for k := range bins {
			s, c := 0.0, 0.0
			for di := -2; di <= 2; di++ {
				for dk := -2; dk <= 2; dk++ {
					ii, kk := i+di, k+dk
					if ii < 0 || ii >= len(frames) || kk < 0 || kk >= bins {
						continue
					}
					s += gains[ii][kk]
					c++
				}
			}
			f[k] *= complex(min(1, 0.5*gains[i][k]+0.5*s/c), 0)
		}
	}
	return istft(frames, len(x))
}

func pow10(x float64) float64 { return math.Pow(10, x) }

func sin2pi(x float64) float64 { return math.Sin(2 * math.Pi * x) }
