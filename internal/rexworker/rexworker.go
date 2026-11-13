// Package rexworker launches interactive harness workers in Rex terminals.
package rexworker

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Spec struct {
	Harness, Model, Effort string
	OwnWorktree            bool
	BriefPath, Cwd         string
}

type Launch struct {
	ID       string `json:"id"`
	Session  string `json:"session"`
	Block    string `json:"block"`
	Cwd      string `json:"cwd"`
	StateDir string `json:"state_dir"`
}

type Status struct {
	ID       string
	State    string
	Result   string
	ExitCode *int
}

type Runner func(name string, args ...string) ([]byte, error)

func StateRoot(getenv func(string) string) string {
	root := getenv("XDG_STATE_HOME")
	if root == "" {
		root = filepath.Join(getenv("HOME"), ".local", "state")
	}
	return filepath.Join(root, "pstack-rex", "rex")
}

func Start(r Runner, stateRoot string, getenv func(string) string, s Spec) (Launch, error) {
	session := getenv("REX_SESSION")
	if session == "" {
		return Launch{}, errors.New("not inside Rex")
	}
	if s.Harness != "claude" && s.Harness != "codex" && s.Harness != "pi" {
		return Launch{}, fmt.Errorf("unsupported harness %q", s.Harness)
	}
	if s.Model == "" {
		return Launch{}, errors.New("model is required")
	}
	brief, err := os.ReadFile(s.BriefPath)
	if err != nil {
		return Launch{}, fmt.Errorf("read brief: %w", err)
	}
	cwd, err := filepath.Abs(s.Cwd)
	if err != nil {
		return Launch{}, err
	}
	root, err := filepath.Abs(stateRoot)
	if err != nil {
		return Launch{}, err
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return Launch{}, err
	}
	id := fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(random[:]))
	l := Launch{ID: id, Session: session, Cwd: cwd, StateDir: filepath.Join(root, id)}
	if err := os.MkdirAll(root, 0700); err != nil {
		return Launch{}, err
	}
	if err := os.Mkdir(l.StateDir, 0700); err != nil {
		return Launch{}, err
	}
	briefPath := filepath.Join(l.StateDir, "brief.md")
	if err := os.WriteFile(briefPath, brief, 0600); err != nil {
		return Launch{}, err
	}
	if s.OwnWorktree {
		l.Cwd = filepath.Join(l.StateDir, "worktree")
		if output, err := r("git", "-C", cwd, "worktree", "add", "-b", "pstack-rex/"+id, l.Cwd, "HEAD"); err != nil {
			return Launch{}, fmt.Errorf("create worktree: %w: %s", err, output)
		}
	}
	draft := filepath.Join(l.StateDir, "result-draft.md")
	prompt := fmt.Sprintf("Read and do the task in %s. When you are finished, write your final report (results and the evidence asked for) to %s, then run: pstack-rex rex report --id %s --file %s", briefPath, draft, id, draft)
	cmd := []string{s.Harness, "--model", s.Model}
	if s.Effort != "" {
		switch s.Harness {
		case "claude":
			cmd = append(cmd, "--effort", s.Effort)
		case "codex":
			cmd = append(cmd, "-c", fmt.Sprintf("model_reasoning_effort=%q", s.Effort))
		case "pi":
			cmd = append(cmd, "--thinking", s.Effort)
		}
	}
	cmd = append(cmd, prompt)
	args := []string{"run", "-s", session, "--split", "right", "--focus=false", "--label", "pstack-" + id, "-c", l.Cwd, "--json", "--"}
	output, err := r("rex", append(args, cmd...)...)
	if err != nil {
		return Launch{}, fmt.Errorf("launch Rex worker: %w: %s", err, output)
	}
	var created struct {
		BlockID  string   `json:"block_id"`
		BlockIDs []string `json:"block_ids"`
	}
	if err := json.Unmarshal(output, &created); err != nil {
		return Launch{}, fmt.Errorf("decode Rex launch: %w", err)
	}
	l.Block = created.BlockID
	if l.Block == "" && len(created.BlockIDs) > 0 {
		l.Block = created.BlockIDs[0]
	}
	if l.Block == "" {
		return Launch{}, errors.New("Rex launch returned no block id")
	}
	data, err := json.Marshal(l)
	if err != nil {
		return Launch{}, err
	}
	if err := os.WriteFile(filepath.Join(l.StateDir, "launch.json"), data, 0600); err != nil {
		return Launch{}, err
	}
	return l, nil
}

func workerDir(stateRoot, id string) (string, error) {
	if id == "" || id == "." || id == ".." || filepath.Base(id) != id {
		return "", errors.New("invalid worker id")
	}
	return filepath.Join(stateRoot, id), nil
}

func Report(stateRoot, id string, result []byte) error {
	dir, err := workerDir(stateRoot, id)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".result-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(result)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), filepath.Join(dir, "result.md")); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "\x1b]7501;state=done:id=%s\x1b\\", id)
	return err
}

func Wait(r Runner, stateRoot, id string, timeout time.Duration) (Status, error) {
	status := Status{ID: id, State: "running"}
	dir, err := workerDir(stateRoot, id)
	if err != nil {
		return status, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "launch.json"))
	if err != nil {
		return status, err
	}
	var l Launch
	if err := json.Unmarshal(data, &l); err != nil {
		return status, fmt.Errorf("decode launch: %w", err)
	}
	deadline := time.Now().Add(timeout)
	readResult := func() (bool, error) {
		result, err := os.ReadFile(filepath.Join(dir, "result.md"))
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		status.State, status.Result = "reported", string(result)
		return true, nil
	}
	for {
		if found, err := readResult(); found || err != nil {
			return status, err
		}
		output, err := r("rex", "block", "call", "-s", l.Session, "-b", l.Block, "process")
		if err != nil {
			return status, fmt.Errorf("query Rex process: %w: %s", err, output)
		}
		var process struct {
			LastExit *struct {
				ExitCode *int `json:"exit_code"`
			} `json:"last_exit"`
		}
		if err := json.Unmarshal(output, &process); err != nil {
			return status, fmt.Errorf("decode Rex process: %w", err)
		}
		if process.LastExit != nil && process.LastExit.ExitCode != nil {
			if found, err := readResult(); found || err != nil {
				return status, err
			}
			status.State, status.ExitCode = "exited", process.LastExit.ExitCode
			return status, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return status, nil
		}
		time.Sleep(min(time.Second, remaining))
	}
}
