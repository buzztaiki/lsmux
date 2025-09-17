package lsmux

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig_Servers(t *testing.T) {
	tests := []struct {
		name        string
		serverNames []string
		data        string
		want        []ServerConfig
	}{
		{
			name:        "empty serverNames",
			serverNames: nil,
			data:        `servers: [{name: server1, command: cmd1}, {name: server2, command: cmd2}]`,
			want: []ServerConfig{
				{Name: "server1", Command: "cmd1"},
				{Name: "server2", Command: "cmd2"},
			},
		},
		{
			name:        "select servers",
			serverNames: []string{"server2"},
			data:        `servers: [{name: server1, command: cmd1}, {name: server2, command: cmd2}]`,
			want: []ServerConfig{
				{Name: "server2", Command: "cmd2"},
			},
		},
		{
			name:        "reorder",
			serverNames: []string{"server2", "server1"},
			data:        `servers: [{name: server1, command: cmd1}, {name: server2, command: cmd2}]`,
			want: []ServerConfig{
				{Name: "server2", Command: "cmd2"},
				{Name: "server1", Command: "cmd1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(bytes.NewBufferString(tt.data), tt.serverNames)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, cfg.Servers); diff != "" {
				t.Errorf("cfg.Servers mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadConfig_Errors(t *testing.T) {
	tests := []struct {
		name        string
		serverNames []string
		data        string
		wantErr     string
	}{
		{
			name:        "non-existent server",
			serverNames: []string{"server2"},
			data:        `servers: [{name: server, command: cmd}]`,
			wantErr:     "server not found in config: server2",
		},
		{
			name:    "command required",
			data:    `servers: [{name: server}]`,
			wantErr: "servers[0]: command is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(bytes.NewBufferString(tt.data), tt.serverNames)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want contains %v", err.Error(), tt.wantErr)
			}
		})
	}
}
