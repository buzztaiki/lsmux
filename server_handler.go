package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

type ServerHandler struct {
	name       string
	clientConn *jsonrpc2.Connection
}

func NewServerHandler(name string, clientConn *jsonrpc2.Connection) *ServerHandler {
	return &ServerHandler{
		name:       name,
		clientConn: clientConn,
	}
}

func (h *ServerHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	logger := slog.With("component", "ServerHandler", "method", r.Method, "id", r.ID.Raw(), "type", RequestType(r), "name", h.name)
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
