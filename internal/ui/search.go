package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/app"
)

// searchDebounceDelay is how long to wait after the last keystroke before
// actually querying the index.
const searchDebounceDelay = 200 * time.Millisecond

// searchDebounceMsg fires searchDebounceDelay after a search query change.
// generation lets stale ticks (superseded by further typing) be ignored.
type searchDebounceMsg struct {
	query      string
	generation int
}

// updateSearchPanel handles key input local to the search panel
// (focusSearch). Unlike other panels, it deliberately does NOT go through
// updateMain's global-key preamble: every key except enter/esc must reach
// the textinput untouched, including digits and letters that would
// otherwise be global shortcuts.
func (m Model) updateSearchPanel(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Keep the query and its filtered results; just return focus
			// to the list.
			next, cmd := m.switchFocus(focusList)
			return next, cmd
		case "esc":
			m.search.Reset()
			// Invalidate any in-flight debounce tick from prior keystrokes
			// so it doesn't land after this and re-apply a stale filter.
			m.searchGeneration++

			next, cmd := m.switchFocus(focusList)
			notes, err := next.refreshNotes()
			if err != nil {
				next.err = err
				return next, cmd
			}
			next.notes = notes
			next.cursor = 0
			next.err = nil
			return next, cmd
		}
	}

	var inputCmd tea.Cmd
	m.search, inputCmd = m.search.Update(msg)

	m.searchGeneration++
	gen := m.searchGeneration
	query := m.search.Value()
	debounceCmd := tea.Tick(searchDebounceDelay, func(time.Time) tea.Msg {
		return searchDebounceMsg{query: query, generation: gen}
	})

	return m, tea.Batch(inputCmd, debounceCmd)
}

func (m Model) handleSearchDebounce(msg searchDebounceMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.searchGeneration {
		return m, nil // superseded by further typing
	}

	notes, err := app.FilterNotes(m.idx, msg.query, m.selectedTags)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.notes = notes
	m.cursor = 0
	m.err = nil
	return m, nil
}

// searchPanelContent renders the search panel's body: just the input box,
// since the notes list panel (always visible) shows the filtered results.
func (m Model) searchPanelContent() string {
	return m.search.View()
}
