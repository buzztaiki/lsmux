package lspmux

import (
	"sync"

	"github.com/myleshyson/lsprotocol-go/protocol"
)

type DiagnosticRegistry struct {
	mu sync.Mutex
	// document uri -> server name -> list of diags
	allDiags map[protocol.DocumentUri]map[string][]protocol.Diagnostic
}

func NewDiagnosticRegistry() *DiagnosticRegistry {
	return &DiagnosticRegistry{
		allDiags: make(map[protocol.DocumentUri]map[string][]protocol.Diagnostic),
	}
}

func (r *DiagnosticRegistry) UpdateDiagnostics(uri protocol.DocumentUri, serverName string, diags []protocol.Diagnostic) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.allDiags[uri]; !ok {
		r.allDiags[uri] = make(map[string][]protocol.Diagnostic)
	}

	r.allDiags[uri][serverName] = diags
}

func (r *DiagnosticRegistry) GetDiagnostics(uri protocol.DocumentUri) []protocol.Diagnostic {
	r.mu.Lock()
	defer r.mu.Unlock()

	var combined []protocol.Diagnostic
	if serverDiags, ok := r.allDiags[uri]; ok {
		for _, diags := range serverDiags {
			combined = append(combined, diags...)
		}
	}
	return combined
}
