package capability

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCollectSupported(t *testing.T) {
	kvCaps := map[string]any{
		"key": true,
		"foo": map[string]any{
			"bar":  true,
			"baz":  false,
			"qux":  nil,
			"quux": "value",
		},
	}
	want := SupportedSet{
		"key":      struct{}{},
		"foo":      struct{}{},
		"foo.bar":  struct{}{},
		"foo.quux": struct{}{},
	}
	got := CollectSupported(kvCaps)
	if cmp.Diff(want, got) != "" {
		t.Errorf("CollectSupported() mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestSupportedSet_IsSupportedMethod(t *testing.T) {
	supportedSet := SupportedSet{
		"executeCommandProvider": struct{}{},
	}

	tests := []struct {
		name   string
		method string
		want   bool
	}{
		{
			name:   "basic",
			method: "workspace/executeCommand",
			want:   true,
		},
		{
			name:   "unsupported method",
			method: "textDocument/rename",
			want:   false,
		},
		{
			name:   "no capability mapping",
			method: "initialize",
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportedSet.IsSupportedMethod(tt.method)
			if got != tt.want {
				t.Errorf("IsSupportedMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}
