// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build freebsd && 386

package syscall

type Timespec struct {
	Sec  int32
	Nsec int32
}

type Timeval struct {
	Sec  int32
	Usec int32
}

type Rusage struct {
	Utime    Timeval
	Stime    Timeval
	Maxrss   int32
	Ixrss    int32
	Idrss    int32
	Isrss    int32
	Minflt   int32
	Majflt   int32
	Nswap    int32
	Inblock  int32
	Oublock  int32
	Msgsnd   int32
	Msgrcv   int32
	Nsignals int32
	Nvcsw    int32
	Nivcsw   int32
}

type Rlimit struct {
	Cur int64
	Max int64
}

type Stat_t struct {
	Dev           uint64
	Ino           uint64
	Nlink         uint64
	Mode          uint16
	Padding0      int16
	Uid           uint32
	Gid           uint32
	Padding1      int32
	Rdev          uint64
	Atim_ext      int32
	Atimespec     Timespec
	Mtim_ext      int32
	Mtimespec     Timespec
	Ctim_ext      int32
	Ctimespec     Timespec
	Btim_ext      int32
	Birthtimespec Timespec
	Size          int64
	Blocks        int64
	Blksize       int32
	Flags         uint32
	Gen           uint64
	Spare         [10]uint64
}

type Dirent struct {
	Fileno uint64
	Off    int64
	Reclen uint16
	Type   uint8
	Pad0   uint8
	Namlen uint16
	Pad1   uint16
	Name   [256]int8
}
