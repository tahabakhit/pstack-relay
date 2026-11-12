// Package bundle preserves the pinned upstream source and generates an installation.
package bundle

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var UpstreamVersion = Version()

//go:embed all:_upstream manifest.json compatibility.json runtime-contract.md adaptations.json
var files embed.FS

// File locks every source file's bytes and executable bits.
type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}
type Manifest struct {
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
	Version    string `json:"version"`
	SourceRoot string `json:"source_root"`
	Files      []File `json:"files"`
}
type Compatibility struct {
	ManifestSHA256 string `json:"manifest_sha256"`
	Note           string `json:"note"`
}

func Version() string        { var m Manifest; _ = json.Unmarshal(manifestBytes(), &m); return m.Version }
func Revision() string       { var m Manifest; _ = json.Unmarshal(manifestBytes(), &m); return m.Revision }
func manifestBytes() []byte  { b, _ := files.ReadFile("manifest.json"); return b }
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Check verifies the embedded file set, bytes, and compatibility approval.
// Embedded filesystems omit original mode bits; CheckDirectory checks those on disk.
func Check() error {
	var m Manifest
	if err := json.Unmarshal(manifestBytes(), &m); err != nil {
		return err
	}
	var c Compatibility
	b, err := files.ReadFile("compatibility.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	if c.ManifestSHA256 != digest(manifestBytes()) {
		return errors.New("upstream compatibility review pending: run upstream accept only after reviewing the change report")
	}
	source, err := fs.Sub(files, "_upstream")
	if err != nil {
		return err
	}
	if err = checkFS(source, m, false); err != nil {
		return err
	}
	data, err := files.ReadFile("adaptations.json")
	if err != nil {
		return err
	}
	return ValidateAdaptations(source, data)
}

func ValidateAdaptations(source fs.FS, data []byte) error {
	var rules []adaptation
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}
	for _, rule := range rules {
		data, err := fs.ReadFile(source, rule.Path)
		if err != nil {
			return err
		}
		if digest(data) != rule.SourceSHA256 || rule.Old == "" || bytes.Count(data, []byte(rule.Old)) != 1 {
			return fmt.Errorf("adaptation needs review for %s", rule.Path)
		}
	}
	return nil
}

type adaptation struct {
	Path         string `json:"path"`
	SourceSHA256 string `json:"source_sha256"`
	Old          string `json:"old"`
	Replacement  string `json:"replacement"`
}

func checkFS(source fs.FS, m Manifest, modes bool) error {
	expected := make(map[string]File, len(m.Files))
	for _, f := range m.Files {
		if !fs.ValidPath(f.Path) || f.Path == "." || (f.Mode != 0644 && f.Mode != 0755) {
			return fmt.Errorf("invalid source manifest entry %q", f.Path)
		}
		if _, exists := expected[f.Path]; exists {
			return fmt.Errorf("duplicate manifest path %s", f.Path)
		}
		expected[f.Path] = f
	}
	err := fs.WalkDir(source, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, ok := expected[p]
		if !ok {
			return fmt.Errorf("unlocked source file %s", p)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular source file %s", p)
		}
		if modes && uint32(info.Mode().Perm()) != f.Mode {
			return fmt.Errorf("source mode drift %s: got %04o want %04o", p, info.Mode().Perm(), f.Mode)
		}
		b, err := fs.ReadFile(source, p)
		if err != nil {
			return err
		}
		if digest(b) != f.SHA256 {
			return fmt.Errorf("source byte drift %s", p)
		}
		delete(expected, p)
		return nil
	})
	if err != nil {
		return err
	}
	if len(expected) > 0 {
		return errors.New("locked source files are missing")
	}
	return nil
}

// CheckDirectory validates a physical source tree, including executable bits.
func CheckDirectory(root string, manifest Manifest) error {
	return checkFS(os.DirFS(root), manifest, true)
}

func entrypoint(p string) bool {
	return path.Base(p) == "SKILL.md" || path.Base(p) == "FOR_AGENTS.md" || (strings.HasPrefix(p, "agents/") && strings.HasSuffix(p, ".md"))
}
func transform(p string, b []byte) []byte {
	var rules []adaptation
	data, _ := files.ReadFile("adaptations.json")
	_ = json.Unmarshal(data, &rules)
	for _, rule := range rules {
		if rule.Path == p {
			b = bytes.Replace(b, []byte(rule.Old), []byte(rule.Replacement), 1)
		}
	}
	if !entrypoint(p) {
		return b
	}
	rel, _ := filepath.Rel(filepath.FromSlash(path.Dir(p)), "runtime-contract.md")
	pointer := []byte("\n> Mandatory pstack-rex runtime: read [the shared runtime contract](" + filepath.ToSlash(rel) + ") before following this entrypoint. It overrides all upstream delegation, lifecycle, bootstrap and model-configuration instructions.\n\n")
	offset := 0
	if bytes.HasPrefix(b, []byte("---\n")) {
		if end := bytes.Index(b[4:], []byte("\n---\n")); end >= 0 {
			offset = 4 + end + 5
		}
	}
	out := make([]byte, 0, len(b)+len(pointer))
	out = append(out, b[:offset]...)
	out = append(out, pointer...)
	out = append(out, b[offset:]...)
	return out
}

// Install writes the complete generated bundle into an absent or empty directory.
// It stages generated files privately and never replaces a nonempty destination.
func Install(dest string) (err error) {
	if err = Check(); err != nil {
		return err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	if err = os.Mkdir(dest, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(dest)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("install destination must be a real directory")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("install destination must be empty")
	}
	// Generate in a private sibling first so malformed input never leaves a partial bundle.
	stage, err := os.MkdirTemp(filepath.Dir(dest), ".pstack-rex-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	var m Manifest
	if err := json.Unmarshal(manifestBytes(), &m); err != nil {
		return err
	}
	for _, f := range m.Files {
		b, err := files.ReadFile("_upstream/" + f.Path)
		if err != nil {
			return err
		}
		target := filepath.Join(stage, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, transform(f.Path, b), fs.FileMode(f.Mode)); err != nil {
			return err
		}
		if err := os.Chmod(target, fs.FileMode(f.Mode)); err != nil {
			return err
		}
	}
	contract, err := files.ReadFile("runtime-contract.md")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, "runtime-contract.md"), contract, 0644); err != nil {
		return err
	}
	// Remove only an empty directory. A concurrent file creation makes this fail.
	entries, err = os.ReadDir(dest)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("install destination changed during generation")
	}
	if err := os.Remove(dest); err != nil {
		return err
	}
	if err := os.Rename(stage, dest); err != nil {
		return err
	}
	return nil
}
