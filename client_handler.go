package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

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
	logger := slog.With("component", "ClientHandler", "method", r.Method, "id", r.ID)
	logger.Info("handle")

	// TODO
	for _, conn := range h.serverConns[:1] {
		if !r.IsCall() {
			return nil, conn.Notify(ctx, r.Method, r.Params)
		}

		var res json.RawMessage
		if err := conn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
			logger.Error("call error", "error", err)
			return nil, err
		}
		return res, nil
	}

	return nil, jsonrpc2.ErrNotHandled
}
