// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build linux && (mips || mipsle)

package syscall

const (
	EINTR      = Errno(0x4)
	ENOTSUP    = Errno(0x7a)
	EPERM      = Errno(0x1)
	F_GETFD    = 0x1
	F_SETFD    = 0x2
	F_GETFL    = 0x3
	F_SETFL    = 0x4
	FD_CLOEXEC = 0x1
	O_CLOEXEC  = 0x80000
	O_NONBLOCK = 0x80
	S_IFMT     = 0xf000
	S_IFREG    = 0x8000
	S_IFDIR    = 0x4000
	SIGPROF    = Signal(0x1d)
	SYS_CLONE  = 4120
)
