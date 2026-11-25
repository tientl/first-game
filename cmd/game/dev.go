package main

import (
	"log"
	"path/filepath"

	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// Dev mode (-dev) skips the menu: a new world of devSize tiles from a fixed
// seed (unless -size and -seed say otherwise), kept in its own save file,
// played by a default character in god mode with the whole map seen, and
// the console on the ` key. The server runs dev commands only in this mode.
const (
	devSize = sim.SizeMedium
	devSeed = 20240601
	devName = "Dev"
)

// devSavePath is the dev save, next to the normal one.
func devSavePath(normal string) string {
	return filepath.Join(filepath.Dir(normal), "dev-save.json")
}

// startDev starts making the dev world; Update joins it once it is ready.
func (a *app) startDev() {
	if a.seed == 0 {
		a.seed = devSeed
	}
	if a.size <= 0 {
		a.size = devSize
	}
	a.look, a.join = sim.Look{}, a.name
	if a.join == "" {
		a.join = devName
	}
	cfg := a.config()
	cfg.Seed, cfg.Width, cfg.Height, cfg.Style = a.seed, a.size, a.size, sim.StyleLatest
	log.Printf("dev mode: new world, seed %d, %dx%d", cfg.Seed, a.size, a.size)
	a.background("Đang tạo thế giới dev", func() loaded {
		srv, err := server.New(cfg)
		return loaded{srv, err, "Không tạo được thế giới"}
	})
}
