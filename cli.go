package lspmux

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	slogctx "github.com/veqryn/slog-context"
)

func CLI() error {
	configPath := "config.yaml"
	serverNamesValue := ""

	flag.StringVar(&configPath, "config", configPath, "path to config file")
	flag.StringVar(&serverNamesValue, "servers", serverNamesValue, "comma-separated server names to start (or empty to start all servers)")
	flag.Parse()

	logHandler := slogctx.NewHandler(
		tint.NewHandler(os.Stderr, &tint.Options{NoColor: true, TimeFormat: time.DateTime + ".000"}),
		nil)
	slog.SetDefault(slog.New(logHandler))

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
