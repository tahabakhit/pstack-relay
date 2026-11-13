// Package sheet reads the shared pstack role sheet.
package sheet

import (
	"bufio"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type Harness string

const (
	Claude Harness = "claude"
	Codex  Harness = "codex"
	Pi     Harness = "pi"
)

func ValidHarness(h Harness) bool { return h == Claude || h == Codex || h == Pi }
func ValidEffort(e string) bool {
	switch e {
	case "low", "medium", "high", "xhigh", "max", "session":
		return true
	}
	return false
}

type Seat struct {
	Harness Harness `json:"harness"`
	Model   string  `json:"model"`
	Effort  string  `json:"effort"`
}
type Sheet struct {
	Roles         map[string][]Seat `json:"roles"`
	Order         []string          `json:"order"`
	DefaultEffort string            `json:"default_effort"`
	SessionHook   bool              `json:"session_hook"`
}

func Parse(text string) (Sheet, error) {
	s := Sheet{Roles: make(map[string][]Seat), SessionHook: true}
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		key, value, ok := strings.Cut(raw, ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		fail := func(reason string) (Sheet, error) { return Sheet{}, fmt.Errorf("line %d: %s", line, reason) }
		if !ok || key == "" || value == "" {
			return fail("expected role: value")
		}
		switch key {
		case "default effort":
			if !ValidEffort(value) {
				return fail("unknown effort " + value)
			}
			s.DefaultEffort = value
			continue
		case "session hook":
			if value != "on" && value != "off" {
				return fail("session hook must be on or off")
			}
			s.SessionHook = value == "on"
			continue
		}
		var seats []Seat
		for _, part := range strings.Split(value, ",") {
			fields := strings.Fields(part)
			if len(fields) < 1 || len(fields) > 2 {
				return fail("expected [harness:]model [@effort]")
			}
			seat := Seat{Model: fields[0]}
			if h, m, qualified := strings.Cut(seat.Model, ":"); qualified {
				seat.Harness, seat.Model = Harness(h), m
				if !ValidHarness(seat.Harness) {
					return fail("unknown harness " + h)
				}
				if m == "" || strings.Contains(m, ":") {
					return fail("invalid model " + m)
				}
			}
			if len(fields) == 2 {
				e := strings.TrimPrefix(fields[1], "@")
				if e == fields[1] || !ValidEffort(e) {
					return fail("unknown effort " + fields[1])
				}
				seat.Effort = e
			}
			seats = append(seats, seat)
		}
		for _, role := range strings.Split(key, ",") {
			role = strings.TrimSpace(role)
			if role == "" || role == "default effort" || role == "session hook" {
				return fail("invalid role " + role)
			}
			if _, exists := s.Roles[role]; !exists {
				s.Order = append(s.Order, role)
			}
			s.Roles[role] = append([]Seat(nil), seats...)
		}
	}
	if err := scanner.Err(); err != nil {
		return Sheet{}, fmt.Errorf("line %d: %w", line+1, err)
	}
	return s, nil
}

func (s Sheet) Format() string {
	var b strings.Builder
	seen := make(map[string]bool)
	order := append([]string(nil), s.Order...)
	var extra []string
	for role := range s.Roles {
		if !contains(order, role) {
			extra = append(extra, role)
		}
	}
	sort.Strings(extra)
	order = append(order, extra...)
	for _, role := range order {
		seats, exists := s.Roles[role]
		if !exists || seen[role] {
			continue
		}
		seen[role] = true
		values := make([]string, 0, len(seats))
		for _, seat := range seats {
			value := seat.Model
			if seat.Harness != "" {
				value = string(seat.Harness) + ":" + value
			}
			if seat.Effort != "" {
				value += " @" + seat.Effort
			}
			values = append(values, value)
		}
		fmt.Fprintf(&b, "%s: %s\n", role, strings.Join(values, ", "))
	}
	if s.DefaultEffort != "" {
		fmt.Fprintf(&b, "default effort: %s\n", s.DefaultEffort)
	}
	hook := "off"
	if s.SessionHook {
		hook = "on"
	}
	fmt.Fprintf(&b, "session hook: %s\n", hook)
	return b.String()
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func Path(getenv func(string) string) string {
	if override := getenv("PSTACK_REX_SHEET"); override != "" {
		return override
	}
	root := getenv("XDG_CONFIG_HOME")
	if root == "" {
		root = filepath.Join(getenv("HOME"), ".config")
	}
	return filepath.Join(root, "pstack-rex", "pstack-models.md")
}
