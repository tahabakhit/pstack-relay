// Package discover queries installed harnesses without making inference requests.
package discover

import (
	"encoding/json"
	"fmt"
	"github.com/tahabakhit/pstack-rex/internal/sheet"
	"strings"
)

type Harness = sheet.Harness
type Catalog struct {
	Harness   Harness  `json:"harness"`
	Installed bool     `json:"installed"`
	Models    []string `json:"models"`
	Note      string   `json:"note,omitempty"`
}

func Discover(run func(name string, args ...string) (string, error)) []Catalog {
	var catalogs []Catalog
	for _, h := range []Harness{sheet.Codex, sheet.Claude, sheet.Pi} {
		c := Catalog{Harness: h, Models: []string{}}
		if _, err := run(string(h), "--version"); err != nil {
			c.Note = "not available: " + err.Error()
			catalogs = append(catalogs, c)
			continue
		}
		c.Installed = true
		switch h {
		case sheet.Claude:
			c.Models = []string{"opus", "sonnet", "haiku", "fable"}
			c.Note = "aliases; Claude has no model list command"
		case sheet.Codex:
			output, err := run("codex", "debug", "models", "--bundled")
			if err == nil {
				var data struct {
					Models []struct {
						Slug string `json:"slug"`
					} `json:"models"`
				}
				err = json.Unmarshal([]byte(output), &data)
				if err == nil {
					for _, m := range data.Models {
						if m.Slug != "" {
							c.Models = append(c.Models, m.Slug)
						}
					}
				}
			}
			if err != nil {
				c.Note = fmt.Sprintf("model discovery failed: %v", err)
			}
		case sheet.Pi:
			output, err := run("pi", "--offline", "--list-models")
			if err != nil {
				c.Note = fmt.Sprintf("model discovery failed: %v", err)
			} else {
				c.Models = ParsePi(output)
			}
		}
		if len(c.Models) == 0 && c.Note == "" {
			c.Note = "model discovery returned no models"
		}
		catalogs = append(catalogs, c)
	}
	return catalogs
}

// ParsePi accepts Pi's provider/model table and one provider/model id per line.
func ParsePi(output string) []string {
	models := []string{}
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "Warning:") || strings.EqualFold(fields[0], "provider") {
			continue
		}
		id := ""
		if strings.Contains(fields[0], "/") {
			id = fields[0]
		} else if len(fields) >= 2 && !strings.HasPrefix(fields[0], "-") {
			id = fields[0] + "/" + fields[1]
		}
		if id != "" && !seen[id] {
			models = append(models, id)
			seen[id] = true
		}
	}
	return models
}

// PreferConfigured moves the model named by Codex's top-level `model = "..."`
// setting to the front of the Codex catalog, so setup starts from the model
// the user already runs instead of the catalog's first and costliest entry.
func PreferConfigured(catalogs []Catalog, codexConfig string) {
	want := topLevelModel(codexConfig)
	for i := range catalogs {
		if catalogs[i].Harness != sheet.Codex {
			continue
		}
		models := catalogs[i].Models
		for j, m := range models {
			if m == want && j > 0 {
				catalogs[i].Models = append([]string{m}, append(append([]string{}, models[:j]...), models[j+1:]...)...)
			}
		}
	}
}

// topLevelModel reads the top-level TOML key `model` without a TOML library.
// It tracks multi-line arrays and strings so that a `[` inside a value is not
// taken for a table header.
func topLevelModel(config string) string {
	depth, triple := 0, ""
	for _, raw := range strings.Split(config, "\n") {
		line := strings.TrimSpace(raw)
		if triple != "" {
			if strings.Contains(line, triple) {
				triple = ""
			}
			continue
		}
		if depth > 0 {
			depth += bracketDelta(line)
			continue
		}
		if strings.HasPrefix(line, "[") {
			return ""
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.Trim(strings.TrimSpace(key), `"'`)
		value = strings.TrimSpace(value)
		for _, q := range []string{`"""`, "'''"} {
			if strings.HasPrefix(value, q) && !strings.Contains(value[3:], q) {
				triple = q
			}
		}
		if triple != "" {
			continue
		}
		if strings.HasPrefix(value, "[") {
			depth = bracketDelta(value)
			continue
		}
		if key == "model" {
			return tomlString(value)
		}
	}
	return ""
}

func bracketDelta(s string) int {
	d, quote := 0, rune(0)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '#':
			return d
		case r == '[':
			d++
		case r == ']':
			d--
		}
	}
	return d
}

func tomlString(value string) string {
	if value == "" || (value[0] != '"' && value[0] != '\'') {
		return ""
	}
	q := value[0]
	var b strings.Builder
	for i := 1; i < len(value); i++ {
		c := value[i]
		switch {
		case c == q:
			return b.String()
		case c == '\\' && q == '"' && i+1 < len(value):
			i++
			b.WriteByte(value[i])
		default:
			b.WriteByte(c)
		}
	}
	return ""
}
