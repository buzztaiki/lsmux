package capability

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMerge(t *testing.T) {
	type kv = map[string]any

	for _, tt := range []struct {
		name string
		dst  kv
		src  kv
		want kv
	}{
		{
			name: "just copy",
			dst:  kv{},
			src:  kv{"items": []any{1, 2, 3}, "key": "value"},
			want: kv{"items": []any{1, 2, 3}, "key": "value"},
		},
		{
			name: "merge",
			dst:  kv{"key1": "k1", "key2": "k2"},
			src:  kv{"key1": "k1-2", "key3": "k3"},
			want: kv{"key1": "k1", "key2": "k2", "key3": "k3"},
		},
		{
			name: "append and dedup slice",
			dst:  kv{"items": []any{1, 2}},
			src:  kv{"items": []any{1, 3, "4"}},
			want: kv{"items": []any{1, 2, 3, "4"}},
		},
		{
			name: "dedup map value",
			dst: kv{
				"items": []any{
					kv{"key": "k1", "value": "v1"},
					kv{"key": "k2", "value": "v2"},
				}},
			src: kv{
				"items": []any{
					kv{"key": "k1", "value": "v1"},
					kv{"key": "k2", "value": "v2-2"},
					kv{"key": "k3", "value": "v3"},
				}},
			want: kv{
				"items": []any{
					kv{"key": "k1", "value": "v1"},
					kv{"key": "k2", "value": "v2"},
					kv{"key": "k2", "value": "v2-2"},
					kv{"key": "k3", "value": "v3"},
				}},
		},
		{
			name: "complex",
			dst: kv{
				"items": []any{1, 2},
				"map":   kv{"key1": "k1", "key2": "k2"},
				"composite": kv{
					"items": []any{1, 2},
					"map":   kv{"key1": "k1", "key2": "k2"},
				},
			},
			src: kv{
				"items": []any{1, 3},
				"map":   kv{"key1": "k1-2", "key3": "k3"},
				"composite": kv{
					"items": []any{1, 3},
					"map":   kv{"key1": "k1-2", "key3": "k3"},
				},
			},
			want: kv{
				"items": []any{1, 2, 3},
				"map":   kv{"key1": "k1", "key2": "k2", "key3": "k3"},
				"composite": kv{
					"items": []any{1, 2, 3},
					"map":   kv{"key1": "k1", "key2": "k2", "key3": "k3"},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			Merge(tt.dst, tt.src)
			if diff := cmp.Diff(tt.want, tt.dst); diff != "" {
				t.Errorf("unexpected merge result (-want +got):\n%s", diff)
			}
		})
	}
}
