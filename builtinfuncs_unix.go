//go:build linux || darwin

package main

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

var (
	builtinFuncs = map[string]func(v ...any) []any{
		"OS_NAME": func(v ...any) []any {
			return []any{runtime.GOOS}
		},
		"print": func(v ...any) []any {
			fmt.Println(format(v[2:]...))
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
		"tochar": func(v ...any) []any {
			argsCheck(v, 1, 1, "number")

			v = v[2:]

			b := byte(v[0].(float64))

			return []any{string(b)}
		},

		"callasm": func(v ...any) []any {
			argsCheck(v, 1, 1, "number")

			v = v[2:]

			b := byte(v[0].(float64))

			return []any{string(b)}
		},
	}
)
