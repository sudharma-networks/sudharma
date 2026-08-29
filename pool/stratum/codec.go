package stratum

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	protocolParseError     = -32700
	protocolInvalidRequest = -32600
	protocolMethodNotFound = -32601
	protocolInvalidParams  = -32602
)

var errDuplicateJSONKey = errors.New("duplicate JSON object key")

type Request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Message interface {
	isStratumMessage()
}

type Response struct {
	ID     json.RawMessage `json:"id"`
	Result any             `json:"result"`
	Error  *ProtocolError  `json:"error"`
}

func (Response) isStratumMessage() {}

type Notification struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params any             `json:"params"`
}

func (Notification) isStratumMessage() {}

type ProtocolError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Message
}

func DecodeRequest(data []byte) (Request, error) {
	if len(data) == 0 || !utf8.Valid(data) {
		return Request{}, newProtocolError(protocolParseError)
	}
	if len(data) > maxMessageBytes {
		return Request{}, newProtocolError(protocolInvalidRequest)
	}

	root, err := validateJSONStructure(data)
	if err != nil {
		if errors.Is(err, errDuplicateJSONKey) {
			return Request{}, newProtocolError(protocolInvalidRequest)
		}
		return Request{}, newProtocolError(protocolParseError)
	}
	if root != '{' {
		return Request{}, newProtocolError(protocolInvalidRequest)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	var request Request
	if err := dec.Decode(&request); err != nil {
		return Request{}, newProtocolError(protocolInvalidRequest)
	}
	if err := expectJSONEOF(dec); err != nil {
		return Request{}, newProtocolError(protocolParseError)
	}
	if !validRequestID(request.ID) || request.Method == "" {
		return Request{}, newProtocolError(protocolInvalidRequest)
	}

	switch request.Method {
	case "mining.subscribe", "mining.authorize", "mining.submit":
	default:
		return Request{}, newProtocolError(protocolMethodNotFound)
	}

	params := bytes.TrimSpace(request.Params)
	if len(params) == 0 || params[0] != '[' {
		return Request{}, newProtocolError(protocolInvalidParams)
	}
	var values []json.RawMessage
	if err := json.Unmarshal(params, &values); err != nil {
		return Request{}, newProtocolError(protocolInvalidParams)
	}

	return request, nil
}

func EncodeMessage(message any) ([]byte, error) {
	encoded, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("encode stratum message: %w", err)
	}
	encoded = append(encoded, '\n')
	return encoded, nil
}

func newProtocolError(code int) *ProtocolError {
	switch code {
	case protocolParseError:
		return &ProtocolError{Code: code, Message: "parse error"}
	case protocolInvalidRequest:
		return &ProtocolError{Code: code, Message: "invalid request"}
	case protocolMethodNotFound:
		return &ProtocolError{Code: code, Message: "method not found"}
	case protocolInvalidParams:
		return &ProtocolError{Code: code, Message: "invalid params"}
	default:
		return &ProtocolError{Code: code, Message: "protocol error"}
	}
}

func validRequestID(raw json.RawMessage) bool {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 {
		return false
	}
	if value[0] == '"' {
		var id string
		return json.Unmarshal(value, &id) == nil
	}

	i := 0
	if value[0] == '-' {
		i++
	}
	if i == len(value) {
		return false
	}
	for ; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func validateJSONStructure(data []byte) (json.Delim, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	token, err := dec.Token()
	if err != nil {
		return 0, err
	}
	root, ok := token.(json.Delim)
	if !ok || (root != '{' && root != '[') {
		return 0, errors.New("top-level JSON value must be an object or array")
	}
	if err := consumeContainer(dec, root); err != nil {
		return 0, err
	}
	if err := expectJSONEOF(dec); err != nil {
		return 0, err
	}
	return root, nil
}

func consumeContainer(dec *json.Decoder, open json.Delim) error {
	switch open {
	case '{':
		seen := make(map[string]struct{})
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return errDuplicateJSONKey
			}
			seen[key] = struct{}{}
			if err := consumeValue(dec); err != nil {
				return err
			}
		}
		closing, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := closing.(json.Delim); !ok || delim != '}' {
			return errors.New("object not closed")
		}
	case '[':
		for dec.More() {
			if err := consumeValue(dec); err != nil {
				return err
			}
		}
		closing, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := closing.(json.Delim); !ok || delim != ']' {
			return errors.New("array not closed")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

func consumeValue(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := token.(json.Delim); ok {
		if delim != '{' && delim != '[' {
			return errors.New("unexpected closing JSON delimiter")
		}
		return consumeContainer(dec, delim)
	}
	return nil
}

func expectJSONEOF(dec *json.Decoder) error {
	var trailing any
	if err := dec.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("trailing JSON value")
}
