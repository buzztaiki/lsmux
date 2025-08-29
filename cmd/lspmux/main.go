package main

import (
	"fmt"
	"os"

	"github.com/buzztaiki/lspmux"
)

func main() {
	if err := lspmux.CLI(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
