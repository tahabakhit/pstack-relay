package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func snapshot(ctx context.Context, cwd string) (string, error) {
	root, err := git(ctx, cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root = strings.TrimSpace(root)
	diff, err := git(ctx, root, "diff", "--binary", "HEAD")
	if err != nil {
		return "", err
	}
	untracked, err := git(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	revision, err := git(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	fmt.Fprintf(hash, "head:%s\x00", strings.TrimSpace(revision))
	fmt.Fprintf(hash, "diff:%d\x00%s", len(diff), diff)
	for _, name := range strings.Split(untracked, "\x00") {
		if name == "" {
			continue
		}
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "file:%d:%s:%o:%d\x00", len(name), name, info.Mode(), info.Size())
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return "", err
			}
			io.WriteString(hash, target)
		} else {
			file, err := os.Open(path)
			if err != nil {
				return "", err
			}
			_, err = io.Copy(hash, file)
			closeErr := file.Close()
			if err != nil {
				return "", err
			}
			if closeErr != nil {
				return "", closeErr
			}
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
