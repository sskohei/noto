package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/sskohei/noto/internal/app"
	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func newTodoTestModel(t *testing.T) (Model, config.Config, *index.DB) {
	t.Helper()

	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	if err := os.MkdirAll(app.TodosDir(cfg), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() returned error: %v", err)
	}

	return skipSplash(New(cfg, idx)), cfg, idx
}

func seedTodos(t *testing.T, cfg config.Config, idx *index.DB) {
	t.Helper()

	fixtures := []struct {
		id, title string
		done      bool
		updatedAt string
	}{
		{"018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "経費精算を提出する", false, "2026-01-03T00:00:00Z"},
		{"018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "資料を印刷する", false, "2026-01-02T00:00:00Z"},
		{"018f2e4a-cccc-cccc-cccc-cccccccccccc", "完了済みタスク", true, "2026-01-01T00:00:00Z"},
	}

	todosDir := app.TodosDir(cfg)
	for _, f := range fixtures {
		ts, err := time.Parse(time.RFC3339, f.updatedAt)
		if err != nil {
			t.Fatalf("failed to parse fixture time %q: %v", f.updatedAt, err)
		}
		todo := storage.Todo{ID: f.id, Title: f.title, Done: f.done, CreatedAt: ts, UpdatedAt: ts}
		path, err := storage.GenerateTodoPath(todosDir, todo)
		if err != nil {
			t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
		}
		if err := storage.WriteTodo(path, todo); err != nil {
			t.Fatalf("storage.WriteTodo() returned error: %v", err)
		}
	}

	if err := idx.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}
}

func TestUpdateMain_DigitFourSwitchesFocusToTodos(t *testing.T) {
	m, _, _ := newTodoTestModel(t)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	final := got.(Model)

	if final.focusedPanel != focusTodos {
		t.Fatalf("focusedPanel after '4' = %v, want focusTodos", final.focusedPanel)
	}
}

func TestTodoPanelFlow_ShowsTodosAndCursorMoves(t *testing.T) {
	m, cfg, idx := newTodoTestModel(t)
	seedTodos(t, cfg, idx)
	m = skipSplash(New(cfg, idx))

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	// incomplete-first ordering: the two not-done todos come before the
	// done one, most recently updated first. This (and the cursor mark on
	// the first row) is already true in the very first rendered frame.
	teatest.WaitFor(t, out, containsBytes("> [ ] 経費精算を提出する"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	teatest.WaitFor(t, out, containsBytes("▶ 4:todoリスト"), teatest.WithDuration(2*time.Second))

	// The cursor mark moving onto this row is new content relative to the
	// frames seen so far, so it's safe to wait for.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	teatest.WaitFor(t, out, containsBytes("> [ ] 資料を印刷する"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.todoCursor != 1 {
		t.Errorf("todoCursor = %d, want 1", final.todoCursor)
	}
}

func TestTodoPanelFlow_SpaceTogglesDoneAndReorders(t *testing.T) {
	m, cfg, idx := newTodoTestModel(t)
	seedTodos(t, cfg, idx)
	m = skipSplash(New(cfg, idx))
	m.focusedPanel = focusTodos

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("> [ ] 経費精算を提出する"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	// The toggled todo moves out from under the (unmoved) cursor, so its
	// row no longer carries the "> " mark; only the checkbox changes.
	teatest.WaitFor(t, out, containsBytes("[x] 経費精算を提出する"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)

	// The toggled todo is now done, so it drops behind the one remaining
	// incomplete todo ("資料を印刷する"); within the done group it sorts
	// ahead of "完了済みタスク" since it was just updated (more recently
	// than that older done todo).
	if len(final.todos) != 3 {
		t.Fatalf("final.todos = %+v, want 3 todos", final.todos)
	}
	wantOrder := []string{"資料を印刷する", "経費精算を提出する", "完了済みタスク"}
	for i, want := range wantOrder {
		if final.todos[i].Title != want {
			t.Errorf("final.todos[%d].Title = %q, want %q", i, final.todos[i].Title, want)
		}
	}
	if !final.todos[1].Done {
		t.Error("final.todos[1].Done = false, want true (経費精算を提出する was just toggled done)")
	}
}

func TestNewTodoFlow_FullRoundTrip(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")

	m, cfg, _ := newTodoTestModel(t)
	m.focusedPanel = focusTodos

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	teatest.WaitFor(t, out, containsBytes("新規todoのタイトル"), teatest.WithDuration(2*time.Second))

	typeRunes(tm, "買い物に行く")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, out, containsBytes("買い物に行く"), teatest.WithDuration(2*time.Second))

	paths, err := storage.ListTodos(app.TodosDir(cfg))
	if err != nil {
		t.Fatalf("storage.ListTodos() returned error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("storage.ListTodos() = %v, want exactly 1 todo", paths)
	}

	todo, err := storage.ReadTodo(paths[0])
	if err != nil {
		t.Fatalf("storage.ReadTodo() returned error: %v", err)
	}
	if todo.Title != "買い物に行く" {
		t.Errorf("todo.Title = %q, want %q", todo.Title, "買い物に行く")
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

func TestTodoDeleteFlow_ConfirmRemovesTodoAndFile(t *testing.T) {
	m, cfg, idx := newTodoTestModel(t)
	seedTodos(t, cfg, idx)
	m = skipSplash(New(cfg, idx))
	m.focusedPanel = focusTodos

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("経費精算を提出する"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return containsBytes("経費精算を提出する")(bts) && containsBytes("削除しますか")(bts)
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return !containsBytes("経費精算を提出する")(bts)
	}, teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.todos) != 2 {
		t.Errorf("final.todos = %+v, want 2 remaining", final.todos)
	}

	paths, err := storage.ListTodos(app.TodosDir(cfg))
	if err != nil {
		t.Fatalf("storage.ListTodos() returned error: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("storage.ListTodos() = %v, want 2 remaining", paths)
	}
}

// seedExtraCompletedTodo writes one more completed todo into cfg's todos
// dir (on top of whatever seedTodos already wrote) and re-syncs the index,
// so tests can exercise bulk-deleting more than one completed todo.
func seedExtraCompletedTodo(t *testing.T, cfg config.Config, idx *index.DB, id, title string) {
	t.Helper()

	todosDir := app.TodosDir(cfg)
	ts, err := time.Parse(time.RFC3339, "2026-01-04T00:00:00Z")
	if err != nil {
		t.Fatalf("failed to parse fixture time: %v", err)
	}
	todo := storage.Todo{ID: id, Title: title, Done: true, CreatedAt: ts, UpdatedAt: ts}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}
	if err := idx.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}
}

func TestTodoDeleteCompletedFlow_ConfirmRemovesOnlyCompletedTodos(t *testing.T) {
	m, cfg, idx := newTodoTestModel(t)
	seedTodos(t, cfg, idx)
	seedExtraCompletedTodo(t, cfg, idx, "018f2e4a-dddd-dddd-dddd-dddddddddddd", "もう一つの完了済みタスク")
	m = skipSplash(New(cfg, idx))
	m.focusedPanel = focusTodos

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("完了済みタスク"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	teatest.WaitFor(t, out, containsBytes("完了済みのtodoを2件削除しますか"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return !containsBytes("完了済みタスク")(bts) && !containsBytes("もう一つの完了済みタスク")(bts)
	}, teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.todos) != 2 {
		t.Fatalf("final.todos = %+v, want 2 remaining (only the not-done todos)", final.todos)
	}
	for _, todo := range final.todos {
		if todo.Done {
			t.Errorf("final.todos contains a done todo after bulk delete: %+v", todo)
		}
	}

	paths, err := storage.ListTodos(app.TodosDir(cfg))
	if err != nil {
		t.Fatalf("storage.ListTodos() returned error: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("storage.ListTodos() = %v, want 2 remaining", paths)
	}
}

func TestTodoDeleteCompletedFlow_CancelKeepsTodos(t *testing.T) {
	m, cfg, idx := newTodoTestModel(t)
	seedTodos(t, cfg, idx)
	m = skipSplash(New(cfg, idx))
	m.focusedPanel = focusTodos

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("完了済みタスク"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	teatest.WaitFor(t, out, containsBytes("完了済みのtodoを1件削除しますか"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.todos) != 3 {
		t.Errorf("final.todos = %+v, want all 3 todos to remain after cancel", final.todos)
	}
}

func TestTodoDeleteCompletedFlow_NoOpWhenNoneCompleted(t *testing.T) {
	m, cfg, idx := newTodoTestModel(t)
	// Seed only not-done todos so there is nothing to bulk-delete.
	todosDir := app.TodosDir(cfg)
	todo := storage.Todo{ID: "018f2e4a-eeee-eeee-eeee-eeeeeeeeeeee", Title: "not done", Done: false}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}
	if err := idx.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}
	m = skipSplash(New(cfg, idx))
	m.focusedPanel = focusTodos

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain (D with no completed todos should be a no-op)", final.mode)
	}
}

func TestTodoDeleteFlow_NoOpWhenEmpty(t *testing.T) {
	m, _, _ := newTodoTestModel(t)
	m.focusedPanel = focusTodos
	m.pendingDelete = true

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain (dd on empty todo list should be a no-op)", final.mode)
	}
}

func TestTodoEditFlow_EnterEditsExistingTodo(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")

	m, cfg, idx := newTodoTestModel(t)
	seedTodos(t, cfg, idx)
	m = skipSplash(New(cfg, idx))
	m.focusedPanel = focusTodos

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("経費精算を提出する"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, out, containsBytes("経費精算を提出する"), teatest.WithDuration(2*time.Second))

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

func TestUpdateTodoPanel_CursorClampsAtBounds(t *testing.T) {
	m, _, _ := newTodoTestModel(t)
	m.focusedPanel = focusTodos
	m.todos = []index.TodoSummary{{ID: "a"}, {ID: "b"}, {ID: "c"}}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = got.(Model)
	if m.todoCursor != 0 {
		t.Fatalf("todoCursor after k at top = %d, want 0", m.todoCursor)
	}

	for range m.todos {
		got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = got.(Model)
	}
	if m.todoCursor != len(m.todos)-1 {
		t.Fatalf("todoCursor after repeated j = %d, want %d", m.todoCursor, len(m.todos)-1)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = got.(Model)
	if m.todoCursor != len(m.todos)-1 {
		t.Errorf("todoCursor after j past bottom = %d, want %d", m.todoCursor, len(m.todos)-1)
	}
}
