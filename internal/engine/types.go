package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tahabakhit/pstack-rex/internal/config"
	"github.com/tahabakhit/pstack-rex/internal/harness"
)

type Request struct {
	ID               string        `json:"id"`
	Role             string        `json:"role"`
	Seat             int           `json:"seat"`
	Cwd              string        `json:"cwd"`
	Prompt           string        `json:"prompt"`
	Access           string        `json:"access"`
	Target           config.Target `json:"target"`
	TimeoutSeconds   int           `json:"timeout_seconds"`
	RequireMCP       bool          `json:"require_mcp,omitempty"`
	RequireShell     bool          `json:"require_shell,omitempty"`
	ExpectedRevision string        `json:"expected_revision,omitempty"`
	Executable       string        `json:"executable,omitempty"`
}

type Launch struct {
	Session   string `json:"session_id,omitempty"`
	Block     string `json:"block_id,omitempty"`
	Error     string `json:"error,omitempty"`
	Uncertain bool   `json:"uncertain,omitempty"`
}

type Started struct {
	PID         int       `json:"pid"`
	Session     string    `json:"session_id"`
	Block       string    `json:"block_id"`
	At          time.Time `json:"at"`
	Revision    string    `json:"revision"`
	Cwd         string    `json:"cwd"`
	Enforcement string    `json:"enforcement"`
}

type Result struct {
	Status              string           `json:"status"`
	ExitCode            int              `json:"exit_code"`
	Error               string           `json:"error,omitempty"`
	Outcome             *harness.Outcome `json:"outcome,omitempty"`
	Finished            time.Time        `json:"finished"`
	StartRevision       string           `json:"start_revision"`
	EndRevision         string           `json:"end_revision"`
	DiffSHA256          string           `json:"diff_sha256,omitempty"`
	Verification        string           `json:"verification"`
	StartSnapshotSHA256 string           `json:"start_snapshot_sha256,omitempty"`
	EndSnapshotSHA256   string           `json:"end_snapshot_sha256,omitempty"`
}

type Record struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	Role      string        `json:"role"`
	Target    config.Target `json:"target"`
	Artifacts string        `json:"artifacts"`
	Launch    *Launch       `json:"launch,omitempty"`
	Started   *Started      `json:"started,omitempty"`
	Result    *Result       `json:"result,omitempty"`
	Error     string        `json:"error,omitempty"`
}

type Service struct {
	Root             string
	Config           config.Config
	RexExecutable    string
	WorkerExecutable string
}

func DefaultRoot() string {
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(root, "pstack-rex")
}

var validID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

func (s *Service) dir(id string) (string, error) {
	if !validID.MatchString(id) {
		return "", fmt.Errorf("invalid worker id, use 1-80 letters, digits, underscore or hyphen")
	}
	return filepath.Join(s.Root, "workers", id), nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".receipt-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(append(data, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("corrupt receipt %s: %w", path, err)
	}
	return nil
}

func git(ctx context.Context, cwd string, args ...string) (string, error) {
	base := []string{"--no-pager", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-C", cwd}
	{
		query := exec.CommandContext(ctx, "git", append(append([]string{}, base...), "config", "--null", "--name-only", "--get-regexp", `^filter\..*\.(smudge|clean|process|required)$`)...)
		data, err := query.Output()
		if err != nil {
			if failure, ok := err.(*exec.ExitError); !ok || failure.ExitCode() != 1 {
				return "", fmt.Errorf("inspect Git filters: %w", err)
			}
		}
		for _, key := range strings.Split(string(data), "\x00") {
			if key == "" {
				continue
			}
			value := ""
			if strings.HasSuffix(key, ".required") {
				value = "false"
			}
			base = append(base, "-c", key+"="+value)
		}
	}
	if len(args) > 0 && args[0] == "diff" {
		args = append([]string{"diff", "--no-ext-diff", "--no-textconv"}, args[1:]...)
	}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(data), nil
}

func terminal(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "timed_out", "launch_failed":
		return true
	}
	return false
}
