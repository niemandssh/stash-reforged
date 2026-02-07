package api

import (
	"fmt"
)

func UnmarshalBoolMap(v interface{}) (map[string]bool, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%T is not a map", v)
	}

	result := make(map[string]bool)
	for k, v := range m {
		key := k
		val, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("key %s (%T) is not a bool", k, v)
		}

		result[key] = val
	}

	return result, nil
}
