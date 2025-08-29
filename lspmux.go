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

	clientHandler := NewClientHandler()
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(),
		jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: clientHandler})
	if err != nil {
		return err
	}
	defer clientConn.Close()

	for _, lsp := range cfg.LSPS {
		serverPipe, err := NewCmdPipeListener(ctx, exec.CommandContext(ctx, lsp.Command, lsp.Args...))
		if err != nil {
			return err
		}
		defer serverPipe.Close()

		serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(),
			jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: NewServerHandler(clientConn)})
		if err != nil {
			return err
		}
		defer serverConn.Close()

		clientHandler.AddServerConn(serverConn)
	}
	slog.Info("lspmux started")

	// TODO wait server connections
	clientConn.Wait()

	return nil
}
