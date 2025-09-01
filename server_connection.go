package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/myleshyson/lsprotocol-go/protocol"
	"golang.org/x/exp/jsonrpc2"
)

type ServerConnection struct {
	Name                  string
	conn                  *jsonrpc2.Connection
	InitOptions           map[string]any
	SupportedCapabilities map[string]struct{}
	Capabilities          *protocol.ServerCapabilities
}

func (c *ServerConnection) CallWithRawResult(ctx context.Context, method string, params any) (json.RawMessage, error) {
	var res json.RawMessage
	if err := c.Call(ctx, method, params, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ServerConnection) Call(ctx context.Context, method string, params any, res any) error {
	slog.InfoContext(ctx, "send request to "+c.Name, "method", method)
	return c.conn.Call(ctx, method, params).Await(ctx, &res)
}

func (c *ServerConnection) Notify(ctx context.Context, method string, params any) error {
	slog.InfoContext(ctx, "notify to "+c.Name, "method", method)
	return c.conn.Notify(ctx, method, params)
}

func (c *ServerConnection) Close() error {
	return c.conn.Close()
}

type ServerConnectionRegistry struct {
	servers  []*ServerConnection
	nservers int
	ready    chan (struct{})
}

func NewServerConnectionRegistry(nservers int) *ServerConnectionRegistry {
	return &ServerConnectionRegistry{
		servers:  make([]*ServerConnection, 0, nservers),
		nservers: nservers,
		ready:    make(chan struct{}),
	}
}

func (r *ServerConnectionRegistry) Add(ctx context.Context, name string, conn *jsonrpc2.Connection, initOptions map[string]any) {
	if len(r.servers) < r.nservers {
		r.servers = append(r.servers, &ServerConnection{
			Name:        name,
			conn:        conn,
			InitOptions: initOptions,
		})
	}
	if len(r.servers) == r.nservers {
		close(r.ready)
		slog.InfoContext(ctx, "all server connections established")
	}
}

func (r *ServerConnectionRegistry) Servers() []*ServerConnection {
	return r.servers
}

func (r *ServerConnectionRegistry) WaitReady() {
	<-r.ready
}
