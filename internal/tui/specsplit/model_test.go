package specsplit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlCCancelsSplit(t *testing.T) {
	m := newModel(Options{})
	m.phase = phaseRunning
	m.running = true
	canceled := false
	m.cancelSplit = func() { canceled = true }

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if !m.canceled {
		t.Fatalf("model not marked canceled")
	}
	if !canceled {
		t.Fatalf("cancelSplit was not called")
	}
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Fatalf("expected tea.QuitMsg, got %T", msg)
		}
	}
}
