package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/index"
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
