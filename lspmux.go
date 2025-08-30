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

	clientHandler := NewClientHandler(len(cfg.Servers))
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(),
		jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: clientHandler})
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

		serverConn, err := jsonrpc2.Dial(ctx, serverPipe.Dialer(),
			jsonrpc2.ConnectionOptions{Framer: headerFramer, Handler: NewServerHandler(serverCfg.Name, clientConn)})
		if err != nil {
			return err
		}
		defer serverConn.Close()

		clientHandler.AddServerConn(serverCfg.Name, serverConn)
	}
	slog.Info("lspmux started")

	// TODO wait server connections
	clientConn.Wait()

	return nil
}
