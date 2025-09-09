package lsmux

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

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
			slog.InfoContext(ctx, "intercept tsserver/request notification")
			return nil, h.handleTsServerRequest(ctx, r)
		}
		return next.Handle(ctx, r)
	}
	return jsonrpc2.HandlerFunc(f)
}

func (h *VuelsTSServerRequestInterceptor) handleTsServerRequest(ctx context.Context, r *jsonrpc2.Request) error {
	h.serverRegistry.WaitReady()

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

	execCommand := "typescript.tsserverRequest"
	tsServer, err := h.findCommandSupported(execCommand)
	if err != nil {
		return err
	}
	server, err := h.findSelf()
	if err != nil {
		return err
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

func (h *VuelsTSServerRequestInterceptor) findSelf() (*ServerConnection, error) {
	for _, sc := range h.serverRegistry.Servers() {
		if sc.Name == h.name {
			return sc, nil
		}
	}
	return nil, fmt.Errorf("connection not found: %s", h.name)
}

func (h *VuelsTSServerRequestInterceptor) findCommandSupported(command string) (*ServerConnection, error) {
	for _, sc := range h.serverRegistry.Servers() {
		if slices.Index(sc.Capabilities.ExecuteCommandProvider.Commands, command) != -1 {
			return sc, nil
		}
	}
	return nil, fmt.Errorf("connection not found for command: %s", command)
}
