// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build aix && ppc64

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

type StTimespec_t struct {
	Sec       int64
	Nsec      int32
	Pad_cgo_0 [4]byte
}

type Stat_t struct {
	Dev      uint64
	Ino      uint64
	Mode     uint32
	Nlink    int16
	Flag     uint16
	Uid      uint32
	Gid      uint32
	Rdev     uint64
	Ssize    int32
	Atim     StTimespec_t
	Mtim     StTimespec_t
	Ctim     StTimespec_t
	Blksize  int64
	Blocks   int64
	Vfstype  int32
	Vfs      uint32
	Type     uint32
	Gen      uint32
	Reserved [9]uint32
	Padto_ll uint32
	Size     int64
}

type Dirent struct {
	Offset    uint64
	Ino       uint64
	Reclen    uint16
	Namlen    uint16
	Name      [256]uint8
	Pad_cgo_0 [4]byte
}
