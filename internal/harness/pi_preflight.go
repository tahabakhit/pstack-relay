package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pstack-rex/pstack-rex/internal/config"
)

type PiSelection struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Effort   string `json:"effort"`
	Evidence string `json:"evidence"`
}

func ProbePi(ctx context.Context, plan Plan, target config.Target, cwd string) (PiSelection, error) {
	args := make([]string, 0, len(plan.Args))
	for i := 0; i < len(plan.Args); i++ {
		if plan.Args[i] == "--print" {
			continue
		}
		if plan.Args[i] == "--mode" && i+1 < len(plan.Args) {
			args = append(args, "--mode", "rpc")
			i++
			continue
		}
		args = append(args, plan.Args[i])
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, plan.Executable, args...)
	cmd.Dir = cwd
	cmd.Env = plan.Environment
	cmd.Stdin = strings.NewReader("{\"id\":\"selection\",\"type\":\"get_state\"}\n")
	cmd.WaitDelay = time.Second
	output, err := cmd.Output()
	if err != nil {
		return PiSelection{}, fmt.Errorf("Pi selection preflight failed: %w", err)
	}
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var response struct {
			Type    string `json:"type"`
			ID      string `json:"id"`
			Command string `json:"command"`
			Success bool   `json:"success"`
			Data    struct {
				Model struct {
					ID       string `json:"id"`
					Provider string `json:"provider"`
				} `json:"model"`
				Effort string `json:"thinkingLevel"`
			} `json:"data"`
		}
		if err = json.Unmarshal(line, &response); err != nil {
			return PiSelection{}, fmt.Errorf("invalid Pi selection response")
		}
		if response.ID != "selection" {
			continue
		}
		if !response.Success || response.Command != "get_state" {
			return PiSelection{}, fmt.Errorf("Pi selection request rejected")
		}
		actual := response.Data.Model.Provider + "/" + response.Data.Model.ID
		if actual != target.Model {
			return PiSelection{}, fmt.Errorf("Pi selected %q instead of exact requested %q; no inference dispatched", actual, target.Model)
		}
		if target.Effort != "" && target.Effort != "default" && response.Data.Effort != target.Effort {
			return PiSelection{}, fmt.Errorf("Pi normalized effort from %q to %q; no inference dispatched", target.Effort, response.Data.Effort)
		}
		return PiSelection{Provider: response.Data.Model.Provider, Model: response.Data.Model.ID, Effort: response.Data.Effort, Evidence: "pi-rpc-get_state-before-inference"}, nil
	}
	return PiSelection{}, fmt.Errorf("Pi preflight lacks a correlated state response; no inference dispatched")
}
