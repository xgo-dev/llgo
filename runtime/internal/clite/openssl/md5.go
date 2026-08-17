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

const (
	MD5_CBLOCK = 64
	MD5_LBLOCK = MD5_CBLOCK / 4
)

// -----------------------------------------------------------------------------

type MD5_LONG = c.Uint

type MD5_CTX struct {
	A, B, C, D MD5_LONG
	Nl, Nh     MD5_LONG
	Data       [MD5_LBLOCK]MD5_LONG
	Num        c.Uint
}

// OSSL_DEPRECATEDIN_3_0 int MD5_Init(MD5_CTX *c);
//
// llgo:link (*MD5_CTX).Init C.MD5_Init
func (c *MD5_CTX) Init() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int MD5_Update(MD5_CTX *c, const void *data, size_t len);
//
// llgo:link (*MD5_CTX).Update C.MD5_Update
func (c *MD5_CTX) Update(data unsafe.Pointer, n uintptr) c.Int { return 0 }

func (c *MD5_CTX) UpdateBytes(data []byte) c.Int {
	return c.Update(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)))
}

func (c *MD5_CTX) UpdateString(data string) c.Int {
	return c.Update(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)))
}

// OSSL_DEPRECATEDIN_3_0 int MD5_Final(unsigned char *md, MD5_CTX *c);
//
//go:linkname md5Final C.MD5_Final
func md5Final(md *byte, c *MD5_CTX) c.Int

func (c *MD5_CTX) Final(md *byte) c.Int {
	return md5Final(md, c)
}

// OSSL_DEPRECATEDIN_3_0 unsigned char *MD5(const unsigned char *d, size_t n, unsigned char *md);
//
//go:linkname MD5 C.MD5
func MD5(data unsafe.Pointer, n uintptr, md *byte) *byte

func MD5Bytes(data []byte, md *byte) *byte {
	return MD5(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)), md)
}

func MD5String(data string, md *byte) *byte {
	return MD5(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)), md)
}

// OSSL_DEPRECATEDIN_3_0 void MD5_Transform(MD5_CTX *c, const unsigned char *b);

// -----------------------------------------------------------------------------
