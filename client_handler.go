package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"dario.cat/mergo"
	"golang.org/x/exp/jsonrpc2"
)

type ClientHandler struct {
	serverConns []*jsonrpc2.Connection
}

func NewClientHandler() *ClientHandler {
	return &ClientHandler{}
}

func (h *ClientHandler) AddServerConn(conn *jsonrpc2.Connection) {
	h.serverConns = append(h.serverConns, conn)
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	// TODO logger and logging middlere
	logger := slog.With("component", "ClientHandler", "method", r.Method, "id", r.ID, "call", r.IsCall())
	logger.Info("handle")

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
		return merged, nil

	// TODO Capability
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
