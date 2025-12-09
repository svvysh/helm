package specsplit

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/components"
)

// Guard against the running view overflowing small terminals, which can hide
// the spinner/status line.
func TestRunningViewRespectsHeight(t *testing.T) {
	cases := []struct {
		height int
	}{
		{height: 18},
		{height: 12},
	}

	for _, tc := range cases {
		m := &model{
			width:     80,
			height:    tc.height,
			phase:     phaseRunning,
			spinner:   components.NewSpinner(),
			vp:        viewport.New(0, 0),
			running:   true,
			draft:     "# test",
			opts:      Options{},
			resumeCmd: "",
		}

		m.resize()

		h := lipgloss.Height(m.View())
		if h > m.height {
			t.Fatalf("running view height %d exceeds window height %d for height %d", h, m.height, tc.height)
		}
	}
}
