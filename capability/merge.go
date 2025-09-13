package capability

import (
	"reflect"
	"slices"
)

func Merge(dst, src map[string]any) {
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
