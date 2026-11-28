package client

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// The credits table must hold every recording in assets/voices/SOURCES.md and
// CREDITS.md, so a new sound cannot be forgotten.
func TestCreditsHoldEverySound(t *testing.T) {
	have := map[string]soundCredit{}
	for _, s := range creditSounds {
		if s.File == "" || s.What == "" || s.Author == "" || s.Licence == "" || s.URL == "" {
			t.Errorf("incomplete credit: %+v", s)
		}
		have[s.File] = s
	}
	src, err := os.ReadFile("../../assets/voices/SOURCES.md")
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile(`(?m)^\| \[([^\]]+)\]\([^)]*\)(?: \(\[page\]\([^|]*\)\))? \| ([^|]*) \| ([^|]*) \| ([^|]*) \|`)
	rows := row.FindAllStringSubmatch(string(src), -1)
	if len(rows) < 30 {
		t.Fatalf("parsed only %d recordings from SOURCES.md", len(rows))
	}
	for _, m := range rows {
		s, ok := have[m[1]]
		if !ok {
			t.Errorf("SOURCES.md recording %s is not in creditSounds (credits.go)", m[1])
			continue
		}
		if s.Author != strings.TrimSpace(m[3]) || s.Licence != strings.TrimSpace(m[4]) {
			t.Errorf("%s: credits say %q / %q, SOURCES.md says %q / %q", m[1], s.Author, s.Licence, m[3], m[4])
		}
		delete(have, m[1])
	}
	for f := range have {
		t.Errorf("creditSounds has %s, which SOURCES.md does not", f)
	}

	// Every file CREDITS.md names, "(file.ext)", is in the table too.
	md, err := os.ReadFile("../../CREDITS.md")
	if err != nil {
		t.Fatal(err)
	}
	named := regexp.MustCompile(`\(([\w.\-]+\.(?:wav|ogg|mp3))\)`).FindAllStringSubmatch(string(md), -1)
	if len(named) == 0 {
		t.Fatal("no sound files named in CREDITS.md")
	}
	all := map[string]bool{}
	for _, s := range creditSounds {
		all[s.File] = true
	}
	for _, m := range named {
		if !all[m[1]] {
			t.Errorf("CREDITS.md names %s, not in creditSounds", m[1])
		}
	}
	// The CC BY ones are listed on the screen, with their licence.
	text := creditsText()
	for _, s := range creditSounds {
		if s.Licence == "CC BY 4.0" && !strings.Contains(text, s.File) {
			t.Errorf("CC BY recording %s is not on the credits screen", s.File)
		}
	}
}

func creditsText() string {
	var b strings.Builder
	for _, sec := range creditSections() {
		b.WriteString(sec.Title + "\n" + sec.Intro + "\n")
		for _, e := range sec.Entries {
			b.WriteString(e.Name + "\n" + e.By + "\n" + e.Licence + "\n" + e.URL + "\n")
		}
	}
	return b.String()
}

// linkedModules are the Go modules ./cmd/game links, as `go list` finds them.
func linkedModules(t *testing.T) (paths, versions []string) {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", "-f",
		"{{if .Module}}{{if not .Module.Main}}{{.Module.Path}} {{.Module.Version}}{{end}}{{end}}", "./cmd/game")
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}
	seen := map[string]bool{}
	for _, l := range strings.Split(string(out), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && !seen[f[0]] {
			seen[f[0]] = true
			paths, versions = append(paths, f[0]), append(versions, f[1])
		}
	}
	if len(paths) == 0 {
		t.Fatal("no linked modules found")
	}
	return paths, versions
}

// Every Go module the game links is on the credits screen with a licence.
func TestCreditsHoldEveryModule(t *testing.T) {
	paths, _ := linkedModules(t)
	for _, p := range paths {
		found := false
		for _, e := range creditLibraries {
			if strings.Contains(e.Name, p) && e.Licence != "" {
				found = true
			}
		}
		if !found {
			t.Errorf("linked module %s is not in creditLibraries (credits.go)", p)
		}
	}
}

// LICENSES.txt names every linked module and its version, and holds the
// texts of the licences the game's parts come under.
func TestLicensesFileHoldsEveryModule(t *testing.T) {
	b, err := os.ReadFile("../../LICENSES.txt")
	if err != nil {
		t.Fatal(err)
	}
	txt := string(b)
	paths, versions := linkedModules(t)
	for i, p := range paths {
		if !strings.Contains(txt, p+" "+versions[i]) {
			t.Errorf("LICENSES.txt lacks %s %s: run `go run ./cmd/licenses`", p, versions[i])
		}
	}
	for _, want := range []string{
		"Apache License", "Version 2.0, January 2004", // Apache-2.0
		"Permission is hereby granted, free of charge",      // MIT (pixelfx)
		"Redistribution and use in source and binary forms", // BSD
		"SIL OPEN FONT LICENSE Version 1.1",
		"creativecommons.org/licenses/by/4.0",
		"Copyright 2014 The Nunito Project Authors",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("LICENSES.txt lacks %q", want)
		}
	}
}

// The credits text fits its width and its entries are not empty.
func TestCreditsLinesFit(t *testing.T) {
	lines, total := creditsLines(creditSections(), crW-8)
	if total <= crH {
		t.Errorf("credits are only %.0f tall: nothing to scroll", total)
	}
	for _, l := range lines {
		if l.s == "" {
			t.Errorf("empty line at y=%.0f", l.y)
		}
		sc := map[crKind]float64{crHead: 1.6, crName: crNameSz, crInfo: crInfoSz, crURL: crURLSz, crNote: 1}[l.kind]
		if w := textWidth(l.s, sc); w > crW-8 {
			t.Errorf("line %q is %.0f wide, the text %d", l.s, w, crW-8)
		}
	}
}
