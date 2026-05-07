package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	oldExit := exitFunc
	exitFunc = func(int) {}
	defer func() { exitFunc = oldExit }()

	code := run([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run(--help) = %d", code)
	}
	if !strings.Contains(out.String(), "Slack Web API CLI") {
		t.Fatalf("help output = %q", out.String())
	}
}

func TestMainCallsRun(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oldArgs := os.Args
	oldExit := exitFunc

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout) error = %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stderr) error = %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr
	os.Args = []string{"slack-cli", "--help"}
	exitFunc = func(int) {}
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		os.Args = oldArgs
		exitFunc = oldExit
	}()

	main()
	_ = wOut.Close()
	_ = wErr.Close()
	_, _ = out.ReadFrom(rOut)
	_, _ = errOut.ReadFrom(rErr)

	if !strings.Contains(out.String(), "Slack Web API CLI") {
		t.Fatalf("main help output = %q stderr=%q", out.String(), errOut.String())
	}
}

func TestRunBadArgs(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"nope"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(bad args) = %d, want 2", code)
	}
}

func TestRunSettingsError(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"--json", "--plain", "auth", "status"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run(settings error) = %d, want 2", code)
	}
}

func TestRunSuccessAndCommandError(t *testing.T) {
	t.Setenv("SLACK_CLI_TOKEN", "")
	t.Setenv("SLACK_TOKEN", "")

	var out bytes.Buffer
	var errOut bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "config.json")

	if code := run([]string{"--config", configPath, "auth", "status"}, &out, &errOut); code != 0 {
		t.Fatalf("run(auth status) = %d, want 0", code)
	}
	if code := run([]string{"--config", configPath, "auth", "test"}, &out, &errOut); code != 1 {
		t.Fatalf("run(auth test no token) = %d, want 1", code)
	}
}

func TestRunContextError(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(configPath, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if code := run([]string{"--config", configPath, "auth", "status"}, &out, &errOut); code != 1 {
		t.Fatalf("run(invalid config) = %d, want 1", code)
	}
}
