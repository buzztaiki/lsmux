package lspmux

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"golang.org/x/exp/jsonrpc2"
)

func Start(ctx context.Context) error {
	headerFramer := jsonrpc2.HeaderFramer()

	clientPipe, err := jsonrpc2.NetPipe(ctx)
	if err != nil {
		return err
	}
	defer clientPipe.Close()
	go bindIOToListener(ctx, clientPipe, os.Stdin, os.Stdout)

	serverPipe, err := jsonrpc2.NetPipe(ctx)
	if err != nil {
		return err
	}
	defer serverPipe.Close()

	cmd := exec.CommandContext(ctx, "gopls")
	go bindCmdToListener(ctx, serverPipe, cmd)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	clientHandler := NewClientHandler()
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(), jsonrpc2.ConnectionOptions{
		Framer:  headerFramer,
		Handler: clientHandler,
	})
	if err != nil {
		return err
	}
	defer clientConn.Close()

	serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(), jsonrpc2.ConnectionOptions{
		Framer:  headerFramer,
		Handler: NewServerHandler(clientConn),
	})
	if err != nil {
		return err
	}
	defer serverConn.Close()
	clientHandler.SetServerConn(serverConn)

	slog.Info("server started")
	serverConn.Wait()
	clientConn.Wait()

	return nil
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
