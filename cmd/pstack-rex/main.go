package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pstackrex "github.com/tahabakhit/pstack-rex"
	"github.com/tahabakhit/pstack-rex/internal/discover"
	"github.com/tahabakhit/pstack-rex/internal/plugin"
	"github.com/tahabakhit/pstack-rex/internal/resolve"
	"github.com/tahabakhit/pstack-rex/internal/rexworker"
	"github.com/tahabakhit/pstack-rex/internal/setup"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
)

type runner func(string, ...string) (string, error)
type cli struct {
	out, errOut io.Writer
	in          io.Reader
	getenv      func(string) string
	run         runner
}

func main() {
	c := cli{os.Stdout, os.Stderr, os.Stdin, os.Getenv, runCommand}
	if err := c.execute(os.Args[1:]); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(c.errOut, "pstack-rex:", err)
		os.Exit(1)
	}
}
func runCommand(name string, args ...string) (string, error) {
	// Keep model-list stdout parseable when a harness writes warnings to stderr.
	cmd := exec.Command(name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return string(data), fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return string(data), nil
}
func (c cli) flags(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(c.errOut)
	return f
}
func (c cli) json(value any) error {
	e := json.NewEncoder(c.out)
	e.SetIndent("", "  ")
	return e.Encode(value)
}
func (c cli) execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pstack-rex <models|resolve|setup|doctor|plugin build|rex start|rex report|rex wait|version>")
	}
	switch args[0] {
	case "version":
		f := c.flags("version")
		j := f.Bool("json", false, "print JSON")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		if err := noArgs(f); err != nil {
			return err
		}
		revision := strings.TrimSpace(pstackrex.UpstreamRevision)
		if *j {
			return c.json(map[string]string{"version": plugin.Version, "upstream": revision})
		}
		fmt.Fprintf(c.out, "pstack-rex %s\nupstream %s\n", plugin.Version, revision)
		return nil
	case "models":
		return c.models(args[1:])
	case "resolve":
		return c.resolve(args[1:])
	case "setup":
		return c.setup(args[1:])
	case "doctor":
		return c.doctor(args[1:])
	case "plugin":
		if len(args) < 2 || args[1] != "build" {
			return fmt.Errorf("usage: pstack-rex plugin build [--check] [--json]")
		}
		f := c.flags("plugin build")
		check := f.Bool("check", false, "check generated files without writing")
		j := f.Bool("json", false, "print JSON")
		if err := f.Parse(args[2:]); err != nil {
			return err
		}
		if err := noArgs(f); err != nil {
			return err
		}
		root, err := checkoutRoot()
		if err != nil {
			return err
		}
		if err := plugin.Build(root, *check); err != nil {
			return err
		}
		if *j {
			return c.json(map[string]any{"path": filepath.Join(root, "plugins", "pstack-rex"), "check": *check, "ok": true})
		}
		if *check {
			fmt.Fprintln(c.out, "plugin tree is up to date")
		} else {
			fmt.Fprintln(c.out, "built plugins/pstack-rex and marketplace manifests")
		}
		return nil
	case "rex":
		return c.rex(args[1:])
	case "help", "--help", "-h":
		fmt.Fprintln(c.out, "Usage: pstack-rex <models|resolve|setup|doctor|plugin build|rex start|rex report|rex wait|version>\nUse COMMAND --help for options.")
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func noArgs(f *flag.FlagSet) error {
	if f.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(f.Args(), " "))
	}
	return nil
}
func readSheet(path string) (sheet.Sheet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return sheet.Sheet{}, err
	}
	return sheet.Parse(string(b))
}
func writeSheet(path string, s sheet.Sheet, replace bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	// Atomic replacement avoids exposing a partial sheet to session hooks.
	f, err := os.CreateTemp(filepath.Dir(path), ".pstack-models-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := io.WriteString(f, s.Format()); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if replace {
		return os.Rename(f.Name(), path)
	}
	// A hard link fails when the destination exists, so a sheet that appeared
	// after setup looked is kept rather than replaced.
	return os.Link(f.Name(), path)
}
func (c cli) models(args []string) error {
	f := c.flags("models")
	path := f.String("sheet", sheet.Path(c.getenv), "shared model sheet path")
	set := f.String("set", "", "role=value[, value]")
	j := f.Bool("json", false, "print JSON")
	if err := f.Parse(args); err != nil {
		return err
	}
	if err := noArgs(f); err != nil {
		return err
	}
	s, err := readSheet(*path)
	if errors.Is(err, os.ErrNotExist) && *set != "" {
		s, err = sheet.Parse("")
	}
	if err != nil {
		return err
	}
	if *set != "" {
		role, value, ok := strings.Cut(*set, "=")
		role = strings.TrimSpace(role)
		if !ok || role == "" || strings.ContainsAny(role, ":\n\r") || strings.ContainsAny(value, "\n\r") {
			return fmt.Errorf("--set requires one role=value[, value]")
		}
		edit, err := sheet.Parse(role + ": " + value)
		if err != nil {
			return err
		}
		switch role {
		case "default effort":
			s.DefaultEffort = edit.DefaultEffort
		case "session hook":
			s.SessionHook = edit.SessionHook
		default:
			key := edit.Order[0]
			if _, ok := s.Roles[key]; !ok {
				s.Order = append(s.Order, key)
			}
			s.Roles[key] = edit.Roles[key]
		}
		if err := writeSheet(*path, s, true); err != nil {
			return err
		}
	}
	if *j {
		return c.json(map[string]any{"path": *path, "sheet": s, "text": s.Format()})
	}
	_, err = io.WriteString(c.out, s.Format())
	return err
}
func (c cli) resolve(args []string) error {
	f := c.flags("resolve")
	role := f.String("role", "", "role name")
	own := f.Bool("own-worktree", false, "create a worker worktree")
	f.BoolVar(own, "worktree-own", false, "alias for --own-worktree")
	parent := f.String("parent", "", "override detected parent harness")
	backend := f.String("backend", "", "override detected backend")
	path := f.String("sheet", sheet.Path(c.getenv), "shared sheet path")
	f.Bool("json", true, "placements are always JSON")
	if err := f.Parse(args); err != nil {
		return err
	}
	if err := noArgs(f); err != nil {
		return err
	}
	if *role == "" {
		return fmt.Errorf("--role is required")
	}
	env := resolve.Detect(c.getenv)
	if *parent != "" {
		if !sheet.ValidHarness(sheet.Harness(*parent)) {
			return fmt.Errorf("unknown parent harness %q", *parent)
		}
		env.Parent = sheet.Harness(*parent)
	}
	if *backend != "" {
		env.Backend = resolve.Backend(*backend)
	}
	s, err := readSheet(*path)
	if errors.Is(err, os.ErrNotExist) {
		s, err = sheet.Parse("")
	}
	if err != nil {
		return err
	}
	branch, branchErr := c.run("git", "branch", "--show-current")
	if *own && branchErr != nil {
		return branchErr
	}
	placements, err := resolve.Resolve(s, *role, env, resolve.Options{OwnWorktree: *own, Branch: strings.TrimSpace(branch)})
	if err != nil {
		return err
	}
	return c.json(placements)
}

type commandResult struct {
	Command []string `json:"command"`
	Output  string   `json:"output,omitempty"`
	Error   string   `json:"error,omitempty"`
	DryRun  bool     `json:"dry_run"`
}
type setupResult struct {
	Path     string             `json:"path"`
	Sheet    string             `json:"sheet"`
	Catalogs []discover.Catalog `json:"catalogs"`
	Commands []commandResult    `json:"commands"`
	Notes    []string           `json:"notes"`
}

func (c cli) setup(args []string) error {
	f := c.flags("setup")
	yes := f.Bool("yes", false, "install without prompting")
	force := f.Bool("force", false, "replace the existing sheet with starter choices")
	source := f.String("source", "tahabakhit/pstack-rex", "marketplace path or owner/repo")
	dry := f.Bool("dry-run", false, "show changes and commands without applying them")
	j := f.Bool("json", false, "print JSON")
	path := f.String("sheet", sheet.Path(c.getenv), "shared sheet path")
	if err := f.Parse(args); err != nil {
		return err
	}
	if err := noArgs(f); err != nil {
		return err
	}
	catalogs := discover.Discover(c.run)
	discover.PreferConfigured(catalogs, c.codexConfig())
	s, err := readSheet(*path)
	create := *force || errors.Is(err, os.ErrNotExist)
	if create {
		s, err = setup.Starter(pstackrex.SetupTemplate, catalogs)
	}
	if err != nil {
		return err
	}
	result := setupResult{Path: *path, Sheet: s.Format(), Catalogs: catalogs, Commands: []commandResult{}, Notes: []string{}}
	for _, cat := range catalogs {
		if !cat.Installed {
			continue
		}
		h := string(cat.Harness)
		if cat.Harness == sheet.Pi {
			result.Notes = append(result.Notes, "Pi is discovered for the sheet; Pi plugin installation is deferred until after 1.0.")
			continue
		}
		action := "add"
		if cat.Harness == sheet.Claude {
			action = "install"
		}
		for _, command := range [][]string{{h, "plugin", "marketplace", "add", *source}, {h, "plugin", action, "pstack-rex@pstack-rex"}} {
			result.Commands = append(result.Commands, commandResult{Command: command, DryRun: *dry})
		}
	}
	for _, cat := range catalogs {
		if cat.Harness == sheet.Codex && cat.Installed {
			features, err := c.run("codex", "features", "list")
			enabled := false
			if err == nil {
				for _, line := range strings.Split(features, "\n") {
					fields := strings.Fields(line)
					if len(fields) > 1 && fields[0] == "multi_agent" && fields[len(fields)-1] == "true" {
						enabled = true
					}
				}
			}
			if !enabled {
				result.Notes = append(result.Notes, "Enable Codex native subagents with `codex features enable multi_agent` (or [features] multi_agent = true in ~/.codex/config.toml).")
			}
		}
	}
	if !*dry && !*yes {
		if *j {
			return fmt.Errorf("setup --json requires --yes or --dry-run")
		}
		fmt.Fprintf(c.out, "Shared sheet: %s\n%s", *path, s.Format())
		for _, command := range result.Commands {
			fmt.Fprintln(c.out, displayCommand(command.Command))
		}
		fmt.Fprint(c.out, "Write the sheet and install these plugins? [y/N] ")
		answer, err := bufio.NewReader(c.in).ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if strings.ToLower(strings.TrimSpace(answer)) != "y" && strings.ToLower(strings.TrimSpace(answer)) != "yes" {
			return fmt.Errorf("setup cancelled")
		}
	}
	if create && !*dry {
		err := writeSheet(*path, s, *force)
		if errors.Is(err, os.ErrExist) {
			if s, err = readSheet(*path); err != nil {
				return err
			}
			fmt.Fprintf(c.errOut, "kept the sheet that appeared at %s during setup; rerun with --force to replace it\n", *path)
		} else if err != nil {
			return err
		}
	}
	failed := false
	for i := range result.Commands {
		command := &result.Commands[i]
		if !*dry {
			output, err := c.run(command.Command[0], command.Command[1:]...)
			command.Output = strings.TrimSpace(output)
			if err != nil {
				command.Error = err.Error()
				failed = true
			}
		}
	}
	if *j {
		if err := c.json(result); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(c.out, "Shared sheet: %s\n%s", result.Path, result.Sheet)
		for _, cat := range catalogs {
			fmt.Fprintf(c.out, "%s: installed=%t models=%s %s\n", cat.Harness, cat.Installed, strings.Join(cat.Models, ", "), cat.Note)
		}
		for _, command := range result.Commands {
			fmt.Fprintln(c.out, "$", displayCommand(command.Command))
			switch {
			case command.DryRun:
				fmt.Fprintln(c.out, "  dry-run; not executed")
			case command.Error != "":
				fmt.Fprintln(c.out, "  ERROR:", command.Error)
			default:
				fmt.Fprintln(c.out, "  OK:", command.Output)
			}
		}
		for _, note := range result.Notes {
			fmt.Fprintln(c.out, note)
		}
	}
	if failed {
		return fmt.Errorf("one or more plugin installation commands failed")
	}
	return nil
}
func displayCommand(command []string) string {
	words := make([]string, len(command))
	for i, word := range command {
		if word != "" && !strings.ContainsAny(word, " \t\n'\"$`\\;&|<>()*?![]{}") {
			words[i] = word
		} else {
			words[i] = "'" + strings.ReplaceAll(word, "'", "'\"'\"'") + "'"
		}
	}
	return strings.Join(words, " ")
}

type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func (c cli) doctor(args []string) error {
	f := c.flags("doctor")
	j := f.Bool("json", false, "print JSON")
	path := f.String("sheet", sheet.Path(c.getenv), "shared sheet path")
	if err := f.Parse(args); err != nil {
		return err
	}
	if err := noArgs(f); err != nil {
		return err
	}
	env := resolve.Detect(c.getenv)
	catalogs := discover.Discover(c.run)
	checks := []doctorCheck{{"binary", true, "pstack-rex " + plugin.Version + "; upstream " + strings.TrimSpace(pstackrex.UpstreamRevision)}}
	_, err := readSheet(*path)
	detail := *path
	if err != nil {
		detail = err.Error()
	}
	checks = append(checks, doctorCheck{"sheet", err == nil, detail})
	checks = append(checks, doctorCheck{"environment", env.Parent != "", fmt.Sprintf("parent=%s backend=%s", env.Parent, env.Backend)})
	installed := 0
	for _, cat := range catalogs {
		ok := cat.Installed && len(cat.Models) > 0
		if cat.Installed {
			installed++
		}
		// Missing optional harnesses are informational; failed discovery on an installed one is a problem.
		checks = append(checks, doctorCheck{string(cat.Harness), !cat.Installed || ok, fmt.Sprintf("installed=%t models=%d %s", cat.Installed, len(cat.Models), cat.Note)})
	}
	checks = append(checks, doctorCheck{"harnesses", installed > 0, fmt.Sprintf("%d installed", installed)})
	for _, cmd := range [][]string{{"orca", "--version"}, {"rex", "whoami"}} {
		output, err := c.run(cmd[0], cmd[1:]...)
		required := (cmd[0] == "orca" && env.Backend == resolve.Orca) || (cmd[0] == "rex" && env.Backend == resolve.Rex)
		detail := strings.TrimSpace(output)
		if err != nil {
			detail = err.Error()
		}
		checks = append(checks, doctorCheck{cmd[0], err == nil || !required, detail})
	}
	failed := false
	for _, check := range checks {
		if !check.OK {
			failed = true
		}
	}
	if *j {
		if err := c.json(map[string]any{"ok": !failed, "checks": checks, "environment": env, "catalogs": catalogs}); err != nil {
			return err
		}
	} else {
		for _, check := range checks {
			status := "ok"
			if !check.OK {
				status = "FAIL"
			}
			fmt.Fprintf(c.out, "%s %s: %s\n", status, check.Name, check.Detail)
		}
	}
	if failed {
		return fmt.Errorf("doctor found problems")
	}
	return nil
}
func checkoutRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "upstream", "REVISION")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "internal", "plugin", "adaptations.json")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("plugin build must run inside the pstack-rex checkout")
		}
		dir = parent
	}
}

func (c cli) codexConfig() string {
	home := c.getenv("CODEX_HOME")
	if home == "" {
		home = filepath.Join(c.getenv("HOME"), ".codex")
	}
	data, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		return ""
	}
	return string(data)
}

func (c cli) rex(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pstack-rex rex <start|report|wait>")
	}
	byteRunner := func(name string, a ...string) ([]byte, error) {
		out, err := c.run(name, a...)
		return []byte(out), err
	}
	root := rexworker.StateRoot(c.getenv)
	switch args[0] {
	case "start":
		f := c.flags("rex start")
		h := f.String("harness", "", "worker harness: claude, codex or pi")
		m := f.String("model", "", "worker model")
		e := f.String("effort", "", "reasoning effort")
		own := f.Bool("own-worktree", false, "run the worker in its own git worktree")
		brief := f.String("brief", "", "file holding the complete brief")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		if err := noArgs(f); err != nil {
			return err
		}
		if *h == "" || *m == "" || *brief == "" {
			return fmt.Errorf("rex start needs --harness, --model and --brief")
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		launch, err := rexworker.Start(byteRunner, root, c.getenv, rexworker.Spec{Harness: *h, Model: *m, Effort: *e, OwnWorktree: *own, BriefPath: *brief, Cwd: cwd})
		if err != nil {
			return err
		}
		return c.json(launch)
	case "report":
		f := c.flags("rex report")
		id := f.String("id", "", "worker id")
		file := f.String("file", "-", "result file, or - for stdin")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		if err := noArgs(f); err != nil {
			return err
		}
		var data []byte
		var err error
		if *file == "-" {
			data, err = io.ReadAll(c.in)
		} else {
			data, err = os.ReadFile(*file)
		}
		if err != nil {
			return err
		}
		return rexworker.Report(root, *id, data)
	case "wait":
		f := c.flags("rex wait")
		id := f.String("id", "", "worker id")
		seconds := f.Int("seconds", 540, "longest wait before returning state running")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		if err := noArgs(f); err != nil {
			return err
		}
		status, err := rexworker.Wait(byteRunner, root, *id, time.Duration(*seconds)*time.Second)
		if err != nil {
			return err
		}
		return c.json(status)
	default:
		return fmt.Errorf("unknown rex command %q", args[0])
	}
}
