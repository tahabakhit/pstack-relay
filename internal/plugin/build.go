package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const Version = "2.0.0"
const repository = "https://github.com/tahabakhit/pstack-relay"
const description = "Cursor pstack workflows with native, Orca, Rex, and hc delegation for Claude Code and Codex."

type generatedFile struct {
	data []byte
	mode fs.FileMode
}

// Build generates the installable plugin and both marketplace manifests from a checkout.
// Check compares bytes, executable bits, and the complete file list without writing.
func Build(root string, check bool) error {
	files, err := generate(root)
	if err != nil {
		return err
	}
	if check {
		seen := map[string]bool{}
		for name, want := range files {
			path := filepath.Join(root, filepath.FromSlash(name))
			info, err := os.Lstat(path)
			if err != nil {
				return fmt.Errorf("generated file missing: %s: %w", name, err)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("generated file is not regular: %s", name)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !bytes.Equal(got, want.data) || info.Mode().Perm()&0111 != want.mode&0111 {
				return fmt.Errorf("generated file differs: %s; run go run ./cmd/build-plugin", name)
			}
			seen[name] = true
		}
		return filepath.WalkDir(filepath.Join(root, "plugins", "pstack"), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if !seen[filepath.ToSlash(rel)] {
				return fmt.Errorf("unexpected generated file: %s", rel)
			}
			return nil
		})
	}
	// Generate and validate everything before replacing the owned output tree.
	if err := os.RemoveAll(filepath.Join(root, "plugins", "pstack")); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		file := files[name]
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, file.data, file.mode); err != nil {
			return err
		}
		if err := os.Chmod(path, file.mode); err != nil {
			return err
		}
	}
	return nil
}

func generate(root string) (map[string]generatedFile, error) {
	var rules []Adaptation
	data, err := os.ReadFile(filepath.Join(root, "internal", "plugin", "adaptations.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	source := os.DirFS(filepath.Join(root, "upstream"))
	if err := ValidateAdaptations(source, rules); err != nil {
		return nil, err
	}
	files := map[string]generatedFile{}
	prefix := "plugins/pstack/"
	for _, dir := range []string{"skills", "agents"} {
		err := fs.WalkDir(source, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("upstream file is not regular: %s", path)
			}
			b, err := fs.ReadFile(source, path)
			if err != nil {
				return err
			}
			files[prefix+path] = generatedFile{Transform(path, b, rules), info.Mode().Perm()}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, pair := range [][2]string{{"overlay/bin/pstack-relay", "bin/pstack-relay"}, {"overlay/lib/relay.awk", "lib/relay.awk"}, {"upstream/REVISION", "UPSTREAM-REVISION"}, {"upstream/LICENSE", "LICENSE"}, {"upstream/README.md", "UPSTREAM-README.md"}, {"overlay/delegation.md", "delegation.md"}, {"overlay/hooks/session-start.sh", "claude-hooks/session-start.sh"}} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pair[0])))
		if err != nil {
			return nil, err
		}
		mode := fs.FileMode(0644)
		if pair[1] == "claude-hooks/session-start.sh" || pair[1] == "bin/pstack-relay" {
			mode = 0755
		}
		files[prefix+pair[1]] = generatedFile{b, mode}
	}
	putJSON := func(name string, value any) error {
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		files[name] = generatedFile{append(b, '\n'), 0644}
		return nil
	}
	manifests := map[string]any{
		prefix + ".claude-plugin/plugin.json": map[string]any{"name": "pstack", "version": Version, "description": description, "license": "MIT", "repository": repository, "author": map[string]string{"name": "Taha Bakhit"}, "hooks": "./claude-hooks/hooks.json"},
		prefix + ".codex-plugin/plugin.json":  map[string]any{"name": "pstack", "version": Version, "description": description, "license": "MIT", "repository": repository, "skills": "./skills/", "interface": map[string]string{"displayName": "pstack", "shortDescription": "pstack workflows across Claude Code and Codex"}},
		".claude-plugin/marketplace.json":     map[string]any{"name": "pstack-relay", "owner": map[string]string{"name": "Taha Bakhit"}, "plugins": []any{map[string]any{"name": "pstack", "source": "./plugins/pstack", "description": description}}},
		".agents/plugins/marketplace.json":    map[string]any{"name": "pstack-relay", "interface": map[string]string{"displayName": "pstack-relay"}, "plugins": []any{map[string]any{"name": "pstack", "source": map[string]string{"source": "local", "path": "./plugins/pstack"}, "policy": map[string]any{"installation": "AVAILABLE", "authentication": "ON_INSTALL", "products": []string{"CODEX"}}}}},
	}
	// Codex holds every new plugin hook behind a trust prompt that blocks
	// session start, including unattended Orca workers, and it also loads
	// hooks/hooks.json by convention. So the Claude hook lives in
	// claude-hooks/, named by the Claude manifest, and Codex sees no hook.
	// Each entrypoint's pointer to delegation.md covers Codex.
	manifests[prefix+"claude-hooks/hooks.json"] = map[string]any{"hooks": map[string]any{"SessionStart": []any{map[string]any{"matcher": "startup|resume|clear|compact", "hooks": []any{map[string]string{"type": "command", "command": "\"${CLAUDE_PLUGIN_ROOT}/claude-hooks/session-start.sh\" claude"}}}}}}
	for name, manifest := range manifests {
		if err := putJSON(name, manifest); err != nil {
			return nil, err
		}
	}
	return files, nil
}
