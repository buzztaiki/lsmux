package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"dario.cat/mergo"
	"golang.org/x/exp/jsonrpc2"
)

type ClientHandler struct {
	// TODO add server name to connection for better logging
	serverConns []*jsonrpc2.Connection
	ready 	 chan(struct{})
	nservers int
}

func NewClientHandler(nservers int) *ClientHandler {
	return &ClientHandler{
		ready: make(chan struct{}),
		nservers: nservers,
	}
}

func (h *ClientHandler) AddServerConn(conn *jsonrpc2.Connection) {
	if len(h.serverConns) < h.nservers {
		h.serverConns = append(h.serverConns, conn)
	}
	if len(h.serverConns) == h.nservers {
		close(h.ready)
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
			mergo.Merge(&merged, res["capabilities"])
		}
		return map[string]any{
			"serverInfo": map[string]any{
				"name": "lspmux", // TODO configurable
			},
			"capabilities": merged,
		}, nil

	// TODO Check capability
	default:
		if !r.IsCall() {
			for _, conn := range h.serverConns {
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
		conn := h.serverConns[0]
		if err := conn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
			return nil, err
		}
		return res, nil

	}
}
