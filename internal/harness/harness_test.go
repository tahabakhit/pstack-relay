package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pstack-rex/pstack-rex/internal/config"
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
