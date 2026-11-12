package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tahabakhit/pstack-rex/internal/config"
)

const successStream = `printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"verified fixture"}}' '{"type":"turn.completed"}'`

func TestWorkerProcess(t *testing.T) {
	if os.Getenv("PSTACK_TEST_WORKER") != "1" {
		return
	}
	if err := RunWorker(os.Getenv("PSTACK_TEST_ROOT"), "attempt"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func fixture(t *testing.T, script string) (*Service, Request) {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git: %s: %v", out, err)
		}
	}
	bin := filepath.Join(base, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(bin, "codex")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nset -eu\n"+script+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	root := filepath.Join(base, "state")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	c := config.Default()
	c.TimeoutSeconds = 5
	c.MaxConcurrent = 1
	target := config.Target{Harness: "codex", Model: "fixture-model", Backend: "rex", Auth: "existing"}
	c.Roles["how explorer"] = []config.Target{target}
	rev, err := git(context.Background(), repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	req := Request{ID: "attempt", Role: "how explorer", Cwd: repo, Prompt: "fixture", Access: "read-only", Target: target, TimeoutSeconds: 5, ExpectedRevision: strings.TrimSpace(rev), Executable: executable}
	return &Service{Root: root, Config: c}, req
}

func reserve(t *testing.T, s *Service, req Request) string {
	t.Helper()
	dir, err := s.dir(req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err = writeJSON(filepath.Join(dir, "request.json"), req); err != nil {
		t.Fatal(err)
	}
	return dir
}

func worker(t *testing.T, s *Service) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestWorkerProcess$")
	cmd.Env = append(os.Environ(), "PSTACK_TEST_WORKER=1", "PSTACK_TEST_ROOT="+s.Root, "REX_SESSION=fixture-session", "REX_BLOCK=fixture-block")
	return cmd
}

func result(t *testing.T, dir string) Result {
	t.Helper()
	var r Result
	if err := readJSON(filepath.Join(dir, "result.json"), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func awaitFile(t *testing.T, path string) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("file not created: %s", path)
}

func TestWorkerCompletionRequiresEventAndExit(t *testing.T) {
	cases := []struct {
		name, script, status string
		exit                 int
	}{
		{"success", successStream, "completed", 0},
		{"missing terminal", `printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"unfinished"}}'`, "failed", 0},
		{"nonzero despite terminal", successStream + "\nexit 7", "failed", 7},
		{"malformed", `printf '%s\n' 'not-json'`, "failed", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, req := fixture(t, tc.script)
			dir := reserve(t, s, req)
			out, err := worker(t, s).CombinedOutput()
			r := result(t, dir)
			if r.Status != tc.status || r.ExitCode != tc.exit {
				t.Fatalf("result=%+v, output=%s", r, out)
			}
			if (err == nil) != (tc.status == "completed") {
				t.Fatalf("supervisor error=%v output=%s", err, out)
			}
			if tc.status == "completed" {
				data, e := os.ReadFile(filepath.Join(dir, "output.md"))
				if e != nil || string(data) != "verified fixture" {
					t.Fatalf("output=%q err=%v", data, e)
				}
			}
			before, _ := os.ReadFile(filepath.Join(dir, "result.json"))
			if out, err := worker(t, s).CombinedOutput(); err == nil {
				t.Fatalf("replay allowed: %s", out)
			}
			after, _ := os.ReadFile(filepath.Join(dir, "result.json"))
			if string(before) != string(after) {
				t.Fatal("replay changed terminal receipt")
			}
		})
	}
}

func TestWorkerCancellationAndDeadline(t *testing.T) {
	for _, mode := range []string{"before-start", "cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			s, req := fixture(t, `sleep 60 &
child=$!
printf '%s' "$child" > child.pid
wait "$child"`)
			req.TimeoutSeconds = 1
			dir := reserve(t, s, req)
			if mode == "before-start" {
				if err := writeJSON(filepath.Join(dir, "cancel.json"), map[string]bool{"cancel": true}); err != nil {
					t.Fatal(err)
				}
			}
			cmd := worker(t, s)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if cmd.ProcessState == nil {
					cmd.Process.Kill()
					cmd.Wait()
				}
			})
			if mode != "before-start" {
				awaitFile(t, filepath.Join(req.Cwd, "child.pid"))
			}
			if mode == "cancel" {
				if out, err := worker(t, s).CombinedOutput(); err == nil {
					t.Fatalf("duplicate supervisor started: %s", out)
				}
				if err := writeJSON(filepath.Join(dir, "cancel.json"), map[string]bool{"cancel": true}); err != nil {
					t.Fatal(err)
				}
			}
			if err := cmd.Wait(); err == nil {
				t.Fatal("cancelled/timed out supervisor succeeded")
			}
			r := result(t, dir)
			want := "cancelled"
			if mode == "deadline" {
				want = "timed_out"
			}
			if r.Status != want {
				t.Fatalf("%+v", r)
			}
			if mode == "before-start" {
				if _, err := os.Stat(filepath.Join(req.Cwd, "child.pid")); !os.IsNotExist(err) {
					t.Fatal("cancelled attempt executed")
				}
			} else {
				data, err := os.ReadFile(filepath.Join(req.Cwd, "child.pid"))
				if err != nil {
					t.Fatal(err)
				}
				var pid int
				fmt.Sscan(string(data), &pid)
				// A killed orphan may remain a zombie until the container's init reaps it.
				if err := syscall.Kill(pid, 0); err == nil {
					stat, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
					if e == nil && !strings.Contains(string(stat), ") Z ") {
						t.Fatalf("descendant %d survived: %s", pid, stat)
					}
				}
			}
		})
	}
}

func TestWorkerIsolationAndSnapshot(t *testing.T) {
	s, req := fixture(t, "printf changed > artifact.txt\n"+successStream)
	req.Access = "isolated-write"
	dir := reserve(t, s, req)
	if out, err := worker(t, s).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(req.Cwd, "artifact.txt")); !os.IsNotExist(err) {
		t.Fatal("source checkout changed")
	}
	if data, err := os.ReadFile(filepath.Join(dir, "worktree", "artifact.txt")); err != nil || string(data) != "changed" {
		t.Fatalf("worktree artifact=%q err=%v", data, err)
	}
	r := result(t, dir)
	if r.StartSnapshotSHA256 == "" || r.StartSnapshotSHA256 == r.EndSnapshotSHA256 {
		t.Fatal("untracked worktree change was not detected")
	}
}

func TestDispatchIdempotencyUncertaintyAndAdmission(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(fmt.Sprint(uncertain), func(t *testing.T) {
			s, req := fixture(t, successStream)
			log := filepath.Join(t.TempDir(), "launches")
			launch := `printf '%s\n' '{"session_id":"session","block_ids":["block"]}'`
			if uncertain {
				launch = "exit 1"
			}
			script := "#!/bin/sh\nset -eu\nshift 3\ncase \"$1\" in\nwhoami) printf '%s\\n' '{\"endpoint\":\"unix:///fixture\",\"api_revision\":12}' ;;\nnew|run) printf launched\\n >> '" + log + "'\n" + launch + " ;;\nblock) printf '%s\\n' '{}' ;;\n*) exit 9 ;;\nesac\n"
			rex := filepath.Join(t.TempDir(), "rex")
			if err := os.WriteFile(rex, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			s.RexExecutable = rex
			first, err := s.Dispatch(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			second, err := s.Dispatch(context.Background(), req)
			if err != nil || second.ID != first.ID {
				t.Fatalf("duplicate=%+v err=%v", second, err)
			}
			data, err := os.ReadFile(log)
			if err != nil || strings.Count(string(data), "launched") != 1 {
				t.Fatalf("launches=%q err=%v", data, err)
			}
			conflict := req
			conflict.Prompt = "different"
			if _, err := s.Dispatch(context.Background(), conflict); err == nil {
				t.Fatal("conflicting key accepted")
			}
			next := req
			next.ID = "another"
			if _, err := s.Dispatch(context.Background(), next); err == nil || !strings.Contains(err.Error(), "concurrency") {
				t.Fatalf("unresolved worker did not occupy slot: %v", err)
			}
			if uncertain && first.Status != "unknown" {
				t.Fatalf("uncertain launch=%+v", first)
			}
		})
	}
}

func TestWorkerIDAndWaitBounds(t *testing.T) {
	s := &Service{Root: t.TempDir()}
	for _, id := range []string{"", "../escape", "a/b", "/absolute", "a.b", strings.Repeat("a", 81)} {
		if _, err := s.Inspect(context.Background(), id); err == nil || !strings.Contains(err.Error(), "invalid worker id") {
			t.Fatalf("id %q: %v", id, err)
		}
	}
	for _, d := range []time.Duration{0, -time.Second, 61 * time.Second} {
		if _, err := s.Wait(context.Background(), "attempt", d); err == nil {
			t.Fatalf("wait %v accepted", d)
		}
	}
}

func TestDispatchProcess(t *testing.T) {
	if os.Getenv("PSTACK_TEST_DISPATCH") != "1" {
		return
	}
	var req Request
	if err := readJSON(os.Getenv("PSTACK_TEST_REQUEST"), &req); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c := config.Default()
	c.TimeoutSeconds = 5
	c.MaxConcurrent = 1
	c.Roles[req.Role] = []config.Target{req.Target}
	s := Service{Root: os.Getenv("PSTACK_TEST_ROOT"), Config: c, RexExecutable: os.Getenv("PSTACK_TEST_REX")}
	if _, err := s.Dispatch(context.Background(), req); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestConcurrentProcessesReserveOneLaunch(t *testing.T) {
	s, req := fixture(t, successStream)
	base := t.TempDir()
	log := filepath.Join(base, "launches")
	rex := filepath.Join(base, "rex")
	request := filepath.Join(base, "request.json")
	if err := writeJSON(request, req); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nset -eu\nshift 3\ncase \"$1\" in\nwhoami) printf '%s\\n' '{\"endpoint\":\"unix:///fixture\",\"api_revision\":12}' ;;\nnew|run) printf '%s\\n' launched >> '" + log + "'\nprintf '%s\\n' '{\"session_id\":\"session\",\"block_ids\":[\"block\"]}' ;;\nblock) printf '%s\\n' '{}' ;;\n*) exit 9 ;;\nesac\n"
	if err := os.WriteFile(rex, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	var commands []*exec.Cmd
	for i := 0; i < 2; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestDispatchProcess$")
		cmd.Env = append(os.Environ(), "PSTACK_TEST_DISPATCH=1", "PSTACK_TEST_REQUEST="+request, "PSTACK_TEST_ROOT="+s.Root, "PSTACK_TEST_REX="+rex)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("concurrent dispatch: %v", err)
		}
	}
	data, err := os.ReadFile(log)
	if err != nil || string(data) != "launched\n" {
		t.Fatalf("launch log=%q err=%v", data, err)
	}
}
