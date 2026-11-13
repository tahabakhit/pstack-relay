package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tahabakhit/pstack-rex/internal/resolve"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
)

func testCLI(t *testing.T, run runner) (cli, *bytes.Buffer, string) {
	t.Helper()
	out := new(bytes.Buffer)
	path := filepath.Join(t.TempDir(), "sheet.md")
	c := cli{out: out, errOut: out, in: strings.NewReader("y\n"), getenv: func(k string) string {
		if k == "PSTACK_REX_SHEET" {
			return path
		}
		return ""
	}, run: run}
	return c, out, path
}
func mockRun(name string, args ...string) (string, error) {
	switch name + " " + strings.Join(args, " ") {
	case "codex --version", "claude --version":
		return "version", nil
	case "codex debug models --bundled":
		return `{"models":[{"slug":"gpt-first"}]}`, nil
	case "codex features list":
		return "multi_agent stable true", nil
	case "git branch --show-current":
		return "topic\n", nil
	}
	return "", fmt.Errorf("unexpected command %s %v", name, args)
}
func TestModels(t *testing.T) {
	c, out, path := testCLI(t, mockRun)
	for _, tt := range []struct {
		value string
		seats int
		err   bool
	}{
		{"interrogate reviewers=claude:opus @high, codex:gpt @xhigh", 2, false},
		{"interrogate reviewers=auto", 1, false},
		{"interrogate reviewers=other:model", 0, true},

		{"role=opus\nother: model", 0, true},
	} {
		t.Run(tt.value, func(t *testing.T) {
			before, _ := os.ReadFile(path)
			out.Reset()
			err := c.execute([]string{"models", "--set", tt.value, "--json"})
			if tt.err {
				if err == nil {
					t.Fatal("expected error")
				}
				after, _ := os.ReadFile(path)
				if !bytes.Equal(before, after) {
					t.Fatal("invalid edit changed sheet")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s, err := readSheet(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Roles["interrogate reviewers"]) != tt.seats {
				t.Fatalf("%#v", s)
			}
			if !json.Valid(out.Bytes()) {
				t.Fatalf("invalid JSON: %s", out)
			}
		})
	}
}
func TestResolveCLI(t *testing.T) {
	for _, tt := range []struct {
		name   string
		args   []string
		parent sheet.Harness
		mode   resolve.Mode
		err    bool
	}{
		{"parent claude", []string{"--parent", "claude", "--backend", "orca"}, sheet.Claude, resolve.Native, false},
		{"parent codex", []string{"--parent", "codex", "--backend", "orca"}, sheet.Codex, resolve.OrcaMode, false},
		{"no backend", []string{"--parent", "codex", "--backend", "none"}, sheet.Codex, resolve.Fallback, false},
		{"worktree alias", []string{"--parent", "codex", "--backend", "orca", "--worktree-own"}, sheet.Codex, resolve.OrcaMode, false},
		{"invalid parent", []string{"--parent", "other"}, "", "", true},
		{"invalid backend", []string{"--backend", "other"}, "", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, out, path := testCLI(t, mockRun)
			if err := os.WriteFile(path, []byte("interrogate reviewers: claude:opus @xhigh, codex:gpt @high\n"), 0600); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"resolve", "--role", "interrogate reviewers"}, tt.args...)
			err := c.execute(args)
			if tt.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var placements []resolve.Placement
			if err := json.Unmarshal(out.Bytes(), &placements); err != nil {
				t.Fatal(err)
			}
			if len(placements) != 2 || placements[0].Mode != tt.mode {
				t.Fatalf("%s", out)
			}
			if tt.name == "worktree alias" && !strings.Contains(strings.Join(placements[0].Command, " "), "--base-branch topic") {
				t.Fatalf("%s", out)
			}
		})
	}
}
func TestSetup(t *testing.T) {
	for _, tt := range []struct {
		name, existing        string
		args                  []string
		installs, writes, err bool
	}{
		{"dry run new", "", []string{"--dry-run", "--json"}, false, false, false},
		{"new", "", []string{"--yes", "--json"}, true, true, false},
		{"preserve", "my role: auto\nsession hook: off\n", []string{"--yes", "--json"}, true, false, false},
		{"force", "my role: auto\n", []string{"--force", "--yes", "--json"}, true, true, false},
		{"invalid preserve", "broken", []string{"--yes", "--json"}, false, false, true},
		{"force invalid", "broken", []string{"--force", "--yes", "--json"}, true, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var commands [][]string
			c, out, path := testCLI(t, func(name string, args ...string) (string, error) {
				if len(args) > 0 && args[0] == "plugin" {
					commands = append(commands, append([]string{name}, args...))
					return "installed", nil
				}
				return mockRun(name, args...)
			})
			if tt.existing != "" {
				if err := os.WriteFile(path, []byte(tt.existing), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := c.execute(append([]string{"setup"}, tt.args...))
			if tt.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(out.Bytes()) {
				t.Fatalf("invalid JSON %s", out)
			}
			if (len(commands) > 0) != tt.installs {
				t.Fatalf("commands=%v", commands)
			}
			if tt.installs {
				want := [][]string{{"codex", "plugin", "marketplace", "add", "tahabakhit/pstack-rex"}, {"codex", "plugin", "add", "pstack-rex@pstack-rex"}, {"claude", "plugin", "marketplace", "add", "tahabakhit/pstack-rex"}, {"claude", "plugin", "install", "pstack-rex@pstack-rex"}}
				if !reflect.DeepEqual(commands, want) {
					t.Fatalf("commands=%v", commands)
				}
			}
			b, readErr := os.ReadFile(path)
			if tt.writes {
				if readErr != nil {
					t.Fatal(readErr)
				}
				s, err := sheet.Parse(string(b))
				if err != nil || len(s.Roles["interrogate reviewers"]) != 2 {
					t.Fatalf("sheet=%s err=%v", b, err)
				}
			} else if tt.existing != "" {
				if string(b) != tt.existing {
					t.Fatal("existing sheet changed")
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatal("dry run wrote sheet")
			}
		})
	}
}
func TestSetupFailureReported(t *testing.T) {
	c, out, _ := testCLI(t, func(name string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "plugin" {
			return "", fmt.Errorf("install failed")
		}
		return mockRun(name, args...)
	})
	if err := c.execute([]string{"setup", "--yes", "--json"}); err == nil {
		t.Fatal("setup hid failures")
	}
	var result setupResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, command := range result.Commands {
		if command.Error == "" {
			t.Fatalf("missing command error: %#v", command)
		}
	}
}
func TestSetupMultiAgentAdvice(t *testing.T) {
	c, out, _ := testCLI(t, func(name string, args ...string) (string, error) {
		if name == "codex" && len(args) > 0 && args[0] == "features" {
			return "multi_agent stable false", nil
		}
		return mockRun(name, args...)
	})
	if err := c.execute([]string{"setup", "--dry-run", "--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "codex features enable multi_agent") {
		t.Fatal(out)
	}
}

func TestDoctor(t *testing.T) {
	for _, tt := range []struct {
		name, text        string
		vars              map[string]string
		failOrca, wantErr bool
	}{
		{"healthy", "r: auto\n", map[string]string{"CODEX_THREAD_ID": "thread", "ORCA_TERMINAL_HANDLE": "term"}, false, false},
		{"invalid sheet", "broken", map[string]string{"CODEX_THREAD_ID": "thread"}, false, true},
		{"unknown parent", "r: auto\n", nil, false, true},
		{"missing detected backend", "r: auto\n", map[string]string{"CODEX_THREAD_ID": "thread", "ORCA_TERMINAL_HANDLE": "term"}, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, out, path := testCLI(t, func(name string, args ...string) (string, error) {
				if name == "orca" && !tt.failOrca {
					return "1.4.257", nil
				}
				return mockRun(name, args...)
			})
			c.getenv = func(k string) string {
				if k == "PSTACK_REX_SHEET" {
					return path
				}
				return tt.vars[k]
			}
			if err := os.WriteFile(path, []byte(tt.text), 0600); err != nil {
				t.Fatal(err)
			}
			err := c.execute([]string{"doctor", "--json"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v wantErr=%t", err, tt.wantErr)
			}
			var result struct {
				OK     bool          `json:"ok"`
				Checks []doctorCheck `json:"checks"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.OK == tt.wantErr || len(result.Checks) < 5 {
				t.Fatalf("%s", out)
			}
		})
	}
}

func TestWriteSheetKeepsSheetThatAppeared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pstack-models.md")
	if err := os.WriteFile(path, []byte("interrogate reviewers: codex:mine @low\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := sheet.Parse("interrogate reviewers: claude:opus\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSheet(path, s, false); !errors.Is(err, os.ErrExist) {
		t.Fatalf("want ErrExist, got %v", err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "codex:mine") {
		t.Fatalf("existing sheet replaced: %s", b)
	}
	if err := writeSheet(path, s, true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "claude:opus") {
		t.Fatalf("--force did not replace: %s", b)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".pstack-models-*")); len(left) > 0 {
		t.Fatalf("temporary files left: %v", left)
	}
}
