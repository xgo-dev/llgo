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

// int RAND_bytes(unsigned char *buf, int num);
//
//go:linkname RANDBufferWithLen C.RAND_bytes
func RANDBufferWithLen(buf *byte, num c.Int) c.Int

func RANDBytes(buf []byte) c.Int {
	return RANDBufferWithLen(unsafe.SliceData(buf), c.Int(len(buf)))
}

// int RAND_priv_bytes(unsigned char *buf, int num);
//
//go:linkname RANDPrivBufferWithLen C.RAND_priv_bytes
func RANDPrivBufferWithLen(buf *byte, num c.Int) c.Int

func RANDPrivBytes(buf []byte) c.Int {
	return RANDPrivBufferWithLen(unsafe.SliceData(buf), c.Int(len(buf)))
}

// void RAND_seed(const void *buf, int num);
//
//go:linkname RANDSeed C.RAND_seed
func RANDSeed(buf unsafe.Pointer, num c.Int)

// void RAND_keep_random_devices_open(int keep);
//
//go:linkname RANDKeepRandomDevicesOpen C.RAND_keep_random_devices_open
func RANDKeepRandomDevicesOpen(keep c.Int)

// int RAND_load_file(const char *file, long max_bytes);
//
//go:linkname RANDLoadFile C.RAND_load_file
func RANDLoadFile(file *c.Char, maxBytes c.Long) c.Int

// int RAND_write_file(const char *file);
//
//go:linkname RANDWriteFile C.RAND_write_file
func RANDWriteFile(file *c.Char) c.Int

// const char *RAND_file_name(char *file, size_t num);
//
//go:linkname RANDFileName C.RAND_file_name
func RANDFileName(file *c.Char, num uintptr) *c.Char

// int RAND_status(void);
//
//go:linkname RANDStatus C.RAND_status
func RANDStatus() c.Int

// int RAND_poll(void);
//
//go:linkname RANDPoll C.RAND_poll
func RANDPoll() c.Int

// -----------------------------------------------------------------------------
