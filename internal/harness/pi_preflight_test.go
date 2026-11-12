package harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pstack-rex/pstack-rex/internal/config"
)

func TestPiSelectionBeforeInference(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		exit           bool
		wantError      bool
	}{
		{"exact", `{"type":"response","id":"selection","command":"get_state","success":true,"data":{"model":{"id":"exact","provider":"openai"},"thinkingLevel":"low"}}`, false, false},
		{"fuzzy", `{"type":"response","id":"selection","command":"get_state","success":true,"data":{"model":{"id":"other-model","provider":"openai"},"thinkingLevel":"low"}}`, false, true},
		{"effort-normalized", `{"type":"response","id":"selection","command":"get_state","success":true,"data":{"model":{"id":"exact","provider":"openai"},"thinkingLevel":"medium"}}`, false, true},
		{"unrelated", `{"type":"response","id":"another-request","success":true}`, false, true},
		{"process-error", `{}`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			exe := filepath.Join(dir, "pi")
			script := "#!/bin/sh\n/bin/cat > \"$0.input\"\nprintf '%s\\n' \"$PSTACK_TEST_STATE\"\n"
			if tc.exit {
				script += "exit 7\n"
			}
			if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			plan := Plan{Executable: exe, Args: []string{"--mode", "json", "--print", "--model", "openai/exact"}, Environment: []string{"PSTACK_TEST_STATE=" + tc.response}}
			selection, err := ProbePi(context.Background(), plan, config.Target{Model: "openai/exact", Effort: "low"}, dir)
			if (err != nil) != tc.wantError {
				t.Fatalf("selection=%+v err=%v", selection, err)
			}
			input, err := os.ReadFile(exe + ".input")
			if err != nil {
				t.Fatal(err)
			}
			if string(input) != "{\"id\":\"selection\",\"type\":\"get_state\"}\n" {
				t.Fatalf("preflight sent unexpected input %q", input)
			}
			if tc.wantError && strings.Contains(string(input), `"type":"prompt"`) {
				t.Fatal("inference sent before model validation")
			}
		})
	}
}
