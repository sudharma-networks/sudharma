package transport

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestForbiddenListenerMatcher(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "forbidden.go", "package transport\nimport \"net\"\nfunc listen() { _, _ = net.Listen(\"tcp\", \":1234\") }", 0)
	if err != nil {
		t.Fatal(err)
	}
	got := forbiddenListenerCalls(file)
	if len(got) != 1 || got[0] != "net.Listen" {
		t.Fatalf("forbidden calls = %v, want [net.Listen]", got)
	}
}
