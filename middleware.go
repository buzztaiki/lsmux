package lspmux

import (
	"context"
	"log/slog"
	"time"

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
			attrs := []any{
				slog.String("name", name),
				slog.String("method", r.Method),
			}

			if r.IsCall() {
				attrs = append(attrs, slog.String("type", "request"), slog.Any("id", r.ID.Raw()))
			} else {
				attrs = append(attrs, slog.String("type", "notification"))
			}

			ctx = slogctx.Prepend(ctx, attrs...)
			return next.Handle(ctx, r)
		}
		return jsonrpc2.HandlerFunc(f)
	}
}

type startTimeCtxKey struct{}

func AccessLogMiddleware() Middleware {
	return func(next jsonrpc2.Handler) jsonrpc2.Handler {
		f := func(ctx context.Context, r *jsonrpc2.Request) (any, error) {
			return WithAccessLog(ctx, func(ctx context.Context) (any, error) {
				return next.Handle(ctx, r)
			})
		}
		return jsonrpc2.HandlerFunc(f)
	}
}

func WithAccessLog(ctx context.Context, f func(ctx context.Context) (any, error)) (any, error) {
	var start time.Time
	if st, ok := ctx.Value(startTimeCtxKey{}).(time.Time); ok {
		start = st
	} else {
		start = time.Now()
		ctx = context.WithValue(ctx, startTimeCtxKey{}, start)
	}

	res, err := f(ctx)
	if err != nil && err != jsonrpc2.ErrAsyncResponse {
		slog.ErrorContext(ctx, "ERROR", "error", err, "duration", time.Since(start))
	} else if err == nil {
		slog.InfoContext(ctx, "SUCCESS", "duration", time.Since(start))
	}

	return res, err

}
