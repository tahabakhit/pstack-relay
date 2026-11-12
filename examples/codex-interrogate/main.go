package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pstack-rex/pstack-rex/internal/config"
)

const role = "interrogate reviewers"
const baseSource = "def sum_values(values):\n    return sum(values)\n"
const reviewSource = "def sum_values(values):\n    return sum(values[:-1])\n"
const probeSource = `import json
from calculator import sum_values
cases = [[], [7], [2, 3, 5]]
results = [{"input": values, "actual": sum_values(values), "expected": sum(values)} for values in cases]
print(json.dumps(results, sort_keys=True, separators=(",", ":")))
raise SystemExit(int(any(row["actual"] != row["expected"] for row in results)))
`
const baseOutput = "[{\"actual\":0,\"expected\":0,\"input\":[]},{\"actual\":7,\"expected\":7,\"input\":[7]},{\"actual\":10,\"expected\":10,\"input\":[2,3,5]}]\n"
const reviewOutput = "[{\"actual\":0,\"expected\":0,\"input\":[]},{\"actual\":0,\"expected\":7,\"input\":[7]},{\"actual\":5,\"expected\":10,\"input\":[2,3,5]}]\n"
const probeCommand = "python3 -B probe.py"

var installedReads = []string{"runtime-contract.md", "skills/interrogate/SKILL.md", "skills/arena/SKILL.md"}

type options struct{ Binary, Config, ParentModel, ParentEffort, Out string }
type seat struct {
	Index   int
	Target  config.Target
	Cwd, ID string
}
type manifest struct {
	Version                      int
	BaseRevision, ReviewRevision string
	Binary                       string
	ParentModel, ParentEffort    string
	Seats                        []seat
	InstalledHashes              map[string]string
}
type parentExit struct {
	ExitCode int
	Error    string `json:",omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var o options
	flag.StringVar(&o.Binary, "binary", "", "Built pstack-rex executable")
	flag.StringVar(&o.Config, "config", "", "Explicit central configuration")
	flag.StringVar(&o.ParentModel, "parent-model", "", "Explicit Codex parent model")
	flag.StringVar(&o.ParentEffort, "parent-effort", "medium", "Requested parent effort")
	flag.StringVar(&o.Out, "out", "", "New external private artifact directory")
	replay := flag.String("verify-only", "", "Verify retained artifacts without inference")
	flag.Parse()
	if *replay != "" {
		return report(*replay)
	}
	if o.Binary == "" || o.Config == "" || o.ParentModel == "" || o.Out == "" {
		return fmt.Errorf("required: --binary PATH --config PATH --parent-model MODEL --out NEWDIR; makes real model calls")
	}
	var err error
	for _, path := range []*string{&o.Binary, &o.Config, &o.Out} {
		*path, err = filepath.Abs(*path)
		if err != nil {
			return err
		}
	}
	o.Out = filepath.Clean(o.Out)
	cfg, err := config.Load(o.Config, "")
	if err != nil {
		return err
	}
	if _, err = targets(cfg); err != nil {
		return err
	}
	// Artifacts must stay outside the caller's repository, including ignored paths.
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if root, e := git(cwd, "rev-parse", "--show-toplevel"); e == nil {
		physicalParent, e := filepath.EvalSymlinks(filepath.Dir(o.Out))
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(strings.TrimSpace(root), filepath.Join(physicalParent, filepath.Base(o.Out)))
		if e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return fmt.Errorf("--out must be outside the repository")
		}
	}
	if err = os.Mkdir(o.Out, 0700); err != nil {
		return err
	}
	o.Out, err = filepath.EvalSymlinks(o.Out)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Private artifacts:", o.Out)
	m, err := prepare(o, cfg)
	if err != nil {
		return err
	}
	if err = launch(o, m); err != nil {
		return err
	}
	return report(o.Out)
}
func targets(cfg config.Config) ([]config.Target, error) {
	ts := cfg.Roles[role]
	models := map[string]bool{}
	for i, t := range ts {
		resolved, err := cfg.Resolve(role, i, nil)
		if err != nil {
			return nil, err
		}
		if resolved != t || t.Harness != "codex" || t.Backend != "rex" || t.Effort == "" || t.Effort == "default" {
			return nil, fmt.Errorf("seat %d requires explicit Codex, Rex and concrete effort", i)
		}
		models[t.Model] = true
	}
	if len(models) < 2 {
		return nil, fmt.Errorf("at least two distinct explicit requested Codex models required")
	}
	return ts, nil
}
func prepare(o options, cfg config.Config) (manifest, error) {
	m := manifest{Version: 1, Binary: o.Binary, ParentModel: o.ParentModel, ParentEffort: o.ParentEffort, InstalledHashes: map[string]string{}}
	if err := writeJSON(filepath.Join(o.Out, "config.json"), cfg); err != nil {
		return m, err
	}
	repo := filepath.Join(o.Out, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		return m, err
	}
	if _, err := git(repo, "init", "-q", "--object-format=sha1", "--template="); err != nil {
		return m, err
	}
	for name, data := range map[string]string{"calculator.py": baseSource, "probe.py": probeSource, ".gitignore": ".agents/\n"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(data), 0600); err != nil {
			return m, err
		}
	}
	if _, err := git(repo, "add", "."); err != nil {
		return m, err
	}
	commit := func(message string) (string, error) {
		if _, err := git(repo, "-c", "user.name=pstack-rex example", "-c", "user.email=example@localhost", "-c", "commit.gpgsign=false", "commit", "-qm", message); err != nil {
			return "", err
		}
		rev, err := git(repo, "rev-parse", "HEAD")
		return strings.TrimSpace(rev), err
	}
	var err error
	m.BaseRevision, err = commit("Add correct sum and fixed probe")
	if err != nil {
		return m, err
	}
	if err = probe(repo, baseOutput, 0); err != nil {
		return m, err
	}
	if err = os.WriteFile(filepath.Join(repo, "calculator.py"), []byte(reviewSource), 0600); err != nil {
		return m, err
	}
	if _, err = git(repo, "add", "calculator.py"); err != nil {
		return m, err
	}
	m.ReviewRevision, err = commit("Accidentally omit final input")
	if err != nil {
		return m, err
	}
	if err = probe(repo, reviewOutput, 1); err != nil {
		return m, err
	}
	install := exec.Command(o.Binary, "install", "--dest", filepath.Join(repo, ".agents"))
	if data, e := install.CombinedOutput(); e != nil {
		return m, fmt.Errorf("install: %w: %s", e, data)
	}
	for _, name := range installedReads {
		data, e := os.ReadFile(filepath.Join(repo, ".agents", name))
		if e != nil {
			return m, e
		}
		m.InstalledHashes[name] = digest(data)
	}
	for i, t := range cfg.Roles[role] {
		s := seat{Index: i, Target: t, Cwd: filepath.Join(o.Out, fmt.Sprintf("seat-%d", i)), ID: fmt.Sprintf("interrogate-seat-%d", i)}
		if _, err = git(repo, "worktree", "add", "--detach", s.Cwd, m.ReviewRevision); err != nil {
			return m, err
		}
		m.Seats = append(m.Seats, s)
	}
	if err = writeJSON(filepath.Join(o.Out, "manifest.json"), m); err != nil {
		return m, err
	}
	return m, nil
}
func launch(o options, m manifest) error {
	seats, _ := json.MarshalIndent(m.Seats, "", "  ")
	prompt := fmt.Sprintf(`Run the installed pstack interrogate workflow. Intent: sum_values must sum ALL input values, including the last. Scope is committed %s..%s. Do not fix code. Read each installed file using a separate exact command 'cat ABSOLUTE_PATH': %s. Read any templates those reference from this installation only.
The fixture is intentionally tiny. All reviewer worktrees are already prepared detached at review HEAD. Adopt them as arena output locations; there are no uncommitted changes to copy. Read-only permissions cover parent and workers. Use the committed stdlib probe as the reproduction, without writing notes/files. Do not launch native agents or coding harnesses. Delegate ONLY through pstack-rex MCP.
Call capabilities before dispatch. Dispatch EVERY ordered configured seat below exactly once. Use its exact ID, index and cwd, role "interrogate reviewers", access "read-only", require_shell true, expected_revision %s. Include the installed upstream adversary brief and relevant complete instructions. Every reviewer must execute exactly 'python3 -B ABSOLUTE_SEAT_CWD/probe.py', replacing ABSOLUTE_SEAT_CWD with its supplied absolute cwd, preserve the failing exit/output and report only findings reproduced by commands. Use each configured model without substitutions or retries. Wait for terminal completion of ALL seats. Inspect/list as needed.
%s
After all terminal waits, independently execute exactly 'python3 -B %s' (the absolute parent probe path). Preserve the nonzero result, which proves a defect rather than worker failure. Merge findings by root cause. Keep normal Intent/Findings/Dropped output. This fixture proof permits exactly one surviving root cause (last element omitted); any additional unverified claims must be dropped, never added to Findings.
End your final message with a single fenced JSON block labelled pstack-rex-evidence containing exactly this shape, with every reporting zero-based seat in order: {"findings":[{"root_cause":"last-element-omitted","command":"python3 -B probe.py","output":%s,"required_output":%s,"contributors":[0,1]}]}. Expand contributors to the complete actual configured seat list. No extra findings. Output strings must preserve the trailing newline. This structured block supplements the upstream report and binds its sole surviving finding to actual command evidence. Do not claim provider-served model identity.`, m.BaseRevision, m.ReviewRevision, readPaths(o.Out), m.ReviewRevision, seats, filepath.Join(o.Out, "repo", "probe.py"), strconv.Quote(reviewOutput), strconv.Quote(baseOutput))
	if err := os.WriteFile(filepath.Join(o.Out, "parent-prompt.txt"), []byte(prompt), 0600); err != nil {
		return err
	}
	args := parentArgs(o.Binary, o.Out, m)
	if err := writeJSON(filepath.Join(o.Out, "parent-argv.json"), args); err != nil {
		return err
	}
	transcript, err := os.OpenFile(filepath.Join(o.Out, "parent.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer transcript.Close()
	stderr, err := os.OpenFile(filepath.Join(o.Out, "parent.stderr"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stderr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = transcript
	cmd.Stderr = stderr
	err = cmd.Run()
	exit := parentExit{ExitCode: -1}
	if cmd.ProcessState != nil {
		exit.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		exit.Error = err.Error()
	}
	if e := writeJSON(filepath.Join(o.Out, "parent-exit.json"), exit); e != nil {
		return e
	}
	if err != nil {
		return fmt.Errorf("parent failed; retained artifacts: %w", err)
	}
	return nil
}
func parentArgs(binary, root string, m manifest) []string {
	quote := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	server := fmt.Sprintf("mcp_servers.pstack-rex={command=%s,args=%s,startup_timeout_sec=10,tool_timeout_sec=90,default_tools_approval_mode=\"approve\"}", quote(binary), quote([]string{"--config", filepath.Join(root, "config.json"), "--state-root", filepath.Join(root, "state"), "mcp"}))
	return []string{"-a", "never", "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--disable", "multi_agent", "--disable", "multi_agent_v2", "--disable", "plugins", "--disable", "apps", "--disable", "hooks", "--disable", "memories", "--disable", "skill_mcp_dependency_install", "--enable", "skip_host_skill_discovery", "--model", m.ParentModel, "-c", "model_reasoning_effort=" + quote(m.ParentEffort), "-c", "project_doc_max_bytes=0", "-c", "suppress_unstable_features_warning=true", "-c", server, "--json", "--sandbox", "read-only", "--cd", filepath.Join(root, "repo"), "-"}
}
func readPaths(root string) string {
	var paths []string
	for _, name := range installedReads {
		paths = append(paths, filepath.Join(root, "repo", ".agents", name))
	}
	return strings.Join(paths, ", ")
}
func git(cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prefix := []string{"--no-pager", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-C", cwd}
	cmd := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "GIT_") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %v: %w: %s", args, err, data)
	}
	return string(data), nil
}
func probe(cwd, want string, code int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "-B", "probe.py")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	actual := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			actual = exit.ExitCode()
		} else {
			return err
		}
	}
	if actual != code || string(out) != want {
		return fmt.Errorf("fixed probe mismatch in %s: exit %d output %q", cwd, actual, out)
	}
	return nil
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
