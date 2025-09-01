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
	logger := slog.Default()
	logger.InfoContext(ctx, "handle")

	method := protocol.MethodKind(r.Method)

	if !r.IsCall() {
		switch method {
		case protocol.TextDocumentPublishDiagnosticsMethod:
			return nil, h.handlePublishDiagnosticsNotification(ctx, r, logger)
		default:
			return nil, h.clientConn.Notify(ctx, r.Method, r.Params)
		}
	}

	return HandleRequestAsAsync(ctx, r, h.conn, func() (any, error) {
		return ForwardRequest(ctx, r, h.clientConn, logger)
	}, logger)
}

func (h *ServerHandler) handlePublishDiagnosticsNotification(ctx context.Context, r *jsonrpc2.Request, logger *slog.Logger) error {
	var params protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return err
	}

	logger.InfoContext(ctx, "server diags", "ndiags", len(params.Diagnostics))

	h.diagRegistry.UpdateDiagnostics(params.Uri, h.name, params.Diagnostics)
	params.Diagnostics = h.diagRegistry.GetDiagnostics(params.Uri)

	logger.InfoContext(ctx, "file diags", "ndiags", len(h.diagRegistry.GetDiagnostics(params.Uri)))
	logger.InfoContext(ctx, "return diags", "ndiags", len(params.Diagnostics), "data", params.Diagnostics)

	return h.clientConn.Notify(ctx, r.Method, params)
}
