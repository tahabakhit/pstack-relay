package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUpdateRequiresCompatibilityAcceptance(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.MkdirAll(filepath.Join(root, bundleDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(external, "pstack"), 0755); err != nil {
		t.Fatal(err)
	}
	put := func(p, s string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(s), 0644); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(root, bundleDir, "adaptations.json"), `[]`)
	put(filepath.Join(external, "pstack/plugin.json"), `{"version":"test"}`)
	put(filepath.Join(external, "pstack/SKILL.md"), "# Test\nSpawn the model reviewer.\n")
	command := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-C", external}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	command("init", "-q")
	command("add", ".")
	command("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "initial")
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"update", "--source", external}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check"}); err == nil {
		t.Fatal("accepted unreviewed source")
	}
	if err := run([]string{"accept", "--source", external}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check", "--source", external}); err != nil {
		t.Fatal(err)
	}
	baseline, err := os.ReadFile(filepath.Join(bundleDir, "compatibility.json"))
	if err != nil {
		t.Fatal(err)
	}
	put(filepath.Join(external, "pstack/new-unknown.md"), "# Never-before-seen delegation expression\n")
	command("add", ".")
	command("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "change")
	if err := run([]string{"update", "--source", external}); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(filepath.Join(bundleDir, "compatibility.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(baseline, current) {
		t.Fatal("update silently accepted baseline")
	}
	if err := run([]string{"check"}); err == nil {
		t.Fatal("accepted new source without compatibility review")
	}
	var r Report
	b, err := os.ReadFile(filepath.Join(bundleDir, "review-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Changed) != 1 || r.Changed[0] != "new-unknown.md" {
		t.Fatalf("changed report: %#v", r.Changed)
	}
	if err := run([]string{"accept"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"check", "--revision", "wrong"}); err == nil {
		t.Fatal("ignored revision")
	}
	put(filepath.Join(external, "pstack/SKILL.md"), "dirty")
	if err := run([]string{"update", "--source", external}); err == nil {
		t.Fatal("accepted dirty source")
	}
}
func TestInventoryMatchesAndHasHonestScope(t *testing.T) {
	if !terms.MatchString("Task Agent spawn_agent steer resume model cloud_base_branch MCP") {
		t.Fatal("matcher is broken")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "skills/setup-pstack"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills/setup-pstack/SKILL.md"), []byte("```\nnew role: gpt-test\n```\n"), 0644); err != nil {
		t.Fatal(err)
	}
	roles := roleCandidates(root)
	if len(roles) != 1 || roles[0] != "new role" {
		t.Fatalf("roles %#v", roles)
	}
}
