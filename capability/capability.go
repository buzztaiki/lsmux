package capability

func IsMethodSupported(method string, supportedCaps map[string]struct{}) bool {
	methodCap, useCap := MethodToCapability[method]
	if !useCap {
		return true
	}

	_, supported := supportedCaps[methodCap]
	return supported
}

// CollectSupported returns a map of dot notated capability to whether it's supported or not.
func CollectSupported(kvCaps map[string]any) map[string]struct{} {
	res := map[string]struct{}{}
	collectSupported("", kvCaps, res)
	return res
}

func collectSupported(prefix string, kvCaps map[string]any, res map[string]struct{}) {
	for k, v := range kvCaps {
		switch v := v.(type) {
		case map[string]any:
			res[prefix+k] = struct{}{}
			collectSupported(prefix+k+".", v, res)
		case bool:
			if v {
				res[prefix+k] = struct{}{}
			}
		default:
			if v != nil {
				res[prefix+k] = struct{}{}
			}
		}
	}
}
