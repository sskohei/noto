package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/app"
)

// updateConfirmDelete handles key input for modeConfirmDelete.
func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "y":
		return m.confirmDelete()
	case "n", "esc":
		m.mode = modeMain
		return m, nil
	}

	return m, nil
}

// confirmDelete deletes the note under the cursor and refreshes the list.
func (m Model) confirmDelete() (tea.Model, tea.Cmd) {
	note := m.notes[m.cursor]
	m.mode = modeMain

	if err := app.DeleteNote(m.idx, m.cfg.NotesDir, note.Path); err != nil {
		m.err = err
		return m, nil
	}

	notes, err := m.refreshNotes()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.notes = notes
	if m.cursor >= len(m.notes) && m.cursor > 0 {
		m.cursor--
	}
	m.err = nil
	return m, nil
}

func (m Model) viewConfirmDelete() string {
	title := "(無題)"
	if len(m.notes) > 0 {
		if t := m.notes[m.cursor].Title; t != "" {
			title = t
		}
	}
	return fmt.Sprintf("%q を削除しますか? (y/n)\n", title)
}
