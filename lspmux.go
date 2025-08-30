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
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(), NewBinder(clientHandler))
	if err != nil {
		return err
	}
	defer clientConn.Close()

	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	for _, serverCfg := range cfg.Servers {
		slog.Info(fmt.Sprintf("starting lsp server: %s: %s", serverCfg.Name, strings.Join(append([]string{serverCfg.Command}, serverCfg.Args...), " ")))
		serverPipe, err := NewCmdPipeListener(ctx, exec.CommandContext(ctx, serverCfg.Command, serverCfg.Args...))
		if err != nil {
			return err
		}
		defer serverPipe.Close()

		serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(), NewBinder(NewServerHandler(serverCfg.Name, clientConn)))
		if err != nil {
			return err
		}
		defer serverConn.Close()

		clientHandler.AddServerConn(serverCfg.Name, serverConn, serverCfg.InitializationOptions)
	}
	slog.Info("lspmux started")

	// TODO wait server connections
	clientConn.Wait()

	return nil
}
