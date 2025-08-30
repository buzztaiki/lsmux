package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

type ServerHandler struct {
	name       string
	conn       *jsonrpc2.Connection
	clientConn *jsonrpc2.Connection
}

func NewServerHandler(name string, clientConn *jsonrpc2.Connection) *ServerHandler {
	return &ServerHandler{
		name:       name,
		clientConn: clientConn,
	}
}

func (h *ServerHandler) BindConnection(conn *jsonrpc2.Connection) {
	h.conn = conn
}

func (h *ServerHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	logger := slog.With("component", "ServerHandler", "method", r.Method, "id", r.ID.Raw(), "type", RequestType(r), "name", h.name)
	logger.Info("handle")

	if !r.IsCall() {
		return nil, h.clientConn.Notify(ctx, r.Method, r.Params)
	}

	go func() {
		var res json.RawMessage
		call := h.clientConn.Call(ctx, r.Method, r.Params)
		callErr := call.Await(ctx, &res)
		if err := h.conn.Respond(r.ID, res, callErr); err != nil {
			logger.Error("failed to respond", "error", err)
		}
	}()

	return nil, jsonrpc2.ErrAsyncResponse
}
