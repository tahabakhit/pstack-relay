package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogLimitPreservesBoundedPartialOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stream")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := limitedLog{file: file, left: 3}
	n, err := writer.Write([]byte("12345"))
	if n != 3 || err == nil {
		t.Fatalf("n=%d err=%v", n, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "123" {
		t.Fatalf("retained=%q err=%v", data, err)
	}
	n, err = writer.Write([]byte("more"))
	if n != 0 || err == nil {
		t.Fatalf("overflow resumed n=%d err=%v", n, err)
	}
}
