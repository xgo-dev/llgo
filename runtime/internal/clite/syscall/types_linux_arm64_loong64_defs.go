// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build linux && (arm64 || loong64)

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
	Dev               uint64
	Ino               uint64
	Mode              uint32
	Nlink             uint32
	Uid               uint32
	Gid               uint32
	Rdev              uint64
	X__pad1           uint64
	Size              int64
	Blksize           int32
	X__pad2           int32
	Blocks            int64
	Atim              Timespec
	Mtim              Timespec
	Ctim              Timespec
	X__glibc_reserved [2]int32
}

type Dirent struct {
	Ino       uint64
	Off       int64
	Reclen    uint16
	Type      uint8
	Name      [256]int8
	Pad_cgo_0 [5]byte
}
