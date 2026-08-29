package stratum

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodeRequest(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		method string
		id     string
	}{
		{
			name:   "string id subscribe",
			input:  []byte(`{"id":"req-1","method":"mining.subscribe","params":[]}`),
			method: "mining.subscribe",
			id:     `"req-1"`,
		},
		{
			name:   "integer id authorize",
			input:  []byte(`{"id":7,"method":"mining.authorize","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","x"]}`),
			method: "mining.authorize",
			id:     `7`,
		},
		{
			name:   "negative integer id submit",
			input:  []byte(`{"id":-2,"method":"mining.submit","params":["w","j","00"]}`),
			method: "mining.submit",
			id:     `-2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeRequest(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Method != tt.method {
				t.Fatalf("method = %q, want %q", got.Method, tt.method)
			}
			if string(got.ID) != tt.id {
				t.Fatalf("id = %s, want %s", got.ID, tt.id)
			}
			if len(got.Params) == 0 || got.Params[0] != '[' {
				t.Fatalf("params not preserved as array: %s", got.Params)
			}
		})
	}
}

func TestDecodeRequestRejectsInvalidInput(t *testing.T) {
	const (
		parseErrorCode     = -32700
		invalidRequestCode = -32600
		methodNotFoundCode = -32601
		invalidParamsCode  = -32602
	)
	oversized := append([]byte(`{"id":1,"method":"mining.subscribe","params":["`), bytes.Repeat([]byte("a"), maxMessageBytes)...)
	oversized = append(oversized, []byte(`"]}`)...)

	tests := []struct {
		name string
		data []byte
		code int
	}{
		{"empty", nil, parseErrorCode},
		{"malformed utf8", []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}, parseErrorCode},
		{"batch", []byte(`[{"id":1,"method":"mining.subscribe","params":[]}]`), invalidRequestCode},
		{"duplicate top key", []byte(`{"id":1,"id":2,"method":"mining.subscribe","params":[]}`), invalidRequestCode},
		{"duplicate nested key", []byte(`{"id":1,"method":"mining.subscribe","params":[{"x":1,"x":2}]}`), invalidRequestCode},
		{"unknown top field", []byte(`{"id":1,"method":"mining.subscribe","params":[],"jsonrpc":"2.0"}`), invalidRequestCode},
		{"fractional id", []byte(`{"id":1.5,"method":"mining.subscribe","params":[]}`), invalidRequestCode},
		{"exponent id", []byte(`{"id":1e3,"method":"mining.subscribe","params":[]}`), invalidRequestCode},
		{"null id", []byte(`{"id":null,"method":"mining.subscribe","params":[]}`), invalidRequestCode},
		{"boolean id", []byte(`{"id":true,"method":"mining.subscribe","params":[]}`), invalidRequestCode},
		{"unknown method", []byte(`{"id":1,"method":"mining.extranonce.subscribe","params":[]}`), methodNotFoundCode},
		{"trailing json", []byte(`{"id":1,"method":"mining.subscribe","params":[]} {}`), parseErrorCode},
		{"missing params", []byte(`{"id":1,"method":"mining.subscribe"}`), invalidParamsCode},
		{"object params", []byte(`{"id":1,"method":"mining.subscribe","params":{}}`), invalidParamsCode},
		{"oversized", oversized, invalidRequestCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeRequest(tt.data)
			if err == nil {
				t.Fatal("expected error")
			}
			var protocolErr *ProtocolError
			if !errors.As(err, &protocolErr) {
				t.Fatalf("error type = %T, want *ProtocolError: %v", err, err)
			}
			if protocolErr.Code != tt.code {
				t.Fatalf("code = %d, want %d (%v)", protocolErr.Code, tt.code, protocolErr)
			}
		})
	}
}

func TestEncodeMessage(t *testing.T) {
	got, err := EncodeMessage(Response{ID: json.RawMessage(`7`), Result: true})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":7,"result":true,"error":null}` + "\n"
	if string(got) != want {
		t.Fatalf("encoded = %q, want %q", got, want)
	}
	if !strings.HasSuffix(string(got), "\n") || strings.HasSuffix(string(got), "\n\n") {
		t.Fatalf("message must end in exactly one newline: %q", got)
	}
}
