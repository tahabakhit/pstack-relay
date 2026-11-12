package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pstack-rex/pstack-rex/internal/config"
)

type Plan struct {
	Executable  string
	Args        []string
	Stdin       string
	Environment []string
	Enforcement string
}
type Outcome struct {
	Text          string `json:"text"`
	ReportedModel string `json:"reported_model,omitempty"`
	ModelEvidence string `json:"model_evidence"`
	SessionID     string `json:"session_id,omitempty"`
}

func Build(t config.Target, access, cwd, prompt string) (Plan, error) {
	if err := config.ValidateTarget(t, false); err != nil {
		return Plan{}, err
	}
	if access == "" {
		access = "read-only"
	}
	if access != "read-only" && access != "isolated-write" {
		return Plan{}, fmt.Errorf("unsupported access %q", access)
	}
	if access == "isolated-write" && t.Harness != "codex" {
		return Plan{}, fmt.Errorf("only Codex supports isolated-write")
	}
	exe, err := exec.LookPath(t.Harness)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Executable: exe, Stdin: prompt, Environment: environment(t.Harness, t.Model), Enforcement: "tool-restricted"}
	effort := t.Effort != "" && t.Effort != "default"
	switch t.Harness {
	case "codex":
		sandbox := "read-only"
		if access == "isolated-write" {
			sandbox = "workspace-write"
		}
		p.Enforcement = "sandbox-" + sandbox
		p.Args = []string{"-a", "never", "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--cd", cwd, "--sandbox", sandbox, "--model", t.Model, "--json", "--disable", "hooks", "--disable", "plugins", "--disable", "multi_agent", "--disable", "multi_agent_v2", "--disable", "memories", "--disable", "apps", "--disable", "skill_mcp_dependency_install", "--enable", "skip_host_skill_discovery", "-c", "mcp_servers={}", "-c", "project_doc_max_bytes=0"}
		if effort {
			p.Args = append(p.Args, "-c", "model_reasoning_effort="+strconv.Quote(t.Effort))
		}
		p.Args = append(p.Args, "-")
	case "claude":
		p.Args = []string{"--safe-mode", "--restricted", "-p", "--model", t.Model, "--output-format", "stream-json", "--verbose", "--tools", "Read,Grep,Glob", "--permission-mode", "dontAsk", "--permission-prompts", "none", "--strict-mcp-config"}
		if effort {
			p.Args = append(p.Args, "--effort", t.Effort)
		}
	case "pi":
		provider, model, ok := strings.Cut(t.Model, "/")
		if !ok || provider == "" || model == "" || strings.ContainsAny(model, "*?:") {
			return Plan{}, fmt.Errorf("Pi requires an explicit provider/model without patterns or thinking suffix")
		}
		p.Args = []string{"--offline", "--mode", "json", "--print", "--no-session", "--no-extensions", "--no-mcp", "--no-skills", "--no-context-files", "--no-prompt-templates", "--tools", "read,grep,find,ls", "--model", t.Model}
		if effort {
			p.Args = append(p.Args, "--thinking", t.Effort)
		}
		// Pi consumes positional text; -- prevents prompts from becoming options.
		p.Args = append(p.Args, "--", prompt)
		p.Stdin = ""
	}
	return p, nil
}
func environment(h, model string) []string {
	common := map[string]bool{"HOME": true, "PATH": true, "USER": true, "LOGNAME": true, "SHELL": true, "LANG": true, "LC_ALL": true, "LC_CTYPE": true, "TMPDIR": true, "TZ": true, "XDG_CONFIG_HOME": true, "XDG_DATA_HOME": true, "XDG_CACHE_HOME": true, "SSL_CERT_FILE": true, "SSL_CERT_DIR": true}
	auth := map[string][]string{
		"codex":  {"CODEX_HOME", "OPENAI_API_KEY", "OPENAI_ORG_ID", "OPENAI_ORGANIZATION", "OPENAI_PROJECT_ID"},
		"claude": {"CLAUDE_CONFIG_DIR", "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"},
		"pi":     {"PI_CODING_AGENT_DIR"},
	}
	if h == "pi" {
		provider, _, ok := strings.Cut(model, "/")
		if !ok {
			provider = ""
		}
		auth["pi"] = []string{"PI_CODING_AGENT_DIR"}
		keys := map[string][]string{"openai": {"OPENAI_API_KEY"}, "openai-codex": {"OPENAI_API_KEY"}, "anthropic": {"ANTHROPIC_API_KEY"}, "google": {"GEMINI_API_KEY", "GOOGLE_API_KEY"}, "groq": {"GROQ_API_KEY"}, "mistral": {"MISTRAL_API_KEY"}, "openrouter": {"OPENROUTER_API_KEY"}, "xai": {"XAI_API_KEY"}}
		auth["pi"] = append(auth["pi"], keys[provider]...)
	}
	for _, k := range auth[h] {
		common[k] = true
	}
	var out []string
	for _, s := range os.Environ() {
		k, _, _ := strings.Cut(s, "=")
		if common[k] {
			out = append(out, s)
		}
	}
	return out
}

type event struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"`
	IsError   bool            `json:"is_error"`
	Error     json.RawMessage `json:"error"`
	Result    string          `json:"result"`
	SessionID string          `json:"session_id"`
	ThreadID  string          `json:"thread_id"`
	Aborted   *bool           `json:"aborted"`
	Item      struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
	Message message `json:"message"`
}
type message struct {
	Role          string          `json:"role"`
	Model         string          `json:"model"`
	ResponseModel string          `json:"responseModel"`
	Provider      string          `json:"provider"`
	StopReason    string          `json:"stopReason"`
	Content       json.RawMessage `json:"content"`
}

func textContent(m message) string {
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(m.Content, &blocks) != nil {
		return ""
	}
	var s strings.Builder
	for _, c := range blocks {
		if c.Type == "text" {
			s.WriteString(c.Text)
		}
	}
	return s.String()
}

func matches(h, want, got string) bool {
	if got == want {
		return true
	}
	if h == "claude" && (want == "opus" || want == "sonnet" || want == "haiku") {
		return strings.HasPrefix(got, "claude-"+want+"-")
	}
	if h == "pi" {
		_, id, ok := strings.Cut(want, "/")
		return ok && id == got
	}
	return false
}
func Parse(t config.Target, stdout []byte) (Outcome, error) {
	if err := config.ValidateTarget(t, false); err != nil {
		return Outcome{}, err
	}
	o := Outcome{ModelEvidence: "pinned-argv"}
	terminal := false
	sawMessage := false
	piStop := ""
	lines := bytes.Split(stdout, []byte{'\n'})
	for i, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if terminal {
			return Outcome{}, fmt.Errorf("event after terminal success at line %d", i+1)
		}
		var e event
		if err := json.Unmarshal(line, &e); err != nil || e.Type == "" {
			return Outcome{}, fmt.Errorf("invalid JSON event at line %d", i+1)
		}
		if e.IsError || (len(e.Error) > 0 && string(e.Error) != "null") || e.Type == "error" || e.Type == "turn.failed" {
			return Outcome{}, fmt.Errorf("worker reported error at line %d", i+1)
		}
		switch t.Harness {
		case "codex":
			switch e.Type {
			case "thread.started":
				o.SessionID = e.ThreadID
			case "item.completed":
				if e.Item.Type == "error" {
					return Outcome{}, fmt.Errorf("Codex error item")
				}
				if e.Item.Type == "agent_message" {
					o.Text = e.Item.Text
					sawMessage = true
				}
			case "turn.completed":
				terminal = true
			}
		case "claude":
			if e.SessionID != "" {
				o.SessionID = e.SessionID
			}
			if e.Type == "assistant" {
				sawMessage = true
				if e.Message.Model != "" {
					if !matches(t.Harness, t.Model, e.Message.Model) {
						return Outcome{}, fmt.Errorf("model mismatch: requested %q, reported %q", t.Model, e.Message.Model)
					}
					o.ReportedModel = e.Message.Model
					o.ModelEvidence = "assistant.message.model"
				}
			}
			if e.Type == "result" {
				if e.Subtype != "success" {
					return Outcome{}, fmt.Errorf("Claude terminal result %q", e.Subtype)
				}
				o.Text = e.Result
				terminal = true
			}
		case "pi":
			if e.Type == "message_end" && e.Message.Role == "assistant" {
				sawMessage = true
				piStop = e.Message.StopReason
				if piStop != "stop" && piStop != "toolUse" {
					return Outcome{}, fmt.Errorf("Pi stop reason %q", e.Message.StopReason)
				}
				o.Text = textContent(e.Message)
				if e.Message.Provider != "" {
					provider, _, ok := strings.Cut(t.Model, "/")
					if ok && provider != e.Message.Provider {
						return Outcome{}, fmt.Errorf("provider mismatch: requested %q, reported %q", provider, e.Message.Provider)
					}
				}
				model := e.Message.ResponseModel
				evidence := "message.responseModel"
				if model == "" {
					model = e.Message.Model
					evidence = "message.model-selected"
				}
				if model != "" {
					if !matches(t.Harness, t.Model, model) {
						return Outcome{}, fmt.Errorf("model mismatch: requested %q, reported %q", t.Model, model)
					}
					o.ReportedModel = model
					o.ModelEvidence = evidence
				}
			}
			if e.Type == "agent_settled" {
				if piStop != "stop" {
					return Outcome{}, fmt.Errorf("Pi settled without final stop")
				}
				if e.Aborted == nil || *e.Aborted {
					return Outcome{}, fmt.Errorf("Pi aborted")
				}
				terminal = true
			}
		}
	}
	if !terminal || !sawMessage || strings.TrimSpace(o.Text) == "" {
		return Outcome{}, fmt.Errorf("missing terminal success or final assistant text")
	}
	return o, nil
}
