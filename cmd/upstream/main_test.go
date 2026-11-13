package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	result, err := git(dir, args...)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func write(t *testing.T, path, text string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), mode); err != nil {
		t.Fatal(err)
	}
}

func TestSyncLocalRepository(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "source repo")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, repo, "init", "--initial-branch=main")
	fixtureGit(t, repo, "config", "user.name", "Fixture")
	fixtureGit(t, repo, "config", "user.email", "fixture@example.invalid")
	fixtureGit(t, repo, "config", "core.filemode", "true")
	write(t, filepath.Join(repo, "pstack/keep.md"), "original\n", 0644)
	write(t, filepath.Join(repo, "pstack/removed.md"), "remove me\n", 0644)
	write(t, filepath.Join(repo, "pstack/run.sh"), "#!/bin/sh\nexit 0\n", 0755)
	write(t, filepath.Join(repo, "outside.md"), "exclude me\n", 0644)
	if err := os.Symlink("keep.md", filepath.Join(repo, "pstack/link")); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, repo, "add", ".")
	fixtureGit(t, repo, "commit", "-m", "initial")
	first := fixtureGit(t, repo, "rev-parse", "HEAD")
	dest := filepath.Join(root, "upstream")
	var output bytes.Buffer
	args := []string{"sync", "--repo", "file://" + filepath.ToSlash(repo)}
	if err := run(args, dest, &output); err != nil {
		t.Fatal(err)
	}
	revision, err := readRevision(dest)
	if err != nil || revision != first {
		t.Fatalf("revision=%s err=%v", revision, err)
	}
	directory, err := os.Stat(dest)
	if err != nil || directory.Mode().Perm() != 0755 {
		t.Fatalf("snapshot directory mode: %v %v", directory, err)
	}
	info, err := os.Stat(filepath.Join(dest, "run.sh"))
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("executable mode: %v %v", info, err)
	}
	link, err := os.Readlink(filepath.Join(dest, "link"))
	if err != nil || link != "keep.md" {
		t.Fatalf("symlink: %s %v", link, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "outside.md")); !os.IsNotExist(err) {
		t.Fatal("copied outside pstack")
	}
	output.Reset()
	if err := run(args, dest, &output); err != nil || !strings.HasPrefix(output.String(), "unchanged ") {
		t.Fatalf("repeat: %s %v", &output, err)
	}
	fixtureGit(t, repo, "rm", "pstack/removed.md")
	write(t, filepath.Join(repo, "pstack/keep.md"), "updated\n", 0644)
	write(t, filepath.Join(repo, "pstack/new.md"), "new\n", 0644)
	fixtureGit(t, repo, "add", ".")
	fixtureGit(t, repo, "commit", "-m", "replace")
	second := fixtureGit(t, repo, "rev-parse", "HEAD")
	if err := run(args, dest, &output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "removed.md")); !os.IsNotExist(err) {
		t.Fatal("stale file survived")
	}
	if data, err := os.ReadFile(filepath.Join(dest, "keep.md")); err != nil || string(data) != "updated\n" {
		t.Fatalf("updated content: %q %v", data, err)
	}
	if revision, err := readRevision(dest); err != nil || revision != second {
		t.Fatalf("second revision: %s %v", revision, err)
	}
	if err := run(append(args, "--ref", "missing-branch"), dest, &output); err == nil {
		t.Fatal("missing ref succeeded")
	}
	if revision, err := readRevision(dest); err != nil || revision != second {
		t.Fatal("failed clone changed previous snapshot")
	}
	fixtureGit(t, repo, "rm", "-r", "pstack")
	fixtureGit(t, repo, "commit", "-m", "no pstack")
	if err := run(args, dest, &output); err == nil {
		t.Fatal("missing pstack succeeded")
	}
	if revision, err := readRevision(dest); err != nil || revision != second {
		t.Fatal("failed copy changed previous snapshot")
	}
}

func TestCheckRevision(t *testing.T) {
	for _, value := range []string{"", "bad", strings.Repeat("a", 39), strings.Repeat("g", 40), strings.Repeat("a", 40) + "\nextra"} {
		t.Run(value, func(t *testing.T) {
			dest := t.TempDir()
			write(t, filepath.Join(dest, "REVISION"), value, 0644)
			if err := run([]string{"sync", "--check"}, dest, &bytes.Buffer{}); err == nil {
				t.Fatal("invalid revision accepted")
			}
		})
	}
	dest := t.TempDir()
	if err := run([]string{"sync", "--check"}, dest, &bytes.Buffer{}); err == nil {
		t.Fatal("missing revision accepted")
	}
	write(t, filepath.Join(dest, "REVISION"), strings.Repeat("a", 40)+"\n", 0644)
	if err := run([]string{"sync", "--check", "--repo", "does-not-exist"}, dest, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}
