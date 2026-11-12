package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIProcess(t *testing.T) {
	if os.Getenv("PSTACK_TEST_CLI") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func cli(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "PSTACK_TEST_CLI=1")
	return cmd.CombinedOutput()
}

func TestSetupModelsAndFailureExitCodes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	args := []string{"--config", path, "setup", "--role", "how explorer", "--target", "codex:fixture-model@high"}
	if out, err := cli(t, args...); err != nil {
		t.Fatalf("setup: %v %s", err, out)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := cli(t, args...); err == nil {
		t.Fatalf("setup overwrote config: %s", out)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("existing configuration changed: %v", err)
	}
	out, err := cli(t, "--config", path, "models")
	if err != nil {
		t.Fatalf("models: %v %s", err, out)
	}
	var models struct {
		Roles map[string][]struct{ Model, Effort string } `json:"roles"`
	}
	if err = json.Unmarshal(out, &models); err != nil {
		t.Fatal(err)
	}
	seats := models.Roles["how explorer"]
	if len(seats) != 1 || seats[0].Model != "fixture-model" || seats[0].Effort != "high" {
		t.Fatalf("models=%s", out)
	}
	for _, args := range [][]string{
		{"--config", path, "dispatch", "--id", "attempt"},
		{"--config", path, "inspect", "--id", "../escape"},
		{"--config", path, "wait", "--id", "attempt", "--seconds", "61"},
		{"--config", path, "unknown"},
		{"version", "unexpected"},
	} {
		if out, err := cli(t, args...); err == nil {
			t.Fatalf("invalid invocation succeeded %v: %s", args, out)
		}
	}
}
