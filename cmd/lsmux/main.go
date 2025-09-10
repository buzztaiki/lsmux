package main

import (
	"fmt"
	"os"

	"github.com/buzztaiki/lsmux"
)

func main() {
	if err := lsmux.CLI(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
