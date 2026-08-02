package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

// defaultEditor is used when none of cfg.Editor, $VISUAL, or $EDITOR is set.
const defaultEditor = "vi"

// IndexPath derives the search index database path from cfg.NotesDir, per
// docs/data-model.md (index.db lives alongside the notes/ directory).
func IndexPath(cfg config.Config) string {
	return filepath.Join(filepath.Dir(cfg.NotesDir), "index.db")
}

// PrepareNewNote generates a new note (id, timestamps) with the given
// title and an empty body, and writes its frontmatter-only template file
// into cfg.NotesDir. It returns the note and the path it was written to.
func PrepareNewNote(cfg config.Config, title string) (storage.Note, string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return storage.Note{}, "", fmt.Errorf("app: generate note id: %w", err)
	}

	now := time.Now()
	n := storage.Note{
		ID:        id.String(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	path, err := storage.GeneratePath(cfg.NotesDir, n)
	if err != nil {
		return storage.Note{}, "", fmt.Errorf("app: generate note path: %w", err)
	}

	if err := storage.Write(path, n); err != nil {
		return storage.Note{}, "", fmt.Errorf("app: write note template: %w", err)
	}

	return n, path, nil
}

// EditorCommand builds (without executing) the command used to edit path:
// cfg.Editor, falling back to $VISUAL, then $EDITOR, then defaultEditor.
func EditorCommand(cfg config.Config, path string) *exec.Cmd {
	editor := cfg.Editor
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = defaultEditor
	}

	args := append(strings.Fields(editor), path)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// FinalizeNote re-reads path (as left behind by an editor session) and
// reconciles the search index against notesDir.
func FinalizeNote(idx *index.DB, notesDir, path string) (storage.Note, error) {
	n, err := storage.Read(path)
	if err != nil {
		return storage.Note{}, fmt.Errorf("app: read saved note: %w", err)
	}

	if err := idx.Sync(notesDir); err != nil {
		return storage.Note{}, fmt.Errorf("app: sync index: %w", err)
	}

	return n, nil
}
