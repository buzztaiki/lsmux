package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

type ServerHandler struct {
	clientConn *jsonrpc2.Connection
}

func NewServerHandler(clientConn *jsonrpc2.Connection) *ServerHandler {
	return &ServerHandler{
		clientConn: clientConn,
	}
}

func (h *ServerHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	logger := slog.With("component", "ServerHandler", "method", r.Method, "id", r.ID, "call", r.IsCall())
	logger.Info("handle")


	if !r.IsCall() {
		return nil, h.clientConn.Notify(ctx, r.Method, r.Params)
	}

	var res json.RawMessage
	if err := h.clientConn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
		return nil, err
	}
	return res, nil
}
