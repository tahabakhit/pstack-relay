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

func TestHCStartupWait(t *testing.T) {
	for _, tc := range []struct {
		name, responses, seconds, state string
	}{
		{"idle then first turn", `[{"reason":"idle","status":"idle","turn":0},{"reason":"idle","status":"idle","turn":1}]`, "4", "idle"},
		{"done before first turn", `[{"reason":"status","status":"done","turn":0},{"reason":"status","status":"blocked","turn":0}]`, "4", "blocked"},
		{"message before first turn", `[{"reason":"message","turn":0},{"reason":"exit","turn":0}]`, "4", "exited"},
		{"startup timeout", `[{"reason":"idle","status":"idle","turn":0}]`, "1", "running"},
		{"zero budget", `[{"reason":"idle","status":"idle","turn":0}]`, "0", "running"},
		{"startup error", `[{"reason":"status","status":"error","turn":0}]`, "4", "error"},
		{"startup exit", `[{"reason":"exit","turn":0}]`, "4", "exited"},
		{"reported", `[{"reason":"idle","status":"idle","turn":0}]`, "4", "reported"},
		{"report while waiting", `[{"reason":"idle","status":"idle","turn":0}]`, "4", "reported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, env := isolated(t)
			d := filepath.Join(dir, "state/hc/fixture")
			write(t, filepath.Join(d, "launch.json"), `{"name":"worker"}`)
			env = append(env, "PSTACK_RELAY_STATE="+filepath.Join(dir, "state"), "HC_WAIT_SEQUENCE="+tc.responses)
			if tc.name == "reported" {
				write(t, filepath.Join(d, "result.md"), "finished\n")
			}
			if tc.name == "report while waiting" {
				env = append(env, "FAKE_MODE=hc-report-during-wait", "WAIT_DIR="+d)
			}
			b, err := invoke(t, dir, env, "", "hc", "wait", "--id", "fixture", "--seconds", tc.seconds)
			if err != nil {
				t.Fatal(err)
			}
			v := object(t, b)
			if v["state"] != tc.state {
				t.Fatalf("wait returned %s", b)
			}
			if tc.state == "reported" && v["result"] != "finished\n" {
				t.Fatalf("missing report: %s", b)
			}
			if tc.name == "idle then first turn" || tc.name == "done before first turn" {
				steps, err := os.ReadFile(filepath.Join(dir, "calls.step"))
				if err != nil || len(steps) < 2 {
					t.Fatalf("wait stopped at startup: steps=%q err=%v", steps, err)
				}
			}
		})
	}
}

func TestDefaultWaitBudget(t *testing.T) {
	dir, env := isolated(t)
	write(t, filepath.Join(dir, "state/hc/fixture/launch.json"), `{"name":"worker"}`)
	write(t, filepath.Join(dir, "bin/date"), "#!/bin/sh\nprintf '1000\\n'\n")
	if err := os.Chmod(filepath.Join(dir, "bin/date"), 0700); err != nil {
		t.Fatal(err)
	}
	env = append(env, "PSTACK_RELAY_STATE="+filepath.Join(dir, "state"), `HC_WAIT={"reason":"status","status":"blocked","turn":0}`)
	if _, err := invoke(t, dir, env, "", "hc", "wait", "--id", "fixture"); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(filepath.Join(dir, "calls"))
	if err != nil {
		t.Fatal(err)
	}
	var call []string
	if err := json.Unmarshal(bytes.Split(calls, []byte("\n"))[0], &call); err != nil {
		t.Fatal(err)
	}
	if strings.Join(call, " ") != "hc wait worker --timeout 100 --json" {
		t.Fatalf("default wait budget: %q", call)
	}
}

func TestLargeWorkerReports(t *testing.T) {
	for _, backend := range []string{"rex", "hc"} {
		for _, suffix := range []string{"", "\n", "\n\n"} {
			t.Run(fmt.Sprintf("%s/%d-newlines", backend, len(suffix)), func(t *testing.T) {
				dir, env := isolated(t)
				d := filepath.Join(dir, "state", backend, "fixture")
				write(t, filepath.Join(d, "launch.json"), `{"name":"worker","session":"s","block":"b"}`)
				report := "Unicode é🚀\n" + strings.Repeat("Markdown `code` $(command) \\\"\t\r\n", 5000) + "last line" + suffix
				for c := byte(1); c < 32; c++ {
					report = string(c) + report
				}
				write(t, filepath.Join(d, "result.md"), report)
				if len(report) <= 128*1024 {
					t.Fatal("fixture must exceed the environment argument limit")
				}
				env = append(env, "PSTACK_RELAY_STATE="+filepath.Join(dir, "state"), `HC_WAIT={"reason":"timeout","turn":0}`)
				b, err := invoke(t, dir, env, "", backend, "wait", "--id", "fixture", "--seconds", "0")
				if err != nil {
					t.Fatal(err)
				}
				v := object(t, b)
				if v["state"] != "reported" || v["result"] != report {
					t.Fatalf("report changed, state=%v result bytes=%d want=%d", v["state"], len(fmt.Sprint(v["result"])), len(report))
				}
			})
		}
	}
}

func TestWorkerReportCommand(t *testing.T) {
	for _, backend := range []string{"rex", "hc"} {
		t.Run(backend, func(t *testing.T) {
			dir, env := isolated(t)
			for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", "initial"}} {
				cmd := exec.Command("git", args...)
				cmd.Dir = dir
				if b, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %q: %s %v", args, b, err)
				}
			}
			state := "state 'quoted' $(touch INJECTED)"
			env = append(env, "PSTACK_RELAY_STATE="+state, "REX_SESSION=s", "HC_TERMINAL_ID=parent")
			write(t, filepath.Join(dir, "brief.md"), "Do the task.")
			b, err := invoke(t, dir, env, "", backend, "start", "--harness", "codex", "--model", "model", "--brief", "brief.md", "--own-worktree")
			if err != nil {
				t.Fatal(err)
			}
			launch := object(t, b)
			cwd := launch["cwd"].(string)
			t.Cleanup(func() {
				cmd := exec.Command("git", "worktree", "remove", "--force", cwd)
				cmd.Dir = dir
				cmd.Run()
			})
			calls, err := os.ReadFile(filepath.Join(dir, "calls"))
			if err != nil {
				t.Fatal(err)
			}
			var call []string
			if err := json.Unmarshal(bytes.TrimSpace(calls), &call); err != nil {
				t.Fatal(err)
			}
			prompt := call[len(call)-1]
			_, command, ok := strings.Cut(prompt, "then run: ")
			if !ok {
				t.Fatalf("no report command: %s", prompt)
			}
			draft := "result.md"
			if backend == "rex" {
				draft = "result-draft.md"
			}
			write(t, filepath.Join(launch["state_dir"].(string), draft), "Evidence `code` $(touch INJECTED)\n")
			cmd := exec.Command("sh", "-c", command)
			cmd.Dir = cwd
			cmd.Env = append(env, "PSTACK_RELAY_STATE=wrong-state")
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("report command: %s %v", b, err)
			}
			if backend == "rex" {
				b, err = invoke(t, dir, env, "", "rex", "wait", "--id", launch["id"].(string), "--seconds", "0")
				if err != nil || object(t, b)["result"] != "Evidence `code` $(touch INJECTED)\n" {
					t.Fatalf("report not delivered: %s %v", b, err)
				}
			} else {
				calls, err = os.ReadFile(filepath.Join(dir, "calls"))
				if err != nil {
					t.Fatal(err)
				}
				lines := bytes.Split(bytes.TrimSpace(calls), []byte("\n"))
				json.Unmarshal(lines[len(lines)-1], &call)
				want := []string{"hc", "send", "parent", filepath.Join(launch["state_dir"].(string), "result.md")}
				if strings.Join(call, "\x00") != strings.Join(want, "\x00") {
					t.Fatalf("report send %q want %q", call, want)
				}
			}
			if _, err := os.Stat(filepath.Join(cwd, "INJECTED")); !os.IsNotExist(err) {
				t.Fatal("report command executed path or Markdown as shell code")
			}
		})
	}
}

func TestPrivateStateRoot(t *testing.T) {
	for _, backend := range []string{"rex", "hc"} {
		for _, mode := range []string{"new", "private", "group-writable", "world-writable", "symlink", "wrong-owner"} {
			t.Run(backend+"/"+mode, func(t *testing.T) {
				dir, env := isolated(t)
				base := filepath.Join(dir, "state")
				if mode != "new" {
					if err := os.Mkdir(base, 0700); err != nil {
						t.Fatal(err)
					}
				}
				switch mode {
				case "group-writable":
					os.Chmod(base, 0770)
				case "world-writable":
					os.Chmod(base, 0702)
				case "symlink":
					os.Rename(base, base+"-target")
					os.Symlink(base+"-target", base)
				case "wrong-owner":
					write(t, filepath.Join(dir, "bin/id"), fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' %d\n", os.Getuid()+1))
					os.Chmod(filepath.Join(dir, "bin/id"), 0700)
				}
				write(t, filepath.Join(dir, "brief.md"), "Do the task.")
				env = append(env, "PSTACK_RELAY_STATE="+base, "REX_SESSION=s", "HC_TERMINAL_ID=t1")
				_, err := invoke(t, dir, env, "", backend, "start", "--harness", "claude", "--model", "model", "--brief", "brief.md")
				if mode == "new" || mode == "private" {
					if err != nil {
						t.Fatal(err)
					}
					info, err := os.Stat(base)
					if err != nil || info.Mode().Perm() != 0700 {
						t.Fatalf("state root mode: %v %v", info, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "state directory") {
					t.Fatalf("accepted unsafe root %s: %v", mode, err)
				}
			})
		}
	}
}
