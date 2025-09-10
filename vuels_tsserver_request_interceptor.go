package lsmux

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/myleshyson/lsprotocol-go/protocol"
	"golang.org/x/exp/jsonrpc2"
)

// VuelsTSServerRequestInterceptor intercepts "tsserver/request" notifications to support vuels v3 features.
// see https://github.com/vuejs/language-tools/discussions/5456
type VuelsTSServerRequestInterceptor struct {
	name           string
	serverRegistry *ServerConnectionRegistry
}

func NewVuelsTSServerRequestInterceptor(name string, serverRegistry *ServerConnectionRegistry) *VuelsTSServerRequestInterceptor {
	return &VuelsTSServerRequestInterceptor{
		name:           name,
		serverRegistry: serverRegistry,
	}
}

func (h *VuelsTSServerRequestInterceptor) Handler(next jsonrpc2.Handler) jsonrpc2.Handler {
	f := func(ctx context.Context, r *jsonrpc2.Request) (any, error) {
		if !r.IsCall() && r.Method == "tsserver/request" {
			slog.DebugContext(ctx, "intercept tsserver/request notification")
			return nil, h.handleTsServerRequest(ctx, r)
		}
		return next.Handle(ctx, r)
	}
	return jsonrpc2.HandlerFunc(f)
}

func (h *VuelsTSServerRequestInterceptor) handleTsServerRequest(ctx context.Context, r *jsonrpc2.Request) error {
	var params [][]json.RawMessage
	if err := json.Unmarshal(r.Params, &params); err != nil {
		return err
	}

	// see https://github.com/vuejs/language-tools/wiki/Neovim
	if len(params) != 1 || len(params[0]) != 3 {
		return ErrInvalidParams
	}

	id := params[0][0]
	command := params[0][1]
	args := params[0][2]

	servers := h.serverRegistry.Servers()

	execCommand := "typescript.tsserverRequest"
	tsServer, found := servers.FindByCommand(execCommand)
	if !found {
		return fmt.Errorf("no server found that supports command: %s", execCommand)
	}

	server, found := servers.FindByName(h.name)
	if !found {
		return fmt.Errorf("no server found: %s", h.name)
	}

	type ExecRes struct {
		Body json.RawMessage `json:"body"`
	}

	var execRes ExecRes
	if err := tsServer.Call(ctx, string(protocol.WorkspaceExecuteCommandMethod), protocol.ExecuteCommandParams{
		Command:   execCommand,
		Arguments: []any{command, args},
	}, &execRes); err != nil {
		return err
	}

	return server.Notify(ctx, "tsserver/response", [][]any{{id, execRes.Body}})
}
