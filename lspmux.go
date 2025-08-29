package lspmux

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/exp/jsonrpc2"
)

func Start(ctx context.Context, cfg *Config) error {
	headerFramer := jsonrpc2.HeaderFramer()

	clientPipe, err := NewIOPipeListener(ctx, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	defer clientPipe.Close()

	clientHandler := NewClientHandler(len(cfg.LSPS))
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(),
		jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: clientHandler})
	if err != nil {
		return err
	}
	defer clientConn.Close()

	for name, lsp := range cfg.LSPS {
		slog.Info(fmt.Sprintf("starting lsp server: %s: %s", name, strings.Join(append([]string{lsp.Command}, lsp.Args...), " ")))
		serverPipe, err := NewCmdPipeListener(ctx, exec.CommandContext(ctx, lsp.Command, lsp.Args...))
		if err != nil {
			return err
		}
		defer serverPipe.Close()

		serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(),
			jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: NewServerHandler(clientConn, name)})
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
