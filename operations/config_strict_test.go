package operations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"node_id":`), 0600); err != nil { t.Fatal(err) }
	if _, err := LoadConfig(path); err == nil { t.Fatal("expected malformed JSON error") }
}
