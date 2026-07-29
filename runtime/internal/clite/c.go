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

package c

import (
	"unsafe"
)

type (
	Void    = [0]byte
	Char    = int8
	Float   = float32
	Double  = float64
	Pointer = unsafe.Pointer
	FilePtr = *FILE
)

type FILE struct {
	Unused [8]byte
}

type (
	Int  = int32
	Uint = uint32

	// Long and Ulong are defined in platform-specific files
	// Windows (both 32-bit and 64-bit): int32/uint32
	// Unix/Linux/macOS 32-bit: int32/uint32
	// Unix/Linux/macOS 64-bit: int64/uint64

	LongLong  = int64
	UlongLong = uint64
)

type integer interface {
	~int | ~uint | ~uintptr | ~int32 | ~uint32 | ~int64 | ~uint64
}

type SizeT = uintptr
type SsizeT = int

type IntptrT = uintptr
type UintptrT = uintptr
type Int8T = int8
type Int16T = int16
type Int32T = int32
type Int64T = int64

type Uint8T = uint8
type Uint16T = uint16
type Uint32T = uint32
type Uint64T = uint64

type IntmaxT = LongLong
type UintmaxT = UlongLong

type VaList = Pointer

//go:linkname Str llgo.cstr
func Str(string) *Char

//go:linkname Func llgo.funcAddr
func Func(any) Pointer

// llgo:link Advance llgo.advance
func Advance[PtrT any, I integer](ptr PtrT, offset I) PtrT { return ptr }

// llgo:link Index llgo.index
func Index[T any, I integer](ptr *T, offset I) T { return *ptr }

//go:linkname Alloca llgo.alloca
func Alloca(size uintptr) Pointer

//go:linkname AllocaCStr llgo.allocaCStr
func AllocaCStr(s string) *Char

//go:linkname AllocCStr llgo.allocCStr
func AllocCStr(s string) *Char

//go:linkname AllocaCStrs llgo.allocaCStrs
func AllocaCStrs(strs []string, endWithNil bool) **Char

// TODO(xsw):
// llgo:link AllocaNew llgo.allocaNew
func AllocaNew[T any]() *T { return nil }

//go:linkname Malloc C.malloc
func Malloc(size uintptr) Pointer

//go:linkname Calloc C.calloc
func Calloc(num uintptr, size uintptr) Pointer

//go:linkname Free C.free
func Free(ptr Pointer)

//go:linkname Realloc C.realloc
func Realloc(ptr Pointer, size uintptr) Pointer

//go:linkname Memcpy C.memcpy
func Memcpy(dst, src Pointer, n uintptr) Pointer

//go:linkname Memmove C.memmove
func Memmove(dst, src Pointer, n uintptr) Pointer

//go:linkname Memset C.memset
func Memset(s Pointer, c Int, n uintptr) Pointer

//go:linkname Memchr C.memchr
func Memchr(s Pointer, c Int, n uintptr) Pointer

//go:linkname Memcmp C.memcmp
func Memcmp(s1, s2 Pointer, n uintptr) Int

// -----------------------------------------------------------------------------

//go:linkname Strlen C.strlen
func Strlen(s *Char) uintptr

//go:linkname Strcpy C.strcpy
func Strcpy(dst, src *Char) *Char

//go:linkname Strncpy C.strncpy
func Strncpy(dst, src *Char, n uintptr) *Char

//go:linkname Strcat C.strcat
func Strcat(dst, src *Char) *Char

//go:linkname Strncat C.strncat
func Strncat(dst, src *Char, n uintptr) *Char

//go:linkname Strcmp C.strcmp
func Strcmp(s1, s2 *Char) Int

//go:linkname Strncmp C.strncmp
func Strncmp(s1, s2 *Char, n uintptr) Int

//go:linkname Strchr C.strchr
func Strchr(s *Char, c Int) *Char

//go:linkname Strrchr C.strrchr
func Strrchr(s *Char, c Int) *Char

//go:linkname Strstr C.strstr
func Strstr(s1, s2 *Char) *Char

//go:linkname Strdup C.strdup
func Strdup(s *Char) *Char

//go:linkname Strndup C.strndup
func Strndup(s *Char, n uintptr) *Char

//go:linkname Strtok C.strtok
func Strtok(s, delim *Char) *Char

//go:linkname Strerror C.strerror
func Strerror(errnum Int) *Char

//go:linkname Sprintf C.sprintf
func Sprintf(s *Char, format *Char, __llgo_va_list ...any) Int

//go:linkname Snprintf C.snprintf
func Snprintf(s *Char, n uintptr, format *Char, __llgo_va_list ...any) Int

//go:linkname Vsnprintf C.vsnprintf
func Vsnprintf(s *Char, n uintptr, format *Char, ap Pointer) Int

// -----------------------------------------------------------------------------

// GoString converts a C string to a Go string.
// TODO(xsw): any => int
//
//go:linkname GoString llgo.string
func GoString(cstr *Char, __llgo_va_list /* n */ ...any) string

//go:linkname GoStringData llgo.stringData
func GoStringData(string) *Char

//go:linkname GoDeferData llgo.deferData
func GoDeferData() Pointer

// -----------------------------------------------------------------------------

//go:linkname AllocaSigjmpBuf llgo.sigjmpbuf
func AllocaSigjmpBuf() Pointer

//go:linkname Sigsetjmp llgo.sigsetjmp
func Sigsetjmp(jb Pointer, savemask Int) Int

//go:linkname Siglongjmp llgo.siglongjmp
func Siglongjmp(jb Pointer, retval Int)

//go:linkname Unreachable llgo.unreachable
func Unreachable()

// -----------------------------------------------------------------------------

//go:linkname Exit C.exit
func Exit(Int)

// -----------------------------------------------------------------------------

//go:linkname Rand C.rand
func Rand() Int

//go:linkname Qsort C.qsort
func Qsort(base Pointer, count, elem uintptr, compar func(a, b Pointer) Int)

// -----------------------------------------------------------------------------

//go:linkname Atoi C.atoi
func Atoi(s *Char) Int

// -----------------------------------------------------------------------------

//go:linkname Printf C.printf
func Printf(format *Char, __llgo_va_list ...any) Int

//go:linkname Fprintf C.fprintf
func Fprintf(fp FilePtr, format *Char, __llgo_va_list ...any) Int

//go:linkname Fwrite C.fwrite
func Fwrite(data Pointer, size, count uintptr, fp FilePtr) uintptr

//go:linkname Fputc C.fputc
func Fputc(c Int, fp FilePtr) Int

//go:linkname Fputs C.fputs
func Fputs(s *Char, fp FilePtr) Int

//go:linkname Fread C.fread
func Fread(data Pointer, size, count uintptr, fp FilePtr) uintptr

//go:linkname Fflush C.fflush
func Fflush(fp FilePtr) Int

//go:linkname Fopen C.fopen
func Fopen(c *Char, mod *Char) FilePtr

//go:linkname Fclose C.fclose
func Fclose(fp FilePtr) Int

//go:linkname Perror C.perror
func Perror(s *Char)

// -----------------------------------------------------------------------------

type IconvT = Pointer

// -----------------------------------------------------------------------------

type LocaleT = Pointer

// -----------------------------------------------------------------------------

//go:linkname Usleep C.usleep
func Usleep(useconds Uint) Int

// -----------------------------------------------------------------------------

type Option struct {
	Name   *Char
	HasArg Int
	Flag   *Int
	Val    Int
}

//go:linkname Argc __llgo_argc
var Argc Int

//go:linkname Argv __llgo_argv
var Argv **Char

//go:linkname Optarg optarg
var Optarg *Char

//go:linkname Optind optind
var Optind Int

//go:linkname Opterr opterr
var Opterr Int

//go:linkname Optopt optopt
var Optopt Int

//go:linkname Getopt C.getopt
func Getopt(argc Int, argv **Char, optstring *Char) Int

//go:linkname GetoptLong C.getopt_long
func GetoptLong(argc Int, argv **Char, optstring *Char, longopts *Option, longindex *Int) Int

//go:linkname GetoptLongOnly C.getopt_long_only
func GetoptLongOnly(argc Int, argv **Char, optstring *Char, longopts *Option, longindex *Int) Int

// -----------------------------------------------------------------------------
