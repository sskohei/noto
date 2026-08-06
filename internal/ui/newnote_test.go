package ui

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func newTestModel(t *testing.T) (Model, config.Config, *index.DB) {
	t.Helper()

	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	return skipSplash(New(cfg, idx)), cfg, idx
}

// skipSplash returns m past the startup splash screen, as if the user had
// already dismissed it. Tests other than splash_test.go's own don't care
// about the splash and would otherwise have their first keypress consumed
// by it instead of reaching the screen under test.
func skipSplash(m Model) Model {
	m.mode = modeMain
	return m
}

func TestNewNoteFlow_FullRoundTrip(t *testing.T) {
	// EDITOR is set to a no-op command so tea.ExecProcess completes near
	// instantly without needing a real interactive editor.
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")

	m, cfg, _ := newTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	teatest.WaitFor(t, out, containsBytes("Enter で確定"), teatest.WithDuration(2*time.Second))

	typeRunes(tm, "買い物リスト")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// After the editor "session" (the no-op `true` command) finishes, the
	// model should return to the list screen.
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	paths, err := storage.List(cfg.NotesDir)
	if err != nil {
		t.Fatalf("storage.List() returned error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("storage.List() = %v, want exactly 1 note", paths)
	}

	n, err := storage.Read(paths[0])
	if err != nil {
		t.Fatalf("storage.Read() returned error: %v", err)
	}
	if n.Title != "買い物リスト" {
		t.Errorf("note.Title = %q, want %q", n.Title, "買い物リスト")
	}

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if final.err != nil {
		t.Errorf("final err = %v, want nil", final.err)
	}
}

func TestNewNoteFlow_EscCancelsWithoutCreatingFile(t *testing.T) {
	m, cfg, _ := newTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	teatest.WaitFor(t, out, containsBytes("Enter で確定"), teatest.WithDuration(2*time.Second))

	typeRunes(tm, "捨てるタイトル")
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}

	paths, err := storage.List(cfg.NotesDir)
	if err != nil {
		t.Fatalf("storage.List() returned error: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("storage.List() = %v, want no notes created", paths)
	}
}

func TestHandleEditorFinished(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	m := skipSplash(New(cfg, idx))

	now := time.Now()
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "edited",
		CreatedAt: now,
		UpdatedAt: now,
		Body:      "hello\n",
	}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	m.mode = modeEditing
	got, _ := m.Update(editorFinishedMsg{path: path, err: nil})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain", final.mode)
	}
	if final.err != nil {
		t.Errorf("err = %v, want nil", final.err)
	}
}

func TestHandleEditorFinished_RefreshesTags(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	m := skipSplash(New(cfg, idx))
	if len(m.allTags) != 0 {
		t.Fatalf("allTags before save = %v, want empty", m.allTags)
	}

	// Focus the tags panel before "saving" the new note: switchFocus only
	// refreshes allTags on a transition *into* focusTags, so if
	// handleEditorFinished didn't refresh it itself, staying focused on the
	// tags panel throughout would never pick up the new tag.
	m.focusedPanel = focusTags

	now := time.Now()
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "new note",
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"newtag"},
		Body:      "hello\n",
	}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	m.mode = modeEditing
	got, _ := m.Update(editorFinishedMsg{path: path, err: nil})
	final := got.(Model)

	if final.err != nil {
		t.Fatalf("err = %v, want nil", final.err)
	}
	if !containsTag(final.allTags, "newtag") {
		t.Errorf("allTags = %v, want it to contain %q", final.allTags, "newtag")
	}
}

func TestHandleEditorFinished_PropagatesError(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeEditing

	wantErr := errFake{}
	got, _ := m.Update(editorFinishedMsg{path: "/does/not/matter", err: wantErr})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain", final.mode)
	}
	if final.err != wantErr {
		t.Errorf("err = %v, want %v", final.err, wantErr)
	}
}

type errFake struct{}

func (errFake) Error() string { return "fake editor error" }

func containsTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func containsBytes(s string) func([]byte) bool {
	return func(bts []byte) bool {
		return bytes.Contains(bts, []byte(s))
	}
}

// typeRunes sends s one full rune at a time. Unlike (*teatest.TestModel).Type,
// which iterates over the raw bytes of s, this handles multi-byte UTF-8
// characters (e.g. Japanese text) correctly.
func typeRunes(tm *teatest.TestModel, s string) {
	for _, r := range s {
		tm.Send(tea.KeyMsg{Runes: []rune{r}, Type: tea.KeyRunes})
	}
}
