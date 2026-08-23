package operations

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLoggerJSON(t *testing.T) {
	var out bytes.Buffer
	NewLogger(&out, true).Info("started", map[string]any{"height": 7})
	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["event"] != "started" || record["level"] != "info" || record["height"].(float64) != 7 {
		t.Fatalf("unexpected record: %#v", record)
	}
}

func TestLoggerText(t *testing.T) {
	var out bytes.Buffer
	NewLogger(&out, false).Error("persist_failed", map[string]any{"component": "chain"})
	text := out.String()
	if !strings.Contains(text, "level=error") || !strings.Contains(text, "event=persist_failed") {
		t.Fatalf("unexpected log: %s", text)
	}
}
