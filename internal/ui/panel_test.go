package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func TestUpdateMain_DigitKeysSwitchFocus(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = got.(Model)
	if m.focusedPanel != focusTags {
		t.Fatalf("focusedPanel after '2' = %v, want focusTags", m.focusedPanel)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = got.(Model)
	if m.focusedPanel != focusSearch {
		t.Fatalf("focusedPanel after '1' = %v, want focusSearch", m.focusedPanel)
	}
}

func TestUpdateMain_DigitTypedIntoSearchDoesNotSwitchFocus(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = got.(Model)
	if m.focusedPanel != focusSearch {
		t.Fatalf("focusedPanel after '1' = %v, want focusSearch", m.focusedPanel)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	final := got.(Model)

	if final.focusedPanel != focusSearch {
		t.Errorf("focusedPanel after typing '2' while search-focused = %v, want unchanged focusSearch", final.focusedPanel)
	}
	if final.search.Value() != "2" {
		t.Errorf("search.Value() = %q, want %q (digit should be typed, not intercepted)", final.search.Value(), "2")
	}
}

func TestUpdateMain_SlashAndTAreAliasesForDigits(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = got.(Model)
	if m.focusedPanel != focusTags {
		t.Fatalf("focusedPanel after 't' = %v, want focusTags", m.focusedPanel)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = got.(Model)
	if m.focusedPanel != focusSearch {
		t.Fatalf("focusedPanel after '/' = %v, want focusSearch", m.focusedPanel)
	}
}

func TestUpdateMain_PendingDeleteResetsOnPanelSwitch(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.notes = []index.NoteSummary{{ID: "a", Path: "/tmp/a.md", Title: "a"}}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = got.(Model)
	if !m.pendingDelete {
		t.Fatalf("pendingDelete = false after 'd', want true")
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	final := got.(Model)

	if final.pendingDelete {
		t.Error("pendingDelete = true after switching panels, want reset to false")
	}
	if final.focusedPanel != focusTags {
		t.Errorf("focusedPanel = %v, want focusTags", final.focusedPanel)
	}
	if final.mode == modeConfirmDelete {
		t.Error("mode = modeConfirmDelete, want no delete confirmation triggered by the panel switch")
	}
}

func TestNew_LoadsTagsEagerlyWithoutFocusingTagsPanel(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	now := time.Now()
	n := storage.Note{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "note", Tags: []string{"work"}, CreatedAt: now, UpdatedAt: now}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}
	if err := idx.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	m := New(cfg, idx)

	if len(m.allTags) != 1 || m.allTags[0] != "work" {
		t.Errorf("allTags after New() = %v, want [work] without ever focusing the tags panel", m.allTags)
	}
}

func TestViewMain_FooterReflectsFocusedPanel(t *testing.T) {
	cases := []struct {
		name    string
		panel   focusPanel
		want    []string
		notWant []string
	}{
		{
			name:    "list",
			panel:   focusList,
			want:    []string{"n: 新規メモ", "dd: 削除", "パネル移動"},
			notWant: []string{"新規todo", "完了済み削除", "文字入力: 絞り込み"},
		},
		{
			name:    "todos",
			panel:   focusTodos,
			want:    []string{"n: 新規todo", "Space: 完了切替", "D: 完了済み削除", "パネル移動"},
			notWant: []string{"n: 新規メモ", "文字入力: 絞り込み"},
		},
		{
			name:    "tags",
			panel:   focusTags,
			want:    []string{"Enter/Space: 選択", "Esc: 一覧へ戻る", "パネル移動"},
			notWant: []string{"Space: 完了切替", "文字入力: 絞り込み"},
		},
		{
			name:    "search",
			panel:   focusSearch,
			want:    []string{"文字入力: 絞り込み", "Enter: 一覧へ戻る"},
			notWant: []string{"パネル移動", "?: ヘルプ", "q: 終了", "n: 新規メモ", "dd: 削除"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.focusedPanel = c.panel

			out := m.viewMain()

			for _, want := range c.want {
				if !strings.Contains(out, want) {
					t.Errorf("viewMain() with focus=%v missing %q\n%s", c.panel, want, out)
				}
			}
			for _, notWant := range c.notWant {
				if strings.Contains(out, notWant) {
					t.Errorf("viewMain() with focus=%v unexpectedly contains %q\n%s", c.panel, notWant, out)
				}
			}
		})
	}
}

func TestViewMain_CollapsesUnfocusedOfListAndTodosPanels(t *testing.T) {
	cases := []struct {
		name          string
		panel         focusPanel
		wantExpanded  string
		wantCollapsed string
		// wantTodoSummary is checked only when the todos panel is the
		// collapsed one: it should show a count summary instead of being
		// fully empty.
		wantTodoSummary string
		// wantListSummary is checked only when the list panel is the
		// collapsed one: it should show a count summary instead of being
		// fully empty.
		wantListSummary string
	}{
		{
			name:            "list focused expands list, collapses todos to a count summary",
			panel:           focusList,
			wantExpanded:    "> 一番古いメモ",
			wantCollapsed:   "> [ ] 経費精算を提出する",
			wantTodoSummary: "未完了: 1件 / 完了: 0件",
		},
		{
			name:            "todos focused expands todos, collapses list to a count summary",
			panel:           focusTodos,
			wantExpanded:    "> [ ] 経費精算を提出する",
			wantCollapsed:   "> 一番古いメモ",
			wantListSummary: "メモ: 1件",
		},
		{
			name:            "search focused defaults to list expanded, todos collapsed to a count summary",
			panel:           focusSearch,
			wantExpanded:    "> 一番古いメモ",
			wantCollapsed:   "> [ ] 経費精算を提出する",
			wantTodoSummary: "未完了: 1件 / 完了: 0件",
		},
		{
			name:            "tags focused defaults to list expanded, todos collapsed to a count summary",
			panel:           focusTags,
			wantExpanded:    "> 一番古いメモ",
			wantCollapsed:   "> [ ] 経費精算を提出する",
			wantTodoSummary: "未完了: 1件 / 完了: 0件",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.notes = []index.NoteSummary{{ID: "a", Path: "/tmp/a.md", Title: "一番古いメモ"}}
			m.todos = []index.TodoSummary{{ID: "b", Path: "/tmp/b.md", Title: "経費精算を提出する"}}
			m.focusedPanel = c.panel

			out := m.viewMain()

			if !strings.Contains(out, c.wantExpanded) {
				t.Errorf("viewMain() with focus=%v missing expanded content %q\n%s", c.panel, c.wantExpanded, out)
			}
			if strings.Contains(out, c.wantCollapsed) {
				t.Errorf("viewMain() with focus=%v unexpectedly shows collapsed panel's content %q\n%s", c.panel, c.wantCollapsed, out)
			}
			if c.wantTodoSummary != "" && !strings.Contains(out, c.wantTodoSummary) {
				t.Errorf("viewMain() with focus=%v missing collapsed todo panel's count summary %q\n%s", c.panel, c.wantTodoSummary, out)
			}
			if c.wantListSummary != "" && !strings.Contains(out, c.wantListSummary) {
				t.Errorf("viewMain() with focus=%v missing collapsed list panel's count summary %q\n%s", c.panel, c.wantListSummary, out)
			}
		})
	}
}

// TestViewMain_FocusedListOrTodosPanelFillsRemainingHeight verifies that
// the focused one of the notes-list/todos panels expands to consume the
// terminal's remaining height (after the footer), even when it has very
// little content — not just enough to fit its own rows — and that the
// footer stays pinned to the last line of the rendered output.
func TestViewMain_FocusedListOrTodosPanelFillsRemainingHeight(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.width = 80
	m.height = 30
	m.notes = []index.NoteSummary{{ID: "a", Path: "/tmp/a.md", Title: "唯一のメモ"}}
	m.todos = nil
	m.focusedPanel = focusList

	out := m.viewMain()
	lines := strings.Split(out, "\n")

	if got := len(lines); got != m.height {
		t.Errorf("viewMain() output has %d lines, want %d (== m.height, footer pinned to bottom)\n%s", got, m.height, out)
	}

	footerLine := lines[len(lines)-1]
	if !strings.Contains(footerLine, "?: ヘルプ") {
		t.Errorf("last line = %q, want it to be the footer", footerLine)
	}
}

func TestViewMain_DegradesGracefullyOnVerySmallHeight(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.width = 80
	m.height = 5

	for _, panel := range []focusPanel{focusList, focusTodos, focusSearch, focusTags} {
		m.focusedPanel = panel
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("viewMain() panicked with focus=%v and m.height=5: %v", panel, r)
				}
			}()
			_ = m.viewMain()
		}()
	}
}

func TestCtrlC_QuitsRegardlessOfMode(t *testing.T) {
	for _, mode := range []mode{modeMain, modeHelp, modeTitleInput, modeConfirmDelete} {
		m, _, _ := newTestModel(t)
		m.mode = mode

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Errorf("mode %v: ctrl+c returned nil Cmd, want tea.Quit", mode)
		}
	}
}
