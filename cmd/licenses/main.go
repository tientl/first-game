// Command licenses writes LICENSES.txt: the licence of everything the game
// ships that is not ours, word for word. The Go modules the game binary
// links are found with `go list` and their LICENSE files read from the module
// cache, so the file follows go.mod. Run it from the repository root after
// changing a dependency:
//
//	go run ./cmd/licenses
//
// A test (internal/client/credits_test.go) fails if LICENSES.txt leaves a
// linked module out.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const rule = "================================================================================\n"

func main() {
	out := flag.String("o", "LICENSES.txt", "file to write")
	goos := flag.String("goos", "windows", "the platform the game is released for")
	flag.Parse()

	var b bytes.Buffer
	b.WriteString("Dino Isles: licences of the parts that are not ours\n" + rule +
		"\nThis file holds the licence of every font, recording, library and recipe the game\n" +
		"is built from, word for word. Who made what is on the \"Ghi công\" screen of the\n" +
		"main menu (internal/client/credits.go) and in CREDITS.md.\n" +
		"\nContents\n  1. Sound recordings: Creative Commons Attribution 4.0 (CC BY 4.0)\n" +
		"  2. Font: Nunito, SIL Open Font License 1.1\n" +
		"  3. Effects: pixelfx, MIT\n" +
		"  4. The Go programming language and its standard library, BSD 3-Clause\n" +
		"  5. Go modules linked into the game, each with its own licence (Apache 2.0, MIT,\n" +
		"     BSD 3-Clause)\n\n")

	section(&b, "1. Sound recordings: CC BY 4.0")
	b.WriteString(ccby)

	section(&b, "2. Nunito (internal/client/fonts/OFL.txt)")
	b.WriteString(read("internal/client/fonts/OFL.txt"))

	section(&b, "3. pixelfx (internal/client/LICENSE-pixelfx.txt)")
	b.WriteString(read("internal/client/LICENSE-pixelfx.txt"))

	section(&b, "4. Go (GOROOT/LICENSE), also the licence of Go Bold in golang.org/x/image")
	b.WriteString(read(filepath.Join(strings.TrimSpace(run(nil, "go", "env", "GOROOT")), "LICENSE")))

	section(&b, "5. Go modules linked into the game binary")
	type mod struct{ path, version, dir string }
	var mods []mod
	list := run([]string{"GOOS=" + *goos, "GOARCH=amd64", "CGO_ENABLED=0"}, "go", "list", "-deps",
		"-f", "{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}{{end}}", "./cmd/game")
	seen := map[string]bool{}
	for _, l := range strings.Split(list, "\n") {
		f := strings.Split(strings.TrimSpace(l), "|")
		if len(f) != 3 || seen[f[0]] {
			continue
		}
		seen[f[0]] = true
		mods = append(mods, mod{f[0], f[1], f[2]})
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].path < mods[j].path })
	for _, m := range mods {
		files, _ := filepath.Glob(filepath.Join(m.dir, "*"))
		n := 0
		for _, f := range files {
			base := strings.ToUpper(filepath.Base(f))
			if !(strings.HasPrefix(base, "LICENSE") || strings.HasPrefix(base, "LICENCE") ||
				strings.HasPrefix(base, "COPYING") || strings.HasPrefix(base, "NOTICE")) {
				continue
			}
			if fi, err := os.Stat(f); err != nil || fi.IsDir() {
				continue
			}
			n++
			fmt.Fprintf(&b, "%s %s (%s)\n%s\n", m.path, m.version, filepath.Base(f), strings.Repeat("-", 80))
			b.WriteString(read(f))
			b.WriteString("\n\n")
		}
		if n == 0 {
			log.Fatalf("%s %s has no licence file in the module cache", m.path, m.version)
		}
	}
	text := strings.ReplaceAll(b.String(), "\r\n", "\n")
	if err := os.WriteFile(*out, []byte(text), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %d modules, %d bytes\n", *out, len(mods), len(text))
}

func section(b *bytes.Buffer, title string) {
	fmt.Fprintf(b, "\n%s%s\n%s\n", rule, title, rule)
}

func read(path string) string {
	d, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimRight(string(d), "\r\n\t ") + "\n"
}

func run(env []string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
	}
	return string(out)
}

// ccby is the notice for the CC BY recordings. The licence's full text is at
// the address given; it is not copied here.
const ccby = `Some of the creatures' voices are made from recordings licensed under the Creative
Commons Attribution 4.0 International licence (CC BY 4.0). The recordings were
cut, cleaned, slowed, filtered and mixed with others (changes were made).

  "an Australasian bittern booming" (Australasian-bittern.wav)
      by Department of Conservation (NZ), CC BY 4.0
      https://commons.wikimedia.org/wiki/File:Australasian-bittern.wav

  "New Zealand fur seals growling, Stephens Island" (Arctocephalus_forsteri_-_sound.ogg)
      by Department of Conservation (NZ), CC BY 4.0
      https://commons.wikimedia.org/wiki/File:Arctocephalus_forsteri_-_sound.ogg

Licence summary: https://creativecommons.org/licenses/by/4.0/
Full legal text:  https://creativecommons.org/licenses/by/4.0/legalcode

You are free to share and adapt the material for any purpose, even commercially,
under these terms: you must give appropriate credit, provide a link to the licence,
and indicate if changes were made; you may not add legal terms or technological
measures that legally restrict others from doing anything the licence permits.
The material is provided as is, without warranties.

The other recordings are public domain (works of the U.S. government, or marked so
by their authors) or CC0 1.0 (https://creativecommons.org/publicdomain/zero/1.0/);
they need no licence text. Each is credited on the "Ghi công" screen.
`
