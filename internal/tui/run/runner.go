package run

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/specs"
)

type runnerStartMsg struct {
	cmd    *exec.Cmd
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
		script := filepath.Join(opts.SpecsRoot, "implement-spec.mjs")
		if _, err := os.Stat(script); err != nil {
			return runnerStartMsg{err: fmt.Errorf("implement-spec runner not found at %s: %w", script, err)}
		}
		if folder == nil {
			return runnerStartMsg{err: fmt.Errorf("spec folder is nil")}
		}
		cmd := exec.Command("node", script, folder.Path)
		if opts.Root != "" {
			cmd.Dir = opts.Root
		}
		cmd.Env = buildRunnerEnv(os.Environ(), opts.Settings)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return runnerStartMsg{err: fmt.Errorf("stdout pipe: %w", err)}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return runnerStartMsg{err: fmt.Errorf("stderr pipe: %w", err)}
		}

		if err := cmd.Start(); err != nil {
			return runnerStartMsg{err: err}
		}

		stream := make(chan tea.Msg)
		var wg sync.WaitGroup
		wg.Add(2)
		go streamPipe(stdout, "stdout", stream, &wg)
		go streamPipe(stderr, "stderr", stream, &wg)
		go waitForExit(cmd, stream, &wg)

		return runnerStartMsg{cmd: cmd, stream: stream}
	}
}

func streamPipe(r io.Reader, label string, ch chan<- tea.Msg, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 512*1024)
	for scanner.Scan() {
		line := scanner.Text()
		ch <- runnerLogMsg{stream: label, text: line}
	}
	if err := scanner.Err(); err != nil {
		ch <- runnerLogMsg{stream: "stderr", text: fmt.Sprintf("[%s reader error] %v", label, err)}
	}
}

func waitForExit(cmd *exec.Cmd, ch chan<- tea.Msg, wg *sync.WaitGroup) {
	err := cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	wg.Wait()
	ch <- runnerFinishedMsg{err: err, exitCode: exitCode}
	close(ch)
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

func buildRunnerEnv(base []string, settings *config.Settings) []string {
	env := make([]string, len(base))
	copy(env, base)
	if settings == nil {
		return env
	}
	if !envHas(env, "MAX_ATTEMPTS") && settings.DefaultMaxAttempts > 0 {
		env = append(env, fmt.Sprintf("MAX_ATTEMPTS=%d", settings.DefaultMaxAttempts))
	}
	if !envHas(env, "CODEX_MODEL_IMPL") && settings.CodexRunImpl.Model != "" {
		env = append(env, fmt.Sprintf("CODEX_MODEL_IMPL=%s", settings.CodexRunImpl.Model))
	}
	if !envHas(env, "CODEX_MODEL_VER") && settings.CodexRunVer.Model != "" {
		env = append(env, fmt.Sprintf("CODEX_MODEL_VER=%s", settings.CodexRunVer.Model))
	}
	return env
}

func envHas(env []string, key string) bool {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return true
		}
	}
	return false
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
