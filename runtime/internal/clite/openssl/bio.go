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

type BIO struct {
	Unused [0]byte
}

// BIO *BIO_new_mem_buf(const void *buf, int len);
//
//go:linkname BIONewMemBuf C.BIO_new_mem_buf
func BIONewMemBuf(buf unsafe.Pointer, len c.Int) *BIO

// int BIO_free(BIO *a);
//
// llgo:link (*BIO).Free C.BIO_free
func (*BIO) Free() c.Int { return 0 }

// int BIO_up_ref(BIO *a);
//
// llgo:link (*BIO).UpRef C.BIO_up_ref
func (*BIO) UpRef() c.Int { return 0 }

// int BIO_read_ex(BIO *b, void *data, size_t dlen, size_t *readbytes);
//
// llgo:link (*BIO).ReadEx C.BIO_read_ex
func (*BIO) ReadEx(data unsafe.Pointer, dlen uintptr, readbytes *uintptr) c.Int { return 0 }

// int BIO_write(BIO *b, const void *data, int dlen);
//
// llgo:link (*BIO).Write C.BIO_write
func (*BIO) Write(data unsafe.Pointer, dlen c.Int) c.Int { return 0 }

// int BIO_write_ex(BIO *b, const void *data, size_t dlen, size_t *written);
//
// llgo:link (*BIO).WriteEx C.BIO_write_ex
func (*BIO) WriteEx(data unsafe.Pointer, dlen uintptr, written *uintptr) c.Int { return 0 }

// -----------------------------------------------------------------------------
