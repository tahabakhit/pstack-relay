package rexworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type call struct {
	name string
	args []string
}

func fixture(t *testing.T) (string, Spec) {
	t.Helper()
	root := t.TempDir()
	brief := filepath.Join(root, "task.md")
	if err := os.WriteFile(brief, []byte("Do the task."), 0600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "state"), Spec{Harness: "claude", Model: "opus", BriefPath: brief, Cwd: root}
}

func rexEnv(key string) string {
	if key == "REX_SESSION" {
		return "session-1"
	}
	return ""
}

func TestStart(t *testing.T) {
	for _, harness := range []string{"claude", "codex", "pi"} {
		for _, effort := range []string{"", "high"} {
			for _, own := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/effort=%s/worktree=%v", harness, effort, own), func(t *testing.T) {
					root, spec := fixture(t)
					spec.Harness, spec.Effort, spec.OwnWorktree = harness, effort, own
					var calls []call
					run := func(name string, args ...string) ([]byte, error) {
						calls = append(calls, call{name, args})
						return []byte(`{"block_id":"block-1"}`), nil
					}
					l, err := Start(run, root, rexEnv, spec)
					if err != nil {
						t.Fatal(err)
					}
					cwd := spec.Cwd
					if own {
						cwd = filepath.Join(l.StateDir, "worktree")
						want := call{"git", []string{"-C", spec.Cwd, "worktree", "add", "-b", "pstack-rex/" + l.ID, cwd, "HEAD"}}
						if !reflect.DeepEqual(calls[0], want) {
							t.Fatalf("worktree call = %#v, want %#v", calls[0], want)
						}
					}
					want := []string{"run", "-s", "session-1", "--split", "right", "--focus=false", "--label", "pstack-" + l.ID, "-c", cwd, "--json", "--", harness, "--model", spec.Model}
					if effort != "" {
						switch harness {
						case "claude":
							want = append(want, "--effort", effort)
						case "codex":
							want = append(want, "-c", `model_reasoning_effort="high"`)
						case "pi":
							want = append(want, "--thinking", effort)
						}
					}
					brief := filepath.Join(l.StateDir, "brief.md")
					draft := filepath.Join(l.StateDir, "result-draft.md")
					want = append(want, fmt.Sprintf("Read and do the task in %s. When you are finished, write your final report (results and the evidence asked for) to %s, then run: pstack-rex rex report --id %s --file %s", brief, draft, l.ID, draft))
					last := calls[len(calls)-1]
					if last.name != "rex" || !reflect.DeepEqual(last.args, want) {
						t.Fatalf("launch call = %#v, want %#v", last, want)
					}
					if len(calls) != 1+btoi(own) || l.Block != "block-1" || l.Cwd != cwd || l.Session != "session-1" {
						t.Fatalf("unexpected launch: %+v; calls: %#v", l, calls)
					}
					data, err := os.ReadFile(brief)
					if err != nil || string(data) != "Do the task." {
						t.Fatalf("copied brief: %q, %v", data, err)
					}
					info, err := os.Stat(l.StateDir)
					if err != nil || info.Mode().Perm() != 0700 {
						t.Fatalf("state permissions: %v, %v", info, err)
					}
					data, err = os.ReadFile(filepath.Join(l.StateDir, "launch.json"))
					var stored Launch
					if err != nil || json.Unmarshal(data, &stored) != nil || stored != l {
						t.Fatalf("stored launch: %s, %v", data, err)
					}
				})
			}
		}
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestStartBlockIDs(t *testing.T) {
	root, spec := fixture(t)
	l, err := Start(func(string, ...string) ([]byte, error) {
		return []byte(`{"block_ids":["first","second"]}`), nil
	}, root, rexEnv, spec)
	if err != nil || l.Block != "first" {
		t.Fatalf("launch = %+v, %v", l, err)
	}
}

func TestStartMissingSession(t *testing.T) {
	root, spec := fixture(t)
	_, err := Start(func(string, ...string) ([]byte, error) {
		t.Fatal("runner called outside Rex")
		return nil, nil
	}, root, func(string) string { return "" }, spec)
	if err == nil || err.Error() != "not inside Rex" {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("state created outside Rex: %v", err)
	}
}

func started(t *testing.T) (string, Launch) {
	t.Helper()
	root, spec := fixture(t)
	l, err := Start(func(string, ...string) ([]byte, error) {
		return []byte(`{"block_id":"block-1"}`), nil
	}, root, rexEnv, spec)
	if err != nil {
		t.Fatal(err)
	}
	return root, l
}

func TestReportThenWait(t *testing.T) {
	root, l := started(t)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = original })
	err = Report(root, l.ID, []byte("Results and evidence."))
	os.Stdout = original
	writer.Close()
	escape, readErr := io.ReadAll(reader)
	reader.Close()
	if err != nil || readErr != nil || string(escape) != "\x1b]7501;state=done:id="+l.ID+"\x1b\\" {
		t.Fatalf("report = %q, %v, %v", escape, err, readErr)
	}
	status, err := Wait(func(string, ...string) ([]byte, error) {
		t.Fatal("queried Rex after report")
		return nil, nil
	}, root, l.ID, time.Second)
	if err != nil || status.State != "reported" || status.Result != "Results and evidence." || status.ExitCode != nil {
		t.Fatalf("status = %+v, %v", status, err)
	}
	matches, err := filepath.Glob(filepath.Join(l.StateDir, ".result-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary results remain: %v, %v", matches, err)
	}
}

func TestWaitExited(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			root, l := started(t)
			status, err := Wait(func(name string, args ...string) ([]byte, error) {
				want := []string{"block", "call", "-s", l.Session, "-b", l.Block, "process"}
				if name != "rex" || !reflect.DeepEqual(args, want) {
					t.Fatalf("process call: %s %q", name, args)
				}
				return []byte(fmt.Sprintf(`{"last_exit":{"exit_code":%d}}`, code)), nil
			}, root, l.ID, time.Second)
			if err != nil || status.State != "exited" || status.ExitCode == nil || *status.ExitCode != code {
				t.Fatalf("status = %+v, %v", status, err)
			}
		})
	}
}

func TestWaitTimeout(t *testing.T) {
	root, l := started(t)
	calls := 0
	start := time.Now()
	status, err := Wait(func(string, ...string) ([]byte, error) {
		calls++
		return []byte(`{"last_exit":null}`), nil
	}, root, l.ID, 20*time.Millisecond)
	if err != nil || status.State != "running" || status.ExitCode != nil || calls < 2 || time.Since(start) < 20*time.Millisecond {
		t.Fatalf("status = %+v, calls = %d, %v", status, calls, err)
	}
}

func TestWaitReportDuringExitQuery(t *testing.T) {
	root, l := started(t)
	status, err := Wait(func(string, ...string) ([]byte, error) {
		if err := os.WriteFile(filepath.Join(l.StateDir, "result.md"), []byte("finished"), 0600); err != nil {
			t.Fatal(err)
		}
		return []byte(`{"last_exit":{"exit_code":0}}`), nil
	}, root, l.ID, 0)
	if err != nil || status.State != "reported" || status.Result != "finished" {
		t.Fatalf("status = %+v, %v", status, err)
	}
}

func TestFailures(t *testing.T) {
	for _, output := range []string{`not json`, `{}`, `{"block_ids":[]}`} {
		t.Run(output, func(t *testing.T) {
			root, spec := fixture(t)
			if _, err := Start(func(string, ...string) ([]byte, error) { return []byte(output), nil }, root, rexEnv, spec); err == nil {
				t.Fatal("accepted invalid launch response")
			}
		})
	}
	for _, id := range []string{"", ".", "..", "../escape"} {
		if err := Report(t.TempDir(), id, nil); err == nil {
			t.Fatalf("accepted id %q", id)
		}
	}
	root, l := started(t)
	for _, output := range []string{"broken", `{"last_exit":{"exit_code":"no"}}`} {
		if _, err := Wait(func(string, ...string) ([]byte, error) { return []byte(output), nil }, root, l.ID, 0); err == nil {
			t.Fatalf("accepted process response %q", output)
		}
	}
	_, err := Wait(func(string, ...string) ([]byte, error) { return nil, errors.New("unavailable") }, root, l.ID, 0)
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("query error = %v", err)
	}
}

func TestStateRoot(t *testing.T) {
	for _, xdg := range []string{"", "/tmp/state"} {
		got := StateRoot(func(key string) string {
			if key == "XDG_STATE_HOME" {
				return xdg
			}
			if key == "HOME" {
				return "/home/example"
			}
			return ""
		})
		want := "/home/example/.local/state/pstack-rex/rex"
		if xdg != "" {
			want = "/tmp/state/pstack-rex/rex"
		}
		if got != want {
			t.Fatalf("StateRoot = %q, want %q", got, want)
		}
	}
}
