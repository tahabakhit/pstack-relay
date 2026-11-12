package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/pstack-rex/pstack-rex/internal/engine"
)

func exchange(t *testing.T, s *engine.Service, input string) map[string]json.RawMessage {
	t.Helper()
	var out bytes.Buffer
	if err := Serve(strings.NewReader(input), &out, s); err != nil {
		t.Fatal(err)
	}
	responses := map[string]json.RawMessage{}
	decoder := json.NewDecoder(&out)
	for decoder.More() {
		var message struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := decoder.Decode(&message); err != nil {
			t.Fatal(err)
		}
		if len(message.Error) > 0 {
			responses[string(message.ID)] = message.Error
		} else {
			responses[string(message.ID)] = message.Result
		}
	}
	return responses
}

func TestStdioProtocolAndToolParity(t *testing.T) {
	s := &engine.Service{Root: t.TempDir()}
	responses := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"ping"}
{"jsonrpc":"2.0","id":3,"method":"tools/list"}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"capabilities","arguments":{}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list","arguments":{}}}
`)
	if len(responses) != 5 {
		t.Fatalf("response count=%d: %v", len(responses), responses)
	}
	var initialized struct {
		Protocol string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(responses["1"], &initialized); err != nil || initialized.Protocol == "" {
		t.Fatalf("initialize=%s err=%v", responses["1"], err)
	}
	if string(responses["2"]) != "{}" {
		t.Fatalf("ping=%s", responses["2"])
	}
	var listed struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(responses["3"], &listed); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range listed.Tools {
		names[tool.Name] = true
	}
	if !reflect.DeepEqual(names, map[string]bool{"dispatch": true, "inspect": true, "list": true, "wait": true, "cancel": true, "capabilities": true}) {
		t.Fatalf("tools=%v", names)
	}
	for _, tc := range []struct {
		id   string
		want any
	}{{"4", Capabilities()}, {"5", []engine.Record{}}} {
		var result struct {
			Error   bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(responses[tc.id], &result); err != nil {
			t.Fatal(err)
		}
		if result.Error || len(result.Content) != 1 {
			t.Fatalf("tool=%s", responses[tc.id])
		}
		want, _ := json.Marshal(tc.want)
		if result.Content[0].Text != string(want) {
			t.Fatalf("tool %s content=%s want=%s", tc.id, result.Content[0].Text, want)
		}
	}
	direct, err := s.List(context.Background())
	if err != nil || len(direct) != 0 {
		t.Fatalf("engine parity=%v %v", direct, err)
	}
}

func TestMalformedCallsAndUnsupportedMethods(t *testing.T) {
	s := &engine.Service{Root: t.TempDir()}
	for _, tc := range []struct{ name, args string }{
		{"dispatch", `{"id":"attempt","target":{"harness":"codex"}}`},
		{"dispatch", `{"seat":"zero"}`},
		{"dispatch", `{"seat":0.5}`},
		{"dispatch", `[]`},
		{"inspect", `{"id":"../escape"}`},
		{"wait", `{"id":"attempt","seconds":61}`},
		{"wait", `{"id":"attempt","seconds":-1}`},
		{"list", `{"shell":"echo unsafe"}`},
		{"capabilities", `{"unknown":true}`},
		{"steer", `{}`},
	} {
		t.Run(tc.name+tc.args, func(t *testing.T) {
			request := fmtRequest(tc.name, tc.args)
			responses := exchange(t, s, request)
			var result struct {
				Error bool `json:"isError"`
			}
			if err := json.Unmarshal(responses["1"], &result); err != nil || !result.Error {
				t.Fatalf("invalid call accepted: %s err=%v", responses["1"], err)
			}
		})
	}
	responses := exchange(t, s, "not-json\n"+`{"jsonrpc":"2.0","id":7,"method":"unknown"}`+"\n")
	if !strings.Contains(string(responses["null"]), "-32700") || !strings.Contains(string(responses["7"]), "-32601") {
		t.Fatalf("errors=%v", responses)
	}
}

func fmtRequest(name, args string) string {
	data, _ := json.Marshal(name)
	return `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":` + string(data) + `,"arguments":` + args + `}}` + "\n"
}
