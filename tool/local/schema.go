package local

import "fmt"

func ValidateArgs(schema map[string]any, args map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	t, _ := schema["type"].(string)
	if t != "" && t != "object" {
		return fmt.Errorf("unsupported schema type %q", t)
	}
	props, _ := schema["properties"].(map[string]any)
	required := toStrings(schema["required"])
	for _, k := range required {
		if _, ok := args[k]; !ok {
			return fmt.Errorf("missing required field %q", k)
		}
	}
	for key, val := range args {
		raw, ok := props[key]
		if !ok {
			continue
		}
		decl, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		expected, _ := decl["type"].(string)
		if expected == "" {
			continue
		}
		if !matchesType(expected, val) {
			return fmt.Errorf("field %q type mismatch: expected %s", key, expected)
		}
	}
	return nil
}

func toStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if ok {
			out = append(out, s)
		}
	}
	return out
}

func matchesType(expected string, val any) bool {
	switch expected {
	case "string":
		_, ok := val.(string)
		return ok
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "integer":
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "number":
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return true
		default:
			return false
		}
	case "object":
		_, ok := val.(map[string]any)
		return ok
	case "array":
		_, ok := val.([]any)
		return ok
	default:
		return true
	}
}
