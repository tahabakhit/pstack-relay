package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tahabakhit/pstack-rex/internal/bundle"
	"github.com/tahabakhit/pstack-rex/internal/config"
	"github.com/tahabakhit/pstack-rex/internal/engine"
	"github.com/tahabakhit/pstack-rex/internal/mcp"
)

const version = "0.1.0"

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

type targets []string

func (t *targets) String() string         { return strings.Join(*t, ",") }
func (t *targets) Set(value string) error { *t = append(*t, value); return nil }

func descriptor(text string) (config.Target, error) {
	harness, model, ok := strings.Cut(text, ":")
	if !ok {
		return config.Target{}, fmt.Errorf("target must be harness:model@effort")
	}
	effort := ""
	if at := strings.LastIndex(model, "@"); at >= 0 {
		effort = model[at+1:]
		model = model[:at]
	}
	target := config.Target{Harness: harness, Model: model, Effort: effort, Backend: "rex", Auth: "existing"}
	return target, config.ValidateTarget(target, true)
}

func saveConfig(path string, c config.Config, exclusive bool) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if exclusive {
		flags = os.O_CREATE | os.O_WRONLY | os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	global := flag.NewFlagSet("pstack-rex", flag.ContinueOnError)
	configPath := global.String("config", "", "Explicit config path")
	stateRoot := global.String("state-root", engine.DefaultRoot(), "Private state directory on Rex server host")
	projectConfig := global.String("project-config", "", "Explicit opt-in role override file")
	if err := global.Parse(args); err != nil {
		return err
	}
	args = global.Args()
	if len(args) == 0 {
		return fmt.Errorf("usage: pstack-rex [--config PATH] [--state-root PATH] setup|doctor|models|dispatch|inspect|list|wait|cancel|mcp|install|version")
	}
	command := args[0]
	args = args[1:]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	id := flags.String("id", "", "Stable worker operation key")
	role := flags.String("role", "", "pstack role")
	seat := flags.Int("seat", 0, "Role seat index")
	cwd := flags.String("cwd", ".", "Git checkout on local Rex server host")
	promptFile := flags.String("prompt-file", "", "Task brief file")
	access := flags.String("access", "read-only", "read-only or isolated-write")
	seconds := flags.Int("seconds", 30, "Wait duration, at most 60 seconds")
	timeout := flags.Int("timeout", 0, "Worker deadline in seconds")
	requireMCP := flags.Bool("require-mcp", false, "Require worker MCP access")
	requireShell := flags.Bool("require-shell", false, "Require worker shell execution")
	revision := flags.String("revision", "", "Required exact Git SHA")
	dest := flags.String("dest", "", "Empty installation destination")
	setRole := flags.String("set-role", "", "Replace configured role with target list")
	var selections targets
	flags.Var(&selections, "target", "harness:model@effort, repeat for panels")
	probe := flags.Bool("probe", false, "Run one real configured worker through Rex; may spend credits")
	discover := flags.Bool("discover", false, "Display installed harness model catalogs without inference")
	if command == "worker" {
		flags.StringVar(stateRoot, "state-root", *stateRoot, "Worker state root")
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments %v", flags.Args())
	}
	switch command {
	case "version":
		return printJSON(map[string]any{"version": version, "upstream_version": bundle.UpstreamVersion, "upstream_revision": bundle.Revision()})
	case "worker":
		return engine.RunWorker(*stateRoot, *id)
	case "install":
		if *dest == "" {
			return fmt.Errorf("install requires --dest pointing to an empty directory")
		}
		if err := bundle.Install(*dest); err != nil {
			return err
		}
		executable, _ := os.Executable()
		return printJSON(map[string]any{"installed": *dest, "skills": filepath.Join(*dest, "skills"), "mcp_registration_argv": []string{"codex", "mcp", "add", "pstack-rex", "--", executable, "mcp"}, "native_delegation": "disable multi_agent and multi_agent_v2 in a dedicated parent session for exclusive routing", "conflicts": "disable other pstack installations before registering these skills"})
	case "setup":
		path := *configPath
		if path == "" {
			path = config.DefaultPath()
		}
		c := config.Default()
		if *role != "" {
			for _, text := range selections {
				target, err := descriptor(text)
				if err != nil {
					return err
				}
				c.Roles[*role] = append(c.Roles[*role], target)
			}
		} else if len(selections) > 0 {
			return fmt.Errorf("--target requires --role")
		}
		if err := saveConfig(path, c, true); err != nil {
			return err
		}
		return printJSON(map[string]any{"config": path, "roles": c.Roles, "next": "pstack-rex models --set-role 'how explorer' --target HARNESS:MODEL@EFFORT; then doctor --probe --role 'how explorer'", "funding": "existing harness authentication; API versus subscription funding is not verified"})
	case "upstream":
		return bundle.Check()
	}
	c, err := config.Load(*configPath, "")
	if err != nil {
		return fmt.Errorf("load config: %w; run pstack-rex setup", err)
	}
	if *projectConfig != "" {
		c, err = config.LoadProject(c, *projectConfig)
		if err != nil {
			return err
		}
	}
	service := &engine.Service{Root: *stateRoot, Config: c}
	ctx := context.Background()
	switch command {
	case "models":
		if *setRole != "" {
			if *projectConfig != "" {
				return fmt.Errorf("edit project overrides explicitly; models writes user config only")
			}
			c.Roles[*setRole] = nil
			for _, text := range selections {
				target, e := descriptor(text)
				if e != nil {
					return e
				}
				c.Roles[*setRole] = append(c.Roles[*setRole], target)
			}
			path := *configPath
			if path == "" {
				path = config.DefaultPath()
			}
			if err = saveConfig(path, c, false); err != nil {
				return err
			}
		}
		if *discover {
			return discoverModels()
		}
		return printJSON(map[string]any{"roles": c.Roles, "role_vocabulary": config.Roles, "capabilities": mcp.Capabilities()})
	case "doctor":
		if *probe {
			key := fmt.Sprintf("probe-%d", time.Now().UnixNano())
			record, e := service.Dispatch(ctx, engine.Request{ID: key, Role: *role, Seat: *seat, Cwd: *cwd, Prompt: "Reply with PSTACK_REX_READY. Do not use tools or access files.", Access: "read-only"})
			if e != nil {
				return e
			}
			record, e = service.Wait(ctx, record.ID, 60*time.Second)
			if e != nil {
				return e
			}
			if err = printJSON(record); err != nil {
				return err
			}
			if record.Status != "completed" {
				return fmt.Errorf("probe did not complete successfully; inspect --id %s", key)
			}
			return nil
		}
		return doctor(c)
	case "mcp":
		return mcp.Serve(os.Stdin, os.Stdout, service)
	case "dispatch":
		if *promptFile == "" {
			return fmt.Errorf("dispatch requires --prompt-file")
		}
		data, e := os.ReadFile(*promptFile)
		if e != nil {
			return e
		}
		record, e := service.Dispatch(ctx, engine.Request{ID: *id, Role: *role, Seat: *seat, Cwd: *cwd, Prompt: string(data), Access: *access, TimeoutSeconds: *timeout, RequireMCP: *requireMCP, RequireShell: *requireShell, ExpectedRevision: *revision})
		if e != nil {
			return e
		}
		return printJSON(record)
	case "inspect":
		record, e := service.Inspect(ctx, *id)
		if e != nil {
			return e
		}
		return printJSON(record)
	case "list":
		records, e := service.List(ctx)
		if e != nil {
			return e
		}
		return printJSON(records)
	case "wait":
		record, e := service.Wait(ctx, *id, time.Duration(*seconds)*time.Second)
		if e != nil {
			return e
		}
		if err = printJSON(record); err != nil {
			return err
		}
		if record.Status != "completed" {
			return fmt.Errorf("worker %s has status %s; no successful completion", record.ID, record.Status)
		}
		return nil
	case "cancel":
		record, e := service.Cancel(ctx, *id)
		if e != nil {
			return e
		}
		return printJSON(record)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func probeCommand(exe string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	data, err := cmd.CombinedOutput()
	return string(data), err
}

func doctor(c config.Config) error {
	checks := map[string]any{}
	problems := []string{}
	for _, name := range []string{"rex", "git", "codex", "claude", "pi"} {
		path, err := exec.LookPath(name)
		if err != nil {
			checks[name] = map[string]any{"installed": false}
			if name == "rex" || name == "git" {
				problems = append(problems, name+" missing")
			}
			continue
		}
		checks[name] = map[string]any{"installed": true, "path": path}
	}
	raw, err := probeCommand("rex", "--autostart=false", "whoami", "--json")
	if err != nil {
		checks["rex_connection"] = map[string]any{"connected": false}
		problems = append(problems, "Rex unreachable")
	} else {
		var who struct {
			Version  string `json:"rex_version"`
			Revision int    `json:"api_revision"`
			Endpoint string `json:"endpoint"`
		}
		if err = json.Unmarshal([]byte(raw), &who); err != nil {
			return err
		}
		local := strings.HasPrefix(who.Endpoint, "unix://")
		checks["rex_connection"] = map[string]any{"connected": true, "local": local, "version": who.Version, "api_revision": who.Revision}
		if !local || who.Revision < 12 {
			problems = append(problems, "require local Rex Unix socket with API revision >=12")
		}
	}
	auth := map[string]any{}
	used := map[string]bool{}
	for _, seats := range c.Roles {
		for _, target := range seats {
			key := target.Harness + ":" + target.Model
			if used[key] {
				continue
			}
			used[key] = true
			var args []string
			switch target.Harness {
			case "codex":
				args = []string{"login", "status"}
			case "claude":
				args = []string{"auth", "status", "--json"}
			case "pi":
				args = []string{"auth", "check", "--model", target.Model, "--json", "--no-refresh"}
			default:
				auth[key] = "unresolved inheritance"
				continue
			}
			_, e := probeCommand(target.Harness, args...)
			auth[key] = map[string]any{"status_command_passed": e == nil, "model_execution_verified": false, "funding": "unknown"}
			if e != nil {
				problems = append(problems, key+" auth/prerequisite check failed")
			}
		}
	}
	if err = printJSON(map[string]any{"checks": checks, "authentication": auth, "problems": problems, "capabilities": mcp.Capabilities(), "next": "Use doctor --probe --role ROLE for a real inference test through Rex. It may spend credits."}); err != nil {
		return err
	}
	if len(problems) > 0 {
		return errors.New("doctor found unavailable prerequisites")
	}
	return nil
}

func discoverModels() error {
	catalogs := map[string]any{}
	for name, args := range map[string][]string{"codex": {"debug", "models", "--bundled"}, "pi": {"--offline", "--list-models"}} {
		text, err := probeCommand(name, args...)
		if err != nil {
			catalogs[name] = map[string]any{"available": false, "reason": "catalog command unavailable"}
		} else {
			catalogs[name] = map[string]any{"catalog": text, "execution_verified": false}
		}
	}
	catalogs["claude"] = map[string]any{"available": false, "reason": "no installed CLI model-list command; supply explicit model or documented alias and run doctor --probe"}
	return printJSON(catalogs)
}
