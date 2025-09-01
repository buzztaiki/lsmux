package lspmux

import (
	"context"

	slogctx "github.com/veqryn/slog-context"
	"golang.org/x/exp/jsonrpc2"
)

type Middleware func(next jsonrpc2.Handler) jsonrpc2.Handler

type MiddlewareBinder struct {
	binder      jsonrpc2.Binder
	middlewares []Middleware
}

func NewMiddlewareBinder(binder jsonrpc2.Binder, middlewares ...Middleware) *MiddlewareBinder {
	return &MiddlewareBinder{binder: binder, middlewares: middlewares}
}

func (mb *MiddlewareBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	opts, err := mb.binder.Bind(ctx, conn)
	if err != nil {
		return opts, err
	}

	h := opts.Handler
	for i := len(mb.middlewares) - 1; i >= 0; i-- {
		h = mb.middlewares[i](h)
	}

	opts.Handler = h
	return opts, nil
}

func ContextLogMiddleware(name string) Middleware {
	return func(next jsonrpc2.Handler) jsonrpc2.Handler {
		f := func(ctx context.Context, r *jsonrpc2.Request) (any, error) {
			ctx = slogctx.Append(ctx, "name", name, "method", r.Method)
			if r.IsCall() {
				ctx = slogctx.Append(ctx, "type", "request", "id", r.ID.Raw())
			} else {
				ctx = slogctx.Append(ctx, "type", "notification")
			}
			return next.Handle(ctx, r)
		}
		return jsonrpc2.HandlerFunc(f)
	}
}
