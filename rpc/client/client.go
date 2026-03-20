// Package client provides a JSON-RPC 2.0 client that connects to the
// XDB daemon over a Unix domain socket.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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
