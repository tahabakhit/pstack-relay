package relaytest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type templateCase struct{ Name, Template, Mode string }

func templateCases() []templateCase {
	template := "prose\n```\n---\ndescription: example\n---\nfeature, refactoring: grok-fast\nnew role: claude-opus-5-5-xhigh, grok-fast\n```\n"
	var c []templateCase
	for _, mode := range []string{"", "only-claude", "only-codex", "only-pi", "missing"} {
		c = append(c, templateCase{"starter-shape/" + mode, template, mode})
	}
	c = append(c, templateCase{"missing template", "no block", ""}, templateCase{"invalid template", "```\nfeature, refactoring: bad:model\n```", ""})
	return c
}
func templateRun(t *testing.T, c templateCase) expected {
	t.Helper()
	dir, env := isolated(t)
	script := fixturePlugin(t, dir, c.Template)
	env = append(env, "FAKE_MODE="+c.Mode)
	cmd := exec.Command("sh", script, "setup", "--dry-run", "--json")
	cmd.Env = env
	cmd.Dir = dir
	b, e := cmd.Output()
	if e != nil {
		return expected{Error: true}
	}
	v := object(t, b)
	return expected{Output: v["sheet"]}
}
func fixturePlugin(t *testing.T, dir, template string) string {
	t.Helper()
	root := rootpath(t)
	for _, p := range []string{"bin/pstack-relay", "lib/relay.awk"} {
		b, e := os.ReadFile(filepath.Join(root, "overlay", p))
		if e != nil {
			t.Fatal(e)
		}
		write(t, filepath.Join(dir, "plugin", p), string(b))
	}
	write(t, filepath.Join(dir, "plugin/lib/setup-pstack.md"), template)
	return filepath.Join(dir, "plugin/bin/pstack-relay")
}
func TestTemplatePort(t *testing.T) {
	data, e := os.ReadFile("testdata/template-expectations.json")
	if e != nil {
		t.Fatal(e)
	}
	var wants map[string]expected
	if e = json.Unmarshal(data, &wants); e != nil {
		t.Fatal(e)
	}
	for _, c := range templateCases() {
		t.Run(c.Name, func(t *testing.T) {
			got := templateRun(t, c)
			want, ok := wants[c.Name]
			if !ok || got.Error != want.Error || got.Output != want.Output {
				t.Fatalf("got %+v want %+v", got, want)
			}
		})
	}
}
