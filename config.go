package lsmux

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/goccy/go-yaml"
)

type Config struct {
	LogLevel slog.Level     `yaml:"logLevel"`
	Servers  []ServerConfig `yaml:"servers"` // use slice to respect config order
}

type ServerConfig struct {
	Name                  string         `yaml:"name"`
	Command               string         `yaml:"command"`
	Args                  []string       `yaml:"args"`
	InitializationOptions map[string]any `yaml:"initializationOptions"`
}

func LoadConfigFile(fname string, serverNames []string) (*Config, error) {
	r, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return LoadConfig(r, serverNames)
}

func LoadConfig(r io.Reader, serverNames []string) (*Config, error) {
	cfg := Config{
		LogLevel: slog.LevelInfo,
	}

	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, err
	}

	for i := range cfg.Servers {
		if cfg.Servers[i].Command == "" {
			return nil, fmt.Errorf("servers[%d]: command is required", i)
		}

		if cfg.Servers[i].Name == "" {
			cfg.Servers[i].Name = cfg.Servers[i].Command
		}
	}

	if len(serverNames) == 0 {
		return &cfg, nil
	}

	var servers []ServerConfig
	for _, name := range serverNames {
		i := slices.IndexFunc(cfg.Servers, func(s ServerConfig) bool { return s.Name == name })
		if i == -1 {
			return nil, fmt.Errorf("server not found in config: %s", name)
		}
		servers = append(servers, cfg.Servers[i])
	}
	cfg.Servers = servers

	return &cfg, nil
}
