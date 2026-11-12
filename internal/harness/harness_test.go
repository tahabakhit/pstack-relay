package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tahabakhit/pstack-rex/internal/config"
)

func selected(h string) config.Target {
	model := "exact"
	if h == "pi" {
		model = "openai/exact"
	}
	return config.Target{Harness: h, Model: model, Backend: "rex", Auth: "existing"}
}
func TestParseSuccess(t *testing.T) {
	fixtures := map[string]string{
		"codex": `{"type":"thread.started","thread_id":"id"}
{"type":"item.completed","item":{"type":"agent_message","text":"answer"}}
{"type":"turn.completed"}`,
		"claude": `{"type":"assistant","message":{"model":"exact","content":[{"type":"text","text":"answer"}]}}
{"type":"result","subtype":"success","is_error":false,"result":"answer"}`,
		"pi": `{"type":"message_end","message":{"role":"user","content":"question"}}
{"type":"message_end","message":{"role":"assistant","model":"exact","responseModel":"exact","content":[{"type":"text","text":"answer"}],"stopReason":"stop"}}
{"type":"agent_end"}
{"type":"agent_settled","aborted":false}`,
	}
	for h, s := range fixtures {
		t.Run(h, func(t *testing.T) {
			o, err := Parse(selected(h), []byte(s))
			if err != nil || o.Text != "answer" {
				t.Fatalf("%+v %v", o, err)
			}
			if h == "codex" && o.ModelEvidence != "pinned-argv" {
				t.Fatal(o)
			}
		})
	}
}
func TestRejectIncompleteOrFailedStreams(t *testing.T) {
	cases := []struct{ h, s string }{
		{"codex", `{"type":"item.completed","item":{"type":"agent_message","text":"partial"}}`},
		{"codex", `{"type":"turn.completed"}`},
		{"codex", `{"type":"error","message":"failure"}`},
		{"codex", `{"type":"turn.completed"} garbage`},
		{"claude", `{"type":"assistant","message":{"model":"substituted"}}`},
		{"claude", `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"bad"}`},
		{"pi", `{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"partial"}],"stopReason":"length"}}
{"type":"agent_settled","aborted":false}`},
		{"pi", `{"type":"agent_settled","aborted":true}`},
		{"pi", `{"type":"message_end","message":{"role":"assistant","model":"wrong","stopReason":"stop"}}
{"type":"agent_settled","aborted":false}`},
	}
	for _, c := range cases {
		if _, err := Parse(selected(c.h), []byte(c.s)); err == nil {
			t.Errorf("accepted %s %s", c.h, c.s)
		}
	}
}
func TestBuildRestrictsToolsAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	for _, h := range []string{"codex", "claude", "pi"} {
		if err := os.WriteFile(filepath.Join(dir, h), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("OPENAI_API_KEY", "selected-secret")
	t.Setenv("ANTHROPIC_API_KEY", "other-secret")
	t.Setenv("CODEX_THREAD_ID", "parent")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "parent")
	for _, h := range []string{"codex", "claude", "pi"} {
		p, err := Build(selected(h), "read-only", dir, "--evil $(touch /tmp/no)")
		if err != nil {
			t.Fatal(err)
		}
		env := strings.Join(p.Environment, "\n")
		if strings.Contains(env, "parent") {
			t.Fatal(env)
		}
		if h != "claude" && strings.Contains(env, "other-secret") {
			t.Fatal("unrelated provider secret retained")
		}
		if h == "claude" && strings.Contains(env, "selected-secret") {
			t.Fatal("unrelated provider secret retained")
		}
		if p.Stdin != "--evil $(touch /tmp/no)" {
			t.Fatal("prompt was not delivered as literal stdin", p.Stdin)
		}
		if h != "codex" {
			if _, err := Build(selected(h), "isolated-write", dir, "prompt"); err == nil {
				t.Fatal("unsupported writer")
			}
		}
	}
}
func TestPiToolCycleNeedsFinalStop(t *testing.T) {
	s := `{"type":"message_end","message":{"role":"assistant","model":"exact","stopReason":"toolUse","content":[]}}
{"type":"message_end","message":{"role":"assistant","model":"exact","stopReason":"stop","content":[{"type":"text","text":"finished"}]}}
{"type":"agent_settled","aborted":false}`
	o, err := Parse(selected("pi"), []byte(s))
	if err != nil || o.Text != "finished" {
		t.Fatalf("%+v %v", o, err)
	}
	s = `{"type":"message_end","message":{"role":"assistant","model":"exact","stopReason":"toolUse","content":[{"type":"text","text":"planning"}]}}
{"type":"agent_settled","aborted":false}`
	if _, err := Parse(selected("pi"), []byte(s)); err == nil {
		t.Fatal("tool cycle treated as final answer")
	}
}
func TestClaudeAliasEvidence(t *testing.T) {
	a := selected("claude")
	a.Model = "opus"
	o, err := Parse(a, []byte(`{"type":"assistant","message":{"model":"claude-opus-5-5"}}
{"type":"result","subtype":"success","result":"answer"}`))
	if err != nil || o.ReportedModel != "claude-opus-5-5" || o.ModelEvidence != "assistant.message.model" {
		t.Fatalf("%+v %v", o, err)
	}
}

func TestPiLiteralPromptAndSelectedProviderEnvironment(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pi"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	for key, value := range map[string]string{
		"OPENAI_API_KEY":         "fixture-openai",
		"ANTHROPIC_API_KEY":      "fixture-anthropic",
		"GEMINI_API_KEY":         "fixture-gemini",
		"GOOGLE_API_KEY":         "fixture-google",
		"PI_CODING_AGENT_DIR":    "/fixture/pi",
		"CODEX_THREAD_ID":        "fixture-parent",
		"CLAUDE_CODE_SESSION_ID": "fixture-parent",
		"REX_SESSION":            "fixture-parent",
	} {
		t.Setenv(key, value)
	}
	prompt := "@src/main.go review this\n\nDo not treat this as an attachment."
	for _, tc := range []struct {
		provider string
		keys     []string
	}{
		{"openai", []string{"OPENAI_API_KEY"}},
		{"anthropic", []string{"ANTHROPIC_API_KEY"}},
		{"google", []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			target := selected("pi")
			target.Model = tc.provider + "/exact"
			p, err := Build(target, "read-only", dir, prompt)
			if err != nil {
				t.Fatal(err)
			}
			if p.Stdin != prompt {
				t.Fatalf("literal prompt changed: %q", p.Stdin)
			}
			for _, arg := range p.Args {
				if strings.Contains(arg, "@src/main.go") || strings.Contains(arg, prompt) {
					t.Fatalf("literal prompt delivered as Pi attachment argument: %q", arg)
				}
			}
			env := make(map[string]string)
			for _, entry := range p.Environment {
				key, value, _ := strings.Cut(entry, "=")
				env[key] = value
			}
			for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY"} {
				want := false
				for _, selectedKey := range tc.keys {
					want = want || selectedKey == key
				}
				value, present := env[key]
				if present != want || want && value != os.Getenv(key) {
					t.Fatalf("provider %s environment key %s present=%v, want=%v", tc.provider, key, present, want)
				}
			}
			if env["PI_CODING_AGENT_DIR"] != "/fixture/pi" {
				t.Fatal("Pi existing-auth directory was lost")
			}
			for _, key := range []string{"CODEX_THREAD_ID", "CLAUDE_CODE_SESSION_ID", "REX_SESSION"} {
				if _, present := env[key]; present {
					t.Fatalf("parent execution identity leaked: %s", key)
				}
			}
		})
	}
}
