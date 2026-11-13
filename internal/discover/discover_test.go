package discover

import (
	"errors"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDiscover(t *testing.T) {
	for _, tt := range []struct {
		name      string
		responses map[string]string
		want      []Catalog
	}{
		{"all", map[string]string{"codex --version": "1", "claude --version": "2", "pi --version": "3", "codex debug models --bundled": `{"models":[{"slug":"gpt-a"},{"slug":"gpt-b"}]}`, "pi --offline --list-models": "provider model context\nopenai gpt-a 400K"}, []Catalog{{sheet.Codex, true, []string{"gpt-a", "gpt-b"}, ""}, {sheet.Claude, true, []string{"opus", "sonnet", "haiku", "fable"}, "aliases; Claude has no model list command"}, {sheet.Pi, true, []string{"openai/gpt-a"}, ""}}},
		{"missing", nil, nil},
		{"bad JSON", map[string]string{"codex --version": "1", "codex debug models --bundled": "not json"}, nil},
		{"empty models", map[string]string{"codex --version": "1", "codex debug models --bundled": `{}`}, nil},
		{"failed lists", map[string]string{"codex --version": "1", "pi --version": "1"}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Discover(func(name string, args ...string) (string, error) {
				v, ok := tt.responses[name+" "+strings.Join(args, " ")]
				if !ok {
					return "", errors.New("unavailable")
				}
				return v, nil
			})
			if tt.want != nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("got %#v want %#v", got, tt.want)
				}
				return
			}
			for _, c := range got {
				if c.Note == "" {
					t.Fatalf("missing failure detail: %#v", c)
				}
				if tt.name == "missing" && c.Installed {
					t.Fatal("missing binary installed")
				}
			}
		})
	}
}
func TestParsePi(t *testing.T) {
	sample, err := os.ReadFile("testdata/pi-models.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, input string
		want        []string
	}{
		{"table", "provider model context\nopenai gpt-a 400K\nopenai gpt-a 400K\nanthropic opus 200K", []string{"openai/gpt-a", "anthropic/opus"}},
		{"ids and warnings", "Warning: No models match pattern\nopenai/gpt-a\nanthropic/opus\n", []string{"openai/gpt-a", "anthropic/opus"}},
		{"empty", "", []string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsePi(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
		})
	}
	got := ParsePi(string(sample))
	if len(got) < 2 || got[0] != "openai/chat-latest" || got[len(got)-1] != "openai/o4-mini" {
		t.Fatalf("captured Pi 1.5.2 sample: %#v", got)
	}
}

func TestPreferConfigured(t *testing.T) {
	catalogs := []Catalog{{Harness: sheet.Codex, Installed: true, Models: []string{"a", "b", "c"}}, {Harness: sheet.Claude, Models: []string{"opus"}}}
	PreferConfigured(catalogs, "model = \"c\"\n[profiles.x]\nmodel = \"b\"\n")
	if strings.Join(catalogs[0].Models, ",") != "c,a,b" || catalogs[1].Models[0] != "opus" {
		t.Fatalf("got %v", catalogs)
	}
	PreferConfigured(catalogs, "model = \"missing\"")
	if catalogs[0].Models[0] != "c" {
		t.Fatalf("unknown model reordered catalog: %v", catalogs[0].Models)
	}
}
