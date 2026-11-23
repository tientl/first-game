package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"survivalgame/internal/sim"
)

// SaveFile is what is written to disk: the world plus a little metadata for
// the main menu.
type SaveFile struct {
	SavedAt time.Time
	World   sim.SaveData
}

// DefaultSavePath is where single player keeps its save: the user's config
// directory (%AppData% on Windows).
func DefaultSavePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SinhTon", "save.json"), nil
}

// ReadSave loads a save file. A missing file returns an error that satisfies
// errors.Is(err, os.ErrNotExist).
func ReadSave(path string) (*SaveFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f SaveFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%w: %v", sim.ErrBadSave, err)
	}
	return &f, nil
}

// writeSave writes f atomically: a crash mid-write leaves the old save intact.
func writeSave(path string, f SaveFile) error {
	if path == "" {
		return errors.New("server: no save path configured")
	}
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".save-*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return errors.Join(werr, cerr)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}
