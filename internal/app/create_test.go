package app

import (
	"path/filepath"
	"testing"
	"time"

	"noto/internal/config"
	"noto/internal/index"
	"noto/internal/storage"
)

func TestIndexPath(t *testing.T) {
	cfg := config.Config{NotesDir: "/home/user/.local/share/noto/notes"}
	got := IndexPath(cfg)
	want := "/home/user/.local/share/noto/index.db"
	if got != want {
		t.Errorf("IndexPath() = %q, want %q", got, want)
	}
}

func TestPrepareNewNote(t *testing.T) {
	cfg := config.Config{NotesDir: t.TempDir()}

	n, path, err := PrepareNewNote(cfg, "買い物リスト")
	if err != nil {
		t.Fatalf("PrepareNewNote() returned error: %v", err)
	}

	if n.ID == "" {
		t.Error("PrepareNewNote() note ID is empty")
	}
	if n.Title != "買い物リスト" {
		t.Errorf("note.Title = %q, want %q", n.Title, "買い物リスト")
	}
	if n.Body != "" {
		t.Errorf("note.Body = %q, want empty", n.Body)
	}
	if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
		t.Error("note.CreatedAt/UpdatedAt should be set")
	}

	got, err := storage.Read(path)
	if err != nil {
		t.Fatalf("storage.Read(%s) returned error: %v", path, err)
	}
	if got.ID != n.ID || got.Title != n.Title || got.Body != n.Body {
		t.Errorf("Read() = %+v, want %+v", got, n)
	}
}

func TestPrepareNewNoteEmptyTitleFallsBackToID(t *testing.T) {
	cfg := config.Config{NotesDir: t.TempDir()}

	n, path, err := PrepareNewNote(cfg, "")
	if err != nil {
		t.Fatalf("PrepareNewNote() returned error: %v", err)
	}

	wantName := storage.FileName(n)
	if filepath.Base(path) != wantName {
		t.Errorf("path base = %q, want %q", filepath.Base(path), wantName)
	}
}

func TestEditorCommand(t *testing.T) {
	tests := []struct {
		name       string
		cfgEditor  string
		visual     string
		editor     string
		wantArgs   []string
		wantSuffix string // last element of wantArgs, for clarity in failure messages
	}{
		{
			name:      "cfg.Editor takes precedence",
			cfgEditor: "code --wait",
			visual:    "nano",
			editor:    "emacs",
			wantArgs:  []string{"code", "--wait", "/tmp/note.md"},
		},
		{
			name:     "falls back to $VISUAL",
			visual:   "nano",
			editor:   "emacs",
			wantArgs: []string{"nano", "/tmp/note.md"},
		},
		{
			name:     "falls back to $EDITOR",
			editor:   "emacs",
			wantArgs: []string{"emacs", "/tmp/note.md"},
		},
		{
			name:     "falls back to default when nothing is set",
			wantArgs: []string{defaultEditor, "/tmp/note.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			cfg := config.Config{Editor: tt.cfgEditor}
			cmd := EditorCommand(cfg, "/tmp/note.md")

			if len(cmd.Args) != len(tt.wantArgs) {
				t.Fatalf("Args = %v, want %v", cmd.Args, tt.wantArgs)
			}
			for i := range tt.wantArgs {
				if cmd.Args[i] != tt.wantArgs[i] {
					t.Errorf("Args = %v, want %v", cmd.Args, tt.wantArgs)
					break
				}
			}
		})
	}
}

func TestFinalizeNote(t *testing.T) {
	notesDir := t.TempDir()

	now := time.Now()
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "saved by editor",
		Tags:      []string{"a", "b"},
		CreatedAt: now,
		UpdatedAt: now,
		Body:      "edited body\n",
	}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	got, err := FinalizeNote(idx, notesDir, path)
	if err != nil {
		t.Fatalf("FinalizeNote() returned error: %v", err)
	}

	if got.ID != n.ID || got.Title != n.Title || got.Body != n.Body {
		t.Errorf("FinalizeNote() = %+v, want %+v", got, n)
	}
}
