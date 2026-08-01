package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
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

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Keep the query and its filtered results; just return focus
			// to the list.
			m.search.Blur()
			m.mode = modeList
			return m, nil
		case "esc":
			m.search.Reset()
			m.search.Blur()
			m.mode = modeList
			// Invalidate any in-flight debounce tick from prior keystrokes
			// so it doesn't land after this and re-apply a stale filter.
			m.searchGeneration++

			notes, err := m.refreshNotes()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.notes = notes
			m.cursor = 0
			m.err = nil
			return m, nil
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

func (m Model) viewSearch() string {
	return m.search.View() + "\n\n" + renderNoteRows(m.notes, m.cursor) +
		"\n(Enter でフォーカスのみ戻す / Esc でクリア)\n"
}
