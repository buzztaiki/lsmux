package lsmux

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	slogctx "github.com/veqryn/slog-context"
)

func CLI() error {
	configPath := "config.yaml"
	serverNamesValue := ""

	flag.StringVar(&configPath, "config", configPath, "path to config file")
	flag.StringVar(&serverNamesValue, "servers", serverNamesValue, "comma-separated server names to start (or empty to start all servers)")
	flag.Parse()

	logHandler := slogctx.NewHandler(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.String(a.Key, a.Value.Time().Format(time.DateTime+".000"))
				}
				return a
			},
		}),
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

	return Execute(context.Background(), cfg)
}
