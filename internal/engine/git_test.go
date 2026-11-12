package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOwnedGitDoesNotRunRepositoryPrograms(t *testing.T) {
	_, req := fixture(t, successStream)
	marker := filepath.Join(t.TempDir(), "executed")
	t.Setenv("PSTACK_GIT_MARKER", marker)
	script := filepath.Join(t.TempDir(), "repo-program")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf ran >> \"$PSTACK_GIT_MARKER\"\n/bin/cat\n"), 0700); err != nil {
		t.Fatal(err)
	}
	raw := func(args ...string) {
		t.Helper()
		base := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "commit.gpgsign=false", "-C", req.Cwd}
		if data, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("fixture git: %v: %s", err, data)
		}
	}
	if err := os.WriteFile(filepath.Join(req.Cwd, ".gitattributes"), []byte("*.data filter=sentinel diff=sentinel\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(req.Cwd, "payload.data"), []byte("original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	raw("add", ".")
	raw("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "attributed data")
	for _, key := range []string{"filter.sentinel.clean", "filter.sentinel.smudge", "diff.sentinel.textconv", "core.fsmonitor"} {
		raw("config", key, script)
	}
	raw("config", "filter.sentinel.required", "true")
	hooks := filepath.Join(t.TempDir(), "hooks")
	if err := os.Mkdir(hooks, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "post-checkout"), []byte("#!/bin/sh\nprintf hook >> \"$PSTACK_GIT_MARKER\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	raw("config", "core.hooksPath", hooks)
	if err := os.WriteFile(filepath.Join(req.Cwd, "payload.data"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := git(ctx, req.Cwd, "status", "--porcelain"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(ctx, req.Cwd, "diff", "--binary", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot(ctx, req.Cwd); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(t.TempDir(), "checkout")
	if _, err := git(ctx, req.Cwd, "worktree", "add", "--detach", worktree, "HEAD"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(worktree, "payload.data"))
	if err != nil || string(data) != "original\n" {
		t.Fatalf("raw checkout=%q err=%v", data, err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("repository program executed outside harness sandbox: %v", err)
	}
}
