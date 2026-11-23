// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build wasm

package syscall

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

func init() {
	// Try to set stdio to non-blocking mode before the os package
	// calls NewFile for each fd. NewFile queries the non-blocking flag
	// but doesn't change it, even if the runtime supports non-blocking
	// stdio. Since WebAssembly modules are single-threaded, blocking
	// system calls temporarily halt execution of the module. If the
	// runtime supports non-blocking stdio, the Go runtime is able to
	// use the WASI net poller to poll for read/write readiness and is
	// able to schedule goroutines while waiting.
	SetNonblock(0, true)
	SetNonblock(1, true)
	SetNonblock(2, true)
}

type uintptr32 = uint32
type size = uint32
type fdflags = uint32
type filesize = uint64
type filetype = uint8
type lookupflags = uint32
type oflags = uint32
type rights = uint64
type timestamp = uint64
type dircookie = uint64
type filedelta = int64
type fstflags = uint32

type iovec struct {
	_      structs.HostLayout
	buf    uintptr32
	bufLen size
}

const (
	LOOKUP_SYMLINK_FOLLOW = 0x00000001
)

const (
	OFLAG_CREATE    = 0x0001
	OFLAG_DIRECTORY = 0x0002
	OFLAG_EXCL      = 0x0004
	OFLAG_TRUNC     = 0x0008
)

const (
	FDFLAG_APPEND   = 0x0001
	FDFLAG_DSYNC    = 0x0002
	FDFLAG_NONBLOCK = 0x0004
	FDFLAG_RSYNC    = 0x0008
	FDFLAG_SYNC     = 0x0010
)

const (
	RIGHT_FD_DATASYNC = 1 << iota
	RIGHT_FD_READ
	RIGHT_FD_SEEK
	RIGHT_FDSTAT_SET_FLAGS
	RIGHT_FD_SYNC
	RIGHT_FD_TELL
	RIGHT_FD_WRITE
	RIGHT_FD_ADVISE
	RIGHT_FD_ALLOCATE
	RIGHT_PATH_CREATE_DIRECTORY
	RIGHT_PATH_CREATE_FILE
	RIGHT_PATH_LINK_SOURCE
	RIGHT_PATH_LINK_TARGET
	RIGHT_PATH_OPEN
	RIGHT_FD_READDIR
	RIGHT_PATH_READLINK
	RIGHT_PATH_RENAME_SOURCE
	RIGHT_PATH_RENAME_TARGET
	RIGHT_PATH_FILESTAT_GET
	RIGHT_PATH_FILESTAT_SET_SIZE
	RIGHT_PATH_FILESTAT_SET_TIMES
	RIGHT_FD_FILESTAT_GET
	RIGHT_FD_FILESTAT_SET_SIZE
	RIGHT_FD_FILESTAT_SET_TIMES
	RIGHT_PATH_SYMLINK
	RIGHT_PATH_REMOVE_DIRECTORY
	RIGHT_PATH_UNLINK_FILE
	RIGHT_POLL_FD_READWRITE
	RIGHT_SOCK_SHUTDOWN
	RIGHT_SOCK_ACCEPT
)

const (
	WHENCE_SET = 0
	WHENCE_CUR = 1
	WHENCE_END = 2
)

const (
	FILESTAT_SET_ATIM     = 0x0001
	FILESTAT_SET_ATIM_NOW = 0x0002
	FILESTAT_SET_MTIM     = 0x0004
	FILESTAT_SET_MTIM_NOW = 0x0008
)

const (
	// Despite the rights being defined as a 64 bits integer in the spec,
	// wasmtime crashes the program if we set any of the upper 32 bits.
	fullRights  = rights(^uint32(0))
	readRights  = rights(RIGHT_FD_READ | RIGHT_FD_READDIR)
	writeRights = rights(RIGHT_FD_DATASYNC | RIGHT_FD_WRITE | RIGHT_FD_ALLOCATE | RIGHT_PATH_FILESTAT_SET_SIZE)

	// Some runtimes have very strict expectations when it comes to which
	// rights can be enabled on files opened by path_open. The fileRights
	// constant is used as a mask to retain only bits for operations that
	// are supported on files.
	fileRights rights = RIGHT_FD_DATASYNC |
		RIGHT_FD_READ |
		RIGHT_FD_SEEK |
		RIGHT_FDSTAT_SET_FLAGS |
		RIGHT_FD_SYNC |
		RIGHT_FD_TELL |
		RIGHT_FD_WRITE |
		RIGHT_FD_ADVISE |
		RIGHT_FD_ALLOCATE |
		RIGHT_PATH_CREATE_DIRECTORY |
		RIGHT_PATH_CREATE_FILE |
		RIGHT_PATH_LINK_SOURCE |
		RIGHT_PATH_LINK_TARGET |
		RIGHT_PATH_OPEN |
		RIGHT_FD_READDIR |
		RIGHT_PATH_READLINK |
		RIGHT_PATH_RENAME_SOURCE |
		RIGHT_PATH_RENAME_TARGET |
		RIGHT_PATH_FILESTAT_GET |
		RIGHT_PATH_FILESTAT_SET_SIZE |
		RIGHT_PATH_FILESTAT_SET_TIMES |
		RIGHT_FD_FILESTAT_GET |
		RIGHT_FD_FILESTAT_SET_SIZE |
		RIGHT_FD_FILESTAT_SET_TIMES |
		RIGHT_PATH_SYMLINK |
		RIGHT_PATH_REMOVE_DIRECTORY |
		RIGHT_PATH_UNLINK_FILE |
		RIGHT_POLL_FD_READWRITE

	// Runtimes like wasmtime and wasmedge will refuse to open directories
	// if the rights requested by the application exceed the operations that
	// can be performed on a directory.
	dirRights rights = RIGHT_FD_SEEK |
		RIGHT_FDSTAT_SET_FLAGS |
		RIGHT_FD_SYNC |
		RIGHT_PATH_CREATE_DIRECTORY |
		RIGHT_PATH_CREATE_FILE |
		RIGHT_PATH_LINK_SOURCE |
		RIGHT_PATH_LINK_TARGET |
		RIGHT_PATH_OPEN |
		RIGHT_FD_READDIR |
		RIGHT_PATH_READLINK |
		RIGHT_PATH_RENAME_SOURCE |
		RIGHT_PATH_RENAME_TARGET |
		RIGHT_PATH_FILESTAT_GET |
		RIGHT_PATH_FILESTAT_SET_SIZE |
		RIGHT_PATH_FILESTAT_SET_TIMES |
		RIGHT_FD_FILESTAT_GET |
		RIGHT_FD_FILESTAT_SET_TIMES |
		RIGHT_PATH_SYMLINK |
		RIGHT_PATH_REMOVE_DIRECTORY |
		RIGHT_PATH_UNLINK_FILE
)

// https://github.com/WebAssembly/WASI/blob/a2b96e81c0586125cc4dc79a5be0b78d9a059925/legacy/preview1/docs.md#-fdstat-record
// fdflags must be at offset 2, hence the uint16 type rather than the
// fdflags (uint32) type.
type fdstat struct {
	_                structs.HostLayout
	filetype         filetype
	fdflags          uint16
	rightsBase       rights
	rightsInheriting rights
}

type preopentype = uint8

const (
	preopentypeDir preopentype = iota
)

type prestatDir struct {
	_         structs.HostLayout
	prNameLen size
}

type prestat struct {
	_   structs.HostLayout
	typ preopentype
	dir prestatDir
}

// Current working directory. We maintain this as a string and resolve paths in
// the code because wasmtime does not allow relative path lookups outside of the
// scope of a directory; a previous approach we tried consisted in maintaining
// open a file descriptor to the current directory so we could perform relative
// path lookups from that location, but it resulted in breaking path resolution
// from the current directory to its parent.
var cwd string

//go:nosplit
func appendCleanPath(buf []byte, path string, lookupParent bool) ([]byte, bool) {
	i := 0
	for i < len(path) {
		for i < len(path) && path[i] == '/' {
			i++
		}

		j := i
		for j < len(path) && path[j] != '/' {
			j++
		}

		s := path[i:j]
		i = j

		switch s {
		case "":
			continue
		case ".":
			continue
		case "..":
			if !lookupParent {
				k := len(buf)
				for k > 0 && buf[k-1] != '/' {
					k--
				}
				for k > 1 && buf[k-1] == '/' {
					k--
				}
				buf = buf[:k]
				if k == 0 {
					lookupParent = true
				} else {
					s = ""
					continue
				}
			}
		default:
			lookupParent = false
		}

		if len(buf) > 0 && buf[len(buf)-1] != '/' {
			buf = append(buf, '/')
		}
		buf = append(buf, s...)
	}
	return buf, lookupParent
}

// joinPath concatenates dir and file paths, producing a cleaned path where
// "." and ".." have been removed, unless dir is relative and the references
// to parent directories in file represented a location relative to a parent
// of dir.
//
// This function is used for path resolution of all wasi functions expecting
// a path argument; the returned string is heap allocated, which we may want
// to optimize in the future. Instead of returning a string, the function
// could append the result to an output buffer that the functions in this
// file can manage to have allocated on the stack (e.g. initializing to a
// fixed capacity). Since it will significantly increase code complexity,
// we prefer to optimize for readability and maintainability at this time.
func joinPath(dir, file string) string {
	buf := make([]byte, 0, len(dir)+len(file)+1)
	if isAbs(dir) {
		buf = append(buf, '/')
	}
	buf, lookupParent := appendCleanPath(buf, dir, false)
	buf, _ = appendCleanPath(buf, file, lookupParent)
	// The appendCleanPath function cleans the path so it does not inject
	// references to the current directory. If both the dir and file args
	// were ".", this results in the output buffer being empty so we handle
	// this condition here.
	if len(buf) == 0 {
		buf = append(buf, '.')
	}
	// If the file ended with a '/' we make sure that the output also ends
	// with a '/'. This is needed to ensure that programs have a mechanism
	// to represent dereferencing symbolic links pointing to directories.
	if buf[len(buf)-1] != '/' && isDir(file) {
		buf = append(buf, '/')
	}
	return unsafe.String(&buf[0], len(buf))
}

func isAbs(path string) bool {
	return hasPrefix(path, "/")
}

func isDir(path string) bool {
	return hasSuffix(path, "/")
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
