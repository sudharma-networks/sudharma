package loopback

import (
	"go/parser"
	"go/token"
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
