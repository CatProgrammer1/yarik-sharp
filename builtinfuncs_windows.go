//go:build windows

package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
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

			x, y := v[0].(int), v[1].(int)

			v = v[2:]

			val := v[0]
			switch val := val.(type) {
			case float64:
				return []any{PTR(val)}
			default:
				ptr, _ := valueToPtr(val, x, y)
				return []any{PTR(ptrToFloat(ptr))}
				/*case string:
					utf16p, _ := syscall.UTF16PtrFromString(val)

					return []any{PTR(ptrToFloat(uintptr(unsafe.Pointer(utf16p))))}
				case *StructObject:
					return []any{PTR(ptrToFloat(uintptr(unsafe.Pointer(&val.ToMemoryLayout(val.Layout())[0]))))}*/
			}
		},

		"syscallnt": func(v ...any) []any {
			argsCheck(v, 2, 2, "string", "table")

			x, y := v[0].(int), v[1].(int)
			v = v[2:]
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
			for _, v := range paramsMap.AllFromFront() {
				instance, ok := v.(*StructObject)
				if ok {
					instance.FromMemoryLayout(instance.Layout())
				}
			}

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
				return []any{PTR(0), PTR(0), err}
			}

			tempAllocs := []uintptr{}

			argsAny := mapToSliceAny(argsTable)
			args := make([]uintptr, len(argsAny))

			for i, v := range argsAny {
				ptr, _ := valueToPtr(v, x, y)
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

			//Эээ ну типа если в побайтовую версию структуры чото записали то надо же блин коммитнуть изменения на основную
			for _, v := range argsAny {
				instance, ok := v.(*StructObject)
				if ok {
					instance.FromMemoryLayout(instance.Layout())
				}
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
