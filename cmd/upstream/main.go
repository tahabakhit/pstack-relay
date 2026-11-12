// Command upstream checks or explicitly updates the vendored Cursor pstack snapshot.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pstack-rex/pstack-rex/internal/bundle"
)

const bundleDir = "internal/bundle"
const lockPath = "upstream.lock.json"

var terms = regexp.MustCompile(`(?i)\b(task|agent|spawn\w*|delegat\w*|model\w*|effort|role\w*|steer|resume|wait|cancel|stop|hold|readonly|read.only|isolation|worktree|cloud\w*|mcp|background|inherit.parent|auto)\b`)

type Candidate struct {
	Path  string   `json:"path"`
	Line  int      `json:"line"`
	Text  string   `json:"text"`
	Terms []string `json:"terms"`
}
type Report struct {
	Revision          string      `json:"revision"`
	Changed           []string    `json:"changed_files"`
	ChangedDelegation []string    `json:"changed_delegation_files"`
	NewTerms          []string    `json:"new_terms"`
	Roles             []string    `json:"role_candidates"`
	NewRoles          []string    `json:"new_role_candidates"`
	Candidates        []Candidate `json:"candidates"`
	Limit             string      `json:"limit"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: upstream check|update|accept [--source /external/cursor] [--revision SHA]")
	}
	flags := flag.NewFlagSet("upstream "+args[0], flag.ContinueOnError)
	source := flags.String("source", "", "external pristine Cursor plugins checkout")
	revision := flags.String("revision", "", "expected exact checkout commit")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	switch args[0] {
	case "check":
		if err := verifyExternal(*source, *revision); err != nil {
			return err
		}
		return check()
	case "update":
		if *source == "" {
			return errors.New("update requires --source outside this repository")
		}
		return update(*source, *revision)
	case "accept":
		if err := verifyExternal(*source, *revision); err != nil {
			return err
		}
		m, raw, err := readManifest()
		if err != nil {
			return err
		}
		if err := bundle.CheckDirectory(filepath.Join(bundleDir, "_upstream"), m); err != nil {
			return err
		}
		if err := checkAdaptations(); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(bundleDir, "compatibility.json"), bundle.Compatibility{ManifestSHA256: hash(raw), Note: "Explicitly accepted source inventory. Runtime contract overrides upstream delegation; lexical inventory is a review aid, not a semantic guarantee."}); err != nil {
			return err
		}
		fmt.Println("Accepted current compatibility baseline. Review and commit the source, manifest, report and baseline together.")
		return nil
	default:
		return errors.New("unknown upstream action")
	}
}
func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0644)
}
func readManifest() (bundle.Manifest, []byte, error) {
	var m bundle.Manifest
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return m, nil, err
	}
	embedded, err := os.ReadFile(filepath.Join(bundleDir, "manifest.json"))
	if err != nil {
		return m, nil, err
	}
	if !bytes.Equal(raw, embedded) {
		return m, nil, errors.New("root lock and embedded manifest differ")
	}
	err = json.Unmarshal(raw, &m)
	return m, raw, err
}
func check() error {
	m, raw, err := readManifest()
	if err != nil {
		return err
	}
	if err := bundle.CheckDirectory(filepath.Join(bundleDir, "_upstream"), m); err != nil {
		return err
	}
	var c bundle.Compatibility
	b, err := os.ReadFile(filepath.Join(bundleDir, "compatibility.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	if hash(raw) != c.ManifestSHA256 {
		return errors.New("upstream compatibility review pending; inspect internal/bundle/review-report.json before explicit accept")
	}
	if err := checkAdaptations(); err != nil {
		return err
	}
	fmt.Printf("Verified %d files, bytes, modes and accepted compatibility at %s (%s).\n", len(m.Files), m.Revision, m.Version)
	return nil
}

func checkAdaptations() error {
	data, err := os.ReadFile(filepath.Join(bundleDir, "adaptations.json"))
	if err != nil {
		return err
	}
	return bundle.ValidateAdaptations(os.DirFS(filepath.Join(bundleDir, "_upstream")), data)
}
func git(root string, args ...string) ([]byte, error) {
	c := exec.Command("git", append([]string{"-C", root}, args...)...)
	b, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, b)
	}
	return b, nil
}
func update(source, revision string) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	if source == root || strings.HasPrefix(source, root+string(os.PathSeparator)) {
		return errors.New("third-party source must be outside the project checkout")
	}
	head, err := git(source, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	sha := strings.TrimSpace(string(head))
	if revision != "" && revision != sha {
		return fmt.Errorf("revision mismatch: got %s", sha)
	}
	status, err := git(source, "status", "--porcelain", "--untracked-files=all", "--", "pstack")
	if err != nil {
		return err
	}
	if len(status) != 0 {
		return errors.New("source pstack tree is dirty")
	}
	listed, err := git(source, "ls-files", "-z", "--", "pstack")
	if err != nil {
		return err
	}
	paths := strings.Split(strings.TrimSuffix(string(listed), "\x00"), "\x00")
	if len(paths) == 0 || paths[0] == "" {
		return errors.New("source has no tracked pstack files")
	}
	sort.Strings(paths)
	var plugin struct {
		Version string `json:"version"`
	}
	b, err := os.ReadFile(filepath.Join(source, "pstack", "plugin.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &plugin); err != nil {
		return err
	}
	if plugin.Version == "" {
		return errors.New("source plugin version missing")
	}
	m := bundle.Manifest{Repository: "https://github.com/cursor/plugins.git", Revision: sha, Version: plugin.Version, SourceRoot: "pstack"}
	stage, err := os.MkdirTemp(bundleDir, ".upstream-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, p := range paths {
		rel := strings.TrimPrefix(p, "pstack/")
		if rel == p || !fs.ValidPath(rel) {
			return fmt.Errorf("unsafe source path %q", p)
		}
		info, err := os.Lstat(filepath.Join(source, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular source %s", p)
		}
		mode := uint32(0644)
		if info.Mode()&0111 != 0 {
			mode = 0755
		}
		b, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		m.Files = append(m.Files, bundle.File{Path: rel, SHA256: hash(b), Mode: mode})
		target := filepath.Join(stage, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, b, fs.FileMode(mode)); err != nil {
			return err
		}
		if err := os.Chmod(target, fs.FileMode(mode)); err != nil {
			return err
		}
	}
	if err := bundle.CheckDirectory(stage, m); err != nil {
		return err
	}
	report, err := makeReport(stage, m)
	if err != nil {
		return err
	}
	dest := filepath.Join(bundleDir, "_upstream")
	backup := dest + ".previous"
	if _, err := os.Lstat(backup); !os.IsNotExist(err) {
		return errors.New("upstream.previous exists; resolve interrupted update before retry")
	}
	hadOld := false
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, backup); err != nil {
			return err
		}
		hadOld = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stage, dest); err != nil {
		if hadOld {
			_ = os.Rename(backup, dest)
		}
		return err
	}
	if err := writeJSON(lockPath, m); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(bundleDir, "manifest.json"), m); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(bundleDir, "review-report.json"), report); err != nil {
		return err
	}
	if hadOld {
		if err := os.RemoveAll(backup); err != nil {
			return err
		}
	}
	fmt.Printf("Imported %s (%s), %d files. Compatibility baseline unchanged; review report then explicitly accept.\n", sha, plugin.Version, len(m.Files))
	return nil
}
func inventory(root string, m bundle.Manifest) ([]Candidate, map[string]bool, error) {
	var out []Candidate
	vocabulary := map[string]bool{}
	for _, f := range m.Files {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path)))
		if err != nil {
			return nil, nil, err
		}
		if !utf8.Valid(b) {
			continue
		}
		for i, line := range strings.Split(string(b), "\n") {
			found := terms.FindAllString(line, -1)
			if len(found) == 0 {
				continue
			}
			set := map[string]bool{}
			for _, s := range found {
				s = strings.ToLower(s)
				set[s] = true
				vocabulary[s] = true
			}
			var sorted []string
			for s := range set {
				sorted = append(sorted, s)
			}
			sort.Strings(sorted)
			out = append(out, Candidate{f.Path, i + 1, line, sorted})
		}
	}
	return out, vocabulary, nil
}
func makeReport(stage string, m bundle.Manifest) (Report, error) {
	report := Report{Revision: m.Revision, Limit: "Broad lexical candidates include false positives, ordinary subprocesses and documentation. They are not semantic proof. Review changed source, role vocabulary, references, unsupported requirements and runtime-contract compatibility before accept."}
	candidates, vocabulary, err := inventory(stage, m)
	if err != nil {
		return report, err
	}
	report.Candidates = candidates
	report.Roles = roleCandidates(stage)
	old, _, _ := readManifest()
	previousRoles := map[string]bool{}
	if len(old.Files) > 0 {
		for _, role := range roleCandidates(filepath.Join(bundleDir, "_upstream")) {
			previousRoles[role] = true
		}
	}
	for _, role := range report.Roles {
		if !previousRoles[role] {
			report.NewRoles = append(report.NewRoles, role)
		}
	}
	prior := map[string]bundle.File{}
	for _, f := range old.Files {
		prior[f.Path] = f
	}
	changed := map[string]bool{}
	for _, f := range m.Files {
		if before, ok := prior[f.Path]; !ok || before != f {
			changed[f.Path] = true
		}
		delete(prior, f.Path)
	}
	for p := range prior {
		changed[p] = true
	}
	for p := range changed {
		report.Changed = append(report.Changed, p)
	}
	sort.Strings(report.Changed)
	delegation := map[string]bool{}
	for _, c := range candidates {
		if changed[c.Path] {
			delegation[c.Path] = true
		}
	}
	oldCandidates, oldVocabulary, _ := inventory(filepath.Join(bundleDir, "_upstream"), old)
	for _, c := range oldCandidates {
		if changed[c.Path] {
			delegation[c.Path] = true
		}
	}
	for p := range delegation {
		report.ChangedDelegation = append(report.ChangedDelegation, p)
	}
	sort.Strings(report.ChangedDelegation)
	for term := range vocabulary {
		if !oldVocabulary[term] {
			report.NewTerms = append(report.NewTerms, term)
		}
	}
	sort.Strings(report.NewTerms)
	return report, nil
}

func verifyExternal(source, revision string) error {
	m, _, err := readManifest()
	if err != nil {
		return err
	}
	if revision != "" && revision != m.Revision {
		return errors.New("requested revision differs from lock")
	}
	if source == "" {
		return nil
	}
	head, err := git(source, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != m.Revision {
		return errors.New("external source revision differs from lock")
	}
	return bundle.CheckDirectory(filepath.Join(source, "pstack"), m)
}

func roleCandidates(root string) []string {
	b, err := os.ReadFile(filepath.Join(root, "skills/setup-pstack/SKILL.md"))
	if err != nil {
		return nil
	}
	roles := map[string]bool{}
	fenced := false
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			continue
		}
		if !fenced || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, ": ")
		if ok && !strings.ContainsAny(name, "/`#") && (strings.Contains(value, "grok-") || strings.Contains(value, "claude-") || strings.Contains(value, "gpt-") || value == "auto" || value == "inherit-parent") {
			roles[name] = true
		}
	}
	var out []string
	for role := range roles {
		out = append(out, role)
	}
	sort.Strings(out)
	return out
}
