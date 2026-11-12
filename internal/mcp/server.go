package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pstack-rex/pstack-rex/internal/engine"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func Capabilities() map[string]any {
	return map[string]any{"backends": []string{"rex"}, "native": false, "steer": false, "resume": false, "worker_mcp": false, "writer_harnesses": []string{"codex"}, "auth_modes": []string{"existing"}, "same_host_required": true, "completion_is_verification": false, "model_fallback": false}
}

func schema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": append([]string{}, required...), "additionalProperties": false}
}
func stringField(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func tools() []map[string]any {
	return []map[string]any{
		{"name": "dispatch", "description": "Start one configured role seat in Rex. Use a stable unique idempotency key. Does not fall back or mark reviews verified.", "inputSchema": schema(map[string]any{"id": stringField("Immutable operation key, 1-80 letters/digits/underscore/hyphen"), "role": stringField("Configured pstack role"), "seat": map[string]any{"type": "integer", "minimum": 0}, "cwd": stringField("Absolute Git checkout on local Rex server host"), "prompt": stringField("Complete task brief including required skill and file paths"), "access": map[string]any{"type": "string", "enum": []string{"read-only", "isolated-write"}}, "timeout_seconds": map[string]any{"type": "integer", "minimum": 1}, "require_mcp": map[string]any{"type": "boolean"}, "expected_revision": stringField("Exact starting Git SHA")}, "id", "role", "cwd", "prompt")},
		{"name": "inspect", "description": "Inspect worker receipts and Rex process. Missing completion is never success.", "inputSchema": schema(map[string]any{"id": stringField("Worker id")}, "id")},
		{"name": "list", "description": "List all retained worker attempts including unresolved launches.", "inputSchema": schema(map[string]any{})},
		{"name": "wait", "description": "Wait up to 60 seconds; pending or unknown status is not success.", "inputSchema": schema(map[string]any{"id": stringField("Worker id"), "seconds": map[string]any{"type": "integer", "minimum": 1, "maximum": 60}}, "id")},
		{"name": "cancel", "description": "Request cancellation and wait for termination. Unknown status retains ownership and artifacts.", "inputSchema": schema(map[string]any{"id": stringField("Worker id")}, "id")},
		{"name": "capabilities", "description": "Return actual backend and worker profile limitations.", "inputSchema": schema(map[string]any{})},
	}
}

func strict(data []byte, value any) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("tool arguments must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON object")
	}
	return nil
}

func call(ctx context.Context, s *engine.Service, name string, args json.RawMessage) (any, error) {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	if name == "dispatch" {
		var input struct {
			ID         string `json:"id"`
			Role       string `json:"role"`
			Seat       int    `json:"seat"`
			Cwd        string `json:"cwd"`
			Prompt     string `json:"prompt"`
			Access     string `json:"access"`
			Timeout    int    `json:"timeout_seconds"`
			RequireMCP bool   `json:"require_mcp"`
			Revision   string `json:"expected_revision"`
		}
		if err := strict(args, &input); err != nil {
			return nil, err
		}
		return s.Dispatch(ctx, engine.Request{ID: input.ID, Role: input.Role, Seat: input.Seat, Cwd: input.Cwd, Prompt: input.Prompt, Access: input.Access, TimeoutSeconds: input.Timeout, RequireMCP: input.RequireMCP, ExpectedRevision: input.Revision})
	}
	if name == "capabilities" || name == "list" {
		var input struct{}
		if err := strict(args, &input); err != nil {
			return nil, err
		}
		if name == "capabilities" {
			return Capabilities(), nil
		}
		return s.List(ctx)
	}
	if name != "inspect" && name != "wait" && name != "cancel" {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	if name == "inspect" || name == "cancel" {
		var input struct {
			ID string `json:"id"`
		}
		if err := strict(args, &input); err != nil {
			return nil, err
		}
		if name == "inspect" {
			return s.Inspect(ctx, input.ID)
		}
		return s.Cancel(ctx, input.ID)
	}
	var input struct {
		ID      string `json:"id"`
		Seconds int    `json:"seconds"`
	}
	if err := strict(args, &input); err != nil {
		return nil, err
	}
	switch name {
	case "inspect":
		return s.Inspect(ctx, input.ID)
	case "cancel":
		return s.Cancel(ctx, input.ID)
	default:
		if input.Seconds == 0 {
			input.Seconds = 30
		}
		return s.Wait(ctx, input.ID, time.Duration(input.Seconds)*time.Second)
	}
}

func Serve(in io.Reader, out io.Writer, service *engine.Service) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mutex sync.Mutex
	write := func(value any) { mutex.Lock(); defer mutex.Unlock(); json.NewEncoder(out).Encode(value) }
	var workers sync.WaitGroup
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var request rpcRequest
		if err := json.Unmarshal(line, &request); err != nil {
			write(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32700, "message": "invalid JSON"}})
			continue
		}
		if len(request.ID) == 0 {
			continue
		}
		workers.Add(1)
		go func(req rpcRequest) {
			defer workers.Done()
			response := map[string]any{"jsonrpc": "2.0", "id": req.ID}
			var result any
			var rpcErr error
			switch req.Method {
			case "initialize":
				result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]any{"name": "pstack-rex", "version": "0.1.0"}, "capabilities": map[string]any{"tools": map[string]any{}}}
			case "ping":
				result = map[string]any{}
			case "tools/list":
				result = map[string]any{"tools": tools()}
			case "tools/call":
				var params struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}
				if rpcErr = json.Unmarshal(req.Params, &params); rpcErr == nil {
					value, err := call(ctx, service, params.Name, params.Arguments)
					content := ""
					if err != nil {
						content = err.Error()
					} else {
						data, e := json.Marshal(value)
						if e != nil {
							err = e
							content = e.Error()
						} else {
							content = string(data)
						}
					}
					result = map[string]any{"content": []map[string]any{{"type": "text", "text": content}}, "isError": err != nil}
				}
			default:
				rpcErr = fmt.Errorf("unsupported method %s", req.Method)
			}
			if rpcErr != nil {
				response["error"] = map[string]any{"code": -32601, "message": rpcErr.Error()}
			} else {
				response["result"] = result
			}
			write(response)
		}(request)
	}
	workers.Wait()
	return scanner.Err()
}
