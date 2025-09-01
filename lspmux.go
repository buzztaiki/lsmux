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
	clientPipe, err := NewIOPipeListener(ctx, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	defer clientPipe.Close()

	clientHandler := NewClientHandler(len(cfg.Servers))
	clientBinder := NewMiddlewareBinder(NewBinder(clientHandler), ContextLogMiddleware("ClientHandler"))
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(), clientBinder)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	diagRegistry := NewDiagnosticRegistry()
	for _, serverCfg := range cfg.Servers {
		slog.InfoContext(ctx, fmt.Sprintf("starting lsp server: %s: %s", serverCfg.Name, strings.Join(append([]string{serverCfg.Command}, serverCfg.Args...), " ")))
		serverPipe, err := NewCmdPipeListener(ctx, exec.CommandContext(ctx, serverCfg.Command, serverCfg.Args...))
		if err != nil {
			return err
		}
		defer serverPipe.Close()

		serverHandler := NewServerHandler(serverCfg.Name, clientConn, diagRegistry)
		serverBinder := NewMiddlewareBinder(NewBinder(serverHandler), ContextLogMiddleware("ServerHandler("+serverCfg.Name+")"))
		serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(), serverBinder)
		if err != nil {
			return err
		}
		defer serverConn.Close()

		clientHandler.AddServerConn(ctx, serverCfg.Name, serverConn, serverCfg.InitializationOptions)
	}
	slog.InfoContext(ctx, "lspmux started")

	// TODO wait server connections
	clientConn.Wait()

	return nil
}
