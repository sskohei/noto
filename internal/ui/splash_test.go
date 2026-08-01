package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestViewSplash_CentersWhenTerminalLargeEnough(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeSplash

	got := m.viewSplash()

	// lipgloss.Align centers each line of the art independently, so the
	// multi-line splashArt constant won't appear verbatim; check that each
	// of its lines survives (padded, not corrupted) instead.
	for _, line := range strings.Split(splashArt, "\n") {
		if !strings.Contains(got, line) {
			t.Errorf("viewSplash() output is missing art line %q:\n%s", line, got)
		}
	}
	if !strings.Contains(got, splashHint) {
		t.Errorf("viewSplash() output does not contain the hint text:\n%s", got)
	}
}

func TestViewSplash_DegradesGracefullyWhenTerminalTooSmall(t *testing.T) {
	// Bypass New() to simulate a model that never received a
	// tea.WindowSizeMsg and has width/height at their zero values.
	m := Model{mode: modeSplash}

	got := m.viewSplash()

	if !strings.Contains(got, splashArt) {
		t.Errorf("viewSplash() with zero width/height corrupted or dropped the art:\n%s", got)
	}
}

func TestUpdateSplash_AnyKeyDismisses(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeSplash

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	final := got.(Model)

	if final.mode == modeSplash {
		t.Error("mode is still modeSplash after a keypress, want dismissed")
	}
}

func TestHandleSplashTimeout_DismissesWhileStillOnSplash(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeSplash

	got, _ := m.Update(splashTimeoutMsg{})
	final := got.(Model)

	if final.mode == modeSplash {
		t.Error("mode is still modeSplash after the timeout, want dismissed")
	}
}

func TestHandleSplashTimeout_NoOpIfAlreadyDismissed(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeSplash

	dismissed, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	mm := dismissed.(Model)
	modeAfterKey := mm.mode

	got, _ := mm.Update(splashTimeoutMsg{})
	final := got.(Model)

	if final.mode != modeAfterKey {
		t.Errorf("mode changed on a stale timeout tick: got %v, want unchanged %v", final.mode, modeAfterKey)
	}
}

func TestSplashFlow_KeypressDismissesToMainScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeSplash

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes(splashHint), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode == modeSplash {
		t.Errorf("final mode = %v, want dismissed from splash", final.mode)
	}
}

func TestSplashFlow_TimesOutAutomatically(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeSplash

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes(splashHint), teatest.WithDuration(2*time.Second))
	teatest.WaitFor(t, out, containsBytes("n: 新規メモ"), teatest.WithDuration(3*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode == modeSplash {
		t.Errorf("final mode = %v, want dismissed after timeout", final.mode)
	}
}
