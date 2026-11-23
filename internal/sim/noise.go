package sim

import "math"

// hash2 maps a lattice point and a seed to a well-mixed 32-bit value.
func hash2(x, y int32, seed uint32) uint32 {
	h := uint32(x)*0x27d4eb2d ^ uint32(y)*0x165667b1 ^ seed*0x9e3779b1
	// murmur3 fmix32 finaliser to spread the bits.
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

// HashTile returns a stable pseudo-random value for a tile, for cosmetic
// variation (the client uses it to pick tile art variants).
func HashTile(tx, ty int) uint32 { return hash2(int32(tx), int32(ty), 0x7a11e) }

func unit(h uint32) float64 { return float64(h) / (1 << 32) }

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// valueNoise returns smooth noise in [0, 1) at (x, y).
func valueNoise(x, y float64, seed uint32) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int32(x0), int32(y0)
	sx, sy := smoothstep(x-x0), smoothstep(y-y0)
	v00 := unit(hash2(ix, iy, seed))
	v10 := unit(hash2(ix+1, iy, seed))
	v01 := unit(hash2(ix, iy+1, seed))
	v11 := unit(hash2(ix+1, iy+1, seed))
	return lerp(lerp(v00, v10, sx), lerp(v01, v11, sx), sy)
}

// fbm sums octaves of value noise; the result is in [0, 1).
func fbm(x, y float64, seed uint32, octaves int) float64 {
	sum, amp, norm, freq := 0.0, 1.0, 0.0, 1.0
	for i := 0; i < octaves; i++ {
		sum += amp * valueNoise(x*freq, y*freq, seed+uint32(i)*1013)
		norm += amp
		amp *= 0.5
		freq *= 2
	}
	return sum / norm
}
