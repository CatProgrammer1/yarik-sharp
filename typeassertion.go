package main

func assertType(v any, targetType string) (any, bool) {
	switch targetType {
	case "i64", "i32", "i16", "i8":
		return toInt(toInt64(v), -twoDigitStr(targetType[1:])), true
	case "u64", "u32", "u16", "u8":
		return toUint(toUint64(v), twoDigitStr(targetType[1:])), true
	case "f64":
		return mustNTOF64(v), true
	case "f32":
		return float32(mustNTOF64(v)), true
	case "pointer":
		return uintptr(toUint64(v)), true
	}
	return v, false
}
