//go:build windows

package main

import "unsafe"

type NTSTATUS uint32

const STATUS_SUCCESS NTSTATUS = 0x00000000

type (
	USHORT  = uint16
	ULONG   = uint32
	ULONG64 = uint64
	HANDLE  = uintptr
	PVOID   = unsafe.Pointer
)


// UNICODE_STRING (ntdef.h)
type UNICODE_STRING struct {
	Length        USHORT
	MaximumLength USHORT
	Buffer        *uint16
}

// ANSI_STRING (ntdef.h)
type ANSI_STRING struct {
	Length        USHORT
	MaximumLength USHORT
	Buffer        *byte
}

// LARGE_INTEGER (ntdef.h)
type LARGE_INTEGER struct {
	LowPart  uint32
	HighPart int32
}

// LIST_ENTRY (ntdef.h)
type LIST_ENTRY struct {
	Flink *LIST_ENTRY
	Blink *LIST_ENTRY
}

// OBJECT_ATTRIBUTES (ntdef.h)
type OBJECT_ATTRIBUTES struct {
	Length             ULONG
	RootDirectory      HANDLE
	ObjectName         *UNICODE_STRING
	Attributes         ULONG
	SecurityDescriptor PVOID
	SecurityQoS        PVOID
}

// IO_STATUS_BLOCK (ntioapi.h)
type IO_STATUS_BLOCK struct {
	Status      uintptr
	Information uintptr
}

// CLIENT_ID (ntdef.h)
type CLIENT_ID struct {
	UniqueProcess HANDLE
	UniqueThread  HANDLE
}

//
// ─── Константы для OBJECT_ATTRIBUTES ─────────────────────────────────────────
//

const (
	OBJ_INHERIT             = 0x00000002
	OBJ_PERMANENT           = 0x00000010
	OBJ_EXCLUSIVE           = 0x00000020
	OBJ_CASE_INSENSITIVE    = 0x00000040
	OBJ_OPENIF              = 0x00000080
	OBJ_OPENLINK            = 0x00000100
	OBJ_KERNEL_HANDLE       = 0x00000200
	OBJ_FORCE_ACCESS_CHECK  = 0x00000400
	OBJ_VALID_ATTRIBUTES    = 0x000007F2
)

//
// ─── Константы для файловых операций ─────────────────────────────────────────
//

const (
	FILE_SUPERSEDE             = 0x00000000
	FILE_OPEN                  = 0x00000001
	FILE_CREATE                = 0x00000002
	FILE_OPEN_IF               = 0x00000003
	FILE_OVERWRITE             = 0x00000004
	FILE_OVERWRITE_IF          = 0x00000005

	FILE_DIRECTORY_FILE        = 0x00000001
	FILE_WRITE_THROUGH         = 0x00000002
	FILE_SEQUENTIAL_ONLY       = 0x00000004
	FILE_NO_INTERMEDIATE_BUFFERING = 0x00000008
	FILE_SYNCHRONOUS_IO_NONALERT   = 0x00000020
	FILE_NON_DIRECTORY_FILE        = 0x00000040
	FILE_RANDOM_ACCESS             = 0x00000800
)