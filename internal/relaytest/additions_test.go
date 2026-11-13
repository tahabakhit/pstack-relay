package relaytest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func isolated(t *testing.T) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, h := range []string{"codex", "claude", "pi", "orca", "rex", "hc"} {
		p := filepath.Join(dir, "bin", h)
		os.MkdirAll(filepath.Dir(p), 0700)
		if e := os.Symlink(exe, p); e != nil {
			t.Fatal(e)
		}
	}
	env := []string{"HOME=" + dir, "XDG_CONFIG_HOME=" + filepath.Join(dir, "config"), "TMPDIR=" + dir, "PATH=" + filepath.Join(dir, "bin") + ":/usr/bin:/bin", "RELAY_FAKE=1", "CALL_LOG=" + filepath.Join(dir, "calls"), "REX_LAUNCH=" + `{"block_id":"block-1"}`}
	return dir, env
}
func invoke(t *testing.T, dir string, env []string, stdin string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{filepath.Join(rootpath(t), "overlay/bin/pstack-relay")}, args...)...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, e := cmd.Output()
	if e != nil {
		return b, fmt.Errorf("%w: %s", e, stderr.String())
	}
	return b, nil
}
func object(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal(b, &v); e != nil {
		t.Fatalf("invalid JSON %q: %v", b, e)
	}
	return v
}
func TestHC(t *testing.T) {
	for _, h := range []string{"claude", "codex", "pi"} {
		for _, effort := range []string{"", "high"} {
			for _, own := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%v", h, effort, own), func(t *testing.T) {
					dir, env := isolated(t)
					env = append(env, "HC_TERMINAL_ID=t1")
					if own {
						for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", "initial"}} {
							cmd := exec.Command("git", args...)
							cmd.Dir = dir
							if b, e := cmd.CombinedOutput(); e != nil {
								t.Fatalf("git %q: %s %v", args, b, e)
							}
						}
					}
					write(t, filepath.Join(dir, "brief with spaces.md"), "Do the task.")
					args := []string{"hc", "start", "--harness", h, "--model", "model with spaces", "--brief", "brief with spaces.md"}
					if effort != "" {
						args = append(args, "--effort", effort)
					}
					if own {
						args = append(args, "--own-worktree")
					}
					b, e := invoke(t, dir, env, "", args...)
					if e != nil {
						t.Fatal(e)
					}
					launch := object(t, b)
					if launch["terminal"] != "t7" {
						t.Fatalf("terminal: %s", b)
					}
					id := launch["id"].(string)
					name := launch["name"].(string)
					if len(name) > 32 || strings.ToLower(name) != name {
						t.Fatalf("invalid hc name %s", name)
					}
					stateDir := launch["state_dir"].(string)
					if own {
						if launch["cwd"] != filepath.Join(stateDir, "worktree") {
							t.Fatal("no worktree")
						}
						t.Cleanup(func() {
							cmd := exec.Command("git", "worktree", "remove", "--force", filepath.Join(stateDir, "worktree"))
							cmd.Dir = dir
							cmd.Run()
						})
					}
					data, e := os.ReadFile(filepath.Join(dir, "calls"))
					if e != nil {
						t.Fatal(e)
					}
					lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
					var call []string
					if e = json.Unmarshal(lines[0], &call); e != nil {
						t.Fatal(e)
					}
					want := []string{"hc", "run", "--json", "--bg", "--name", name, "--cwd", launch["cwd"].(string), h, "--model", "model with spaces"}
					if effort != "" {
						switch h {
						case "claude":
							want = append(want, "--effort", effort)
						case "codex":
							want = append(want, "-c", `model_reasoning_effort="high"`)
						case "pi":
							want = append(want, "--thinking", effort)
						}
					}
					if strings.Join(call[:len(call)-1], "\x00") != strings.Join(want, "\x00") {
						t.Fatalf("call=%q want=%q", call, want)
					}
					prompt := call[len(call)-1]
					for _, part := range []string{filepath.Join(stateDir, "brief.md"), "hc send 't1'", filepath.Join(stateDir, "result.md")} {
						if !strings.Contains(prompt, part) {
							t.Fatalf("prompt missing %s: %s", part, prompt)
						}
					}
					for _, wait := range []struct{ raw, state string }{{`{"terminal":"t7","reason":"timeout","status":"working"}`, "running"}, {`{"terminal":"t7","reason":"status","status":"done"}`, "reported"}, {`{"terminal":"t7","reason":"status","status":"blocked"}`, "blocked"}, {`{"terminal":"t7","reason":"status","status":"error"}`, "error"}, {`{"terminal":"t7","reason":"exit"}`, "exited"}, {`{"terminal":"t7","reason":"message"}`, "running"}} {
						b, e = invoke(t, dir, append(env, "HC_WAIT="+wait.raw), "", "hc", "wait", "--id", id, "--seconds", "0")
						if e != nil {
							t.Fatal(e)
						}
						v := object(t, b)
						if v["state"] != wait.state || v["result"] != "final screen\n" {
							t.Fatalf("wait %s", b)
						}
					}
					write(t, filepath.Join(stateDir, "result.md"), "report\n\n")
					b, e = invoke(t, dir, append(env, `HC_WAIT={"reason":"timeout"}`), "", "hc", "wait", "--id", id, "--seconds", "0")
					if e != nil {
						t.Fatal(e)
					}
					v := object(t, b)
					if v["state"] != "reported" || v["result"] != "report\n\n" {
						t.Fatalf("file report %s", b)
					}
				})
			}
		}
	}
}
func TestHCResolver(t *testing.T) {
	for _, p := range []string{"claude", "codex", "pi"} {
		for _, own := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", p, own), func(t *testing.T) {
				dir, env := isolated(t)
				env = append(env, "HC_TERMINAL_ID=t1")
				init := exec.Command("git", "init", "-q")
				init.Dir = dir
				if b, e := init.CombinedOutput(); e != nil {
					t.Fatalf("git init: %s %v", b, e)
				}
				write(t, filepath.Join(dir, "sheet.md"), "r: claude:opus @high, codex:gpt @max, pi:openai/gpt @medium")
				args := []string{"resolve", "--sheet", "sheet.md", "--role", "r", "--parent", p}
				if own {
					args = append(args, "--own-worktree")
				}
				b, e := invoke(t, dir, env, "", args...)
				if e != nil {
					t.Fatal(e)
				}
				var placements []map[string]any
				if e = json.Unmarshal(b, &placements); e != nil {
					t.Fatal(e)
				}
				for _, v := range placements {
					h := v["harness"].(string)
					if h == p {
						if v["mode"] != "native" {
							t.Fatal(v)
						}
						continue
					}
					if v["mode"] != "hc" {
						t.Fatal(v)
					}
					cmd := v["command"].([]any)
					want := []any{filepath.Join(rootpath(t), "overlay/bin/pstack-relay"), "hc", "start", "--harness", h, "--model", v["model"], "--effort", v["effort"]}
					if own {
						want = append(want, "--own-worktree")
					}
					want = append(want, "--brief", "{brief}")
					a, _ := json.Marshal(cmd)
					z, _ := json.Marshal(want)
					if !bytes.Equal(a, z) {
						t.Fatalf("%s want %s", a, z)
					}
				}
			})
		}
	}
	// Backend priority: Orca, complete Rex, hc, none.
	for _, vars := range [][]string{{"HC_TERMINAL_ID=t1", "ORCA_TERMINAL_HANDLE=term"}, {"HC_TERMINAL_ID=t1", "REX_SESSION=s", "REX_BLOCK=b"}, {"HC_TERMINAL_ID=t1"}} {
		dir, env := isolated(t)
		write(t, filepath.Join(dir, "sheet.md"), "r: claude:opus")
		b, e := invoke(t, dir, append(env, vars...), "", "resolve", "--sheet", "sheet.md", "--role", "r", "--parent", "codex")
		if e != nil {
			t.Fatal(e)
		}
		var v []map[string]any
		json.Unmarshal(b, &v)
		want := "hc"
		if len(vars) == 2 {
			want = "orca"
		}
		if len(vars) == 3 {
			want = "rex"
		}
		if v[0]["mode"] != want {
			t.Fatalf("%s", b)
		}
	}
}
func TestPathsMigrationAndAtomicWrites(t *testing.T) {
	for _, mode := range []string{"home", "xdg", "override", "migration", "force", "dry"} {
		t.Run(mode, func(t *testing.T) {
			dir, env := isolated(t)
			sheet := filepath.Join(dir, "config/pstack-relay/pstack-models.md")
			if mode == "home" {
				env = append(env, "XDG_CONFIG_HOME=")
				sheet = filepath.Join(dir, ".config/pstack-relay/pstack-models.md")
			}
			if mode == "override" {
				sheet = filepath.Join(dir, "custom.md")
				env = append(env, "PSTACK_RELAY_SHEET="+sheet)
			}
			args := []string{"setup", "--yes", "--json"}
			if mode == "migration" || mode == "force" || mode == "dry" {
				write(t, filepath.Join(dir, ".config/pstack-rex/pstack-models.md"), "mine: codex:mine @low\nsession hook: off\n")
			}
			if mode == "force" {
				args = append(args, "--force")
			}
			if mode == "dry" {
				args = append(args, "--dry-run")
			}
			b, e := invoke(t, dir, env, "", args...)
			if e != nil {
				t.Fatal(e)
			}
			v := object(t, b)
			if v["path"] != sheet {
				t.Fatalf("%s", b)
			}
			data, e := os.ReadFile(sheet)
			if mode == "dry" {
				if !os.IsNotExist(e) {
					t.Fatal("dry-run wrote sheet")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if mode == "migration" {
				if !strings.Contains(string(data), "mine: codex:mine @low") || !strings.Contains(string(b), "Migrated model sheet") {
					t.Fatal(string(b))
				}
			}
			// Invalid edits must leave the existing file unchanged.
			_, e = invoke(t, dir, env, "", "models", "--set", "r=bad:model")
			if e == nil {
				t.Fatal("accepted invalid edit")
			}
			after, e := os.ReadFile(sheet)
			if e != nil || !bytes.Equal(data, after) {
				t.Fatal("invalid edit mutated sheet")
			}
			left, _ := filepath.Glob(filepath.Join(filepath.Dir(sheet), ".pstack-models-*"))
			if len(left) != 0 {
				t.Fatalf("temporary files: %v", left)
			}
		})
	}
}
func TestRexReport(t *testing.T) {
	dir, env := isolated(t)
	env = append(env, "PSTACK_RELAY_STATE="+filepath.Join(dir, "state"))
	d := filepath.Join(dir, "state/rex/fixture")
	write(t, filepath.Join(d, "launch.json"), `{"session":"s","block":"b"}`)
	for _, input := range []string{"Results and evidence.", "\n\n", ""} {
		b, e := invoke(t, dir, env, input, "rex", "report", "--id", "fixture")
		if e != nil {
			t.Fatal(e)
		}
		if string(b) != "\x1b]7501;state=done:id=fixture\x1b\\" {
			t.Fatalf("OSC %q", b)
		}
		b, e = invoke(t, dir, env, "", "rex", "wait", "--id", "fixture", "--seconds", "0")
		if e != nil {
			t.Fatal(e)
		}
		v := object(t, b)
		if v["state"] != "reported" || (input != "" && v["result"] != input) {
			t.Fatal(v)
		}
		left, _ := filepath.Glob(filepath.Join(d, ".result-*"))
		if len(left) != 0 {
			t.Fatal(left)
		}
	}
}

func TestSheetLineNumbersAndRoundTrip(t *testing.T) {
	for _, tc := range []struct{ text, problem string }{{"# hi\nrole: cursor:model", "line 2: unknown harness cursor"}, {"role: opus @ultra", "line 1: unknown effort"}, {"default effort: ultra", "line 1: unknown effort"}, {"session hook: yes", "line 1: session hook"}, {"role, : model", "line 1: invalid role"}} {
		dir, env := isolated(t)
		write(t, filepath.Join(dir, "sheet.md"), tc.text)
		_, e := invoke(t, dir, env, "", "models", "--sheet", "sheet.md")
		if e == nil || !strings.Contains(e.Error(), tc.problem) {
			t.Fatalf("%q: %v", tc.text, e)
		}
	}
	dir, env := isolated(t)
	write(t, filepath.Join(dir, "sheet.md"), "# prose\nz: auto\na: pi:openai/gpt @session\nz: codex:gpt @max\ndefault effort: session\n")
	b, e := invoke(t, dir, env, "", "models", "--sheet", "sheet.md")
	if e != nil {
		t.Fatal(e)
	}
	write(t, filepath.Join(dir, "sheet.md"), string(b))
	again, e := invoke(t, dir, env, "", "models", "--sheet", "sheet.md")
	if e != nil || !bytes.Equal(b, again) {
		t.Fatalf("round trip %q -> %q %v", b, again, e)
	}
}
func TestNoReplaceRace(t *testing.T) {
	dir, env := isolated(t)
	sheet := filepath.Join(dir, "config/pstack-relay/pstack-models.md")
	// Simulate the sheet appearing after discovery, immediately before link(2).
	write(t, filepath.Join(dir, "bin/ln"), "#!/bin/sh\nprintf 'race: auto\\nsession hook: off\\n' > \"$3\"\nexit 1\n")
	if e := os.Chmod(filepath.Join(dir, "bin/ln"), 0700); e != nil {
		t.Fatal(e)
	}
	b, e := invoke(t, dir, env, "", "setup", "--yes", "--json")
	if e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile(sheet)
	if e != nil || string(data) != "race: auto\nsession hook: off\n" {
		t.Fatalf("race overwrote %q %v", data, e)
	}
	// The returned sheet reflects the winner of the race.
	if object(t, b)["sheet"] != string(data) {
		t.Fatalf("stale sheet in response: %s", b)
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(sheet), ".pstack-models-*"))
	if len(left) > 0 {
		t.Fatal(left)
	}
}
