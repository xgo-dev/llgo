// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build solaris && amd64

package syscall

type Timespec struct {
	Sec  int64
	Nsec int64
}

type Timeval struct {
	Sec  int64
	Usec int64
}

type Rusage struct {
	Utime    Timeval
	Stime    Timeval
	Maxrss   int64
	Ixrss    int64
	Idrss    int64
	Isrss    int64
	Minflt   int64
	Majflt   int64
	Nswap    int64
	Inblock  int64
	Oublock  int64
	Msgsnd   int64
	Msgrcv   int64
	Nsignals int64
	Nvcsw    int64
	Nivcsw   int64
}

type Rlimit struct {
	Cur uint64
	Max uint64
}

type Stat_t struct {
	Dev       uint64
	Ino       uint64
	Mode      uint32
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	Rdev      uint64
	Size      int64
	Atim      Timespec
	Mtim      Timespec
	Ctim      Timespec
	Blksize   int32
	Pad_cgo_0 [4]byte
	Blocks    int64
	Fstype    [16]int8
}

type Dirent struct {
	Ino       uint64
	Off       int64
	Reclen    uint16
	Name      [1]int8
	Pad_cgo_0 [5]byte
}
