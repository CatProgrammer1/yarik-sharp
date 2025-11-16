//go:build linux || darwin

package main

import (
	"syscall"
)

var (
	sysFuncs = []*FuncDec{
		newFTemp("Open", func(v ...any) []any {
			argsCheck(v, 3, 3, "string", "number", "number")

			v = v[2:]

			name := v[0].(string)
			flag := int(v[1].(float64))
			perm := uint32(v[2].(float64))

			fd, err := syscall.Open(name, flag, perm)

			return []any{fd, err}
		}),

		newFTemp("Close", func(v ...any) []any {
			argsCheck(v, 1, 1, "string", "number", "number")

			v = v[2:]

			fd := v[0].(float64)

			err := syscall.Close(int(fd))

			return []any{err}
		}),

		newFTemp("Read", func(v ...any) []any {
			argsCheck(v, 3, 3, "string", "table", "number")

			v = v[2:]

			fd := int(v[0].(float64))
			p := mapToSlice[byte](v[1].(map[any]any))

			n, err := syscall.Read(fd, p)

			return []any{n, err}
		}),
	}
)
