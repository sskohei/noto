package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	const defaultNotesDir = "/default/notes"

	tests := []struct {
		name         string
		fileContent  string // if empty, no file is written
		wantNotesDir string
		wantEditor   string
	}{
		{
			name:         "file missing falls back to defaults",
			wantNotesDir: defaultNotesDir,
			wantEditor:   "",
		},
		{
			name: "all fields set",
			fileContent: `
notes_dir = "/custom/notes"
editor = "vim"
`,
			wantNotesDir: "/custom/notes",
			wantEditor:   "vim",
		},
		{
			name: "only notes_dir set",
			fileContent: `
notes_dir = "/custom/notes"
`,
			wantNotesDir: "/custom/notes",
			wantEditor:   "",
		},
		{
			name: "only editor set",
			fileContent: `
editor = "vim"
`,
			wantNotesDir: defaultNotesDir,
			wantEditor:   "vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if tt.fileContent != "" {
				if err := os.WriteFile(path, []byte(tt.fileContent), 0o600); err != nil {
					t.Fatalf("failed to write test config file: %v", err)
				}
			}

			got, err := load(path, defaultNotesDir)
			if err != nil {
				t.Fatalf("load() returned error: %v", err)
			}

			if got.NotesDir != tt.wantNotesDir {
				t.Errorf("NotesDir = %q, want %q", got.NotesDir, tt.wantNotesDir)
			}
			if got.Editor != tt.wantEditor {
				t.Errorf("Editor = %q, want %q", got.Editor, tt.wantEditor)
			}
		})
	}
}

func TestLoadExpandsHomeInNotesDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(t.TempDir(), "config.toml")
	content := "notes_dir = \"~/notes\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	got, err := load(path, "/default/notes")
	if err != nil {
		t.Fatalf("load() returned error: %v", err)
	}

	want := filepath.Join(home, "notes")
	if got.NotesDir != want {
		t.Errorf("NotesDir = %q, want %q", got.NotesDir, want)
	}
}

func TestLoad_XDGResolution(t *testing.T) {
	configHome := t.TempDir()
	dataHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)

	configDir := filepath.Join(configHome, "noto")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	content := "editor = \"vim\"\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	wantNotesDir := filepath.Join(dataHome, "noto", "notes")
	if got.NotesDir != wantNotesDir {
		t.Errorf("NotesDir = %q, want %q", got.NotesDir, wantNotesDir)
	}
	if got.Editor != "vim" {
		t.Errorf("Editor = %q, want %q", got.Editor, "vim")
	}
}
