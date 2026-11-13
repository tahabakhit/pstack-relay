// Package relaytest tests the installed shell artifact using isolated, fake tools.
package relaytest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fixture struct {
	Text      bool
	Name      string
	Args      []string
	Sheet     string
	Missing   bool
	Env       map[string]string
	Config    string
	Template  string
	Stdin     string
	Result    string
	Process   string
	RexLaunch string
	Fake      string
}
type expected struct {
	Error  bool   `json:"error"`
	Output any    `json:"output,omitempty"`
	Calls  any    `json:"calls,omitempty"`
	Sheet  string `json:"sheet,omitempty"`
}

// Fake harnesses are this test executable behind symlinks, never installed agents.
func TestMain(m *testing.M) {
	if os.Getenv("RELAY_FAKE") == "1" {
		fake()
		return
	}
	os.Exit(m.Run())
}
func fake() {
	h := filepath.Base(os.Args[0])
	args := os.Args[1:]
	key := h + " " + strings.Join(args, " ")
	mode := os.Getenv("FAKE_MODE")
	f, e := os.OpenFile(os.Getenv("CALL_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if e != nil {
		panic(e)
	}
	json.NewEncoder(f).Encode(append([]string{h}, args...))
	f.Close()
	if mode == "missing" || mode == "only-claude" && (h == "codex" || h == "pi") || mode == "only-codex" && (h == "claude" || h == "pi") || mode == "only-pi" && (h == "codex" || h == "claude") || mode == "backend-failure" && (h == "orca" || h == "rex" || h == "hc") {
		os.Exit(1)
	}
	if mode == "failed-lists" && (key == "codex debug models --bundled" || key == "pi --offline --list-models") {
		os.Exit(1)
	}
	switch key {
	case "codex --version", "claude --version", "pi --version", "orca --version", "hc --version", "rex whoami":
		fmt.Print("version")
	case "codex debug models --bundled":
		if v, ok := os.LookupEnv("CODEX_MODELS"); ok {
			fmt.Print(v)
		} else {
			fmt.Print(`{"models":[{"slug":"gpt-first"},{"slug":"gpt-next"},{"slug":"c"}]}`)
		}
	case "codex features list":
		if mode == "no-multi-agent" {
			fmt.Print("multi_agent stable false")
		} else {
			fmt.Print("multi_agent stable true")
		}
	case "pi --offline --list-models":
		if v, ok := os.LookupEnv("PI_MODELS"); ok {
			fmt.Print(v)
		} else {
			fmt.Print("provider model context\nopenai gpt-a 400K")
		}
	case "git branch --show-current":
		if mode != "detached" {
			fmt.Print("topic\n")
		}
	default:
		switch {
		case len(args) > 0 && args[0] == "plugin":
			if mode == "install-failure" {
				os.Exit(1)
			}
			fmt.Print("installed")
		case h == "git": // Worktree effects are tested separately against real git.
		case h == "rex" && len(args) > 0 && args[0] == "run":
			fmt.Print(os.Getenv("REX_LAUNCH"))
		case h == "rex" && len(args) > 0 && args[0] == "block":
			if mode == "query-failure" {
				os.Exit(1)
			}
			if mode == "report-during-query" {
				os.WriteFile(filepath.Join(os.Getenv("WAIT_DIR"), "result.md"), []byte("finished"), 0600)
			}
			fmt.Print(os.Getenv("REX_PROCESS"))
		case h == "hc" && len(args) > 0 && args[0] == "run":
			fmt.Print(`{"address":"t7","state":"live"}`)
		case h == "hc" && len(args) > 0 && args[0] == "wait":
			fmt.Print(os.Getenv("HC_WAIT"))
		case h == "hc" && len(args) > 0 && args[0] == "read":
			fmt.Print("final screen\n")
		default:
			fmt.Fprint(os.Stderr, "unexpected command: "+key)
			os.Exit(1)
		}
	}
}
func cases() []fixture {
	var all []fixture
	add := func(name, text string, args ...string) {
		all = append(all, fixture{Name: name, Args: args, Sheet: text})
	}
	parse := []struct{ name, text string }{
		{"group and panel", "# comment\nfeature, refactoring: claude:opus @high, codex:gpt-6.1-sol @xhigh\ndefault effort: session\nsession hook: off\n"},
		{"aliases", "role: inherit-parent, auto, pi:openai/gpt-6.1-terra @medium"},
		{"all efforts", "role: m @low, m @medium, m @high, m @xhigh, m @max, m @session"},
		{"empty", "\n# only comment"}, {"unknown harness", "# hi\nrole: cursor:model"}, {"unknown effort", "role: claude:opus @ultra"}, {"unknown default effort", "default effort: ultra"}, {"invalid hook", "session hook: yes"}, {"empty value", "role:"}, {"missing separator", "role"}, {"empty role", "role, : model"}, {"empty panel seat", "role: opus,"}, {"empty model", "role: codex:"}, {"extra colon", "role: codex:a:b"}, {"effort without at", "role: opus high"}, {"extra fields", "role: opus @high extra"},
		{"duplicate roles", "z: auto\na: opus\nz: codex:gpt"}, {"quote and backslash", `role: codex:a"b\c @high`},
	}
	for _, p := range parse {
		add("parse/"+p.name, p.text, "models", "--json")
	}
	for _, r := range []string{"feature", "refactoring", "feature,refactoring"} {
		add("lookup/"+r, parse[0].text, "resolve", "--role", r, "--parent", "codex", "--backend", "orca")
	}
	resolve := []struct{ name, text string }{
		{"missing role", ""}, {"bare native inherits default", "r: opus\ndefault effort: high"}, {"same harness explicit session", "r: codex:gpt @session\ndefault effort: high"}, {"aliases", "r: inherit-parent, auto @high"}, {"qualified aliases", "r: claude:inherit-parent, pi:auto @high"}, {"orca shared", "r: claude:opus @xhigh"}, {"orca own no effort", "r: codex:gpt\ndefault effort: session"}, {"rex own", "r: claude:opus\ndefault effort: max"}, {"rex pi", "r: pi:openai/gpt"}, {"no backend fallback", "r: claude:opus @high"}, {"pi orca fallback", "r: pi:openai/gpt"}, {"unknown parent", "r: claude:opus @high, codex:gpt"},
	}
	for _, r := range resolve {
		for _, p := range []string{"claude", "codex", "pi", ""} {
			for _, b := range []string{"orca", "rex", "none"} {
				args := []string{"resolve", "--role", "r", "--backend", b}
				if p != "" {
					args = append(args, "--parent", p)
				}
				add("resolve/"+r.name+"/"+p+"/"+b, r.text, args...)
			}
		}
	}
	for _, p := range []string{"claude", "codex"} {
		add("panel reversal/"+p, "interrogate reviewers: claude:opus @xhigh, codex:gpt @high", "resolve", "--role", "interrogate reviewers", "--parent", p, "--backend", "orca")
	}
	for _, b := range []string{"orca", "rex"} {
		for _, alias := range []string{"--own-worktree", "--worktree-own"} {
			add("own/"+b+alias, "r: claude:opus @max", "resolve", "--role", "r", "--parent", "codex", "--backend", b, alias)
		}
	}
	add("resolve/bad backend", "", "resolve", "--role", "r", "--backend", "bad")
	add("resolve/bad parent", "", "resolve", "--role", "r", "--parent", "bad")
	all = append(all, fixture{Name: "resolve/detached own", Sheet: "r: claude:opus", Args: []string{"resolve", "--role", "r", "--parent", "codex", "--backend", "orca", "--own-worktree"}, Fake: "detached"})
	envs := []map[string]string{nil, {"CLAUDECODE": "1", "CODEX_THREAD_ID": "thread", "PI_SESSION": "session"}, {"CODEX_THREAD_ID": "thread"}, {"PI_CODING_AGENT": "1"}, {"PI_SESSION": "session"}, {"ORCA_TERMINAL_HANDLE": "term", "REX_SESSION": "s", "REX_BLOCK": "b"}, {"REX_SESSION": "s", "REX_BLOCK": "b"}, {"REX_SESSION": "s"}}
	for i, e := range envs {
		all = append(all, fixture{Name: fmt.Sprintf("detect/%d", i), Sheet: "r: claude:opus, codex:gpt", Args: []string{"resolve", "--role", "r"}, Env: e})
	}
	for i, v := range []string{"interrogate reviewers=claude:opus @high, codex:gpt @xhigh", "interrogate reviewers=auto", "interrogate reviewers=other:model", "role=opus\nother: model", "default effort=low", "session hook=off", "=opus", "a:b=opus", "r=", "r=opus\r"} {
		add(fmt.Sprintf("models set/%d", i), "interrogate reviewers: auto\n", "models", "--set", v, "--json")
	}
	all = append(all, fixture{Name: "models set/new", Missing: true, Args: []string{"models", "--set", "role=auto", "--json"}})
	for _, m := range []string{"", "only-claude", "only-codex", "only-pi", "missing"} {
		all = append(all, fixture{Name: "starter/" + m, Missing: true, Args: []string{"setup", "--dry-run", "--json"}, Fake: m})
	}
	setups := []struct {
		name, text string
		args       []string
	}{
		{"dry run new", "", []string{"--dry-run", "--json"}}, {"new", "", []string{"--yes", "--json"}}, {"preserve", "my role: auto\nsession hook: off\n", []string{"--yes", "--json"}}, {"force", "my role: auto\n", []string{"--force", "--yes", "--json"}}, {"invalid preserve", "broken", []string{"--yes", "--json"}}, {"force invalid", "broken", []string{"--force", "--yes", "--json"}},
	}
	for _, s := range setups {
		all = append(all, fixture{Name: "setup/" + s.name, Sheet: s.text, Missing: s.text == "", Args: append([]string{"setup"}, s.args...)})
	}
	for _, m := range []string{"install-failure", "no-multi-agent"} {
		all = append(all, fixture{Name: "setup/" + m, Missing: true, Fake: m, Args: []string{"setup", "--yes", "--json"}})
	}
	all = append(all, fixture{Name: "setup/cancel", Missing: true, Args: []string{"setup"}, Stdin: "n\n"}, fixture{Name: "setup/json requires yes", Missing: true, Args: []string{"setup", "--json"}})
	for _, m := range []string{"", "backend-failure"} {
		all = append(all, fixture{Name: "doctor/healthy/" + m, Sheet: "r: auto\n", Args: []string{"doctor", "--json"}, Env: map[string]string{"CODEX_THREAD_ID": "thread", "ORCA_TERMINAL_HANDLE": "term"}, Fake: m})
	}
	add("doctor/invalid sheet", "broken", "doctor", "--json")
	add("doctor/unknown parent", "r: auto\n", "doctor", "--json")
	configs := []string{`model = "c"`, `model = "c" # pinned`, `"model" = "c"`, `model = 'c'`, `model = "a\"b" # note`, `model = "c`, "x = [\n 1,\n [2]\n]\nmodel = \"c\"", "notes = \"\"\"\n[hi]\n\"\"\"\nmodel = \"c\"", "[profiles.x]\nmodel = \"b\"", "a = 1\n[t]\nmodel = \"b\"", `model = "missing"`}
	for i, c := range configs {
		all = append(all, fixture{Name: fmt.Sprintf("topLevelModel/%d", i), Missing: true, Config: c, Args: []string{"setup", "--dry-run", "--json"}})
	}
	for i, pi := range []string{"provider model context\nopenai gpt-a 400K\nopenai gpt-a 400K\nanthropic opus 200K", "Warning: No models match pattern\nopenai/gpt-a\nanthropic/opus\n", ""} {
		all = append(all, fixture{Name: fmt.Sprintf("parsePi/%d", i), Sheet: "r: auto", Env: map[string]string{"PI_MODELS": pi}, Args: []string{"setup", "--dry-run", "--json"}})
	}
	for i, codex := range []string{"not json", "{}", "{\"models\":[]}"} {
		all = append(all, fixture{Name: fmt.Sprintf("discover/%d", i), Sheet: "r: auto", Env: map[string]string{"CODEX_MODELS": codex, "CODEX_THREAD_ID": "thread"}, Args: []string{"doctor", "--json"}})
	}
	for _, h := range []string{"claude", "codex", "pi"} {
		for _, e := range []string{"", "high"} {
			for _, own := range []bool{false, true} {
				args := []string{"rex", "start", "--harness", h, "--model", "opus", "--brief", "{brief}"}
				if e != "" {
					args = append(args, "--effort", e)
				}
				if own {
					args = append(args, "--own-worktree")
				}
				all = append(all, fixture{Name: fmt.Sprintf("rex start/%s/%s/%v", h, e, own), Args: args, Env: map[string]string{"REX_SESSION": "session-1"}})
			}
		}
	}
	for i, l := range []string{`{"block_ids":["first","second"]}`, "not json", "{}", `{"block_ids":[]}`} {
		all = append(all, fixture{Name: fmt.Sprintf("rex response/%d", i), Args: []string{"rex", "start", "--harness", "claude", "--model", "opus", "--brief", "{brief}"}, Env: map[string]string{"REX_SESSION": "session-1"}, RexLaunch: l})
	}
	all = append(all, fixture{Name: "rex missing session", Args: []string{"rex", "start", "--harness", "claude", "--model", "opus", "--brief", "{brief}"}})
	for _, id := range []string{"", ".", "..", "../escape"} {
		all = append(all, fixture{Name: "report invalid/" + id, Args: []string{"rex", "report", "--id", id}, Stdin: "report"})
	}
	for i, process := range []string{`{"last_exit":{"exit_code":0}}`, `{"last_exit":{"exit_code":7}}`, `{"last_exit":null}`, "broken", `{"last_exit":{"exit_code":"no"}}`} {
		all = append(all, fixture{Name: fmt.Sprintf("rex wait/%d", i), Args: []string{"rex", "wait", "--id", "fixture", "--seconds", "0"}, Process: process})
	}
	for _, m := range []string{"query-failure", "report-during-query"} {
		all = append(all, fixture{Name: "rex wait/" + m, Args: []string{"rex", "wait", "--id", "fixture", "--seconds", "0"}, Process: `{"last_exit":{"exit_code":0}}`, Fake: m})
	}
	all = append(all, fixture{Name: "rex reported", Args: []string{"rex", "wait", "--id", "fixture", "--seconds", "0"}, Result: "Results and evidence.\n", Process: `{"last_exit":null}`})
	all = append(all, fixture{Name: "discover/failed lists", Sheet: "r: auto", Fake: "failed-lists", Env: map[string]string{"CODEX_THREAD_ID": "thread"}, Args: []string{"doctor", "--json"}})
	for i, config := range []string{`model = "a\"b" # note`, "x = [\n \"[ignored]\", # ]\n [2]\n]\nmodel = \"c\"", "notes = '''\n[ignored]\n'''\nmodel = \"c\""} {
		all = append(all, fixture{Name: fmt.Sprintf("topLevelModel/extra-%d", i), Missing: true, Config: config, Env: map[string]string{"CODEX_MODELS": `{"models":[{"slug":"first"},{"slug":"a\"b"},{"slug":"c"}]}`}, Args: []string{"setup", "--dry-run", "--json"}})
	}
	for i, args := range [][]string{{"models", "--bogus"}, {"models", "--role", "r"}, {"resolve"}, {"resolve", "--role"}, {"setup", "--bogus"}, {"doctor", "--yes"}, {"rex", "start"}, {"rex", "start", "--harness", "bad", "--model", "m", "--brief", "{brief}"}, {"rex", "wait", "--id", "missing", "--seconds", "0"}, {"rex", "wait", "--id", "fixture", "--seconds", "bad"}} {
		all = append(all, fixture{Name: fmt.Sprintf("CLI errors/%d", i), Args: args})
	}
	sample, e := os.ReadFile("testdata/pi-models.txt")
	if e != nil {
		panic(e)
	}
	all = append(all, fixture{Name: "parsePi/captured 1.5.2", Sheet: "r: auto", Env: map[string]string{"PI_MODELS": string(sample)}, Args: []string{"setup", "--dry-run", "--json"}})
	for i, p := range []string{`[]`, `{"last_exit":[]}`, `{"last_exit":{"exit_code":1.0}}`} {
		all = append(all, fixture{Name: fmt.Sprintf("rex invalid process shape/%d", i), Args: []string{"rex", "wait", "--id", "fixture", "--seconds", "0"}, Process: p})
	}
	for i, p := range []string{`[]`, `{"block_id":3}`, `{"block_ids":[1]}`} {
		all = append(all, fixture{Name: fmt.Sprintf("rex invalid launch shape/%d", i), Args: []string{"rex", "start", "--harness", "claude", "--model", "opus", "--brief", "{brief}"}, Env: map[string]string{"REX_SESSION": "s"}, RexLaunch: p})
	}
	all = append(all,
		fixture{Name: "unicode/sheet whitespace", Sheet: "\u00a0r,\u2003s:\u00a0claude:opus\u202f@high\u3000", Args: []string{"models", "--json"}},
		fixture{Name: "unicode/escaped model", Missing: true, Config: "model = 'é'", Env: map[string]string{"CODEX_MODELS": `{"models":[{"slug":"first"},{"slug":"\u00e9"}]}`}, Args: []string{"setup", "--dry-run", "--json"}},
		fixture{Name: "unicode/surrogate model", Missing: true, Env: map[string]string{"CODEX_MODELS": `{"models":[{"slug":"model-\ud83d\ude80"}]}`}, Args: []string{"setup", "--dry-run", "--json"}})
	all = append(all, fixture{Name: "text/models", Text: true, Sheet: "r: claude:opus @high\nsession hook: off", Args: []string{"models"}}, fixture{Name: "text/setup", Text: true, Missing: true, Config: `model = "c"`, Args: []string{"setup", "--yes"}}, fixture{Name: "text/doctor", Text: true, Sheet: "r: auto", Env: map[string]string{"CODEX_THREAD_ID": "thread"}, Args: []string{"doctor"}}, fixture{Name: "text/version", Text: true, Args: []string{"version"}})
	return all
}

func rootpath(t *testing.T) string {
	t.Helper()
	p, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func write(t *testing.T, p, s string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(s), 0600); e != nil {
		t.Fatal(e)
	}
}
func normalized(v any, dir, script, id string) any {
	switch x := v.(type) {
	case string:
		x = strings.ReplaceAll(x, script, "{script}")
		x = strings.ReplaceAll(x, "pstack-relay 1.0.0", "pstack-relay 2.0.0")
		x = strings.ReplaceAll(x, "then run: pstack-relay rex", "then run: {script} rex")
		if x == "pstack-relay" {
			x = "{script}"
		}
		if id != "" {
			x = strings.ReplaceAll(x, id, "{id}")
		}
		x = strings.ReplaceAll(x, dir, "{tmp}")
		// The new launcher quotes paths in its report instruction.
		x = strings.ReplaceAll(x, "'{script}'", "{script}")
		x = strings.ReplaceAll(x, "'{tmp}", "{tmp}")
		x = strings.ReplaceAll(x, ".md'", ".md")
		return x
	case []any:
		for i := range x {
			x[i] = normalized(x[i], dir, script, id)
		}
		return x
	case map[string]any:
		for k := range x {
			x[k] = normalized(x[k], dir, script, id)
		}
		return x
	default:
		return x
	}
}
func execute(t *testing.T, c fixture, exe, shell string) expected {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(rootpath(t), "overlay/bin/pstack-relay")
	testbin, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, h := range []string{"codex", "claude", "pi", "orca", "rex", "hc", "git"} {
		p := filepath.Join(dir, "bin", h)
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.Symlink(testbin, p); e != nil {
			t.Fatal(e)
		}
	}
	sheet := filepath.Join(dir, "sheet.md")
	if !c.Missing {
		write(t, sheet, c.Sheet)
	}
	write(t, filepath.Join(dir, "brief.md"), "Do the task.")
	write(t, filepath.Join(dir, ".codex/config.toml"), c.Config)
	log := filepath.Join(dir, "calls.jsonl")
	write(t, log, "")
	env := map[string]string{"HOME": dir, "PATH": filepath.Join(dir, "bin") + ":/usr/bin:/bin", "TMPDIR": dir, "PSTACK_RELAY_SHEET": sheet, "PSTACK_RELAY_STATE": filepath.Join(dir, "state"), "RELAY_FAKE": "1", "GORACE": "atexit_sleep_ms=0", "CALL_LOG": log, "FAKE_MODE": c.Fake, "REX_LAUNCH": `{"block_id":"block-1"}`, "REX_PROCESS": c.Process, "WAIT_DIR": filepath.Join(dir, "state/rex/fixture")}
	for k, v := range c.Env {
		env[k] = v
	}
	if c.RexLaunch != "" {
		env["REX_LAUNCH"] = c.RexLaunch
	}
	args := append([]string(nil), c.Args...)
	for i := range args {
		if args[i] == "{brief}" {
			args[i] = filepath.Join(dir, "brief.md")
		}
	}
	if len(args) > 1 && args[1] == "wait" {
		write(t, filepath.Join(dir, "state/rex/fixture/launch.json"), `{"id":"fixture","session":"session-1","block":"block-1"}`)
		if c.Result != "" {
			write(t, filepath.Join(dir, "state/rex/fixture/result.md"), c.Result)
		}
	}
	var cmd *exec.Cmd
	if shell != "" {
		cmd = exec.Command(shell, append([]string{exe}, args...)...)
	} else {
		cmd = exec.Command(exe, args...)
	}
	// The actual relay is a shell executable; fake-mode only affects symlinked tools.
	cmd.Dir = dir
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdin = strings.NewReader(c.Stdin)
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	got := expected{Error: err != nil}
	if err != nil {
		return got
	} // Error spelling is deliberately not an API contract.
	var output any
	if c.Text {
		output = out.String()
	} else if e := json.Unmarshal(out.Bytes(), &output); e != nil {
		t.Fatalf("%s: invalid JSON %q stderr=%s", c.Name, out.String(), errout.String())
	}
	id := ""
	if m, ok := output.(map[string]any); ok && len(args) > 1 && args[1] == "start" {
		id, _ = m["id"].(string)
		if id == "" {
			t.Fatal("empty worker id")
		}
		d, _ := m["state_dir"].(string)
		b, e := os.ReadFile(filepath.Join(d, "brief.md"))
		if e != nil || string(b) != "Do the task." {
			t.Fatalf("brief %q %v", b, e)
		}
		info, e := os.Stat(d)
		if e != nil || info.Mode().Perm() != 0700 {
			t.Fatalf("state permissions %v %v", info, e)
		}
		b, e = os.ReadFile(filepath.Join(d, "launch.json"))
		var stored any
		if e != nil || json.Unmarshal(b, &stored) != nil || !reflect.DeepEqual(stored, output) {
			t.Fatal("stored launch differs")
		}
	}
	got.Output = normalized(output, dir, script, id)
	data, e := os.ReadFile(log)
	if e != nil {
		t.Fatal(e)
	}
	var calls []any
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var v any
		if e := json.Unmarshal(line, &v); e != nil {
			t.Fatal(e)
		}
		calls = append(calls, normalized(v, dir, script, id))
	}
	if len(calls) > 0 {
		got.Calls = calls
	}
	if data, e := os.ReadFile(sheet); e == nil {
		got.Sheet = string(data)
	}
	return got
}
func TestRuntime(t *testing.T) {
	root := rootpath(t)
	script := filepath.Join(root, "overlay/bin/pstack-relay")
	var golden map[string]expected
	data, e := os.ReadFile("testdata/go-expectations.json")
	if e == nil {
		if e = json.Unmarshal(data, &golden); e != nil {
			t.Fatal(e)
		}
	} else {
		t.Fatal(e)
	}
	if golden == nil {
		golden = map[string]expected{}
	}
	shells := []string{"sh"}
	if p, e := exec.LookPath("dash"); e == nil {
		shells = append(shells, p)
	}
	cases := cases()
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			want, ok := golden[c.Name]
			if !ok {
				t.Fatal("missing captured Go expectation")
			}
			for _, shell := range shells {
				t.Run(filepath.Base(shell), func(t *testing.T) {
					got := execute(t, c, script, shell)
					if !reflect.DeepEqual(got, want) {
						a, _ := json.MarshalIndent(want, "", "  ")
						b, _ := json.MarshalIndent(got, "", "  ")
						t.Fatalf("want %s\ngot %s", a, b)
					}
				})
			}
		})
	}
	if !t.Failed() {
		t.Logf("%d captured Go cases passed across %d shell(s)", len(cases), len(shells))
	}
}
