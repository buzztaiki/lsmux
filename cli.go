package lspmux

import (
	"context"
	"flag"
)

func CLI() error {
	flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		return err
	}

	return Start(context.Background(), cfg)
}
