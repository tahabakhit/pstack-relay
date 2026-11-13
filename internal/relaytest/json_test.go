package relaytest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLargeJSONScan(t *testing.T) {
	dir := t.TempDir()
	const count = 6000
	models := make([]string, count)
	for i := range models {
		models[i] = fmt.Sprintf(`{"slug":"model-%d","description":%q,"active":true,"values":[null,1e2,false]}`, i, strings.Repeat("catalog data ", 5))
	}
	data := "{\"models\":[" + strings.Join(models, ",\n") + "]}\n"
	path := filepath.Join(dir, "models.json")
	write(t, path, data)
	for _, tc := range []struct{ action, field, raw, want string }{
		{"codexvalidate", "", "", ""},
		{"json", "models.6000.slug", "false", "model-5999"},
		{"json", "models.6000", "true", models[count-1] + "\n"},
	} {
		cmd := exec.Command("awk", "-f", filepath.Join(rootpath(t), "overlay/lib/relay.awk"))
		cmd.Env = []string{"LC_ALL=C", "input=" + path, "action=" + tc.action, "field=" + tc.field, "raw=" + tc.raw}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil || string(out) != tc.want {
			t.Fatalf("%s %s: bytes=%d want=%d err=%v stderr=%s", tc.action, tc.field, len(out), len(tc.want), err, stderr.String())
		}
		if tc.raw == "true" && !json.Valid(out) {
			t.Fatal("raw JSON field is invalid")
		}
	}
	t.Logf("validated %d model records in %d bytes", count, len(data))
}

func TestJSONQuoteActions(t *testing.T) {
	text := "quotes \" and \\ and Unicode é"
	for c := byte(1); c < 32; c++ {
		text += string(c)
	}
	for _, action := range []string{"quote", "quotefile"} {
		dir := t.TempDir()
		want := text
		env := []string{"action=" + action, "LC_ALL=C"}
		if action == "quote" {
			env = append(env, "text="+text)
		} else {
			want = strings.Repeat(text, 3000) + "\n\n"
			path := filepath.Join(dir, "report.md")
			write(t, path, want)
			env = append(env, "input="+path, fmt.Sprintf("bytes=%d", len(want)))
		}
		cmd := exec.Command("awk", "-f", filepath.Join(rootpath(t), "overlay/lib/relay.awk"))
		cmd.Env = env
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		var got string
		if err := json.Unmarshal(out, &got); err != nil || got != want {
			t.Fatalf("%s changed text: bytes=%d want=%d err=%v", action, len(got), len(want), err)
		}
	}
}
