package api

import (
	"fmt"
)

func UnmarshalPluginConfigMap(v interface{}) (map[string]map[string]interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%T is not a plugin config map", v)
	}

	result := make(map[string]map[string]interface{})
	for k, v := range m {
		val, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("key %s (%T) is not a map", k, v)
		}

		result[k] = val
	}

	return result, nil
}
