// Package client provides a JSON-RPC 2.0 client that connects to the
// XDB daemon over a Unix domain socket.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/xdb-dev/xdb/rpc"
)

// Client is a JSON-RPC 2.0 client that communicates with the XDB daemon
// over a Unix domain socket.
type Client struct {
	http *http.Client
	base string
	seq  atomic.Int64
}

// New creates a [Client] that connects to the daemon at the given Unix socket path.
func New(socketPath string) *Client {
	dialer := &net.Dialer{}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		http: &http.Client{Transport: transport},
		base: "http://localhost",
	}
}

// Call invokes the named RPC method with the given params and unmarshals
// the result into result. If the server returns a JSON-RPC error, it is
// returned as an [*rpc.Error].
func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	id := fmt.Sprintf("%d", c.seq.Add(1))

	var rawParams json.RawMessage
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("rpc client: marshal params: %w", err)
		}

		rawParams = p
	}

	reqBody := rpc.Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("rpc client: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("rpc client: build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("rpc client: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	var resp rpc.Response
	if decErr := json.NewDecoder(httpResp.Body).Decode(&resp); decErr != nil {
		return fmt.Errorf("rpc client: decode response: %w", decErr)
	}

	if resp.Error != nil {
		return resp.Error
	}

	if result != nil && len(resp.Result) > 0 {
		if unmarshalErr := json.Unmarshal(resp.Result, result); unmarshalErr != nil {
			return fmt.Errorf("rpc client: unmarshal result: %w", unmarshalErr)
		}
	}

	return nil
}

// StreamFrame is one server-sent event from a streaming method.
type StreamFrame struct {
	Event string          // ready | event | done | error
	Data  json.RawMessage // frame payload
}

// Stream invokes a streaming RPC method and calls handle for every
// frame until the stream ends. A terminal "error" frame is returned as
// an [*rpc.Error]; a "done" frame ends the stream with nil. The context
// cancels the stream.
func (c *Client) Stream(
	ctx context.Context,
	method string,
	params any,
	handle func(StreamFrame) error,
) error {
	id := fmt.Sprintf("%d", c.seq.Add(1))

	var rawParams json.RawMessage
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("rpc client: marshal params: %w", err)
		}

		rawParams = p
	}

	body, err := json.Marshal(rpc.Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	})
	if err != nil {
		return fmt.Errorf("rpc client: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("rpc client: build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("rpc client: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	return consumeSSE(ctx, httpResp.Body, handle)
}

// consumeSSE reads server-sent-event frames until a terminal frame or
// EOF, dispatching non-terminal frames to handle.
func consumeSSE(
	ctx context.Context,
	body io.Reader,
	handle func(StreamFrame) error,
) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var event string
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")

		case strings.HasPrefix(line, "data: "):
			data := json.RawMessage(strings.TrimPrefix(line, "data: "))

			switch event {
			case "done":
				return nil
			case "error":
				var rpcErr rpc.Error
				if unmarshalErr := json.Unmarshal(data, &rpcErr); unmarshalErr != nil {
					return fmt.Errorf("rpc client: stream error: %s", data)
				}
				return &rpcErr
			default:
				if handleErr := handle(StreamFrame{Event: event, Data: data}); handleErr != nil {
					return handleErr
				}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && ctx.Err() == nil {
		return fmt.Errorf("rpc client: stream read: %w", scanErr)
	}

	return nil
}
