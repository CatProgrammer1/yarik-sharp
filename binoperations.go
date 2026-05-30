package main

var (
	binOperations = map[string]func(inter *Interpreter, a, b any, x, y int) any{
		"add": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation add or concat on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				bits := twoDigitStr(aType[1:])

				return toInt(toInt64(a)+toInt64(b), -bits)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				bits := twoDigitStr(aType[1:])

				return toUint(toUint64(a)+toUint64(b), bits)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				bits := twoDigitStr(aType[1:])

				if bits == 32 {
					return float32(mustNTOF64(a) + mustNTOF64(b))
				}
				return a.(float64) + b.(float64)
			} else if checkType[string](a) && checkType[string](b) {

				return a.(string) + b.(string)
			}
			throw(inter.CurrentFileName, "Unable to perform operation add or concat on non-number and non-string values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"sub": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation sub on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				bits := twoDigitStr(aType[1:])

				return toInt(toInt64(a)-toInt64(b), -bits)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				bits := twoDigitStr(aType[1:])

				return toUint(toUint64(a)-toUint64(b), bits)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				bits := twoDigitStr(aType[1:])

				if bits == 32 {
					return float32(mustNTOF64(a) - mustNTOF64(b))
				}
				return a.(float64) - b.(float64)
			}
			throw(inter.CurrentFileName, "Unable to perform operation sub on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"div": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation div on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				bits := twoDigitStr(aType[1:])

				return toInt(toInt64(a)/toInt64(b), -bits)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				bits := twoDigitStr(aType[1:])

				return toUint(toUint64(a)/toUint64(b), bits)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				bits := twoDigitStr(aType[1:])

				if bits == 32 {
					return float32(mustNTOF64(a) / mustNTOF64(b))
				}
				return a.(float64) / b.(float64)
			}
			throw(inter.CurrentFileName, "Unable to perform operation div on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"mul": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation sub on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				bits := twoDigitStr(aType[1:])

				return toInt(toInt64(a)*toInt64(b), -bits)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				bits := twoDigitStr(aType[1:])

				return toUint(toUint64(a)*toUint64(b), bits)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				bits := twoDigitStr(aType[1:])

				if bits == 32 {
					return float32(mustNTOF64(a) * mustNTOF64(b))
				}
				return a.(float64) * b.(float64)
			}
			throw(inter.CurrentFileName, "Unable to perform operation sub on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},

		"bitor": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation bitor on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				bits := twoDigitStr(aType[1:])

				return toInt(toInt64(a)|toInt64(b), -bits)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				bits := twoDigitStr(aType[1:])

				return toUint(toUint64(a)|toUint64(b), bits)
			}
			throw(inter.CurrentFileName, "Unable to perform operation bitor on non-integer values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},

		"greater": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation greater on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				return toInt64(a) > toInt64(b)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				return toUint64(a) > toUint64(b)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				return mustNTOF64(a) > mustNTOF64(b)
			}
			throw(inter.CurrentFileName, "Unable to perform operation greater on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"less": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation less on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				return toInt64(a) < toInt64(b)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				return toUint64(a) < toUint64(b)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				return mustNTOF64(a) < mustNTOF64(b)
			}
			throw(inter.CurrentFileName, "Unable to perform operation less on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"greatereq": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation greater/equals on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				return toInt64(a) >= toInt64(b)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				return toUint64(a) >= toUint64(b)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				return mustNTOF64(a) >= mustNTOF64(b)
			}
			throw(inter.CurrentFileName, "Unable to perform operation greater/equals on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},
		"lesseq": func(inter *Interpreter, a, b any, x, y int) any {
			aType, bType := getValueType(a), getValueType(b)
			if aType != bType {
				throw(inter.CurrentFileName, "Unable to perform operation less/equals on values with different data types: '%s' and '%s'.", x, y, aType, bType)
			}

			if checkDataType("int", a) && checkDataType("int", b) {
				return toInt64(a) <= toInt64(b)
			} else if checkDataType("uint", a) && checkDataType("uint", b) {
				return toUint64(a) <= toUint64(b)
			} else if checkDataType("float", a) && checkDataType("float", b) {
				return mustNTOF64(a) <= mustNTOF64(b)
			}
			throw(inter.CurrentFileName, "Unable to perform operation less/equals on non-number values: %s and %s.", x, y, getValueType(a), getValueType(b))
			return nil
		},

		"equals": func(inter *Interpreter, a, b any, x, y int) any {
			return a == b
		},
		"notequals": func(inter *Interpreter, a, b any, x, y int) any {
			return a != b
		},
	}
)
