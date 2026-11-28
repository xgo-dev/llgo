// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package syscall

type Errno uint32

type Signal uint8

type Dircookie = uint64

type Filetype = uint8

const (
	EINTR   Errno = 27
	ENOTSUP Errno = 58
	EPERM   Errno = 63

	SIGPROF Signal = 26

	F_GETFD = 1
	F_SETFD = 2
	F_GETFL = 3
	F_SETFL = 4

	O_CLOEXEC = 0

	S_IFMT  = 0000370000
	S_IFREG = 0000100000
	S_IFDIR = 0000040000
)

type Dirent struct {
	Next   Dircookie
	Ino    uint64
	Namlen uint32
	Type   Filetype
	Name   *byte
}

type Timespec struct {
	Sec  int64
	Nsec int64
}

type Timeval struct {
	Sec  int64
	Usec int64
}

type Rlimit struct {
	Cur uint64
	Max uint64
}

type Rusage struct {
	Utime Timeval
	Stime Timeval
}
