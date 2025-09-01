package lspmux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"dario.cat/mergo"
	"github.com/myleshyson/lsprotocol-go/protocol"
	"golang.org/x/exp/jsonrpc2"
	"golang.org/x/sync/errgroup"
)

type ClientHandler struct {
	conn           Respondable
	serverRegistry *ServerConnectionRegistry
}

func NewClientHandler(serverRegistry *ServerConnectionRegistry) *ClientHandler {
	return &ClientHandler{
		serverRegistry: serverRegistry,
	}
}

func (h *ClientHandler) BindConnection(conn *jsonrpc2.Connection) {
	h.conn = conn
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	h.serverRegistry.WaitReady()

	method := protocol.MethodKind(r.Method)
	if method == protocol.InitializeMethod {
		return h.handleInitializeRequest(ctx, r)
	}

	serverConns := []*ServerConnection{}
	for _, conn := range h.serverRegistry.Servers() {
		if IsMethodSupported(r.Method, conn.SupportedCapabilities) {
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

	return HandleRequestAsAsync(ctx, r, h.conn, func(ctx context.Context) (any, error) {
		switch method {
		case protocol.WorkspaceExecuteCommandMethod:
			return h.handleExecuteCommandRequest(ctx, r, serverConns)
		case protocol.TextDocumentCompletionMethod:
			return h.handleCompletionRequest(ctx, r, serverConns)
		case protocol.TextDocumentCodeActionMethod:
			return h.handleCodeActionRequest(ctx, r, serverConns)
		case protocol.CodeActionResolveMethod:
			return h.handleCodeActionResolveRequest(ctx, r, serverConns)

		default:
			// Currently, request is sent to the first server only
			// TODO Some methods should have their results merged
			// TODO It would be nice if we could set how each method behaves
			return serverConns[0].DefaultCall(ctx, r.Method, r.Params)
		}
	})

}

func (h *ClientHandler) handleExecuteCommandRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*ServerConnection) (any, error) {
	var params protocol.ExecuteCommandParams
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, err
	}

	commandSupported := func(c *ServerConnection) bool {
		return slices.Index(c.Capabilities.ExecuteCommandProvider.Commands, params.Command) != -1
	}

	conn := serverConns[0]
	if i := slices.IndexFunc(serverConns, commandSupported); i != -1 {
		conn = serverConns[i]
	}

	return conn.DefaultCall(ctx, r.Method, r.Params)
}

func (h *ClientHandler) handleCompletionRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*ServerConnection) (any, error) {
	g := new(errgroup.Group)
	results := SliceFor(protocol.CompletionResponse{}.Result, len(serverConns))
	for i, conn := range serverConns {
		g.Go(func() error {
			if err := conn.Call(ctx, r.Method, r.Params, &results[i]); err != nil {
				return err
			}
			return nil
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
		case nil:
		// do nothing
		default:
			panic(fmt.Sprintf("invalid completion result type: %T", v))
		}
	}

	return &res, nil
}

const codeActionDataServerKey = "lspmux.server"
const codeActionDataOriginalDataKey = "lspmux.originalData"

func (h *ClientHandler) handleCodeActionRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*ServerConnection) (any, error) {
	g := new(errgroup.Group)
	results := SliceFor(protocol.CodeActionResponse{}.Result, len(serverConns))
	for i, conn := range serverConns {
		g.Go(func() error {
			if err := conn.Call(ctx, r.Method, r.Params, &results[i]); err != nil {
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	res := OrZeroValue(protocol.CodeActionResponse{}.Result)
	for i, r := range results {
		for _, action := range OrZeroValue(r) {
			if v, ok := action.Value.(protocol.CodeAction); ok {
				// add server name to code action data for future resolve
				v.Data = map[string]any{codeActionDataServerKey: serverConns[i].Name, codeActionDataOriginalDataKey: v.Data}
				action.Value = v
			}
			res = append(res, action)
		}
	}

	return &res, nil
}

func (h *ClientHandler) handleCodeActionResolveRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*ServerConnection) (any, error) {
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

	i := slices.IndexFunc(serverConns, func(c *ServerConnection) bool { return c.Name == serverName })
	if i == -1 {
		return nil, ErrMethodNotFound
	}

	return serverConns[i].DefaultCall(ctx, r.Method, params)
}

func (h *ClientHandler) handleInitializeRequest(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	var merged map[string]any
	for _, conn := range h.serverRegistry.Servers() {
		var kvParams map[string]any
		if err := json.Unmarshal(r.Params, &kvParams); err != nil {
			return nil, err
		}

		// override initializationOptions if configured
		if len(conn.InitOptions) != 0 {
			slog.InfoContext(ctx, "override initializationOptions", "server", conn.Name, "initOptions", conn.InitOptions)
			kvParams["initializationOptions"] = conn.InitOptions
		}

		var rawRes json.RawMessage
		if err := conn.Call(ctx, r.Method, kvParams, &rawRes); err != nil {
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
		conn.Capabilities = &typedRes.Capabilities
		conn.SupportedCapabilities = CollectSupportedCapabilities(kvCaps)
	}

	return map[string]any{
		"serverInfo": map[string]any{
			"name": "lspmux", // TODO configurable
		},
		"capabilities": merged,
	}, nil
}
