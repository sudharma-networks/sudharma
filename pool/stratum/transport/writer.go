package transport

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum"
)

type messageWriter struct {
	mu           sync.Mutex
	conn         net.Conn
	writeTimeout time.Duration
}

func (w *messageWriter) WriteMessages(messages []stratum.Message) error {
	if len(messages) == 0 {
		return nil
	}

	var encoded bytes.Buffer
	for _, message := range messages {
		line, err := stratum.EncodeMessage(message)
		if err != nil {
			return err
		}
		_, _ = encoded.Write(line)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)); err != nil {
		return err
	}
	return writeAll(w.conn, encoded.Bytes())
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
