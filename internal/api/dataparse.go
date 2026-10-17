package api

import (
	"encoding/json"
	"sort"
)

// firstArray finds the first JSON array inside the "data" object of a
// response, preferring the named keys. It lets list-style endpoints whose
// exact wrapper key is unverified still produce results.
func firstArray(data json.RawMessage, prefer ...string) []json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal(data, &obj) != nil {
		var arr []json.RawMessage
		if json.Unmarshal(data, &arr) == nil {
			return arr
		}
		return nil
	}
	for _, k := range prefer {
		var arr []json.RawMessage
		if json.Unmarshal(obj[k], &arr) == nil && arr != nil {
			return arr
		}
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var arr []json.RawMessage
		if json.Unmarshal(obj[k], &arr) == nil && arr != nil {
			return arr
		}
	}
	return nil
}

// field returns the first non-empty string/number found under any of names.
func field(obj map[string]any, names ...string) string {
	for _, n := range names {
		if v, ok := obj[n]; ok && v != nil {
			if s := FormatValue(v); s != "" {
				return s
			}
		}
	}
	return ""
}

type dataEnvelope struct {
	Data json.RawMessage `json:"data"`
}
