package server

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestForbiddenBindMatcherDetectsListenerCreation(t *testing.T) {
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"forbidden.go",
		"package server\nimport \"net\"\nfunc open() { _, _ = net.Listen(\"tcp\", \":1234\") }",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	got := forbiddenBindCalls(file)
	if len(got) != 1 || got[0] != "net.Listen" {
		t.Fatalf("forbidden calls = %v, want [net.Listen]", got)
	}
}

func TestForbiddenBindMatcherDetectsSocketOwningHelper(t *testing.T) {
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"forbidden.go",
		"package server\nfunc ListenAndServePool() {}",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	got := socketOwningHelpers(file)
	if len(got) != 1 || got[0] != "ListenAndServePool" {
		t.Fatalf("socket-owning helpers = %v, want [ListenAndServePool]", got)
	}
}

func TestServerHasNoPortBindingPrimitive(t *testing.T) {
	assertServerSourceHasNoPortBinding(t)
}
