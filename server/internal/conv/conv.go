package conv

import (
	"encoding/json"
	"strconv"
)

func AsInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func Stringify(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strconv.FormatInt(int64(t), 10), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case int:
		return strconv.Itoa(t), true
	case json.Number:
		return t.String(), true
	default:
		return "", false
	}
}

func AsMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func AsList(v any) []any {
	list, _ := v.([]any)
	return list
}

func JSONText(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func ParseJSONMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

func ParseJSONAny(s string) any {
	if s == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil
	}
	return v
}
