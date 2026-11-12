package config

import (
	"os"
	"path/filepath"
	"testing"
)

func target() Target {
	return Target{Harness: "codex", Model: "exact-model", Backend: "auto", Auth: "existing"}
}
func TestResolveSeatsAndInheritance(t *testing.T) {
	c := Default()
	a := target()
	b := a
	b.Model = "second"
	c.Roles["arena runners"] = []Target{a, b}
	got, err := c.Resolve("arena runners", 1, nil)
	if err != nil || got.Model != "second" || got.Backend != "rex" {
		t.Fatalf("%+v %v", got, err)
	}
	b.Model = "inherit-parent"
	b.Effort = "inherit-session"
	c.Roles["arena runners"][1] = b
	if _, err = c.Resolve("arena runners", 1, nil); err == nil {
		t.Fatal("accepted missing parent")
	}
	p := a
	p.Effort = "high"
	got, err = c.Resolve("arena runners", 1, &p)
	if err != nil || got.Model != p.Model || got.Effort != "high" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = c.Resolve("typo", 0, nil); err == nil {
		t.Fatal("accepted unknown role")
	}
	if len(Roles) != 17 {
		t.Fatalf("roles %d", len(Roles))
	}
}
func TestStrictConfigAndExplicitProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"version":1,"session":"test","roles":{},"typo":true}`)
	if _, err := Load(path, dir); err == nil {
		t.Fatal("accepted unknown field")
	}
	write(`{"version":1,"session":"test","roles":{}} {}`)
	if _, err := Load(path, dir); err == nil {
		t.Fatal("accepted trailing JSON")
	}
	write(`{"version":1,"session":"test","roles":{}}`)
	if err := os.WriteFile(filepath.Join(dir, ".pstack-rex.json"), []byte(`invalid`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, dir)
	if err != nil || c.MaxConcurrent != 4 || c.TimeoutSeconds != 600 {
		t.Fatalf("%+v %v", c, err)
	}
	write(`{"version":1,"session":"forbidden","roles":{}}`)
	if _, err = LoadProject(c, path); err == nil {
		t.Fatal("project changed session")
	}
}
func TestRejectUnsupportedSelections(t *testing.T) {
	for _, change := range []func(*Target){func(t *Target) { t.Model = "" }, func(t *Target) { t.Auth = "api" }, func(t *Target) { t.Auth = "subscription" }, func(t *Target) { t.Backend = "native" }, func(t *Target) { t.Harness = "auto" }} {
		a := target()
		change(&a)
		if ValidateTarget(a, false) == nil {
			t.Fatalf("accepted %+v", a)
		}
	}
}

func TestAutoModelRequiresConcreteParent(t *testing.T) {
	c := Default()
	a := target()
	a.Model = "auto"
	c.Roles["how explorer"] = []Target{a}
	for _, model := range []string{"", "auto", "inherit-parent"} {
		t.Run("unresolved_"+model, func(t *testing.T) {
			parent := target()
			parent.Model = model
			if got, err := c.Resolve("how explorer", 0, &parent); err == nil {
				t.Fatalf("resolved auto using unresolved parent: %+v", got)
			}
		})
	}
	if got, err := c.Resolve("how explorer", 0, nil); err == nil {
		t.Fatalf("resolved auto without a parent: %+v", got)
	}
	parent := target()
	parent.Model = "concrete-parent-model"
	got, err := c.Resolve("how explorer", 0, &parent)
	if err != nil || got.Model != parent.Model || got.Backend != "rex" {
		t.Fatalf("concrete parent resolution=%+v err=%v", got, err)
	}
	if err := ValidateTarget(a, false); err == nil {
		t.Fatal("auto model accepted at execution boundary")
	}
}
