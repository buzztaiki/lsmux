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

type serverConn struct {
	name string
	Callable
	initOptions   map[string]any
	supportedCaps map[string]struct{}
	caps          *protocol.ServerCapabilities
}

func (c *serverConn) call(ctx context.Context, method string, params any) (any, error) {
	var res json.RawMessage
	if err := c.callWithRes(ctx, method, params, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *serverConn) callWithRes(ctx context.Context, method string, params any, res any) error {
	slog.InfoContext(ctx, "send request to "+c.name)
	return c.Call(ctx, method, params).Await(ctx, &res)
}

type ClientHandler struct {
	conn Respondable
	// TODO add server name to connection for better logging
	serverConns []*serverConn
	ready       chan (struct{})
	nservers    int
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

func (h *ClientHandler) AddServerConn(ctx context.Context, name string, conn *jsonrpc2.Connection, initOptions map[string]any) {
	if len(h.serverConns) < h.nservers {
		h.serverConns = append(h.serverConns, &serverConn{
			name:        name,
			Callable:    conn,
			initOptions: initOptions,
		})
	}
	if len(h.serverConns) == h.nservers {
		close(h.ready)
		slog.InfoContext(ctx, "all server connections established")
	}
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	<-h.ready

	method := protocol.MethodKind(r.Method)
	if method == protocol.InitializeMethod {
		return h.handleInitializeRequest(ctx, r)
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
			return serverConns[0].call(ctx, r.Method, r.Params)
		}
	})

}

func (h *ClientHandler) handleExecuteCommandRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*serverConn) (any, error) {
	var params protocol.ExecuteCommandParams
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return nil, err
	}

	commandSupported := func(c *serverConn) bool {
		return slices.Index(c.caps.ExecuteCommandProvider.Commands, params.Command) != -1
	}

	conn := serverConns[0]
	if i := slices.IndexFunc(h.serverConns, commandSupported); i != -1 {
		conn = serverConns[i]
	}

	return conn.call(ctx, r.Method, r.Params)
}

func (h *ClientHandler) handleCompletionRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*serverConn) (any, error) {
	g := new(errgroup.Group)
	results := SliceFor(protocol.CompletionResponse{}.Result, len(serverConns))
	for i, conn := range serverConns {
		g.Go(func() error {
			if err := conn.callWithRes(ctx, r.Method, r.Params, &results[i]); err != nil {
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

func (h *ClientHandler) handleCodeActionRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*serverConn) (any, error) {
	g := new(errgroup.Group)
	results := SliceFor(protocol.CodeActionResponse{}.Result, len(serverConns))
	for i, conn := range serverConns {
		g.Go(func() error {
			if err := conn.callWithRes(ctx, r.Method, r.Params, &results[i]); err != nil {
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
				v.Data = map[string]any{codeActionDataServerKey: serverConns[i].name, codeActionDataOriginalDataKey: v.Data}
				action.Value = v
			}
			res = append(res, action)
		}
	}

	return &res, nil
}

func (h *ClientHandler) handleCodeActionResolveRequest(ctx context.Context, r *jsonrpc2.Request, serverConns []*serverConn) (any, error) {
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

	i := slices.IndexFunc(serverConns, func(c *serverConn) bool { return c.name == serverName })
	if i == -1 {
		return nil, ErrMethodNotFound
	}

	return serverConns[i].call(ctx, r.Method, params)
}

func (h *ClientHandler) handleInitializeRequest(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	var merged map[string]any
	for _, conn := range h.serverConns {
		var kvParams map[string]any
		if err := json.Unmarshal(r.Params, &kvParams); err != nil {
			return nil, err
		}

		// override initializationOptions if configured
		if len(conn.initOptions) != 0 {
			slog.InfoContext(ctx, "override initializationOptions", "server", conn.name, "initOptions", conn.initOptions)
			kvParams["initializationOptions"] = conn.initOptions
		}

		var rawRes json.RawMessage
		if err := conn.callWithRes(ctx, r.Method, kvParams, &rawRes); err != nil {
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
