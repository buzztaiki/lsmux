package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/myleshyson/lsprotocol-go/protocol"
	"golang.org/x/exp/jsonrpc2"
)

type ServerHandler struct {
	name         string
	conn         Respondable
	clientConn   Callable
	diagRegistry *DiagnosticRegistry
}

func NewServerHandler(name string, clientConn *jsonrpc2.Connection, diagRegistry *DiagnosticRegistry) *ServerHandler {
	return &ServerHandler{
		name:         name,
		clientConn:   clientConn,
		diagRegistry: diagRegistry,
	}
}

func (h *ServerHandler) BindConnection(conn *jsonrpc2.Connection) {
	h.conn = conn
}

func (h *ServerHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	slog.InfoContext(ctx, "handle")

	method := protocol.MethodKind(r.Method)

	if !r.IsCall() {
		switch method {
		case protocol.TextDocumentPublishDiagnosticsMethod:
			return nil, h.handlePublishDiagnosticsNotification(ctx, r)
		default:
			return nil, h.clientConn.Notify(ctx, r.Method, r.Params)
		}
	}

	return HandleRequestAsAsync(ctx, r, h.conn, func() (any, error) {
		return ForwardRequest(ctx, r, h.clientConn)
	})
}

func (h *ServerHandler) handlePublishDiagnosticsNotification(ctx context.Context, r *jsonrpc2.Request) error {
	var params protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return err
	}

	h.diagRegistry.UpdateDiagnostics(params.Uri, h.name, params.Diagnostics)
	params.Diagnostics = h.diagRegistry.GetDiagnostics(params.Uri)

	return h.clientConn.Notify(ctx, r.Method, params)
}
