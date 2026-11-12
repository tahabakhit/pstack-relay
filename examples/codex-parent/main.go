package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tahabakhit/pstack-rex/internal/config"
	"github.com/tahabakhit/pstack-rex/internal/engine"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	binary := flag.String("binary", "", "Built pstack-rex executable")
	configPath := flag.String("config", "", "Existing config with external how explainer role")
	parentModel := flag.String("parent-model", "", "Explicit Codex parent model")
	effort := flag.String("parent-effort", "low", "Requested parent effort")
	out := flag.String("out", "", "New artifact directory, defaults to an external temporary directory")
	flag.Parse()
	if *binary == "" || *configPath == "" || *parentModel == "" {
		return fmt.Errorf("required flags: --binary PATH --config PATH --parent-model MODEL; this example makes real model calls")
	}
	var err error
	*binary, err = filepath.Abs(*binary)
	if err != nil {
		return err
	}
	*configPath, err = filepath.Abs(*configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(*configPath, "")
	if err != nil {
		return err
	}
	target, err := cfg.Resolve("how explainer", 0, nil)
	if err != nil {
		return err
	}
	if target.Harness == "codex" {
		return fmt.Errorf("configure how explainer as Claude or Pi to demonstrate cross-harness delegation")
	}
	if *out == "" {
		*out, err = os.MkdirTemp("", "pstack-rex-example-")
	} else {
		*out, err = filepath.Abs(*out)
		if err == nil {
			err = os.Mkdir(*out, 0700)
		}
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Artifacts:", *out)
	repo := filepath.Join(*out, "repo")
	if err = os.Mkdir(repo, 0700); err != nil {
		return err
	}
	fixture := "package calculator\n\nfunc Sum(values []int) int {\n total := 0\n for _, value := range values { total += value }\n return total\n}\n"
	if err = os.WriteFile(filepath.Join(repo, "calculator.go"), []byte(fixture), 0600); err != nil {
		return err
	}
	execute := func(executable string, args ...string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, args...)
		data, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %w: %s", executable, err, data)
		}
		return nil
	}
	if err = execute("git", "-C", repo, "init", "-q"); err != nil {
		return err
	}
	if err = execute("git", "-C", repo, "add", "calculator.go"); err != nil {
		return err
	}
	if err = execute("git", "-C", repo, "-c", "user.name=pstack-rex example", "-c", "user.email=example@localhost", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-qm", "Add example fixture"); err != nil {
		return err
	}
	if err = execute(*binary, "install", "--dest", filepath.Join(repo, ".agents")); err != nil {
		return err
	}
	state := filepath.Join(*out, "state")
	quote := func(value any) string { data, _ := json.Marshal(value); return string(data) }
	server := fmt.Sprintf("mcp_servers.pstack-rex={command=%s,args=%s,startup_timeout_sec=10,tool_timeout_sec=90,default_tools_approval_mode=\"approve\"}", quote(*binary), quote([]string{"--config", *configPath, "--state-root", state, "mcp"}))
	prompt := fmt.Sprintf("Use the installed pstack how workflow at %s to explain calculator.go Sum. Read the mandatory runtime contract it links. This is a simple question, so use its simple path with exactly one how explainer worker. Use pstack-rex MCP dispatch for role how explainer seat 0 and read-only access. Do not use native agents or launch coding harnesses directly. Wait for the worker's terminal result, use its returned explanation, and include worker id, model evidence and Rex block id in your final answer.", filepath.Join(repo, ".agents", "skills", "how", "SKILL.md"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	args := []string{"-a", "never", "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--disable", "multi_agent", "--disable", "multi_agent_v2", "--disable", "plugins", "--disable", "apps", "--disable", "hooks", "--model", *parentModel, "-c", "model_reasoning_effort=" + quote(*effort), "-c", server, "--json", "--sandbox", "read-only", "--cd", repo, prompt}
	cmd := exec.CommandContext(ctx, "codex", args...)
	transcript, err := os.OpenFile(filepath.Join(*out, "parent.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	stderr, err := os.OpenFile(filepath.Join(*out, "parent.stderr"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		transcript.Close()
		return err
	}
	cmd.Stdout = transcript
	cmd.Stderr = stderr
	err = cmd.Run()
	transcript.Close()
	stderr.Close()
	if err != nil {
		return fmt.Errorf("Codex parent failed: %w; inspect retained artifacts", err)
	}
	file, err := os.Open(filepath.Join(*out, "parent.jsonl"))
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	dispatched := map[string]bool{}
	completed := map[string]engine.Record{}
	finalText := ""
	parentComplete := false
	for scanner.Scan() {
		var event struct {
			Type string `json:"type"`
			Item struct {
				Type   string `json:"type"`
				Server string `json:"server"`
				Tool   string `json:"tool"`
				Text   string `json:"text"`
				Result struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
			} `json:"item"`
		}
		if err = json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return err
		}
		if event.Type == "turn.failed" || event.Type == "error" {
			return fmt.Errorf("parent stream reported failure")
		}
		if event.Type == "turn.completed" {
			parentComplete = true
		}
		if event.Type != "item.completed" {
			continue
		}
		if event.Item.Type == "agent_message" {
			finalText = event.Item.Text
		}
		if event.Item.Type != "mcp_tool_call" || event.Item.Server != "pstack-rex" {
			continue
		}
		for _, content := range event.Item.Result.Content {
			var record engine.Record
			if json.Unmarshal([]byte(content.Text), &record) != nil || record.ID == "" {
				continue
			}
			if event.Item.Tool == "dispatch" {
				dispatched[record.ID] = true
			}
			if event.Item.Tool == "wait" && record.Status == "completed" && record.Result != nil {
				completed[record.ID] = record
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if !parentComplete || strings.TrimSpace(finalText) == "" {
		return fmt.Errorf("parent lacks terminal success or final text")
	}
	service := engine.Service{Root: state, Config: cfg}
	for id, record := range completed {
		if !dispatched[id] {
			continue
		}
		actual, err := service.Inspect(context.Background(), id)
		if err != nil {
			return err
		}
		if actual.Status != "completed" || actual.Target.Harness != target.Harness || actual.Launch == nil || actual.Launch.Block == "" || actual.Result == nil || actual.Result.Outcome == nil {
			return fmt.Errorf("worker receipt failed independent artifact check")
		}
		if record.Result.Outcome == nil || !strings.Contains(finalText, "Sum") {
			return fmt.Errorf("parent did not return the requested explanation")
		}
		report := map[string]any{"status": "verified-execution-path", "parent_model_requested": *parentModel, "worker": actual, "parent_output": finalText, "artifacts": *out, "scope": "one simple pstack how workflow, not general workflow parity"}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(*out, "evidence.json"), append(data, '\n'), 0600); err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	return fmt.Errorf("no successful MCP dispatch/wait pair and durable worker result; inspect retained artifacts")
}
