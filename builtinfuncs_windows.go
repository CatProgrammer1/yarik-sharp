//go:build windows

package main

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/elliotchance/orderedmap/v3"
)

const (
	BUILTIN_SPECIALS = 3
)

var (
	builtinFuncs = map[string]func(v ...any) []any{
		"OS_NAME": func(v ...any) []any {
			return []any{runtime.GOOS}
		},

		"print": func(v ...any) []any {
			fmt.Println(format(v[BUILTIN_SPECIALS:]...))
			return nil
		},

		"delete": func(v ...any) []any {
			argsCheck(v, 2, 2, "table", "any")

			v = v[BUILTIN_SPECIALS:]

			table := v[0].(*orderedmap.OrderedMap[any, any])
			key := v[1]

			table.Delete(key)
			return nil
		},

		"sleep": func(v ...any) []any {
			x, y := v[0].(int), v[1].(int)

			if len(v) == 0 {
				throw("Function must have one argument.", x, y)
			}

			v = v[BUILTIN_SPECIALS:]

			switch t := v[0].(type) {
			case float64:
				time.Sleep(time.Duration(t * float64(time.Second)))
			default:
				throw("Time value must be a number.", x, y)
			}
			return nil
		},

		"throw": func(v ...any) []any {
			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]
			if len(v) <= 0 {
				throw("Function requires one or more arguments.", x, y)
			}

			throw(format(v...), x, y)
			return nil
		},

		"len": func(v ...any) []any {
			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]
			if len(v) <= 0 && len(v) > 1 {
				throw("Function requires only one argument.", x, y)
			}

			a := v[0]
			switch a := a.(type) {
			case *orderedmap.OrderedMap[any, any]:
				return []any{float64(a.Len())}
			case string:
				return []any{float64(len(a))}
			case *StructObject:
				layout := a.Layout()
				if len(layout) == 0 {
					return []any{float64(0)}
				}

				lastFieldLayout := layout[len(layout)-1]

				return []any{float64(lastFieldLayout.Offset + lastFieldLayout.Size)}
			default:
				throw("Cannot get lenght of non-string, non-table or non-instance value.", x, y)
			}
			return nil
		},
		"tostr": func(v ...any) []any {
			return []any{format(v[BUILTIN_SPECIALS:]...)}
		},
		"gettype": func(v ...any) []any {
			argsCheck(v, 1, 1, "any")

			v = v[BUILTIN_SPECIALS:]

			return []any{getValueType(v[0])}
		},
		"tonum": func(v ...any) []any {
			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]
			if len(v) <= 0 && len(v) > 1 {
				throw("Function requires only one argument.", x, y)
			}

			str, ok := v[0].(string)
			if !ok {
				throw("Argument must be a string value.", x, y)
			}

			n, err := strconv.ParseFloat(str, 64)
			switch err {
			case strconv.ErrSyntax:
				throw("Syntax error while trying to parse number value.", x, y)
			case strconv.ErrRange:
				throw("Number value is out of range.", x, y)
			}

			return []any{n}
		},

		"bytestostr": func(v ...any) []any {
			argsCheck(v, 1, 1, "table")

			v = v[BUILTIN_SPECIALS:]

			b := v[0].(*orderedmap.OrderedMap[Cell, *Cell])

			return []any{string(mapToSlice[byte](b))}
		},

		"unicodetostr": func(v ...any) []any {
			argsCheck(v, 1, 1, "number")

			v = v[BUILTIN_SPECIALS:]

			r := rune(v[0].(float64))

			return []any{string(r)}
		},

		"syscallnt": func(v ...any) []any {
			argsCheck(v, 2, 2, "string", "table")

			x, y := v[0].(int), v[1].(int)
			inter := v[2].(*Interpreter)

			v = v[BUILTIN_SPECIALS:]

			procName := v[0].(string)
			paramsMap := v[1].(*orderedmap.OrderedMap[any, any])

			params := make([]uintptr, paramsMap.Len())
			buffers := make([]any, paramsMap.Len()) // ← Храним буферы здесь
			i := 0

			// Подготавливаем параметры и сохраняем буферы
			for _, v := range paramsMap.AllFromFront() {
				ptr, buf := valueToPtr(v, x, y)
				if buf != nil {
					buffers[i] = buf
				}

				params[i] = ptr
				i++
			}

			ntdll := syscall.NewLazyDLL("ntdll.dll")
			proc := ntdll.NewProc(procName)

			procerr := proc.Find()
			if procerr != nil {
				return []any{PTR(0), PTR(0), procerr}
			}

			r1, r2, err := proc.Call(params...)

			// Обновляем структуры из памяти
			for _, ptr := range params {
				value := inter.CurrentScope.GetWithAddress(ptr)
				if value == nil {
					continue
				}

				switch instance := value.(type) {
				case *StructObject:
					layout := instance.Layout()

					instance.FromMemoryLayout(layout)
				}
			}

			return []any{PTR(ptrToFloat(r1)), PTR(ptrToFloat(r2)), err}
		},

		"ptrconv": func(v ...any) []any {
			argsCheck(v, 2, 2, "number", "string")

			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]

			ptrUintptr := floatToPtr(v[0].(float64))
			if ptrUintptr == 0 {
				throw("Void address.", x, y)
			}
			if ptrUintptr%unsafe.Alignof(int(0)) != 0 {
				throw("Unaligned address.", x, y)
			}

			ptr := unsafe.Pointer(ptrUintptr)
			dataType := strings.ToLower(v[1].(string))

			switch dataType {
			case "number":
				throw("Use int, float etc. instead of number.", x, y)
			case "int":
				return []any{float64(*(*int)(ptr))}
			case "float":
				return []any{*(*float64)(ptr)}
			/*case "table":
				throw("Use array instead of table.", x, y)
			case "array":
			return []any{sliceToMap(unsafe.Slice((*byte)(ptr), arrayLenght))}*/
			default:
				throw("Invalid type to convert into.", x, y)
			}

			return []any{nil}
		},
	}
)

func valueToPtr(v any, x, y int) (uintptr, any) {
	switch val := v.(type) {
	case float64:
		if math.Floor(val) == val {
			return uintptr(uint32(val)), nil
		} else {
			return floatToPtr(val), nil
		}
	case PTR:
		return floatToPtr(float64(val)), nil
	case unsafe.Pointer:
		return uintptr(val), nil
	case *StructObject:
		layout := val.Layout()
		val.ToMemoryLayout(layout)

		return uintptr(unsafe.Pointer(&val.LastMem[0])), val.LastMem
	case []any:
		return uintptr(unsafe.Pointer(&val[0])), val
	case string:
		utf16p, _ := syscall.UTF16FromString(val)

		return uintptr(unsafe.Pointer(&utf16p[0])), utf16p
	case nil:
		return 0, nil
	default:
		throw("Unsupported type.", x, y)
	}
	return 0, nil
}
