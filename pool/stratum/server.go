package stratum

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/pool"
)

type Engine interface {
	Config() pool.Config
	CurrentJob() pool.Job
	RefreshWork(ctx context.Context) (pool.Job, error)
	SubmitShare(ctx context.Context, job pool.Job, worker pool.WorkerIdentity, nonce uint64) (pool.ShareResult, pool.ShareCredit, error)
}

type Server struct {
	engine Engine
	logf   func(string, ...any)
}

func NewServer(engine Engine, logf func(string, ...any)) *Server {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Server{engine: engine, logf: logf}
}

func (s *Server) ListenAndServe(ctx context.Context, address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	return s.Serve(ctx, listener)
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	if listener == nil {
		return fmt.Errorf("stratum listener is nil")
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	s.logf("stratum listening on %s scheme=%s", listener.Addr(), s.engine.Config().PayoutScheme)

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(root context.Context, conn net.Conn) {
	defer conn.Close()

	sessionCtx, cancel := context.WithCancel(root)
	defer cancel()

	reader := bufio.NewReader(conn)
	writer := &connWriter{conn: conn}
	extranonce1 := randomHex(4)
	subscribed := false
	authorized := false
	var worker pool.WorkerIdentity

	for {
		if err := sessionCtx.Err(); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.logf("stratum read error: %v", err)
			}
			return
		}

		req, err := DecodeRequest(line)
		if err != nil {
			_ = writer.WriteError(nil, 20, err.Error())
			continue
		}

		switch req.Method {
		case "mining.subscribe":
			subscribed = true
			result := []any{
				[][]any{{"mining.notify", extranonce1, ProtocolVersion}},
				extranonce1,
				4,
			}
			_ = writer.WriteResponse(req.ID, result)
		case "mining.authorize":
			login, err := ParamString(req.Params, 0)
			if err != nil {
				_ = writer.WriteError(req.ID, 24, err.Error())
				continue
			}
			worker, err = pool.ParseWorkerIdentity(login)
			if err != nil {
				_ = writer.WriteError(req.ID, 24, err.Error())
				continue
			}
			authorized = true
			_ = writer.WriteResponse(req.ID, true)
			if subscribed {
				if err := s.pushJob(sessionCtx, writer); err != nil {
					s.logf("stratum job push failed: %v", err)
				}
			}
		case "mining.submit":
			if !authorized {
				_ = writer.WriteError(req.ID, 25, "authorize required")
				continue
			}
			jobID, err := ParamString(req.Params, 1)
			if err != nil {
				_ = writer.WriteError(req.ID, 21, err.Error())
				continue
			}
			nonce, err := ParamUint64(req.Params, 2)
			if err != nil {
				_ = writer.WriteError(req.ID, 21, err.Error())
				continue
			}
			job := s.engine.CurrentJob()
			if job.ID != jobID {
				_ = writer.WriteError(req.ID, 21, "stale job")
				continue
			}
			result, credit, err := s.engine.SubmitShare(sessionCtx, job, worker, nonce)
			if err != nil {
				_ = writer.WriteError(req.ID, 23, err.Error())
				continue
			}
			_ = writer.WriteResponse(req.ID, true)
			s.logf("share %s worker=%s job=%s value=%d", result.Kind, worker.Login, jobID, credit.Value)
			if result.Kind == pool.ShareBlock {
				if err := s.pushJob(sessionCtx, writer); err != nil {
					s.logf("stratum refresh after block failed: %v", err)
				}
			}
		default:
			_ = writer.WriteError(req.ID, 20, fmt.Sprintf("unsupported method %q", req.Method))
		}
	}
}

func (s *Server) pushJob(ctx context.Context, writer *connWriter) error {
	job, err := s.engine.RefreshWork(ctx)
	if err != nil {
		return err
	}
	params := []any{
		job.ID,
		fmt.Sprintf("%d", job.Height),
		job.Parent,
		job.Block.MerkleRoot,
		fmt.Sprintf("%d", job.PoolDifficulty),
		fmt.Sprintf("%d", job.BlockDifficulty),
		job.PoolTarget,
		fmt.Sprintf("%d", job.Timestamp),
		fmt.Sprintf("%d", job.Version),
		job.Block.MinerAddress,
		true,
	}
	return writer.WriteNotification("mining.notify", params...)
}

type connWriter struct {
	conn net.Conn
	mu   sync.Mutex
}

func (w *connWriter) WriteResponse(id any, result any) error {
	payload, err := EncodeResponse(id, result)
	if err != nil {
		return err
	}
	return w.write(payload)
}

func (w *connWriter) WriteError(id any, code int, message string) error {
	payload, err := EncodeError(id, code, message)
	if err != nil {
		return err
	}
	return w.write(payload)
}

func (w *connWriter) WriteNotification(method string, params ...any) error {
	payload, err := EncodeNotification(method, params...)
	if err != nil {
		return err
	}
	return w.write(payload)
}

func (w *connWriter) write(payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.conn.Write(payload)
	return err
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
