package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

type ClientHandler struct {
	serverConn *jsonrpc2.Connection
}

func NewClientHandler() *ClientHandler {
	return &ClientHandler{}
}

func (h *ClientHandler) SetServerConn(conn *jsonrpc2.Connection) {
	h.serverConn = conn
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	logger := slog.With("component", "ClientHandler", "method", r.Method, "id", r.ID)
	logger.Info("handle")

	if !r.IsCall() {
		return nil, h.serverConn.Notify(ctx, r.Method, r.Params)
	}

	var res json.RawMessage
	if err := h.serverConn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
		logger.Error("call error", "error", err)
		return nil, err
	}
	return res, nil
}
