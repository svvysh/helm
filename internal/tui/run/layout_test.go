package run

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/metadata"
	"github.com/polarzero/helm/internal/specs"
	"github.com/polarzero/helm/internal/tui/components"
)

// Ensure the running view never grows taller than the terminal height, which
// would push the spinner/status line off-screen.
func TestRunningViewRespectsHeight(t *testing.T) {
	cases := []struct {
		height int
	}{
		{height: 18},
		{height: 12},
	}

	for _, tc := range cases {
		m := &model{
			width:    80,
			height:   tc.height,
			phase:    phaseRunning,
			spinner:  components.NewSpinner(),
			viewport: viewport.New(0, 0),
			running: &runState{
				spec: &specs.SpecFolder{
					Metadata: &metadata.SpecMetadata{ID: "spec-01-test", Name: "Test Spec"},
				},
				attempt:       1,
				totalAttempts: 2,
				stage:         "implementing",
				started:       true,
			},
		}

		m.resize()

		h := lipgloss.Height(m.View())
		if h > m.height {
			t.Fatalf("running view height %d exceeds window height %d for height %d (chrome=%d)", h, m.height, tc.height, m.logsChromeHeight())
		}
	}
}
