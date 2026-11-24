// Copyright 2024 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// Platform definitions extracted from Go's stdlib syscall package
// (see LICENSES/Go-BSD-3-Clause.txt).

//go:build freebsd && (386 || amd64 || arm || arm64 || riscv64)

package syscall

const (
	EINTR      = Errno(0x4)
	ENOTSUP    = Errno(0x2d)
	EPERM      = Errno(0x1)
	F_GETFD    = 0x1
	F_SETFD    = 0x2
	F_GETFL    = 0x3
	F_SETFL    = 0x4
	FD_CLOEXEC = 0x1
	O_CLOEXEC  = 0x100000
	O_NONBLOCK = 0x4
	SIGPROF    = Signal(0x1b)
)
