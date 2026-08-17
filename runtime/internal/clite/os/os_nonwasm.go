//go:build !wasm

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package os

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
)

//go:linkname Getuid C.getuid
func Getuid() UidT

//go:linkname Geteuid C.geteuid
func Geteuid() UidT

//go:linkname Getgid C.getgid
func Getgid() GidT

//go:linkname Getegid C.getegid
func Getegid() GidT

//go:linkname Chown C.chown
func Chown(path *c.Char, owner UidT, group GidT) c.Int

//go:linkname Lchown C.lchown
func Lchown(path *c.Char, owner UidT, group GidT) c.Int

//go:linkname Getrlimit C.getrlimit
func Getrlimit(resource c.Int, rlp *syscall.Rlimit) c.Int

//go:linkname Setrlimit C.setrlimit
func Setrlimit(resource c.Int, rlp *syscall.Rlimit) c.Int

//go:linkname Wait4 C.wait4
func Wait4(pid PidT, statLoc *c.Int, options c.Int, rusage *syscall.Rusage) PidT

/* TODO(xsw):
 On Alpha, IA-64, MIPS, SuperH, and SPARC/SPARC64, pipe() has the following prototype:
 struct fd_pair {
	 long fd[2];
 };
 struct fd_pair pipe(void);
*/
//go:linkname Pipe C.pipe
func Pipe(fds *[2]c.Int) c.Int

//go:linkname Fork C.fork
func Fork() PidT

//go:linkname Kill C.kill
func Kill(pid PidT, sig c.Int) c.Int
