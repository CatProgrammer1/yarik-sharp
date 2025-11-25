//go:build windows

package main

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
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
			case int64:
				time.Sleep(time.Duration(t * int64(time.Millisecond)))
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
			argsCheck(v, 1, 1, "any")
			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]

			a := v[0]
			switch a := a.(type) {
			case *orderedmap.OrderedMap[any, any]:
				return []any{int64(a.Len())}
			case string:
				return []any{int64(len(a))}
			case *StructObject:
				layout := a.Layout()
				if len(layout) == 0 {
					return []any{int64(0)}
				}

				lastFieldLayout := layout[len(layout)-1]

				return []any{int64(lastFieldLayout.Offset + lastFieldLayout.Size)}
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
			argsCheck(v, 2, 2, "string", "bool")
			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]

			str := v[0].(string)
			isint := v[1].(bool)

			if !isint {
				n, err := strconv.ParseFloat(str, 64)
				switch err {
				case strconv.ErrSyntax:
					throw("Syntax error while trying to parse number value.", x, y)
				case strconv.ErrRange:
					throw("Number value is out of range.", x, y)
				}
				return []any{n}
			} else {
				n, err := strconv.ParseInt(str, 0, 64)
				switch err {
				case strconv.ErrSyntax:
					throw("Syntax error while trying to parse number value.", x, y)
				case strconv.ErrRange:
					throw("Number value is out of range.", x, y)
				}

				return []any{n}
			}
		},

		"bytestostr": func(v ...any) []any {
			argsCheck(v, 1, 1, "table")

			v = v[BUILTIN_SPECIALS:]

			b := v[0].(*orderedmap.OrderedMap[Cell, *Cell])

			return []any{string(mapToSlice[byte](b))}
		},

		"unicodetostr": func(v ...any) []any {
			argsCheck(v, 1, 1, "int")

			v = v[BUILTIN_SPECIALS:]

			r := rune(v[0].(int64))

			return []any{string(r)}
		},

		"syscallnt": func(v ...any) []any {
			argsCheck(v, 2, 2, "string", "table")

			x, y := v[0].(int), v[1].(int)
			inter := v[2].(*Interpreter)

			v = v[BUILTIN_SPECIALS:]

			procName := v[0].(string)
			paramsMap := v[1].(*orderedmap.OrderedMap[Cell, *Cell])

			params := make([]uintptr, paramsMap.Len())
			buffers := make([]any, paramsMap.Len())
			i := 0

			for _, v := range paramsMap.AllFromFront() {
				ptr, buf := valueToPtr(v.Get(), x, y)
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
				return []any{uintptr(0), uintptr(0), procerr}
			}

			r1, r2, err := proc.Call(params...)

			for _, ptr := range params {
				value := inter.CurrentScope.GetCellWithAddress(unsafe.Pointer(ptr))
				if value == nil {
					continue
				}

				instance, ok := value.Get().(*StructObject)
				if !ok {
					continue
				}

				layout := instance.Layout()

				instance.FromMemoryLayout(layout)
			}

			return []any{r1, r2, err}
		},

		"call": func(v ...any) []any {

			argsCheck(v, 3, 3, "string", "string", "table")

			x, y := v[0].(int), v[1].(int)
			inter := v[2].(*Interpreter)

			v = v[BUILTIN_SPECIALS:]

			dllname := v[0].(string)
			procName := v[1].(string)
			paramsMap := v[2].(*orderedmap.OrderedMap[Cell, *Cell])

			params := make([]uintptr, paramsMap.Len())
			buffers := make([]any, paramsMap.Len())
			i := 0

			for _, v := range paramsMap.AllFromFront() {
				ptr, buf := valueToPtr(v.Get(), x, y)
				if buf != nil {
					buffers[i] = buf
				}

				params[i] = ptr
				i++
			}

			ntdll := syscall.NewLazyDLL(dllname)
			proc := ntdll.NewProc(procName)

			procerr := proc.Find()
			if procerr != nil {
				return []any{uintptr(0), uintptr(0), procerr}
			}

			r1, r2, err := proc.Call(params...)

			for _, ptr := range params {
				value := inter.CurrentScope.GetCellWithAddress(unsafe.Pointer(ptr))
				if value == nil {
					continue
				}

				instance, ok := value.Get().(*StructObject)
				if !ok {
					continue
				}

				layout := instance.Layout()

				instance.FromMemoryLayout(layout)
			}

			return []any{r1, r2, err}
		},

		"ptr": func(v ...any) []any {
			argsCheck(v, 1, 1, "int")

			v = v[BUILTIN_SPECIALS:]

			return []any{uintptr(v[0].(int64))}
		},

		"pvoid": func(v ...any) []any {
			argsCheck(v, 1, 1, "int")

			v = v[BUILTIN_SPECIALS:]

			return []any{unsafe.Pointer(uintptr(v[0].(int64)))}
		},
	}
)

func valueToPtr(v any, x, y int) (uintptr, any) {
	switch val := v.(type) {
	case float64:
		fmt.Println("\n\nNISDADSD FLOATT\n\n")
		return uintptr(math.Float64bits(val)), val
	case int64:
		return uintptr(val), val
	case ValuePtr:
		return uintptr(val), val
	case uintptr:
		return val, nil
	case unsafe.Pointer:
		return uintptr(val), val
	case *StructObject:
		return val.Address(), val.LastMem
	case []any:
		return uintptr(unsafe.Pointer(&val[0])), val
	case string:
		utf16p, _ := syscall.UTF16FromString(val)

		return uintptr(unsafe.Pointer(&utf16p[0])), utf16p
	case nil:
		return 0, nil
	default:
		fmt.Printf("%T\n", val)
		throw("Unsupported type.", x, y)
	}
	fmt.Println("\n\n\nSUPER NIGGGAAAAAAA\n\n\n")
	return 0, nil
}
