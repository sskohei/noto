package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
)

// editSessionFinishedMsg is sent once the $EDITOR process launched to edit
// an existing note (via tea.ExecProcess) exits.
type editSessionFinishedMsg struct {
	path string
	err  error
}

// startEditingExisting hands off editing of the currently selected note to
// $EDITOR via tea.ExecProcess, the only place this may be called from (see
// docs/architecture.md and the plan for issue #5/#7). It is a no-op if the
// list is empty.
func (m Model) startEditingExisting() (tea.Model, tea.Cmd) {
	if len(m.notes) == 0 {
		return m, nil
	}

	path := m.notes[m.cursor].Path
	m.mode = modeEditing
	m.err = nil

	cmd := app.EditorCommand(m.cfg, path)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editSessionFinishedMsg{path: path, err: err}
	})
}

func (m Model) handleEditSessionFinished(msg editSessionFinishedMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList

	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	if _, err := app.FinalizeEdit(m.idx, m.cfg.NotesDir, msg.path); err != nil {
		m.err = err
		return m, nil
	}

	notes, err := app.ListNotes(m.idx)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.notes = notes
	m.cursor = 0 // the edited note's updated_at is now newest, so it moves to the top
	m.err = nil
	return m, nil
}
