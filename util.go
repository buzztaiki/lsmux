package lspmux

import "golang.org/x/exp/jsonrpc2"

func RequestType(r *jsonrpc2.Request) string {
	if r.IsCall() {
		return "request"
	}
	return "notification"
}
