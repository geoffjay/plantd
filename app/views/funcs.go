// Package views provides view rendering functionality.
package views

import "errors"

// Args creates a map from key-value pairs for template rendering.
func Args(kvs ...interface{}) (map[string]interface{}, error) {
	if len(kvs)%2 != 0 {
		return nil, errors.New("args requires even number of arguments")
	}
	m := make(map[string]interface{})
	for i := 0; i < len(kvs); i += 2 {
		s, ok := kvs[i].(string)
		if !ok {
			return nil, errors.New("even args to args must be strings")
		}
		m[s] = kvs[i+1]
	}
	return m, nil
}
