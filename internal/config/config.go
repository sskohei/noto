package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds noto's user-configurable settings.
type Config struct {
	// NotesDir is the directory notes are stored in.
	NotesDir string
	// Editor is the command used to edit a note. If empty, callers should
	// fall back to $EDITOR / $VISUAL.
	Editor string
}

// Load reads the config file from its XDG-resolved location. If the file
// does not exist, or a field is missing from it, the corresponding default
// value is used instead.
func Load() (Config, error) {
	path, err := configFilePath()
	if err != nil {
		return Config{}, err
	}

	notesDir, err := defaultNotesDir()
	if err != nil {
		return Config{}, err
	}

	return load(path, notesDir)
}

func load(path, defaultNotesDir string) (Config, error) {
	cfg := Config{NotesDir: defaultNotesDir}

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return expandNotesDir(cfg)
	case err != nil:
		return Config{}, err
	}

	var raw struct {
		NotesDir string `toml:"notes_dir"`
		Editor   string `toml:"editor"`
	}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return Config{}, err
	}

	if raw.NotesDir != "" {
		cfg.NotesDir = raw.NotesDir
	}
	cfg.Editor = raw.Editor

	return expandNotesDir(cfg)
}

func expandNotesDir(cfg Config) (Config, error) {
	expanded, err := expandHome(cfg.NotesDir)
	if err != nil {
		return Config{}, err
	}
	cfg.NotesDir = expanded
	return cfg, nil
}

// expandHome expands a leading "~" in path to the user's home directory.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

func configFilePath() (string, error) {
	dir, err := xdgConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "noto", "config.toml"), nil
}

func defaultNotesDir() (string, error) {
	dir, err := xdgDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "noto", "notes"), nil
}

func xdgConfigHome() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func xdgDataHome() (string, error) {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
