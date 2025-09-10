package lsmux

import (
	"context"
	"encoding/base32"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
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
			traceUUID := uuid.New()
			traceID := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(traceUUID[:]))
			attrs := []any{
				slog.String("name", name),
				slog.String("traceID", traceID),
				slog.String("reqMethod", r.Method),
			}

			if r.IsCall() {
				attrs = append(attrs, slog.String("reqType", "request"), slog.Any("reqID", r.ID.Raw()))
			} else {
				attrs = append(attrs, slog.String("reqType", "notification"))
			}

			ctx = slogctx.Prepend(ctx, attrs...)
			return next.Handle(ctx, r)
		}
		return jsonrpc2.HandlerFunc(f)
	}
}

func LoggingMiddleware() Middleware {
	return func(next jsonrpc2.Handler) jsonrpc2.Handler {
		f := func(ctx context.Context, r *jsonrpc2.Request) (any, error) {
			start := time.Now()
			res, err := next.Handle(ctx, r)

			log := slog.With("duration", time.Since(start))
			if err == nil {
				log.DebugContext(ctx, "SUCCESS")
			} else if errors.Is(err, jsonrpc2.ErrAsyncResponse) {
				log.DebugContext(ctx, "SUCCESS (async)")
			} else {
				log.ErrorContext(ctx, "ERROR", "error", err)
			}

			return res, err
		}
		return jsonrpc2.HandlerFunc(f)
	}
}
