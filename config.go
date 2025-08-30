package lspmux

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	// TODO init args
	// TODO lsp server selection
	// TODO request priority / merge policy
	Servers map[string]struct {
		Command string   `yaml:"command"`
		Args    []string `yaml:"args"`
	}
}

func LoadConfig(fname string) (*Config, error) {
	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, err
}
