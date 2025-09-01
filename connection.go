package lspmux

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

type Callable interface {
	// see [jsonrpc2.Connection.Call]
	Call(ctx context.Context, method string, params any) *jsonrpc2.AsyncCall
	// see [jsonrpc2.Connection.Notify]
	Notify(ctx context.Context, method string, params any) error
}

type Respondable interface {
	// see [jsonrpc2.Connection.Respond]
	Respond(id jsonrpc2.ID, result any, rerr error) error
}

func ForwardRequest(ctx context.Context, r *jsonrpc2.Request, callTo Callable) (any, error) {
	var res json.RawMessage
	if err := callTo.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func HandleRequestAsAsync(ctx context.Context, r *jsonrpc2.Request, respondTo Respondable, call func() (any, error)) (any, error) {
	go func() {
		res, callErr := call()
		if err := respondTo.Respond(r.ID, res, callErr); err != nil {
			slog.ErrorContext(ctx, "failed to respond", "error", err)
			return
		}
	}()

	return nil, jsonrpc2.ErrAsyncResponse
}
