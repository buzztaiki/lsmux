package lspmux

import (
	"context"
	"io"
	"os"
	"os/exec"

	"golang.org/x/exp/jsonrpc2"
)

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
	defer pipe.Close()

	go bindCmdToListener(ctx, pipe, cmd)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

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

func bindCmdToListener(ctx context.Context, l jsonrpc2.Listener, cmd *exec.Cmd) error {
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	return bindIOToListener(ctx, l, stdoutPipe, stdinPipe)
}
