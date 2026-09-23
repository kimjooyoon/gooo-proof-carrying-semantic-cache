package protocol

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCorpusRejectsTrailingJSONValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corpus.json")
	raw := []byte(`{"schema":"gooo.corpus/v1","corpus_id":"fixture","cases":[{"id":"case-1"}]}` + "\n{}\n")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadCorpus(path); err == nil {
		t.Fatal("LoadCorpus accepted a trailing JSON value")
	}
}
