package transport

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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

func forbiddenListenerCalls(file *ast.File) []string {
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

func declaresNetListener(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if ok && pkg.Name == "net" && selector.Sel.Name == "Listener" {
			found = true
			return false
		}
		return true
	})
	return found
}

func TestTransportHasNoListener(t *testing.T) {
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
		if calls := forbiddenListenerCalls(file); len(calls) != 0 {
			t.Fatalf("%s contains forbidden listener calls: %v", name, calls)
		}
		if declaresNetListener(file) {
			t.Fatalf("%s declares or references net.Listener", name)
		}
	}
}
