package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
	"noto/internal/config"
	"noto/internal/index"
)

// mode identifies which screen the Model is currently showing.
type mode int

const (
	// modeList is the note list screen.
	modeList mode = iota
	// modeTitleInput is the new-note title prompt.
	modeTitleInput
	// modeEditing is shown while $EDITOR has taken over the terminal via
	// tea.ExecProcess; View is a no-op in this mode.
	modeEditing
)

// Model is noto's top-level Bubble Tea model.
type Model struct {
	cfg    config.Config
	idx    *index.DB
	mode   mode
	input  textinput.Model
	notes  []index.NoteSummary
	cursor int
	err    error
}

// New builds the initial Model, loading the current note list from idx.
// idx must already be open (and, in practice, freshly Sync'd); the caller
// owns its lifecycle (opening/closing).
func New(cfg config.Config, idx *index.DB) Model {
	input := textinput.New()
	input.Placeholder = "タイトル(空欄可)"

	m := Model{
		cfg:   cfg,
		idx:   idx,
		mode:  modeList,
		input: input,
	}

	notes, err := app.ListNotes(idx)
	if err != nil {
		m.err = err
		return m
	}
	m.notes = notes

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		return m.handleEditorFinished(msg)
	case editSessionFinishedMsg:
		return m.handleEditSessionFinished(msg)
	}

	switch m.mode {
	case modeTitleInput:
		return m.updateTitleInput(msg)
	default:
		return m.updateList(msg)
	}
}

func (m Model) View() string {
	switch m.mode {
	case modeTitleInput:
		return m.viewTitleInput()
	case modeEditing:
		return ""
	default:
		return m.viewList()
	}
}
