package lspmux

import "golang.org/x/exp/jsonrpc2"

func RequestType(r *jsonrpc2.Request) string {
	if r.IsCall() {
		return "request"
	}
	return "notification"
}

// SliceFor creates a slice of type T with length n.
func SliceFor[T any](t T, n int) []T {
	return make([]T, n)
}

func OrZeroValue[T any](t *T) T {
	if t == nil {
		var zero T
		return zero
	}
	return *t
}
