package lspmux

import (
	"context"
	"encoding/json"
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

	go func() {
		rw, err := clientPipe.Accept(ctx)
		if err != nil {
			// TODO errgroup?
			panic(err)
		}
		go io.Copy(rw, os.Stdin)
		go io.Copy(os.Stdout, rw)
		slog.Info("langclient connected")
	}()

	clientHandler := NewClientHandler()
	clientConn, err := jsonrpc2.Dial(ctx, clientPipe.Dialer(), jsonrpc2.ConnectionOptions{
		Framer:  headerFramer,
		Handler: clientHandler,
	})
	if err != nil {
		return err
	}
	defer clientConn.Close()

	cmd := exec.CommandContext(ctx, "gopls")
	serverPipe, err := jsonrpc2.NetPipe(ctx)
	if err != nil {
		return err
	}
	defer serverPipe.Close()

	go func() {
		rw, err := serverPipe.Accept(ctx)
		if err != nil {
			// TODO errgroup?
			panic(err)
		}

		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			panic(err)
		}
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		go io.Copy(stdinPipe, rw)
		go io.Copy(rw, stdoutPipe)
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			panic(err)
		}
		slog.Info("langserver connected")
	}()

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

type ClientHandler struct {
	serverConn *jsonrpc2.Connection
}

func NewClientHandler() *ClientHandler {
	return &ClientHandler{}
}

func (h *ClientHandler) SetServerConn(conn *jsonrpc2.Connection) {
	h.serverConn = conn
}

func (h *ClientHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	logger := slog.With("component", "ClientHandler", "method", r.Method, "id", r.ID)
	logger.Info("handle")

	if !r.IsCall() {
		return nil, h.serverConn.Notify(ctx, r.Method, r.Params)
	}

	var res json.RawMessage
	if err := h.serverConn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
		logger.Error("call error", "error", err)
		return nil, err
	}
	return res, nil
}

type ServerHandler struct {
	clientConn *jsonrpc2.Connection
}

func NewServerHandler(clientConn *jsonrpc2.Connection) *ServerHandler {
	return &ServerHandler{
		clientConn: clientConn,
	}
}

func (h *ServerHandler) Handle(ctx context.Context, r *jsonrpc2.Request) (any, error) {
	logger := slog.With("component", "ServerHandler", "method", r.Method, "id", r.ID, "isCall", r.IsCall())
	logger.Info("handle")


	if !r.IsCall() {
		return nil, h.clientConn.Notify(ctx, r.Method, r.Params)
	}

	var res json.RawMessage
	if err := h.clientConn.Call(ctx, r.Method, r.Params).Await(ctx, &res); err != nil {
		logger.Error("call error", "error", err)
		return nil, err
	}
	return res, nil
}
