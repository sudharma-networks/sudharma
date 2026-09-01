package stratum

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// Client connects a GPU miner to a Sudharma Stratum v1 pool.
type Client struct {
	address  string
	password string
	conn     net.Conn
	reader   *bufio.Reader
	mu       sync.Mutex
	nextID   int
}

func Dial(ctx context.Context, poolURL, login, password string) (*Client, error) {
	host, err := ParsePoolURL(poolURL)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("stratum dial failed: %w", err)
	}

	client := &Client{
		address:  strings.TrimSpace(login),
		password: password,
		conn:     conn,
		reader:   bufio.NewReader(conn),
	}

	if err := client.handshake(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Login() string {
	if c == nil {
		return ""
	}
	return c.address
}

func (c *Client) handshake(ctx context.Context) error {
	if err := c.call(ctx, "mining.subscribe", []any{c.address}); err != nil {
		return err
	}
	return c.call(ctx, "mining.authorize", []any{c.address, c.password})
}

func (c *Client) SubmitShare(ctx context.Context, jobID string, nonce uint64) error {
	return c.call(ctx, "mining.submit", []any{c.address, jobID, fmt.Sprintf("%d", nonce)})
}

func (c *Client) NextJob(ctx context.Context) (Job, error) {
	for {
		if err := ctx.Err(); err != nil {
			return Job{}, err
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return Job{}, err
		}
		var msg struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Method != "mining.notify" {
			continue
		}
		return ParseNotify(msg.Params)
	}
}

func (c *Client) call(ctx context.Context, method string, params []any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id := c.nextRequestID()
	payload, err := json.Marshal(map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	})
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.conn.Write(payload); err != nil {
		return err
	}

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		var resp struct {
			ID    any `json:"id"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Result any `json:"result"`
		}
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if resp.ID == nil {
			continue
		}
		if fmt.Sprint(resp.ID) != fmt.Sprint(id) {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("stratum %s failed: %s", method, resp.Error.Message)
		}
		return nil
	}
}

func (c *Client) nextRequestID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	return c.nextID
}

// ParsePoolURL accepts stratum+tcp://host:port or host:port.
func ParsePoolURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("stratum pool URL is required")
	}
	trimmed = strings.TrimPrefix(trimmed, "stratum+tcp://")
	trimmed = strings.TrimPrefix(trimmed, "tcp://")
	if !strings.Contains(trimmed, ":") {
		return "", fmt.Errorf("stratum pool URL must include host:port")
	}
	return trimmed, nil
}

// ServeConn supports tests with an in-memory connection pair.
func ServeConn(ctx context.Context, conn net.Conn, login, password string) (*Client, error) {
	client := &Client{
		address:  strings.TrimSpace(login),
		password: password,
		conn:     conn,
		reader:   bufio.NewReader(conn),
	}
	if err := client.handshake(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func WriteNotification(conn net.Conn, method string, params ...any) error {
	payload, err := json.Marshal(map[string]any{
		"id":     nil,
		"method": method,
		"params": params,
	})
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = conn.Write(payload)
	return err
}

func ReadRequest(reader *bufio.Reader) (string, []any, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return "", nil, err
	}
	var req struct {
		Method string `json:"method"`
		Params []any  `json:"params"`
	}
	if err := json.Unmarshal(line, &req); err != nil {
		return "", nil, err
	}
	return req.Method, req.Params, nil
}

func WriteResponse(conn net.Conn, id int, result any) error {
	payload, err := json.Marshal(map[string]any{"id": id, "result": result})
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = io.WriteString(conn, string(payload))
	return err
}
