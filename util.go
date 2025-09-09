package lsmux

// SliceFor creates a slice of type T with length n.
func SliceFor[T any](t T, n int) []T {
	return make([]T, n)
}

func Deref[T any](t *T) T {
	if t == nil {
		var zero T
		return zero
	}
	return *t
}
