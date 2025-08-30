package lspmux

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"

	"dario.cat/mergo"
	"github.com/myleshyson/lsprotocol-go/protocol"
	"golang.org/x/exp/jsonrpc2"
)

type ClientHandler struct {
	conn Respondable
	// TODO add server name to connection for better logging
	serverConns []*serverConn
	ready       chan (struct{})
	nservers    int
}

type serverConn struct {
	name string
	Callable
	initOptions   map[string]any
	supportedCaps map[string]struct{}
	caps          *protocol.ServerCapabilities
}

func NewClientHandler(nservers int) *ClientHandler {
	return &ClientHandler{
		ready:    make(chan struct{}),
		nservers: nservers,
	}
}

func (h *ClientHandler) BindConnection(conn *jsonrpc2.Connection) {
	h.conn = conn
}

func (h *ClientHandler) AddServerConn(name string, conn *jsonrpc2.Connection, initOptions map[string]any) {
	if len(h.serverConns) < h.nservers {
		h.serverConns = append(h.serverConns, &serverConn{
			name:        name,
			Callable:    conn,
			initOptions: initOptions,
		})
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

	method := protocol.MethodKind(r.Method)
	if method == protocol.InitializeMethod {
		return h.handleInitializeRequest(ctx, r, logger)
	}

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

	switch method {
	case protocol.WorkspaceExecuteCommandMethod:
		return h.handleExecuteCommandRequest(ctx, r, serverConns, logger)

	default:
		// Currently, request is sent to the first server only
		// TODO Some methods should have their results merged
		// TODO It would be nice if we could set how each method behaves
		conn := serverConns[0]
		return ForwardRequestAsync(ctx, r, conn, h.conn, logger.With("server", conn.name))
	}
}

func (h *ClientHandler) handleExecuteCommandRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*serverConn, logger *slog.Logger) (any, error) {
	var params protocol.ExecuteCommandParams
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, err
	}

	for _, conn := range serverConns {
		if slices.Index(conn.caps.ExecuteCommandProvider.Commands, params.Command) == -1 {
			continue
		}

		return ForwardRequestAsync(ctx, r, conn, h.conn, logger.With("command", params.Command, "server", conn.name))
	}

	return nil, ErrMethodNotFound
}

func (h *ClientHandler) handleInitializeRequest(ctx context.Context, r *jsonrpc2.Request, logger *slog.Logger) (any, error) {
	var merged map[string]any
	for _, conn := range h.serverConns {
		var kvParams map[string]any
		if err := json.Unmarshal(r.Params, &kvParams); err != nil {
			return nil, err
		}

		// override initializationOptions if configured
		if len(conn.initOptions) != 0 {
			logger.Info("override initializationOptions", "initOptions", conn.initOptions)
			kvParams["initializationOptions"] = conn.initOptions
		}

		var rawRes json.RawMessage
		if err := conn.Call(ctx, r.Method, kvParams).Await(ctx, &rawRes); err != nil {
			return nil, err
		}

		var typedRes protocol.InitializeResult
		if err := json.Unmarshal(rawRes, &typedRes); err != nil {
			return nil, err
		}

		var kvRes map[string]any
		if err := json.Unmarshal(rawRes, &kvRes); err != nil {
			return nil, err
		}

		kvCaps, ok := kvRes["capabilities"].(map[string]any)
		if !ok {
			return nil, errors.New("no capabilities in initialize response")
		}

		// respect the preceding value
		// NOTE: mergo.Merge did not union array values
		mergo.Merge(&merged, kvCaps)
		conn.caps = &typedRes.Capabilities
		conn.supportedCaps = CollectSupportedCapabilities(kvCaps)
	}

	return map[string]any{
		"serverInfo": map[string]any{
			"name": "lspmux", // TODO configurable
		},
		"capabilities": merged,
	}, nil
}
