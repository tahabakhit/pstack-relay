// Package plugin transforms generated entrypoints without changing upstream files.
package plugin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// Adaptation replaces an exact anchor in a generated file.
type Adaptation struct {
	Path         string `json:"path"`
	SourceSHA256 string `json:"source_sha256"`
	Old          string `json:"old"`
	Replacement  string `json:"replacement"`
}

func entrypoint(p string) bool {
	return path.Base(p) == "SKILL.md" || path.Base(p) == "FOR_AGENTS.md" || (strings.HasPrefix(p, "agents/") && strings.HasSuffix(p, ".md"))
}
func Transform(p string, b []byte, rules []Adaptation) []byte {
	for _, rule := range rules {
		if rule.Path == p {
			b = bytes.Replace(b, []byte(rule.Old), []byte(rule.Replacement), 1)
		}
	}
	if !entrypoint(p) {
		return b
	}
	rel, _ := filepath.Rel(filepath.FromSlash(path.Dir(p)), "delegation.md")
	pointer := []byte("\n> pstack delegation: read [the delegation reference](" + filepath.ToSlash(rel) + ") before following this entrypoint. It maps upstream delegation to this host.\n\n")
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

// ValidateAdaptations checks that exact anchors still match the source before
// a caller uses the retained adaptations in a generated plugin.
func ValidateAdaptations(source fs.FS, rules []Adaptation) error {
	for _, rule := range rules {
		data, err := fs.ReadFile(source, rule.Path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != rule.SourceSHA256 || rule.Old == "" || bytes.Count(data, []byte(rule.Old)) != 1 {
			return fmt.Errorf("adaptation needs review for %s", rule.Path)
		}
	}
	return nil
}
