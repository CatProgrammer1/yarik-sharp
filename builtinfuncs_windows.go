//go:build windows

package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/elliotchance/orderedmap/v3"
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
	MEM_RELEASE            = 0x8000
	PAGE_READWRITE         = 0x04
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	virtualAlloc = kernel32.NewProc("VirtualAlloc")
	virtualFree  = kernel32.NewProc("VirtualFree")

	builtinFuncs = map[string]func(v ...any) []any{
		"OS_NAME": func(v ...any) []any {
			return []any{runtime.GOOS}
		},
		"print": func(v ...any) []any {
			fmt.Println(format(v[2:]...))
			return nil
		},
		"delete": func(v ...any) []any {
			argsCheck(v, 2, 2, "table", "any")

			v = v[2:]

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

			v = v[2:]

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

			v = v[2:]
			if len(v) <= 0 {
				throw("Function requires one or more arguments.", x, y)
			}

			throw(format(v...), x, y)
			return nil
		},
		"len": func(v ...any) []any {
			x, y := v[0].(int), v[1].(int)

			v = v[2:]
			if len(v) <= 0 && len(v) > 1 {
				throw("Function requires only one argument.", x, y)
			}

			a := v[0]
			switch reflect.TypeOf(a).Kind() {
			case reflect.Map:
				return []any{float64(len(a.(map[any]any)))}
			case reflect.String:
				return []any{float64(len(a.(string)))}
			default:
				throw("Cannot get lenght of non-string or non-table value.", x, y)
			}
			return nil
		},
		"tostr": func(v ...any) []any {
			return []any{format(v[2:]...)}
		},
		"gettype": func(v ...any) []any {
			argsCheck(v, 1, 1, "any")

			v = v[2:]

			return []any{getValueType(v[0])}
		},
		"tonum": func(v ...any) []any {
			x, y := v[0].(int), v[1].(int)

			v = v[2:]
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

			v = v[2:]

			b := v[0].(*orderedmap.OrderedMap[any, any])

			return []any{string(mapToSlice[byte](b))}
		},

		"unicodetostr": func(v ...any) []any {
			argsCheck(v, 1, 1, "number")

			v = v[2:]

			r := rune(v[0].(float64))

			return []any{string(r)}
		},

		"ptr": func(v ...any) []any {
			argsCheck(v, 1, 1, "any")

			v = v[2:]

			val := v[0]
			switch val := val.(type) {
			case float64:
				return []any{PTR(val)}
			case *StructObject:
				return []any{PTR(ptrToFloat(uintptr(unsafe.Pointer(&val.ToMemoryLayout(val.Layout())[0]))))}
			}

			return []any{PTR(0)}
		},

		"syscallnt": func(v ...any) []any {
			argsCheck(v, 2, 2, "string", "table")

			x, y := v[0].(int), v[1].(int)

			v = v[2:]

			procName := v[0].(string)
			paramsMap := v[1].(*orderedmap.OrderedMap[any, any])

			params := make([]uintptr, paramsMap.Len())
			i := 0
			for _, v := range paramsMap.AllFromFront() {
				params[i] = valueToPtr(v, x, y)
				i++
			}

			ntdll := syscall.NewLazyDLL("ntdll.dll")
			proc := ntdll.NewProc(procName)

			r1, r2, err := proc.Call(params...)

			return []any{PTR(ptrToFloat(r1)), PTR(ptrToFloat(r2)), err}
		},

		"callbin": func(v ...any) []any {
			argsCheck(v, 2, 2, "string", "table")

			x, y := v[0].(int), v[1].(int)

			v = v[2:]

			asmbPath := v[0].(string)
			argsTable := v[1].(*orderedmap.OrderedMap[any, any])

			asmb, err := os.ReadFile(asmbPath)
			if err != nil {
				return []any{0, 0, err}
			}

			tempAllocs := []uintptr{}

			argsAny := mapToSliceAny(argsTable)
			args := make([]uintptr, len(argsAny))

			for i, v := range argsAny {
				ptr := valueToPtr(v, x, y)
				if ptr == 0 {
					for _, t := range tempAllocs {
						virtualFree.Call(t)
					}
					return []any{PTR(0), PTR(0), nil}
				}
				tempAllocs = append(tempAllocs, ptr)

				args[i] = ptr
			}

			addr, _, err := virtualAlloc.Call(0, uintptr(len(asmb)), uintptr(MEM_RESERVE|MEM_COMMIT), uintptr(PAGE_EXECUTE_READWRITE))
			if addr == 0 {
				return []any{PTR(0), PTR(0), err}
			}

			ptr := unsafe.Pointer(addr)
			dst := unsafe.Slice((*byte)(ptr), len(asmb))
			copy(dst, asmb)

			r1, r2, errno := syscall.SyscallN(addr, args...)

			for _, t := range tempAllocs {
				virtualFree.Call(t, 0, uintptr(MEM_RELEASE))
			}

			virtualFree.Call(addr, 0, uintptr(MEM_RELEASE))
			return []any{PTR(ptrToFloat(r1)), PTR(ptrToFloat(r2)), error(errno)}
		},

		"ptrconv": func(v ...any) []any {
			argsCheck(v, 2, 2, "number", "string")

			x, y := v[0].(int), v[1].(int)

			v = v[2:]

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
				return []any{*(*int)(ptr)}
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

func valueToPtr(v any, x, y int) uintptr {
	var ptr uintptr

	switch val := v.(type) {
	case float64:
		ptr = floatToPtr(val)
	case PTR:
		ptr = floatToPtr(float64(val))
	case *StructObject:
		mem := val.ToMemoryLayout(val.Layout())

		ptr = uintptr(unsafe.Pointer(&mem[0]))
	case string:
		var b []byte
		if strings.HasSuffix(val, "W") {
			u16 := utf16.Encode([]rune(val))
			b = make([]byte, len(u16)*2+2)
			for i, r := range u16 {
				b[i*2] = byte(r)
				b[i*2+1] = byte(r >> 8)
			}

			b[len(b)-2] = 0
			b[len(b)-1] = 0
		} else {
			b = append([]byte(val), 0)

			fmt.Println(b)
		}

		p, _, err := virtualAlloc.Call(0, uintptr(len(b)), uintptr(MEM_RESERVE|MEM_COMMIT), uintptr(PAGE_READWRITE))
		if p == 0 {
			throw(fmt.Sprintf("virtualAlloc failed: %v", err), x, y)
		}

		pb := unsafe.Slice((*byte)(unsafe.Pointer(p)), len(b))
		copy(pb, b)
		ptr = p

	default:
		throw("Unsupported type.", x, y)
	}

	return ptr
}
