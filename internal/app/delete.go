package app

import (
	"fmt"

	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

// DeleteNote deletes the note file at path and reconciles the search index
// against notesDir.
func DeleteNote(idx *index.DB, notesDir, path string) error {
	if err := storage.Delete(path); err != nil {
		return fmt.Errorf("app: delete note: %w", err)
	}

	if err := idx.Sync(notesDir); err != nil {
		return fmt.Errorf("app: sync index: %w", err)
	}

	return nil
}
