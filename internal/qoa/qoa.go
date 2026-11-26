// Package qoa reads and writes the "Quite OK Audio" format (qoaformat.org):
// lossy, 3.2 bits a sample, decoded with a few integer operations a sample.
// The game keeps its recorded voices in it (internal/client/voices).
package qoa

import (
	"encoding/binary"
	"errors"
	"math"
)

const (
	magic          = 0x716f6166 // "qoaf"
	sliceLen       = 20
	slicesPerFrame = 256
	frameLen       = sliceLen * slicesPerFrame
)

var scalefactors = [16]int{1, 7, 21, 45, 84, 138, 211, 304, 421, 562, 731, 928, 1157, 1419, 1715, 2048}

// quantTab maps a residual divided by the scalefactor, -8..8, to its code.
var quantTab = [17]int{7, 7, 7, 5, 5, 3, 3, 1, 0, 0, 2, 2, 4, 4, 6, 6, 6}

var (
	reciprocals [16]int
	dequantTab  [16][8]int
)

func init() {
	steps := [8]float64{0.75, -0.75, 2.5, -2.5, 4.5, -4.5, 7, -7}
	for s, sf := range scalefactors {
		reciprocals[s] = ((1 << 16) + sf - 1) / sf
		for q, d := range steps {
			dequantTab[s][q] = int(math.Round(float64(sf) * d))
		}
	}
}

type lms struct{ history, weights [4]int }

func (l *lms) predict() int {
	p := 0
	for i := range 4 {
		p += l.weights[i] * l.history[i]
	}
	return p >> 13
}

func (l *lms) update(sample, residual int) {
	delta := residual >> 4
	for i := range 4 {
		if l.history[i] < 0 {
			l.weights[i] -= delta
		} else {
			l.weights[i] += delta
		}
	}
	l.history[0], l.history[1], l.history[2], l.history[3] = l.history[1], l.history[2], l.history[3], sample
}

func clampS16(v int) int { return max(-32768, min(32767, v)) }

func div(v, sf int) int {
	n := (v*reciprocals[sf] + (1 << 15)) >> 16
	return n + sign(v) - sign(n) // round away from zero
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// Encode compresses mono 16-bit samples at the given sample rate.
func Encode(samples []int16, rate int) []byte {
	out := binary.BigEndian.AppendUint64(nil, uint64(magic)<<32|uint64(len(samples)))
	st := lms{weights: [4]int{0, 0, -(1 << 13), 1 << 14}}
	prevSF := 0
	for start := 0; start < len(samples); start += frameLen {
		n := min(frameLen, len(samples)-start)
		slices := (n + sliceLen - 1) / sliceLen
		size := 8 + 16 + 8*slices
		out = binary.BigEndian.AppendUint64(out, 1<<56|uint64(rate)<<32|uint64(n)<<16|uint64(size))
		var h, w uint64
		for i := range 4 {
			h = h<<16 | uint64(uint16(int16(st.history[i])))
			w = w<<16 | uint64(uint16(int16(st.weights[i])))
		}
		out = binary.BigEndian.AppendUint64(out, h)
		out = binary.BigEndian.AppendUint64(out, w)
		for s := start; s < start+n; s += sliceLen {
			sl := min(sliceLen, start+n-s)
			bestRank, best, bestSF := uint64(math.MaxUint64), uint64(0), 0
			var bestLMS lms
			for i := range 16 {
				sf := (i + prevSF) % 16
				l := st
				slice := uint64(sf)
				rank := uint64(0)
				for _, v := range samples[s : s+sl] {
					p := l.predict()
					q := quantTab[max(-8, min(8, div(int(v)-p, sf)))+8]
					dq := dequantTab[sf][q]
					rec := clampS16(p + dq)
					pen := (l.weights[0]*l.weights[0]+l.weights[1]*l.weights[1]+l.weights[2]*l.weights[2]+l.weights[3]*l.weights[3])>>18 - 0x8ff
					pen = max(0, pen)
					e := int64(int(v) - rec)
					rank += uint64(e*e) + uint64(pen*pen)
					if rank > bestRank {
						break
					}
					l.update(rec, dq)
					slice = slice<<3 | uint64(q)
				}
				if rank < bestRank {
					bestRank, best, bestLMS, bestSF = rank, slice, l, sf
				}
			}
			prevSF = bestSF
			st = bestLMS
			out = binary.BigEndian.AppendUint64(out, best<<(3*(sliceLen-sl)))
		}
	}
	return out
}

// Decode decompresses a mono QOA file: its samples and sample rate.
func Decode(b []byte) ([]int16, int, error) {
	if len(b) < 8 || binary.BigEndian.Uint32(b) != magic {
		return nil, 0, errors.New("qoa: not a QOA file")
	}
	total := int(binary.BigEndian.Uint32(b[4:]))
	out := make([]int16, 0, total)
	p, rate := 8, 0
	for len(out) < total {
		if p+24 > len(b) {
			return nil, 0, errors.New("qoa: truncated")
		}
		hdr := binary.BigEndian.Uint64(b[p:])
		ch, r, n, size := int(hdr>>56), int(hdr>>32&0xffffff), int(hdr>>16&0xffff), int(hdr&0xffff)
		if ch != 1 {
			return nil, 0, errors.New("qoa: only mono is supported")
		}
		if rate == 0 {
			rate = r
		}
		slices := (n + sliceLen - 1) / sliceLen
		if n == 0 || n > frameLen || size != 24+8*slices || p+size > len(b) || len(out)+n > total {
			return nil, 0, errors.New("qoa: bad frame")
		}
		var l lms
		h, w := binary.BigEndian.Uint64(b[p+8:]), binary.BigEndian.Uint64(b[p+16:])
		for i := range 4 {
			l.history[i] = int(int16(h >> (48 - 16*i)))
			l.weights[i] = int(int16(w >> (48 - 16*i)))
		}
		p += 24
		for s := 0; s < n; s += sliceLen {
			slice := binary.BigEndian.Uint64(b[p:])
			p += 8
			sf := int(slice >> 60)
			slice <<= 4
			for range min(sliceLen, n-s) {
				pr := l.predict()
				dq := dequantTab[sf][slice>>61]
				rec := clampS16(pr + dq)
				out = append(out, int16(rec))
				slice <<= 3
				l.update(rec, dq)
			}
		}
	}
	return out, rate, nil
}
