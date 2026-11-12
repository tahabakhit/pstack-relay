package bundle

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedInstall(t *testing.T) {
	if err := Check(); err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := json.Unmarshal(manifestBytes(), &m); err != nil {
		t.Fatal(err)
	}
	if Version() != m.Version || UpstreamVersion != m.Version {
		t.Fatal("version mismatch")
	}
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	if err := Install(a); err != nil {
		t.Fatal(err)
	}
	if err := Install(b); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, f := range m.Files {
		original, err := files.ReadFile("upstream/" + f.Path)
		if err != nil {
			t.Fatal(err)
		}
		first, err := os.ReadFile(filepath.Join(a, filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		second, err := os.ReadFile(filepath.Join(b, filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("nondeterministic %s", f.Path)
		}
		info, err := os.Stat(filepath.Join(a, filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if uint32(info.Mode().Perm()) != f.Mode {
			t.Fatalf("mode %s", f.Path)
		}
		if entrypoint(f.Path) {
			count++
			marker := []byte("\n> Mandatory pstack-rex runtime:")
			if bytes.Count(first, marker) != 1 {
				t.Fatalf("pointer %s", f.Path)
			}
			start := bytes.Index(first, marker)
			end := bytes.Index(first[start+1:], []byte("\n\n"))
			if end < 0 {
				t.Fatal("missing pointer ending")
			}
			restored := append(append([]byte{}, first[:start]...), first[start+1+end+2:]...)
			if !bytes.Equal(restored, original) {
				t.Fatalf("upstream body changed %s", f.Path)
			}
			text := string(first[start : start+1+end])
			left := strings.Index(text, "(")
			right := strings.Index(text, ")")
			resolved := filepath.Join(filepath.Dir(filepath.Join(a, filepath.FromSlash(f.Path))), filepath.FromSlash(text[left+1:right]))
			if filepath.Clean(resolved) != filepath.Join(a, "runtime-contract.md") {
				t.Fatalf("bad link %s", f.Path)
			}
		} else if !bytes.Equal(first, original) {
			t.Fatalf("modified dependency %s", f.Path)
		}
	}
	if count < 60 {
		t.Fatalf("expected all nested skill and agent entries, got %d", count)
	}
	expected := len(m.Files) + 1
	actual := 0
	if err := filepath.WalkDir(a, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			actual++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("file set %d != %d", actual, expected)
	}
}
func TestInstallRefusesConflicts(t *testing.T) {
	dest := t.TempDir()
	p := filepath.Join(dest, "keep")
	if err := os.WriteFile(p, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Install(dest); err == nil {
		t.Fatal("accepted nonempty destination")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "existing" {
		t.Fatal("overwrote existing file")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	if err := Install(link); err == nil {
		t.Fatal("accepted symlink")
	}
}
func TestDirectoryDrift(t *testing.T) {
	for _, kind := range []string{"bytes", "new", "missing", "mode", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			p := filepath.Join(root, "test")
			if err := os.WriteFile(p, []byte("original"), 0755); err != nil {
				t.Fatal(err)
			}
			m := Manifest{Files: []File{{Path: "test", SHA256: digest([]byte("original")), Mode: 0755}}}
			if err := CheckDirectory(root, m); err != nil {
				t.Fatal(err)
			}
			var err error
			switch kind {
			case "bytes":
				err = os.WriteFile(p, []byte("changed"), 0755)
			case "new":
				err = os.WriteFile(filepath.Join(root, "new"), nil, 0644)
			case "missing":
				err = os.Remove(p)
			case "mode":
				err = os.Chmod(p, 0644)
			case "symlink":
				if err = os.Remove(p); err == nil {
					err = os.Symlink("elsewhere", p)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := CheckDirectory(root, m); err == nil {
				t.Fatalf("missed %s drift", kind)
			}
		})
	}
}
