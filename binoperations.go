package main

var (
	binOperations = map[string]func(a, b any, x, y int) any{
		"add": func(a, b any, x, y int) any {
			if checkDataType("number", a) && checkDataType("number", b) {
				if checkType[float64](a) && checkType[float64](b) {
					return a.(float64) + b.(float64)
				} else if checkType[int64](a) && checkType[int64](b) {
					return a.(int64) + b.(int64)
				} else if checkType[float64](a) && checkType[int64](b) ||
					checkType[int64](a) && checkType[float64](b) {
					return mustNTOF64(a) + mustNTOF64(b)
				}
			} else if checkType[string](a) && checkType[string](b) {

				return a.(string) + b.(string)
			} else if checkType[string](a) && checkDataType("number", b) ||
				checkDataType("number", a) && checkType[string](b) {

				return format(a, b)
			}
			throw("Unable to perform operation add or concat on non-number and non-string values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"sub": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation sub on non-number values.", x, y)
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) - b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) - b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) - mustNTOF64(b)
			}
			throw("Unable to perform operation sub on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"div": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation div on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) / b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) / b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) / mustNTOF64(b)
			}
			throw("Unable to perform operation div on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"mul": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation mul on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) * b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) * b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) * mustNTOF64(b)
			}
			throw("Unable to perform operation mul on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},

		"bitor": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation sub on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) | b.(int64)
			}
			throw("Can only perform operation bitor on integer values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},

		"greater": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation greater non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) > b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) > b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) > mustNTOF64(b)
			}
			return nil
		},
		"less": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation less non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) < b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) < b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) < mustNTOF64(b)
			}
			return nil
		},
		"greatereq": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation greater-equals non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) >= b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) >= b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) >= mustNTOF64(b)
			}
			return nil
		},
		"lesseq": func(a, b any, x, y int) any {
			if !(checkDataType("number", a) && checkDataType("number", b)) {
				throw("Unable to perform operation less-equals non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			}
			if checkType[float64](a) && checkType[float64](b) {
				return a.(float64) <= b.(float64)
			} else if checkType[int64](a) && checkType[int64](b) {
				return a.(int64) <= b.(int64)
			} else if checkType[float64](a) && checkType[int64](b) ||
				checkType[int64](a) && checkType[float64](b) {
				return mustNTOF64(a) <= mustNTOF64(b)
			}
			return nil
		},

		"equals": func(a, b any, x, y int) any {

			return a == b
		},
		"notequals": func(a, b any, x, y int) any {
			return a != b
		},

		"valbits": func(a, b any, x, y int) any {
			bits, ok := b.(int64)
			if !ok {
				println("SHO", b)
				throw("Bits count must be an integer value, got '%s'.", x, y, getValueType(b))
			}

			switch n := a.(type) {
			case int64, int32, int16, int8:
				return toInt(toInt64(n), int(bits))
			case float64:
				switch bits {
				case 32:
					return float32(n)
				case 64:
					return n
				}
			case float32:
				switch bits {
				case 32:
					return n
				case 64:
					return float64(n)
				}
			}
			throw("Cannot convert '%s' to a value with %d bits.", x, y, getValueType(a), bits)
			return nil
		},
	}
)
