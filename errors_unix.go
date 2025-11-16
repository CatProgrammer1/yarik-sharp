//go:build linux || darwin

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

	//unix
	case syscall.EPERM:
		throw("Operation not permitted (EPERM).", x, y, v...)
	case syscall.EIO:
		throw("I/O error occurred (EIO).", x, y, v...)
	case syscall.EBADF:
		throw("Bad file descriptor (EBADF).", x, y, v...)
	case syscall.EACCES:
		throw("Permission denied (EACCES).", x, y, v...)
	case syscall.EEXIST:
		throw("File already exists (EEXIST).", x, y, v...)
	case syscall.EINVAL:
		throw("Invalid argument (EINVAL).", x, y, v...)
	case syscall.EINTR:
		throw("Interrupted system call (EINTR).", x, y, v...)
	}
}
