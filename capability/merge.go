package capability

import (
	"reflect"
	"slices"
)

// Merge merges capabilities from src into dst.
// dst should not be nil.
func Merge(dst, src map[string]any) {
	if dst == nil {
		panic("dst should not be nil")
	}

	for k, v := range src {
		if dv := dst[k]; dv == nil {
			dst[k] = v
			continue
		}

		switch sv := v.(type) {
		case map[string]any:
			if dv, ok := dst[k].(map[string]any); ok {
				Merge(dv, sv)
			}
		case []any:
			if dv, ok := dst[k].([]any); ok {
				for _, sx := range sv {
					if !slices.ContainsFunc(dv, func(dx any) bool { return reflect.DeepEqual(dx, sx) }) {
						dv = append(dv, sx)
					}
				}
				dst[k] = dv
			}
		}
	}
}
