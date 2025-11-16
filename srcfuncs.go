package main

func argsCheck(v []any, min, max int, expectedDataTypes ...string) {
	if min == 0 && max == 0 {
		return
	}

	x, y := v[0].(int), v[1].(int)

	if len(v) < min+2 {
		throw("Attempt to pass less arguments to a function call than function actually need, minimum is %d.", x, y, min)
	} else if len(v) > max+2 {
		throw("Attempt to pass more arguments to a function call than function actually need, maximum is %d.", x, y, max)
	} else {
		args := v[2:]

		for i := 0; i < min; i++ {
			expectedDataType := expectedDataTypes[i]

			argument := args[i]

			if !checkDataType(expectedDataType, argument) {
				throw("Invalid argument #%d. Expected %s.", x, y, i+1, expectedDataType)
			}
		}
	}
}

var (
	srcFuncs = map[string][]*FuncDec{}
)
