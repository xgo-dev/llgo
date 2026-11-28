// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build netbsd && (amd64 || arm64)

package syscall

type Timespec struct {
	Sec  int64
	Nsec int64
}

type Timeval struct {
	Sec       int64
	Usec      int32
	Pad_cgo_0 [4]byte
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
	Dev           uint64
	Mode          uint32
	Pad_cgo_0     [4]byte
	Ino           uint64
	Nlink         uint32
	Uid           uint32
	Gid           uint32
	Pad_cgo_1     [4]byte
	Rdev          uint64
	Atimespec     Timespec
	Mtimespec     Timespec
	Ctimespec     Timespec
	Birthtimespec Timespec
	Size          int64
	Blocks        int64
	Blksize       uint32
	Flags         uint32
	Gen           uint32
	Spare         [2]uint32
	Pad_cgo_2     [4]byte
}

type Dirent struct {
	Fileno    uint64
	Reclen    uint16
	Namlen    uint16
	Type      uint8
	Name      [512]int8
	Pad_cgo_0 [3]byte
}
