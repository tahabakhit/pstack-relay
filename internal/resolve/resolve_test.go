package resolve

import (
	"github.com/tahabakhit/pstack-rex/internal/sheet"
	"reflect"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	for _, tt := range []struct {
		name string
		vars map[string]string
		want Env
	}{
		{"unknown", nil, Env{"", None}},
		{"claude priority", map[string]string{"CLAUDECODE": "1", "CODEX_THREAD_ID": "thread", "PI_SESSION": "session"}, Env{sheet.Claude, None}},
		{"codex", map[string]string{"CODEX_THREAD_ID": "thread"}, Env{sheet.Codex, None}},
		{"pi flag", map[string]string{"PI_CODING_AGENT": "1"}, Env{sheet.Pi, None}},
		{"pi session", map[string]string{"PI_SESSION": "session"}, Env{sheet.Pi, None}},
		{"orca priority", map[string]string{"ORCA_TERMINAL_HANDLE": "term", "REX_SESSION": "s", "REX_BLOCK": "b"}, Env{"", Orca}},
		{"rex", map[string]string{"REX_SESSION": "s", "REX_BLOCK": "b"}, Env{"", Rex}},
		{"incomplete rex", map[string]string{"REX_SESSION": "s"}, Env{"", None}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(func(k string) string { return tt.vars[k] })
			if got != tt.want {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
		})
	}
}
func TestResolve(t *testing.T) {
	for _, tt := range []struct {
		name, text, role string
		env              Env
		opts             Options
		want             []Placement
		err              string
	}{
		{name: "missing role", text: "", role: "missing", env: Env{sheet.Codex, Orca}, want: []Placement{{Role: "missing", Harness: sheet.Codex, Mode: Native, Note: "role not in sheet; use the skill's default model"}}},
		{name: "bare native inherits default", text: "r: opus\ndefault effort: high", role: "r", env: Env{sheet.Claude, Orca}, want: []Placement{{Role: "r", Harness: sheet.Claude, Mode: Native, Model: "opus", Effort: "high"}}},
		{name: "same harness native and explicit session", text: "r: codex:gpt @session\ndefault effort: high", role: "r", env: Env{sheet.Codex, Orca}, want: []Placement{{Role: "r", Harness: sheet.Codex, Mode: Native, Model: "gpt"}}},
		{name: "aliases", text: "r: inherit-parent, auto @high", role: "r", env: Env{sheet.Codex, None}, want: []Placement{{Role: "r", Harness: sheet.Codex, Mode: Native}, {Role: "r", Seat: 1, Harness: sheet.Codex, Mode: Native, Effort: "high"}}},
		{name: "qualified aliases retain parent model", text: "r: claude:inherit-parent, pi:auto @high", role: "r", env: Env{sheet.Codex, Orca}, want: []Placement{{Role: "r", Harness: sheet.Codex, Mode: Native}, {Role: "r", Seat: 1, Harness: sheet.Codex, Mode: Native, Effort: "high"}}},
		{name: "orca shared", text: "r: claude:opus @xhigh", role: "r", env: Env{sheet.Codex, Orca}, want: []Placement{{Role: "r", Harness: sheet.Claude, Mode: OrcaMode, Model: "opus", Effort: "xhigh", Command: []string{"orca", "orchestration", "worker-start", "--agent", "claude", "--model", "opus", "--effort", "xhigh", "--worktree", "current", "--spec-file", "{brief}"}}}},
		{name: "orca own no effort", text: "r: codex:gpt\ndefault effort: session", role: "r", env: Env{sheet.Claude, Orca}, opts: Options{OwnWorktree: true, Branch: "topic"}, want: []Placement{{Role: "r", Harness: sheet.Codex, Mode: OrcaMode, Model: "gpt", Command: []string{"orca", "orchestration", "worker-start", "--agent", "codex", "--model", "gpt", "--worktree", "new-child", "--base-branch", "topic", "--spec-file", "{brief}"}}}},
		{name: "rex own", text: "r: claude:opus\ndefault effort: max", role: "r", env: Env{sheet.Codex, Rex}, opts: Options{OwnWorktree: true}, want: []Placement{{Role: "r", Harness: sheet.Claude, Mode: RexMode, Model: "opus", Effort: "max", Command: []string{"pstack-rex", "rex", "start", "--harness", "claude", "--model", "opus", "--effort", "max", "--own-worktree", "--brief", "{brief}"}}}},
		{name: "rex pi custom binary", text: "r: pi:openai/gpt", role: "r", env: Env{sheet.Codex, Rex}, opts: Options{RexBinary: "/path/pstack-rex"}, want: []Placement{{Role: "r", Harness: sheet.Pi, Mode: RexMode, Model: "openai/gpt", Command: []string{"/path/pstack-rex", "rex", "start", "--harness", "pi", "--model", "openai/gpt", "--brief", "{brief}"}}}},
		{name: "no backend fallback", text: "r: claude:opus @high", role: "r", env: Env{sheet.Codex, None}, want: []Placement{{Role: "r", Harness: sheet.Codex, Mode: Fallback, Effort: "high", Note: "configured seat claude:opus: no delegation backend detected; fall back to parent model"}}},
		{name: "pi orca fallback", text: "r: pi:openai/gpt", role: "r", env: Env{sheet.Codex, Orca}, want: []Placement{{Role: "r", Harness: sheet.Codex, Mode: Fallback, Note: "configured seat pi:openai/gpt: Orca has no pi agent; fall back to parent model"}}},
		{name: "unknown parent", text: "r: claude:opus @high, codex:gpt", role: "r", env: Env{"", Orca}, want: []Placement{{Role: "r", Mode: Native, Model: "opus", Effort: "high", Note: "parent harness not detected"}, {Role: "r", Seat: 1, Mode: Native, Model: "gpt", Note: "parent harness not detected"}}},
		{name: "bad backend", text: "", role: "r", env: Env{sheet.Codex, "bad"}, err: "unknown backend"},
		{name: "bad parent", text: "", role: "r", env: Env{"bad", None}, err: "unknown parent"},
		{name: "detached own", text: "r: claude:opus", role: "r", env: Env{sheet.Codex, Orca}, opts: Options{OwnWorktree: true}, err: "current branch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, err := sheet.Parse(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Resolve(s, tt.role, tt.env, tt.opts)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}
func TestPanelParentReversal(t *testing.T) {
	s, err := sheet.Parse("interrogate reviewers: claude:opus @xhigh, codex:gpt @high")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		parent Harness
		want   []Mode
	}{{sheet.Claude, []Mode{Native, OrcaMode}}, {sheet.Codex, []Mode{OrcaMode, Native}}} {
		t.Run(string(tt.parent), func(t *testing.T) {
			p, err := Resolve(s, "interrogate reviewers", Env{tt.parent, Orca}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if p[0].Mode != tt.want[0] || p[1].Mode != tt.want[1] {
				t.Fatalf("%#v", p)
			}
		})
	}
}
