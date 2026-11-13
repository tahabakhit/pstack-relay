package setup

import (
	pstackrex "github.com/tahabakhit/pstack-rex"
	"github.com/tahabakhit/pstack-rex/internal/discover"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
	"strings"
	"testing"
)

func TestStarter(t *testing.T) {
	for _, tt := range []struct {
		name     string
		catalogs []discover.Catalog
		want     []sheet.Seat
		err      bool
	}{
		{"both", []discover.Catalog{{Harness: sheet.Claude, Installed: true, Models: []string{"opus"}}, {Harness: sheet.Codex, Installed: true, Models: []string{"gpt-first", "gpt-next"}}}, []sheet.Seat{{Harness: sheet.Claude, Model: "opus", Effort: "xhigh"}, {Harness: sheet.Codex, Model: "gpt-first", Effort: "high"}}, false},
		{"claude only", []discover.Catalog{{Harness: sheet.Claude, Installed: true, Models: []string{"opus"}}}, []sheet.Seat{{Harness: sheet.Claude, Model: "opus", Effort: "xhigh"}, {Harness: sheet.Claude, Model: "opus", Effort: "high"}}, false},
		{"codex only", []discover.Catalog{{Harness: sheet.Codex, Installed: true, Models: []string{"gpt-first"}}}, []sheet.Seat{{Harness: sheet.Codex, Model: "gpt-first", Effort: "xhigh"}, {Harness: sheet.Codex, Model: "gpt-first", Effort: "high"}}, false},
		{"pi only", []discover.Catalog{{Harness: sheet.Pi, Installed: true, Models: []string{"openai/gpt-first"}}}, []sheet.Seat{{Harness: sheet.Pi, Model: "openai/gpt-first", Effort: "xhigh"}, {Harness: sheet.Pi, Model: "openai/gpt-first", Effort: "high"}}, false},
		{"none", nil, nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			template := "prose\n```\n---\ndescription: example\n---\nfeature, refactoring: grok-fast\nnew role: claude-opus-5-5-xhigh, grok-fast\n```\n"
			s, err := Starter(template, tt.catalogs)
			if tt.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s.DefaultEffort != "session" || len(s.Roles) != 3 || len(s.Roles["new role"]) != 2 {
				t.Fatalf("lost upstream shape: %#v", s)
			}
			for i, seat := range s.Roles["new role"] {
				if seat != tt.want[i] {
					t.Fatalf("seat %d got %#v want %#v", i, seat, tt.want[i])
				}
			}
		})
	}
}
func TestRealUpstreamTemplate(t *testing.T) {
	s, err := Starter(pstackrex.SetupTemplate, []discover.Catalog{{Harness: sheet.Codex, Installed: true, Models: []string{"gpt"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Roles["interrogate reviewers"]) != 2 || s.Roles["feature"][0].Effort != "xhigh" {
		t.Fatalf("unexpected upstream mapping: %#v", s)
	}
}
func TestMissingTemplate(t *testing.T) {
	_, err := Starter("no block", nil)
	if err == nil || !strings.Contains(err.Error(), "role block") {
		t.Fatal(err)
	}
}
