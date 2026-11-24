// Command icon exports the application icon drawn by client.GameIcons: one
// PNG per size, a multi-size Windows .ico for the executable's resources, and
// preview sheets for checking how the small sizes read; with -syso, also the
// resource object that gives the game's executable its icon.
//
//	go run ./cmd/icon -o <dir>
//	go run ./cmd/icon -o <dir> -syso cmd/game/rsrc_windows_amd64.syso
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"survivalgame/internal/client"
)

func main() {
	dir := flag.String("o", "icon", "directory to write to")
	preview := flag.Bool("preview", true, "also write the preview sheets")
	syso := flag.String("syso", "", "also write the executable's icon resource here (cmd/game/rsrc_windows_amd64.syso)")
	flag.Parse()

	if *syso != "" {
		if err := os.WriteFile(*syso, encodeSyso(client.GameIcons()), 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s", *syso)
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		log.Fatal(err)
	}
	icons := client.GameIcons()
	for _, img := range icons {
		writePNG(filepath.Join(*dir, fmt.Sprintf("icon_%d.png", img.Bounds().Dx())), img)
	}
	ico := filepath.Join(*dir, "dino-isles.ico")
	if err := os.WriteFile(ico, encodeICO(icons), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", ico)
	if *preview {
		writePNG(filepath.Join(*dir, "preview_all_4x.png"), sheet(icons, 4, color.RGBA{0x6a, 0x6a, 0x72, 255}))
		writePNG(filepath.Join(*dir, "preview_small_8x.png"), taskbar(icons, 8))
		writePNG(filepath.Join(*dir, "preview_taskbar_1x.png"), taskbar(icons, 1))
	}
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", path)
}

// encodeICO packs the images into one .ico: 32-bit bitmaps, except that 256
// pixels and up are stored as PNG, which is what Windows expects of them.
func encodeICO(imgs []image.Image) []byte {
	var blobs [][]byte
	for _, img := range imgs {
		blobs = append(blobs, iconBlob(img))
	}
	var out bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&out, le, [3]uint16{0, 1, uint16(len(imgs))})
	offset := 6 + 16*len(imgs)
	for i, img := range imgs {
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		out.Write([]byte{byte(w), byte(h), 0, 0}) // 256 is written as 0
		binary.Write(&out, le, [2]uint16{1, 32})  // planes, bits per pixel
		binary.Write(&out, le, [2]uint32{uint32(len(blobs[i])), uint32(offset)})
		offset += len(blobs[i])
	}
	for _, b := range blobs {
		out.Write(b)
	}
	return out.Bytes()
}

// iconBlob is one image as an icon stores it: a bitmap, or PNG from 256
// pixels up.
func iconBlob(img image.Image) []byte {
	if img.Bounds().Dx() < 256 {
		return encodeDIB(img)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}

// encodeDIB is an icon's bitmap: a BITMAPINFOHEADER of twice the height, the
// BGRA rows bottom-up, then the 1-bit transparency mask.
func encodeDIB(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var out bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&out, le, struct {
		Size          uint32
		Width, Height int32
		Planes, Bits  uint16
		Rest          [6]uint32
	}{Size: 40, Width: int32(w), Height: int32(2 * h), Planes: 1, Bits: 32})
	maskRow := (w + 31) / 32 * 4
	mask := make([]byte, maskRow*h)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			out.Write([]byte{c.B, c.G, c.R, c.A})
			if c.A == 0 {
				mask[(h-1-y)*maskRow+x/8] |= 0x80 >> (x % 8)
			}
		}
	}
	out.Write(mask)
	return out.Bytes()
}

// scaled enlarges img k times with nearest neighbour.
func scaled(img image.Image, k int) image.Image {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := 0; y < b.Dy()*k; y++ {
		for x := 0; x < b.Dx()*k; x++ {
			out.Set(x, y, img.At(b.Min.X+x/k, b.Min.Y+y/k))
		}
	}
	return out
}

// sheet lays the icons out in a row, each enlarged k times.
func sheet(imgs []image.Image, k int, bg color.Color) image.Image {
	const gap = 16
	w, h := gap, 0
	for _, img := range imgs {
		w += img.Bounds().Dx()*k + gap
		h = max(h, img.Bounds().Dy()*k)
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h+2*gap))
	draw.Draw(out, out.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	x := gap
	for _, img := range imgs {
		s := scaled(img, k)
		draw.Draw(out, s.Bounds().Add(image.Pt(x, gap)), s, image.Point{}, draw.Over)
		x += s.Bounds().Dx() + gap
	}
	return out
}

// taskbar shows the sizes up to 48 on a dark and on a light bar, as they sit
// in the Windows taskbar and title bar, enlarged k times.
func taskbar(imgs []image.Image, k int) image.Image {
	var small []image.Image
	for _, img := range imgs {
		if img.Bounds().Dx() <= 48 {
			small = append(small, img)
		}
	}
	bars := []image.Image{
		sheet(small, k, color.RGBA{0x20, 0x20, 0x20, 255}),
		sheet(small, k, color.RGBA{0xf3, 0xf3, 0xf3, 255}),
	}
	b := bars[0].Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), 2*b.Dy()))
	for i, bar := range bars {
		draw.Draw(out, b.Add(image.Pt(0, i*b.Dy())), bar, image.Point{}, draw.Src)
	}
	return out
}
