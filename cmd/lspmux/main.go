package main

import (
	"context"

	"github.com/buzztaiki/lspmux"
)

func main() {
	ctx := context.Background()
	if err := lspmux.Start(ctx); err != nil {
		panic(err)
	}
}
