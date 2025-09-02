package lsmux

import (
	"fmt"
	"os"
	"slices"

	"github.com/goccy/go-yaml"
)

type Config struct {
	// TODO init args
	// TODO request priority / merge policy
	Servers []ServerConfig `yaml:"servers"` // use slice to respect config order
}

type ServerConfig struct {
	Name                  string         `yaml:"name"`
	Command               string         `yaml:"command"`
	Args                  []string       `yaml:"args"`
	InitializationOptions map[string]any `yaml:"initializationOptions"`
}

type ServerConfigList []ServerConfig

func LoadConfig(fname string, serverNames []string) (*Config, error) {
	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	for i := range cfg.Servers {
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

	return &cfg, err
}
