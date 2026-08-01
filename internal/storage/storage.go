package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Read reads and parses the note file at path.
func Read(path string) (Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Note{}, err
	}

	n, err := Parse(data)
	if err != nil {
		return Note{}, fmt.Errorf("storage: read %s: %w", path, err)
	}
	return n, nil
}

// Write serializes n and writes it to path, creating the parent directory
// if necessary. It does not compute a file name; callers that are creating
// a new note should obtain path via GeneratePath first. Callers updating an
// existing note can pass the path it was read from.
func Write(path string, n Note) error {
	data, err := n.Marshal()
	if err != nil {
		return fmt.Errorf("storage: write %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// List returns the sorted paths of all note files (*.md) directly inside
// dir.
func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}

	sort.Strings(paths)
	return paths, nil
}

// Delete removes the note file at path.
func Delete(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("storage: delete %s: %w", path, err)
	}
	return nil
}

// GeneratePath returns a path in dir for a new note n, based on
// FileName(n). If that name is already taken, a "-2", "-3", ... suffix is
// appended before the extension until a free path is found.
func GeneratePath(dir string, n Note) (string, error) {
	name := FileName(n)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for i := 1; ; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d%s", base, i, ext)
		}

		path := filepath.Join(dir, candidate)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
	}
}
