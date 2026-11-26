package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// writeSources writes assets/voices/SOURCES.md: every recording, where it
// came from and under what licence, the clips cut from it and the voices
// each clip goes into.
func writeSources() error {
	by := usedBy()
	var b strings.Builder
	b.WriteString(`# Where the creatures' voices come from

The creatures' voices (internal/client/voices) are made by ` + "`go run ./cmd/voicebuild render`" + `
from the clips in [src](src), which ` + "`go run ./cmd/voicebuild trim -raw DIR`" + ` cuts out of the
recordings below, downloaded into DIR. The recordings themselves are not kept here.

Only recordings that may be sold inside a game are used: works of the U.S. government
(public domain), recordings their authors put in the public domain (CC0 or a public-domain
mark), and CC BY recordings, credited in [CREDITS.md](../../CREDITS.md). No CC BY-NC, CC BY-ND
or CC BY-SA recording, nothing from xeno-canto or the BBC, nothing from a game or a film; Freesound's CC0 recordings only by way of their copies on Wikimedia Commons.

This file is written by ` + "`voicebuild render`" + `; edit cmd/voicebuild/sources.go instead.

## Recordings

| File | What | Author | Licence | Fetched |
|---|---|---|---|---|
`)
	for _, s := range sources {
		fmt.Fprintf(&b, "| [%s](%s) ([page](%s)) | %s | %s | %s | %s |\n", filepath.Base(s.file), s.url, s.page, s.what, s.author, s.licence, s.fetched)
	}
	b.WriteString("\n## Clips\n\nEach clip is a part of one recording, cleaned of its background, and the voices made from it.\n\n| Clip | From | Seconds | Used for |\n|---|---|---|---|\n")
	for _, c := range clips {
		used := by[c.name]
		if len(used) == 0 {
			used = []string{"(nothing yet)"}
		}
		fmt.Fprintf(&b, "| %s | %s | %.2f–%.2f | %s |\n", c.name, filepath.Base(c.src), c.t0, c.t1, strings.Join(used, ", "))
	}
	return os.WriteFile(filepath.Join("assets", "voices", "SOURCES.md"), []byte(b.String()), 0o644)
}

// writeCredits writes CREDITS.md: the recordings whose licence asks to be
// credited, and thanks to the rest.
func writeCredits() error {
	var b strings.Builder
	b.WriteString("# Credits\n\n## Sounds\n\nThe creatures' voices are made from recordings of real animals (see\n[assets/voices/SOURCES.md](assets/voices/SOURCES.md) for every recording, its licence and its use).\n\n")
	b.WriteString("These recordings are used under the Creative Commons Attribution licence, and changed\n(cut, cleaned, slowed, filtered and mixed with others):\n\n")
	for _, s := range sources {
		if strings.HasPrefix(s.licence, "CC BY") {
			fmt.Fprintf(&b, "- \"%s\" (%s), by %s, %s (<https://creativecommons.org/licenses/by/4.0/>), <%s>\n",
				s.what, filepath.Base(s.file), strings.TrimSuffix(s.author, ", via Wikimedia Commons"), s.licence, s.page)
		}
	}
	b.WriteString("\nWith thanks for the public-domain and CC0 recordings to the U.S. National Park Service,\nthe U.S. National Oceanic and Atmospheric Administration, the U.S. Fish and Wildlife Service,\nand to ")
	var who []string
	for _, s := range sources {
		if strings.HasSuffix(s.author, "via Wikimedia Commons") && !strings.HasPrefix(s.licence, "CC BY") && !strings.Contains(s.author, "U.S.") {
			a := strings.TrimSuffix(s.author, ", via Wikimedia Commons")
			if i := strings.Index(a, " (Freesound"); i > 0 {
				a = a[:i]
			}
			if !slices.Contains(who, a) {
				who = append(who, a)
			}
		}
	}
	b.WriteString(strings.Join(who, ", ") + " on Wikimedia Commons.\n")
	return os.WriteFile("CREDITS.md", []byte(b.String()), 0o644)
}
