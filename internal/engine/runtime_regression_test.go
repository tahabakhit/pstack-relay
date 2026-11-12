package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func fixturePID(t *testing.T, path string) int {
	t.Helper()
	awaitFile(t, path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		t.Fatalf("invalid fixture PID %q: %v", data, err)
	}
	return pid
}

func processAlive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		return false
	}
	// Containers may leave a dead orphan as a zombie until init reaps it.
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	return err != nil || !strings.Contains(string(stat), ") Z ")
}

func TestSupervisorSignalsAndUnconfirmedCancellation(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		t.Run(sig.String(), func(t *testing.T) {
			s, req := fixture(t, `printf '%s' "$$" > harness.pid.tmp
mv harness.pid.tmp harness.pid
sleep 60 &
child=$!
printf '%s' "$child" > child.pid.tmp
mv child.pid.tmp child.pid
wait "$child"`)
			dir := reserve(t, s, req)
			cmd := worker(t, s)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var harnessPID, childPID int
			t.Cleanup(func() {
				if harnessPID == 0 {
					data, _ := os.ReadFile(filepath.Join(req.Cwd, "harness.pid"))
					harnessPID, _ = strconv.Atoi(string(data))
				}
				if childPID == 0 {
					data, _ := os.ReadFile(filepath.Join(req.Cwd, "child.pid"))
					childPID, _ = strconv.Atoi(string(data))
				}
				if harnessPID > 1 {
					syscall.Kill(-harnessPID, syscall.SIGKILL)
				}
				if childPID > 1 {
					syscall.Kill(childPID, syscall.SIGKILL)
				}
				if cmd.ProcessState == nil {
					cmd.Process.Kill()
					cmd.Wait()
				}
			})
			harnessPID = fixturePID(t, filepath.Join(req.Cwd, "harness.pid"))
			childPID = fixturePID(t, filepath.Join(req.Cwd, "child.pid"))
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err == nil {
				t.Fatal("interrupted supervisor succeeded")
			}
			if sig == syscall.SIGTERM {
				r := result(t, dir)
				if r.Status != "cancelled" {
					t.Fatalf("SIGTERM result=%+v", r)
				}
				if processAlive(harnessPID) || processAlive(childPID) {
					t.Fatal("SIGTERM left a harness process group member alive")
				}
				return
			}
			if !processAlive(harnessPID) || !processAlive(childPID) {
				t.Fatal("SIGKILL fixture did not leave the harness group alive")
			}
			if _, err := os.Stat(filepath.Join(dir, "result.json")); !os.IsNotExist(err) {
				t.Fatalf("killed supervisor has a result: %v", err)
			}
			rex := filepath.Join(t.TempDir(), "rex")
			script := `#!/bin/sh
set -eu
shift 3
case "$1" in
whoami) printf '%s\n' '{"endpoint":"unix:///fixture","api_revision":12}' ;;
block) printf '%s\n' '{"last_exit":{"exit_code":137}}' ;;
*) exit 9 ;;
esac
`
			if err := os.WriteFile(rex, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			s.RexExecutable = rex
			if err := writeJSON(filepath.Join(dir, "launch.json"), Launch{Session: "fixture-session", Block: "fixture-block"}); err != nil {
				t.Fatal(err)
			}
			record, err := s.Inspect(context.Background(), req.ID)
			if err != nil || record.Status != "unknown" || record.Result != nil {
				t.Fatalf("inspect after supervisor death=%+v err=%v", record, err)
			}
			record, err = s.Cancel(context.Background(), req.ID)
			if err != nil || record.Status != "unknown" || !strings.Contains(record.Error, "not confirmed") {
				t.Fatalf("cancel after supervisor death=%+v err=%v", record, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "cancel.json")); err != nil {
				t.Fatalf("unconfirmed cancellation was not requested: %v", err)
			}
			if !processAlive(harnessPID) || !processAlive(childPID) {
				t.Fatal("cancellation killed a process using an unverified stale PID")
			}
			next := req
			next.ID = "next"
			if _, err := s.Dispatch(context.Background(), next); err == nil || !strings.Contains(err.Error(), "concurrency") {
				t.Fatalf("unconfirmed worker released admission capacity: %v", err)
			}
		})
	}
}

func TestIsolatedWriterPreservesNestedCwd(t *testing.T) {
	s, req := fixture(t, "pwd > actual-cwd.txt\nprintf changed > artifact.txt\n"+successStream)
	sourceRoot := req.Cwd
	nested := filepath.Join("packages", "api")
	if err := os.MkdirAll(filepath.Join(sourceRoot, nested), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, nested, "tracked.txt"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "nested fixture"}} {
		if out, err := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-C", sourceRoot}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git: %s: %v", out, err)
		}
	}
	rev, err := git(context.Background(), sourceRoot, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	req.ExpectedRevision = strings.TrimSpace(rev)
	req.Cwd = filepath.Join(sourceRoot, nested)
	req.Access = "isolated-write"
	dir := reserve(t, s, req)
	if out, err := worker(t, s).CombinedOutput(); err != nil {
		t.Fatalf("worker: %v %s", err, out)
	}
	wantCwd := filepath.Join(dir, "worktree", nested)
	if data, err := os.ReadFile(filepath.Join(wantCwd, "actual-cwd.txt")); err != nil || strings.TrimSpace(string(data)) != wantCwd {
		t.Fatalf("worker cwd=%q want=%q err=%v", data, wantCwd, err)
	}
	if data, err := os.ReadFile(filepath.Join(wantCwd, "artifact.txt")); err != nil || string(data) != "changed" {
		t.Fatalf("nested artifact=%q err=%v", data, err)
	}
	for _, path := range []string{filepath.Join(req.Cwd, "artifact.txt"), filepath.Join(sourceRoot, "actual-cwd.txt"), filepath.Join(dir, "worktree", "artifact.txt")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("relative write escaped nested worktree: %s, %v", path, err)
		}
	}
	if r := result(t, dir); r.Status != "completed" || r.StartSnapshotSHA256 == r.EndSnapshotSHA256 {
		t.Fatalf("nested writer receipt=%+v", r)
	}
}

func TestSnapshotReceiptsIncludeCleanRevisionAndUntrackedContent(t *testing.T) {
	s, req := fixture(t, successStream)
	run := func() Result {
		t.Helper()
		s.Root = t.TempDir()
		rev, err := git(context.Background(), req.Cwd, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		req.ExpectedRevision = strings.TrimSpace(rev)
		dir := reserve(t, s, req)
		if out, err := worker(t, s).CombinedOutput(); err != nil {
			t.Fatalf("worker: %v %s", err, out)
		}
		r := result(t, dir)
		if r.Status != "completed" || r.StartSnapshotSHA256 == "" || r.StartSnapshotSHA256 != r.EndSnapshotSHA256 {
			t.Fatalf("unchanged worker receipt=%+v", r)
		}
		return r
	}
	first := run()
	if out, err := exec.Command("git", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-C", req.Cwd, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "another clean revision").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %s: %v", out, err)
	}
	second := run()
	if first.StartRevision == second.StartRevision || first.StartSnapshotSHA256 == second.StartSnapshotSHA256 {
		t.Fatal("different clean commits have the same snapshot identity")
	}
	path := filepath.Join(req.Cwd, "untracked.txt")
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	third := run()
	if second.StartSnapshotSHA256 == third.StartSnapshotSHA256 {
		t.Fatal("snapshot omitted an untracked file")
	}
	if err := os.WriteFile(path, []byte("other"), 0600); err != nil {
		t.Fatal(err)
	}
	fourth := run()
	if third.StartSnapshotSHA256 == fourth.StartSnapshotSHA256 {
		t.Fatal("snapshot omitted same-size untracked content change")
	}
}
