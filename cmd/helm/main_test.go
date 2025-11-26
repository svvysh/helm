package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAAACmdAppearsInHelp(t *testing.T) {
	t.Setenv("HELM_CONFIG_DIR", t.TempDir())
	repoRoot := findRepoRoot(t)
	switchTo(t, repoRoot)

	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}

	if !strings.Contains(buf.String(), "aaa") {
		t.Fatalf("expected aaa command in help output; got %q", buf.String())
	}
}

func TestAAACmdPrintsMessage(t *testing.T) {
	t.Setenv("HELM_CONFIG_DIR", t.TempDir())
	repoRoot := findRepoRoot(t)
	switchTo(t, repoRoot)

	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"aaa"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute aaa: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "aaa") {
		t.Fatalf("expected aaa command output to contain 'aaa'; got %q", output)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	dir := cwd
	for {
		if dir == "." || dir == "/" {
			t.Fatalf("could not find go.mod above %s", cwd)
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("reached filesystem root while searching for go.mod from %s", cwd)
		}
		dir = parent
	}
}

func switchTo(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
}
