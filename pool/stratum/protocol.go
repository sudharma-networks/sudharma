package stratum

import (
	"encoding/json"
	"fmt"
)

const ProtocolVersion = "sudharma-stratum/1.0.0"

// Request is a line-delimited JSON-RPC request from a miner.
type Request struct {
	ID     any    `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

// Response is a JSON-RPC response to a miner.
type Response struct {
	ID     any            `json:"id"`
	Result any            `json:"result,omitempty"`
	Error  *ResponseError `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Notification is an unsolicited JSON-RPC message from pool to miner.
type Notification struct {
	ID     any    `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

func EncodeResponse(id any, result any) ([]byte, error) {
	payload, err := json.Marshal(Response{ID: id, Result: result})
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')
	return payload, nil
}

func EncodeError(id any, code int, message string) ([]byte, error) {
	payload, err := json.Marshal(Response{
		ID: id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
		},
	})
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')
	return payload, nil
}

func EncodeNotification(method string, params ...any) ([]byte, error) {
	payload, err := json.Marshal(Notification{
		ID:     nil,
		Method: method,
		Params: params,
	})
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')
	return payload, nil
}

func DecodeRequest(line []byte) (Request, error) {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return Request{}, fmt.Errorf("invalid stratum JSON: %w", err)
	}
	if req.Method == "" {
		return Request{}, fmt.Errorf("missing stratum method")
	}
	return req, nil
}

func ParamString(params []any, index int) (string, error) {
	if index >= len(params) {
		return "", fmt.Errorf("missing stratum param %d", index)
	}
	switch v := params[index].(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprint(v), nil
	}
}

func ParamUint64(params []any, index int) (uint64, error) {
	raw, err := ParamString(params, index)
	if err != nil {
		return 0, err
	}
	var value uint64
	if _, err := fmt.Sscan(raw, &value); err != nil {
		return 0, fmt.Errorf("invalid nonce %q", raw)
	}
	return value, nil
}
