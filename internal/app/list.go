package app

import "noto/internal/index"

// ListNotes returns the notes currently in the index, most recently
// updated first.
func ListNotes(idx *index.DB) ([]index.NoteSummary, error) {
	return idx.List()
}
