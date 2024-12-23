package rdb

/*
#cgo CFLAGS: -g -Wall
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/pkg/native/aarch64-linux/librdb.a
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/pkg/native/x86_64-linux/librdb.a
#cgo windows,amd64 LDFLAGS: -L${SRCDIR}/pkg/native/x86_64-windows -lrdb -lntdll -lkernel32
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/pkg/native/aarch64-macos/librdb.a
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/pkg/native/x86_64-macos/librdb.a

#include <stdlib.h>
#include <stdint.h>
#include "./pkg/native/rdb.h"

extern bool rdb_go_callback(uintptr_t, struct Bytes, struct Bytes);

*/
import "C"
import (
	"errors"
	"fmt"
	"runtime/cgo"
	"unsafe"
)

type ErrNotFound struct{}

func (ErrNotFound) Error() string {
	return "not found"
}

// This bytes are allocated on the C side.
// Keep the original value unmodified so the Free function works as intended
type AllocatedBytes struct {
	Bytes []byte
}

func (b *AllocatedBytes) Free() {}

func toCBytes(str []byte) C.struct_Bytes {
	ptr := unsafe.Pointer(&str[0])
	length := len(str)
	return C.struct_Bytes{
		ptr: (*C.char)(ptr),
		len: C.uint64_t(length),
	}
}

func fromCBytes(b C.struct_Bytes) []byte {
	return C.GoBytes(unsafe.Pointer(b.ptr), C.int(b.len))
}

var (
	ErrCloseDB           = errors.New("database is closed")
	ErrUnexpectedEOF     = errors.New("unexpected end of file")
	ErrUnseekable        = errors.New("file is unseekable")
	ErrBrokenPipe        = errors.New("broken pipe")
	ErrNotOpenForWriting = errors.New("file not open for writing")
	ErrLockViolation     = errors.New("lock violation in file")
	ErrIsDir             = errors.New("database path is directory")
	ErrOutOfMemory       = errors.New("out of memory")
	ErrAccessDenied      = errors.New("access denied")
	ErrUnexpected        = errors.New("unexpected")
	ErrNotDocumented     = errors.New("error code not documented")
)

func rdb_error() error {
	errcode := C.rdb_error_code()
	switch errcode {
	case 0:
		return nil
	case 1:
		return ErrCloseDB
	case 2:
		return ErrUnexpectedEOF
	case 50:
		return ErrUnseekable
	case 51:
		return ErrBrokenPipe
	case 52:
		return ErrNotOpenForWriting
	case 53:
		return ErrLockViolation
	case 54:
		return ErrIsDir
	case 55:
		return ErrOutOfMemory
	case 56:
		return ErrAccessDenied
	case 99:
		return ErrUnexpected
	case 100:
		return ErrNotDocumented
	default:
		return fmt.Errorf("error code %d %w", errcode, ErrNotDocumented)
	}
}

type Database struct {
	pointer unsafe.Pointer
}

func New(path []byte) (Database, error) {
	r := C.rdb_open(toCBytes(path))
	if r.database == nil {
		return Database{}, rdb_error()
	}
	return Database{pointer: r.database}, nil

}

func (db Database) Set(key []byte, value []byte) error {
	if !bool(C.rdb_set(db.pointer, toCBytes(key), toCBytes(value))) {
		return rdb_error()
	}
	return nil
}

func (db Database) Get(key []byte) (AllocatedBytes, error) {
	ret := C.rdb_get(db.pointer, toCBytes(key))
	if !ret.valid {
		err := rdb_error()
		if err != nil {
			return AllocatedBytes{}, err
		} else {
			return AllocatedBytes{}, ErrNotFound{}
		}
	}
	return AllocatedBytes{Bytes: fromCBytes(ret.bytes)}, nil
}

func (db Database) Remove(key []byte) error {
	if !bool(C.rdb_remove(db.pointer, toCBytes(key))) {
		return rdb_error()
	}
	return nil
}

type GoCallback = func([]byte, []byte) bool

//export rdb_go_callback
func rdb_go_callback(handle C.uintptr_t, key C.struct_Bytes, value C.struct_Bytes) C._Bool {
	h := cgo.Handle(handle)
	callback := h.Value().(GoCallback)
	return C._Bool(callback(fromCBytes(key), fromCBytes(value)))
}

// Bytes in the callback are C owned bytes DO NOT USE THEM OUTSIDE THE CALLBACK BODY.
// In case you need to store them and use them the solution is to copy the bytes to
// golang object
//
// calling a [Database] function inside the body is ilegal behaviour
func (db Database) ForEach(fn GoCallback) error {
	handle := cgo.NewHandle(fn)
	defer handle.Delete()

	if !bool(C.rdb_foreach(db.pointer, (unsafe.Pointer)(handle), C.Callback(C.rdb_go_callback))) {
		return rdb_error()
	}
	return nil
}

func (db Database) Close() {
	C.rdb_close(db.pointer)
}
