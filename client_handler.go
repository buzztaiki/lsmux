package lspmux

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"dario.cat/mergo"
	"golang.org/x/exp/jsonrpc2"
)

type ClientHandler struct {
	// TODO add server name to connection for better logging
	serverConns []*serverConn
	ready       chan (struct{})
	nservers    int
}

type serverConn struct {
	name string
	*jsonrpc2.Connection
	supportedCaps map[string]struct{}
}

func NewClientHandler(nservers int) *ClientHandler {
	return &ClientHandler{
		ready:    make(chan struct{}),
		nservers: nservers,
	}
}

func (h *ClientHandler) AddServerConn(name string, conn *jsonrpc2.Connection) {
	if len(h.serverConns) < h.nservers {
		h.serverConns = append(h.serverConns, &serverConn{name, conn, nil})
	}
	if len(h.serverConns) == h.nservers {
		close(h.ready)
		slog.Info("all server connections established")
	}
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	// TODO logger and logging middlere
	logger := slog.With("component", "ClientHandler", "method", r.Method, "id", r.ID.Raw(), "type", RequestType(r))
	logger.Info("handle")

	<-h.ready

	switch r.Method {
	case "initialize":
		var merged map[string]any
		for _, conn := range h.serverConns {
			var res map[string]any
			if err := conn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
				return nil, err
			}

			caps, ok := res["capabilities"].(map[string]any)
			if !ok {
				return nil, errors.New("no capabilities in initialize response")
			}

			mergo.Merge(&merged, caps)
			conn.supportedCaps = CollectSupportedCapabilities(caps)
			logger.Info("supported server capabilities", "server", conn.name, "capabilities", conn.supportedCaps)
		}

		return map[string]any{
			"serverInfo": map[string]any{
				"name": "lspmux", // TODO configurable
			},
			"capabilities": merged,
		}, nil

	default:
		serverConns := []*serverConn{}
		for _, conn := range h.serverConns {
			if IsMethodSupported(r.Method, conn.supportedCaps) {
				serverConns = append(serverConns, conn)
			}
		}

		if len(serverConns) == 0 {
			return nil, ErrMethodNotFound
		}

		if !r.IsCall() {
			for _, conn := range serverConns {
				if err := conn.Notify(ctx, r.Method, r.Params); err != nil {
					return nil, err
				}
			}
			return nil, nil
		}

		var res json.RawMessage
		// Currently, request is sent to the first server only
		// TODO Some methods should have their results merged
		// TODO It would be nice if we could set how each method behaves
		conn := serverConns[0]
		if err := conn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
			return nil, err
		}
		return res, nil

	}
}
