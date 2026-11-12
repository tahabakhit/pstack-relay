package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/pstack-rex/pstack-rex/internal/config"
	"github.com/pstack-rex/pstack-rex/internal/engine"
	"github.com/pstack-rex/pstack-rex/internal/harness"
)

type commandRun struct {
	ID, Command, Output string
	ExitCode, Line      int
	StartLine           int
}
type toolCall struct {
	ID, Tool  string
	Arguments json.RawMessage
	Content   string
	Line      int
	StartLine int
}
type transcript struct {
	Commands []commandRun
	Calls    []toolCall
	Final    string
	Hash     string
}
type finding struct {
	RootCause      string `json:"root_cause"`
	Command        string `json:"command"`
	Output         string `json:"output"`
	RequiredOutput string `json:"required_output"`
	Contributors   []int  `json:"contributors"`
}
type finalEvidence struct {
	Findings []finding `json:"findings"`
}
type seatEvidence struct {
	Seat           int    `json:"seat"`
	RequestedModel string `json:"requested_model"`
	ModelEvidence  string `json:"model_evidence"`
	ServedModel    string `json:"served_model"`
	Status         string `json:"status"`
	TraceSHA256    string `json:"trace_sha256"`
	RequestSHA256  string `json:"request_sha256"`
	ResultSHA256   string `json:"result_sha256"`
}
type evidence struct {
	Status            string         `json:"status"`
	Scope             string         `json:"scope"`
	BaseRevision      string         `json:"base_revision"`
	ReviewRevision    string         `json:"review_revision"`
	ProbeSHA256       string         `json:"probe_sha256"`
	ParentTraceSHA256 string         `json:"parent_trace_sha256"`
	Seats             []seatEvidence `json:"seats"`
	Finding           finding        `json:"finding"`
}

func report(root string) error {
	e, err := verify(root)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return decode(data, value)
}
func decode(data []byte, value any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}
func readTrace(path string, target config.Target) (transcript, error) {
	var t transcript
	data, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	outcome, err := harness.Parse(target, data)
	if err != nil {
		return t, err
	}
	t.Final = outcome.Text
	t.Hash = digest(data)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 32<<20)
	type executionStart struct {
		Type string
		Line int
	}
	starts := map[string]executionStart{}
	completed := map[string]bool{}
	line := 0
	for scanner.Scan() {
		line++
		var event struct {
			Type string `json:"type"`
			Item struct {
				ID        string          `json:"id"`
				Type      string          `json:"type"`
				Command   string          `json:"command"`
				Output    string          `json:"aggregated_output"`
				Exit      *int            `json:"exit_code"`
				Status    string          `json:"status"`
				Server    string          `json:"server"`
				Tool      string          `json:"tool"`
				Arguments json.RawMessage `json:"arguments"`
				Error     json.RawMessage `json:"error"`
				Result    struct {
					IsError bool `json:"isError"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
			} `json:"item"`
		}
		if err = json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return t, err
		}
		item := event.Item
		switch event.Type {
		case "thread.started", "turn.started", "turn.completed":
			continue
		case "item.started", "item.completed":
		default:
			return t, fmt.Errorf("unsupported transcript event %q", event.Type)
		}
		switch item.Type {
		case "agent_message":
			continue
		case "command_execution", "mcp_tool_call":
		default:
			return t, fmt.Errorf("unsupported transcript item %q", item.Type)
		}
		if item.ID == "" {
			return t, fmt.Errorf("missing execution item id")
		}
		if event.Type == "item.started" {
			if _, ok := starts[item.ID]; ok {
				return t, fmt.Errorf("duplicate execution item")
			}
			starts[item.ID] = executionStart{item.Type, line}
			continue
		}
		if completed[item.ID] || starts[item.ID].Type != item.Type {
			return t, fmt.Errorf("unpaired execution item %s", item.ID)
		}
		completed[item.ID] = true
		if item.Status != "completed" {
			return t, fmt.Errorf("execution item did not complete")
		}
		if item.Type == "command_execution" {
			if item.Exit == nil || item.Command == "" {
				return t, fmt.Errorf("incomplete command event")
			}
			t.Commands = append(t.Commands, commandRun{item.ID, item.Command, item.Output, *item.Exit, line, starts[item.ID].Line})
		} else {
			if item.Server != "pstack-rex" || item.Result.IsError || (len(item.Error) > 0 && string(item.Error) != "null") || len(item.Result.Content) != 1 || item.Result.Content[0].Type != "text" {
				return t, fmt.Errorf("unsupported or failed MCP call")
			}
			t.Calls = append(t.Calls, toolCall{item.ID, item.Tool, item.Arguments, item.Result.Content[0].Text, line, starts[item.ID].Line})
		}
	}
	for id := range starts {
		if !completed[id] {
			return t, fmt.Errorf("unfinished execution item %s", id)
		}
	}
	return t, scanner.Err()
}

// Codex currently reports these exact shell wrappers. No shell is evaluated here.
func isCommand(actual, want string) bool {
	if actual == want {
		return true
	}
	for _, shell := range []string{"/usr/bin/zsh", "/bin/zsh", "/usr/bin/bash", "/bin/bash", "/bin/sh"} {
		if actual == shell+" -lc "+shellQuote(want) || (!strings.ContainsAny(want, "$`\\\"!") && actual == shell+" -lc "+strconv.Quote(want)) {
			return true
		}
	}
	return false
}
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
func exactProbe(t transcript, cwd string, after int) (commandRun, error) {
	for _, c := range t.Commands {
		if c.StartLine > after && (isCommand(c.Command, "python3 -B "+filepath.Join(cwd, "probe.py")) || isCommand(c.Command, "python3 -B "+shellQuote(filepath.Join(cwd, "probe.py")))) && c.Output == reviewOutput && c.ExitCode == 1 {
			return c, nil
		}
	}
	return commandRun{}, fmt.Errorf("missing exact executed probe after line %d", after)
}
func verify(root string) (e evidence, err error) {
	root, err = filepath.Abs(root)
	if err != nil {
		return e, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return e, err
	}
	var m manifest
	if err = readJSON(filepath.Join(root, "manifest.json"), &m); err != nil {
		return e, err
	}
	if m.Version != 1 || m.BaseRevision == m.ReviewRevision || m.ParentModel == "" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(m.BaseRevision) || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(m.ReviewRevision) {
		return e, fmt.Errorf("invalid manifest")
	}
	cfg, err := config.Load(filepath.Join(root, "config.json"), "")
	if err != nil {
		return e, err
	}
	ts, err := targets(cfg)
	if err != nil {
		return e, err
	}
	if len(ts) != len(m.Seats) {
		return e, fmt.Errorf("manifest omitted configured seats")
	}
	for i, s := range m.Seats {
		if s.Index != i || s.Target != ts[i] || s.ID != fmt.Sprintf("interrogate-seat-%d", i) || s.Cwd != filepath.Join(root, fmt.Sprintf("seat-%d", i)) {
			return e, fmt.Errorf("manifest seat mismatch")
		}
	}
	if err = verifyFixture(root, m); err != nil {
		return e, err
	}
	var exit parentExit
	if err = readJSON(filepath.Join(root, "parent-exit.json"), &exit); err != nil {
		return e, err
	}
	if exit.ExitCode != 0 || exit.Error != "" {
		return e, fmt.Errorf("parent process failed")
	}
	var argv []string
	if err = readJSON(filepath.Join(root, "parent-argv.json"), &argv); err != nil {
		return e, err
	}
	if err = verifyParentArgs(argv, root, m); err != nil {
		return e, err
	}
	parent, err := readTrace(filepath.Join(root, "parent.jsonl"), config.Target{Harness: "codex", Model: m.ParentModel, Effort: m.ParentEffort, Backend: "rex", Auth: "existing"})
	if err != nil {
		return e, err
	}
	readsBefore := 0
	for _, name := range installedReads {
		path := filepath.Join(root, "repo", ".agents", name)
		data, er := os.ReadFile(path)
		if er != nil {
			return e, er
		}
		if digest(data) != m.InstalledHashes[name] {
			return e, fmt.Errorf("installed file changed: %s", name)
		}
		found := false
		for _, c := range parent.Commands {
			if (isCommand(c.Command, "cat "+path) || isCommand(c.Command, "cat "+shellQuote(path))) && c.ExitCode == 0 && c.Output == string(data) {
				found = true
				if c.Line > readsBefore {
					readsBefore = c.Line
				}
				break
			}
		}
		if !found {
			return e, fmt.Errorf("parent did not demonstrably read installed %s", name)
		}
	}
	if len(m.InstalledHashes) != len(installedReads) {
		return e, fmt.Errorf("installed hash manifest mismatch")
	}
	entries, err := os.ReadDir(filepath.Join(root, "state", "workers"))
	if err != nil {
		return e, err
	}
	if len(entries) != len(m.Seats) {
		return e, fmt.Errorf("missing or extra durable attempts")
	}
	known := map[string]bool{}
	for _, s := range m.Seats {
		known[s.ID] = true
	}
	for _, entry := range entries {
		if !entry.IsDir() || !known[entry.Name()] {
			return e, fmt.Errorf("unexpected attempt")
		}
	}
	dispatch := map[string]engine.Request{}
	dispatchReceipt := map[string]engine.Record{}
	waits := map[string]engine.Record{}
	waitLine := map[string]int{}
	capabilityLine := 0
	for _, call := range parent.Calls {
		switch call.Tool {
		case "capabilities":
			var caps struct {
				Shell    []string `json:"shell_harnesses"`
				Backends []string `json:"backends"`
			}
			if json.Unmarshal([]byte(call.Content), &caps) != nil || !reflect.DeepEqual(caps.Shell, []string{"codex"}) || !reflect.DeepEqual(caps.Backends, []string{"rex"}) {
				return e, fmt.Errorf("missing shell/Rex capability")
			}
			capabilityLine = call.Line
		case "dispatch":
			var req engine.Request
			if err = decodeDispatch(call.Arguments, &req); err != nil {
				return e, err
			}
			if capabilityLine == 0 || call.StartLine <= capabilityLine || call.StartLine <= readsBefore || !known[req.ID] {
				return e, fmt.Errorf("dispatch preceded required reads/capabilities or unknown seat")
			}
			if _, exists := dispatch[req.ID]; exists {
				return e, fmt.Errorf("duplicate dispatch")
			}
			dispatch[req.ID] = req
			var rec engine.Record
			if err = decode([]byte(call.Content), &rec); err != nil {
				return e, err
			}
			dispatchReceipt[req.ID] = rec
		case "wait", "inspect":
			var args struct {
				ID      string `json:"id"`
				Seconds int    `json:"seconds,omitempty"`
			}
			if err = decode(call.Arguments, &args); err != nil {
				return e, err
			}
			if !known[args.ID] || dispatch[args.ID].ID == "" {
				return e, fmt.Errorf("wait/inspect without expected dispatch")
			}
			var rec engine.Record
			if err = decode([]byte(call.Content), &rec); err != nil {
				return e, err
			}
			if rec.ID != args.ID {
				return e, fmt.Errorf("wait/inspect identity mismatch")
			}
			if call.Tool == "wait" && rec.Status == "completed" {
				waits[args.ID] = rec
				waitLine[args.ID] = call.Line
			}
		case "list":
			var records []engine.Record
			if err = decode([]byte(call.Content), &records); err != nil {
				return e, err
			}
			for _, r := range records {
				if !known[r.ID] {
					return e, fmt.Errorf("unexpected listed attempt")
				}
			}
		default:
			return e, fmt.Errorf("unexpected MCP action %s", call.Tool)
		}
	}
	latestWait := 0
	for _, s := range m.Seats {
		dir := filepath.Join(root, "state", "workers", s.ID)
		var req engine.Request
		var started engine.Started
		var result engine.Result
		var launch engine.Launch
		for name, value := range map[string]any{"request.json": &req, "started.json": &started, "result.json": &result, "launch.json": &launch} {
			if err = readJSON(filepath.Join(dir, name), value); err != nil {
				return e, err
			}
		}
		if req.ID != s.ID || req.Role != role || req.Seat != s.Index || req.Target != s.Target || req.Cwd != s.Cwd || req.Access != "read-only" || !req.RequireShell || req.RequireMCP || req.ExpectedRevision != m.ReviewRevision || req.Prompt == "" || req.Executable == "" {
			return e, fmt.Errorf("durable request mismatch seat %d", s.Index)
		}
		observed, ok := dispatch[s.ID]
		if !ok || observed.ID != req.ID || observed.Role != req.Role || observed.Seat != req.Seat || observed.Cwd != req.Cwd || observed.Prompt != req.Prompt || observed.Access != req.Access || observed.RequireShell != req.RequireShell || observed.RequireMCP != req.RequireMCP || observed.ExpectedRevision != req.ExpectedRevision || (observed.TimeoutSeconds != 0 && observed.TimeoutSeconds != req.TimeoutSeconds) {
			return e, fmt.Errorf("MCP/durable request mismatch")
		}
		if started.Cwd != s.Cwd || started.Revision != m.ReviewRevision || started.Enforcement != "sandbox-read-only" || started.PID <= 0 || started.At.IsZero() || started.Session == "" || started.Block == "" || launch.Session != started.Session || launch.Block != started.Block || launch.Error != "" || launch.Uncertain {
			return e, fmt.Errorf("start/launch mismatch")
		}
		snapshot := digest([]byte("head:" + m.ReviewRevision + "\x00diff:0\x00"))
		if result.Status != "completed" || result.ExitCode != 0 || result.Error != "" || result.StartRevision != m.ReviewRevision || result.EndRevision != m.ReviewRevision || result.StartSnapshotSHA256 != snapshot || result.EndSnapshotSHA256 != snapshot || result.DiffSHA256 != digest(nil) || result.Finished.Before(started.At) || result.Outcome == nil {
			return e, fmt.Errorf("worker failed or changed fixture")
		}
		expected := engine.Record{ID: s.ID, Status: "completed", Role: role, Target: s.Target, Artifacts: dir, Launch: &launch, Started: &started, Result: &result}
		if !reflect.DeepEqual(waits[s.ID], expected) {
			return e, fmt.Errorf("terminal wait/durable result mismatch")
		}
		d := dispatchReceipt[s.ID]
		if d.ID != s.ID || d.Role != role || d.Target != s.Target || d.Artifacts != dir || d.Launch == nil || *d.Launch != launch || d.Error != "" || (d.Status != "starting" && d.Status != "running" && d.Status != "completed") {
			return e, fmt.Errorf("dispatch receipt mismatch")
		}
		raw, er := readTrace(filepath.Join(dir, "stdout.jsonl"), s.Target)
		if er != nil {
			return e, er
		}
		rawData, er := os.ReadFile(filepath.Join(dir, "stdout.jsonl"))
		if er != nil {
			return e, er
		}
		outcome, er := harness.Parse(s.Target, rawData)
		if er != nil {
			return e, er
		}
		if !reflect.DeepEqual(*result.Outcome, outcome) || outcome.ModelEvidence != "pinned-argv" || outcome.ReportedModel != "" || len(raw.Calls) > 0 {
			return e, fmt.Errorf("worker raw outcome mismatch")
		}
		if _, err = exactProbe(raw, s.Cwd, 0); err != nil {
			return e, fmt.Errorf("seat %d: %w", s.Index, err)
		}
		requestData, er := os.ReadFile(filepath.Join(dir, "request.json"))
		if er != nil {
			return e, er
		}
		resultData, er := os.ReadFile(filepath.Join(dir, "result.json"))
		if er != nil {
			return e, er
		}
		e.Seats = append(e.Seats, seatEvidence{s.Index, s.Target.Model, "pinned-argv", "unknown", "completed", raw.Hash, digest(requestData), digest(resultData)})
		if waitLine[s.ID] > latestWait {
			latestWait = waitLine[s.ID]
		}
	}
	if _, err = exactProbe(parent, filepath.Join(root, "repo"), latestWait); err != nil {
		return e, fmt.Errorf("parent: %w", err)
	}
	final, err := parseFinal(parent.Final, len(m.Seats))
	if err != nil {
		return e, err
	}
	e.Status = "verified-seeded-interrogate-path"
	e.Scope = "controlled committed fixture; requested model diversity only; harness transcripts are trusted observations, not attestation"
	e.BaseRevision = m.BaseRevision
	e.ReviewRevision = m.ReviewRevision
	e.ProbeSHA256 = digest([]byte(probeSource))
	e.ParentTraceSHA256 = parent.Hash
	e.Finding = final.Findings[0]
	return e, nil
}
func decodeDispatch(data []byte, req *engine.Request) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]bool{"id": true, "role": true, "seat": true, "cwd": true, "prompt": true, "access": true, "timeout_seconds": true, "require_mcp": true, "require_shell": true, "expected_revision": true}
	for name := range fields {
		if !allowed[name] {
			return fmt.Errorf("unsupported dispatch argument %s", name)
		}
	}
	return decode(data, req)
}
func parseFinal(text string, count int) (finalEvidence, error) {
	var f finalEvidence
	marker := "```pstack-rex-evidence\n"
	if strings.Count(text, marker) != 1 {
		return f, fmt.Errorf("missing unique structured finding block")
	}
	_, tail, _ := strings.Cut(text, marker)
	body, _, ok := strings.Cut(tail, "```")
	if !ok {
		return f, fmt.Errorf("unclosed finding block")
	}
	if err := decode([]byte(body), &f); err != nil {
		return f, err
	}
	if len(f.Findings) != 1 {
		return f, fmt.Errorf("unverified extra findings or missing seeded finding")
	}
	want := finding{RootCause: "last-element-omitted", Command: probeCommand, Output: reviewOutput, RequiredOutput: baseOutput}
	for i := 0; i < count; i++ {
		want.Contributors = append(want.Contributors, i)
	}
	if !reflect.DeepEqual(f.Findings[0], want) {
		return f, fmt.Errorf("finding does not bind reproduced defect and all contributors")
	}
	return f, nil
}
func verifyFixture(root string, m manifest) error {
	repo := filepath.Join(root, "repo")
	parent, err := git(repo, "rev-parse", m.ReviewRevision+"^")
	if err != nil || strings.TrimSpace(parent) != m.BaseRevision {
		return fmt.Errorf("fixture commit ancestry changed")
	}
	for rev, source := range map[string]string{m.BaseRevision: baseSource, m.ReviewRevision: reviewSource} {
		for name, want := range map[string]string{"calculator.py": source, "probe.py": probeSource, ".gitignore": ".agents/\n"} {
			actual, e := git(repo, "show", rev+":"+name)
			if e != nil || actual != want {
				return fmt.Errorf("committed fixture mismatch: %s", name)
			}
		}
		tree, e := git(repo, "ls-tree", "--name-only", rev)
		if e != nil || tree != ".gitignore\ncalculator.py\nprobe.py\n" {
			return fmt.Errorf("unexpected fixture file set")
		}
	}
	paths := []string{repo}
	for _, s := range m.Seats {
		paths = append(paths, s.Cwd)
	}
	for _, cwd := range paths {
		top, e := git(cwd, "rev-parse", "--show-toplevel")
		if e != nil || strings.TrimSpace(top) != cwd {
			return fmt.Errorf("fixture worktree path changed")
		}
		if cwd != repo {
			if branch, e := git(cwd, "symbolic-ref", "-q", "HEAD"); e == nil || branch != "" {
				return fmt.Errorf("reviewer worktree is not detached")
			}
			common, e := git(cwd, "rev-parse", "--git-common-dir")
			if e != nil || strings.TrimSpace(common) != filepath.Join(repo, ".git") {
				return fmt.Errorf("reviewer worktree common directory changed")
			}
		}

		head, e := git(cwd, "rev-parse", "HEAD")
		if e != nil || strings.TrimSpace(head) != m.ReviewRevision {
			return fmt.Errorf("checkout revision changed")
		}
		status, e := git(cwd, "status", "--porcelain", "--untracked-files=all")
		if e != nil || status != "" {
			return fmt.Errorf("fixture checkout changed")
		}
		for name, want := range map[string]string{"calculator.py": reviewSource, "probe.py": probeSource, ".gitignore": ".agents/\n"} {
			actual, e := os.ReadFile(filepath.Join(cwd, name))
			if e != nil || string(actual) != want {
				return fmt.Errorf("fixture bytes changed: %s", name)
			}
		}
	}
	oracle, err := os.MkdirTemp("", "pstack-rex-probe-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(oracle)
	if err = os.WriteFile(filepath.Join(oracle, "probe.py"), []byte(probeSource), 0600); err != nil {
		return err
	}
	for _, check := range []struct {
		Source, Output string
		Code           int
	}{{baseSource, baseOutput, 0}, {reviewSource, reviewOutput, 1}} {
		if err = os.WriteFile(filepath.Join(oracle, "calculator.py"), []byte(check.Source), 0600); err != nil {
			return err
		}
		if err = probe(oracle, check.Output, check.Code); err != nil {
			return err
		}
	}
	return nil
}
func verifyParentArgs(args []string, root string, m manifest) error {
	if !filepath.IsAbs(m.Binary) || !reflect.DeepEqual(args, parentArgs(m.Binary, root, m)) {
		return fmt.Errorf("parent launch arguments changed")
	}
	return nil
}
