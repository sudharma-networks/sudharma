package transport

import (
	"bufio"
	"errors"
	"io"
)

func newRequestReader(reader io.Reader) *bufio.Reader {
	return bufio.NewReaderSize(reader, maxRequestBytes+2)
}

func readRequestLine(reader *bufio.Reader) ([]byte, error) {
	data, err := reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return nil, ErrLineTooLong
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(data) == 0 && errors.Is(err, io.EOF) {
		return nil, io.EOF
	}

	if data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	if len(data) > 0 && data[len(data)-1] == '\r' {
		data = data[:len(data)-1]
	}
	if len(data) > maxRequestBytes {
		return nil, ErrLineTooLong
	}

	line := append([]byte(nil), data...)
	return line, nil
}
