package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	"survivalgame/internal/qoa"
)

// rate is the sample rate of everything here, the game's (client/audio.go).
const rate = 44100

// load reads a sound file (MP3, Ogg Vorbis, WAV or QOA) as mono at rate.
func load(path string) ([]float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var x []float64
	sr := 0
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		s, err := mp3.DecodeF32(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		x, sr, err = stereoF32(s, s.SampleRate())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	case ".ogg", ".oga":
		s, err := vorbis.DecodeF32(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		x, sr, err = stereoF32(s, s.SampleRate())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	case ".qoa":
		v, r, err := qoa.Decode(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		x, sr = make([]float64, len(v)), r
		for i, s := range v {
			x[i] = float64(s) / 32768
		}
	default:
		x, sr, err = readWAV(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	if sr != rate {
		x = resample(x, float64(sr)/rate)
	}
	return x, nil
}

// stereoF32 reads a stream of stereo float32 frames and mixes it to mono.
func stereoF32(r io.Reader, sr int) ([]float64, int, error) {
	b, err := io.ReadAll(r)
	if err != nil && len(b) == 0 {
		return nil, 0, err
	}
	n := len(b) / 8
	x := make([]float64, n)
	for i := range x {
		l := math.Float32frombits(binary.LittleEndian.Uint32(b[8*i:]))
		rr := math.Float32frombits(binary.LittleEndian.Uint32(b[8*i+4:]))
		x[i] = float64(l+rr) / 2
	}
	return x, sr, nil
}

// readWAV reads PCM (8, 16, 24 or 32 bits) or 32-bit float WAV, mixed to mono.
func readWAV(b []byte) ([]float64, int, error) {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, 0, errors.New("not a WAV file")
	}
	le := binary.LittleEndian
	var format, ch, bits, sr int
	p := 12
	for p+8 <= len(b) {
		id, size := string(b[p:p+4]), int(le.Uint32(b[p+4:]))
		body := b[p+8 : min(len(b), p+8+size)]
		switch id {
		case "fmt ":
			format, ch, sr, bits = int(le.Uint16(body)), int(le.Uint16(body[2:])), int(le.Uint32(body[4:])), int(le.Uint16(body[14:]))
			if format == 0xfffe && len(body) >= 26 {
				format = int(le.Uint16(body[24:]))
			}
		case "data":
			if ch == 0 {
				return nil, 0, errors.New("data before fmt")
			}
			bps := bits / 8
			frames := len(body) / (bps * ch)
			x := make([]float64, frames)
			for i := range frames {
				s := 0.0
				for c := range ch {
					o := (i*ch + c) * bps
					var v float64
					switch {
					case format == 3 && bits == 32:
						v = float64(math.Float32frombits(le.Uint32(body[o:])))
					case bits == 8:
						v = (float64(body[o]) - 128) / 128
					case bits == 16:
						v = float64(int16(le.Uint16(body[o:]))) / 32768
					case bits == 24:
						v = float64(int32(uint32(body[o])<<8|uint32(body[o+1])<<16|uint32(body[o+2])<<24)>>8) / 8388608
					case bits == 32:
						v = float64(int32(le.Uint32(body[o:]))) / 2147483648
					default:
						return nil, 0, fmt.Errorf("format %d, %d bits", format, bits)
					}
					s += v
				}
				x[i] = s / float64(ch)
			}
			return x, sr, nil
		}
		p += 8 + size + size&1
	}
	return nil, 0, errors.New("no data")
}

// writeWAV writes x as 16-bit mono PCM at rate.
func writeWAV(path string, x []float64) error {
	var b bytes.Buffer
	n := uint32(len(x))
	le := binary.LittleEndian
	b.WriteString("RIFF")
	b.Write(le.AppendUint32(nil, 36+2*n))
	b.WriteString("WAVEfmt ")
	b.Write(le.AppendUint32(nil, 16))
	b.Write(le.AppendUint16(nil, 1))
	b.Write(le.AppendUint16(nil, 1))
	b.Write(le.AppendUint32(nil, rate))
	b.Write(le.AppendUint32(nil, rate*2))
	b.Write(le.AppendUint16(nil, 2))
	b.Write(le.AppendUint16(nil, 16))
	b.WriteString("data")
	b.Write(le.AppendUint32(nil, 2*n))
	for _, v := range toS16(x) {
		b.Write(le.AppendUint16(nil, uint16(v)))
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// writeQOA writes x as QOA at rate.
func writeQOA(path string, x []float64, sr int) error {
	return os.WriteFile(path, qoa.Encode(toS16(x), sr), 0o644)
}

func toS16(x []float64) []int16 {
	out := make([]int16, len(x))
	for i, v := range x {
		out[i] = int16(math.Round(max(-1, min(1, v)) * 32767))
	}
	return out
}
