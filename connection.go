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

func ForwardRequest(ctx context.Context, r *jsonrpc2.Request, callTo Callable, logger *slog.Logger) (any, error) {
	logger.Info("forwarding request")
	var res json.RawMessage
	if err := callTo.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func ForwardRequestAsync(ctx context.Context, r *jsonrpc2.Request, callTo Callable, respondTo Respondable, logger *slog.Logger) (any, error) {
	logger.Info("forwarding request async")
	call := callTo.Call(ctx, r.Method, r.Params)
	go func() {
		var res json.RawMessage
		callErr := call.Await(ctx, &res)
		if err := respondTo.Respond(r.ID, res, callErr); err != nil {
			logger.Error("failed to respond", "error", err)
			return
		}
	}()
	return nil, jsonrpc2.ErrAsyncResponse
}

func CallRequestAsync(r *jsonrpc2.Request, respondTo Respondable, call func() (any, error), logger *slog.Logger) (any, error) {
	wait := make(chan struct{}, 1)
	defer close(wait)
	go func() {
		<- wait
		res, callErr := call()
		if err := respondTo.Respond(r.ID, res, callErr); err != nil {
			logger.Error("failed to respond", "error", err)
			return
		}
	}()
		

	return nil, jsonrpc2.ErrAsyncResponse
}
