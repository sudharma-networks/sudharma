package transport

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadRequestLineLF(t *testing.T) {
	r := newRequestReader(strings.NewReader("abc\n"))
	got, err := readRequestLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abc" {
		t.Fatalf("line = %q, want abc", got)
	}
}

func TestReadRequestLineCRLF(t *testing.T) {
	r := newRequestReader(strings.NewReader("abc\r\n"))
	got, err := readRequestLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abc" {
		t.Fatalf("line = %q, want abc", got)
	}
}

func TestReadRequestLineProcessesBoundedFinalFragment(t *testing.T) {
	r := newRequestReader(strings.NewReader("abc"))
	got, err := readRequestLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abc" {
		t.Fatalf("line = %q, want abc", got)
	}
	if _, err := readRequestLine(r); !errors.Is(err, io.EOF) {
		t.Fatalf("second read error = %v, want EOF", err)
	}
}

func TestReadRequestLineCleanEOF(t *testing.T) {
	r := newRequestReader(strings.NewReader(""))
	if _, err := readRequestLine(r); !errors.Is(err, io.EOF) {
		t.Fatalf("error = %v, want EOF", err)
	}
}

func TestReadRequestLineAcceptsExactLimitLF(t *testing.T) {
	content := strings.Repeat("x", 64*1024)
	r := newRequestReader(strings.NewReader(content + "\n"))
	got, err := readRequestLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 64*1024 {
		t.Fatalf("line length = %d, want %d", len(got), 64*1024)
	}
}

func TestReadRequestLineAcceptsExactLimitCRLF(t *testing.T) {
	content := strings.Repeat("x", 64*1024)
	r := newRequestReader(strings.NewReader(content + "\r\n"))
	got, err := readRequestLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 64*1024 {
		t.Fatalf("line length = %d, want %d", len(got), 64*1024)
	}
}

func TestReadRequestLineRejectsOverLimit(t *testing.T) {
	r := newRequestReader(strings.NewReader(strings.Repeat("x", 64*1024+1) + "\n"))
	if _, err := readRequestLine(r); !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("error = %v, want ErrLineTooLong", err)
	}
}

func TestReadRequestLineRejectsOverlongUnterminatedFragment(t *testing.T) {
	r := newRequestReader(strings.NewReader(strings.Repeat("x", 64*1024+3)))
	if _, err := readRequestLine(r); !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("error = %v, want ErrLineTooLong", err)
	}
}
