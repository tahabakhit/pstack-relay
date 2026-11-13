package plugin

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	hook, err := os.ReadFile("../../overlay/hooks/session-start.sh")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"internal/plugin/adaptations.json": "[]", "upstream/skills/example/SKILL.md": "---\nname: example\n---\nbody\n", "upstream/skills/example/references/ref.md": "unchanged\n", "upstream/agents/example.md": "agent\n", "upstream/LICENSE": "MIT\n", "upstream/README.md": "README\n", "overlay/delegation.md": "delegation\n", "overlay/hooks/session-start.sh": string(hook)} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func TestBuild(t *testing.T) {
	root := fixture(t)
	if err := Build(root, false); err != nil {
		t.Fatal(err)
	}
	if err := Build(root, true); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, path, content string }{
		{"entrypoint pointer", "plugins/pstack-rex/skills/example/SKILL.md", "> Read [pstack delegation on this host](../../delegation.md) before following this file. It maps Cursor tools and models to this host and says how to start every subagent."},
		{"agent pointer", "plugins/pstack-rex/agents/example.md", "(../delegation.md)"},
		{"reference unchanged", "plugins/pstack-rex/skills/example/references/ref.md", "unchanged\n"},
		{"verbatim delegation", "plugins/pstack-rex/delegation.md", "delegation\n"},
		{"license", "plugins/pstack-rex/LICENSE", "MIT\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(root, tt.path))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), tt.content) {
				t.Fatalf("got %s", b)
			}
		})
	}
	for _, name := range []string{".agents/plugins/marketplace.json", ".claude-plugin/marketplace.json", "plugins/pstack-rex/.codex-plugin/plugin.json", "plugins/pstack-rex/.claude-plugin/plugin.json", "plugins/pstack-rex/hooks/hooks.json", "plugins/pstack-rex/hooks/codex-hooks.json"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(b) {
			t.Fatalf("invalid JSON %s", name)
		}
	}
	for _, tt := range []struct {
		name   string
		mutate func(string) error
	}{
		{"stale file", func(root string) error {
			return os.WriteFile(filepath.Join(root, "plugins/pstack-rex/delegation.md"), []byte("stale"), 0644)
		}},
		{"missing file", func(root string) error { return os.Remove(filepath.Join(root, "plugins/pstack-rex/LICENSE")) }},
		{"extra file", func(root string) error {
			return os.WriteFile(filepath.Join(root, "plugins/pstack-rex/extra"), nil, 0644)
		}},
		{"non-executable hook", func(root string) error {
			return os.Chmod(filepath.Join(root, "plugins/pstack-rex/hooks/session-start.sh"), 0644)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mutate(root); err != nil {
				t.Fatal(err)
			}
			if err := Build(root, true); err == nil {
				t.Fatal("check accepted drift")
			}
			if err := Build(root, false); err != nil {
				t.Fatal(err)
			}
			if err := Build(root, true); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestSessionHook(t *testing.T) {
	for _, tt := range []struct {
		name, sheet string
		empty       bool
	}{
		{"default", "", false}, {"off", "session hook: off\n", true}, {"whitespace", " session hook : off  \n", true}, {"last assignment", "session hook: off\nsession hook: on\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := fixture(t)
			if err := Build(root, false); err != nil {
				t.Fatal(err)
			}
			// Exercise the installed script under a path that requires JSON escaping.
			pluginPath := filepath.Join(root, "plugin \"quoted\"\\name\tpath\nline")
			if err := os.Rename(filepath.Join(root, "plugins/pstack-rex"), pluginPath); err != nil {
				t.Fatal(err)
			}
			sheetPath := filepath.Join(root, "sheet with spaces.md")
			if tt.sheet != "" {
				if err := os.WriteFile(sheetPath, []byte(tt.sheet), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("sh", filepath.Join(pluginPath, "hooks/session-start.sh"), "codex")
			cmd.Env = append(os.Environ(), "PSTACK_REX_SHEET="+sheetPath)
			b, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			if tt.empty {
				if len(b) > 0 {
					t.Fatalf("hook disabled: %s", b)
				}
				return
			}
			var v struct {
				Output struct {
					Event   string `json:"hookEventName"`
					Context string `json:"additionalContext"`
				} `json:"hookSpecificOutput"`
			}
			if err := json.Unmarshal(b, &v); err != nil {
				t.Fatalf("invalid JSON: %s: %v", b, err)
			}
			if v.Output.Event != "SessionStart" || !strings.Contains(v.Output.Context, filepath.Join(pluginPath, "delegation.md")) {
				t.Fatalf("wrong installed pointer: %s", b)
			}
		})
	}
}
