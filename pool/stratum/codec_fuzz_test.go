package stratum

import (
	"bytes"
	"encoding/json"
	"testing"
	"unicode/utf8"
)

func FuzzDecodeRequest(f *testing.F) {
	f.Add([]byte(`{"id":1,"method":"mining.subscribe","params":[]}`))
	f.Add([]byte(`{"id":"a","method":"mining.authorize","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","x"]}`))
	f.Add([]byte(`{"id":2,"method":"mining.submit","params":["worker","job","00"]}`))
	f.Add([]byte(`{"id":1,"id":2,"method":"mining.subscribe","params":[]}`))
	f.Add(append(bytes.Repeat([]byte(" "), maxMessageBytes), 'x'))
	f.Add([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})
	f.Add([]byte(`{"id":1,"method":"mining.subscribe","params":[]} {}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := DecodeRequest(data)
		if !utf8.Valid(data) && err == nil {
			t.Fatal("invalid UTF-8 decoded successfully")
		}
		if hasMultipleJSONValues(data) && err == nil {
			t.Fatal("multiple JSON values decoded successfully")
		}
	})
}

func hasMultipleJSONValues(data []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(data))
	var first any
	if err := dec.Decode(&first); err != nil {
		return false
	}
	var second any
	return dec.Decode(&second) == nil
}
