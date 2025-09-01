package lspmux

import (
	"context"
	"io"
	"os"
	"os/exec"

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

type Binder struct {
	h jsonrpc2.Handler
}

func NewBinder(h jsonrpc2.Handler) *Binder {
	return &Binder{h}
}

func (b Binder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	return jsonrpc2.ConnectionOptions{
		Framer:  jsonrpc2.HeaderFramer(),
		Handler: b.h,
	}, nil
}

func NewIOPipeListener(ctx context.Context, r io.Reader, w io.Writer) (jsonrpc2.Listener, error) {
	pipe, err := jsonrpc2.NetPipe(ctx)
	if err != nil {
		return nil, err
	}

	go bindIOToListener(ctx, pipe, r, w)
	return pipe, nil
}

func NewCmdPipeListener(ctx context.Context, cmd *exec.Cmd) (jsonrpc2.Listener, error) {
	pipe, err := jsonrpc2.NetPipe(ctx)
	if err != nil {
		return nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go bindIOToListener(ctx, pipe, stdout, stdin)
	return pipe, nil
}

func bindIOToListener(ctx context.Context, l jsonrpc2.Listener, r io.Reader, w io.Writer) error {
	rwc, err := l.Accept(ctx)
	if err != nil {
		return err
	}
	go io.Copy(rwc, r)
	go io.Copy(w, rwc)
	return nil
}
