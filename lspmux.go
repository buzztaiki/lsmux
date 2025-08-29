package lspmux

import (
	"context"
	"log/slog"
	"os"
	"os/exec"

	"golang.org/x/exp/jsonrpc2"
)

func Start(ctx context.Context, cfg *Config) error {
	headerFramer := jsonrpc2.HeaderFramer()

	clientPipe, err := NewIOPipeListener(ctx, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	defer clientPipe.Close()

	serverPipe, err := NewCmdPipeListener(ctx, exec.CommandContext(ctx, "gopls"))
	if err != nil {
		return err
	}
	defer serverPipe.Close()

	clientHandler := NewClientHandler()
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(),
		jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: clientHandler})
	if err != nil {
		return err
	}
	defer clientConn.Close()

	serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(),
		jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: NewServerHandler(clientConn)})
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
