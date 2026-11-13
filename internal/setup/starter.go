// Package setup builds the shared starter sheet from upstream's current template.
package setup

import (
	"fmt"
	"strings"

	"github.com/tahabakhit/pstack-rex/internal/discover"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
)

func Starter(template string, catalogs []discover.Catalog) (sheet.Sheet, error) {
	// Only the fenced block containing upstream's model rule is a template.
	var body string
	for i, block := range strings.Split(template, "```") {
		if i%2 == 1 && strings.Contains(block, "feature, refactoring:") {
			lines := strings.Split(block, "\n")
			for j, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "feature, refactoring:") {
					body = strings.Join(lines[j:], "\n")
					break
				}
			}
			break
		}
	}
	if body == "" {
		return sheet.Sheet{}, fmt.Errorf("upstream setup template has no role block")
	}
	s, err := sheet.Parse(body)
	if err != nil {
		return sheet.Sheet{}, fmt.Errorf("upstream template: %w", err)
	}
	available := map[sheet.Harness][]string{}
	for _, c := range catalogs {
		if c.Installed && len(c.Models) > 0 {
			available[c.Harness] = c.Models
		}
	}
	if len(available) == 0 {
		return sheet.Sheet{}, fmt.Errorf("no installed harness offers a model")
	}
	for role, seats := range s.Roles {
		for i, seat := range seats {
			family, effort := slugParts(seat.Model)
			h := sheet.Codex
			model := ""
			if family != "" && len(available[sheet.Claude]) > 0 {
				h, model = sheet.Claude, family
			} else if len(available[sheet.Codex]) > 0 {
				model = available[sheet.Codex][0]
			} else if len(available[sheet.Claude]) > 0 {
				h, model = sheet.Claude, available[sheet.Claude][0]
			} else {
				h, model = sheet.Pi, available[sheet.Pi][0]
			}
			seats[i] = sheet.Seat{Harness: h, Model: model, Effort: effort}
		}
		s.Roles[role] = seats
	}
	s.DefaultEffort = "session"
	return s, nil
}

func slugParts(slug string) (string, string) {
	parts := strings.Split(slug, "-")
	family, effort := "", "high"
	if len(parts) > 1 && parts[0] == "claude" {
		family = parts[1]
	}
	for i := len(parts) - 1; i >= 0; i-- {
		if sheet.ValidEffort(parts[i]) && parts[i] != "session" {
			effort = parts[i]
			break
		}
	}
	return family, effort
}
