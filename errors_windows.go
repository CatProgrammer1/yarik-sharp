//go:build windows

package main

import (
	"os"
	"syscall"
)

func throwGoError(err error, x, y int, v ...any) {
	switch err {
	case os.ErrNotExist:
		throw("File \"%s\" doesn't exist.", x, y, v...)
	case os.ErrPermission:
		throw("File \"%s\" access denied.", x, y, v...)
	case os.ErrClosed:
		throw("File \"%s\" is already closed.", x, y, v...)

	//windows
	case syscall.ERROR_FILE_NOT_FOUND:
		throw("File not found.", x, y, v...)
	case syscall.ERROR_ACCESS_DENIED:
		throw("Access denied.", x, y, v...)
	case syscall.ERROR_HANDLE_EOF:
		throw("End of file reached.", x, y, v...)
	}
}
