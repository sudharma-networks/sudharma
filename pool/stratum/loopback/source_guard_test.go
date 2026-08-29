package loopback

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLoopbackGuardDetectsUnsafeListenAddress(t *testing.T) {
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"unsafe.go",
		"package loopback\nimport \"net\"\nfunc Listen() (net.Listener, error) { return net.Listen(\"tcp\", \":3333\") }",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := listenerViolations(file)
	if len(violations) == 0 {
		t.Fatal("unsafe public/listen address was not detected")
	}
}

func TestLoopbackGuardDetectsConfigurableAddressSource(t *testing.T) {
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"unsafe.go",
		"package loopback\nimport (\"net\"; \"os\")\nfunc Listen() (net.Listener, error) { return net.Listen(\"tcp4\", os.Getenv(\"STRATUM_ADDR\")) }",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := listenerViolations(file)
	if len(violations) == 0 {
		t.Fatal("environment-selected listener address was not detected")
	}
}

func TestLoopbackGuardDetectsParameterizedExport(t *testing.T) {
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"unsafe.go",
		"package loopback\nimport \"net\"\nfunc Listen(address string) (net.Listener, error) { return net.Listen(\"tcp4\", address) }",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := exportedListenViolations(file)
	if len(violations) == 0 {
		t.Fatal("parameterized exported Listen API was not detected")
	}
}

func TestLoopbackProductionSourceIsFixedAndLocalOnly(t *testing.T) {
	assertProductionLoopbackOnly(t)
}

func listenerViolations(file *ast.File) []string {
	constants := stringConstants(file)
	var violations []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if pkg, ok := selector.X.(*ast.Ident); ok {
			name := pkg.Name + "." + selector.Sel.Name
			switch name {
			case "net.Listen":
				if len(call.Args) != 2 {
					violations = append(violations, "net.Listen must have exactly two arguments")
					return true
				}
				network, networkOK := resolveString(call.Args[0], constants)
				address, addressOK := resolveString(call.Args[1], constants)
				if !networkOK || network != loopbackNetwork {
					violations = append(violations, "net.Listen network must resolve to tcp4")
				}
				if !addressOK || address != loopbackAddress {
					violations = append(violations, "net.Listen address must resolve to 127.0.0.1:0")
				}
			case "os.Getenv", "os.LookupEnv", "flag.String", "flag.StringVar", "net.ResolveTCPAddr":
				violations = append(violations, "configurable listener input: "+name)
			}
			return true
		}

		// A method-form Listen call can hide net.ListenConfig.Listen behind a
		// local variable. Stage G does not need any method-form socket owner.
		if selector.Sel.Name == "Listen" {
			violations = append(violations, "method-form Listen call is forbidden")
		}
		return true
	})
	return violations
}

func exportedListenViolations(file *ast.File) []string {
	var violations []string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name == nil || !ast.IsExported(function.Name.Name) {
			continue
		}
		if !strings.HasPrefix(function.Name.Name, "Listen") {
			continue
		}
		if function.Name.Name != "Listen" {
			violations = append(violations, "unexpected exported listener helper: "+function.Name.Name)
		}
		if function.Type.Params != nil && len(function.Type.Params.List) != 0 {
			violations = append(violations, function.Name.Name+" must have zero parameters")
		}
	}
	return violations
}

func assertProductionLoopbackOnly(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	files := token.NewFileSet()
	listenCalls := 0
	exportedListenFunctions := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if violations := listenerViolations(file); len(violations) != 0 {
			t.Fatalf("%s violates loopback-only listener policy: %v", name, violations)
		}
		if violations := exportedListenViolations(file); len(violations) != 0 {
			t.Fatalf("%s violates exported listener API policy: %v", name, violations)
		}
		listenCalls += countDirectNetListenCalls(file)
		exportedListenFunctions += countExportedListenFunctions(file)
	}
	if listenCalls != 1 {
		t.Fatalf("production net.Listen calls = %d, want exactly 1", listenCalls)
	}
	if exportedListenFunctions != 1 {
		t.Fatalf("production exported Listen functions = %d, want exactly 1", exportedListenFunctions)
	}
}

func stringConstants(file *ast.File) map[string]string {
	constants := make(map[string]string)
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			valueSpec, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				if value, ok := resolveString(valueSpec.Values[i], constants); ok {
					constants[name.Name] = value
				}
			}
		}
	}
	return constants
}

func resolveString(expression ast.Expr, constants map[string]string) (string, bool) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		if value.Kind != token.STRING {
			return "", false
		}
		decoded, err := strconv.Unquote(value.Value)
		if err != nil {
			return "", false
		}
		return decoded, true
	case *ast.Ident:
		resolved, ok := constants[value.Name]
		return resolved, ok
	default:
		return "", false
	}
}

func countDirectNetListenCalls(file *ast.File) int {
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Listen" {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if ok && pkg.Name == "net" {
			count++
		}
		return true
	})
	return count
}

func countExportedListenFunctions(file *ast.File) int {
	count := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name != nil && function.Name.Name == "Listen" && ast.IsExported(function.Name.Name) {
			count++
		}
	}
	return count
}
