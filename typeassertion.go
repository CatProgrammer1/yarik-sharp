package main

func assertType(v any, t string) (any, bool) {
	switch t {
	case "int":
		switch v := v.(type) {
		case int64:
			return v, true
		case float64:
			return int64(v), true
		case uintptr:
			return int64(v), true
		}
	case "float":
		switch v := v.(type) {
		case float64:
			return v, true
		case int64:
			return float64(v), true
		case uintptr:
			return float64(v), true
		}
	case "pointer":
		switch v := v.(type) {
		case uintptr:
			return v, true
		case int64:
			return uintptr(v), true
		case float64:
			return uintptr(v), true
		}
	}
	return v, false
}
