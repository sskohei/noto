package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/app"
)

// editSessionFinishedMsg is sent once the $EDITOR process launched to edit
// an existing note (via tea.ExecProcess) exits.
type editSessionFinishedMsg struct {
	path string
	err  error
}

// startEditingExisting hands off editing of the currently selected note (or,
// if the todo panel is focused, todo) to $EDITOR via tea.ExecProcess, the
// only place this may be called from (see docs/architecture.md and the plan
// for issue #5/#7). It is a no-op if the relevant list is empty.
func (m Model) startEditingExisting() (tea.Model, tea.Cmd) {
	var path string
	if m.focusedPanel == focusTodos {
		if len(m.todos) == 0 {
			return m, nil
		}
		path = m.todos[m.todoCursor].Path
	} else {
		if len(m.notes) == 0 {
			return m, nil
		}
		path = m.notes[m.cursor].Path
	}

	m.mode = modeEditing
	m.err = nil

	cmd := app.EditorCommand(m.cfg, path)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editSessionFinishedMsg{path: path, err: err}
	})
}

func (m Model) handleEditSessionFinished(msg editSessionFinishedMsg) (tea.Model, tea.Cmd) {
	m.mode = modeMain

	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	if m.focusedPanel == focusTodos {
		edited, err := app.FinalizeTodoEdit(m.idx, app.TodosDir(m.cfg), msg.path)
		if err != nil {
			m.err = err
			return m, nil
		}

		todos, err := m.refreshTodos()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.todos = todos
		// Unlike notes (always sorted purely by updated_at), todos are
		// grouped by done/not-done first, so bumping updated_at doesn't
		// necessarily move the edited todo to index 0. Find its new
		// position instead of assuming it's at the top.
		m.todoCursor = indexOfTodo(todos, edited.ID)
		m.err = nil
		return m, nil
	}

	if _, err := app.FinalizeEdit(m.idx, m.cfg.NotesDir, msg.path); err != nil {
		m.err = err
		return m, nil
	}

	notes, err := m.refreshNotes()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.notes = notes
	m.cursor = 0 // the edited note's updated_at is now newest, so it moves to the top

	tags, err := m.refreshTags()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.allTags = tags
	if m.tagCursor >= len(m.allTags) {
		m.tagCursor = 0
	}

	m.err = nil
	return m, nil
}
