package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/runner"
	"github.com/polarzero/helm/internal/specs"
)

type runnerStartMsg struct {
	stream <-chan tea.Msg
	err    error
}

type runnerLogMsg struct {
	stream string
	text   string
}

type runnerFinishedMsg struct {
	err      error
	exitCode int
}

type runnerStreamClosedMsg struct{}

func startRunnerCmd(opts Options, folder *specs.SpecFolder) tea.Cmd {
	return func() tea.Msg {
		if folder == nil {
			return runnerStartMsg{err: fmt.Errorf("spec folder is nil")}
		}
		stream := make(chan tea.Msg)

		// Line emitters route stdout/stderr into the TUI.
		outEmitter := newLineEmitter("stdout", stream)
		errEmitter := newLineEmitter("stderr", stream)

		go func() {
			defer close(stream)
			defer outEmitter.close()
			defer errEmitter.close()

			r := &runner.Runner{
				Root:                      opts.Root,
				SpecsRoot:                 opts.SpecsRoot,
				Mode:                      opts.Settings.Mode,
				MaxAttempts:               opts.Settings.DefaultMaxAttempts,
				WorkerChoice:              opts.Settings.CodexRunImpl,
				VerifierChoice:            opts.Settings.CodexRunVer,
				DefaultAcceptanceCommands: opts.Settings.AcceptanceCommands,
				Stdout:                    outEmitter,
				Stderr:                    errEmitter,
			}
			// Pass folder ID; runner resolves paths using SpecsRoot.
			err := r.Run(context.Background(), folder.Metadata.ID)
			exitCode := 0
			if err != nil {
				exitCode = 1
			}
			stream <- runnerFinishedMsg{err: err, exitCode: exitCode}
		}()

		return runnerStartMsg{stream: stream}
	}
}

func listenStream(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return runnerStreamClosedMsg{}
		}
		return msg
	}
}

var attemptRe = regexp.MustCompile(`Attempt\s+(\d+)\s+of\s+(\d+)`)

func parseAttemptLine(line string) (int, int, bool) {
	matches := attemptRe.FindStringSubmatch(line)
	if len(matches) != 3 {
		return 0, 0, false
	}
	current, err1 := strconv.Atoi(matches[1])
	total, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil || current <= 0 || total <= 0 {
		return 0, 0, false
	}
	return current, total, true
}

func parseRemainingTasks(reportPath string) []string {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil
	}
	idx := bytes.Index(bytes.ToLower(data), []byte("## remaining tasks"))
	if idx == -1 {
		return nil
	}
	section := data[idx:]
	start := bytes.IndexByte(section, '{')
	if start == -1 {
		return nil
	}
	section = section[start:]
	dec := json.NewDecoder(bytes.NewReader(section))
	var payload struct {
		RemainingTasks []string `json:"remainingTasks"`
	}
	if err := dec.Decode(&payload); err != nil {
		return nil
	}
	return payload.RemainingTasks
}

// lineEmitter buffers writes and emits whole lines as runnerLogMsg.
type lineEmitter struct {
	stream string
	ch     chan<- tea.Msg
	mu     sync.Mutex
	buf    bytes.Buffer
	closed bool
}

func newLineEmitter(stream string, ch chan<- tea.Msg) *lineEmitter {
	return &lineEmitter{stream: stream, ch: ch}
}

func (w *lineEmitter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.EOF
	}
	n, _ := w.buf.Write(p)
	for {
		data := w.buf.Bytes()
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			line := string(data[:idx])
			w.ch <- runnerLogMsg{stream: w.stream, text: line}
			w.buf.Next(idx + 1)
		} else {
			break
		}
	}
	return n, nil
}

func (w *lineEmitter) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	if rem := strings.TrimSpace(w.buf.String()); rem != "" {
		w.ch <- runnerLogMsg{stream: w.stream, text: rem}
	}
	w.closed = true
}
