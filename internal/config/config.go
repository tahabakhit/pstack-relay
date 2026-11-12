package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Target struct {
	Harness string `json:"harness"`
	Model   string `json:"model"`
	Effort  string `json:"effort,omitempty"`
	Backend string `json:"backend"`
	Auth    string `json:"auth"`
}
type Config struct {
	Version        int                 `json:"version"`
	Session        string              `json:"session"`
	MaxConcurrent  int                 `json:"max_concurrent"`
	TimeoutSeconds int                 `json:"timeout_seconds"`
	Roles          map[string][]Target `json:"roles"`
}

var Roles = []string{"feature, refactoring", "bug-fix", "perf-issue", "hillclimb", "judgment and prose", "hardest tasks", "how explorer", "how explainer", "why investigators", "why synthesizer", "reflect tooling", "reflect judgment, divergent, synthesizer", "arena runners", "arena cross-judge pool", "swarm workers", "architect runners", "interrogate reviewers"}

func Default() Config {
	return Config{Version: 1, Session: "pstack-rex", MaxConcurrent: 4, TimeoutSeconds: 600, Roles: map[string][]Target{}}
}
func DefaultPath() string {
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "pstack-rex", "config.json")
}
func decode(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("config %s: expected JSON object", path)
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(dst); err != nil {
		return fmt.Errorf("config %s: %w", path, err)
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("config %s: expected one JSON object", path)
	}
	return nil
}
func Load(path, cwd string) (Config, error) {
	c := Default()
	if path == "" {
		path = DefaultPath()
	} else if !filepath.IsAbs(path) && cwd != "" {
		path = filepath.Join(cwd, path)
	}
	if err := decode(path, &c); err != nil {
		return Config{}, err
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// LoadProject is opt-in. Project policy may replace role lists but cannot change auth policy.
func LoadProject(base Config, path string) (Config, error) {
	var p struct {
		Version int                 `json:"version"`
		Roles   map[string][]Target `json:"roles"`
	}
	if err := decode(path, &p); err != nil {
		return Config{}, err
	}
	if p.Version != 1 {
		return Config{}, fmt.Errorf("unsupported project config version %d", p.Version)
	}
	c := base
	c.Roles = make(map[string][]Target, len(base.Roles)+len(p.Roles))
	for k, v := range base.Roles {
		c.Roles[k] = append([]Target(nil), v...)
	}
	for k, v := range p.Roles {
		for _, t := range v {
			if t.Auth != "existing" {
				return Config{}, fmt.Errorf("project role %s must use existing auth", k)
			}
		}
		c.Roles[k] = append([]Target(nil), v...)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}
func knownRole(role string) bool {
	for _, r := range Roles {
		if role == r {
			return true
		}
	}
	return false
}
func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Session == "" || c.MaxConcurrent < 1 || c.TimeoutSeconds < 1 {
		return fmt.Errorf("session, positive max_concurrent and timeout_seconds required")
	}
	for r, seats := range c.Roles {
		if !knownRole(r) {
			return fmt.Errorf("unknown role %q", r)
		}
		if len(seats) == 0 {
			return fmt.Errorf("role %s has no seats", r)
		}
		for _, t := range seats {
			if err := ValidateTarget(t, true); err != nil {
				return fmt.Errorf("role %s: %w", r, err)
			}
		}
	}
	return nil
}
func Validate(c Config) error { return c.Validate() }
func ValidateTarget(t Target, allowInherit bool) error {
	switch t.Harness {
	case "codex", "claude", "pi":
	case "inherit-parent":
		if !allowInherit {
			return fmt.Errorf("unresolved harness inheritance")
		}
	default:
		return fmt.Errorf("unsupported harness %q", t.Harness)
	}
	if strings.TrimSpace(t.Model) == "" {
		return fmt.Errorf("explicit model required")
	}
	if !allowInherit && (t.Model == "inherit-parent" || t.Effort == "inherit-session" || t.Effort == "inherit-parent") {
		return fmt.Errorf("unresolved model or effort inheritance")
	}
	if t.Backend != "auto" && t.Backend != "rex" {
		return fmt.Errorf("unsupported backend %q; native execution is unavailable", t.Backend)
	}
	if t.Auth != "existing" {
		return fmt.Errorf("auth %q unsupported: only existing auth is available; funding source is unknown", t.Auth)
	}
	if t.Effort != "" && t.Effort != "default" && t.Effort != "inherit-session" && t.Effort != "inherit-parent" {
		allowed := map[string]bool{"off": true, "minimal": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true}
		if !allowed[t.Effort] {
			return fmt.Errorf("unsupported effort %q", t.Effort)
		}
	}
	return nil
}
func (c Config) Resolve(role string, index int, parent *Target) (Target, error) {
	if !knownRole(role) {
		return Target{}, fmt.Errorf("unknown role %q", role)
	}
	seats := c.Roles[role]
	if index < 0 || index >= len(seats) {
		return Target{}, fmt.Errorf("role %q seat %d unavailable", role, index)
	}
	t := seats[index]
	if t.Harness == "inherit-parent" || t.Model == "inherit-parent" || t.Effort == "inherit-session" || t.Effort == "inherit-parent" {
		if parent == nil {
			return Target{}, fmt.Errorf("concrete parent required for inheritance")
		}
		if err := ValidateTarget(*parent, false); err != nil {
			return Target{}, fmt.Errorf("parent: %w", err)
		}
		if t.Harness == "inherit-parent" {
			t.Harness = parent.Harness
		}
		if t.Model == "inherit-parent" {
			t.Model = parent.Model
		}
		if t.Effort == "inherit-session" || t.Effort == "inherit-parent" {
			if parent.Effort == "" || parent.Effort == "default" {
				return Target{}, fmt.Errorf("concrete parent effort required")
			}
			t.Effort = parent.Effort
		}
	}
	if t.Backend == "auto" {
		t.Backend = "rex"
	}
	if err := ValidateTarget(t, false); err != nil {
		return Target{}, err
	}
	return t, nil
}
