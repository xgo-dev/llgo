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

const (
	/* get file status flags */
	F_GETFL = 3
	/* set file status flags */
	F_SETFL = 4

	/* open for reading only */
	O_RDONLY = 0x0000
	/* open for writing only */
	O_WRONLY = 0x0001
	/* open for reading and writing */
	O_RDWR = 0x0002
	/* mask for above modes */
	O_ACCMODE = 0x0003

	/* no delay */
	O_NONBLOCK = 0x00000004
	/* create if nonexistant */
	O_CREAT = 0x00000200
	/* truncate to zero length */
	O_TRUNC = 0x00000400
)

const (
	EAGAIN = 35
)

type (
	StatT = syscall.Stat_t
)

//go:linkname Errno C.cliteErrno
func Errno() c.Int

//go:linkname Umask C.umask
func Umask(cmask ModeT) ModeT

//go:linkname Mkdir C.mkdir
func Mkdir(path *c.Char, mode ModeT) c.Int

//go:linkname Rmdir C.rmdir
func Rmdir(path *c.Char) c.Int

//go:linkname Link C.link
func Link(oldpath *c.Char, newpath *c.Char) c.Int

//go:linkname Symlink C.symlink
func Symlink(target *c.Char, linkpath *c.Char) c.Int

//go:linkname Readlink C.readlink
func Readlink(path *c.Char, buf c.Pointer, bufsize uintptr) int

//go:linkname Unlink C.unlink
func Unlink(path *c.Char) c.Int

//go:linkname Unlinkat C.unlinkat
func Unlinkat(dirfd c.Int, path *c.Char, flags c.Int) c.Int

//go:linkname Remove C.remove
func Remove(path *c.Char) c.Int

//go:linkname Rename C.rename
func Rename(oldpath *c.Char, newpath *c.Char) c.Int

//go:linkname Truncate C.truncate
func Truncate(path *c.Char, length OffT) c.Int

//go:linkname Chmod C.chmod
func Chmod(path *c.Char, mode ModeT) c.Int

// -----------------------------------------------------------------------------

//go:linkname Getcwd C.getcwd
func Getcwd(buffer c.Pointer, size uintptr) *c.Char

//go:linkname Chdir C.chdir
func Chdir(path *c.Char) c.Int

//go:linkname Chroot C.chroot
func Chroot(path *c.Char) c.Int

// -----------------------------------------------------------------------------

//go:linkname Environ environ
var Environ **c.Char

//go:linkname Getenv C.getenv
func Getenv(name *c.Char) *c.Char

//go:linkname Setenv C.setenv
func Setenv(name *c.Char, value *c.Char, overwrite c.Int) c.Int

//go:linkname Putenv C.putenv
func Putenv(env *c.Char) c.Int

//go:linkname Unsetenv C.unsetenv
func Unsetenv(name *c.Char) c.Int

// -----------------------------------------------------------------------------

//go:linkname Fchdir C.fchdir
func Fchdir(dirfd c.Int) c.Int

//go:linkname Faccessat C.faccessat
func Faccessat(dirfd c.Int, path *c.Char, mode c.Int, flags c.Int) c.Int

//go:linkname Fchmodat C.fchmodat
func Fchmodat(dirfd c.Int, path *c.Char, mode ModeT, flags c.Int) c.Int

//go:linkname Fchownat C.fchownat
func Fchownat(dirfd c.Int, path *c.Char, owner UidT, group GidT, flags c.Int) c.Int

// -----------------------------------------------------------------------------

//go:linkname Open C.open
func Open(path *c.Char, flags c.Int, __llgo_va_list ...any) c.Int

//go:linkname Openat C.openat
func Openat(dirfd c.Int, path *c.Char, flags c.Int, mode ModeT) c.Int

//go:linkname Creat C.creat
func Creat(path *c.Char, mode ModeT) c.Int

//go:linkname Fcntl C.fcntl
func Fcntl(a c.Int, b c.Int, __llgo_va_list ...any) c.Int

//go:linkname Dup C.dup
func Dup(fd c.Int) c.Int

//go:linkname Dup2 C.dup2
func Dup2(oldfd c.Int, newfd c.Int) c.Int

//go:linkname Dup3 C.dup3
func Dup3(oldfd c.Int, newfd c.Int, flags c.Int) c.Int

//go:linkname Mkfifo C.mkfifo
func Mkfifo(path *c.Char, mode ModeT) c.Int

//go:linkname Mknod C.mknod
func Mknod(path *c.Char, mode ModeT, dev DevT) c.Int

//go:linkname Close C.close
func Close(fd c.Int) c.Int

//go:linkname Read C.read
func Read(fd c.Int, buf c.Pointer, count uintptr) int

//go:linkname Write C.write
func Write(fd c.Int, buf c.Pointer, count uintptr) int

//go:linkname Pread C.pread
func Pread(fd c.Int, buf c.Pointer, count uintptr, offset OffT) int

//go:linkname Pwrite C.pwrite
func Pwrite(fd c.Int, buf c.Pointer, count uintptr, offset OffT) int

//go:linkname Lseek C.lseek
func Lseek(fd c.Int, offset OffT, whence c.Int) OffT

//go:linkname Fsync C.fsync
func Fsync(fd c.Int) c.Int

//go:linkname Ftruncate C.ftruncate
func Ftruncate(fd c.Int, length OffT) c.Int

//go:linkname Fchmod C.fchmod
func Fchmod(fd c.Int, mode ModeT) c.Int

//go:linkname Fchown C.fchown
func Fchown(fd c.Int, owner UidT, group GidT) c.Int

//go:linkname Isatty C.isatty
func Isatty(fd c.Int) c.Int

// -----------------------------------------------------------------------------

// Execl(const char *path, const char *arg0, ..., /*, (char *)0, */)
//
// Execl requires the full path of the program to be provided.
//
//go:linkname Execl C.execl
func Execl(path *c.Char, arg0 *c.Char, __llgo_va_list ...any) c.Int

// Execle(const char *path, const char *arg0, ..., /* (char *)0, char *const envp[] */)
//
//go:linkname Execle C.execle
func Execle(path *c.Char, arg0 *c.Char, __llgo_va_list ...any) c.Int

// Execlp(const char *file, const char *arg0, ..., /*, (char *)0, */)
//
// Execlp only needs to provide the program name and it will search for the program in the
// paths specified in the PATH environment variable.
//
//go:linkname Execlp C.execlp
func Execlp(file *c.Char, arg0 *c.Char, __llgo_va_list ...any) c.Int

//go:linkname Execv C.execv
func Execv(path *c.Char, argv **c.Char) c.Int

//go:linkname Execve C.execve
func Execve(path *c.Char, argv **c.Char, envp **c.Char) c.Int

//go:linkname Execvp C.execvp
func Execvp(file *c.Char, argv **c.Char) c.Int

// -----------------------------------------------------------------------------

type PidT c.Int

//go:linkname Getpid C.getpid
func Getpid() PidT

//go:linkname Getppid C.getppid
func Getppid() PidT

// Invoke `system call' number SYSNO, passing it the remaining arguments.
// This is completely system-dependent, and not often useful.

// In Unix, `syscall' sets `errno' for all errors and most calls return -1
// for errors; in many systems you cannot pass arguments or get return
// values for all system calls (`pipe', `fork', and `getppid' typically
// among them).

// In Mach, all system calls take normal arguments and always return an
// error code (zero for success).
//
//go:linkname Syscall C.syscall
func Syscall(sysno c.Long, __llgo_va_list ...any) c.Long

// If wait() returns due to a stopped or terminated child process, the process ID
// of the child is returned to the calling process.  Otherwise, a value of -1 is
// returned and errno is set to indicate the error.
//
//go:linkname Wait C.wait
func Wait(statLoc *c.Int) PidT

// If wait3(), wait4(), or waitpid() returns due to a stopped or terminated child
// process, the process ID of the child is returned to the calling process.  If
// there are no children not previously awaited, -1 is returned with errno set to
// [ECHILD].  Otherwise, if WNOHANG is specified and there are no stopped or exited
// children, 0 is returned.  If an error is detected or a caught signal aborts the
// call, a value of -1 is returned and errno is set to indicate the error.
//
//go:linkname Wait3 C.wait3
func Wait3(statLoc *c.Int, options c.Int, rusage *syscall.Rusage) PidT

//go:linkname Waitpid C.waitpid
func Waitpid(pid PidT, statLoc *c.Int, options c.Int) PidT

// -----------------------------------------------------------------------------

//go:linkname Exit C.exit
func Exit(c.Int)

// -----------------------------------------------------------------------------

// Upon successful completion, the value 0 is returned; otherwise the value -1
// is returned and the global variable errno is set to indicate the error.
//
//go:linkname Sysctl C.sysctl
func Sysctl(
	name *c.Int, namelen c.Uint,
	oldp c.Pointer, oldlenp *uintptr,
	newp c.Pointer, newlen uintptr) c.Int

//go:linkname Sysctlbyname C.sysctlbyname
func Sysctlbyname(
	name *c.Char, oldp c.Pointer, oldlenp *uintptr,
	newp c.Pointer, newlen uintptr) c.Int

// The sysctlnametomib() function accepts an ASCII representation  of  the
// name,  looks up the integer name	vector,	and returns the	numeric	repre-
// sentation in the	mib array pointed to by	mibp.  The number of  elements
// in the mib array	is given by the	location specified by sizep before the
// call, and that location gives the number	of entries copied after	a suc-
// cessful	call.	The  resulting	mib and	size may be used in subsequent
// sysctl()	calls to get the data  associated  with	 the  requested	 ASCII
// name.   This interface is intended for use by applications that want to
// repeatedly request the same variable (the  sysctl()  function  runs  in
// about  a	third the time as the same request made	via the	sysctlbyname()
// function).  The sysctlnametomib() function is also useful for  fetching
// mib  prefixes and then adding a final component.
//
//go:linkname Sysctlnametomib C.sysctlnametomib
func Sysctlnametomib(name *c.Char, mibp *c.Int, sizep *uintptr) c.Int

// -----------------------------------------------------------------------------
