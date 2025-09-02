package lsmux

import "golang.org/x/exp/jsonrpc2"

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#errorCodes
var (
	ErrParse            = jsonrpc2.ErrParse
	ErrInvalidRequest   = jsonrpc2.ErrInvalidRequest
	ErrMethodNotFound   = jsonrpc2.ErrMethodNotFound
	ErrInvalidParams    = jsonrpc2.ErrInvalidParams
	ErrUnknown          = jsonrpc2.ErrUnknown
	ErrInternal         = jsonrpc2.ErrInternal
	ErrServerOverloaded = jsonrpc2.ErrServerOverloaded
)
