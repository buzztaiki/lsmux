package lspmux

import (
	"context"

	"golang.org/x/exp/jsonrpc2"
)

type ConnectionBindableHandler interface {
	jsonrpc2.Handler
	BindConnection(conn *jsonrpc2.Connection)
}

type Binder struct {
	h ConnectionBindableHandler
}

func NewBinder(h ConnectionBindableHandler) *Binder {
	return &Binder{h}
}

func (b Binder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	b.h.BindConnection(conn)
	return jsonrpc2.ConnectionOptions{
		Framer:  jsonrpc2.HeaderFramer(),
		Handler: b.h,
	}, nil
}
