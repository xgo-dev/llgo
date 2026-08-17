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

package openssl

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// -----------------------------------------------------------------------------

/*-
 * The error code packs differently depending on if it records a system
 * error or an OpenSSL error.
 *
 * A system error packs like this (we follow POSIX and only allow positive
 * numbers that fit in an |int|):
 *
 * +-+-------------------------------------------------------------+
 * |1|                     system error number                     |
 * +-+-------------------------------------------------------------+
 *
 * An OpenSSL error packs like this:
 *
 * <---------------------------- 32 bits -------------------------->
 *    <--- 8 bits ---><------------------ 23 bits ----------------->
 * +-+---------------+---------------------------------------------+
 * |0|    library    |                    reason                   |
 * +-+---------------+---------------------------------------------+
 *
 * A few of the reason bits are reserved as flags with special meaning:
 *
 *                    <5 bits-<>--------- 19 bits ----------------->
 *                   +-------+-+-----------------------------------+
 *                   | rflags| |          reason                   |
 *                   +-------+-+-----------------------------------+
 *                            ^
 *                            |
 *                           ERR_RFLAG_FATAL = ERR_R_FATAL
 *
 * The reason flags are part of the overall reason code for practical
 * reasons, as they provide an easy way to place different types of
 * reason codes in different numeric ranges.
 *
 * The currently known reason flags are:
 *
 * ERR_RFLAG_FATAL      Flags that the reason code is considered fatal.
 *                      For backward compatibility reasons, this flag
 *                      is also the code for ERR_R_FATAL (that reason
 *                      code served the dual purpose of flag and reason
 *                      code in one in pre-3.0 OpenSSL).
 * ERR_RFLAG_COMMON     Flags that the reason code is common to all
 *                      libraries.  All ERR_R_ macros must use this flag,
 *                      and no other _R_ macro is allowed to use it.
 */
type Errno c.Ulong

// ERR_get_error returns the earliest error code from the thread's error queue and
// removes the entry. This function can be called repeatedly until there are no more
// error codes to return.
//
// unsigned long ERR_get_error(void);
//
//go:linkname ERRGetError C.ERR_get_error
func ERRGetError() Errno

// ERR_get_error_all() is the same as ERR_get_error(), but on success it additionally
// stores the filename, line number and function where the error occurred in *file,
// *line and *func, and also extra text and flags in *data, *flags. If any of those
// parameters are NULL, it will not be changed.
//
// unsigned long ERR_get_error_all(
// const char **file, int *line, const char **func, const char **data, int *flags);
//
//go:linkname ERRGetErrorAll C.ERR_get_error_all
func ERRGetErrorAll(
	file **c.Char, line *c.Int, function **c.Char, data **c.Char, flags *c.Int) Errno

// unsigned long ERR_peek_error(void);
//
//go:linkname ERRPeekError C.ERR_peek_error
func ERRPeekError() Errno

// unsigned long ERR_peek_error_all(
// const char **file, int *line, const char **func, const char **data, int *flags);
//
//go:linkname ERRPeekErrorAll C.ERR_peek_error_all
func ERRPeekErrorAll(
	file **c.Char, line *c.Int, function **c.Char, data **c.Char, flags *c.Int) Errno

// unsigned long ERR_peek_last_error(void);
//
//go:linkname ERRPeekLastError C.ERR_peek_last_error
func ERRPeekLastError() Errno

// unsigned long ERR_peek_last_error_all(
// const char **file, int *line, const char **func, const char **data, int *flags);
//
//go:linkname ERRPeekLastErrorAll C.ERR_peek_last_error_all
func ERRPeekLastErrorAll(
	file **c.Char, line *c.Int, function **c.Char, data **c.Char, flags *c.Int) Errno

// void ERR_clear_error(void);
//
//go:linkname ERRClearError C.ERR_clear_error
func ERRClearError()

// ERR_error_string() generates a human-readable string representing the error code e,
// and places it at buf. buf must be at least 256 bytes long.
//
// char *ERR_error_string(unsigned long e, char *buf);
//
//go:linkname ERRErrorString C.ERR_error_string
func ERRErrorString(e Errno, buf *c.Char) *c.Char

// ERR_lib_error_string() and ERR_reason_error_string() return the library name and
// reason string respectively.
//
// const char *ERR_lib_error_string(unsigned long e);
//
//go:linkname ERRLibErrorString C.ERR_lib_error_string
func ERRLibErrorString(e Errno) *c.Char

// const char *ERR_reason_error_string(unsigned long e);
//
//go:linkname ERRReasonErrorString C.ERR_reason_error_string
func ERRReasonErrorString(e Errno) *c.Char

// void ERR_print_errors_cb(int (*cb) (const char *str, size_t len, void *u), void *u);
//
// [pid]:error:[error code]:[library name]:[function name]:[reason string]:[filename]:[line]:[optional text message]
//
//go:linkname ERRPrintErrorsCb C.ERR_print_errors_cb
func ERRPrintErrorsCb(cb func(str *c.Char, len uintptr, u unsafe.Pointer) c.Int, u unsafe.Pointer)

// void ERR_print_errors_fp(FILE *fp);
//
//go:linkname ERRPrintErrorsFp C.ERR_print_errors_fp
func ERRPrintErrorsFp(fp c.FilePtr)

// -----------------------------------------------------------------------------
