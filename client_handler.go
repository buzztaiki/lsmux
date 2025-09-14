package lsmux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"dario.cat/mergo"
	"github.com/buzztaiki/lsmux/capability"
	"github.com/myleshyson/lsprotocol-go/protocol"
	"golang.org/x/exp/jsonrpc2"
	"golang.org/x/sync/errgroup"
)

type ClientHandler struct {
	serverRegistry *ServerConnectionRegistry
	shutdown       bool
	done           chan struct{}
}

func NewClientHandler(serverRegistry *ServerConnectionRegistry) *ClientHandler {
	return &ClientHandler{
		serverRegistry: serverRegistry,
		done:           make(chan struct{}),
	}
}

func (h *ClientHandler) WaitExit() {
	<-h.done
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	if protocol.MethodKind(r.Method) == protocol.ExitMethod {
		return nil, h.handleExitNotification(ctx)
	}

	if h.shutdown {
		return nil, ErrInvalidRequest
	}

	servers := h.serverRegistry.Servers().FilterBySupportedMethod(r.Method)
	if len(servers) == 0 {
		return nil, ErrMethodNotFound
	}

	if !r.IsCall() {
		for _, server := range servers {
			if err := server.Notify(ctx, r.Method, r.Params); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	switch protocol.MethodKind(r.Method) {
	case protocol.InitializeMethod:
		return h.handleInitializeRequest(ctx, r, servers)
	case protocol.WorkspaceExecuteCommandMethod:
		return h.handleExecuteCommandRequest(ctx, r, servers)
	case protocol.TextDocumentCompletionMethod:
		return h.handleCompletionRequest(ctx, r, servers)
	case protocol.TextDocumentCodeActionMethod:
		return h.handleCodeActionRequest(ctx, r, servers)
	case protocol.CodeActionResolveMethod:
		return h.handleCodeActionResolveRequest(ctx, r, servers)
	case protocol.ShutdownMethod:
		return h.handleShutdownRequest(ctx, r, servers)

	default:
		// Currently, request is sent to the first server only
		return servers[0].CallWithRawResult(ctx, r.Method, r.Params)
	}
}

func (h *ClientHandler) handleInitializeRequest(ctx context.Context, r *jsonrpc2.Request, servers ServerConnectionList) (any, error) {
	merged := map[string]any{}
	for _, server := range servers {
		var kvParams map[string]any
		if err := json.Unmarshal(r.Params, &kvParams); err != nil {
			return nil, err
		}

		// override initializationOptions if configured
		if len(server.InitOptions) != 0 {
			slog.DebugContext(ctx, "override initializationOptions", "server", server.Name, "initOptions", server.InitOptions)
			kvParams["initializationOptions"] = server.InitOptions
		}

		var rawRes json.RawMessage
		if err := server.Call(ctx, r.Method, kvParams, &rawRes); err != nil {
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
		capability.Merge(merged, kvCaps)
		server.Capabilities = &typedRes.Capabilities
		server.SupportedCapabilities = capability.CollectSupported(kvCaps)

		slog.DebugContext(ctx, "server capabilities",
			"server", server.Name,
			"capabilities", kvCaps,
			"supportedCapabilities", slices.Collect(maps.Keys(server.SupportedCapabilities)))
	}

	return map[string]any{
		"serverInfo": map[string]any{
			"name": "lsmux", // TODO configurable
		},
		"capabilities": merged,
	}, nil
}

func (h *ClientHandler) handleExecuteCommandRequest(ctx context.Context, r *jsonrpc2.Request, servers ServerConnectionList) (any, error) {
	var params protocol.ExecuteCommandParams
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, err
	}

	server, found := servers.FindByCommand(params.Command)
	if !found {
		server = servers[0]
	}

	return server.CallWithRawResult(ctx, r.Method, r.Params)
}

func (h *ClientHandler) handleCompletionRequest(ctx context.Context, r *jsonrpc2.Request, servers ServerConnectionList) (any, error) {
	results := SliceFor(protocol.CompletionResponse{}.Result, len(servers))
	g, gctx := errgroup.WithContext(ctx)
	for i, server := range servers {
		g.Go(func() error {
			return server.Call(gctx, r.Method, r.Params, &results[i])
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var res protocol.CompletionList
	for _, r := range results {
		if v, ok := r.Value.(protocol.CompletionList); ok {
			mergo.Merge(&res, v)
		}
	}

	res.Items = []protocol.CompletionItem{}
	for _, r := range results {
		switch v := r.Value.(type) {
		case []protocol.CompletionItem:
			res.Items = append(res.Items, v...)
		case protocol.CompletionList:
			res.Items = append(res.Items, v.Items...)
		case nil: // do nothing
		default:
			panic(fmt.Sprintf("invalid completion result type: %T", v))
		}
	}

	return &res, nil
}

const codeActionDataServerKey = "lsmux.server"
const codeActionDataOriginalDataKey = "lsmux.originalData"

func (h *ClientHandler) handleCodeActionRequest(ctx context.Context, r *jsonrpc2.Request, servers ServerConnectionList) (any, error) {
	results := SliceFor(protocol.CodeActionResponse{}.Result, len(servers))
	g, gctx := errgroup.WithContext(ctx)
	for i, server := range servers {
		g.Go(func() error {
			return server.Call(gctx, r.Method, r.Params, &results[i])
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	res := Deref(protocol.CodeActionResponse{}.Result)
	for i, r := range results {
		for _, action := range Deref(r) {
			if v, ok := action.Value.(protocol.CodeAction); ok {
				// add server name to code action data for future resolve
				v.Data = map[string]any{codeActionDataServerKey: servers[i].Name, codeActionDataOriginalDataKey: v.Data}
				action.Value = v
			}
			res = append(res, action)
		}
	}

	return &res, nil
}

func (h *ClientHandler) handleCodeActionResolveRequest(ctx context.Context, r *jsonrpc2.Request, servers ServerConnectionList) (any, error) {
	params := protocol.CodeActionResolveRequest{}.Params
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, err
	}

	extractData := func() (string, any, error) {
		data, ok := params.Data.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("invalid code action data")
		}
		serverName, ok := data[codeActionDataServerKey].(string)
		if !ok {
			return "", nil, fmt.Errorf("%s not found in code action data", codeActionDataServerKey)
		}
		return serverName, data[codeActionDataOriginalDataKey], nil
	}

	serverName, originalData, err := extractData()
	if err != nil {
		return nil, err
	}
	params.Data = originalData

	server, found := servers.FindByName(serverName)
	if !found {
		return nil, ErrMethodNotFound
	}

	return server.CallWithRawResult(ctx, r.Method, params)
}

func (h *ClientHandler) handleShutdownRequest(ctx context.Context, r *jsonrpc2.Request, servers ServerConnectionList) (any, error) {
	g := new(errgroup.Group)
	for _, server := range servers {
		g.Go(func() error {
			log := slog.With("server", server.Name)
			if err := server.Call(ctx, r.Method, r.Params, nil); err != nil {
				log.WarnContext(ctx, "shutdown error", "error", err)
			}
			if err := server.Notify(ctx, string(protocol.ExitMethod), nil); err != nil {
				log.WarnContext(ctx, "exit notification error", "error", err)
			}
			if err := server.Close(); err != nil {
				log.WarnContext(ctx, "connection close error", "error", err)
			}
			log.InfoContext(ctx, "server shutdown completed")

			return nil
		})
	}
	g.Wait()

	h.shutdown = true
	return []any{}, nil
}

func (h *ClientHandler) handleExitNotification(_ context.Context) error {
	close(h.done)
	return nil
}
