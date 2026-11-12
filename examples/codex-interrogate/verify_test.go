package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tahabakhit/pstack-rex/internal/config"
	"github.com/tahabakhit/pstack-rex/internal/engine"
	"github.com/tahabakhit/pstack-rex/internal/harness"
)

type traceBuilder struct {
	events []map[string]any
	next   int
}

func (b *traceBuilder) item(item map[string]any) {
	b.next++
	item["id"] = "item_" + string(rune('A'+b.next))
	started := map[string]any{}
	for k, v := range item {
		started[k] = v
	}
	started["status"] = "in_progress"
	b.events = append(b.events, map[string]any{"type": "item.started", "item": started}, map[string]any{"type": "item.completed", "item": item})
}
func (b *traceBuilder) command(command, output string, exit int) {
	status := "completed"
	if exit != 0 {
		status = "failed"
	}
	b.item(map[string]any{"type": "command_execution", "command": "/usr/bin/zsh -lc " + shellQuote(command), "aggregated_output": output, "exit_code": exit, "status": status})
}
func (b *traceBuilder) call(tool string, args, value any) {
	data, _ := json.Marshal(value)
	b.item(map[string]any{"type": "mcp_tool_call", "server": "pstack-rex", "tool": tool, "arguments": args, "result": map[string]any{"content": []map[string]any{{"type": "text", "text": string(data)}}}, "status": "completed"})
}
func (b *traceBuilder) finish(text string) {
	b.events = append(b.events, map[string]any{"type": "item.completed", "item": map[string]any{"id": "final", "type": "agent_message", "text": text}}, map[string]any{"type": "turn.completed", "usage": map[string]int{}})
}
func (b *traceBuilder) bytes() []byte {
	var out []byte
	for _, e := range b.events {
		line, _ := json.Marshal(e)
		out = append(out, append(line, '\n')...)
	}
	return out
}
func TestReadTraceCommandTerminalStatus(t *testing.T) {
	for _, test := range []struct {
		name, status string
		exit         any
		wantError    bool
	}{
		{"success", "completed", 0, false},
		{"failed probe", "failed", 1, false},
		{"other nonzero exit", "failed", 2, false},
		{"failed zero exit", "failed", 0, true},
		{"completed nonzero exit", "completed", 1, true},
		{"missing exit", "failed", nil, true},
		{"cancelled", "cancelled", 1, true},
		{"incomplete", "in_progress", 1, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var b traceBuilder
			b.item(map[string]any{"type": "command_execution", "command": "python3 -B /proof/probe.py", "aggregated_output": reviewOutput, "exit_code": test.exit, "status": test.status})
			b.finish("probe executed")
			path := filepath.Join(t.TempDir(), "trace.jsonl")
			must(t, os.WriteFile(path, b.bytes(), 0600))
			trace, err := readTrace(path, config.Target{Harness: "codex", Model: "model-a", Effort: "medium", Backend: "rex", Auth: "existing"})
			if test.wantError {
				if err == nil {
					t.Fatal("invalid command terminal state accepted")
				}
				return
			}
			if err != nil {
				t.Fatalf("terminal command rejected: %v", err)
			}
			if len(trace.Commands) != 1 || trace.Commands[0].ExitCode != test.exit || trace.Commands[0].Output != reviewOutput {
				t.Fatalf("command evidence changed: %+v", trace.Commands)
			}
		})
	}
}

func TestReadTraceMCPRequiresCompletion(t *testing.T) {
	for _, status := range []string{"completed", "failed", "in_progress", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			var b traceBuilder
			b.call("capabilities", map[string]any{}, map[string]any{})
			b.events[1]["item"].(map[string]any)["status"] = status
			b.finish("tool returned")
			path := filepath.Join(t.TempDir(), "trace.jsonl")
			must(t, os.WriteFile(path, b.bytes(), 0600))
			_, err := readTrace(path, config.Target{Harness: "codex", Model: "model-a", Effort: "medium", Backend: "rex", Auth: "existing"})
			if (err == nil) != (status == "completed") {
				t.Fatalf("MCP status %q: %v", status, err)
			}
		})
	}
}

func testArtifacts(t *testing.T) (string, manifest) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "pstack-rex")
	cmd := exec.Command("go", "build", "-o", binary, "../../cmd/pstack-rex")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, data)
	}
	root := filepath.Join(t.TempDir(), "proof")
	must(t, os.Mkdir(root, 0700))
	cfg := config.Default()
	cfg.Roles[role] = []config.Target{{Harness: "codex", Model: "model-a", Effort: "medium", Backend: "rex", Auth: "existing"}, {Harness: "codex", Model: "model-b", Effort: "medium", Backend: "rex", Auth: "existing"}}
	o := options{Binary: binary, Out: root, ParentModel: "parent-model", ParentEffort: "medium"}
	m, err := prepare(o, cfg)
	must(t, err)
	must(t, writeJSON(filepath.Join(root, "parent-exit.json"), parentExit{ExitCode: 0}))
	must(t, writeJSON(filepath.Join(root, "parent-argv.json"), parentArgs(binary, root, m)))
	var parent traceBuilder
	for _, name := range installedReads {
		path := filepath.Join(root, "repo", ".agents", name)
		data, err := os.ReadFile(path)
		must(t, err)
		parent.command("cat "+path, string(data), 0)
	}
	parent.call("capabilities", map[string]any{}, map[string]any{"shell_harnesses": []string{"codex"}, "backends": []string{"rex"}})
	for _, s := range m.Seats {
		dir := filepath.Join(root, "state", "workers", s.ID)
		must(t, os.MkdirAll(dir, 0700))
		req := engine.Request{ID: s.ID, Role: role, Seat: s.Index, Cwd: s.Cwd, Prompt: "Break this change with the installed adversary brief", Access: "read-only", Target: s.Target, TimeoutSeconds: 600, RequireShell: true, ExpectedRevision: m.ReviewRevision, Executable: "/usr/bin/codex"}
		var worker traceBuilder
		worker.command("python3 -B "+filepath.Join(s.Cwd, "probe.py"), reviewOutput, 1)
		worker.finish("Last input is dropped. Observed singleton result 0, required 7.")
		data := worker.bytes()
		must(t, os.WriteFile(filepath.Join(dir, "stdout.jsonl"), data, 0600))
		outcome, err := harness.Parse(s.Target, data)
		must(t, err)
		start := engine.Started{PID: 1234, Session: "session-test", Block: "block-test", At: time.Now().UTC(), Revision: m.ReviewRevision, Cwd: s.Cwd, Enforcement: "sandbox-read-only"}
		launch := engine.Launch{Session: start.Session, Block: start.Block}
		snapshot := digest([]byte("head:" + m.ReviewRevision + "\x00diff:0\x00"))
		result := engine.Result{Status: "completed", ExitCode: 0, Outcome: &outcome, Finished: time.Now().UTC(), StartRevision: m.ReviewRevision, EndRevision: m.ReviewRevision, DiffSHA256: digest(nil), Verification: "not-performed", StartSnapshotSHA256: snapshot, EndSnapshotSHA256: snapshot}
		for name, v := range map[string]any{"request.json": req, "started.json": start, "launch.json": launch, "result.json": result} {
			must(t, writeJSON(filepath.Join(dir, name), v))
		}
		record := engine.Record{ID: s.ID, Status: "running", Role: role, Target: s.Target, Artifacts: dir, Launch: &launch}
		args := map[string]any{"id": req.ID, "role": req.Role, "seat": req.Seat, "cwd": req.Cwd, "prompt": req.Prompt, "access": req.Access, "require_shell": true, "expected_revision": req.ExpectedRevision}
		parent.call("dispatch", args, record)
		record.Status = "completed"
		record.Started = &start
		record.Result = &result
		parent.call("wait", map[string]any{"id": s.ID, "seconds": 60}, record)
	}
	parent.command("python3 -B "+filepath.Join(root, "repo", "probe.py"), reviewOutput, 1)
	finding := finalEvidence{Findings: []finding{{RootCause: "last-element-omitted", Command: probeCommand, Output: reviewOutput, RequiredOutput: baseOutput, Contributors: []int{0, 1}}}}
	data, _ := json.Marshal(finding)
	parent.finish("Intent: sum all inputs.\nFindings: one reproduced defect.\nDropped: zero per adversary.\n```pstack-rex-evidence\n" + string(data) + "\n```")
	must(t, os.WriteFile(filepath.Join(root, "parent.jsonl"), parent.bytes(), 0600))
	return root, m
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func mutateFile(t *testing.T, path string, fn func([]byte) []byte) {
	t.Helper()
	before, err := os.ReadFile(path)
	must(t, err)
	must(t, os.WriteFile(path, fn(before), 0600))
	t.Cleanup(func() { must(t, os.WriteFile(path, before, 0600)) })
}
func changeJSON(t *testing.T, path string, fn func(map[string]any)) {
	mutateFile(t, path, func(data []byte) []byte {
		var v map[string]any
		must(t, json.Unmarshal(data, &v))
		fn(v)
		out, err := json.Marshal(v)
		must(t, err)
		return out
	})
}
func changeEvents(t *testing.T, path string, fn func([]map[string]any) []map[string]any) {
	mutateFile(t, path, func(data []byte) []byte {
		var events []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var v map[string]any
			must(t, json.Unmarshal([]byte(line), &v))
			events = append(events, v)
		}
		return (&traceBuilder{events: fn(events)}).bytes()
	})
}
func TestRetainedEvidenceRejectsMutations(t *testing.T) {
	root, m := testArtifacts(t)
	positive, err := verify(root)
	must(t, err)
	if len(positive.Seats) != 2 || positive.Finding.RootCause != "last-element-omitted" {
		t.Fatal("missing complete panel and merged finding")
	}
	public, err := json.Marshal(positive)
	must(t, err)
	for _, private := range []string{root, "session-test", "block-test", "Last input is dropped", "/usr/bin/codex"} {
		if strings.Contains(string(public), private) {
			t.Fatalf("public evidence leaked %q", private)
		}
	}
	parent := filepath.Join(root, "parent.jsonl")
	seatDir := filepath.Join(root, "state", "workers", m.Seats[0].ID)
	request := filepath.Join(seatDir, "request.json")
	result := filepath.Join(seatDir, "result.json")
	raw := filepath.Join(seatDir, "stdout.jsonl")
	tests := []struct {
		name   string
		mutate func(*testing.T)
	}{
		{"missing seat", func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "removed")
			must(t, os.Rename(seatDir, dest))
			t.Cleanup(func() { must(t, os.Rename(dest, seatDir)) })
		}},
		{"extra attempt", func(t *testing.T) {
			path := filepath.Join(root, "state", "workers", "extra")
			must(t, os.Mkdir(path, 0700))
			t.Cleanup(func() { must(t, os.Remove(path)) })
		}},
		{"wrong role", func(t *testing.T) { changeJSON(t, request, func(v map[string]any) { v["role"] = "how explainer" }) }},
		{"wrong seat", func(t *testing.T) { changeJSON(t, request, func(v map[string]any) { v["seat"] = 1 }) }},
		{"wrong cwd", func(t *testing.T) { changeJSON(t, request, func(v map[string]any) { v["cwd"] = m.Seats[1].Cwd }) }},
		{"wrong revision", func(t *testing.T) {
			changeJSON(t, request, func(v map[string]any) { v["expected_revision"] = m.BaseRevision })
		}},
		{"wrong model", func(t *testing.T) {
			changeJSON(t, request, func(v map[string]any) { v["target"].(map[string]any)["model"] = "other-model" })
		}},
		{"no shell", func(t *testing.T) { changeJSON(t, request, func(v map[string]any) { v["require_shell"] = false }) }},
		{"write permission", func(t *testing.T) { changeJSON(t, request, func(v map[string]any) { v["access"] = "isolated-write" }) }},
		{"worker failure", func(t *testing.T) {
			changeJSON(t, result, func(v map[string]any) { v["status"] = "failed"; v["exit_code"] = 1 })
		}},
		{"changed snapshot", func(t *testing.T) {
			changeJSON(t, result, func(v map[string]any) { v["end_snapshot_sha256"] = "different" })
		}},
		{"worker transcript prose", func(t *testing.T) {
			changeEvents(t, raw, func(events []map[string]any) []map[string]any { return events[2:] })
		}},
		{"worker echo", func(t *testing.T) {
			changeEvents(t, raw, func(events []map[string]any) []map[string]any {
				events[1]["item"].(map[string]any)["command"] = "/usr/bin/zsh -lc 'echo python3 -B probe.py'"
				return events
			})
		}},
		{"wrong absolute probe path", func(t *testing.T) {
			changeEvents(t, raw, func(events []map[string]any) []map[string]any {
				events[1]["item"].(map[string]any)["command"] = "/usr/bin/zsh -lc " + shellQuote("python3 -B "+filepath.Join(m.Seats[1].Cwd, "probe.py"))
				return events
			})
		}},
		{"irrelevant worker output", func(t *testing.T) {
			changeEvents(t, raw, func(events []map[string]any) []map[string]any {
				events[1]["item"].(map[string]any)["aggregated_output"] = "irrelevant\n"
				return events
			})
		}},
		{"missing parent rerun", func(t *testing.T) {
			changeEvents(t, parent, func(events []map[string]any) []map[string]any {
				var kept []map[string]any
				for _, ev := range events {
					item, _ := ev["item"].(map[string]any)
					if c, _ := item["command"].(string); strings.Contains(c, "python3 -B") {
						continue
					}
					kept = append(kept, ev)
				}
				return kept
			})
		}},
		{"parent rerun before results", func(t *testing.T) {
			changeEvents(t, parent, func(events []map[string]any) []map[string]any {
				var early, rest []map[string]any
				for _, ev := range events {
					item, _ := ev["item"].(map[string]any)
					if c, _ := item["command"].(string); strings.Contains(c, "python3 -B") {
						early = append(early, ev)
					} else {
						rest = append(rest, ev)
					}
				}
				return append(early, rest...)
			})
		}},
		{"missing capabilities", func(t *testing.T) {
			changeEvents(t, parent, func(events []map[string]any) []map[string]any {
				var kept []map[string]any
				for _, ev := range events {
					item, _ := ev["item"].(map[string]any)
					if item["tool"] != "capabilities" {
						kept = append(kept, ev)
					}
				}
				return kept
			})
		}},
		{"missing installed read", func(t *testing.T) {
			changeEvents(t, parent, func(events []map[string]any) []map[string]any { return events[2:] })
		}},
		{"parent failure", func(t *testing.T) {
			changeJSON(t, filepath.Join(root, "parent-exit.json"), func(v map[string]any) { v["ExitCode"] = 1 })
		}},
		{"parent terminal failure", func(t *testing.T) {
			changeEvents(t, parent, func(events []map[string]any) []map[string]any {
				events[len(events)-1]["type"] = "turn.failed"
				return events
			})
		}},
		{"unsafe parent argv", func(t *testing.T) {
			mutateFile(t, filepath.Join(root, "parent-argv.json"), func(data []byte) []byte {
				var args []string
				must(t, json.Unmarshal(data, &args))
				args = append(args, "--dangerously-bypass-approvals-and-sandbox")
				out, _ := json.Marshal(args)
				return out
			})
		}},
		{"changed fixture", func(t *testing.T) {
			mutateFile(t, filepath.Join(m.Seats[0].Cwd, "calculator.py"), func([]byte) []byte { return []byte(baseSource) })
		}},
		{"changed probe", func(t *testing.T) {
			mutateFile(t, filepath.Join(root, "repo", "probe.py"), func(data []byte) []byte { return append(data, []byte("print('forged')\n")...) })
		}},
		{"changed installed skill", func(t *testing.T) {
			mutateFile(t, filepath.Join(root, "repo", ".agents", "skills", "interrogate", "SKILL.md"), func([]byte) []byte { return []byte("not interrogate") })
		}},
		{"truncated trace", func(t *testing.T) { mutateFile(t, parent, func(data []byte) []byte { return data[:len(data)-20] }) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.mutate(t)
			if _, err := verify(root); err == nil {
				t.Fatal("mutated evidence accepted")
			}
		})
	}
	replay, err := verify(root)
	must(t, err)
	if !reflect.DeepEqual(positive, replay) {
		t.Fatal("replay changed verified result")
	}
}
func TestStructuredFindingRejectsUnsupportedClaims(t *testing.T) {
	f := finalEvidence{Findings: []finding{{RootCause: "last-element-omitted", Command: probeCommand, Output: reviewOutput, RequiredOutput: baseOutput, Contributors: []int{0, 1}}}}
	encode := func(f finalEvidence) string {
		data, _ := json.Marshal(f)
		return "```pstack-rex-evidence\n" + string(data) + "\n```"
	}
	if _, err := parseFinal(encode(f), 2); err != nil {
		t.Fatal(err)
	}
	f.Findings = append(f.Findings, f.Findings[0])
	if _, err := parseFinal(encode(f), 2); err == nil {
		t.Fatal("extra surviving finding accepted")
	}
	f.Findings = f.Findings[:1]
	f.Findings[0].Contributors = []int{0}
	if _, err := parseFinal(encode(f), 2); err == nil {
		t.Fatal("omitted contributor accepted")
	}
}
func TestExactCommandRejectsShellComposition(t *testing.T) {
	want := "python3 -B /private/seat-0/probe.py"
	for _, command := range []string{want, "/usr/bin/zsh -lc " + shellQuote(want), "/bin/bash -lc " + `"` + want + `"`} {
		if !isCommand(command, want) {
			t.Fatalf("supported invocation rejected: %s", command)
		}
	}
	for _, command := range []string{"echo " + want, want + "; true", want + " && echo pass", want + " # echoed", "/usr/bin/zsh -lc 'echo " + want + "'", "/usr/bin/zsh -lc '" + want + "' extra"} {
		if isCommand(command, want) {
			t.Fatalf("unsupported invocation accepted: %s", command)
		}
	}
}
