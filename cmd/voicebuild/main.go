// Command voicebuild makes the creatures' voices (internal/client/voices)
// from recordings of real animals, the way films make dinosaurs: calls
// slowed, pitched, filtered and layered over one another.
//
//	go run ./cmd/voicebuild render             # assets/voices/src -> internal/client/voices
//	go run ./cmd/voicebuild trim -raw DIR      # the raw downloads -> assets/voices/src
//	go run ./cmd/voicebuild info FILE...       # a recording's calls and levels
//	go run ./cmd/voicebuild spec FILE OUT.png [t0 t1 [fmax]]
//
// The raw downloads are not kept; assets/voices/SOURCES.md says where each
// came from, under what licence, and which part of it is used.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: voicebuild render|trim|info|spec ...")
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "info":
		for _, p := range args {
			x, e := load(p)
			if e != nil {
				fmt.Println(e)
				continue
			}
			info(p, x)
		}
	case "spec":
		if len(args) < 2 {
			err = fmt.Errorf("spec FILE OUT.png [t0 t1 [fmax]]")
			break
		}
		x, e := load(args[0])
		if e != nil {
			err = e
			break
		}
		t0, t1, fmax := 0.0, 0.0, 8000.0
		if len(args) >= 4 {
			t0, _ = strconv.ParseFloat(args[2], 64)
			t1, _ = strconv.ParseFloat(args[3], 64)
		}
		if len(args) >= 5 {
			fmax, _ = strconv.ParseFloat(args[4], 64)
		}
		err = spectrogram(args[1], x, t0, t1, fmax)
	case "trim":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		raw := fs.String("raw", "", "the folder of raw downloads")
		_ = fs.Parse(args)
		only := map[string]bool{}
		for _, a := range fs.Args() {
			only[a] = true
		}
		err = trim(*raw, only)
	case "compare":
		if len(args) != 3 {
			err = fmt.Errorf("compare OLD_DUMP NEW_DUMP OUT.wav")
			break
		}
		err = compare(args[0], args[1], args[2])
	case "levels":
		if len(args) != 2 {
			err = fmt.Errorf("levels OLD_DUMP NEW_DUMP")
			break
		}
		err = matchLevels(args[0], args[1])
	case "render":
		err = render(args)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "voicebuild:", err)
		os.Exit(1)
	}
}
