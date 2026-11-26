// Command dinopaint paints the HD dinosaurs over their density-1 drawings.
//
//	go run ./cmd/dinopaint rex trike      # writes internal/client/dinos/hd/rex.png, trike.png
//	go run ./cmd/dinopaint -debug dir rex # also the maps it was painted from, enlarged
//
// See internal/client/dinos/hd/README.md for how to add a species.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"survivalgame/internal/dinopaint"
)

func main() {
	in := flag.String("guide", "internal/client/dinos", "folder of the density-1 drawings")
	out := flag.String("o", "internal/client/dinos/hd", "folder to write the paintings to")
	debug := flag.String("debug", "", "folder to write the material, part and light maps to")
	flag.Parse()
	names := flag.Args()
	if len(names) == 0 || (len(names) == 1 && names[0] == "all") {
		names = dinopaint.Names()
	}
	for _, n := range names {
		sp, ok := dinopaint.Lookup(n)
		if !ok {
			fmt.Fprintf(os.Stderr, "dinopaint: no description of %q (have %v)\n", n, dinopaint.Names())
			os.Exit(1)
		}
		g, err := load(filepath.Join(*in, sp.Name+".png"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "dinopaint:", err)
			os.Exit(1)
		}
		img, p := dinopaint.Paint(sp, g)
		if err := save(filepath.Join(*out, sp.Name+".png"), img); err != nil {
			fmt.Fprintln(os.Stderr, "dinopaint:", err)
			os.Exit(1)
		}
		fmt.Println(p.Report())
		if legs, ok := dinopaint.LookupLegs(sp.Name); ok {
			mask := dinopaint.PaintLegs(legs, p)
			under := dinopaint.PaintLegsUnder(sp, legs, g, p, mask)
			for name, img := range map[string]image.Image{"_legs.png": mask, "_legsunder.png": under} {
				if err := save(filepath.Join(*out, sp.Name+name), img); err != nil {
					fmt.Fprintln(os.Stderr, "dinopaint:", err)
					os.Exit(1)
				}
			}
		}
		if t, ok := dinopaint.LookupThighs(sp.Name); ok {
			mask, under, body := dinopaint.PaintThighs(sp, t, g, p)
			for name, img := range map[string]image.Image{"_legs.png": mask, "_legsunder.png": under, "_legsbody.png": body} {
				if err := save(filepath.Join(*out, sp.Name+name), img); err != nil {
					fmt.Fprintln(os.Stderr, "dinopaint:", err)
					os.Exit(1)
				}
			}
		}
		if u, ok := dinopaint.LookupUnder(sp.Name); ok {
			if err := save(filepath.Join(*out, sp.Name+"_under.png"), dinopaint.PaintUnder(sp, u, g, p)); err != nil {
				fmt.Fprintln(os.Stderr, "dinopaint:", err)
				os.Exit(1)
			}
		}
		if *debug != "" {
			for name, m := range debugMaps(p) {
				if err := save(filepath.Join(*debug, sp.Name+"_"+name+".png"), m); err != nil {
					fmt.Fprintln(os.Stderr, "dinopaint:", err)
				}
			}
		}
	}
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// debugMaps are the maps a painting is made from, as pictures.
func debugMaps(p *dinopaint.Painter) map[string]image.Image {
	pal := []color.NRGBA{{230, 80, 60, 255}, {60, 160, 230, 255}, {240, 200, 60, 255}, {120, 210, 110, 255},
		{200, 110, 220, 255}, {90, 220, 210, 255}, {250, 140, 40, 255}, {150, 150, 150, 255}}
	mat := image.NewNRGBA(image.Rect(0, 0, p.W, p.H))
	part := image.NewNRGBA(image.Rect(0, 0, p.W, p.H))
	light := image.NewNRGBA(image.Rect(0, 0, p.W, p.H))
	for i, in := range p.In {
		if !in {
			continue
		}
		x, y := i%p.W, i/p.W
		mat.SetNRGBA(x, y, pal[p.Mat[i]%len(pal)])
		if p.Part[i] >= 0 {
			part.SetNRGBA(x, y, pal[p.Part[i]%len(pal)])
		}
		v := uint8(min(255, max(0, p.Val[i]*255)))
		light.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
	}
	return map[string]image.Image{"mat": mat, "part": part, "light": light}
}
