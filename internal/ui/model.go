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
	// modeSearch is the search field, focused.
	modeSearch
	// modeTagFilter is the tag selection screen.
	modeTagFilter
	// modeConfirmDelete is the y/n/Esc delete confirmation prompt.
	modeConfirmDelete
	// modeHelp is the keybindings help overlay.
	modeHelp
)

// Model is noto's top-level Bubble Tea model.
type Model struct {
	cfg              config.Config
	idx              *index.DB
	mode             mode
	input            textinput.Model
	search           textinput.Model
	searchGeneration int
	notes            []index.NoteSummary
	cursor           int
	selectedTags     []string
	allTags          []string
	tagCursor        int
	pendingDelete    bool
	err              error
}

// New builds the initial Model, loading the current note list from idx.
// idx must already be open (and, in practice, freshly Sync'd); the caller
// owns its lifecycle (opening/closing).
func New(cfg config.Config, idx *index.DB) Model {
	input := textinput.New()
	input.Placeholder = "タイトル(空欄可)"

	search := textinput.New()
	search.Placeholder = "検索"
	search.Prompt = "/ "

	m := Model{
		cfg:    cfg,
		idx:    idx,
		mode:   modeList,
		input:  input,
		search: search,
	}

	notes, err := app.ListNotes(idx)
	if err != nil {
		m.err = err
		return m
	}
	m.notes = notes

	return m
}

// refreshNotes re-fetches m.notes, honoring the current search query and
// selected tags (if any) so that creating or editing a note while filtered
// doesn't silently drop back to the unfiltered list.
func (m Model) refreshNotes() ([]index.NoteSummary, error) {
	return app.FilterNotes(m.idx, m.search.Value(), m.selectedTags)
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
	case searchDebounceMsg:
		return m.handleSearchDebounce(msg)
	}

	switch m.mode {
	case modeTitleInput:
		return m.updateTitleInput(msg)
	case modeSearch:
		return m.updateSearch(msg)
	case modeTagFilter:
		return m.updateTagFilter(msg)
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	case modeHelp:
		return m.updateHelp(msg)
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
	case modeSearch:
		return m.viewSearch()
	case modeTagFilter:
		return m.viewTagFilter()
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	case modeHelp:
		return m.viewHelp()
	default:
		return m.viewList()
	}
}
