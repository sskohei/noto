package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestHelpFlow_QuestionMarkOpensAndCloses(t *testing.T) {
	m, _, _ := newTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	teatest.WaitFor(t, out, containsBytes("ヘルプを閉じて一覧に戻る"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
}

func TestHelpFlow_EscCloses(t *testing.T) {
	m, _, _ := newTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	teatest.WaitFor(t, out, containsBytes("ヘルプを閉じて一覧に戻る"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
}

func TestHelpFlow_QClosesWithoutQuittingApp(t *testing.T) {
	m, _, _ := newTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	teatest.WaitFor(t, out, containsBytes("ヘルプを閉じて一覧に戻る"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	// If "q" had quit the whole program instead of just closing the help
	// overlay, this explicit Quit would be redundant but harmless; the real
	// assertion is that the model is still alive and back in modeMain.
	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
}

func TestUpdateList_QuestionMarkEntersHelp(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	final := got.(Model)

	if final.mode != modeHelp {
		t.Errorf("mode = %v, want modeHelp", final.mode)
	}
}

func TestUpdateHelp_UnhandledKeyIsNoOp(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeHelp

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	final := got.(Model)

	if final.mode != modeHelp {
		t.Errorf("mode = %v, want modeHelp (unchanged)", final.mode)
	}
}
