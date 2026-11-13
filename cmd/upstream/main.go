// Command upstream synchronizes Cursor's pristine pstack directory.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var revisionPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func main() {
	if err := run(os.Args[1:], "upstream", os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, dest string, out io.Writer) error {
	if len(args) == 0 || args[0] != "sync" {
		return errors.New("usage: upstream sync [--repo URL] [--ref main] [--check]")
	}
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	repo := flags.String("repo", "https://github.com/cursor/plugins", "source repository")
	ref := flags.String("ref", "main", "source branch or tag")
	check := flags.Bool("check", false, "validate the recorded revision without network access")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *check {
		rev, err := readRevision(dest)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "upstream revision valid:", rev)
		return err
	}
	return syncUpstream(*repo, *ref, dest, out)
}

func readRevision(dest string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dest, "REVISION"))
	if err != nil {
		return "", err
	}
	rev := strings.TrimSpace(string(data))
	if !revisionPattern.MatchString(rev) {
		return "", errors.New("upstream/REVISION must contain a 40-hex SHA")
	}
	return rev, nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", args[0], err, data)
	}
	return strings.TrimSpace(string(data)), nil
}

func syncUpstream(repo, ref, dest string, out io.Writer) error {
	temp, err := os.MkdirTemp("", "pstack-upstream-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	clone := filepath.Join(temp, "source")
	if _, err := git(temp, "clone", "--depth=1", "--filter=blob:none", "--sparse", "--branch", ref, "--", repo, clone); err != nil {
		return err
	}
	if _, err := git(clone, "sparse-checkout", "set", "pstack"); err != nil {
		return err
	}
	revision, err := git(clone, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if !revisionPattern.MatchString(revision) {
		return fmt.Errorf("invalid source revision %q", revision)
	}
	old, _ := readRevision(dest)
	if old == revision {
		_, err := fmt.Fprintln(out, "unchanged", revision)
		return err
	}
	if err := replaceTree(filepath.Join(clone, "pstack"), dest, revision); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "updated upstream %s -> %s\n", old, revision)
	return err
}

// replaceTree stages the whole tree before replacing the old snapshot, so removed
// source files disappear and a failed copy leaves the previous snapshot intact.
func replaceTree(source, dest, revision string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("source pstack is not a directory")
	}
	parent := filepath.Dir(dest)
	stage, err := os.MkdirTemp(parent, ".upstream-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	err = filepath.WalkDir(source, func(p string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, p)
		if err != nil {
			return err
		}
		target := filepath.Join(stage, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case entry.IsDir():
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chmod(target, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.Mode().IsRegular():
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if err := os.WriteFile(target, data, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chmod(target, info.Mode().Perm())
		default:
			return fmt.Errorf("unsupported source mode: %s", p)
		}
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, "REVISION"), []byte(revision+"\n"), 0644); err != nil {
		return err
	}
	backup, err := os.MkdirTemp(parent, ".upstream-backup-")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.RemoveAll(backup)
		}
	}()
	previous := filepath.Join(backup, "previous")
	moved := false
	if err := os.Rename(dest, previous); err == nil {
		moved = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stage, dest); err != nil {
		if moved {
			if restoreErr := os.Rename(previous, dest); restoreErr != nil {
				keepBackup = true
				return fmt.Errorf("replace failed: %v; restore failed: %v; previous snapshot at %s", err, restoreErr, previous)
			}
		}
		return err
	}
	return nil
}
