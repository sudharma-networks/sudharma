package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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

func forbiddenBindCalls(file *ast.File) []string {
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		name := pkg.Name + "." + selector.Sel.Name
		switch name {
		case "net.Listen", "tls.Listen", "http.Serve", "http.ListenAndServe", "http.ListenAndServeTLS":
			found = append(found, name)
		}
		return true
	})
	return found
}

func socketOwningHelpers(file *ast.File) []string {
	var found []string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name == nil {
			continue
		}
		if strings.HasPrefix(function.Name.Name, "ListenAndServe") {
			found = append(found, function.Name.Name)
		}
	}
	return found
}

func assertServerSourceHasNoPortBinding(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	files := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if calls := forbiddenBindCalls(file); len(calls) != 0 {
			t.Fatalf("%s contains forbidden listener calls: %v", name, calls)
		}
		if helpers := socketOwningHelpers(file); len(helpers) != 0 {
			t.Fatalf("%s exposes forbidden socket-owning helpers: %v", name, helpers)
		}
	}
}
