// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build openbsd && (arm64 || mips64)

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
	Mode           uint32
	Dev            int32
	Ino            uint64
	Nlink          uint32
	Uid            uint32
	Gid            uint32
	Rdev           int32
	Atim           Timespec
	Mtim           Timespec
	Ctim           Timespec
	Size           int64
	Blocks         int64
	Blksize        int32
	Flags          uint32
	Gen            uint32
	X__st_birthtim Timespec
}

type Dirent struct {
	Fileno       uint64
	Off          int64
	Reclen       uint16
	Type         uint8
	Namlen       uint8
	X__d_padding [4]uint8
	Name         [256]int8
}
