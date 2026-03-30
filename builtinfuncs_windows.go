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

		"lockOSThread": func(v ...any) []any {
			runtime.LockOSThread()

			return nil
		},

		"unlockOSThread": func(v ...any) []any {
			runtime.UnlockOSThread()

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
			case *Map:
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
		"time": func(v ...any) []any {
			return []any{time.Now().UnixMilli()}
		},
		"strformat": func(v ...any) []any {
			return []any{format(v[BUILTIN_SPECIALS:]...)}
		},
		"gettype": func(v ...any) []any {
			argsCheck(v, 1, 1, "any")

			v = v[BUILTIN_SPECIALS:]

			return []any{getValueType(v[0])}
		},
		"numformat": func(v ...any) []any {
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

		"string": func(v ...any) []any {
			argsCheck(v, 1, 1, "table")

			v = v[BUILTIN_SPECIALS:]

			b := v[0].(*Map)
			bstring := []byte{}

		APPEND:
			for _, v := range b.AllFromFront() {
				switch v := v.Get().(type) {
				case int64:
					bstring = append(bstring, byte(v))
				default:
					break APPEND
				}
			}

			return []any{string(bstring)}
		},

		"unicodetostr": func(v ...any) []any {
			argsCheck(v, 1, 1, "int")

			v = v[BUILTIN_SPECIALS:]

			r := rune(v[0].(int64))

			return []any{string(r)}
		},

		"bytes": func(v ...any) []any {
			argsCheck(v, 1, 1, "string")

			v = v[BUILTIN_SPECIALS:]

			str := v[0].(string)

			slice, err := syscall.ByteSliceFromString(str)
			handle(err)

			m := &Map{
				OrderedMap: orderedmap.NewOrderedMap[any, *Cell](),
				Bits:       8,
				Pointers:   []any{},
				Layout:     []string{},
				Mem:        []byte{},
			}

			for i, v := range slice {
				m.Set(int64(i), CLPTR(int64(v)))
			}
			m.ToMemory()

			return []any{
				m,
			}
		},

		//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!------------------------------------------------!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

		/*"syscallnt": func(v ...any) []any {
		argsCheck(v, 2, 2, "string", "table")

		x, y := v[0].(int), v[1].(int)
		inter := v[2].(*Interpreter)

		v = v[BUILTIN_SPECIALS:]

		procName := v[0].(string)
		paramsMap := v[1].(*Map)

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

		/*for _, ptr := range params {
			value := inter.CurrentScope.GetCellWithAddress(unsafe.Pointer(ptr))
			if value == nil {
				continue
			}

			switch value := value.Get().(type) {
			case *StructObject:
				layout := value.Layout()

				value.FromMemoryLayout(layout)
			case *Map:
				value.FromMemory()
			}
		}*/

		//refreshPointerValues(inter, params)

		//return []any{r1, r2, err}
		//},

		"closehandle": func(v ...any) []any {
			argsCheck(v, 1, 1, "ptr")

			v = v[BUILTIN_SPECIALS:]

			ptr := v[0].(uintptr)

			err := syscall.CloseHandle(syscall.Handle(ptr))

			return []any{err}
		},

		/*"kernelcall": func(v ...any) []any {
		argsCheck(v, 2, 2, "string", "table")

		x, y := v[0].(int), v[1].(int)
		inter := v[2].(*Interpreter)

		v = v[BUILTIN_SPECIALS:]

		procName := v[0].(string)
		paramsMap := v[1].(*Map)

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

		kernel := syscall.NewLazyDLL("kernel32.dll")
		proc := kernel.NewProc(procName)

		procerr := proc.Find()
		if procerr != nil {
			return []any{uintptr(0), uintptr(0), procerr}
		}

		r1, r2, err := proc.Call(params...)

		/*for _, ptr := range params {
			value := inter.CurrentScope.GetCellWithAddress(unsafe.Pointer(ptr))
			if value == nil {
				continue
			}

			switch value := value.Get().(type) {
			case *StructObject:
				layout := value.Layout()

				value.FromMemoryLayout(layout)
			case *Map:
				value.FromMemory()
			}
		}*/

		//refreshPointerValues(inter, params)

		//return []any{r1, r2, err}
		//},
	}
)

func refreshPointerValues(inter *Interpreter, ptrs []uintptr) {
	for _, ptr := range ptrs {
		value := inter.CurrentScope.GetCellWithAddress(unsafe.Pointer(ptr))
		if value == nil {
			continue
		}

		switch value := value.Get().(type) {
		case *StructObject:
			layout := value.Layout()

			value.FromMemoryLayout(layout)
		case *Map:
			value.FromMemory()
		}
	}
}

func valueToPtr(v any, x, y int) (uintptr, any) {
	switch val := v.(type) {
	case float64:
		return uintptr(math.Float64bits(val)), val
	case int64:
		return uintptr(val), val
	case uintptr:
		return val, nil
	case unsafe.Pointer:
		return uintptr(val), val
	case *StructObject:
		return val.Address(), val.LastMem
	case *Map:
		return val.Address(), val.Mem
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
	return 0, nil
}

func syscallAddress(inter *Interpreter, node Node, argsLen uint, argsValues [][]Node, addr uintptr) (uintptr, uintptr, error) { //go run yks run test.yks
	args := inter.CookValues(argsLen, argsValues, node.Position(), node.Line())

	params := make([]uintptr, len(args))
	buffers := make([]any, len(args))
	i := 0

	//println("Debug 2")
	for _, v := range args {
		ptr, buf := valueToPtr(v, node.Position(), node.Line())
		if buf != nil {
			buffers[i] = buf
		}

		params[i] = ptr
		i++
	}
	//println("debug 2 ended")

	//println("debug 3", addr, node)

	

	r1, r2, err := syscall.SyscallN(addr, params...)
	runtime.KeepAlive(params)
	runtime.KeepAlive(buffers)
	runtime.KeepAlive(args)
	//println("debug 3 ended")

	refreshPointerValues(inter, params)

	return r1, r2, error(err)
}
