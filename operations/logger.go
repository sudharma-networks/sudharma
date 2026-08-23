package operations

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

type Logger struct {
	out  io.Writer
	json bool
	mu   sync.Mutex
}

func NewLogger(out io.Writer, jsonOutput bool) *Logger {
	return &Logger{out: out, json: jsonOutput}
}

func (l *Logger) Info(event string, fields map[string]any) {
	l.write("info", event, fields)
}

func (l *Logger) Error(event string, fields map[string]any) {
	l.write("error", event, fields)
}

func (l *Logger) write(level, event string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.json {
		record := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "level": level, "event": event}
		for k, v := range fields {
			record[k] = v
		}
		_ = json.NewEncoder(l.out).Encode(record)
		return
	}
	fmt.Fprintf(l.out, "%s level=%s event=%s", time.Now().UTC().Format(time.RFC3339), level, event)
	for k, v := range fields {
		fmt.Fprintf(l.out, " %s=%v", k, v)
	}
	fmt.Fprintln(l.out)
}
