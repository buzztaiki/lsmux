package lsmux

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"

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

type ServerConnectionList []*ServerConnection

func (r *ServerConnectionRegistry) Servers() ServerConnectionList {
	<-r.ready
	return r.servers
}

func (l ServerConnectionList) FilterBySupportedMethod(method string) ServerConnectionList {
	servers := []*ServerConnection{}
	for _, s := range l {
		if IsMethodSupported(method, s.SupportedCapabilities) {
			servers = append(servers, s)
		}
	}
	return servers
}

func (l ServerConnectionList) FindByName(name string) (*ServerConnection, bool) {
	i := slices.IndexFunc(l, func(s *ServerConnection) bool { return s.Name == name })
	if i == -1 {
		return nil, false
	}
	return l[i], true
}

func (l ServerConnectionList) FindByCommand(command string) (*ServerConnection, bool) {
	commandSupported := func(s *ServerConnection) bool {
		return slices.Index(s.Capabilities.ExecuteCommandProvider.Commands, command) != -1
	}

	i := slices.IndexFunc(l, commandSupported)
	if i == -1 {
		return nil, false
	}
	return l[i], true
}
