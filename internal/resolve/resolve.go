// Package resolve maps configured seats to the parent's native tools or host backend.
package resolve

import (
	"fmt"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
)

type Harness = sheet.Harness
type Sheet = sheet.Sheet
type Backend string
type Mode string

const (
	Orca     Backend = "orca"
	Rex      Backend = "rex"
	None     Backend = "none"
	Native   Mode    = "native"
	OrcaMode Mode    = "orca"
	RexMode  Mode    = "rex"
	Fallback Mode    = "fallback"
)

type Env struct {
	Parent  Harness `json:"parent"`
	Backend Backend `json:"backend"`
}

func Detect(getenv func(string) string) Env {
	env := Env{Backend: None}
	switch {
	case getenv("CLAUDECODE") == "1":
		env.Parent = sheet.Claude
	case getenv("CODEX_THREAD_ID") != "":
		env.Parent = sheet.Codex
	case getenv("PI_CODING_AGENT") == "1" || getenv("PI_SESSION") != "":
		env.Parent = sheet.Pi
	}
	switch {
	case getenv("ORCA_TERMINAL_HANDLE") != "":
		env.Backend = Orca
	case getenv("REX_SESSION") != "" && getenv("REX_BLOCK") != "":
		env.Backend = Rex
	}
	return env
}

type Placement struct {
	Role    string   `json:"role"`
	Seat    int      `json:"seat"`
	Mode    Mode     `json:"mode"`
	Harness Harness  `json:"harness"`
	Model   string   `json:"model"`
	Effort  string   `json:"effort"`
	Command []string `json:"command,omitempty"`
	Note    string   `json:"note,omitempty"`
}
type Options struct {
	OwnWorktree bool
	Branch      string
	RexBinary   string
}

func Resolve(s Sheet, role string, env Env, opts Options) ([]Placement, error) {
	if env.Parent != "" && !sheet.ValidHarness(env.Parent) {
		return nil, fmt.Errorf("unknown parent harness %q", env.Parent)
	}
	if env.Backend != Orca && env.Backend != Rex && env.Backend != None && env.Backend != "" {
		return nil, fmt.Errorf("unknown backend %q", env.Backend)
	}
	seats, ok := s.Roles[role]
	if !ok {
		return []Placement{{Role: role, Mode: Native, Harness: env.Parent, Note: "role not in sheet; use the skill's default model"}}, nil
	}
	placements := make([]Placement, 0, len(seats))
	for i, seat := range seats {
		model := seat.Model
		if model == "inherit-parent" || model == "auto" {
			model = ""
		}
		effort := seat.Effort
		if effort == "" {
			effort = s.DefaultEffort
		}
		if effort == "session" {
			effort = ""
		}
		p := Placement{Role: role, Seat: i, Mode: Native, Harness: env.Parent, Model: model, Effort: effort}
		switch {
		case env.Parent == "":
			p.Note = "parent harness not detected"
		case seat.Harness == "" || seat.Harness == env.Parent:
		case env.Backend == Orca && seat.Harness != sheet.Pi:
			p.Mode, p.Harness = OrcaMode, seat.Harness
			p.Command = []string{"orca", "orchestration", "worker-start", "--agent", string(seat.Harness), "--model", model}
			if effort != "" {
				p.Command = append(p.Command, "--effort", effort)
			}
			if opts.OwnWorktree {
				if opts.Branch == "" {
					return nil, fmt.Errorf("own worktree requires a current branch for --base-branch")
				}
				p.Command = append(p.Command, "--worktree", "new-child", "--base-branch", opts.Branch)
			} else {
				p.Command = append(p.Command, "--worktree", "current")
			}
			p.Command = append(p.Command, "--spec-file", "{brief}")
		case env.Backend == Rex:
			p.Mode, p.Harness = RexMode, seat.Harness
			binary := opts.RexBinary
			if binary == "" {
				binary = "pstack-rex"
			}
			p.Command = []string{binary, "rex", "start", "--harness", string(seat.Harness), "--model", model}
			if effort != "" {
				p.Command = append(p.Command, "--effort", effort)
			}
			if opts.OwnWorktree {
				p.Command = append(p.Command, "--own-worktree")
			}
			p.Command = append(p.Command, "--brief", "{brief}")
		default:
			p.Mode, p.Model = Fallback, ""
			reason := "no delegation backend detected"
			if env.Backend == Orca && seat.Harness == sheet.Pi {
				reason = "Orca has no pi agent"
			}
			p.Note = fmt.Sprintf("configured seat %s:%s: %s; fall back to parent model", seat.Harness, seat.Model, reason)
		}
		placements = append(placements, p)
	}
	return placements, nil
}
