package sheet

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct{ name, text, problem string }{
		{"group and panel", "# comment\nfeature, refactoring: claude:opus @high, codex:gpt-6.1-sol @xhigh\ndefault effort: session\nsession hook: off\n", ""},
		{"aliases", "role: inherit-parent, auto, pi:openai/gpt-6.1-terra @medium", ""},
		{"all efforts", "role: m @low, m @medium, m @high, m @xhigh, m @max, m @session", ""},
		{"empty", "\n# only comment", ""},
		{"unknown harness", "# hi\nrole: cursor:model", "line 2: unknown harness cursor"},
		{"unknown effort", "role: claude:opus @ultra", "line 1: unknown effort"},
		{"unknown default effort", "default effort: ultra", "line 1: unknown effort"},
		{"invalid hook", "session hook: yes", "line 1: session hook"},
		{"empty value", "role:", "line 1:"},
		{"missing separator", "role", "line 1:"},
		{"empty role", "role, : model", "line 1:"},
		{"empty panel seat", "role: opus,", "line 1:"},
		{"empty model", "role: codex:", "line 1:"},
		{"extra colon", "role: codex:a:b", "line 1:"},
		{"effort without at", "role: opus high", "line 1:"},
		{"extra fields", "role: opus @high extra", "line 1:"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Parse(tt.text)
			if tt.problem != "" {
				if err == nil || !strings.Contains(err.Error(), tt.problem) {
					t.Fatalf("error=%v want %q", err, tt.problem)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			again, err := Parse(s.Format())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s, again) {
				t.Fatalf("round trip changed sheet: %#v -> %#v", s, again)
			}
			if tt.name == "group and panel" {
				if len(s.Roles["feature"]) != 2 || !reflect.DeepEqual(s.Roles["feature"], s.Roles["refactoring"]) || s.SessionHook || s.DefaultEffort != "session" {
					t.Fatalf("unexpected sheet: %#v", s)
				}
				s.Roles["feature"][0].Model = "changed"
				if s.Roles["refactoring"][0].Model != "opus" {
					t.Fatal("group roles share mutable seats")
				}
			}
		})
	}
}
func TestPath(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"home", map[string]string{"HOME": "/home/a"}, "/home/a/.config/pstack-rex/pstack-models.md"},
		{"xdg", map[string]string{"HOME": "/a", "XDG_CONFIG_HOME": "/config"}, "/config/pstack-rex/pstack-models.md"},
		{"override", map[string]string{"PSTACK_REX_SHEET": "/custom.md", "XDG_CONFIG_HOME": "/config"}, "/custom.md"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Path(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
func TestFormatUnorderedRoles(t *testing.T) {
	s := Sheet{Roles: map[string][]Seat{"z": {{Model: "auto"}}, "a": {{Model: "auto"}}}}
	if !strings.HasPrefix(s.Format(), "a: auto\nz: auto\n") {
		t.Fatal(s.Format())
	}
}
