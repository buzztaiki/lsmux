package lspmux

import (
	"context"
	"flag"
	"strings"
)

func CLI() error {
	configPath := "config.yaml"
	serverNamesValue := ""

	flag.StringVar(&configPath, "config", configPath, "path to config file")
	flag.StringVar(&serverNamesValue, "servers", serverNamesValue, "comma-separated server names to start (or empty to start all servers)")
	flag.Parse()

	var serverNames []string
	for name := range strings.SplitSeq(serverNamesValue, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		serverNames = append(serverNames, name)
	}

	cfg, err := LoadConfig(configPath, serverNames)
	if err != nil {
		return err
	}

	return Start(context.Background(), cfg)
}
