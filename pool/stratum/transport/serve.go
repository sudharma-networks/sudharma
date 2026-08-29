package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/sudharma-networks/sudharma/pool/stratum"
)

func ServeConn(ctx context.Context, conn net.Conn, factory SessionFactory, config Config) error {
	if conn == nil || factory == nil {
		return ErrInvalidConfig
	}
	if _, err := normalizeConfig(config); err != nil {
		return err
	}
	defer conn.Close()

	session, err := factory()
	if err != nil {
		return fmt.Errorf("create Stratum session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("create Stratum session: %w", ErrInvalidConfig)
	}

	reader := newRequestReader(conn)
	writer := &messageWriter{conn: conn}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		line, err := readRequestLine(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read Stratum request: %w", err)
		}

		request, _ := stratum.DecodeRequest(line)
		messages, err := session.Handle(ctx, line)
		if err != nil {
			var protocolErr *stratum.ProtocolError
			if !errors.As(err, &protocolErr) {
				return fmt.Errorf("handle Stratum request: %w", err)
			}
			response := stratum.Response{
				ID:     json.RawMessage("null"),
				Result: nil,
				Error:  protocolErr,
			}
			if err := writer.WriteMessages([]stratum.Message{response}); err != nil {
				return fmt.Errorf("write Stratum response: %w", err)
			}
			continue
		}

		if err := writer.WriteMessages(messages); err != nil {
			return fmt.Errorf("write Stratum response: %w", err)
		}
		if request.Method == "mining.authorize" && authorizationSucceeded(messages) {
			workMessages, err := session.RefreshWork(ctx)
			if err != nil {
				return fmt.Errorf("refresh Stratum work: %w", err)
			}
			if err := writer.WriteMessages(workMessages); err != nil {
				return fmt.Errorf("write Stratum response: %w", err)
			}
		}
	}
}

func authorizationSucceeded(messages []stratum.Message) bool {
	if len(messages) != 1 {
		return false
	}
	response, ok := messages[0].(stratum.Response)
	if !ok || response.Error != nil {
		return false
	}
	accepted, ok := response.Result.(bool)
	return ok && accepted
}
