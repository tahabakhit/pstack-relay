package plugin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"testing/fstest"
)

func TestTransform(t *testing.T) {
	source := []byte("---\nname: example\n---\n\nOriginal body\n")
	original := bytes.Clone(source)
	rules := []Adaptation{{Path: "skills/example/SKILL.md", Old: "Original", Replacement: "Adapted"}}
	got := Transform("skills/example/SKILL.md", source, rules)
	if !bytes.HasPrefix(got, []byte("---\nname: example\n---\n")) || !bytes.Contains(got, []byte("../../delegation.md")) || !bytes.Contains(got, []byte("Adapted body")) {
		t.Fatalf("unexpected generated entrypoint: %s", got)
	}
	if !bytes.Equal(source, original) {
		t.Fatal("mutated pristine input")
	}
	for _, path := range []string{"agents/reviewer.md", "automations/tool/FOR_AGENTS.md", "SKILL.md"} {
		if !bytes.Contains(Transform(path, []byte("body"), nil), []byte("delegation.md")) {
			t.Errorf("missing pointer: %s", path)
		}
	}
	if got := Transform("references/guide.md", source, rules); !bytes.Equal(got, source) {
		t.Fatal("changed a non-entrypoint")
	}
}

func TestValidateAdaptations(t *testing.T) {
	source := fstest.MapFS{"file.md": {Data: []byte("original")}}
	sum := sha256.Sum256(source["file.md"].Data)
	rules := []Adaptation{{Path: "file.md", Old: "original", Replacement: "adapted", SourceSHA256: hex.EncodeToString(sum[:])}}
	if err := ValidateAdaptations(source, rules); err != nil {
		t.Fatal(err)
	}
	source["file.md"].Data = []byte("changed")
	if err := ValidateAdaptations(source, rules); err == nil {
		t.Fatal("source drift accepted")
	}
	rules[0].Path = "missing"
	if err := ValidateAdaptations(source, rules); err == nil {
		t.Fatal("missing source accepted")
	}
}

func TestAdaptationsJSON(t *testing.T) {
	data, err := os.ReadFile("adaptations.json")
	if err != nil {
		t.Fatal(err)
	}
	var rules []Adaptation
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules) == 0 {
		t.Fatal("missing adaptations")
	}
	for _, rule := range rules {
		if rule.Path == "" || rule.Old == "" || len(rule.SourceSHA256) != 64 {
			t.Fatalf("incomplete adaptation for %s", rule.Path)
		}
	}
}
