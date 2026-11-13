package relaytest

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tahabakhit/pstack-rex/internal/discover"
	"github.com/tahabakhit/pstack-rex/internal/setup"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
)

// This capture-only oracle is removed once the shell and captured expectations agree.
func TestCaptureTemplateOracle(t *testing.T) {
	if os.Getenv("RELAY_CAPTURE") != "1" {
		t.Skip("capture requested explicitly")
	}
	wants := map[string]expected{}
	for _, c := range templateCases() {
		var cats []discover.Catalog
		if c.Mode == "" || c.Mode == "only-codex" {
			cats = append(cats, discover.Catalog{Harness: sheet.Codex, Installed: true, Models: []string{"gpt-first", "gpt-next", "c"}})
		}
		if c.Mode == "" || c.Mode == "only-claude" {
			cats = append(cats, discover.Catalog{Harness: sheet.Claude, Installed: true, Models: []string{"opus", "sonnet", "haiku", "fable"}})
		}
		if c.Mode == "" || c.Mode == "only-pi" {
			cats = append(cats, discover.Catalog{Harness: sheet.Pi, Installed: true, Models: []string{"openai/gpt-a"}})
		}
		s, e := setup.Starter(c.Template, cats)
		want := expected{Error: e != nil}
		if e == nil {
			want.Output = s.Format()
		}
		wants[c.Name] = want
		got := templateRun(t, c)
		if got.Error != want.Error || got.Output != want.Output {
			t.Fatalf("%s: got %+v want %+v", c.Name, got, want)
		}
	}
	b, e := json.MarshalIndent(wants, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile("testdata/template-expectations.json", append(b, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
	t.Logf("%d template cases matched Go", len(wants))
}
