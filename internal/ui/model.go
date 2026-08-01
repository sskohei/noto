package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
	"noto/internal/config"
	"noto/internal/index"
)

// mode identifies which screen the Model is currently showing.
type mode int

const (
	// modeSplash is the startup splash screen.
	modeSplash mode = iota
	// modeMain is the main screen: the always-visible search/tags/list
	// panel layout. Which panel receives input is tracked separately by
	// Model.focusedPanel.
	modeMain
	// modeTitleInput is the new-note title prompt.
	modeTitleInput
	// modeEditing is shown while $EDITOR has taken over the terminal via
	// tea.ExecProcess; View is a no-op in this mode.
	modeEditing
	// modeConfirmDelete is the y/n/Esc delete confirmation prompt.
	modeConfirmDelete
	// modeHelp is the keybindings help overlay.
	modeHelp
)

// focusPanel identifies which panel of the main screen currently receives
// non-global key input. It is orthogonal to mode: it only matters while
// mode == modeMain, and is left untouched by modal detours (title input,
// editing, delete confirmation, help) so focus is restored automatically
// once those finish.
type focusPanel int

const (
	// focusList is the notes list panel (the default).
	focusList focusPanel = iota
	// focusSearch is the search input panel.
	focusSearch
	// focusTags is the tag selection panel.
	focusTags
)

const (
	// defaultWidth/defaultHeight are used until the first tea.WindowSizeMsg
	// arrives, so rendering (and tests that never send one) stays sane.
	defaultWidth  = 80
	defaultHeight = 24
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
	focusedPanel     focusPanel
	width            int
	height           int
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
		mode:   modeSplash,
		input:  input,
		search: search,
		width:  defaultWidth,
		height: defaultHeight,
	}

	notes, err := app.ListNotes(idx)
	if err != nil {
		m.err = err
		return m
	}
	m.notes = notes

	tags, err := app.ListTags(idx)
	if err != nil {
		m.err = err
		return m
	}
	m.allTags = tags

	return m
}

// refreshNotes re-fetches m.notes, honoring the current search query and
// selected tags (if any) so that creating or editing a note while filtered
// doesn't silently drop back to the unfiltered list.
func (m Model) refreshNotes() ([]index.NoteSummary, error) {
	return app.FilterNotes(m.idx, m.search.Value(), m.selectedTags)
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(splashDuration, func(time.Time) tea.Msg {
		return splashTimeoutMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case editorFinishedMsg:
		return m.handleEditorFinished(msg)
	case editSessionFinishedMsg:
		return m.handleEditSessionFinished(msg)
	case searchDebounceMsg:
		return m.handleSearchDebounce(msg)
	case splashTimeoutMsg:
		return m.handleSplashTimeout(msg)
	}

	switch m.mode {
	case modeSplash:
		return m.updateSplash(msg)
	case modeTitleInput:
		return m.updateTitleInput(msg)
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	case modeHelp:
		return m.updateHelp(msg)
	default:
		return m.updateMain(msg)
	}
}

func (m Model) View() string {
	switch m.mode {
	case modeSplash:
		return m.viewSplash()
	case modeTitleInput:
		return m.viewTitleInput()
	case modeEditing:
		return ""
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	case modeHelp:
		return m.viewHelp()
	default:
		return m.viewMain()
	}
}
