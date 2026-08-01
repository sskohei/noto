package app

import (
	"fmt"
	"time"

	"noto/internal/index"
	"noto/internal/storage"
)

// FinalizeEdit re-reads path (as left behind by an editor session), bumps
// its updated_at to now, persists that change, and reconciles the search
// index against notesDir.
func FinalizeEdit(idx *index.DB, notesDir, path string) (storage.Note, error) {
	n, err := storage.Read(path)
	if err != nil {
		return storage.Note{}, fmt.Errorf("app: read edited note: %w", err)
	}

	n.UpdatedAt = time.Now()
	if err := storage.Write(path, n); err != nil {
		return storage.Note{}, fmt.Errorf("app: write edited note: %w", err)
	}

	if err := idx.Sync(notesDir); err != nil {
		return storage.Note{}, fmt.Errorf("app: sync index: %w", err)
	}

	return n, nil
}
