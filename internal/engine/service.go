package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/pstack-rex/pstack-rex/internal/harness"
)

func (s *Service) lock() (*os.File, error) {
	if !filepath.IsAbs(s.Root) {
		return nil, fmt.Errorf("state root must be absolute")
	}
	if err := os.MkdirAll(filepath.Join(s.Root, "workers"), 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(s.Root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("state root must be a private directory with mode 0700")
	}
	f, err := os.OpenFile(filepath.Join(s.Root, "admission.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func unlock(f *os.File) { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }

func (s *Service) rex(ctx context.Context, args ...string) ([]byte, error) {
	executable := s.RexExecutable
	if executable == "" {
		executable = "rex"
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, append([]string{"--autostart=false", "--timeout", "10s"}, args...)...)
	cmd.Env = os.Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("rex %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (s *Service) localServer(ctx context.Context) error {
	data, err := s.rex(ctx, "whoami", "--json")
	if err != nil {
		return err
	}
	var who struct {
		Endpoint string `json:"endpoint"`
		API      int    `json:"api_revision"`
	}
	if err = json.Unmarshal(data, &who); err != nil {
		return err
	}
	if !strings.HasPrefix(who.Endpoint, "unix://") {
		return fmt.Errorf("remote Rex endpoint unsupported: run pstack-rex on the server host using its local Unix socket")
	}
	if who.API < 12 {
		return fmt.Errorf("Rex API revision %d is unsupported; require at least 12", who.API)
	}
	return nil
}

func (s *Service) Dispatch(ctx context.Context, req Request) (Record, error) {
	dir, err := s.dir(req.ID)
	if err != nil {
		return Record{}, err
	}
	if req.RequireMCP || strings.HasPrefix(req.Role, "why ") || strings.HasPrefix(req.Role, "reflect ") {
		return Record{}, fmt.Errorf("worker MCP access is unsupported by the controlled one-shot profiles; this workflow is blocked")
	}
	if req.Prompt == "" || !utf8.ValidString(req.Prompt) || len(req.Prompt) > 256*1024 || strings.ContainsRune(req.Prompt, 0) {
		return Record{}, fmt.Errorf("prompt must be nonempty UTF-8 text without NUL, at most 256 KiB")
	}
	if req.Seat < 0 {
		return Record{}, fmt.Errorf("seat must be nonnegative")
	}
	req.Target, err = s.Config.Resolve(req.Role, req.Seat, nil)
	if err != nil {
		return Record{}, err
	}
	if (req.RequireShell || req.Role == "interrogate reviewers") && req.Target.Harness != "codex" {
		return Record{}, fmt.Errorf("worker requires shell execution, unavailable in the controlled %s profile; this workflow is blocked", req.Target.Harness)
	}
	if req.Access == "" {
		req.Access = "read-only"
	}
	req.Cwd, err = filepath.Abs(req.Cwd)
	if err != nil {
		return Record{}, err
	}
	req.Cwd, err = filepath.EvalSymlinks(req.Cwd)
	if err != nil {
		return Record{}, err
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = s.Config.TimeoutSeconds
	}
	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > s.Config.TimeoutSeconds {
		return Record{}, fmt.Errorf("timeout must be between 1 and configured maximum %d seconds", s.Config.TimeoutSeconds)
	}
	plan, err := harness.Build(req.Target, req.Access, req.Cwd, req.Prompt)
	if err != nil {
		return Record{}, err
	}
	req.Executable = plan.Executable
	rev, err := git(ctx, req.Cwd, "rev-parse", "HEAD")
	if err != nil {
		return Record{}, err
	}
	rev = strings.TrimSpace(rev)
	if req.ExpectedRevision != "" && req.ExpectedRevision != rev {
		return Record{}, fmt.Errorf("expected revision differs from checkout HEAD")
	}
	req.ExpectedRevision = rev
	lock, err := s.lock()
	if err != nil {
		return Record{}, err
	}
	defer unlock(lock)
	var existing Request
	err = readJSON(filepath.Join(dir, "request.json"), &existing)
	if err == nil {
		a, _ := json.Marshal(req)
		b, _ := json.Marshal(existing)
		if !bytes.Equal(a, b) {
			return Record{}, fmt.Errorf("idempotency conflict: %s already has a different request", req.ID)
		}
		return s.Inspect(ctx, req.ID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Record{}, err
	}
	records, err := s.List(ctx)
	if err != nil {
		return Record{}, err
	}
	active := 0
	for _, record := range records {
		if !terminal(record.Status) {
			active++
		}
	}
	if active >= s.Config.MaxConcurrent {
		return Record{}, fmt.Errorf("concurrency limit reached (%d); inspect unresolved workers before dispatching more", s.Config.MaxConcurrent)
	}
	if err = s.localServer(ctx); err != nil {
		return Record{}, err
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		return Record{}, fmt.Errorf("reserve worker directory: %w", err)
	}
	if err = writeJSON(filepath.Join(dir, "request.json"), req); err != nil {
		return Record{}, err
	}
	executable := s.WorkerExecutable
	if executable == "" {
		executable, err = os.Executable()
		if err != nil {
			return Record{}, err
		}
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return Record{}, err
	}
	common := []string{"-c", req.Cwd, "--shell=none", "--keep-open", "--json", "--", executable, "worker", "--state-root", s.Root, "--id", req.ID}
	var args []string
	if s.Config.Session == "" {
		args = append([]string{"new", "pstack-" + req.ID}, common...)
	} else {
		args = append([]string{"run", "-s", s.Config.Session, "--focus=false", "--label", "pstack-" + req.ID}, common...)
	}
	data, launchErr := s.rex(ctx, args...)
	launch := Launch{}
	if launchErr != nil {
		launch.Error = launchErr.Error()
		launch.Uncertain = true
	} else {
		var response struct {
			Session string   `json:"session_id"`
			Blocks  []string `json:"block_ids"`
			Windows []struct {
				Blocks []string `json:"block_ids"`
			} `json:"initial_windows"`
		}
		if err = json.Unmarshal(data, &response); err != nil {
			launch.Error = "invalid Rex creation response"
			launch.Uncertain = true
		} else {
			launch.Session = response.Session
			if launch.Session == "" {
				launch.Session = s.Config.Session
			}
			if len(response.Blocks) > 0 {
				launch.Block = response.Blocks[0]
			} else if len(response.Windows) > 0 && len(response.Windows[0].Blocks) > 0 {
				launch.Block = response.Windows[0].Blocks[0]
			}
			if launch.Block == "" || launch.Session == "" {
				launch.Error = "Rex creation response lacks stable identities"
				launch.Uncertain = true
			}
		}
	}
	if err = writeJSON(filepath.Join(dir, "launch.json"), launch); err != nil {
		return Record{}, err
	}
	return s.Inspect(ctx, req.ID)
}

func (s *Service) Inspect(ctx context.Context, id string) (Record, error) {
	dir, err := s.dir(id)
	if err != nil {
		return Record{}, err
	}
	var req Request
	if err = readJSON(filepath.Join(dir, "request.json"), &req); err != nil {
		return Record{}, err
	}
	record := Record{ID: id, Status: "starting", Role: req.Role, Target: req.Target, Artifacts: dir}
	var launch Launch
	if err = readJSON(filepath.Join(dir, "launch.json"), &launch); err == nil {
		record.Launch = &launch
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, err
	}
	var started Started
	if err = readJSON(filepath.Join(dir, "started.json"), &started); err == nil {
		record.Started = &started
		record.Status = "running"
		if record.Launch == nil || record.Launch.Block == "" {
			record.Launch = &Launch{Session: started.Session, Block: started.Block}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, err
	}
	var result Result
	if err = readJSON(filepath.Join(dir, "result.json"), &result); err == nil {
		if !terminal(result.Status) {
			return Record{}, fmt.Errorf("invalid terminal result status %q", result.Status)
		}
		record.Result = &result
		record.Status = result.Status
		return record, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, err
	}
	if record.Launch == nil {
		record.Status = "unknown"
		record.Error = "launch receipt missing; do not retry automatically"
		return record, nil
	}
	if record.Launch.Block == "" {
		record.Status = "unknown"
		record.Error = record.Launch.Error
		return record, nil
	}
	data, err := s.rex(ctx, "block", "call", "-s", record.Launch.Session, "-b", record.Launch.Block, "process")
	if err != nil {
		record.Status = "unknown"
		record.Error = err.Error()
		return record, nil
	}
	var process struct {
		LastExit *struct {
			Code int `json:"exit_code"`
		} `json:"last_exit"`
	}
	if err = json.Unmarshal(data, &process); err != nil {
		return Record{}, fmt.Errorf("invalid Rex process response: %w", err)
	}
	if process.LastExit != nil {
		record.Status = "unknown"
		record.Error = fmt.Sprintf("supervisor exited (%d) without a terminal result; child termination is unconfirmed and capacity remains reserved", process.LastExit.Code)
	} else {
		record.Status = "running"
	}
	return record, nil
}

func (s *Service) List(ctx context.Context) ([]Record, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, "workers"))
	if errors.Is(err, os.ErrNotExist) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		record, err := s.Inspect(ctx, entry.Name())
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *Service) Cancel(ctx context.Context, id string) (Record, error) {
	dir, err := s.dir(id)
	if err != nil {
		return Record{}, err
	}
	record, err := s.Inspect(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if terminal(record.Status) {
		return record, nil
	}
	if err = writeJSON(filepath.Join(dir, "cancel.json"), map[string]any{"requested_at": time.Now().UTC()}); err != nil {
		return Record{}, err
	}
	record, err = s.Wait(ctx, id, 5*time.Second)
	if err == nil && !terminal(record.Status) {
		record.Error = "cancellation requested; worker termination is not confirmed"
	}
	return record, err
}

func (s *Service) Wait(ctx context.Context, id string, duration time.Duration) (Record, error) {
	if duration <= 0 || duration > 60*time.Second {
		return Record{}, fmt.Errorf("wait must be greater than zero and at most 60 seconds")
	}
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		record, err := s.Inspect(ctx, id)
		if err != nil {
			return Record{}, err
		}
		if terminal(record.Status) {
			return record, nil
		}
		select {
		case <-ctx.Done():
			return record, ctx.Err()
		case <-deadline.C:
			return record, nil
		case <-tick.C:
		}
	}
}
